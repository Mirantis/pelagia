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
	"reflect"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	lcmv1alpha1 "github.com/Mirantis/pelagia/pkg/apis/ceph.pelagia.lcm/v1alpha1"
	lcmcommon "github.com/Mirantis/pelagia/pkg/common"
)

func (c *cephOsdRemoveConfig) handleTask() *lcmv1alpha1.CephOsdRemoveTaskStatus {
	// this should not happen ever - but to double check and avoid out of range error
	if len(c.taskConfig.task.Status.Conditions) == 0 {
		reason := "status conditions section unexpectedly missed, task should be re-created"
		c.log.Error().Msg(reason)
		return prepareAbortStatus(c.taskConfig.task.Status, reason)
	}

	specChanges := func() []string {
		reasons := []string{}
		// check prev state for detecting cephcluster changes and task spec nodes section changes
		latestCondition := c.taskConfig.task.Status.Conditions[len(c.taskConfig.task.Status.Conditions)-1]
		if latestCondition.CephClusterSpecVersion == nil || latestCondition.CephClusterSpecVersion.Generation != c.taskConfig.cephCluster.Generation {
			reasons = append(reasons, "CephCluster has a new generation version")
		}
		var currentNodes map[string]lcmv1alpha1.NodeCleanUpSpec
		if c.taskConfig.task.Spec != nil {
			currentNodes = c.taskConfig.task.Spec.Nodes
		}
		if !reflect.DeepEqual(latestCondition.Nodes, currentNodes) {
			reasons = append(reasons, "task has changed nodes section")
		}
		return reasons
	}

	switch c.taskConfig.task.Status.Phase {
	case lcmv1alpha1.TaskPhasePending:
		c.taskConfig.requeueNow = true
		c.log.Info().Msg("ready to validation")
		return c.taskConfig.moveTaskPhase(lcmv1alpha1.TaskPhaseValidating, "validation", nil)
	case lcmv1alpha1.TaskPhaseValidating:
		if *c.taskConfig.cephHealthOsdAnalysis.CephClusterSpecGeneration != c.taskConfig.cephCluster.Generation {
			c.log.Info().Msgf("related CephDeploymentHealth has not validated yet latest CephCluster spec (validated: %d, current: %d)",
				*c.taskConfig.cephHealthOsdAnalysis.CephClusterSpecGeneration, c.taskConfig.cephCluster.Generation)
			break
		}
		validationRes := c.validateTask()
		if len(validationRes.Issues) == 0 {
			if len(validationRes.CleanupMap) == 0 {
				msg := "validation completed, nothing to remove"
				c.log.Info().Msg(msg)
				return c.taskConfig.moveTaskPhase(lcmv1alpha1.TaskPhaseCompleted, msg, validationRes)
			}
			if c.taskConfig.task.Spec != nil && c.taskConfig.task.Spec.Approve {
				c.taskConfig.requeueNow = true
				msg := "validation completed, approve pre-set"
				c.log.Info().Msg(msg)
				return c.taskConfig.moveTaskPhase(lcmv1alpha1.TaskPhaseWaitingOperator, msg, validationRes)
			}
			msg := "validation completed, waiting approve"
			c.log.Info().Msg(msg)
			return c.taskConfig.moveTaskPhase(lcmv1alpha1.TaskPhaseApproveWaiting, msg, validationRes)
		}
		c.log.Error().Msgf("validation failed, found next issues: %s", strings.Join(validationRes.Issues, ","))
		return c.taskConfig.moveTaskPhase(lcmv1alpha1.TaskPhaseValidationFailed, "validation failed", validationRes)
	case lcmv1alpha1.TaskPhaseApproveWaiting:
		if c.taskConfig.task.Spec != nil && c.taskConfig.task.Spec.Approve {
			c.log.Info().Msg("approve received")
			c.taskConfig.requeueNow = true
			return c.taskConfig.moveTaskPhase(lcmv1alpha1.TaskPhaseWaitingOperator, "approve received, wait rook-operator stop", c.taskConfig.task.Status.RemoveInfo)
		}
		// check no changes in ceph cluster before approve received
		if reasonsToRevalidate := specChanges(); len(reasonsToRevalidate) > 0 {
			c.log.Info().Msgf("revalidation required due to %s", strings.Join(reasonsToRevalidate, ", "))
			c.taskConfig.requeueNow = true
			return c.taskConfig.moveTaskPhase(lcmv1alpha1.TaskPhaseValidating, "revalidation triggered", nil)
		}
		c.log.Info().Msg("waiting for approve")
	case lcmv1alpha1.TaskPhaseWaitingOperator:
		// check no changes in ceph cluster after approve received
		// otherwise abort current task
		if reasonsToAbort := specChanges(); len(reasonsToAbort) > 0 {
			c.log.Error().Msgf("aborting, %s", strings.Join(reasonsToAbort, ","))
			return c.taskConfig.moveTaskPhase(lcmv1alpha1.TaskPhaseAborted, "detected inappropriate spec changes after receiving approval", nil)
		}
		if c.checkOperatorStopped() {
			c.log.Info().Msg("rook-operator is shutted down")
			c.taskConfig.requeueNow = true
			return c.taskConfig.moveTaskPhase(lcmv1alpha1.TaskPhaseProcessing, "processing", c.taskConfig.task.Status.RemoveInfo)
		}
		c.log.Info().Msg("waiting for rook-operator is shutted down for task processing")
	case lcmv1alpha1.TaskPhaseProcessing:
		finished, processingRes := c.processTask()
		if !finished {
			newStatus := c.taskConfig.task.Status.DeepCopy()
			newStatus.RemoveInfo = processingRes
			return newStatus
		}
		if len(processingRes.Issues) == 0 {
			phase := lcmv1alpha1.TaskPhaseCompleted
			if len(processingRes.Warnings) > 0 {
				phase = lcmv1alpha1.TaskPhaseCompletedWithWarnings
			}
			return c.taskConfig.moveTaskPhase(phase, "osd remove completed", processingRes)
		}
		c.log.Error().Msgf("processing failed with next issues: %s", strings.Join(processingRes.Issues, ","))
		return c.taskConfig.moveTaskPhase(lcmv1alpha1.TaskPhaseFailed, "osd remove failed", processingRes)
	}
	return c.taskConfig.task.Status
}

