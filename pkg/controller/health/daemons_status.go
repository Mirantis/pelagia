/*
Copyright 2025 Mirantis IT.

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

package health

import (
	"fmt"
	"sort"
	"strings"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	lcmv1alpha1 "github.com/Mirantis/pelagia/pkg/apis/ceph.pelagia.lcm/v1alpha1"
	lcmcommon "github.com/Mirantis/pelagia/pkg/common"
)

func (c *cephDeploymentHealthConfig) daemonsStatusVerification() (*lcmv1alpha1.CephDaemonsStatus, []string) {
	newDaemonsStatus := &lcmv1alpha1.CephDaemonsStatus{}
	daemonsIssues := []string{}

	cephDaemonsStatus, cephDaemonsIssues := c.getCephDaemonsStatus()
	if len(cephDaemonsIssues) > 0 {
		daemonsIssues = append(daemonsIssues, cephDaemonsIssues...)
	}
	newDaemonsStatus.CephDaemons = cephDaemonsStatus

	cephCSIPluginDaemonsStatus, cephCSIIssues := c.getCSIPluginDaemonsStatus()
	if len(cephCSIIssues) > 0 {
		daemonsIssues = append(daemonsIssues, cephCSIIssues...)
	}
	newDaemonsStatus.CephCSIPluginDaemons = cephCSIPluginDaemonsStatus
	// to avoid api diff since section is optional and omit empty set
	if cephDaemonsStatus == nil && cephCSIPluginDaemonsStatus == nil {
		newDaemonsStatus = nil
	}
	sort.Strings(daemonsIssues)
	return newDaemonsStatus, daemonsIssues
}

func (c *cephDeploymentHealthConfig) getCephDaemonsStatus() (map[string]lcmv1alpha1.DaemonStatus, []string) {
	if lcmcommon.Contains(c.lcmConfig.HealthParams.ChecksSkip, cephDaemonsCheck) {
		c.log.Debug().Msgf("skipping ceph daemons state check, set '%s' to skip through lcm config settings", cephDaemonsCheck)
		return nil, nil
	}
	var cephStatus lcmcommon.CephStatus
	cmd := "ceph status -f json"
	err := lcmcommon.RunAndParseCephToolboxCLI(c.context, c.api.Kubeclientset, c.api.Config, c.lcmConfig.RookNamespace, cmd, &cephStatus)
	if err != nil {
		c.log.Error().Err(err).Msg("")
		return nil, []string{fmt.Sprintf("failed to run '%s' command to check daemons status", cmd)}
	}

	var cephMgrDump mgrDump
	cmd = "ceph mgr dump -f json"
	err = lcmcommon.RunAndParseCephToolboxCLI(c.context, c.api.Kubeclientset, c.api.Config, c.lcmConfig.RookNamespace, cmd, &cephMgrDump)
	if err != nil {
		c.log.Error().Err(err).Msg("")
		return nil, []string{fmt.Sprintf("failed to run '%s' command to check daemons status", cmd)}
	}

	daemonsIssues := make([]string, 0)
	daemonsStatus := map[string]lcmv1alpha1.DaemonStatus{}
	// check osds daemons
	// expected/running osds will be checked separately as part of storage spec analysis
	osdDaemonStatus := lcmv1alpha1.DaemonStatus{
		Status:   lcmv1alpha1.DaemonStateOk,
		Messages: []string{fmt.Sprintf("%d osds, %d up, %d in", cephStatus.OsdMap.NumOsd, cephStatus.OsdMap.NumUpOsd, cephStatus.OsdMap.NumInOsd)},
	}
	if cephStatus.OsdMap.NumInOsd < cephStatus.OsdMap.NumOsd {
		osdDaemonStatus.Issues = append(osdDaemonStatus.Issues, "not all osds are in")
	}
	if cephStatus.OsdMap.NumUpOsd < cephStatus.OsdMap.NumOsd {
		osdDaemonStatus.Issues = append(osdDaemonStatus.Issues, "not all osds are up")
	}
	if len(osdDaemonStatus.Issues) > 0 {
		sort.Strings(osdDaemonStatus.Issues)
		osdDaemonStatus.Status = lcmv1alpha1.DaemonStateFailed
		daemonsIssues = append(daemonsIssues, osdDaemonStatus.Issues...)
	}
	daemonsStatus["osd"] = osdDaemonStatus

	// check expected/running mons
	actualMonsRunning := len(cephStatus.QuorumNames)
	monsTarget := cephStatus.MonMap.NumMons
	monDaemonsStatus := lcmv1alpha1.DaemonStatus{
		Status:   lcmv1alpha1.DaemonStateOk,
		Messages: []string{fmt.Sprintf("%d mons, quorum %v", actualMonsRunning, cephStatus.QuorumNames)},
	}
	if !c.healthConfig.cephCluster.Spec.External.Enable {
		expectedMons := c.healthConfig.cephCluster.Spec.Mon.Count
		if expectedMons > monsTarget {
			monDaemonsStatus.Issues = append(monDaemonsStatus.Issues, fmt.Sprintf("not all (%d/%d) mons are deployed", monsTarget, expectedMons))
		} else if expectedMons < monsTarget {
			monDaemonsStatus.Issues = append(monDaemonsStatus.Issues, fmt.Sprintf("unexpected (%d/%d) mons are deployed", monsTarget, expectedMons))
		}
	}
	if actualMonsRunning < monsTarget {
		monDaemonsStatus.Issues = append(monDaemonsStatus.Issues, fmt.Sprintf("not all (%d/%d) mons are running", actualMonsRunning, monsTarget))
	}
	if len(monDaemonsStatus.Issues) > 0 {
		sort.Strings(monDaemonsStatus.Issues)
		monDaemonsStatus.Status = lcmv1alpha1.DaemonStateFailed
		daemonsIssues = append(daemonsIssues, monDaemonsStatus.Issues...)
	}
	daemonsStatus["mon"] = monDaemonsStatus

	// check expected/running mgrs
	mgrDaemonsStatus := lcmv1alpha1.DaemonStatus{
		Status: lcmv1alpha1.DaemonStateOk,
	}
	if cephMgrDump.Available {
		actualMgrs := 1 + len(cephMgrDump.Standbys)
		if !c.healthConfig.cephCluster.Spec.External.Enable {
			if actualMgrs < c.healthConfig.cephCluster.Spec.Mgr.Count {
				mgrDaemonsStatus.Issues = []string{fmt.Sprintf("not all mgrs (%d/%d) running", actualMgrs, c.healthConfig.cephCluster.Spec.Mgr.Count)}
			} else if actualMgrs > c.healthConfig.cephCluster.Spec.Mgr.Count {
				mgrDaemonsStatus.Issues = []string{fmt.Sprintf("unexpected mgrs (%d/%d) running", actualMgrs, c.healthConfig.cephCluster.Spec.Mgr.Count)}
			}
		}

		standByMgrsStr := func(mgrs []mgrStandby) string {
			if len(mgrs) == 0 {
				return ""
			}
			var result []string
			for _, mgr := range mgrs {
				result = append(result, mgr.Name)
			}
			return fmt.Sprintf(", standbys: %v", result)
		}(cephMgrDump.Standbys)

		mgrDaemonsStatus.Messages = []string{fmt.Sprintf("%s is active mgr%s", cephMgrDump.ActiveName, standByMgrsStr)}
	} else {
		mgrDaemonsStatus.Issues = []string{"no active mgr"}
	}
	if len(mgrDaemonsStatus.Issues) > 0 {
		sort.Strings(mgrDaemonsStatus.Issues)
		mgrDaemonsStatus.Status = lcmv1alpha1.DaemonStateFailed
		daemonsIssues = append(daemonsIssues, mgrDaemonsStatus.Issues...)
	}
	daemonsStatus["mgr"] = mgrDaemonsStatus

	// check expected/running rgws
	expectedRgws := int(c.healthConfig.rgwOpts.desiredRgwDaemons)
	actualRgws := 0
	namesRgws := make([]string, 0)
	for rgw := range cephStatus.ServiceMap.Services.Rgw.Daemons {
		if rgw != "summary" {
			actualRgws++
			namesRgws = append(namesRgws, rgw)
		}
	}
	if actualRgws > 0 || expectedRgws > 0 || c.healthConfig.rgwOpts.external {
		sort.Strings(namesRgws)
		rgwDaemonsStatus := lcmv1alpha1.DaemonStatus{
			Status:   lcmv1alpha1.DaemonStateOk,
			Messages: []string{fmt.Sprintf("%d rgws running, daemons: %v", actualRgws, namesRgws)},
		}
		// check cephcluster external, since we may have rgw on another side,
		// but external ceph cluster has no rgw running, just put overall daemon info
		if c.healthConfig.cephCluster.Spec.External.Enable {
			if actualRgws == 0 {
				rgwDaemonsStatus.Status = lcmv1alpha1.DaemonStateFailed
				rgwDaemonsStatus.Issues = append(rgwDaemonsStatus.Issues, "no rgws are running")
				daemonsIssues = append(daemonsIssues, rgwDaemonsStatus.Issues...)
			}
			daemonsStatus["rgw"] = rgwDaemonsStatus
		} else {
			if expectedRgws != actualRgws {
				if actualRgws < expectedRgws {
					rgwDaemonsStatus.Issues = append(rgwDaemonsStatus.Issues, fmt.Sprintf("not all (%d/%d) rgws are running", actualRgws, expectedRgws))
				} else {
					rgwDaemonsStatus.Issues = append(rgwDaemonsStatus.Issues, fmt.Sprintf("unexpected rgws (%d/%d) rgws are running", actualRgws, expectedRgws))
				}
				if len(rgwDaemonsStatus.Issues) > 0 {
					sort.Strings(rgwDaemonsStatus.Issues)
					daemonsIssues = append(daemonsIssues, rgwDaemonsStatus.Issues...)
					rgwDaemonsStatus.Status = lcmv1alpha1.DaemonStateFailed
				}
			}
			daemonsStatus["rgw"] = rgwDaemonsStatus
		}
	}

	// check expected/running mds
	// in external case cephfs resources must be present on provider cluster side
	if !c.healthConfig.cephCluster.Spec.External.Enable {
		showMdsStatus := false
		mdsStandbyTotal := cephStatus.FsMap.Standby
		mdsDaemonsRunning := map[string]map[string]int{}
		for _, mdsInfo := range cephStatus.FsMap.ByRank {
			showMdsStatus = true
			tmp := strings.Split(mdsInfo.Name, "-")
			cephfsName := strings.Join(tmp[:len(tmp)-1], "-")
			if _, ok := mdsDaemonsRunning[cephfsName]; ok {
				mdsDaemonsRunning[cephfsName][mdsInfo.Status]++
			} else {
				mdsDaemonsRunning[cephfsName] = map[string]int{
					mdsInfo.Status: 1,
				}
			}
		}
		mdsStandbyExpected := c.healthConfig.sharedFilesystemOpts.mdsStandbyDesired
		mdsDaemonsExpected := c.healthConfig.sharedFilesystemOpts.mdsDaemonsDesired
		if showMdsStatus || mdsStandbyExpected > 0 || len(mdsDaemonsExpected) > 0 {
			mdsDaemonsStatus := lcmv1alpha1.DaemonStatus{
				Status:   lcmv1alpha1.DaemonStateOk,
				Messages: []string{},
			}
			if mdsStandbyExpected != mdsStandbyTotal {
				mdsDaemonsStatus.Issues = append(mdsDaemonsStatus.Issues, fmt.Sprintf("unexpected number (%d/%d) of mds standby are running", mdsStandbyTotal, mdsStandbyExpected))
			}
			for cephfs := range mdsDaemonsRunning {
				if _, ok := mdsDaemonsExpected[cephfs]; !ok {
					c.log.Error().Msgf("detected mds daemons for unknown CephFS '%s'. Rook object CephFilesystem '%s/%s' is not exist", cephfs, c.lcmConfig.RookNamespace, cephfs)
					mdsDaemonsStatus.Issues = append(mdsDaemonsStatus.Issues, fmt.Sprintf("unexpected mds daemons running (CephFS '%s')", cephfs))
					continue
				}
				if mdsDaemonsExpected[cephfs]["up:active"] != mdsDaemonsRunning[cephfs]["up:active"] {
					mdsDaemonsStatus.Issues = append(mdsDaemonsStatus.Issues,
						fmt.Sprintf("unexpected number (%d/%d) of mds active are running for CephFS '%s'", mdsStandbyTotal, mdsStandbyExpected, cephfs))
				}
				if mdsDaemonsExpected[cephfs]["up:standby-replay"] != mdsDaemonsRunning[cephfs]["up:standby-replay"] {
					mdsDaemonsStatus.Issues = append(mdsDaemonsStatus.Issues,
						fmt.Sprintf("unexpected number (%d/%d) of mds standby-replay are running for CephFS '%s'", mdsStandbyTotal, mdsStandbyExpected, cephfs))
				}
				if mdsDaemonsExpected[cephfs]["up:standby-replay"] == 0 && mdsDaemonsRunning[cephfs]["up:standby-replay"] == 0 {
					mdsDaemonsStatus.Messages = append(mdsDaemonsStatus.Messages,
						fmt.Sprintf("mds active: %d/%d (cephfs '%s')", mdsDaemonsRunning[cephfs]["up:active"], mdsDaemonsExpected[cephfs]["up:active"], cephfs))
				} else {
					mdsDaemonsStatus.Messages = append(mdsDaemonsStatus.Messages,
						fmt.Sprintf("mds active: %d/%d, standby-replay: %d/%d (cephfs '%s')", mdsDaemonsRunning[cephfs]["up:active"], mdsDaemonsExpected[cephfs]["up:standby-replay"],
							mdsDaemonsRunning[cephfs]["up:standby-replay"], mdsDaemonsExpected[cephfs]["up:standby-replay"], cephfs))
				}
				delete(mdsDaemonsExpected, cephfs)
			}
			for cephfs := range mdsDaemonsExpected {
				mdsDaemonsStatus.Issues = append(mdsDaemonsStatus.Issues, fmt.Sprintf("mds daemons are not running (cephfs '%s')", cephfs))
			}
			if len(mdsDaemonsStatus.Issues) > 0 {
				sort.Strings(mdsDaemonsStatus.Issues)
				mdsDaemonsStatus.Status = lcmv1alpha1.DaemonStateFailed
				daemonsIssues = append(daemonsIssues, mdsDaemonsStatus.Issues...)
			}
			sort.Strings(mdsDaemonsStatus.Messages)
			daemonsStatus["mds"] = mdsDaemonsStatus
		}
	}

	if len(daemonsIssues) > 0 {
		sort.Strings(daemonsIssues)
		c.log.Error().Msgf("found some issue(s) with Ceph Daemons: [%s]", strings.Join(daemonsIssues, ", "))
	}
	return daemonsStatus, daemonsIssues
}

func (c *cephDeploymentHealthConfig) getCSIPluginDaemonsStatus() (map[string]lcmv1alpha1.DaemonStatus, []string) {
	if lcmcommon.Contains(c.lcmConfig.HealthParams.ChecksSkip, cephCSIPluginDaemonsCheck) {
		c.log.Debug().Msgf("skipping ceph csi plugin daemons check, set '%s' to skip through lcm config settings", cephCSIPluginDaemonsCheck)
		return nil, nil
	}
	rookOperatorMap, err := c.api.Kubeclientset.CoreV1().ConfigMaps(c.lcmConfig.RookNamespace).Get(c.context, lcmcommon.RookOperatorConfigMapName, metav1.GetOptions{})
	if err != nil {
		c.log.Error().Err(err).Msg("")
		return nil, []string{fmt.Sprintf("failed to get configmap '%s/%s'", c.lcmConfig.RookNamespace, lcmcommon.RookOperatorConfigMapName)}
	}
	csiPluginsStatus := map[string]lcmv1alpha1.DaemonStatus{}
	csiPluginsIssues := make([]string, 0)

	if rookOperatorMap.Data["ROOK_CSI_ENABLE_RBD"] == "true" {
		rbdDaemonStatus, _ := c.getDaemonSetStatus(c.lcmConfig.RookNamespace, lcmcommon.CephCSIRBDPluginDaemonSetName)
		if len(rbdDaemonStatus.Issues) > 0 {
			csiPluginsIssues = append(csiPluginsIssues, rbdDaemonStatus.Issues...)
		}
		csiPluginsStatus[lcmcommon.CephCSIRBDPluginDaemonSetName] = rbdDaemonStatus
	}

	if rookOperatorMap.Data["ROOK_CSI_ENABLE_CEPHFS"] == "true" {
		cephFSDaemonStatus, _ := c.getDaemonSetStatus(c.lcmConfig.RookNamespace, lcmcommon.CephCSICephFSPluginDaemonSetName)
		if len(cephFSDaemonStatus.Issues) > 0 {
			csiPluginsIssues = append(csiPluginsIssues, cephFSDaemonStatus.Issues...)
		}
		csiPluginsStatus[lcmcommon.CephCSICephFSPluginDaemonSetName] = cephFSDaemonStatus
	}

	return csiPluginsStatus, csiPluginsIssues
}

func (c *cephDeploymentHealthConfig) getDaemonSetStatus(daemonSetNamespace, daemonSetName string) (lcmv1alpha1.DaemonStatus, int) {
	daemonStatus := lcmv1alpha1.DaemonStatus{
		Status: lcmv1alpha1.DaemonStateFailed,
	}
	ds, err := c.api.Kubeclientset.AppsV1().DaemonSets(daemonSetNamespace).Get(c.context, daemonSetName, metav1.GetOptions{})
	if err != nil {
		c.log.Error().Err(err).Msg("")
		if apierrors.IsNotFound(err) {
			daemonStatus.Issues = []string{fmt.Sprintf("daemonset '%s/%s' is not found", daemonSetNamespace, daemonSetName)}
		} else {
			daemonStatus.Issues = []string{fmt.Sprintf("failed to get '%s/%s' daemonset", daemonSetNamespace, daemonSetName)}
		}
		return daemonStatus, 0
	}
	daemonStatus.Messages = []string{fmt.Sprintf("%d/%d ready", ds.Status.NumberReady, ds.Status.DesiredNumberScheduled)}
	if lcmcommon.IsDaemonSetReady(ds) {
		daemonStatus.Status = lcmv1alpha1.DaemonStateOk
	} else {
		daemonStatus.Issues = []string{fmt.Sprintf("daemonset '%s/%s' is not ready", daemonSetNamespace, daemonSetName)}
	}
	return daemonStatus, int(ds.Status.NumberReady)
}
