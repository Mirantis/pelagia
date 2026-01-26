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
	"time"

	"github.com/pkg/errors"
	cephv1 "github.com/rook/rook/pkg/apis/ceph.rook.io/v1"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/runtime"

	lcmv1alpha1 "github.com/Mirantis/pelagia/pkg/apis/ceph.pelagia.lcm/v1alpha1"
	lcmcommon "github.com/Mirantis/pelagia/pkg/common"
	faketestclients "github.com/Mirantis/pelagia/test/unit/clients"
	unitinputs "github.com/Mirantis/pelagia/test/unit/inputs"
)

var osdClusterDetails = map[string]nodeDetails{
	"node-1": {
		"osd.20": osdDetails{
			ClusterFSID:      "8668f062-3faa-358a-85f3-f80fe6c1e306",
			DeviceName:       "vde",
			DeviceByID:       "2926ff77-7491-4447-a",
			DeviceByPath:     "/dev/disk/by-path/pci-0000:00:0f.0",
			BlockPartition:   "/dev/dm-0",
			MetaDeviceName:   "vdd",
			MetaDeviceByID:   "e8d89e2f-ffc6-4988-9",
			MetaDeviceByPath: "/dev/disk/by-path/pci-0000:00:0e.0",
			MetaPartition:    "/dev/dm-1",
			UUID:             "vbsgs3a3-sdcv-casq-sd11-asd12dasczsf",
			Up:               true,
			In:               true,
		},
		"osd.25": osdDetails{
			ClusterFSID:      "8668f062-3faa-358a-85f3-f80fe6c1e306",
			DeviceName:       "vdf",
			DeviceByID:       "b7ea1c8c-89b8-4354-8",
			DeviceByPath:     "/dev/disk/by-path/pci-0000:00:10.0",
			BlockPartition:   "/dev/dm-2",
			MetaDeviceName:   "vdd",
			MetaDeviceByID:   "e8d89e2f-ffc6-4988-9",
			MetaDeviceByPath: "/dev/disk/by-path/pci-0000:00:0e.0",
			MetaPartition:    "/dev/dm-3",
			UUID:             "d49fd9bf-d2dd-4c3d-824d-87f3f17ea44a",
			Up:               true,
			In:               true,
		},
		"osd.30": osdDetails{
			ClusterFSID:      "8668f062-3faa-358a-85f3-f80fe6c1e306",
			DeviceName:       "vdb",
			DeviceByID:       "996ea59f-7f47-4fac-b",
			DeviceByPath:     "/dev/disk/by-path/pci-0000:00:0a.0",
			BlockPartition:   "/dev/dm-4",
			MetaDeviceName:   "vda",
			MetaDeviceByID:   "8dad5ae9-ddf7-40bf-8",
			MetaDeviceByPath: "/dev/disk/by-path/pci-0000:00:09.0",
			MetaPartition:    "/dev/vda14",
			UUID:             "f4edb5cd-fb1e-4620-9419-3f9a4fcecba5",
			Up:               true,
			In:               true,
		},
	},
	"node-2": {
		"osd.0": osdDetails{
			ClusterFSID:    "8668f062-3faa-358a-85f3-f80fe6c1e306",
			DeviceName:     "vdb",
			DeviceByID:     "b4eaf39c-b561-4269-1",
			DeviceByPath:   "/dev/disk/by-path/pci-0000:00:0a.0",
			BlockPartition: "/dev/dm-0",
			UUID:           "69481cd1-38b1-42fd-ac07-06bf4d7c0e19",
			Up:             true,
			In:             true,
		},
		"osd.4": osdDetails{
			ClusterFSID:    "8668f062-3faa-358a-85f3-f80fe6c1e306",
			DeviceName:     "vdd",
			DeviceByID:     "35a15532-8b56-4f83-9",
			DeviceByPath:   "/dev/disk/by-path/pci-0000:00:1e.0",
			BlockPartition: "/dev/dm-2",
			UUID:           "ad76cf53-5cb5-48fe-a39a-343734f5ccde",
			Up:             true,
			In:             true,
		},
		"osd.5": osdDetails{
			ClusterFSID:    "8668f062-3faa-358a-85f3-f80fe6c1e306",
			DeviceName:     "vdd",
			DeviceByID:     "35a15532-8b56-4f83-9",
			DeviceByPath:   "/dev/disk/by-path/pci-0000:00:1e.0",
			BlockPartition: "/dev/dm-3",
			UUID:           "af39b794-e1c6-41c0-8997-d6b6c631b8f2",
			Up:             true,
			In:             true,
		},
	},
	"__stray": {
		"osd.2": osdDetails{
			ClusterFSID: "8668f062-3faa-358a-85f3-f80fe6c1e306",
			UUID:        "61869d90-2c45-4f02-b7c3-96955f41e2ca",
		},
	},
}

func TestGetDeviceMappings(t *testing.T) {
	baseConfig := getEmtpyHealthConfig()
	baseConfig.cephCluster = &unitinputs.CephClusterReady
	tests := []struct {
		name              string
		cephOsdMetaOutput string
		cephOsdInfoOutput string
		expectedStatus    map[string]nodeDetails
		expectedIssue     string
	}{
		{
			name:          "failed to run ceph osd metadata",
			expectedIssue: "failed to run command 'ceph osd metadata -f json': failed command",
		},
		{
			name:              "failed to run ceph osd info",
			cephOsdMetaOutput: unitinputs.CephOsdMetadataOutput,
			expectedIssue:     "failed to run command 'ceph osd info -f json': failed command",
		},
		{
			name:              "get ceph osd metadata and info",
			cephOsdMetaOutput: unitinputs.CephOsdMetadataOutput,
			cephOsdInfoOutput: unitinputs.CephOsdInfoOutput,
			expectedStatus:    osdClusterDetails,
		},
		{
			name:              "get ceph osd metadata with osd down and not in",
			cephOsdMetaOutput: `[{"devices": "vdb","bluestore_bdev_devices": "vdb","bluestore_bdev_type": "hdd","bluestore_bdev_partition_path": "/dev/dm-0","bluefs_dedicated_db": "0","id": 0,"hostname": "fake","device_ids": "vdb=fakeid","device_paths": "vdb=fakepath"}]`,
			cephOsdInfoOutput: `[{"osd": 0, "uuid": "fakeuuid", "up": 0, "in": 0}]`,
			expectedStatus: map[string]nodeDetails{
				"fake": {
					"osd.0": osdDetails{
						ClusterFSID:    "8668f062-3faa-358a-85f3-f80fe6c1e306",
						DeviceName:     "vdb",
						DeviceByID:     "fakeid",
						DeviceByPath:   "fakepath",
						BlockPartition: "/dev/dm-0",
						UUID:           "fakeuuid",
					},
				},
			},
		},
	}
	oldCmdRun := lcmcommon.RunPodCommand
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := fakeCephReconcileConfig(&baseConfig, nil)
			faketestclients.FakeReaction(c.api.Kubeclientset.CoreV1(), "list", []string{"pods"}, map[string]runtime.Object{"pods": unitinputs.ToolBoxPodList}, nil)
			lcmcommon.RunPodCommand = func(e lcmcommon.ExecConfig) (string, string, error) {
				switch e.Command {
				case "ceph osd metadata -f json":
					if test.cephOsdMetaOutput != "" {
						return test.cephOsdMetaOutput, "", nil
					}
				case "ceph osd info -f json":
					if test.cephOsdInfoOutput != "" {
						return test.cephOsdInfoOutput, "", nil
					}
				}
				return "", "", errors.New("failed command")
			}

			status, err := c.getOsdClusterDetails()
			assert.Equal(t, test.expectedStatus, status)
			if test.expectedIssue != "" {
				assert.NotNil(t, err)
				assert.Equal(t, test.expectedIssue, err.Error())
			} else {
				assert.Nil(t, err)
			}
		})
	}
	lcmcommon.RunPodCommand = oldCmdRun
}

