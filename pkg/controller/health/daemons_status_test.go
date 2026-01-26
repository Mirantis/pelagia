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
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"

	lcmv1alpha1 "github.com/Mirantis/pelagia/pkg/apis/ceph.pelagia.lcm/v1alpha1"
	lcmcommon "github.com/Mirantis/pelagia/pkg/common"
	faketestclients "github.com/Mirantis/pelagia/test/unit/clients"
	unitinputs "github.com/Mirantis/pelagia/test/unit/inputs"
)

func TestDaemonsStatusVerification(t *testing.T) {
	baseConfig := getEmtpyHealthConfig()
	baseConfig.cephCluster = &unitinputs.CephClusterReady
	tests := []struct {
		name           string
		inputResources map[string]runtime.Object
		skipChecks     bool
		cephStatus     string
		cephMgrDump    string
		expectedStatus *lcmv1alpha1.CephDaemonsStatus
		expectedIssues []string
	}{
		{
			name: "healthy daemons verification",
			inputResources: map[string]runtime.Object{
				"cephclusters":     &unitinputs.CephClusterListReady,
				"cephobjectstores": &unitinputs.CephObjectStoreListEmpty,
				"cephfilesystems":  &unitinputs.CephFilesystemListEmpty,
				"configmaps":       unitinputs.ConfigMapList,
				"daemonsets":       unitinputs.DaemonSetListReady,
			},
			cephStatus:     unitinputs.CephStatusBaseHealthy,
			cephMgrDump:    unitinputs.CephMgrDumpBaseHealthy,
			expectedStatus: unitinputs.CephDaemonsStatusHealthy,
			expectedIssues: []string{},
		},
		{
			name: "unhealthy daemons verification",
			inputResources: map[string]runtime.Object{
				"cephclusters":     &unitinputs.CephClusterListReady,
				"cephobjectstores": &unitinputs.CephObjectStoreListEmpty,
				"cephfilesystems":  &unitinputs.CephFilesystemListEmpty,
				"configmaps":       unitinputs.ConfigMapList,
				"daemonsets":       unitinputs.DaemonSetListNotReady,
			},
			cephStatus:     unitinputs.CephStatusBaseUnhealthy,
			cephMgrDump:    unitinputs.CephMgrDumpBaseUnhealthy,
			expectedStatus: unitinputs.CephDaemonsStatusUnhealthy,
			expectedIssues: []string{
				"daemonset 'rook-ceph/csi-cephfsplugin' is not ready", "daemonset 'rook-ceph/csi-rbdplugin' is not ready",
				"no active mgr", "not all (2/3) mons are running", "not all osds are in", "not all osds are up",
			},
		},
		{
			name:           "skip daemons verification",
			skipChecks:     true,
			expectedIssues: []string{},
		},
	}
	oldCephCmdFunc := lcmcommon.RunPodCommand
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			lcmConfigData := map[string]string{}
			if test.skipChecks {
				lcmConfigData["HEALTH_CHECKS_SKIP"] = "ceph_daemons,ceph_csi_plugin_daemons"
			}
			c := fakeCephReconcileConfig(&baseConfig, lcmConfigData)
			faketestclients.FakeReaction(c.api.Kubeclientset.CoreV1(), "list", []string{"pods"}, map[string]runtime.Object{"pods": unitinputs.ToolBoxPodList}, nil)
			faketestclients.FakeReaction(c.api.Kubeclientset.CoreV1(), "get", []string{"configmaps"}, test.inputResources, nil)
			faketestclients.FakeReaction(c.api.Kubeclientset.AppsV1(), "get", []string{"daemonsets"}, test.inputResources, nil)

			lcmcommon.RunPodCommand = func(e lcmcommon.ExecConfig) (string, string, error) {
				switch e.Command {
				case "ceph status -f json":
					if test.cephStatus != "" {
						return test.cephStatus, "", nil
					}
				case "ceph mgr dump -f json":
					if test.cephMgrDump != "" {
						return test.cephMgrDump, "", nil
					}
				}
				return "", "", errors.New("command failed")
			}

			status, issues := c.daemonsStatusVerification()
			assert.Equal(t, test.expectedStatus, status)
			assert.Equal(t, test.expectedIssues, issues)
			faketestclients.CleanupFakeClientReactions(c.api.Kubeclientset.CoreV1())
			faketestclients.CleanupFakeClientReactions(c.api.Kubeclientset.AppsV1())
		})
	}
	lcmcommon.RunPodCommand = oldCephCmdFunc
}