// validateTask function check what could be removed from cluster
// if task.Spec.Nodes are not specified or checking that provided spec is executable
// and then return full info
func (c *cephOsdRemoveConfig) validateTask() *lcmv1alpha1.TaskRemoveInfo {
	// get main info required for validation info before actual validation
	clusterHostList, err := c.getOsdHostsFromCluster()
	if err != nil {
		errMsg := fmt.Sprintf("failed to get ceph cluster nodes list: %v", err)
		return &lcmv1alpha1.TaskRemoveInfo{Issues: []string{errMsg}}
	}
	var clusterOsdsMetadata []lcmcommon.OsdMetadataInfo
	err = lcmcommon.RunAndParseCephToolboxCLI(c.context, c.api.Kubeclientset, c.api.Config, c.taskConfig.cephCluster.Namespace, "ceph osd metadata -f json", &clusterOsdsMetadata)
	if err != nil {
		c.log.Error().Err(err).Msg("")
		errMsg := fmt.Sprintf("failed to get ceph osd metadata info: %v", err)
		return &lcmv1alpha1.TaskRemoveInfo{Issues: []string{errMsg}}
	}
	nodesList, err := lcmcommon.GetNodeList(c.context, c.api.Kubeclientset, metav1.ListOptions{})
	if err != nil {
		c.log.Error().Err(err).Msg("")
		errMsg := fmt.Sprintf("failed to get k8s nodes list: %v", err)
		return &lcmv1alpha1.TaskRemoveInfo{Issues: []string{errMsg}}
	}
	var osdsInfo []lcmcommon.OsdInfo
	err = lcmcommon.RunAndParseCephToolboxCLI(c.context, c.api.Kubeclientset, c.api.Config, c.taskConfig.cephCluster.Namespace, "ceph osd info -f json", &osdsInfo)
	if err != nil {
		c.log.Error().Err(err).Msg("")
		errMsg := fmt.Sprintf("failed to get osds info: %v", err)
		return &lcmv1alpha1.TaskRemoveInfo{Issues: []string{errMsg}}
	}
	// check and validate provided nodes/devices/osds from spec otherwise compile all possible osds to remove
	if c.taskConfig.task.Spec != nil && len(c.taskConfig.task.Spec.Nodes) > 0 {
		c.log.Info().Msg("starting validation for provided hosts in spec")
	} else {
		c.log.Info().Msg("starting validation for possible hosts/osds to remove, no provided hosts in spec")
	}
	newRemoveInfo := c.getOsdsForCleanup(clusterHostList, clusterOsdsMetadata, osdsInfo, nodesList.Items)
	if len(newRemoveInfo.Issues) > 0 {
		c.log.Error().Msg("found issues during validation")
	} else if len(newRemoveInfo.CleanupMap) == 0 {
		c.log.Info().Msg("validated, nothing to remove")
	}
	return newRemoveInfo
}

// processTask handling actual osd removal
// returns whether task finished and updated remove info
func (c *cephOsdRemoveConfig) processTask() (bool, *lcmv1alpha1.TaskRemoveInfo) {
	if c.taskConfig.task.Status == nil || c.taskConfig.task.Status.RemoveInfo == nil {
		c.log.Error().Msg("unexpectedly empty status, aborting")
		return true, &lcmv1alpha1.TaskRemoveInfo{Issues: []string{"empty remove info, aborting"}}
	}
	c.log.Info().Msg("processing osd remove task")
	return c.processOsdRemoveTask()
}