func TestGetSpecAnalysisStatus(t *testing.T) {
	nodesList := unitinputs.GetNodesList(
		[]unitinputs.NodeAttrs{{Name: "node-1", Labeled: true}, {Name: "node-2", Labeled: true}})
	baseConfig := getEmtpyHealthConfig()
	baseConfig.cephCluster = &unitinputs.CephClusterReady
	tests := []struct {
		name           string
		inputResources map[string]runtime.Object
		healthConfig   healthConfig
		checkDisabled  bool
		cephCliOutput  map[string]string
		daemonReport   map[string]string
		expectedStatus *lcmv1alpha1.OsdSpecAnalysisState
		expectedIssues []string
	}{
		{
			name: "spec analyse is disabled for external case",
			healthConfig: func() healthConfig {
				hc := getEmtpyHealthConfig()
				hc.cephCluster = &unitinputs.CephClusterExternal
				return hc
			}(),
		},
		{
			name:          "spec analyse check is disabled",
			checkDisabled: true,
			healthConfig:  baseConfig,
		},
		{
			name:           "failed to get osd cluster details",
			healthConfig:   baseConfig,
			expectedIssues: []string{"failed to get osd cluster info"},
		},
		{
			name: "failed to get disk-daemon daemonset",
			inputResources: map[string]runtime.Object{
				"daemonsets": unitinputs.DaemonSetListEmpty,
			},
			healthConfig: baseConfig,
			cephCliOutput: map[string]string{
				"ceph osd metadata -f json": unitinputs.CephOsdMetadataOutput,
				"ceph osd info -f json":     unitinputs.CephOsdInfoOutput,
			},
			expectedStatus: &lcmv1alpha1.OsdSpecAnalysisState{
				DiskDaemon: lcmv1alpha1.DaemonStatus{
					Status: lcmv1alpha1.DaemonStateFailed,
					Issues: []string{"daemonset 'lcm-namespace/pelagia-disk-daemon' is not found"},
				},
			},
			expectedIssues: []string{"daemonset 'lcm-namespace/pelagia-disk-daemon' is not found"},
		},
		{
			name: "disk-daemon daemonset is not ready yet",
			inputResources: map[string]runtime.Object{
				"daemonsets": unitinputs.DaemonSetListNotReady,
			},
			healthConfig: baseConfig,
			cephCliOutput: map[string]string{
				"ceph osd metadata -f json": unitinputs.CephOsdMetadataOutput,
				"ceph osd info -f json":     unitinputs.CephOsdInfoOutput,
			},
			expectedStatus: &lcmv1alpha1.OsdSpecAnalysisState{
				DiskDaemon: lcmv1alpha1.DaemonStatus{
					Status:   lcmv1alpha1.DaemonStateFailed,
					Issues:   []string{"daemonset 'lcm-namespace/pelagia-disk-daemon' is not ready"},
					Messages: []string{"0/2 ready"},
				},
			},
			expectedIssues: []string{"daemonset 'lcm-namespace/pelagia-disk-daemon' is not ready"},
		},
		{
			name: "spec analyse failed - disk report contains issues",
			inputResources: map[string]runtime.Object{
				"daemonsets": unitinputs.DaemonSetListReady,
				"nodes":      &nodesList,
			},
			healthConfig: baseConfig,
			cephCliOutput: map[string]string{
				"ceph osd metadata -f json": unitinputs.CephOsdMetadataOutput,
				"ceph osd info -f json":     unitinputs.CephOsdInfoOutput,
			},
			daemonReport: map[string]string{
				"node-1": "{||}",
				"node-2": "{||}",
			},
			expectedStatus: unitinputs.OsdSpecAnalysisNotOk,
			expectedIssues: []string{
				"node 'node-1' has failed spec analyse",
				"node 'node-2' has failed spec analyse",
			},
		},
		{
			name: "spec analyse no issues found",
			inputResources: map[string]runtime.Object{
				"daemonsets": unitinputs.DaemonSetListReady,
				"nodes":      &nodesList,
			},
			healthConfig: baseConfig,
			cephCliOutput: map[string]string{
				"ceph osd metadata -f json": unitinputs.CephOsdMetadataOutput,
				"ceph osd info -f json":     unitinputs.CephOsdInfoOutput,
			},
			daemonReport: map[string]string{
				"node-1": unitinputs.CephDiskDaemonDiskReportStringNode1,
				"node-2": unitinputs.CephDiskDaemonDiskReportStringNode2,
			},
			expectedStatus: unitinputs.OsdSpecAnalysisOk,
			expectedIssues: []string{},
		},
		{
			name: "spec has no nodes",
			inputResources: map[string]runtime.Object{
				"daemonsets": unitinputs.DaemonSetListReady,
				"nodes":      &nodesList,
			},
			healthConfig: func() healthConfig {
				hc := getEmtpyHealthConfig()
				hc.cephCluster = unitinputs.CephClusterReady.DeepCopy()
				hc.cephCluster.Spec.Storage.Nodes = nil
				return hc
			}(),
			cephCliOutput: map[string]string{
				"ceph osd metadata -f json": unitinputs.CephOsdMetadataOutput,
				"ceph osd info -f json":     unitinputs.CephOsdInfoOutput,
			},
			expectedStatus: func() *lcmv1alpha1.OsdSpecAnalysisState {
				status := unitinputs.OsdSpecAnalysisOk.DeepCopy()
				status.SpecAnalysis = nil
				return status
			}(),
			expectedIssues: []string{},
		},
	}
	oldCmdFunc := lcmcommon.RunPodCommand
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			lcmConfigData := map[string]string{}
			if test.checkDisabled {
				lcmConfigData["HEALTH_CHECKS_SKIP"] = "spec_analysis"
			}
			c := fakeCephReconcileConfig(&test.healthConfig, lcmConfigData)
			faketestclients.FakeReaction(c.api.Kubeclientset.CoreV1(), "list", []string{"pods"}, map[string]runtime.Object{"pods": unitinputs.ToolBoxAndDiskDaemonPodsList}, nil)
			faketestclients.FakeReaction(c.api.Kubeclientset.AppsV1(), "get", []string{"daemonsets"}, test.inputResources, nil)
			faketestclients.FakeReaction(c.api.Kubeclientset.CoreV1(), "get", []string{"nodes"}, test.inputResources, nil)

			lcmcommon.RunPodCommand = func(e lcmcommon.ExecConfig) (string, string, error) {
				if e.Command == "pelagia-disk-daemon --full-report --port 9999" {
					if test.daemonReport != nil {
						return test.daemonReport[e.Nodename], "", nil
					}
				} else if output, ok := test.cephCliOutput[e.Command]; ok {
					return output, "", nil
				}
				return "", "", errors.New("failed command")
			}

			analyseSpecStatus, issues := c.getSpecAnalysisStatus()
			assert.Equal(t, test.expectedStatus, analyseSpecStatus)
			assert.Equal(t, test.expectedIssues, issues)
			faketestclients.CleanupFakeClientReactions(c.api.Kubeclientset.CoreV1())
			faketestclients.CleanupFakeClientReactions(c.api.Kubeclientset.AppsV1())
		})
	}
	lcmcommon.RunPodCommand = oldCmdFunc
}