func TestGetCephDaemonsStatus(t *testing.T) {
	baseConfig := getEmtpyHealthConfig()
	baseConfig.cephCluster = &unitinputs.CephClusterReady
	tests := []struct {
		name           string
		cephStatus     string
		cephMgrDump    string
		healthConfig   healthConfig
		expectedStatus map[string]lcmv1alpha1.DaemonStatus
		expectedIssues []string
	}{
		{
			name:           "failed to get ceph status",
			healthConfig:   baseConfig,
			expectedIssues: []string{"failed to run 'ceph status -f json' command to check daemons status"},
		},
		{
			name:           "failed to get ceph mgr dump",
			healthConfig:   baseConfig,
			cephStatus:     unitinputs.CephStatusBaseHealthy,
			expectedIssues: []string{"failed to run 'ceph mgr dump -f json' command to check daemons status"},
		},
		{
			name:           "healthy daemons state, no extra daemons",
			healthConfig:   baseConfig,
			cephStatus:     unitinputs.CephStatusBaseHealthy,
			cephMgrDump:    unitinputs.CephMgrDumpBaseHealthy,
			expectedStatus: unitinputs.CephDaemonsBaseHealthy,
			expectedIssues: []string{},
		},
		{
			name: "healthy daemons state, mgr ha, no extra daemons",
			healthConfig: func() healthConfig {
				hc := getEmtpyHealthConfig()
				hc.cephCluster = unitinputs.CephClusterReady.DeepCopy()
				hc.cephCluster.Spec.Mgr.Count = 2
				return hc
			}(),
			cephStatus:  unitinputs.CephStatusBaseHealthy,
			cephMgrDump: unitinputs.CephMgrDumpHAHealthy,
			expectedStatus: func() map[string]lcmv1alpha1.DaemonStatus {
				return map[string]lcmv1alpha1.DaemonStatus{
					"osd": unitinputs.CephDaemonsBaseHealthy["osd"],
					"mon": unitinputs.CephDaemonsBaseHealthy["mon"],
					"mgr": {
						Status:   lcmv1alpha1.DaemonStateOk,
						Messages: []string{"a is active mgr, standbys: [b]"},
					},
				}
			}(),
			expectedIssues: []string{},
		},
		{
			name: "unhealthy daemons state, no extra daemons",
			healthConfig: func() healthConfig {
				hc := getEmtpyHealthConfig()
				hc.cephCluster = &unitinputs.CephClusterNotReady
				return hc
			}(),
			cephStatus:     unitinputs.CephStatusBaseUnhealthy,
			cephMgrDump:    unitinputs.CephMgrDumpBaseUnhealthy,
			expectedStatus: unitinputs.CephDaemonsBaseUnhealthy,
			expectedIssues: []string{"no active mgr", "not all (2/3) mons are running", "not all osds are in", "not all osds are up"},
		},
		{
			name: "unhealthy daemons state, unexpected base daemons count #1",
			healthConfig: func() healthConfig {
				hc := getEmtpyHealthConfig()
				hc.cephCluster = &unitinputs.CephClusterNotReady
				return hc
			}(),
			cephStatus:  unitinputs.BuildCliOutput(unitinputs.CephStatusTmpl, "status", map[string]string{"quorum_names": `["a", "b"]`, "monmap": `{"num_mons": 2}`}),
			cephMgrDump: unitinputs.CephMgrDumpHAHealthy,
			expectedStatus: func() map[string]lcmv1alpha1.DaemonStatus {
				return map[string]lcmv1alpha1.DaemonStatus{
					"osd": unitinputs.CephDaemonsBaseHealthy["osd"],
					"mon": {
						Status:   lcmv1alpha1.DaemonStateFailed,
						Messages: []string{"2 mons, quorum [a b]"},
						Issues:   []string{"not all (2/3) mons are deployed"},
					},
					"mgr": {
						Status:   lcmv1alpha1.DaemonStateFailed,
						Messages: []string{"a is active mgr, standbys: [b]"},
						Issues:   []string{"unexpected mgrs (2/1) running"},
					},
				}
			}(),
			expectedIssues: []string{"not all (2/3) mons are deployed", "unexpected mgrs (2/1) running"},
		},
		{
			name: "unhealthy daemons state, unexpected base daemons count #2",
			healthConfig: func() healthConfig {
				hc := getEmtpyHealthConfig()
				hc.cephCluster = unitinputs.CephClusterNotReady.DeepCopy()
				hc.cephCluster.Spec.Mgr.Count = 2
				return hc
			}(),
			cephStatus:  unitinputs.BuildCliOutput(unitinputs.CephStatusTmpl, "status", map[string]string{"monmap": `{"num_mons": 4}`}),
			cephMgrDump: unitinputs.CephMgrDumpHAUnealthy,
			expectedStatus: func() map[string]lcmv1alpha1.DaemonStatus {
				return map[string]lcmv1alpha1.DaemonStatus{
					"osd": unitinputs.CephDaemonsBaseHealthy["osd"],
					"mon": {
						Status:   lcmv1alpha1.DaemonStateFailed,
						Messages: []string{"3 mons, quorum [a b c]"},
						Issues:   []string{"not all (3/4) mons are running", "unexpected (4/3) mons are deployed"},
					},
					"mgr": {
						Status:   lcmv1alpha1.DaemonStateFailed,
						Messages: []string{"b is active mgr"},
						Issues:   []string{"not all mgrs (1/2) running"},
					},
				}
			}(),
			expectedIssues: []string{"not all (3/4) mons are running", "not all mgrs (1/2) running", "unexpected (4/3) mons are deployed"},
		},
		{
			name: "daemons healthy verification, cephfs, rgw daemons",
			healthConfig: func() healthConfig {
				hc := getEmtpyHealthConfig()
				hc.cephCluster = unitinputs.CephClusterReady.DeepCopy()
				hc.rgwOpts.storeName = "rgw-store"
				hc.rgwOpts.desiredRgwDaemons = 2
				hc.sharedFilesystemOpts.mdsDaemonsDesired["cephfs-1"] = map[string]int{"up:active": 1}
				hc.sharedFilesystemOpts.mdsStandbyDesired = 1
				return hc
			}(),
			cephStatus:     unitinputs.CephStatusCephFsRgwHealthy,
			cephMgrDump:    unitinputs.CephMgrDumpBaseHealthy,
			expectedStatus: unitinputs.CephDaemonsCephFsRgwHealthy,
			expectedIssues: make([]string, 0),
		},
		{
			name: "daemons unhealthy verification, cephfs, rgw daemons",
			healthConfig: func() healthConfig {
				hc := getEmtpyHealthConfig()
				hc.cephCluster = unitinputs.CephClusterNotReady.DeepCopy()
				hc.rgwOpts.storeName = "rgw-store"
				hc.rgwOpts.desiredRgwDaemons = 2
				hc.sharedFilesystemOpts.mdsDaemonsDesired["cephfs-1"] = map[string]int{"up:active": 1}
				hc.sharedFilesystemOpts.mdsStandbyDesired = 1
				return hc
			}(),
			cephStatus:     unitinputs.CephStatusCephFsRgwUnhealthy,
			cephMgrDump:    unitinputs.CephMgrDumpBaseHealthy,
			expectedStatus: unitinputs.CephDaemonsCephFsRgwUnhealthy,
			expectedIssues: []string{
				"not all (0/2) rgws are running",
				"unexpected number (0/1) of mds active are running for CephFS 'cephfs-1'",
				"unexpected number (0/1) of mds standby are running",
			},
		},
		{
			name: "daemons healthy verification, multiple cephfs, rgw daemons",
			healthConfig: func() healthConfig {
				hc := getEmtpyHealthConfig()
				hc.cephCluster = unitinputs.CephClusterReady.DeepCopy()
				hc.rgwOpts.storeName = "rgw-store"
				hc.rgwOpts.desiredRgwDaemons = 3
				hc.sharedFilesystemOpts.mdsDaemonsDesired["cephfs-1"] = map[string]int{"up:active": 1}
				hc.sharedFilesystemOpts.mdsDaemonsDesired["cephfs-2"] = map[string]int{"up:active": 1, "up:standby-replay": 1}
				hc.sharedFilesystemOpts.mdsStandbyDesired = 1
				return hc
			}(),
			cephStatus:  unitinputs.CephStatusCephFewFsRgwHealthy,
			cephMgrDump: unitinputs.CephMgrDumpBaseHealthy,
			expectedStatus: func() map[string]lcmv1alpha1.DaemonStatus {
				return map[string]lcmv1alpha1.DaemonStatus{
					"mon": unitinputs.CephDaemonsBaseHealthy["mon"],
					"mgr": unitinputs.CephDaemonsBaseHealthy["mgr"],
					"osd": unitinputs.CephDaemonsBaseHealthy["osd"],
					"rgw": {
						Status:   lcmv1alpha1.DaemonStateOk,
						Messages: []string{"3 rgws running, daemons: [10223488 11556688 12065099]"},
					},
					"mds": {
						Status: lcmv1alpha1.DaemonStateOk,
						Messages: []string{
							"mds active: 1/1 (cephfs 'cephfs-1')", "mds active: 1/1, standby-replay: 1/1 (cephfs 'cephfs-2')",
						},
					},
				}
			}(),
			expectedIssues: make([]string, 0),
		},
		{
			name: "daemons unhealthy verification, multiple cephfs, rgw daemons",
			healthConfig: func() healthConfig {
				hc := getEmtpyHealthConfig()
				hc.cephCluster = unitinputs.CephClusterReady.DeepCopy()
				hc.rgwOpts.storeName = "rgw-store"
				hc.rgwOpts.desiredRgwDaemons = 3
				hc.sharedFilesystemOpts.mdsDaemonsDesired["cephfs-1"] = map[string]int{"up:active": 1}
				hc.sharedFilesystemOpts.mdsDaemonsDesired["cephfs-2"] = map[string]int{"up:active": 1, "up:standby-replay": 1}
				hc.sharedFilesystemOpts.mdsStandbyDesired = 1
				return hc
			}(),
			cephStatus:  unitinputs.CephStatusCephFewFsRgwUnhealthy,
			cephMgrDump: unitinputs.CephMgrDumpBaseHealthy,
			expectedStatus: func() map[string]lcmv1alpha1.DaemonStatus {
				return map[string]lcmv1alpha1.DaemonStatus{
					"mon": unitinputs.CephDaemonsBaseHealthy["mon"],
					"mgr": unitinputs.CephDaemonsBaseHealthy["mgr"],
					"osd": unitinputs.CephDaemonsBaseHealthy["osd"],
					"rgw": {
						Status:   lcmv1alpha1.DaemonStateFailed,
						Messages: []string{"4 rgws running, daemons: [10223488 11556688 12065099 12065109]"},
						Issues:   []string{"unexpected rgws (4/3) rgws are running"},
					},
					"mds": {
						Status: lcmv1alpha1.DaemonStateFailed,
						Messages: []string{
							"mds active: 0/1 (cephfs 'cephfs-1')", "mds active: 1/1, standby-replay: 0/1 (cephfs 'cephfs-2')",
						},
						Issues: []string{
							"unexpected mds daemons running (CephFS 'cephfs-3')",
							"unexpected number (0/1) of mds active are running for CephFS 'cephfs-1'",
							"unexpected number (0/1) of mds standby are running",
							"unexpected number (0/1) of mds standby-replay are running for CephFS 'cephfs-2'",
						},
					},
				}
			}(),
			expectedIssues: []string{
				"unexpected mds daemons running (CephFS 'cephfs-3')",
				"unexpected number (0/1) of mds active are running for CephFS 'cephfs-1'",
				"unexpected number (0/1) of mds standby are running",
				"unexpected number (0/1) of mds standby-replay are running for CephFS 'cephfs-2'",
				"unexpected rgws (4/3) rgws are running",
			},
		},
		{
			name: "daemons healthy verification, external cluster, no extra daemons",
			healthConfig: func() healthConfig {
				hc := getEmtpyHealthConfig()
				hc.cephCluster = unitinputs.CephClusterExternal.DeepCopy()
				return hc
			}(),
			cephStatus:     unitinputs.CephStatusBaseHealthy,
			cephMgrDump:    unitinputs.CephMgrDumpBaseHealthy,
			expectedStatus: unitinputs.CephDaemonsBaseHealthy,
			expectedIssues: make([]string, 0),
		},
		{
			name: "daemons unhealthy verification, external cluster, no extra daemons",
			healthConfig: func() healthConfig {
				hc := getEmtpyHealthConfig()
				hc.cephCluster = unitinputs.CephClusterExternal.DeepCopy()
				return hc
			}(),
			cephStatus:     unitinputs.CephStatusBaseUnhealthy,
			cephMgrDump:    unitinputs.CephMgrDumpBaseUnhealthy,
			expectedStatus: unitinputs.CephDaemonsBaseUnhealthy,
			expectedIssues: []string{"no active mgr", "not all (2/3) mons are running", "not all osds are in", "not all osds are up"},
		},
		{
			name: "daemons healthy verification, external cluster",
			healthConfig: func() healthConfig {
				hc := getEmtpyHealthConfig()
				hc.cephCluster = unitinputs.CephClusterExternal.DeepCopy()
				hc.rgwOpts.storeName = "rgw-store-external"
				hc.rgwOpts.external = true
				hc.rgwOpts.externalEndpoint = "https://127.0.0.1:8443"
				return hc
			}(),
			cephStatus:  unitinputs.CephStatusCephFsRgwHealthy,
			cephMgrDump: unitinputs.CephMgrDumpBaseHealthy,
			expectedStatus: func() map[string]lcmv1alpha1.DaemonStatus {
				return map[string]lcmv1alpha1.DaemonStatus{
					"mon": unitinputs.CephDaemonsBaseHealthy["mon"],
					"mgr": unitinputs.CephDaemonsBaseHealthy["mgr"],
					"osd": unitinputs.CephDaemonsBaseHealthy["osd"],
					"rgw": unitinputs.CephDaemonsCephFsRgwHealthy["rgw"],
				}
			}(),
			expectedIssues: make([]string, 0),
		},
		{
			name: "daemons unhealthy verification, external cluster",
			healthConfig: func() healthConfig {
				hc := getEmtpyHealthConfig()
				hc.cephCluster = unitinputs.CephClusterExternal.DeepCopy()
				hc.rgwOpts.storeName = "rgw-store-external"
				hc.rgwOpts.external = true
				hc.rgwOpts.externalEndpoint = "https://127.0.0.1:8443"
				return hc
			}(),
			cephStatus:  unitinputs.CephStatusCephFsRgwUnhealthy,
			cephMgrDump: unitinputs.CephMgrDumpBaseHealthy,
			expectedStatus: func() map[string]lcmv1alpha1.DaemonStatus {
				return map[string]lcmv1alpha1.DaemonStatus{
					"mon": unitinputs.CephDaemonsBaseHealthy["mon"],
					"mgr": unitinputs.CephDaemonsBaseHealthy["mgr"],
					"osd": unitinputs.CephDaemonsBaseHealthy["osd"],
					"rgw": {
						Status:   lcmv1alpha1.DaemonStateFailed,
						Issues:   []string{"no rgws are running"},
						Messages: []string{"0 rgws running, daemons: []"},
					},
				}
			}(),
			expectedIssues: []string{"no rgws are running"},
		},
	}
	oldCephCmdFunc := lcmcommon.RunPodCommand
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := fakeCephReconcileConfig(&test.healthConfig, nil)
			faketestclients.FakeReaction(c.api.Kubeclientset.CoreV1(), "list", []string{"pods"}, map[string]runtime.Object{"pods": unitinputs.ToolBoxPodList}, nil)

			lcmcommon.RunPodCommand = func(e lcmcommon.ExecConfig) (string, string, error) {
				switch e.Command {
				case "ceph status -f json":
					if test.cephStatus != "" {
						return test.cephStatus, "", nil
					}
				case "ceph mgr dump -f json":
					if test.cephMgrDump != "" {
						return test.cephMgrDump, "", nil
					}
				}
				return "", "", errors.New("command failed")
			}

			status, issues := c.getCephDaemonsStatus()
			assert.Equal(t, test.expectedStatus, status)
			assert.Equal(t, test.expectedIssues, issues)
		})
	}
	lcmcommon.RunPodCommand = oldCephCmdFunc
}

