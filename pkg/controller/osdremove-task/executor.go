/*
Copyright 2025 The Mirantis Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package osdremove

import (
	"fmt"
	"sort"
	"strings"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"

	lcmv1alpha1 "github.com/Mirantis/pelagia/pkg/apis/ceph.pelagia.lcm/v1alpha1"
	lcmcommon "github.com/Mirantis/pelagia/pkg/common"
)

var (
	retriesForFailedCommand = 5
	commandRetryRunTimeout  = 30 * time.Second
)

type hostOsdPair struct {
	Host  string
	OsdID string
}

// process next osd removal, returns whether its finished or not
// and updated remove info
func (c *cephOsdRemoveConfig) processOsdRemoveTask() (bool, *lcmv1alpha1.TaskRemoveInfo) {
	newRemoveInfo := c.taskConfig.task.Status.RemoveInfo.DeepCopy()
	reqMap := map[lcmv1alpha1.RemovePhase][]hostOsdPair{
		lcmv1alpha1.RemovePending:          {},
		lcmv1alpha1.RemoveWaitingRebalance: {},
		lcmv1alpha1.RemoveInProgress:       {},
		lcmv1alpha1.RemoveFinished:         {},
		lcmv1alpha1.RemoveFailed:           {},
	}
	hostsToRemove := make([]string, 0)
	// collect info about statuses
	for host, hostMap := range c.taskConfig.task.Status.RemoveInfo.CleanupMap {
		osdsNotCompleted := len(hostMap.OsdMapping)
		for osdID, osdMap := range hostMap.OsdMapping {
			// should match only during first run, when nothing has remove status
			hostPair := hostOsdPair{Host: host, OsdID: osdID}
			if osdMap.RemoveStatus == nil {
				osdMap.RemoveStatus = &lcmv1alpha1.RemoveResult{OsdRemoveStatus: &lcmv1alpha1.RemoveStatus{}}
				// process stray remove differently from usual remove, since
				// for stray we can remove it without need to wait out/rebalance and
				// for just stray partition we need drop auth keys if present, job run and deploy remove
				if isStrayOsdID(osdID) || host == lcmcommon.StrayOsdNodeMarker {
					if osdMap.InCrushMap {
						osdMap.RemoveStatus.OsdRemoveStatus.Status = lcmv1alpha1.RemoveInProgress
					} else {
						osdMap.RemoveStatus.OsdRemoveStatus.Status = lcmv1alpha1.RemoveStray
					}
				} else {
					osdMap.RemoveStatus.OsdRemoveStatus.Status = lcmv1alpha1.RemovePending
				}
				reqMap[osdMap.RemoveStatus.OsdRemoveStatus.Status] = append(reqMap[osdMap.RemoveStatus.OsdRemoveStatus.Status], hostPair)
				newRemoveInfo.CleanupMap[host].OsdMapping[osdID] = osdMap
				continue
			}
			if osdMap.RemoveStatus.OsdRemoveStatus != nil {
				// if we found some skipped status that means stray osd already handled
				// but it may require to clean deployment and devices as well
				if osdMap.RemoveStatus.OsdRemoveStatus.Status == lcmv1alpha1.RemoveSkipped {
					reqMap[lcmv1alpha1.RemoveFinished] = append(reqMap[lcmv1alpha1.RemoveFinished], hostPair)
				} else {
					reqMap[osdMap.RemoveStatus.OsdRemoveStatus.Status] = append(reqMap[osdMap.RemoveStatus.OsdRemoveStatus.Status], hostPair)
				}

				if osdMap.RemoveStatus.OsdRemoveStatus.Status == lcmv1alpha1.RemoveFinished || osdMap.RemoveStatus.OsdRemoveStatus.Status == lcmv1alpha1.RemoveSkipped {
					osdsNotCompleted--
				}
			}
		}
		// mark host to remove from crush map if osd already being removed successfully
		if (hostMap.CompleteCleanup || hostMap.DropFromCrush) && osdsNotCompleted == 0 && hostMap.HostRemoveStatus == nil {
			hostsToRemove = append(hostsToRemove, host)
		}
	}

	// drop first marked hosts and update status
	for _, host := range hostsToRemove {
		c.taskConfig.requeueNow = true
		newStatus := c.removeHostFromCrush(host)
		if newStatus.Error != "" {
			// since host crush remove is not a critical error just add warning
			newRemoveInfo.Warnings = append(newRemoveInfo.Warnings, fmt.Sprintf("[node '%s'] failed to remove node from crush map: %s", host, newStatus.Error))
		}
		hostMap := newRemoveInfo.CleanupMap[host]
		hostMap.HostRemoveStatus = newStatus
		newRemoveInfo.CleanupMap[host] = hostMap
	}

	// if we did any host removal - update statuses first
	if c.taskConfig.requeueNow {
		return false, newRemoveInfo
	}

	curIssues := make([]string, 0)
	// by default call requeue w/o interval, since we need to move phases
	// denied implicitly when it is required
	c.taskConfig.requeueNow = true
	// if we failed to remove osd - clean all pending, since we allowing only 1 osd remove/rebalance at time
	if len(reqMap[lcmv1alpha1.RemoveFailed]) > 0 && len(reqMap[lcmv1alpha1.RemovePending]) > 0 {
		reqMap[lcmv1alpha1.RemovePending] = nil
	}
	for _, pair := range reqMap[lcmv1alpha1.RemoveFailed] {
		c.log.Error().Msgf("detected fail during osd '%s' remove", pair.OsdID)
		curIssues = append(curIssues, fmt.Sprintf("[node '%s'] failed to remove osd '%s'", pair.Host, pair.OsdID))
	}
	// counter for checking not finished requests
	notCompleted := 0
	// for all finished osd removes, check first job state and then
	// check deployment remove state if needed
	for _, pair := range reqMap[lcmv1alpha1.RemoveFinished] {
		hostMapping := newRemoveInfo.CleanupMap[pair.Host]
		osdMapping := hostMapping.OsdMapping[pair.OsdID]
		if osdMapping.RemoveStatus.DeviceCleanUpJob == nil || osdMapping.RemoveStatus.DeviceCleanUpJob.Status == lcmv1alpha1.RemoveInProgress ||
			osdMapping.RemoveStatus.DeviceCleanUpJob.Status == lcmv1alpha1.RemovePending {
			if osdMapping.RemoveStatus.DeviceCleanUpJob == nil && (hostMapping.NodeIsDown || hostMapping.VolumesInfoMissed ||
				hostMapping.DropFromCrush || len(osdMapping.DeviceMapping) == 0) || osdMapping.SkipDeviceCleanupJob {
				msg := fmt.Sprintf("skipping device cleanup job for node '%s'", pair.Host)
				if hostMapping.NodeIsDown {
					msg = fmt.Sprintf("%s, because node is down", msg)
				} else if hostMapping.VolumesInfoMissed {
					msg = fmt.Sprintf("%s, because volume info from node is not available", msg)
				} else if hostMapping.DropFromCrush {
					msg = fmt.Sprintf("%s, because set remove from crush only mode (dropFromCrush flag set)", msg)
				} else if osdMapping.SkipDeviceCleanupJob {
					msg = fmt.Sprintf("%s, because set skip cleanup job flag in spec", msg)
				} else {
					msg = fmt.Sprintf("%s, nothing to cleanup", msg)
				}
				c.log.Info().Msg(msg)
				newRemoveInfo.CleanupMap[pair.Host].OsdMapping[pair.OsdID].RemoveStatus.DeviceCleanUpJob = &lcmv1alpha1.RemoveStatus{Status: lcmv1alpha1.RemoveSkipped}
			} else {
				newRemoveInfo.CleanupMap[pair.Host].OsdMapping[pair.OsdID].RemoveStatus.DeviceCleanUpJob = c.handleJobRun(pair.OsdID, pair.Host, hostMapping.OsdMapping)
				if newRemoveInfo.CleanupMap[pair.Host].OsdMapping[pair.OsdID].RemoveStatus.DeviceCleanUpJob.Status != lcmv1alpha1.RemoveCompleted {
					notCompleted++
					// do not requeue - wait for job
					if len(reqMap[lcmv1alpha1.RemoveWaitingRebalance]) == 0 && len(reqMap[lcmv1alpha1.RemovePending]) == 0 {
						c.taskConfig.requeueNow = false
					}
					continue
				}
			}
		} else if osdMapping.RemoveStatus.DeviceCleanUpJob.Status == lcmv1alpha1.RemoveFailed {
			curIssues = append(curIssues, fmt.Sprintf("[node '%s'] disk cleanup job '%s' has failed, clean up disk/partitions manually",
				pair.Host, osdMapping.RemoveStatus.DeviceCleanUpJob.Name))
			curIssues = append(curIssues, fmt.Sprintf("[node '%s'] deployment 'rook-ceph-osd-%s' is not removed, because job '%s' is failed",
				pair.Host, pair.OsdID, osdMapping.RemoveStatus.DeviceCleanUpJob.Name))
			continue
		} else if osdMapping.RemoveStatus.DeviceCleanUpJob.Status != lcmv1alpha1.RemoveCompleted &&
			osdMapping.RemoveStatus.DeviceCleanUpJob.Status != lcmv1alpha1.RemoveSkipped {
			// should not happen - but double check to avoid unexpected deployment remove
			msg := fmt.Sprintf("unexpected remove status '%v' for device cleanup job '%s', deployment remove for osd '%s' on node '%s' skipped",
				osdMapping.RemoveStatus.DeviceCleanUpJob.Status, osdMapping.RemoveStatus.DeviceCleanUpJob.Name, pair.OsdID, pair.Host)
			c.log.Error().Msg(msg)
			newRemoveInfo.Warnings = append(newRemoveInfo.Warnings, msg)
			continue
		}
		// do not try remove deployment if osd remove was skipped or device cleanup job remove skipped through spec
		if osdMapping.RemoveStatus.DeployRemoveStatus == nil {
			if osdMapping.RemoveStatus.OsdRemoveStatus.Status == lcmv1alpha1.RemoveFinished && !osdMapping.SkipDeviceCleanupJob {
				notCompleted++
				newRemoveInfo.CleanupMap[pair.Host].OsdMapping[pair.OsdID].RemoveStatus.DeployRemoveStatus = c.removeDeployment(pair.OsdID)
			} else {
				newRemoveInfo.CleanupMap[pair.Host].OsdMapping[pair.OsdID].RemoveStatus.DeployRemoveStatus = &lcmv1alpha1.RemoveStatus{
					Status: lcmv1alpha1.RemoveSkipped,
				}
			}
		} else if osdMapping.RemoveStatus.DeployRemoveStatus.Status == lcmv1alpha1.RemoveFailed {
			curIssues = append(curIssues, fmt.Sprintf("[node '%s'] failed to remove deployment 'rook-ceph-osd-%s'", pair.Host, pair.OsdID))
		}
	}
	sort.Strings(curIssues)
	newRemoveInfo.Issues = curIssues

	// case when stray osd is only present on a node and not present in crush map
	// so in case if that stray also reflected in crush map - remove it and keyring
	// otherwise - skip crush map change and run only cleanup job
	for _, pair := range reqMap[lcmv1alpha1.RemoveStray] {
		notCompleted++
		newStatus := c.removeStray(pair.OsdID, newRemoveInfo.CleanupMap[pair.Host].OsdMapping[pair.OsdID].RemoveStatus.OsdRemoveStatus)
		newRemoveInfo.CleanupMap[pair.Host].OsdMapping[pair.OsdID].RemoveStatus.OsdRemoveStatus = newStatus
	}
	// start osd removal from cluster
	// since to that stage all osd are out we can safely start removing other osds
	for _, pair := range reqMap[lcmv1alpha1.RemoveInProgress] {
		notCompleted++
		newStatus := c.removeFromCrush(pair.OsdID, newRemoveInfo.CleanupMap[pair.Host].OsdMapping[pair.OsdID].RemoveStatus.OsdRemoveStatus)
		newRemoveInfo.CleanupMap[pair.Host].OsdMapping[pair.OsdID].RemoveStatus.OsdRemoveStatus = newStatus
	}
	// check rebalance progress before continue to next pending
	for _, pair := range reqMap[lcmv1alpha1.RemoveWaitingRebalance] {
		notCompleted++
		newStatus := c.checkRebalance(pair.OsdID, newRemoveInfo.CleanupMap[pair.Host].OsdMapping[pair.OsdID].RemoveStatus.OsdRemoveStatus)
		newRemoveInfo.CleanupMap[pair.Host].OsdMapping[pair.OsdID].RemoveStatus.OsdRemoveStatus = newStatus
		// if we finished with rebalance - no need wait
		if newStatus.Status != lcmv1alpha1.RemoveInProgress {
			c.taskConfig.requeueNow = false
		}
	}
	// allow to remove from crush only 1 osd at time since clusters may have complex crush
	// hierarchy it is hard to determine correct failure domains for each osd, which is
	// going to be removed, so use static 1 at time in case of speed up remove, operator may
	// reweight to 0 all possible osds and then run lcm task to save time for rebalance operations.
	if len(reqMap[lcmv1alpha1.RemoveWaitingRebalance]) > 0 {
		return false, newRemoveInfo
	}

	// do not call requeue immediately if osd can't be stopped right now
	// if we found other osd which can be stopped - procceed with it w/o timeout
	waitingOsd := map[string]string{}
	// check pending osds
	for _, pair := range reqMap[lcmv1alpha1.RemovePending] {
		notCompleted++
		newStatus := c.tryToMoveOsdOut(pair.OsdID, newRemoveInfo.CleanupMap[pair.Host].OsdMapping[pair.OsdID].RemoveStatus.OsdRemoveStatus)
		newRemoveInfo.CleanupMap[pair.Host].OsdMapping[pair.OsdID].RemoveStatus.OsdRemoveStatus = newStatus
		if newStatus.Status == lcmv1alpha1.RemovePending {
			waitingOsd[pair.OsdID] = pair.Host
			continue
		}
		// reset all times for pending waiting, they should have new wait start time
		for osd, host := range waitingOsd {
			newRemoveInfo.CleanupMap[host].OsdMapping[osd].RemoveStatus.OsdRemoveStatus.StartedAt = ""
		}
		waitingOsd = nil
		break
	}

	if notCompleted == 0 {
		return true, newRemoveInfo
	}
	c.taskConfig.requeueNow = c.taskConfig.requeueNow && len(waitingOsd) == 0
	return false, newRemoveInfo
}

func (c *cephOsdRemoveConfig) removeHostFromCrush(host string) *lcmv1alpha1.RemoveStatus {
	c.log.Info().Msgf("removing host '%s' from crush map", host)
	hostRemoveStatus := &lcmv1alpha1.RemoveStatus{StartedAt: lcmcommon.GetCurrentTimeString()}
	_, err := lcmcommon.RunFuncWithRetry(retriesForFailedCommand, commandRetryRunTimeout, func() (interface{}, error) {
		_, cmdErr := lcmcommon.RunCephToolboxCLI(c.context, c.api.Kubeclientset, c.api.Config, c.taskConfig.cephCluster.Namespace, fmt.Sprintf("ceph osd crush remove %s", host))
		if cmdErr != nil {
			c.log.Error().Err(cmdErr).Msg("")
		}
		return false, cmdErr
	})
	if err != nil {
		c.log.Error().Err(err).Msg("")
		hostRemoveStatus.Status = lcmv1alpha1.RemoveFailed
		hostRemoveStatus.Error = err.Error()
	} else {
		hostRemoveStatus.Status = lcmv1alpha1.RemoveFinished
		hostRemoveStatus.FinishedAt = lcmcommon.GetCurrentTimeString()
	}
	return hostRemoveStatus
}

func (c *cephOsdRemoveConfig) removeDeployment(osdID string) *lcmv1alpha1.RemoveStatus {
	osdIDForDeploy := osdID
	if isStrayOsdID(osdIDForDeploy) {
		osdIDForDeploy = strings.Split(osdID, ".")[0]
	}
	deployName := fmt.Sprintf("rook-ceph-osd-%s", osdIDForDeploy)
	deployRemoveStatus := &lcmv1alpha1.RemoveStatus{
		Name:      deployName,
		Status:    lcmv1alpha1.RemoveFinished,
		StartedAt: lcmcommon.GetCurrentTimeString(),
	}
	c.log.Info().Msgf("removing osd deployment '%s'", deployName)
	_, err := lcmcommon.RunFuncWithRetry(retriesForFailedCommand, commandRetryRunTimeout, func() (interface{}, error) {
		apiErr := c.removeOsdDeployment(deployName)
		if apiErr != nil {
			c.log.Error().Err(apiErr).Msg("")
		}
		return nil, apiErr
	})
	if err != nil {
		c.log.Error().Err(err).Msg("")
		deployRemoveStatus.Status = lcmv1alpha1.RemoveFailed
		deployRemoveStatus.Error = err.Error()
	} else {
		deployRemoveStatus.FinishedAt = lcmcommon.GetCurrentTimeString()
	}
	return deployRemoveStatus
}

func (c *cephOsdRemoveConfig) removeStray(osdID string, curRemoveStatus *lcmv1alpha1.RemoveStatus) *lcmv1alpha1.RemoveStatus {
	c.log.Info().Msgf("trying to cleanup stray osd '%s'", osdID)
	osdID = strings.Split(osdID, ".")[0]
	dropOsdEntriesAllowed := false
	// since our stray osd (with its uuid) is not in crush map, but another alive osd
	// with the same id and different uuid may be in crush - do not allow keyring/deployment clean
	_, err := lcmcommon.RunFuncWithRetry(retriesForFailedCommand, commandRetryRunTimeout, func() (interface{}, error) {
		osdLsOutput, cmdErr := lcmcommon.RunCephToolboxCLI(c.context, c.api.Kubeclientset, c.api.Config, c.taskConfig.cephCluster.Namespace, "ceph osd ls")
		if cmdErr != nil {
			c.log.Error().Err(cmdErr).Msg("")
			return false, cmdErr
		}
		dropOsdEntriesAllowed = !lcmcommon.Contains(strings.Split(osdLsOutput, "\n"), osdID)
		return true, nil
	})
	if err == nil {
		if dropOsdEntriesAllowed {
			curRemoveStatus.StartedAt = lcmcommon.GetCurrentTimeString()
			c.log.Info().Msgf("scaling down deployment for stray osd '%s' if present", osdID)
			_, err = lcmcommon.RunFuncWithRetry(retriesForFailedCommand, commandRetryRunTimeout, func() (interface{}, error) {
				scaleErr := lcmcommon.ScaleDeployment(c.context, c.api.Kubeclientset, fmt.Sprintf("rook-ceph-osd-%s", osdID), c.taskConfig.cephCluster.Namespace, 0)
				if scaleErr != nil {
					if apierrors.IsNotFound(scaleErr) {
						return nil, nil
					}
					c.log.Error().Err(scaleErr).Msg("")
				}
				return nil, scaleErr
			})
			if err == nil {
				c.log.Info().Msgf("dropping stray osd '%s' auth keyring", osdID)
				_, err = lcmcommon.RunFuncWithRetry(retriesForFailedCommand, commandRetryRunTimeout, func() (interface{}, error) {
					_, cmdErr := lcmcommon.RunCephToolboxCLI(c.context, c.api.Kubeclientset, c.api.Config, c.taskConfig.cephCluster.Namespace, fmt.Sprintf("ceph auth del osd.%s", osdID))
					if cmdErr != nil {
						c.log.Error().Err(cmdErr).Msg("")
					}
					return false, cmdErr
				})
				if err == nil {
					curRemoveStatus.Status = lcmv1alpha1.RemoveFinished
					curRemoveStatus.FinishedAt = lcmcommon.GetCurrentTimeString()
				}
			}
		} else {
			c.log.Info().Msgf("found osd with different uuid and same id '%s' in crush map, deployment and keyring cleanup skipped", osdID)
			curRemoveStatus.Status = lcmv1alpha1.RemoveSkipped
		}
	}
	if err != nil {
		c.log.Error().Err(err).Msg("")
		curRemoveStatus.Status = lcmv1alpha1.RemoveFailed
		curRemoveStatus.Error = err.Error()
	}
	return curRemoveStatus
}

func (c *cephOsdRemoveConfig) removeFromCrush(osdID string, curRemoveStatus *lcmv1alpha1.RemoveStatus) *lcmv1alpha1.RemoveStatus {
	if isStrayOsdID(osdID) {
		osdID = strings.Split(osdID, ".")[0]
	}
	c.log.Info().Msgf("trying to scale down deployment for osd '%s'", osdID)
	_, err := lcmcommon.RunFuncWithRetry(retriesForFailedCommand, commandRetryRunTimeout, func() (interface{}, error) {
		scaleErr := lcmcommon.ScaleDeployment(c.context, c.api.Kubeclientset, fmt.Sprintf("rook-ceph-osd-%s", osdID), c.taskConfig.cephCluster.Namespace, 0)
		if scaleErr != nil {
			if apierrors.IsNotFound(scaleErr) {
				return nil, nil
			}
			c.log.Error().Err(scaleErr).Msg("")
		}
		return nil, scaleErr
	})
	if err == nil {
		c.log.Info().Msgf("removing osd '%s' from crush map", osdID)
		_, err = lcmcommon.RunFuncWithRetry(retriesForFailedCommand, commandRetryRunTimeout, func() (interface{}, error) {
			_, cmdErr := lcmcommon.RunCephToolboxCLI(c.context, c.api.Kubeclientset, c.api.Config, c.taskConfig.cephCluster.Namespace, fmt.Sprintf("ceph osd purge %s --force --yes-i-really-mean-it", osdID))
			if cmdErr != nil {
				c.log.Error().Err(cmdErr).Msg("")
			}
			return false, cmdErr
		})
		if err == nil {
			c.log.Info().Msgf("dropping osd '%s' auth keyring", osdID)
			_, err = lcmcommon.RunFuncWithRetry(retriesForFailedCommand, commandRetryRunTimeout, func() (interface{}, error) {
				_, cmdErr := lcmcommon.RunCephToolboxCLI(c.context, c.api.Kubeclientset, c.api.Config, c.taskConfig.cephCluster.Namespace, fmt.Sprintf("ceph auth del osd.%s", osdID))
				if cmdErr != nil {
					c.log.Error().Err(cmdErr).Msg("")
				}
				return false, cmdErr
			})
		}
	}
	if err != nil {
		c.log.Error().Err(err).Msg("")
		curRemoveStatus.Status = lcmv1alpha1.RemoveFailed
		curRemoveStatus.Error = err.Error()
	} else {
		curRemoveStatus.Status = lcmv1alpha1.RemoveFinished
		curRemoveStatus.FinishedAt = lcmcommon.GetCurrentTimeString()
	}
	return curRemoveStatus
}

func (c *cephOsdRemoveConfig) checkRebalance(osdID string, curRemoveStatus *lcmv1alpha1.RemoveStatus) *lcmv1alpha1.RemoveStatus {
	c.log.Info().Msgf("checking rebalance completed for osd '%s'", osdID)
	pgsForOsdPresent, err := lcmcommon.RunFuncWithRetry(retriesForFailedCommand, commandRetryRunTimeout, func() (interface{}, error) {
		return c.checkPgsForOsd(osdID)
	})
	if err != nil {
		c.log.Error().Err(err).Msg("")
		curRemoveStatus.Status = lcmv1alpha1.RemoveFailed
		curRemoveStatus.Error = err.Error()
	} else {
		if pgsForOsdPresent.(bool) {
			timeStart, err := time.Parse(time.RFC3339, curRemoveStatus.StartedAt)
			// should not happen, but avoid any unexpected errors
			if err != nil {
				c.log.Error().Err(err).Msgf("incorrect timestamp value for osd '%s' startedAt field, expected RFC3339 format", osdID)
				return curRemoveStatus
			}
			if waitLeft := c.lcmConfig.TaskParams.OsdPgRebalanceTimeout.Minutes() - time.Since(timeStart).Minutes(); waitLeft > 0 {
				c.log.Info().Msgf("rebalance is not finished for osd '%s', waiting within next %v mins", osdID, waitLeft)
				return curRemoveStatus
			}
			msg := fmt.Sprintf("timeout (%v) reached for waiting pg rebalance", c.lcmConfig.TaskParams.OsdPgRebalanceTimeout)
			c.log.Error().Msgf("%s for osd '%s', aborting", msg, osdID)
			curRemoveStatus.Status = lcmv1alpha1.RemoveFailed
			curRemoveStatus.Error = msg
		} else {
			c.log.Info().Msgf("rebalance finished for osd '%s'", osdID)
			curRemoveStatus.Status = lcmv1alpha1.RemoveInProgress
		}
	}
	return curRemoveStatus
}

func (c *cephOsdRemoveConfig) tryToMoveOsdOut(osdID string, curRemoveStatus *lcmv1alpha1.RemoveStatus) *lcmv1alpha1.RemoveStatus {
	osdInfoOut, err := lcmcommon.RunFuncWithRetry(retriesForFailedCommand, commandRetryRunTimeout, func() (interface{}, error) {
		return c.getOsdInfo(osdID)
	})
	if err != nil {
		c.log.Error().Err(err).Msg("")
		return &lcmv1alpha1.RemoveStatus{
			Status: lcmv1alpha1.RemoveFailed,
			Error:  err.Error(),
		}
	}
	osdInfo := osdInfoOut.(lcmcommon.OsdInfo)
	status := &lcmv1alpha1.RemoveStatus{Status: lcmv1alpha1.RemoveInProgress}
	if osdInfo.In == 1 {
		c.log.Info().Msgf("trying to out osd '%s'", osdID)
		if osdInfo.Up == 1 {
			status.Status = lcmv1alpha1.RemoveWaitingRebalance
			cmd := fmt.Sprintf("ceph osd ok-to-stop %s", osdID)
			_, err := lcmcommon.RunCephToolboxCLI(c.context, c.api.Kubeclientset, c.api.Config, c.taskConfig.cephCluster.Namespace, cmd)
			if err != nil {
				c.log.Error().Err(err).Msg("")
				waitLeft := c.lcmConfig.TaskParams.OsdPgRebalanceTimeout.Minutes()
				if curRemoveStatus != nil && curRemoveStatus.StartedAt != "" {
					timeStart, err := time.Parse(time.RFC3339, curRemoveStatus.StartedAt)
					// should not happen, but avoid any unexpected errors
					if err != nil {
						c.log.Error().Err(err).Msgf("incorrect timestamp value for osd '%s' startedAt field, expected RFC3339 format", osdID)
						return curRemoveStatus
					}
					waitLeft = c.lcmConfig.TaskParams.OsdPgRebalanceTimeout.Minutes() - time.Since(timeStart).Minutes()
				}
				if waitLeft > 0 {
					c.log.Warn().Msgf("can't stop osd '%s', retrying within next %v mins", osdID, waitLeft)
					status = &lcmv1alpha1.RemoveStatus{Status: lcmv1alpha1.RemovePending}
					if curRemoveStatus == nil || curRemoveStatus.StartedAt == "" {
						status.StartedAt = lcmcommon.GetCurrentTimeString()
					}
					return status
				}
				status.Status = lcmv1alpha1.RemoveFailed
				status.Error = fmt.Sprintf("timeout (%v) reached for waiting ok-to-stop on osd '%s'", c.lcmConfig.TaskParams.OsdPgRebalanceTimeout, osdID)
				c.log.Error().Msgf("%s, aborting", status.Error)
				return status
			}
		}
		_, err := lcmcommon.RunFuncWithRetry(retriesForFailedCommand, commandRetryRunTimeout, func() (interface{}, error) {
			_, cmdErr := lcmcommon.RunCephToolboxCLI(c.context, c.api.Kubeclientset, c.api.Config, c.taskConfig.cephCluster.Namespace, fmt.Sprintf("ceph osd crush reweight osd.%s 0.0", osdID))
			if cmdErr != nil {
				c.log.Error().Err(cmdErr).Msg("")
			}
			return false, cmdErr
		})
		if err != nil {
			status.Status = lcmv1alpha1.RemoveFailed
			status.Error = err.Error()
			return status
		}
	} else if osdInfo.Up == 1 {
		c.log.Info().Msgf("osd '%s' is already not in", osdID)
		status.Status = lcmv1alpha1.RemoveWaitingRebalance
	} else {
		c.log.Info().Msgf("osd '%s' is already not in and not up", osdID)
	}
	status.StartedAt = lcmcommon.GetCurrentTimeString()
	return status
}

// check can we run job right now and get device mapping with actual disk zapping info
func (c *cephOsdRemoveConfig) getJobData(curOsdID string, hostOsdMapping map[string]lcmv1alpha1.OsdMapping) (bool, map[string]lcmv1alpha1.DeviceInfo) {
	parallelJobDetected := false
	curOsdDeviceMap := map[string]lcmv1alpha1.DeviceInfo{}
	for device, info := range hostOsdMapping[curOsdID].DeviceMapping {
		curOsdDeviceMap[device] = info
	}
	for osdID, osdMapping := range hostOsdMapping {
		if osdID == curOsdID {
			continue
		}
		if osdMapping.RemoveStatus.DeviceCleanUpJob != nil && osdMapping.RemoveStatus.DeviceCleanUpJob.Status == lcmv1alpha1.RemoveInProgress {
			parallelJobDetected = true
			break
		}
		for device := range osdMapping.DeviceMapping {
			if v, ok := curOsdDeviceMap[device]; ok {
				if v.Zap {
					reason := "which is not cleaned up yet"
					if osdMapping.SkipDeviceCleanupJob {
						reason = "which is not going to be clean up"
					} else if osdMapping.RemoveStatus.DeviceCleanUpJob != nil {
						if osdMapping.RemoveStatus.DeviceCleanUpJob.Status == lcmv1alpha1.RemoveCompleted ||
							osdMapping.RemoveStatus.DeviceCleanUpJob.Status == lcmv1alpha1.RemoveSkipped {
							continue
						} else if osdMapping.RemoveStatus.DeviceCleanUpJob.Status == lcmv1alpha1.RemoveFailed {
							reason = "which has failed cleanup job"
						}
					}
					c.log.Info().Msgf("disabling disk '%s' zapping for osd '%s', since disk is used for osd '%s' on the same host, %s", device, curOsdID, osdID, reason)
					v.Zap = false
					curOsdDeviceMap[device] = v
				}
			}
		}
	}
	return parallelJobDetected, curOsdDeviceMap
}

func (c *cephOsdRemoveConfig) handleJobRun(osdID, host string, hostOsdMapping map[string]lcmv1alpha1.OsdMapping) *lcmv1alpha1.RemoveStatus {
	curStatus := hostOsdMapping[osdID].RemoveStatus.DeviceCleanUpJob
	if curStatus == nil || curStatus.Status == lcmv1alpha1.RemovePending {
		parallelJobDetected, jobData := c.getJobData(osdID, hostOsdMapping)
		if parallelJobDetected {
			c.log.Info().Msgf("delaying disk cleanup job for node '%s' for devices associated with osd '%s': detected running job on the same host", host, osdID)
			return &lcmv1alpha1.RemoveStatus{Status: lcmv1alpha1.RemovePending}
		}
		if len(jobData) == 0 {
			c.log.Info().Msgf("skipping disk cleanup job for node '%s': no available devices for clean up", host)
			return &lcmv1alpha1.RemoveStatus{Status: lcmv1alpha1.RemoveSkipped}
		}
		c.log.Info().Msgf("running cleanup job for devices related to osd '%s' on node '%s'", osdID, host)
		jobName, err := c.runCleanupJob(host, osdID, hostOsdMapping[osdID].HostDirectory, jobData)
		if err != nil {
			c.log.Error().Err(err).Msg(jobName)
			return &lcmv1alpha1.RemoveStatus{
				Name:   jobName,
				Status: lcmv1alpha1.RemoveFailed,
				Error:  fmt.Sprintf("failed to run job: %v", err),
			}
		}
		return &lcmv1alpha1.RemoveStatus{
			Name:      jobName,
			Status:    lcmv1alpha1.RemoveInProgress,
			StartedAt: lcmcommon.GetCurrentTimeString(),
		}
	}

	c.log.Info().Msgf("checking device cleanup job '%s'", curStatus.Name)
	job, err := c.getCleanupJob(curStatus.Name)
	if err != nil {
		c.log.Error().Err(err).Msg("")
		curStatus.Error = fmt.Sprintf("failed to get job info: %v", err)
		curStatus.Status = lcmv1alpha1.RemoveFailed
	} else {
		if job.Status.Active > 0 {
			// if in progress do nothing
			c.log.Info().Msgf("device cleanup job '%s' is still running", curStatus.Name)
			return curStatus
		}
		if job.Status.Failed > 0 || lcmcommon.JobConditionsFailed(job.Status) {
			c.log.Error().Msgf("device cleanup job '%s' has failed", curStatus.Name)
			curStatus.Error = "job failed, check logs"
			curStatus.Status = lcmv1alpha1.RemoveFailed
		} else if job.Status.Succeeded > 0 {
			c.log.Info().Msgf("device cleanup job '%s' has been completed", curStatus.Name)
			curStatus.Status = lcmv1alpha1.RemoveCompleted
			curStatus.FinishedAt = lcmcommon.GetCurrentTimeString()
		} else {
			c.log.Error().Msgf("device cleanup job '%s' is pending", curStatus.Name)
		}
	}
	return curStatus
}