func TestThreadsForPrepareSpecAnalysis(t *testing.T) {
	nodesList := unitinputs.GetNodesList(
		[]unitinputs.NodeAttrs{
			{Name: "node-1", Labeled: true},
			{Name: "node-2", Labeled: true},
			{Name: "node-3", Labeled: true},
			{Name: "node-4", Labeled: true},
			{Name: "node-5", Labeled: true},
			{Name: "node-6", Labeled: true},
		})
	inputResources := map[string]runtime.Object{"nodes": &nodesList}
	reportInProgress := unitinputs.GetDiskDaemonReportToString(&lcmcommon.DiskDaemonReport{State: lcmcommon.DiskDaemonStateInProgress})
	getExtraNode := func(name string) cephv1.Node {
		return cephv1.Node{
			Name:      name,
			Selection: cephv1.Selection{Devices: []cephv1.Device{{Name: "vdb"}}},
		}
	}

	tests := []struct {
		name           string
		nodes          []cephv1.Node
		shouldWait     bool
		expectedStatus map[string]lcmv1alpha1.DaemonStatus
		expectedIssues []string
	}{
		{
			name:  "check multiple threads",
			nodes: []cephv1.Node{getExtraNode("node-1"), getExtraNode("node-2"), getExtraNode("node-3"), getExtraNode("node-4"), getExtraNode("node-5")},
			expectedStatus: map[string]lcmv1alpha1.DaemonStatus{
				"node-1": {
					Status: lcmv1alpha1.DaemonStateFailed,
					Issues: []string{"disk report is not ready"},
				},
				"node-2": {
					Status: lcmv1alpha1.DaemonStateFailed,
					Issues: []string{"disk report is not ready"},
				},
				"node-3": {
					Status: lcmv1alpha1.DaemonStateFailed,
					Issues: []string{"disk report is not ready"},
				},
				"node-4": {
					Status: lcmv1alpha1.DaemonStateFailed,
					Issues: []string{"disk report is not ready"},
				},
				"node-5": {
					Status: lcmv1alpha1.DaemonStateFailed,
					Issues: []string{"disk report is not ready"},
				},
			},
			expectedIssues: []string{
				"node 'node-1' has failed spec analyse",
				"node 'node-2' has failed spec analyse",
				"node 'node-3' has failed spec analyse",
				"node 'node-4' has failed spec analyse",
				"node 'node-5' has failed spec analyse",
			},
		},
		{
			name:       "check multiple threads - overflow",
			nodes:      []cephv1.Node{getExtraNode("node-1"), getExtraNode("node-2"), getExtraNode("node-3"), getExtraNode("node-4"), getExtraNode("node-5"), getExtraNode("node-6")},
			shouldWait: true,
			expectedStatus: map[string]lcmv1alpha1.DaemonStatus{
				"node-1": {
					Status: lcmv1alpha1.DaemonStateFailed,
					Issues: []string{"disk report is not ready"},
				},
				"node-2": {
					Status: lcmv1alpha1.DaemonStateFailed,
					Issues: []string{"disk report is not ready"},
				},
				"node-3": {
					Status: lcmv1alpha1.DaemonStateFailed,
					Issues: []string{"disk report is not ready"},
				},
				"node-4": {
					Status: lcmv1alpha1.DaemonStateFailed,
					Issues: []string{"disk report is not ready"},
				},
				"node-5": {
					Status: lcmv1alpha1.DaemonStateFailed,
					Issues: []string{"disk report is not ready"},
				},
				"node-6": {
					Status: lcmv1alpha1.DaemonStateFailed,
					Issues: []string{"disk report is not ready"},
				},
			},
			expectedIssues: []string{
				"node 'node-1' has failed spec analyse",
				"node 'node-2' has failed spec analyse",
				"node 'node-3' has failed spec analyse",
				"node 'node-4' has failed spec analyse",
				"node 'node-5' has failed spec analyse",
				"node 'node-6' has failed spec analyse",
			},
		},
	}
	oldVal := timeRetrySleep
	timeRetrySleep = 1 * time.Second
	oldCmdFunc := lcmcommon.RunPodCommandWithValidation
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cephCluster := unitinputs.CephClusterReady.DeepCopy()
			cephCluster.Spec.Storage.Nodes = test.nodes
			baseConfig := getEmtpyHealthConfig()
			baseConfig.cephCluster = cephCluster
			c := fakeCephReconcileConfig(&baseConfig, nil)
			faketestclients.FakeReaction(c.api.Kubeclientset.CoreV1(), "get", []string{"nodes"}, inputResources, nil)

			lcmcommon.RunPodCommandWithValidation = func(e lcmcommon.ExecConfig) (string, string, error) {
				if e.Command == "pelagia-disk-daemon --full-report --port 9999" {
					return reportInProgress, "", nil
				}
				return "", "", errors.New("failed command")
			}

			// measure time for proper work multithreading
			startTime := time.Now()
			analyseSpecStatus, issues := c.prepareSpecAnalysis(osdClusterDetails)
			endTime := time.Now()
			diff := endTime.Sub(startTime).Seconds()

			assert.Equal(t, test.expectedStatus, analyseSpecStatus)
			assert.Equal(t, test.expectedIssues, issues)
			if test.shouldWait {
				// kind of hardcode, we have 5 nodes in parallel with 3 seconds retry + last node 3 seconds retries, so min time is 6s
				assert.Greater(t, diff, float64(6))
			} else {
				// kind of hardcode, because 3 retries by 1 sec + some min time for code processing
				assert.Less(t, diff, 3.5)
			}
			faketestclients.CleanupFakeClientReactions(c.api.Kubeclientset.CoreV1())
		})
	}
	lcmcommon.RunPodCommandWithValidation = oldCmdFunc
	timeRetrySleep = oldVal
}