func TestGetCSIPluginDaemonsStatus(t *testing.T) {
	tests := []struct {
		name           string
		inputResources map[string]runtime.Object
		expectedStatus map[string]lcmv1alpha1.DaemonStatus
		expectedIssues []string
	}{
		{
			name: "can't get rook operator config map",
			inputResources: map[string]runtime.Object{
				"configmaps": unitinputs.ConfigMapListEmpty,
			},
			expectedStatus: nil,
			expectedIssues: []string{"failed to get configmap 'rook-ceph/rook-ceph-operator-config'"},
		},
		{
			name: "csi plugins disabled",
			inputResources: map[string]runtime.Object{
				"configmaps": &corev1.ConfigMapList{Items: []corev1.ConfigMap{
					*unitinputs.RookOperatorConfig(map[string]string{"ROOK_CSI_ENABLE_RBD": "false", "ROOK_CSI_ENABLE_CEPHFS": "false"}),
				}},
			},
			expectedStatus: map[string]lcmv1alpha1.DaemonStatus{},
			expectedIssues: make([]string, 0),
		},
		{
			name: "csi plugins ready",
			inputResources: map[string]runtime.Object{
				"configmaps": unitinputs.ConfigMapList,
				"daemonsets": unitinputs.DaemonSetListReady,
			},
			expectedStatus: unitinputs.CephCSIDaemonsReady,
			expectedIssues: make([]string, 0),
		},
		{
			name: "csi plugins not ready",
			inputResources: map[string]runtime.Object{
				"configmaps": unitinputs.ConfigMapList,
				"daemonsets": unitinputs.DaemonSetListNotReady,
			},
			expectedStatus: unitinputs.CephCSIDaemonsNotReady,
			expectedIssues: []string{"daemonset 'rook-ceph/csi-rbdplugin' is not ready", "daemonset 'rook-ceph/csi-cephfsplugin' is not ready"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := fakeCephReconcileConfig(nil, nil)
			faketestclients.FakeReaction(c.api.Kubeclientset.CoreV1(), "get", []string{"configmaps"}, test.inputResources, nil)
			faketestclients.FakeReaction(c.api.Kubeclientset.AppsV1(), "get", []string{"daemonsets"}, test.inputResources, nil)
			status, issues := c.getCSIPluginDaemonsStatus()
			assert.Equal(t, test.expectedStatus, status)
			assert.Equal(t, test.expectedIssues, issues)
			faketestclients.CleanupFakeClientReactions(c.api.Kubeclientset.CoreV1())
			faketestclients.CleanupFakeClientReactions(c.api.Kubeclientset.AppsV1())
		})
	}
}