func TestPrepareSpecAnalysis(t *testing.T) {
	nodesList := unitinputs.GetNodesList(
		[]unitinputs.NodeAttrs{{Name: "node-1", Labeled: true}, {Name: "node-2", Labeled: true}})
	inputResources := map[string]runtime.Object{"nodes": &nodesList}
	tests := []struct {
		name           string
		cephCluster    *cephv1.CephCluster
		daemonReport   map[string]string
		expectedStatus map[string]lcmv1alpha1.DaemonStatus
		expectedIssues []string
	}{
		{
			name:        "can't parse disk daemon daemonset report",
			cephCluster: &unitinputs.CephClusterReady,
			daemonReport: map[string]string{
				"node-1": "{||}",
				"node-2": "{||}",
			},
			expectedStatus: unitinputs.OsdStorageSpecAnalysisFailed,
			expectedIssues: []string{
				"node 'node-1' has failed spec analyse",
				"node 'node-2' has failed spec analyse",
			},
		},
		{
			name: "disk daemon daemonset report contains issues for some nodes",
			cephCluster: func() *cephv1.CephCluster {
				cluster := unitinputs.CephClusterReady.DeepCopy()
				cluster.Spec.Storage.Nodes[1] = unitinputs.StorageNodesForAnalysisNotAllSpecified[1]
				return cluster
			}(),
			daemonReport: map[string]string{
				"node-1": unitinputs.GetDiskDaemonReportToString(&unitinputs.DiskDaemonReportOkNode1SomeDevLost),
				"node-2": unitinputs.CephDiskDaemonDiskReportStringNode2,
			},
			expectedStatus: map[string]lcmv1alpha1.DaemonStatus{
				"node-1": {
					Status: lcmv1alpha1.DaemonStateFailed,
					Issues: []string{
						"metadata device '/dev/ceph-metadata/part-2' specified for device 'vdf' is not found on a node",
						"metadata device '/dev/disk/by-id/virtio-e8d89e2f-ffc6-4988-9' specified for device '/dev/disk/by-path/pci-0000:00:0f.0' is not found on a node",
					},
				},
				"node-2": {
					Status: lcmv1alpha1.DaemonStateOk,
					Messages: []string{
						"found ceph block partition '/dev/ceph-0e03d5c6-d0e9-4f04-b9af-38d15e14369f/osd-block-61869d90-2c45-4f02-b7c3-96955f41e2ca', belongs to osd '2' (osd fsid '61869d90-2c45-4f02-b7c3-96955f41e2ca'), placed on '/dev/vde' device, which seems to be stray, can be cleaned up",
						"found ceph block partition '/dev/ceph-c5628abe-ae41-4c3d-bdc6-ef86c54bf78c/osd-block-69481cd1-38b1-42fd-ac07-06bf4d7c0e19', belongs to osd '0' (osd fsid '06bf4d7c-9603-41a4-b250-284ecf3ecb2f'), placed on '/dev/vdc' device, which seems to be stray, can be cleaned up",
						"found ceph block partition '/dev/ceph-dada9f25-41b4-4c26-9a20-448ac01e1d06/osd-block-7d09cceb-4de0-478e-9d8d-bd09cb0c904e', belongs to osd '5' (osd fsid 'af39b794-e1c6-41c0-8997-d6b6c631b8f2'), placed on '/dev/vdd' device, which is not reflected in spec",
						"found ceph block partition '/dev/ceph-dada9f25-41b4-4c26-9a20-448ac01e1d06/osd-block-ad76cf53-5cb5-48fe-a39a-343734f5ccde', belongs to osd '4' (osd fsid 'ad76cf53-5cb5-48fe-a39a-343734f5ccde'), placed on '/dev/vdd' device, which is not reflected in spec",
					},
				},
			},
			expectedIssues: []string{
				"node 'node-1' has failed spec analyse",
				"node 'node-2' has running osd(s), not described in spec",
			},
		},
		{
			name:        "disk daemon daemonset report contains no issues found",
			cephCluster: &unitinputs.CephClusterReady,
			daemonReport: map[string]string{
				"node-1": unitinputs.CephDiskDaemonDiskReportStringNode1,
				"node-2": unitinputs.CephDiskDaemonDiskReportStringNode2,
			},
			expectedStatus: unitinputs.OsdStorageSpecAnalysisOk,
		},
	}
	oldCmdFunc := lcmcommon.RunPodCommandWithValidation
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			baseConfig := getEmtpyHealthConfig()
			baseConfig.cephCluster = test.cephCluster
			c := fakeCephReconcileConfig(&baseConfig, nil)
			faketestclients.FakeReaction(c.api.Kubeclientset.CoreV1(), "get", []string{"nodes"}, inputResources, nil)

			lcmcommon.RunPodCommandWithValidation = func(e lcmcommon.ExecConfig) (string, string, error) {
				if e.Command == "pelagia-disk-daemon --full-report --port 9999" {
					if test.daemonReport != nil {
						return test.daemonReport[e.Nodename], "", nil
					}
				}
				return "", "", errors.New("failed command")
			}

			analyseSpecStatus, issues := c.prepareSpecAnalysis(osdClusterDetails)
			assert.Equal(t, test.expectedStatus, analyseSpecStatus)
			assert.Equal(t, test.expectedIssues, issues)
			faketestclients.CleanupFakeClientReactions(c.api.Kubeclientset.CoreV1())
		})
	}
	lcmcommon.RunPodCommandWithValidation = oldCmdFunc
}

func TestGetNodeAnalyseStatus(t *testing.T) {
	diskReportNode1 := unitinputs.GetDiskDaemonReportToString(&unitinputs.DiskDaemonReportOkNode1SomeDevLost)
	diskReportNotReady := unitinputs.GetDiskDaemonReportToString(&lcmcommon.DiskDaemonReport{State: lcmcommon.DiskDaemonStateInProgress})

	nodesList := unitinputs.GetNodesList(
		[]unitinputs.NodeAttrs{
			{Name: "node-1", Labeled: true},
			{Name: "node-2", Labeled: true},
			{Name: "node-3", Labeled: false},
			{Name: "node-4", Labeled: true, Unreachable: true},
		})
	inputResources := map[string]runtime.Object{"nodes": &nodesList}
	tests := []struct {
		name           string
		node           cephv1.Node
		daemonReport   string
		checkRetry     bool
		expectedStatus lcmv1alpha1.DaemonStatus
		expectedExtra  bool
	}{
		{
			name: "failed to get k8s node",
			node: cephv1.Node{Name: "node-x"},
			expectedStatus: lcmv1alpha1.DaemonStatus{
				Status: lcmv1alpha1.DaemonStateFailed,
				Issues: []string{"failed to get node 'node-x' info"},
			},
		},
		{
			name: "k8s node has no disk-daemon labels",
			node: cephv1.Node{Name: "node-3"},
			expectedStatus: lcmv1alpha1.DaemonStatus{
				Status:   lcmv1alpha1.DaemonStateSkipped,
				Messages: []string{"disk daemon is not running for node (missed daemon label), spec analysis skipped"},
			},
		},
		{
			name: "k8s node is unreachable",
			node: cephv1.Node{Name: "node-4"},
			expectedStatus: lcmv1alpha1.DaemonStatus{
				Status: lcmv1alpha1.DaemonStateFailed,
				Issues: []string{"node 'node-4' has 'node.kubernetes.io/unreachable' taint, assuming node is not available"},
			},
		},
		{
			name:           "failed to get disk report",
			node:           unitinputs.StorageNodesForAnalysisOk[0],
			expectedStatus: unitinputs.OsdStorageSpecAnalysisFailed["node-1"],
		},
		{
			name: "disk report is failed",
			node: unitinputs.StorageNodesForAnalysisOk[0],
			daemonReport: unitinputs.GetDiskDaemonReportToString(&lcmcommon.DiskDaemonReport{
				State:  lcmcommon.DiskDaemonStateFailed,
				Issues: []string{"failed to build report"},
			}),
			expectedStatus: lcmv1alpha1.DaemonStatus{
				Status: lcmv1alpha1.DaemonStateFailed,
				Issues: []string{"disk report is failed"},
			},
		},
		{
			name:         "disk report is preparing and not ready",
			node:         unitinputs.StorageNodesForAnalysisOk[1],
			daemonReport: diskReportNotReady,
			expectedStatus: lcmv1alpha1.DaemonStatus{
				Status: lcmv1alpha1.DaemonStateFailed,
				Issues: []string{"disk report is not ready"},
			},
		},
		{
			name:           "disk report is preparing and spec ok",
			node:           unitinputs.StorageNodesForAnalysisOk[1],
			checkRetry:     true,
			daemonReport:   unitinputs.CephDiskDaemonDiskReportStringNode2,
			expectedStatus: unitinputs.OsdStorageSpecAnalysisOk["node-2"],
		},
		{
			name:         "spec has problems",
			node:         unitinputs.StorageNodesForAnalysisNotAllSpecified[0],
			daemonReport: diskReportNode1,
			expectedStatus: lcmv1alpha1.DaemonStatus{
				Status: lcmv1alpha1.DaemonStateFailed,
				Issues: []string{"metadata device '/dev/ceph-metadata/part-2' specified for device 'vdf' is not found on a node"},
			},
		},
		{
			name: "disk report is skipped for use all devices",
			node: cephv1.Node{
				Name:      "node-2",
				Selection: cephv1.Selection{UseAllDevices: &[]bool{true}[0]},
			},
			daemonReport: unitinputs.CephDiskDaemonDiskReportStringNode2,
			expectedStatus: lcmv1alpha1.DaemonStatus{
				Status:   lcmv1alpha1.DaemonStateSkipped,
				Messages: []string{"used 'useAllDevices' flag for node definition, spec analysis skipped"},
			},
		},
		{
			name: "disk report is skipped for pvc based node",
			node: cephv1.Node{
				Name: "node-2",
				Selection: cephv1.Selection{
					VolumeClaimTemplates: []cephv1.VolumeClaimTemplate{{ObjectMeta: unitinputs.CephClusterExternal.ObjectMeta}},
				},
			},
			daemonReport: unitinputs.CephDiskDaemonDiskReportStringNode2,
			expectedStatus: lcmv1alpha1.DaemonStatus{
				Status:   lcmv1alpha1.DaemonStateSkipped,
				Messages: []string{"pvc based node, spec analysis skipped"},
			},
		},
		{
			name:         "disk report is ready but spec has no any devices",
			node:         cephv1.Node{Name: "node-1"},
			daemonReport: diskReportNode1,
			expectedStatus: lcmv1alpha1.DaemonStatus{
				Status: lcmv1alpha1.DaemonStateOk,
				Messages: []string{
					"found ceph block partition '/dev/ceph-21312wds-sdfv-vs3f-scv3-sdfdsg23edaa/osd-block-vbsgs3a3-sdcv-casq-sd11-asd12dasczsf', belongs to osd '20' (osd fsid 'vbsgs3a3-sdcv-casq-sd11-asd12dasczsf'), placed on '/dev/vde' device, which is not reflected in spec",
					"found ceph block partition '/dev/ceph-2efce189-afb7-452f-bd32-c73b5017a0da/osd-block-d49fd9bf-d2dd-4c3d-824d-87f3f17ea44a', belongs to osd '25' (osd fsid 'd49fd9bf-d2dd-4c3d-824d-87f3f17ea44a'), placed on '/dev/vdf' device, which is not reflected in spec",
					"found ceph block partition '/dev/ceph-992bbd78-3d8e-4cc3-93dc-eae387309364/osd-block-f4edb5cd-fb1e-4620-9419-3f9a4fcecba5', belongs to osd '30' (osd fsid 'f4edb5cd-fb1e-4620-9419-3f9a4fcecba5'), placed on '/dev/vdb' device, which is not reflected in spec",
					"found ceph db partition '/dev/vda14', belongs to osd '30' (osd fsid 'f4edb5cd-fb1e-4620-9419-3f9a4fcecba5'), placed on '/dev/vda' device, which is not reflected in spec",
				},
			},
			expectedExtra: true,
		},
	}
	oldVal := timeRetrySleep
	timeRetrySleep = 1 * time.Second
	oldCmdFunc := lcmcommon.RunPodCommandWithValidation
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := fakeCephReconcileConfig(nil, nil)
			retry := 0
			if test.checkRetry {
				retry++
			}
			faketestclients.FakeReaction(c.api.Kubeclientset.CoreV1(), "get", []string{"nodes"}, inputResources, nil)

			lcmcommon.RunPodCommandWithValidation = func(e lcmcommon.ExecConfig) (string, string, error) {
				if e.Command == "pelagia-disk-daemon --full-report --port 9999" {
					if test.daemonReport == "" {
						return "", "", errors.New("failed to get report")
					}
					if retry > 0 {
						retry--
						return diskReportNotReady, "", nil
					}
					return test.daemonReport, "", nil
				}
				return "", "", errors.New("failed command")
			}

			status, extra := c.getNodeAnalyseStatus("lcm-namespace", test.node, osdClusterDetails)
			assert.Equal(t, test.expectedStatus, status)
			assert.Equal(t, test.expectedExtra, extra)
			faketestclients.CleanupFakeClientReactions(c.api.Kubeclientset.CoreV1())
		})
	}
	lcmcommon.RunPodCommandWithValidation = oldCmdFunc
	timeRetrySleep = oldVal
}