func TestGetDaemonSetStatus(t *testing.T) {
	dsName := "daemonset"
	dsNamespace := "rook-ceph"
	tests := []struct {
		name           string
		daemonSet      *appsv1.DaemonSet
		apiError       bool
		expectedStatus lcmv1alpha1.DaemonStatus
		expectedReady  int
	}{
		{
			name:           "cant get daemonset",
			apiError:       true,
			expectedStatus: lcmv1alpha1.DaemonStatus{Status: lcmv1alpha1.DaemonStateFailed, Issues: []string{"failed to get 'rook-ceph/daemonset' daemonset"}},
		},
		{
			name:           "daemonset not found",
			expectedStatus: lcmv1alpha1.DaemonStatus{Status: lcmv1alpha1.DaemonStateFailed, Issues: []string{"daemonset 'rook-ceph/daemonset' is not found"}},
		},
		{
			name:           "daemonset status ok",
			daemonSet:      unitinputs.DaemonSetWithStatus(dsNamespace, dsName, 2, 2),
			expectedStatus: lcmv1alpha1.DaemonStatus{Status: lcmv1alpha1.DaemonStateOk, Messages: []string{"2/2 ready"}},
			expectedReady:  2,
		},
		{
			name:      "daemonset status not ok",
			daemonSet: unitinputs.DaemonSetWithStatus(dsNamespace, dsName, 2, 0),
			expectedStatus: lcmv1alpha1.DaemonStatus{
				Status:   lcmv1alpha1.DaemonStateFailed,
				Messages: []string{"0/2 ready"},
				Issues:   []string{"daemonset 'rook-ceph/daemonset' is not ready"},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := fakeCephReconcileConfig(nil, nil)
			res := map[string]runtime.Object{}
			if test.daemonSet != nil {
				res["daemonsets"] = &appsv1.DaemonSetList{Items: []appsv1.DaemonSet{*test.daemonSet}}
			} else if !test.apiError {
				res["daemonsets"] = unitinputs.DaemonSetListEmpty
			}
			faketestclients.FakeReaction(c.api.Kubeclientset.AppsV1(), "get", []string{"daemonsets"}, res, nil)

			status, ready := c.getDaemonSetStatus("rook-ceph", dsName)
			assert.Equal(t, test.expectedStatus, status)
			assert.Equal(t, test.expectedReady, ready)
			faketestclients.CleanupFakeClientReactions(c.api.Kubeclientset.AppsV1())
		})
	}
}