func TestRunSpecAnalyse(t *testing.T) {
	daemonInfoMapping := map[string]*lcmcommon.DiskDaemonReport{
		"node-1": &unitinputs.DiskDaemonReportOkNode1,
		"node-2": unitinputs.DiskDaemonNodeReportWithStrayOkNode2(true),
	}

	tests := []struct {
		name             string
		node             cephv1.Node
		expectedIssues   []string
		expectedWarnings []string
		expectedExtra    bool
	}{
		{
			name:             "nodespec 1 - analyse is ok",
			node:             unitinputs.StorageNodesForAnalysisOk[0],
			expectedWarnings: []string{},
		},
		{
			name: "nodespec 2 - analyse is ok, found stray partitions",
			node: unitinputs.StorageNodesForAnalysisOk[1],
			expectedWarnings: []string{
				"found ceph block partition '/dev/ceph-0e03d5c6-d0e9-4f04-b9af-38d15e14369f/osd-block-61869d90-2c45-4f02-b7c3-96955f41e2ca', belongs to osd '2' (osd fsid '61869d90-2c45-4f02-b7c3-96955f41e2ca'), placed on '/dev/vde' device, which seems to be stray, can be cleaned up",
				"found ceph block partition '/dev/ceph-c5628abe-ae41-4c3d-bdc6-ef86c54bf78c/osd-block-69481cd1-38b1-42fd-ac07-06bf4d7c0e19', belongs to osd '0' (osd fsid '06bf4d7c-9603-41a4-b250-284ecf3ecb2f'), placed on '/dev/vdc' device, which seems to be stray, can be cleaned up",
			},
		},
		{
			name: "nodespec 1 - analyse is ok, not all devices in spec",
			node: cephv1.Node{
				Name: "node-1",
				Selection: cephv1.Selection{
					Devices: []cephv1.Device{
						{
							Name: "vdf",
							Config: map[string]string{
								"deviceClass":    "hdd",
								"metadataDevice": "/dev/ceph-metadata/part-2",
							},
						},
					},
				},
			},
			expectedWarnings: []string{
				"found ceph block partition '/dev/ceph-21312wds-sdfv-vs3f-scv3-sdfdsg23edaa/osd-block-vbsgs3a3-sdcv-casq-sd11-asd12dasczsf', belongs to osd '20' (osd fsid 'vbsgs3a3-sdcv-casq-sd11-asd12dasczsf'), placed on '/dev/vde' device, which is not reflected in spec",
				"found ceph block partition '/dev/ceph-992bbd78-3d8e-4cc3-93dc-eae387309364/osd-block-f4edb5cd-fb1e-4620-9419-3f9a4fcecba5', belongs to osd '30' (osd fsid 'f4edb5cd-fb1e-4620-9419-3f9a4fcecba5'), placed on '/dev/vdb' device, which is not reflected in spec",
				"found ceph db partition '/dev/ceph-metadata/part-1', belongs to osd '20' (osd fsid 'vbsgs3a3-sdcv-casq-sd11-asd12dasczsf'), placed on '/dev/vdd' device, which is not reflected in spec",
				"found ceph db partition '/dev/vda14', belongs to osd '30' (osd fsid 'f4edb5cd-fb1e-4620-9419-3f9a4fcecba5'), placed on '/dev/vda' device, which is not reflected in spec",
			},
			expectedExtra: true,
		},
		{
			name: "nodespec 1 with filters - analyse is ok, not all devices in spec",
			node: cephv1.Node{
				Name: "node-1",
				Config: map[string]string{
					"metadataDevice": "/dev/vdd",
				},
				Selection: cephv1.Selection{
					DevicePathFilter: "^/dev/vd[ef]",
				},
			},
			expectedWarnings: []string{
				"found ceph block partition '/dev/ceph-992bbd78-3d8e-4cc3-93dc-eae387309364/osd-block-f4edb5cd-fb1e-4620-9419-3f9a4fcecba5', belongs to osd '30' (osd fsid 'f4edb5cd-fb1e-4620-9419-3f9a4fcecba5'), placed on '/dev/vdb' device, which is not reflected in spec",
				"found ceph db partition '/dev/vda14', belongs to osd '30' (osd fsid 'f4edb5cd-fb1e-4620-9419-3f9a4fcecba5'), placed on '/dev/vda' device, which is not reflected in spec",
			},
			expectedExtra: true,
		},
		{
			name: "nodespec 2 - analyse is ok, not all devices in spec and found stray",
			node: unitinputs.StorageNodesForAnalysisNotAllSpecified[1],
			expectedWarnings: []string{
				"found ceph block partition '/dev/ceph-0e03d5c6-d0e9-4f04-b9af-38d15e14369f/osd-block-61869d90-2c45-4f02-b7c3-96955f41e2ca', belongs to osd '2' (osd fsid '61869d90-2c45-4f02-b7c3-96955f41e2ca'), placed on '/dev/vde' device, which seems to be stray, can be cleaned up",
				"found ceph block partition '/dev/ceph-c5628abe-ae41-4c3d-bdc6-ef86c54bf78c/osd-block-69481cd1-38b1-42fd-ac07-06bf4d7c0e19', belongs to osd '0' (osd fsid '06bf4d7c-9603-41a4-b250-284ecf3ecb2f'), placed on '/dev/vdc' device, which seems to be stray, can be cleaned up",
				"found ceph block partition '/dev/ceph-dada9f25-41b4-4c26-9a20-448ac01e1d06/osd-block-7d09cceb-4de0-478e-9d8d-bd09cb0c904e', belongs to osd '5' (osd fsid 'af39b794-e1c6-41c0-8997-d6b6c631b8f2'), placed on '/dev/vdd' device, which is not reflected in spec",
				"found ceph block partition '/dev/ceph-dada9f25-41b4-4c26-9a20-448ac01e1d06/osd-block-ad76cf53-5cb5-48fe-a39a-343734f5ccde', belongs to osd '4' (osd fsid 'ad76cf53-5cb5-48fe-a39a-343734f5ccde'), placed on '/dev/vdd' device, which is not reflected in spec",
			},
			expectedExtra: true,
		},
		{
			name: "nodespec 2 with filters - analyse is ok, found stray partitions and not all in spec",
			node: cephv1.Node{
				Name: "node-2",
				Config: map[string]string{
					"osdsPerDevice": "2",
				},
				Selection: cephv1.Selection{
					DeviceFilter: "vdd",
				},
			},
			expectedWarnings: []string{
				"found ceph block partition '/dev/ceph-0e03d5c6-d0e9-4f04-b9af-38d15e14369f/osd-block-61869d90-2c45-4f02-b7c3-96955f41e2ca', belongs to osd '2' (osd fsid '61869d90-2c45-4f02-b7c3-96955f41e2ca'), placed on '/dev/vde' device, which seems to be stray, can be cleaned up",
				"found ceph block partition '/dev/ceph-c5628abe-ae41-4c3d-bdc6-ef86c54bf78c/osd-block-69481cd1-38b1-42fd-ac07-06bf4d7c0e19', belongs to osd '0' (osd fsid '06bf4d7c-9603-41a4-b250-284ecf3ecb2f'), placed on '/dev/vdc' device, which seems to be stray, can be cleaned up",
				"found ceph block partition '/dev/ceph-cf7c8b53-27c7-4cfc-94de-6ad4c7d9f92d/osd-block-af39b794-e1c6-41c0-8997-d6b6c631b8f2', belongs to osd '0' (osd fsid '69481cd1-38b1-42fd-ac07-06bf4d7c0e19'), placed on '/dev/vdb' device, which is not reflected in spec",
			},
			expectedExtra: true,
		},
		{
			name: "nodespec 1 - analyse is not ok, found devices not present on a node",
			node: cephv1.Node{
				Name: "node-1",
				Selection: cephv1.Selection{
					Devices: []cephv1.Device{
						{
							Name: "vdb",
							Config: map[string]string{
								"deviceClass":    "hdd",
								"metadataDevice": "/dev/vda14",
							},
						},
						{
							Name: "vdx",
							Config: map[string]string{
								"deviceClass": "hdd",
							},
						},
						{
							FullPath: "/dev/disk/by-path/pci-0000:00:0a.0",
							Config: map[string]string{
								"deviceClass": "hdd",
							},
						},
						{
							Name: "vdd",
							Config: map[string]string{
								"deviceClass": "hdd",
							},
						},
						{
							FullPath: "/dev/ceph-metadata/part-2",
							Config: map[string]string{
								"deviceClass": "hdd",
							},
						},
						{
							Name: "vdf",
							Config: map[string]string{
								"deviceClass":   "hdd",
								"osdsPerDevice": "4",
							},
						},
					},
				},
			},
			expectedIssues: []string{
				"failed to check device 'vdx' specified in spec: device '/dev/vdx' is not found on a node",
				"spec device '/dev/disk/by-path/pci-0000:00:0a.0' is duplication usage for item with device 'vdb' (devices matched by id)",
			},
		},
		{
			name: "nodespec 1 - analyse is not ok, incorrect block device usage",
			node: cephv1.Node{
				Name: "node-1",
				Selection: cephv1.Selection{
					Devices: []cephv1.Device{
						{
							Name: "vdb",
							Config: map[string]string{
								"deviceClass":    "hdd",
								"metadataDevice": "/dev/vda14",
							},
						},
						{
							Name: "vdd",
							Config: map[string]string{
								"deviceClass": "hdd",
							},
						},
						{
							FullPath: "/dev/ceph-metadata/part-2",
							Config: map[string]string{
								"deviceClass": "hdd",
							},
						},
						{
							Name: "vdf",
							Config: map[string]string{
								"deviceClass":   "hdd",
								"osdsPerDevice": "4",
							},
						},
					},
				},
			},
			expectedIssues: []string{
				"device '/dev/ceph-metadata/part-2' is specified as block device, but contains db partition '/dev/ceph-metadata/part-2'",
				"device '/dev/ceph-metadata/part-2' should have 1 osd(s), but actually found 0",
				"device 'vdd' is specified as block device, but contains db partition '/dev/ceph-metadata/part-1'",
				"device 'vdd' is specified as block device, but contains db partition '/dev/ceph-metadata/part-2'",
				"device 'vdd' should have 1 osd(s), but actually found 0",
				"device 'vdf' has no specified metadata device, but found related db partition '/dev/ceph-metadata/part-2' (osd 25)",
				"device 'vdf' should have 4 osd(s), but actually found 1",
			},
		},
		{
			name: "nodespec 1 - analyse is not ok, incorrect metadata device usage #1",
			node: cephv1.Node{
				Name: "node-1",
				Selection: cephv1.Selection{
					Devices: []cephv1.Device{
						{
							FullPath: "/dev/disk/by-path/pci-0000:00:0f.0",
							Config: map[string]string{
								"deviceClass":    "hdd",
								"metadataDevice": "/dev/ceph-metadata/part-1",
							},
						},
						{
							Name: "vdf",
							Config: map[string]string{
								"deviceClass":    "hdd",
								"metadataDevice": "/dev/ceph-metadata/part-1",
							},
						},
					},
				},
			},
			expectedIssues: []string{
				"spec device 'vdf' has metadata device '/dev/ceph-metadata/part-1', which is duplication use as meta for device '/dev/disk/by-path/pci-0000:00:0f.0'",
			},
		},
		{
			name: "nodespec 1 - analyse is not ok, incorrect metadata device usage #2",
			node: cephv1.Node{
				Name: "node-1",
				Selection: cephv1.Selection{
					Devices: []cephv1.Device{
						{
							Name: "vdb",
							Config: map[string]string{
								"deviceClass": "hdd",
							},
						},
					},
				},
			},
			expectedIssues: []string{
				"device 'vdb' has no specified metadata device, but found related db partition '/dev/vda14' (osd 30)",
			},
		},
		{
			name: "nodespec 1 - analyse is not ok, incorrect metadata device usage #3",
			node: cephv1.Node{
				Name: "node-1",
				Selection: cephv1.Selection{
					Devices: []cephv1.Device{
						{
							Name: "vdb",
							Config: map[string]string{
								"deviceClass":    "hdd",
								"metadataDevice": "vdx",
							},
						},
					},
				},
			},
			expectedIssues: []string{
				"metadata device 'vdx' specified for device 'vdb' is not found on a node",
			},
		},
		{
			name: "nodespec 1 - analyse is not ok, incorrect metadata device usage #4",
			node: cephv1.Node{
				Name: "node-1",
				Selection: cephv1.Selection{
					Devices: []cephv1.Device{
						{
							FullPath: "/dev/disk/by-path/pci-0000:00:0f.0",
							Config: map[string]string{
								"deviceClass":    "hdd",
								"metadataDevice": "vda",
							},
						},
					},
				},
			},
			expectedIssues: []string{
				"device '/dev/disk/by-path/pci-0000:00:0f.0' has unknown db partition '/dev/ceph-metadata/part-1', while expected 'vda' (osd 20)",
				"metadata device 'vda' is not found for osd '20' for device '/dev/disk/by-path/pci-0000:00:0f.0'",
			},
		},
		{
			name: "nodespec 1 with filters - analyse is not ok, found incorrect configuration",
			node: cephv1.Node{
				Name: "node-1",
				Config: map[string]string{
					"metadataDevice": "/dev/vdd",
				},
				Selection: cephv1.Selection{
					DevicePathFilter: "^/dev/disk/by-path/pci-0000:00:0[a-f].*",
				},
			},
			expectedWarnings: nil,
			expectedIssues: []string{
				"device '/dev/vdb' filtered by '^/dev/disk/by-path/pci-0000:00:0[a-f].*' has unknown db partition '/dev/vda14', while expected '/dev/vdd' (osd 30)",
				"device '/dev/vdc' filtered by '^/dev/disk/by-path/pci-0000:00:0[a-f].*' should have 1 osd(s), but actually found 0",
				"device '/dev/vdd' filtered by '^/dev/disk/by-path/pci-0000:00:0[a-f].*' is specified as block device, but contains db partition '/dev/ceph-metadata/part-1'",
				"device '/dev/vdd' filtered by '^/dev/disk/by-path/pci-0000:00:0[a-f].*' is specified as block device, but contains db partition '/dev/ceph-metadata/part-2'",
				"device '/dev/vdd' filtered by '^/dev/disk/by-path/pci-0000:00:0[a-f].*' should have 1 osd(s), but actually found 0",
				"device '/dev/vdd1' filtered by '^/dev/disk/by-path/pci-0000:00:0[a-f].*' should have 1 osd(s), but actually found 0",
				"metadata device '/dev/vdd' is not found for osd '20' for device '/dev/vdd' filtered by '^/dev/disk/by-path/pci-0000:00:0[a-f].*'",
				"metadata device '/dev/vdd' is not found for osd '25' for device '/dev/vdd' filtered by '^/dev/disk/by-path/pci-0000:00:0[a-f].*'",
				"metadata device '/dev/vdd' is not found for osd '30' for device '/dev/vdb' filtered by '^/dev/disk/by-path/pci-0000:00:0[a-f].*'",
			},
		},
		{
			name: "nodespec 1 with filters - analyse is not ok, found incorrect configuration #2",
			node: cephv1.Node{
				Name: "node-1",
				Config: map[string]string{
					"metadataDevice": "/dev/vdh",
				},
				Selection: cephv1.Selection{
					DevicePathFilter: "^/dev/vd[bef]",
				},
			},
			expectedWarnings: nil,
			expectedIssues: []string{
				"device '/dev/vdb' filtered by '^/dev/vd[bef]' has unknown db partition '/dev/vda14', while expected '/dev/vdh' (osd 30)",
				"device '/dev/vde' filtered by '^/dev/vd[bef]' has unknown db partition '/dev/ceph-metadata/part-1', while expected '/dev/vdh' (osd 20)",
				"device '/dev/vdf' filtered by '^/dev/vd[bef]' has unknown db partition '/dev/ceph-metadata/part-2', while expected '/dev/vdh' (osd 25)",
				"metadata device '/dev/vdh' is not found for osd '20' for device '/dev/vde' filtered by '^/dev/vd[bef]'",
				"metadata device '/dev/vdh' is not found for osd '25' for device '/dev/vdf' filtered by '^/dev/vd[bef]'",
				"metadata device '/dev/vdh' is not found for osd '30' for device '/dev/vdb' filtered by '^/dev/vd[bef]'",
			},
		},
		{
			name: "nodespec 2 with filters - analyse is not ok, found incorrect number partitions",
			node: cephv1.Node{
				Name: "node-2",
				Selection: cephv1.Selection{
					DeviceFilter: "^vd[bd]",
				},
			},
			expectedWarnings: nil,
			expectedIssues: []string{
				"device '/dev/vdd' filtered by '^vd[bd]' should have 1 osd(s), but actually found 2",
			},
		},
		{
			name: "nodespec without storage configuration - found only extra osds",
			node: cephv1.Node{
				Name: "node-1",
				Selection: cephv1.Selection{
					Devices: []cephv1.Device{},
				},
			},
			expectedWarnings: []string{
				"found ceph block partition '/dev/ceph-21312wds-sdfv-vs3f-scv3-sdfdsg23edaa/osd-block-vbsgs3a3-sdcv-casq-sd11-asd12dasczsf', belongs to osd '20' (osd fsid 'vbsgs3a3-sdcv-casq-sd11-asd12dasczsf'), placed on '/dev/vde' device, which is not reflected in spec",
				"found ceph block partition '/dev/ceph-2efce189-afb7-452f-bd32-c73b5017a0da/osd-block-d49fd9bf-d2dd-4c3d-824d-87f3f17ea44a', belongs to osd '25' (osd fsid 'd49fd9bf-d2dd-4c3d-824d-87f3f17ea44a'), placed on '/dev/vdf' device, which is not reflected in spec",
				"found ceph block partition '/dev/ceph-992bbd78-3d8e-4cc3-93dc-eae387309364/osd-block-f4edb5cd-fb1e-4620-9419-3f9a4fcecba5', belongs to osd '30' (osd fsid 'f4edb5cd-fb1e-4620-9419-3f9a4fcecba5'), placed on '/dev/vdb' device, which is not reflected in spec",
				"found ceph db partition '/dev/ceph-metadata/part-1', belongs to osd '20' (osd fsid 'vbsgs3a3-sdcv-casq-sd11-asd12dasczsf'), placed on '/dev/vdd' device, which is not reflected in spec",
				"found ceph db partition '/dev/ceph-metadata/part-2', belongs to osd '25' (osd fsid 'd49fd9bf-d2dd-4c3d-824d-87f3f17ea44a'), placed on '/dev/vdd' device, which is not reflected in spec",
				"found ceph db partition '/dev/vda14', belongs to osd '30' (osd fsid 'f4edb5cd-fb1e-4620-9419-3f9a4fcecba5'), placed on '/dev/vda' device, which is not reflected in spec",
			},
			expectedExtra: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			issues, warnings, extra := runSpecAnalysis(test.node, daemonInfoMapping[test.node.Name], osdClusterDetails[test.node.Name])
			assert.Equal(t, test.expectedIssues, issues)
			assert.Equal(t, test.expectedWarnings, warnings)
			assert.Equal(t, test.expectedExtra, extra)
		})
	}
}
