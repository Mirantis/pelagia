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
	"encoding/json"
	"testing"

	"github.com/pkg/errors"
	cephv1 "github.com/rook/rook/pkg/apis/ceph.rook.io/v1"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"

	lcmv1alpha1 "github.com/Mirantis/pelagia/pkg/apis/ceph.pelagia.lcm/v1alpha1"
	lcmcommon "github.com/Mirantis/pelagia/pkg/common"
	faketestclients "github.com/Mirantis/pelagia/test/unit/clients"
	unitinputs "github.com/Mirantis/pelagia/test/unit/inputs"
	lcmdiskdaemoninput "github.com/Mirantis/pelagia/test/unit/inputs/disk-daemon"
)

func TestGetOsdsForCleanup(t *testing.T) {
	cephClusterNoNodes := unitinputs.ReefCephClusterReady.DeepCopy()
	cephClusterNoNodes.Spec.Storage.Nodes = nil
	cephClusterReducedNodes := unitinputs.ReefCephClusterReady.DeepCopy()
	cephClusterReducedNodes.Spec.Storage.Nodes = unitinputs.StorageNodesForRequestReduced
	cephClusterFilteredDevices := unitinputs.ReefCephClusterReady.DeepCopy()
	cephClusterFilteredDevices.Spec.Storage.Nodes = unitinputs.StorageNodesForRequestFiltered

	getTaskConfig := func(nodesInTask map[string]lcmv1alpha1.NodeCleanUpSpec, cephCluster *cephv1.CephCluster, specAnalysis map[string]lcmv1alpha1.DaemonStatus) taskConfig {
		newTaskConfig := taskConfig{
			cephCluster: cephCluster,
			cephHealthOsdAnalysis: &lcmv1alpha1.OsdSpecAnalysisState{
				SpecAnalysis: specAnalysis,
			},
		}
		// fill defaults
		if nodesInTask != nil {
			newTaskConfig.task = unitinputs.CephOsdRemoveTaskOnValidation.DeepCopy()
			newTaskConfig.task.Spec = &lcmv1alpha1.CephOsdRemoveTaskSpec{Nodes: nodesInTask}
		} else {
			newTaskConfig.task = unitinputs.CephOsdRemoveTaskOnValidation
		}
		if newTaskConfig.cephCluster == nil {
			newTaskConfig.cephCluster = &unitinputs.ReefCephClusterReady
		}
		if newTaskConfig.cephHealthOsdAnalysis.SpecAnalysis == nil {
			newTaskConfig.cephHealthOsdAnalysis.SpecAnalysis = unitinputs.OsdStorageSpecAnalysisOk
		}
		return newTaskConfig
	}

	tcEmptytaskFullnodesOk := getTaskConfig(nil, nil, nil)
	tcEmptytaskNotalldevsOk := getTaskConfig(nil, &unitinputs.ReefCephClusterHasHealthIssues, nil)
	tcEmptytaskNonodes := getTaskConfig(nil, cephClusterNoNodes, map[string]lcmv1alpha1.DaemonStatus{})
	tcFullremoveNonodes := getTaskConfig(unitinputs.RequestRemoveFullNodeRemove, cephClusterNoNodes, map[string]lcmv1alpha1.DaemonStatus{})

	nodesListLabeledAvailable := unitinputs.GetNodesList([]unitinputs.NodeAttrs{{Name: "node-1", Labeled: true}, {Name: "node-2", Labeled: true}})
	nodesListNotLabeledAvailable := unitinputs.GetNodesList([]unitinputs.NodeAttrs{{Name: "node-1"}, {Name: "node-2"}})
	nodesListNotLabeledUnreachable := unitinputs.GetNodesList([]unitinputs.NodeAttrs{{Name: "node-1", Unreachable: true}, {Name: "node-2", Unreachable: true}})
	nodesListNode1LabeledAvailable := unitinputs.GetNodesList([]unitinputs.NodeAttrs{{Name: "node-1", Labeled: true}})
	nodesListNode2LabeledAvailable := unitinputs.GetNodesList([]unitinputs.NodeAttrs{{Name: "node-2", Labeled: true}})

	tests := []struct {
		name               string
		taskConfig         taskConfig
		hostsFromCluster   string
		osdsMetadata       string
		osdInfo            string
		nodeList           v1.NodeList
		nodeOsdReport      map[string]*lcmcommon.DiskDaemonReport
		removeAllLVMs      bool
		expectedRemoveInfo *lcmv1alpha1.TaskRemoveInfo
	}{
		{
			name:             "no nodes in request - spec analysis is not ok",
			hostsFromCluster: unitinputs.CephOsdTreeOutput,
			osdsMetadata:     unitinputs.CephOsdMetadataOutputNoStray,
			osdInfo:          unitinputs.CephOsdInfoOutputNoStray,
			taskConfig: getTaskConfig(nil, nil, map[string]lcmv1alpha1.DaemonStatus{
				"node-2": {Status: lcmv1alpha1.DaemonStateFailed, Issues: []string{"some problems with spec"}},
			}),
			nodeList: nodesListLabeledAvailable,
			expectedRemoveInfo: &lcmv1alpha1.TaskRemoveInfo{
				CleanupMap: map[string]lcmv1alpha1.HostMapping{},
				Issues: []string{
					"[node 'node-1'] spec analyse status is not available yet",
					"[node 'node-2'] spec analyse status has failed, resolve it first",
				},
				Warnings: []string{},
			},
		},
		{
			name:             "no nodes in request - failed to get disk daemon node reports",
			hostsFromCluster: unitinputs.CephOsdTreeOutput,
			osdsMetadata:     unitinputs.CephOsdMetadataOutputNoStray,
			osdInfo:          unitinputs.CephOsdInfoOutputNoStray,
			taskConfig:       tcEmptytaskFullnodesOk,
			nodeList:         nodesListLabeledAvailable,
			expectedRemoveInfo: &lcmv1alpha1.TaskRemoveInfo{
				CleanupMap: map[string]lcmv1alpha1.HostMapping{},
				Issues: []string{
					"[node 'node-1'] failed to get node osds report: Retries (1/1) exceeded: failed to parse output for command 'pelagia-disk-daemon --osd-report --port 9999': invalid character '|' looking for beginning of object key string",
					"[node 'node-2'] failed to get node osds report: Retries (1/1) exceeded: failed to parse output for command 'pelagia-disk-daemon --osd-report --port 9999': invalid character '|' looking for beginning of object key string",
				},
				Warnings: []string{},
			},
		},
		{
			name:             "no nodes in request - nodes has issues with node report",
			hostsFromCluster: unitinputs.CephOsdTreeOutput,
			osdsMetadata:     unitinputs.CephOsdMetadataOutputNoStray,
			osdInfo:          unitinputs.CephOsdInfoOutputNoStray,
			taskConfig:       tcEmptytaskFullnodesOk,
			nodeList: unitinputs.GetNodesList([]unitinputs.NodeAttrs{
				{Name: "node-1", Labeled: true}, {Name: "node-2", Labeled: true}, {Name: "node-3", Labeled: true},
			}),
			nodeOsdReport: map[string]*lcmcommon.DiskDaemonReport{
				"node-1": {State: lcmcommon.DiskDaemonStateFailed, Issues: []string{"failed to run lsblk command"}},
				"node-2": {State: lcmcommon.DiskDaemonStateInProgress},
				"node-3": {State: lcmcommon.DiskDaemonStateOk},
			},
			expectedRemoveInfo: &lcmv1alpha1.TaskRemoveInfo{
				CleanupMap: map[string]lcmv1alpha1.HostMapping{},
				Issues: []string{
					"[node 'node-1'] failed to run lsblk command",
					"[node 'node-2'] failed to get node osds report: Retries (1/1) exceeded: node report is not prepared yet",
					"[node 'node-3'] node osds report is not available, check daemon logs on related node",
				},
				Warnings: []string{},
			},
		},
		{
			name:             "no nodes in request - all is aligned and nothing to do",
			hostsFromCluster: unitinputs.CephOsdTreeOutput,
			osdsMetadata:     unitinputs.CephOsdMetadataOutputNoStray,
			osdInfo:          unitinputs.CephOsdInfoOutputNoStray,
			taskConfig:       tcEmptytaskFullnodesOk,
			nodeList:         nodesListLabeledAvailable,
			nodeOsdReport: map[string]*lcmcommon.DiskDaemonReport{
				"node-1": &unitinputs.DiskDaemonReportOkNode1,
				"node-2": &unitinputs.DiskDaemonReportOkNode2,
			},
			expectedRemoveInfo: unitinputs.EmptyRemoveMap,
		},
		{
			name:             "no nodes in request - some osds ready to remove",
			hostsFromCluster: unitinputs.CephOsdTreeOutput,
			osdsMetadata:     unitinputs.CephOsdMetadataOutputNoStray,
			osdInfo:          unitinputs.CephOsdInfoOutputNoStray,
			taskConfig:       tcEmptytaskNotalldevsOk,
			nodeList:         nodesListLabeledAvailable,
			nodeOsdReport: map[string]*lcmcommon.DiskDaemonReport{
				"node-1": &unitinputs.DiskDaemonReportOkNode1,
				"node-2": &unitinputs.DiskDaemonReportOkNode2,
			},
			expectedRemoveInfo: unitinputs.DevNotInSpecRemoveMap,
		},
		{
			name:             "no nodes in request - some osds ready to remove 2",
			hostsFromCluster: unitinputs.CephOsdTreeOutput,
			osdsMetadata:     unitinputs.CephOsdMetadataOutputNoStray,
			osdInfo:          unitinputs.CephOsdInfoOutputNoStray,
			taskConfig: getTaskConfig(
				nil,
				func() *cephv1.CephCluster {
					cluster := unitinputs.ReefCephClusterHasHealthIssues.DeepCopy()
					cluster.Spec.Storage.Nodes[0].Devices[1].Config["metadataDevice"] = "/dev/vdd"
					return cluster
				}(), nil),
			nodeList: nodesListLabeledAvailable,
			nodeOsdReport: map[string]*lcmcommon.DiskDaemonReport{
				"node-1": &unitinputs.DiskDaemonReportOkNode1,
				"node-2": &unitinputs.DiskDaemonReportOkNode2,
			},
			expectedRemoveInfo: unitinputs.DevNotInSpecRemoveMap,
		},
		{
			name:             "no nodes in request - device filters, some osds ready to remove",
			hostsFromCluster: unitinputs.CephOsdTreeOutput,
			osdsMetadata:     unitinputs.CephOsdMetadataOutputNoStray,
			osdInfo:          unitinputs.CephOsdInfoOutputNoStray,
			taskConfig: getTaskConfig(
				nil,
				func() *cephv1.CephCluster {
					cluster := cephClusterFilteredDevices.DeepCopy()
					cluster.Spec.Storage.Nodes[0].DevicePathFilter = ""
					cluster.Spec.Storage.Nodes[0].DeviceFilter = "^vd[fe]$"
					return cluster
				}(), nil),
			nodeList: nodesListLabeledAvailable,
			nodeOsdReport: map[string]*lcmcommon.DiskDaemonReport{
				"node-1": &unitinputs.DiskDaemonReportOkNode1,
				"node-2": &unitinputs.DiskDaemonReportOkNode2,
			},
			expectedRemoveInfo: &lcmv1alpha1.TaskRemoveInfo{
				CleanupMap: map[string]lcmv1alpha1.HostMapping{
					"node-1": {
						OsdMapping: map[string]lcmv1alpha1.OsdMapping{
							"30": unitinputs.AdaptOsdMapping("node-1", "30",
								map[string]bool{"inCrush": true}, map[string]map[string]bool{"/dev/vda": {}, "/dev/vdb": {"zap": true}}),
						},
					},
					"node-2": unitinputs.DevNotInSpecRemoveMap.CleanupMap["node-2"],
				},
				Issues: []string{},
				Warnings: []string{
					"[node 'node-1'] found osd db partition '/dev/vda14' for osd '30', which is created not by rook, skipping disk/partition zap",
					"[node 'node-1'] found physical osd db partition '/dev/vda14' for osd '30'",
				},
			},
		},
		{
			name:             "no nodes in request - some osds ready to remove when devices lost and not in spec",
			hostsFromCluster: unitinputs.CephOsdTreeOutput,
			osdsMetadata:     unitinputs.CephOsdMetadataOutputNoStray,
			osdInfo:          unitinputs.CephOsdInfoOutputNoStray,
			taskConfig:       tcEmptytaskNotalldevsOk,
			nodeList:         nodesListLabeledAvailable,
			nodeOsdReport: map[string]*lcmcommon.DiskDaemonReport{
				"node-1": &unitinputs.DiskDaemonReportOkNode1SomeDevLost,
				"node-2": &unitinputs.DiskDaemonReportOkNode2SomeDevLost,
			},
			expectedRemoveInfo: &lcmv1alpha1.TaskRemoveInfo{
				CleanupMap: map[string]lcmv1alpha1.HostMapping{
					"node-1": {
						OsdMapping: map[string]lcmv1alpha1.OsdMapping{
							"20": unitinputs.AdaptOsdMapping("node-1", "20",
								map[string]bool{"inCrush": true}, map[string]map[string]bool{"/dev/vde": {"zap": true}, "/dev/vdd": {"lost": true}}),
						},
					},
					"node-2": {
						OsdMapping: map[string]lcmv1alpha1.OsdMapping{
							"4": unitinputs.AdaptOsdMapping("node-2", "4",
								map[string]bool{"inCrush": true}, map[string]map[string]bool{"/dev/vdd": {"lost": true}}),
							"5": unitinputs.AdaptOsdMapping("node-2", "5",
								map[string]bool{"inCrush": true}, map[string]map[string]bool{"/dev/vdd": {"lost": true}}),
						},
					},
				},
				Issues:   []string{},
				Warnings: []string{},
			},
		},
		{
			name:             "no nodes in request - some nodes is not in spec",
			hostsFromCluster: unitinputs.CephOsdTreeOutput,
			osdsMetadata:     unitinputs.CephOsdMetadataOutputNoStray,
			osdInfo:          unitinputs.CephOsdInfoOutputNoStray,
			taskConfig:       tcEmptytaskNonodes,
			nodeList:         nodesListLabeledAvailable,
			nodeOsdReport: map[string]*lcmcommon.DiskDaemonReport{
				"node-1": &unitinputs.DiskDaemonReportOkNode1,
				"node-2": &unitinputs.DiskDaemonReportOkNode2,
			},
			expectedRemoveInfo: unitinputs.FullNodesRemoveMap,
		},
		{
			name:             "no nodes in request - some nodes is not in spec and have lost devices",
			hostsFromCluster: unitinputs.CephOsdTreeOutput,
			osdsMetadata:     unitinputs.CephOsdMetadataOutputNoStray,
			osdInfo:          unitinputs.CephOsdInfoOutputNoStray,
			taskConfig: getTaskConfig(nil, cephClusterReducedNodes, map[string]lcmv1alpha1.DaemonStatus{
				"node-2": unitinputs.OsdStorageSpecAnalysisOk["node-2"],
			}),
			nodeList: nodesListLabeledAvailable,
			nodeOsdReport: map[string]*lcmcommon.DiskDaemonReport{
				"node-1": &unitinputs.DiskDaemonReportOkNode1SomeDevLost,
				"node-2": &unitinputs.DiskDaemonReportOkNode2,
			},
			expectedRemoveInfo: &lcmv1alpha1.TaskRemoveInfo{
				CleanupMap: map[string]lcmv1alpha1.HostMapping{
					"node-1": {
						CompleteCleanup: true,
						OsdMapping: map[string]lcmv1alpha1.OsdMapping{
							"20": unitinputs.AdaptOsdMapping("node-1", "20",
								map[string]bool{"inCrush": true}, map[string]map[string]bool{"/dev/vde": {"zap": true}, "/dev/vdd": {"lost": true}}),
							"25": unitinputs.AdaptOsdMapping("node-1", "25",
								map[string]bool{"inCrush": true}, map[string]map[string]bool{"/dev/vdf": {"zap": true}, "/dev/vdd": {"lost": true}}),
							"30": unitinputs.AdaptOsdMapping("node-1", "30",
								map[string]bool{"inCrush": true}, map[string]map[string]bool{"/dev/vda": {}, "/dev/vdb": {"zap": true}}),
						},
					},
				},
				Issues: []string{},
				Warnings: []string{
					"[node 'node-1'] found osd db partition '/dev/vda14' for osd '30', which is created not by rook, skipping disk/partition zap",
					"[node 'node-1'] found physical osd db partition '/dev/vda14' for osd '30'",
				},
			},
		},
		{
			name:             "no nodes in request - found some strays in crush only",
			hostsFromCluster: unitinputs.CephOsdTreeOutput,
			osdsMetadata:     unitinputs.CephOsdMetadataOutput,
			osdInfo:          unitinputs.CephOsdInfoOutput,
			taskConfig:       tcEmptytaskFullnodesOk,
			nodeList:         nodesListLabeledAvailable,
			nodeOsdReport: map[string]*lcmcommon.DiskDaemonReport{
				"node-1": &unitinputs.DiskDaemonReportOkNode1,
				"node-2": &unitinputs.DiskDaemonReportOkNode2,
			},
			expectedRemoveInfo: unitinputs.StrayOnlyInCrushRemoveMap,
		},
		{
			name:             "no nodes in request - found some strays on node only",
			hostsFromCluster: unitinputs.CephOsdTreeOutput,
			osdsMetadata:     unitinputs.CephOsdMetadataOutputNoStray,
			osdInfo:          unitinputs.CephOsdInfoOutputNoStray,
			taskConfig:       tcEmptytaskFullnodesOk,
			nodeList:         nodesListLabeledAvailable,
			nodeOsdReport: map[string]*lcmcommon.DiskDaemonReport{
				"node-1": &unitinputs.DiskDaemonReportOkNode1,
				"node-2": unitinputs.DiskDaemonNodeReportWithStrayOkNode2(false),
			},
			expectedRemoveInfo: unitinputs.StrayOnlyOnNodeRemoveMap,
		},
		{
			name:             "no nodes in request - found some strays on node and in crush",
			hostsFromCluster: unitinputs.CephOsdTreeOutput,
			osdsMetadata:     unitinputs.CephOsdMetadataOutput,
			osdInfo:          unitinputs.CephOsdInfoOutput,
			taskConfig:       tcEmptytaskFullnodesOk,
			nodeList:         nodesListLabeledAvailable,
			nodeOsdReport: map[string]*lcmcommon.DiskDaemonReport{
				"node-1": &unitinputs.DiskDaemonReportOkNode1,
				"node-2": unitinputs.DiskDaemonNodeReportWithStrayOkNode2(true),
			},
			expectedRemoveInfo: unitinputs.StrayOnNodeAndInCrushRemoveMap,
		},
		{
			name:             "no nodes in request - some nodes are not labeled and not in spec but available",
			hostsFromCluster: unitinputs.CephOsdTreeOutput,
			osdsMetadata:     unitinputs.CephOsdMetadataOutput,
			osdInfo:          unitinputs.CephOsdInfoOutput,
			taskConfig:       getTaskConfig(nil, cephClusterReducedNodes, map[string]lcmv1alpha1.DaemonStatus{}),
			nodeList:         nodesListNotLabeledAvailable,
			expectedRemoveInfo: &lcmv1alpha1.TaskRemoveInfo{
				CleanupMap: map[string]lcmv1alpha1.HostMapping{
					"node-1":  unitinputs.NotLabeledNodesFullRemoveMap.CleanupMap["node-1"],
					"__stray": unitinputs.StrayOnlyInCrushRemoveMap.CleanupMap["__stray"],
				},
				Issues: []string{},
				Warnings: []string{
					"[node 'node-1'] node is available, but has no disk daemon running, device cleanup jobs will be skipped",
					"[node 'node-2'] node is available and present in spec, but has no disk daemon running, unable to run auto cleanup, skipping",
					"[stray] detected stray osds, but impossible to determine related host/device (probably disk(s) removed or host(s) down), device cleanup jobs will be skipped",
				},
			},
		},
		{
			name:             "no nodes in request - some nodes are not available",
			hostsFromCluster: unitinputs.CephOsdTreeOutput,
			osdsMetadata:     unitinputs.CephOsdMetadataOutput,
			osdInfo:          unitinputs.CephOsdInfoOutput,
			taskConfig:       getTaskConfig(nil, cephClusterReducedNodes, map[string]lcmv1alpha1.DaemonStatus{}),
			nodeList:         nodesListNotLabeledUnreachable,
			expectedRemoveInfo: &lcmv1alpha1.TaskRemoveInfo{
				CleanupMap: map[string]lcmv1alpha1.HostMapping{
					"node-1":  unitinputs.NotAvailableNodesFullRemoveMap.CleanupMap["node-1"],
					"__stray": unitinputs.StrayOnlyInCrushRemoveMap.CleanupMap["__stray"],
				},
				Issues: []string{},
				Warnings: []string{
					"[node 'node-1'] node is not available, device cleanup jobs will be skipped",
					"[node 'node-2'] node is present in spec, but is not available, unable to auto detect osds to remove, please specify manually",
					"[stray] detected stray osds, but impossible to determine related host/device (probably disk(s) removed or host(s) down), device cleanup jobs will be skipped",
				},
			},
		},
		{
			name:             "no nodes in request - nodes are not present in crush",
			hostsFromCluster: `{"nodes": []}`,
			osdsMetadata:     `[{"id": 2}]`,
			osdInfo:          `[{"osd": 2, "uuid": "61869d90-2c45-4f02-b7c3-96955f41e2ca"}]`,
			taskConfig:       tcEmptytaskNonodes,
			nodeList:         unitinputs.GetNodesList([]unitinputs.NodeAttrs{{Name: "node-1"}, {Name: "node-2", Labeled: true}}),
			nodeOsdReport: map[string]*lcmcommon.DiskDaemonReport{
				"node-2": &unitinputs.DiskDaemonReportOkNode2,
			},
			expectedRemoveInfo: &lcmv1alpha1.TaskRemoveInfo{
				CleanupMap: map[string]lcmv1alpha1.HostMapping{
					"node-2": {
						CompleteCleanup: true,
						OsdMapping: map[string]lcmv1alpha1.OsdMapping{
							"0.69481cd1-38b1-42fd-ac07-06bf4d7c0e19.__stray": unitinputs.AdaptOsdMapping(
								"node-2", "0", nil, map[string]map[string]bool{"/dev/vdb": {"zap": true}}),
							"4.ad76cf53-5cb5-48fe-a39a-343734f5ccde.__stray": unitinputs.AdaptOsdMapping(
								"node-2", "4", nil, map[string]map[string]bool{"/dev/vdd": {"zap": true}}),
							"5.af39b794-e1c6-41c0-8997-d6b6c631b8f2.__stray": unitinputs.AdaptOsdMapping(
								"node-2", "5", nil, map[string]map[string]bool{"/dev/vdd": {"zap": true}}),
						},
					},
					"__stray": unitinputs.StrayOnlyInCrushRemoveMap.CleanupMap["__stray"],
				},
				Issues: []string{},
				Warnings: []string{
					"[node 'node-2'] found partition with stray osd uuid '69481cd1-38b1-42fd-ac07-06bf4d7c0e19', id '0', will be cleaned up",
					"[node 'node-2'] found partition with stray osd uuid 'ad76cf53-5cb5-48fe-a39a-343734f5ccde', id '4', will be cleaned up",
					"[node 'node-2'] found partition with stray osd uuid 'af39b794-e1c6-41c0-8997-d6b6c631b8f2', id '5', will be cleaned up",
					"[stray] detected stray osds, but impossible to determine related host/device (probably disk(s) removed or host(s) down), device cleanup jobs will be skipped",
				},
			},
		},
		{
			name:             "no nodes in request - remove node which are only in crush",
			hostsFromCluster: unitinputs.CephOsdTreeOutputNoOsdsOnHost,
			osdsMetadata:     `[]`,
			osdInfo:          `[]`,
			taskConfig:       tcEmptytaskNonodes,
			nodeList:         nodesListNotLabeledAvailable,
			expectedRemoveInfo: &lcmv1alpha1.TaskRemoveInfo{
				CleanupMap: map[string]lcmv1alpha1.HostMapping{
					"node-1": {
						CompleteCleanup:   true,
						VolumesInfoMissed: true,
						OsdMapping:        map[string]lcmv1alpha1.OsdMapping{},
					},
					"node-2": {
						CompleteCleanup:   true,
						VolumesInfoMissed: true,
						OsdMapping:        map[string]lcmv1alpha1.OsdMapping{},
					},
				},
				Issues: []string{},
				Warnings: []string{
					"[node 'node-1'] node is available, but has no disk daemon running, device cleanup jobs will be skipped",
					"[node 'node-2'] node is available, but has no disk daemon running, device cleanup jobs will be skipped",
				},
			},
		},
		{
			name:             "no nodes in request - remove just stray from crush map",
			hostsFromCluster: `{"nodes": []}`,
			osdsMetadata:     `[{"id": 2}]`,
			osdInfo:          `[{"osd": 2, "uuid": "61869d90-2c45-4f02-b7c3-96955f41e2ca"}]`,
			taskConfig:       getTaskConfig(nil, nil, map[string]lcmv1alpha1.DaemonStatus{}),
			nodeList:         nodesListNotLabeledUnreachable,
			expectedRemoveInfo: &lcmv1alpha1.TaskRemoveInfo{
				CleanupMap: unitinputs.StrayOnlyInCrushRemoveMap.CleanupMap,
				Issues:     []string{},
				Warnings: []string{
					"[node 'node-1'] node is present in spec, but is not available, unable to auto detect osds to remove, please specify manually",
					"[node 'node-2'] node is present in spec, but is not available, unable to auto detect osds to remove, please specify manually",
					"[stray] detected stray osds, but impossible to determine related host/device (probably disk(s) removed or host(s) down), device cleanup jobs will be skipped",
				},
			},
		},
		{
			name:             "no nodes in request - device filters, some osds ready to remove",
			hostsFromCluster: unitinputs.CephOsdTreeOutput,
			osdsMetadata:     unitinputs.CephOsdMetadataOutputNoStray,
			osdInfo:          unitinputs.CephOsdInfoOutputNoStray,
			taskConfig:       getTaskConfig(nil, cephClusterFilteredDevices, nil),
			nodeList:         nodesListLabeledAvailable,
			nodeOsdReport: map[string]*lcmcommon.DiskDaemonReport{
				"node-1": &unitinputs.DiskDaemonReportOkNode1,
				"node-2": &unitinputs.DiskDaemonReportOkNode2,
			},
			expectedRemoveInfo: &lcmv1alpha1.TaskRemoveInfo{
				CleanupMap: map[string]lcmv1alpha1.HostMapping{
					"node-1": {
						OsdMapping: map[string]lcmv1alpha1.OsdMapping{
							"30": unitinputs.AdaptOsdMapping("node-1", "30", map[string]bool{"inCrush": true},
								map[string]map[string]bool{"/dev/vda": {}, "/dev/vdb": {"zap": true}}),
						},
					},
					"node-2": unitinputs.DevNotInSpecRemoveMap.CleanupMap["node-2"],
				},
				Issues: []string{},
				Warnings: []string{
					"[node 'node-1'] found osd db partition '/dev/vda14' for osd '30', which is created not by rook, skipping disk/partition zap",
					"[node 'node-1'] found physical osd db partition '/dev/vda14' for osd '30'",
				},
			},
		},
		{
			name:             "no nodes in request - spec analysis skipped, lcm skipped",
			hostsFromCluster: unitinputs.CephOsdTreeOutput,
			osdsMetadata:     unitinputs.CephOsdMetadataOutputNoStray,
			osdInfo:          unitinputs.CephOsdInfoOutputNoStray,
			taskConfig: getTaskConfig(nil, &unitinputs.ReefCephClusterHasHealthIssues,
				map[string]lcmv1alpha1.DaemonStatus{
					"node-1": {Status: lcmv1alpha1.DaemonStateSkipped},
					"node-2": {Status: lcmv1alpha1.DaemonStateSkipped},
				},
			),
			nodeList: nodesListLabeledAvailable,
			expectedRemoveInfo: &lcmv1alpha1.TaskRemoveInfo{
				CleanupMap: map[string]lcmv1alpha1.HostMapping{},
				Issues:     []string{},
				Warnings: []string{
					"[node 'node-1'] spec analyse status skipped, skipping lcm actions",
					"[node 'node-2'] spec analyse status skipped, skipping lcm actions",
				},
			},
		},
		{
			name:             "present nodes in request - remove nodes which are not in crush and not labeled",
			hostsFromCluster: `{"nodes": []}`,
			osdsMetadata:     `[]`,
			osdInfo:          `[]`,
			taskConfig: getTaskConfig(map[string]lcmv1alpha1.NodeCleanUpSpec{
				"node-y": unitinputs.RequestRemoveByOsdID["node-1"],
				"node-x": unitinputs.RequestRemoveByDevice["node-2"],
				"node-z": unitinputs.RequestRemoveFullNodeRemove["node-2"],
				"node-w": unitinputs.RequestRemoveFullNodeRemove["node-1"],
				"node-1": {CleanupStrayPartitions: true},
				"node-2": {CleanupStrayPartitions: true},
			}, nil, map[string]lcmv1alpha1.DaemonStatus{}),
			nodeList: unitinputs.GetNodesList([]unitinputs.NodeAttrs{{Name: "node-1", Unreachable: true}, {Name: "node-2"}, {Name: "node-z"}}),
			expectedRemoveInfo: &lcmv1alpha1.TaskRemoveInfo{
				CleanupMap: map[string]lcmv1alpha1.HostMapping{},
				Issues:     []string{},
				Warnings: []string{
					"[node 'node-1'] node is not available, cleanup stray paritions is not possible, skipping",
					"[node 'node-2'] node is available, but has no disk daemon running, cleanup stray paritions is not possible, use by id or complete remove, skipping",
					"[node 'node-w'] node is not present in Ceph cluster crush map, skipping",
					"[node 'node-x'] node is not present in Ceph cluster crush map, skipping",
					"[node 'node-y'] node is not present in Ceph cluster crush map, skipping",
					"[node 'node-z'] node is not present in Ceph cluster crush map, skipping",
				},
			},
		},
		{
			name:             "present nodes in request - remove node which are only in crush",
			hostsFromCluster: unitinputs.CephOsdTreeOutputNoOsdsOnHost,
			osdsMetadata:     `[{"id": 2}]`,
			osdInfo:          `[{"osd": 2, "uuid": "61869d90-2c45-4f02-b7c3-96955f41e2ca"}]`,
			taskConfig:       tcFullremoveNonodes,
			nodeList:         nodesListNotLabeledAvailable,
			expectedRemoveInfo: &lcmv1alpha1.TaskRemoveInfo{
				CleanupMap: map[string]lcmv1alpha1.HostMapping{
					"node-1": {
						CompleteCleanup:   true,
						VolumesInfoMissed: true,
						OsdMapping:        map[string]lcmv1alpha1.OsdMapping{},
					},
					"node-2": {
						DropFromCrush:     true,
						VolumesInfoMissed: true,
						OsdMapping:        map[string]lcmv1alpha1.OsdMapping{},
					},
				},
				Issues: []string{},
				Warnings: []string{
					"[node 'node-1'] node is available, but has no disk daemon running, device cleanup jobs will be skipped",
					"[node 'node-2'] node is available, but has no disk daemon running, device cleanup jobs will be skipped",
				},
			},
		},
		{
			name:             "present nodes in request - remove nodes completely which are in spec",
			hostsFromCluster: unitinputs.CephOsdTreeOutput,
			osdsMetadata:     unitinputs.CephOsdMetadataOutputNoStray,
			osdInfo:          unitinputs.CephOsdInfoOutputNoStray,
			taskConfig:       getTaskConfig(unitinputs.RequestRemoveFullNodeRemove, nil, map[string]lcmv1alpha1.DaemonStatus{}),
			nodeList:         unitinputs.GetNodesList([]unitinputs.NodeAttrs{{Name: "node-1", Unreachable: true}, {Name: "node-2"}}),
			expectedRemoveInfo: &lcmv1alpha1.TaskRemoveInfo{
				CleanupMap: map[string]lcmv1alpha1.HostMapping{},
				Issues:     []string{},
				Warnings: []string{
					"[node 'node-1'] node is present in spec, complete host remove from crush map is not possible, skipping",
					"[node 'node-2'] node is present in spec, complete host remove from crush map is not possible, skipping",
				},
			},
		},
		{
			name:             "present nodes in request - remove nodes which are not labeled, in spec and available",
			hostsFromCluster: unitinputs.CephOsdTreeOutput,
			osdsMetadata:     unitinputs.CephOsdMetadataOutputNoStray,
			osdInfo:          unitinputs.CephOsdInfoOutputNoStray,
			taskConfig: getTaskConfig(map[string]lcmv1alpha1.NodeCleanUpSpec{
				"node-1": unitinputs.RequestRemoveByOsdID["node-1"],
				"node-2": unitinputs.RequestRemoveByDevice["node-2"],
			}, nil, map[string]lcmv1alpha1.DaemonStatus{}),
			nodeList: nodesListNotLabeledAvailable,
			expectedRemoveInfo: &lcmv1alpha1.TaskRemoveInfo{
				CleanupMap: map[string]lcmv1alpha1.HostMapping{},
				Issues:     []string{},
				Warnings: []string{
					"[node 'node-1'] node is available, but present in spec, has no disk daemon running, skipping",
					"[node 'node-2'] node is available, but present in spec, has no disk daemon running, skipping",
				},
			},
		},
		{
			name:             "present nodes in request - remove nodes completely which are in spec and labeled and available, but not spec analysis",
			hostsFromCluster: unitinputs.CephOsdTreeOutput,
			osdsMetadata:     unitinputs.CephOsdMetadataOutputNoStray,
			osdInfo:          unitinputs.CephOsdInfoOutputNoStray,
			taskConfig: getTaskConfig(unitinputs.RequestRemoveFullNodeRemove, nil, map[string]lcmv1alpha1.DaemonStatus{
				"node-1": {Status: lcmv1alpha1.DaemonStateFailed, Issues: []string{"failed to prepare disk report"}},
			}),
			nodeList: nodesListLabeledAvailable,
			expectedRemoveInfo: &lcmv1alpha1.TaskRemoveInfo{
				CleanupMap: map[string]lcmv1alpha1.HostMapping{},
				Issues: []string{
					"[node 'node-1'] spec analyse status has failed, resolve it first",
					"[node 'node-2'] spec analyse status is not available yet",
				},
				Warnings: []string{},
			},
		},
		{
			name:             "present nodes in request - remove nodes by device which are not labeled or not available",
			hostsFromCluster: unitinputs.CephOsdTreeOutput,
			osdsMetadata:     unitinputs.CephOsdMetadataOutputNoStray,
			osdInfo:          unitinputs.CephOsdInfoOutputNoStray,
			taskConfig:       getTaskConfig(unitinputs.RequestRemoveByDevice, cephClusterReducedNodes, map[string]lcmv1alpha1.DaemonStatus{}),
			nodeList:         unitinputs.GetNodesList([]unitinputs.NodeAttrs{{Name: "node-1"}, {Name: "node-2", Unreachable: true}}),
			expectedRemoveInfo: &lcmv1alpha1.TaskRemoveInfo{
				CleanupMap: map[string]lcmv1alpha1.HostMapping{},
				Issues:     []string{},
				Warnings: []string{
					"[node 'node-1'] node is available, but has no disk daemon running, cleanup by device is not possible, use by id or complete remove, skipping",
					"[node 'node-2'] node is not available, cleanup by device is not possible, skipping",
				},
			},
		},
		{
			name:               "present nodes in request - remove nodes which are not labeled and available",
			hostsFromCluster:   unitinputs.CephOsdTreeOutput,
			osdsMetadata:       unitinputs.CephOsdMetadataOutputNoStray,
			osdInfo:            unitinputs.CephOsdInfoOutputNoStray,
			taskConfig:         tcFullremoveNonodes,
			nodeList:           nodesListNotLabeledAvailable,
			expectedRemoveInfo: unitinputs.NotLabeledNodesFullRemoveMap,
		},
		{
			name:               "present nodes in request - remove nodes which are not available",
			hostsFromCluster:   unitinputs.CephOsdTreeOutput,
			osdsMetadata:       unitinputs.CephOsdMetadataOutputNoStray,
			osdInfo:            unitinputs.CephOsdInfoOutputNoStray,
			taskConfig:         tcFullremoveNonodes,
			nodeList:           nodesListNotLabeledUnreachable,
			expectedRemoveInfo: unitinputs.NotAvailableNodesFullRemoveMap,
		},
		{
			name:             "present nodes in request - remove nodes which has not available devs",
			hostsFromCluster: unitinputs.CephOsdTreeOutput,
			osdsMetadata:     unitinputs.CephOsdMetadataOutputNoStray,
			osdInfo:          unitinputs.CephOsdInfoOutputNoStray,
			taskConfig: getTaskConfig(map[string]lcmv1alpha1.NodeCleanUpSpec{
				"node-1": {CompleteCleanup: true}}, cephClusterNoNodes, map[string]lcmv1alpha1.DaemonStatus{}),
			nodeList: nodesListNode1LabeledAvailable,
			nodeOsdReport: map[string]*lcmcommon.DiskDaemonReport{
				"node-1": &unitinputs.DiskDaemonReportOkNode1SomeDevLost,
			},
			expectedRemoveInfo: &lcmv1alpha1.TaskRemoveInfo{
				CleanupMap: map[string]lcmv1alpha1.HostMapping{
					"node-1": {
						CompleteCleanup: true,
						OsdMapping: map[string]lcmv1alpha1.OsdMapping{
							"20": unitinputs.AdaptOsdMapping("node-1", "20",
								map[string]bool{"inCrush": true}, map[string]map[string]bool{"/dev/vde": {"zap": true}, "/dev/vdd": {"lost": true}}),
							"25": unitinputs.AdaptOsdMapping("node-1", "25",
								map[string]bool{"inCrush": true}, map[string]map[string]bool{"/dev/vdf": {"zap": true}, "/dev/vdd": {"lost": true}}),
							"30": unitinputs.AdaptOsdMapping("node-1", "30",
								map[string]bool{"inCrush": true}, map[string]map[string]bool{"/dev/vda": {}, "/dev/vdb": {"zap": true}}),
						},
					},
				},
				Issues: []string{},
				Warnings: []string{
					"[node 'node-1'] found osd db partition '/dev/vda14' for osd '30', which is created not by rook, skipping disk/partition zap",
					"[node 'node-1'] found physical osd db partition '/dev/vda14' for osd '30'",
				},
			},
		},
		{
			name:             "present nodes in request - remove by osd id when not labeled or not available",
			hostsFromCluster: unitinputs.CephOsdTreeOutput,
			osdsMetadata:     unitinputs.CephOsdMetadataOutputNoStray,
			osdInfo:          unitinputs.CephOsdInfoOutputNoStray,
			taskConfig:       getTaskConfig(unitinputs.RequestRemoveByOsdID, cephClusterReducedNodes, map[string]lcmv1alpha1.DaemonStatus{}),
			nodeList:         unitinputs.GetNodesList([]unitinputs.NodeAttrs{{Name: "node-1"}, {Name: "node-2", Unreachable: true}}),
			expectedRemoveInfo: &lcmv1alpha1.TaskRemoveInfo{
				CleanupMap: map[string]lcmv1alpha1.HostMapping{
					"node-1": {
						VolumesInfoMissed: true,
						OsdMapping: map[string]lcmv1alpha1.OsdMapping{
							"20": unitinputs.NotLabeledNodesFullRemoveMap.CleanupMap["node-1"].OsdMapping["20"],
							"30": unitinputs.NotLabeledNodesFullRemoveMap.CleanupMap["node-1"].OsdMapping["30"],
						},
					},
					"node-2": {
						NodeIsDown: true,
						OsdMapping: map[string]lcmv1alpha1.OsdMapping{
							"4": unitinputs.NotAvailableNodesFullRemoveMap.CleanupMap["node-2"].OsdMapping["4"],
						},
					},
				},
				Issues: []string{},
				Warnings: []string{
					"[node 'node-1'] node is available, but has no disk daemon running, device cleanup jobs will be skipped",
					"[node 'node-2'] node has no osd id '88', skipping",
					"[node 'node-2'] node is not available, but present in spec, device cleanup jobs will be skipped",
				},
			},
		},
		{
			name:             "present nodes in request - cant get disk daemon info and incorrect stray remove",
			hostsFromCluster: unitinputs.CephOsdTreeOutput,
			osdsMetadata:     unitinputs.CephOsdMetadataOutput,
			osdInfo:          unitinputs.CephOsdInfoOutput,
			taskConfig: getTaskConfig(map[string]lcmv1alpha1.NodeCleanUpSpec{
				"__stray": {CompleteCleanup: true},
				"node-1":  {CleanupStrayPartitions: true},
			}, nil, map[string]lcmv1alpha1.DaemonStatus{"node-1": unitinputs.OsdStorageSpecAnalysisOk["node-1"]}),
			nodeList: nodesListLabeledAvailable,
			expectedRemoveInfo: &lcmv1alpha1.TaskRemoveInfo{
				CleanupMap: map[string]lcmv1alpha1.HostMapping{},
				Issues: []string{
					"[node 'node-1'] failed to get node osds report: Retries (1/1) exceeded: failed to parse output for command 'pelagia-disk-daemon --osd-report --port 9999': invalid character '|' looking for beginning of object key string",
				},
				Warnings: []string{
					"[__stray] stray which are present in crush map is possible to remove only be osd id",
				},
			},
		},
		{
			name:             "present nodes in request - remove node stray only",
			hostsFromCluster: unitinputs.CephOsdTreeOutput,
			osdsMetadata:     unitinputs.CephOsdMetadataOutput,
			osdInfo:          unitinputs.CephOsdInfoOutput,
			taskConfig: getTaskConfig(map[string]lcmv1alpha1.NodeCleanUpSpec{"node-2": {CleanupStrayPartitions: true}}, nil, map[string]lcmv1alpha1.DaemonStatus{
				"node-2": unitinputs.OsdStorageSpecAnalysisOk["node-2"]}),
			nodeList: nodesListNode2LabeledAvailable,
			nodeOsdReport: map[string]*lcmcommon.DiskDaemonReport{
				"node-2": unitinputs.DiskDaemonNodeReportWithStrayOkNode2(false),
			},
			expectedRemoveInfo: unitinputs.StrayOnlyOnNodeRemoveMap,
		},
		{
			name:             "present nodes in request - remove by osd id and stray which has partition",
			hostsFromCluster: unitinputs.CephOsdTreeOutput,
			osdsMetadata:     unitinputs.CephOsdMetadataOutput,
			osdInfo:          unitinputs.CephOsdInfoOutput,
			taskConfig: getTaskConfig(map[string]lcmv1alpha1.NodeCleanUpSpec{
				"__stray": {CleanupByOsd: []lcmv1alpha1.OsdCleanupSpec{{ID: 2}}},
				"node-2":  {CleanupStrayPartitions: true},
			}, nil, map[string]lcmv1alpha1.DaemonStatus{"node-2": unitinputs.OsdStorageSpecAnalysisOk["node-2"]}),
			nodeList: nodesListNode2LabeledAvailable,
			nodeOsdReport: map[string]*lcmcommon.DiskDaemonReport{
				"node-2": unitinputs.DiskDaemonNodeReportWithStrayOkNode2(true),
			},
			expectedRemoveInfo: unitinputs.StrayOnNodeAndInCrushRemoveMap,
		},
		{
			name:             "present nodes in request - remove nodes from cluster and stray without partitions",
			hostsFromCluster: unitinputs.CephOsdTreeOutput,
			osdsMetadata:     unitinputs.CephOsdMetadataOutput,
			osdInfo:          unitinputs.CephOsdInfoOutput,
			taskConfig: getTaskConfig(map[string]lcmv1alpha1.NodeCleanUpSpec{
				"__stray": {CleanupByOsd: []lcmv1alpha1.OsdCleanupSpec{{ID: 2}, {ID: 88}}},
				"node-1":  {CompleteCleanup: true},
				"node-2":  {DropFromCrush: true},
			}, cephClusterNoNodes, map[string]lcmv1alpha1.DaemonStatus{}),
			nodeList: nodesListLabeledAvailable,
			nodeOsdReport: map[string]*lcmcommon.DiskDaemonReport{
				"node-1": &unitinputs.DiskDaemonReportOkNode1,
				"node-2": &unitinputs.DiskDaemonReportOkNode2,
			},
			expectedRemoveInfo: &lcmv1alpha1.TaskRemoveInfo{
				CleanupMap: map[string]lcmv1alpha1.HostMapping{
					"__stray": unitinputs.StrayOnlyInCrushRemoveMap.CleanupMap["__stray"],
					"node-1":  unitinputs.FullNodesRemoveMap.CleanupMap["node-1"],
					"node-2": {
						DropFromCrush: true,
						OsdMapping: map[string]lcmv1alpha1.OsdMapping{
							"0": unitinputs.AdaptOsdMapping("node-2", "0",
								map[string]bool{"inCrush": true}, map[string]map[string]bool{"/dev/vdb": {}}),
							"4": unitinputs.AdaptOsdMapping("node-2", "4",
								map[string]bool{"inCrush": true}, map[string]map[string]bool{"/dev/vdd": {}}),
							"5": unitinputs.AdaptOsdMapping("node-2", "5",
								map[string]bool{"inCrush": true}, map[string]map[string]bool{"/dev/vdd": {}}),
						},
					},
				},
				Issues: []string{},
				Warnings: []string{
					"[__stray] stray osd with id '88' is not found in crush map",
					"[node 'node-1'] found osd db partition '/dev/ceph-metadata/part-1' for osd '20', which is created not by rook, skipping disk/partition zap",
					"[node 'node-1'] found osd db partition '/dev/ceph-metadata/part-2' for osd '25', which is created not by rook, skipping disk/partition zap",
					"[node 'node-1'] found osd db partition '/dev/vda14' for osd '30', which is created not by rook, skipping disk/partition zap",
					"[node 'node-1'] found physical osd db partition '/dev/vda14' for osd '30'",
				},
			},
		},
		{
			name:             "present nodes in request - remove by osd ids present in spec",
			hostsFromCluster: unitinputs.CephOsdTreeOutput,
			osdsMetadata:     unitinputs.CephOsdMetadataOutput,
			osdInfo:          unitinputs.CephOsdInfoOutput,
			taskConfig:       getTaskConfig(unitinputs.RequestRemoveByOsdID, nil, nil),
			nodeList:         nodesListLabeledAvailable,
			nodeOsdReport: map[string]*lcmcommon.DiskDaemonReport{
				"node-1": &unitinputs.DiskDaemonReportOkNode1,
				"node-2": &unitinputs.DiskDaemonReportOkNode2,
			},
			expectedRemoveInfo: &lcmv1alpha1.TaskRemoveInfo{
				CleanupMap: map[string]lcmv1alpha1.HostMapping{},
				Issues:     []string{},
				Warnings: []string{
					"[node 'node-1'] osd with id '20' is associated with block device '/dev/disk/by-path/pci-0000:00:0f.0', which is present in spec, can't cleanup, skipping",
					"[node 'node-1'] osd with id '30' is associated with block device '/dev/vdb', which is present in spec, can't cleanup, skipping",
					"[node 'node-2'] osd with id '4' is associated with block device '/dev/vdd', which is present in spec, can't cleanup, skipping",
					"[node 'node-2'] osd with id '88' is not found on a node, skipping",
				},
			},
		},
		{
			name:             "present nodes in request - remove by osd ids with full disks cleanup",
			hostsFromCluster: unitinputs.CephOsdTreeOutput,
			osdsMetadata:     unitinputs.CephOsdMetadataOutputNoStray,
			osdInfo:          unitinputs.CephOsdInfoOutputNoStray,
			taskConfig: getTaskConfig(map[string]lcmv1alpha1.NodeCleanUpSpec{
				"node-1": {CleanupByOsd: []lcmv1alpha1.OsdCleanupSpec{{ID: 20}, {ID: 25}}},
				"node-2": {CleanupByOsd: []lcmv1alpha1.OsdCleanupSpec{{ID: 4}, {ID: 5}}},
			}, cephClusterNoNodes, map[string]lcmv1alpha1.DaemonStatus{}),
			nodeList: nodesListLabeledAvailable,
			nodeOsdReport: map[string]*lcmcommon.DiskDaemonReport{
				"node-1": &unitinputs.DiskDaemonReportOkNode1,
				"node-2": &unitinputs.DiskDaemonReportOkNode2,
			},
			expectedRemoveInfo: &lcmv1alpha1.TaskRemoveInfo{
				CleanupMap: map[string]lcmv1alpha1.HostMapping{
					"node-1": {
						OsdMapping: map[string]lcmv1alpha1.OsdMapping{
							"20": unitinputs.FullNodesRemoveMap.CleanupMap["node-1"].OsdMapping["20"],
							"25": unitinputs.FullNodesRemoveMap.CleanupMap["node-1"].OsdMapping["25"],
						},
					},
					"node-2": unitinputs.DevNotInSpecRemoveMap.CleanupMap["node-2"],
				},
				Issues: []string{},
				Warnings: []string{
					"[node 'node-1'] found osd db partition '/dev/ceph-metadata/part-1' for osd '20', which is created not by rook, skipping disk/partition zap",
					"[node 'node-1'] found osd db partition '/dev/ceph-metadata/part-2' for osd '25', which is created not by rook, skipping disk/partition zap",
				},
			},
		},
		{
			name:             "present nodes in request - remove by osd ids with no disks cleanup",
			hostsFromCluster: unitinputs.CephOsdTreeOutput,
			osdsMetadata:     unitinputs.CephOsdMetadataOutputNoStray,
			osdInfo:          unitinputs.CephOsdInfoOutputNoStray,
			taskConfig: getTaskConfig(map[string]lcmv1alpha1.NodeCleanUpSpec{
				"node-1": {CleanupByOsd: []lcmv1alpha1.OsdCleanupSpec{{ID: 20}}},
				"node-2": {CleanupByOsd: []lcmv1alpha1.OsdCleanupSpec{{ID: 4}}},
			}, &unitinputs.ReefCephClusterHasHealthIssues, nil),
			nodeList: nodesListLabeledAvailable,
			nodeOsdReport: map[string]*lcmcommon.DiskDaemonReport{
				"node-1": &unitinputs.DiskDaemonReportOkNode1,
				"node-2": &unitinputs.DiskDaemonReportOkNode2,
			},
			expectedRemoveInfo: &lcmv1alpha1.TaskRemoveInfo{
				CleanupMap: map[string]lcmv1alpha1.HostMapping{
					"node-1": unitinputs.DevNotInSpecRemoveMap.CleanupMap["node-1"],
					"node-2": {
						OsdMapping: map[string]lcmv1alpha1.OsdMapping{
							"4": unitinputs.AdaptOsdMapping("node-2", "4", map[string]bool{"inCrush": true}, map[string]map[string]bool{"/dev/vdd": {}}),
						},
					},
				},
				Issues: []string{},
				Warnings: []string{
					"[node 'node-1'] found osd db partition '/dev/ceph-metadata/part-1' for osd '20', which is created not by rook, skipping disk/partition zap",
				},
			},
		},
		{
			name:             "present nodes in request - remove by osd ids with no disks cleanup 2",
			hostsFromCluster: unitinputs.CephOsdTreeOutput,
			osdsMetadata:     unitinputs.CephOsdMetadataOutputNoStray,
			osdInfo:          unitinputs.CephOsdInfoOutputNoStray,
			taskConfig: getTaskConfig(
				map[string]lcmv1alpha1.NodeCleanUpSpec{"node-1": {CleanupByOsd: []lcmv1alpha1.OsdCleanupSpec{{ID: 20}}}},
				func() *cephv1.CephCluster {
					cluster := unitinputs.ReefCephClusterHasHealthIssues.DeepCopy()
					cluster.Spec.Storage.Nodes[0].Devices[1].Config["metadataDevice"] = "/dev/vdd"
					return cluster
				}(), nil),
			nodeList: nodesListLabeledAvailable,
			nodeOsdReport: map[string]*lcmcommon.DiskDaemonReport{
				"node-1": &unitinputs.DiskDaemonReportOkNode1,
				"node-2": &unitinputs.DiskDaemonReportOkNode2,
			},
			expectedRemoveInfo: &lcmv1alpha1.TaskRemoveInfo{
				CleanupMap: map[string]lcmv1alpha1.HostMapping{
					"node-1": unitinputs.DevNotInSpecRemoveMap.CleanupMap["node-1"],
				},
				Issues: []string{},
				Warnings: []string{
					"[node 'node-1'] found osd db partition '/dev/ceph-metadata/part-1' for osd '20', which is created not by rook, skipping disk/partition zap",
					"[node 'node-1'] osd with id '20' is associated with metadata device '/dev/vdd', which is present in spec, disk zap will be skipped",
				},
			},
		},
		{
			name:             "present nodes in request - remove by osd id when some dev unavailable",
			hostsFromCluster: unitinputs.CephOsdTreeOutput,
			osdsMetadata:     unitinputs.CephOsdMetadataOutputNoStray,
			osdInfo:          unitinputs.CephOsdInfoOutputNoStray,
			taskConfig: getTaskConfig(map[string]lcmv1alpha1.NodeCleanUpSpec{
				"node-1": {CleanupByOsd: []lcmv1alpha1.OsdCleanupSpec{{ID: 20}}},
			}, &unitinputs.ReefCephClusterHasHealthIssues, map[string]lcmv1alpha1.DaemonStatus{"node-1": unitinputs.OsdStorageSpecAnalysisOk["node-1"]}),
			nodeList: nodesListNode1LabeledAvailable,
			nodeOsdReport: map[string]*lcmcommon.DiskDaemonReport{
				"node-1": &unitinputs.DiskDaemonReportOkNode1SomeDevLost,
			},
			expectedRemoveInfo: &lcmv1alpha1.TaskRemoveInfo{
				CleanupMap: map[string]lcmv1alpha1.HostMapping{
					"node-1": {
						OsdMapping: map[string]lcmv1alpha1.OsdMapping{
							"20": unitinputs.AdaptOsdMapping("node-1", "20",
								map[string]bool{"inCrush": true}, map[string]map[string]bool{"/dev/vde": {"zap": true}, "/dev/vdd": {"lost": true}}),
						},
					},
				},
				Issues:   []string{},
				Warnings: []string{},
			},
		},
		{
			name:             "present nodes in request - remove by devices but devices in spec",
			hostsFromCluster: unitinputs.CephOsdTreeOutput,
			osdsMetadata:     unitinputs.CephOsdMetadataOutputNoStray,
			osdInfo:          unitinputs.CephOsdInfoOutputNoStray,
			taskConfig: getTaskConfig(map[string]lcmv1alpha1.NodeCleanUpSpec{
				"node-1": {
					CleanupByDevice: []lcmv1alpha1.DeviceCleanupSpec{
						{Device: "/dev/disk/by-path/virtio-pci-0000:00:0f.0"},
						{Device: "vda"},
						{Device: "/dev/ceph-metadata/part-2"},
						{Device: "vde"},
						{Device: "/dev/disk/by-id/virtio-996ea59f-7f47-4fac-b"},
						{Device: "/dev/mapper/ceph--metadata-part--2"},
					},
				},
				"node-2": {CleanupByDevice: []lcmv1alpha1.DeviceCleanupSpec{{Device: "/dev/vdd"}}},
			}, nil, nil),
			nodeList: nodesListLabeledAvailable,
			nodeOsdReport: map[string]*lcmcommon.DiskDaemonReport{
				"node-1": &unitinputs.DiskDaemonReportOkNode1,
				"node-2": &unitinputs.DiskDaemonReportOkNode2,
			},
			expectedRemoveInfo: &lcmv1alpha1.TaskRemoveInfo{
				CleanupMap: map[string]lcmv1alpha1.HostMapping{},
				Issues:     []string{},
				Warnings: []string{
					"[node 'node-1'] device '/dev/ceph-metadata/part-2' is marked for clean up, but present in spec as '/dev/ceph-metadata/part-2'",
					"[node 'node-1'] device '/dev/disk/by-id/virtio-996ea59f-7f47-4fac-b' is marked for clean up, but present in spec as '/dev/vdb'",
					"[node 'node-1'] device '/dev/disk/by-path/virtio-pci-0000:00:0f.0' is marked for clean up, but present in spec as '/dev/disk/by-path/pci-0000:00:0f.0'",
					"[node 'node-1'] device '/dev/mapper/ceph--metadata-part--2' is marked for clean up, but present in spec as '/dev/ceph-metadata/part-2'",
					"[node 'node-1'] device 'vda' is marked for clean up, but present in spec as '/dev/vda14'",
					"[node 'node-1'] device 'vde' is marked for clean up, but present in spec as '/dev/disk/by-path/pci-0000:00:0f.0'",
					"[node 'node-2'] device '/dev/vdd' is marked for clean up, but present in spec as '/dev/vdd'",
				},
			},
		},
		{
			name:             "present nodes in request - remove by devices is ok",
			hostsFromCluster: unitinputs.CephOsdTreeOutput,
			osdsMetadata:     unitinputs.CephOsdMetadataOutputNoStray,
			osdInfo:          unitinputs.CephOsdInfoOutputNoStray,
			taskConfig:       getTaskConfig(unitinputs.RequestRemoveByDevice, &unitinputs.ReefCephClusterHasHealthIssues, nil),
			nodeList:         nodesListLabeledAvailable,
			nodeOsdReport: map[string]*lcmcommon.DiskDaemonReport{
				"node-1": &unitinputs.DiskDaemonReportOkNode1,
				"node-2": &unitinputs.DiskDaemonReportOkNode2,
			},
			expectedRemoveInfo: unitinputs.DevNotInSpecRemoveMap,
		},
		{
			name:             "present nodes in request - remove by devices is ok, allow to remove manually created lvms",
			hostsFromCluster: unitinputs.CephOsdTreeOutput,
			osdsMetadata:     unitinputs.CephOsdMetadataOutputNoStray,
			osdInfo:          unitinputs.CephOsdInfoOutputNoStray,
			taskConfig:       getTaskConfig(unitinputs.RequestRemoveByDevice, &unitinputs.ReefCephClusterHasHealthIssues, nil),
			nodeList:         nodesListLabeledAvailable,
			nodeOsdReport: map[string]*lcmcommon.DiskDaemonReport{
				"node-1": &unitinputs.DiskDaemonReportOkNode1,
				"node-2": &unitinputs.DiskDaemonReportOkNode2,
			},
			removeAllLVMs: true,
			expectedRemoveInfo: &lcmv1alpha1.TaskRemoveInfo{
				CleanupMap: unitinputs.DevNotInSpecRemoveMap.CleanupMap,
				Issues:     []string{},
				Warnings:   []string{},
			},
		},
		{
			name:             "present nodes in request - remove by device when some dev unavailable",
			hostsFromCluster: unitinputs.CephOsdTreeOutput,
			osdsMetadata:     unitinputs.CephOsdMetadataOutputNoStray,
			osdInfo:          unitinputs.CephOsdInfoOutputNoStray,
			taskConfig: getTaskConfig(map[string]lcmv1alpha1.NodeCleanUpSpec{
				"node-1": {CleanupByDevice: []lcmv1alpha1.DeviceCleanupSpec{
					{Device: "/dev/disk/by-path/virtio-pci-0000:00:0f.0"}, {Device: "vdd"}}},
			}, &unitinputs.ReefCephClusterHasHealthIssues, map[string]lcmv1alpha1.DaemonStatus{"node-1": unitinputs.OsdStorageSpecAnalysisOk["node-1"]}),
			nodeList: nodesListNode1LabeledAvailable,
			nodeOsdReport: map[string]*lcmcommon.DiskDaemonReport{
				"node-1": &unitinputs.DiskDaemonReportOkNode1SomeDevLost,
			},
			expectedRemoveInfo: &lcmv1alpha1.TaskRemoveInfo{
				CleanupMap: map[string]lcmv1alpha1.HostMapping{
					"node-1": {
						OsdMapping: map[string]lcmv1alpha1.OsdMapping{
							"20": unitinputs.AdaptOsdMapping("node-1", "20",
								map[string]bool{"inCrush": true}, map[string]map[string]bool{"/dev/vde": {"zap": true}, "/dev/vdd": {"lost": true}}),
						},
					},
				},
				Issues: []string{},
				Warnings: []string{
					"[node 'node-1'] device 'vdd' is not found on a node or has no osd partitions to cleanup, skipping",
				},
			},
		},
		{
			name:             "present nodes in request - remove by devices not found and separate partition used in spec",
			hostsFromCluster: unitinputs.CephOsdTreeOutput,
			osdsMetadata:     unitinputs.CephOsdMetadataOutputNoStray,
			osdInfo:          unitinputs.CephOsdInfoOutputNoStray,
			taskConfig: getTaskConfig(map[string]lcmv1alpha1.NodeCleanUpSpec{
				"node-1": {CleanupByDevice: []lcmv1alpha1.DeviceCleanupSpec{{Device: "/dev/disk/by-id/virtio-e8d89e2f-ffc6-4988-9"}}},
				"node-2": {CleanupByDevice: []lcmv1alpha1.DeviceCleanupSpec{{Device: "vdr"}}},
			}, &unitinputs.ReefCephClusterHasHealthIssues, nil),
			nodeList: nodesListLabeledAvailable,
			nodeOsdReport: map[string]*lcmcommon.DiskDaemonReport{
				"node-1": &unitinputs.DiskDaemonReportOkNode1,
				"node-2": &unitinputs.DiskDaemonReportOkNode2,
			},
			expectedRemoveInfo: &lcmv1alpha1.TaskRemoveInfo{
				CleanupMap: map[string]lcmv1alpha1.HostMapping{},
				Issues:     []string{},
				Warnings: []string{
					"[node 'node-1'] device '/dev/disk/by-id/virtio-e8d89e2f-ffc6-4988-9' is marked for clean up, but present in spec as '/dev/ceph-metadata/part-2'",
					"[node 'node-2'] device 'vdr' is not found on a node or has no osd partitions to cleanup, skipping",
				},
			},
		},
		{
			name:             "present nodes in request - remove by devices stray disks",
			hostsFromCluster: unitinputs.CephOsdTreeOutput,
			osdsMetadata:     unitinputs.CephOsdMetadataOutputNoStray,
			osdInfo:          unitinputs.CephOsdInfoOutputNoStray,
			taskConfig: getTaskConfig(map[string]lcmv1alpha1.NodeCleanUpSpec{
				"node-2": {CleanupByDevice: []lcmv1alpha1.DeviceCleanupSpec{{Device: "/dev/disk/by-id/virtio-ffe08946-7614-4f69-b"}}},
			}, nil, map[string]lcmv1alpha1.DaemonStatus{"node-2": unitinputs.OsdStorageSpecAnalysisOk["node-2"]}),
			nodeList: nodesListNode2LabeledAvailable,
			nodeOsdReport: map[string]*lcmcommon.DiskDaemonReport{
				"node-2": {
					State: lcmcommon.DiskDaemonStateOk,
					OsdsReport: &lcmcommon.DiskDaemonOsdsReport{
						Warnings: []string{},
						Osds: map[string][]lcmcommon.OsdDaemonInfo{
							"0": lcmdiskdaemoninput.OsdDevicesInfoNode2["0-stray-nvme"],
						},
					},
				},
			},
			expectedRemoveInfo: func() *lcmv1alpha1.TaskRemoveInfo {
				newInfo := unitinputs.StrayOnlyOnNodeRemoveMap.DeepCopy()
				info := newInfo.CleanupMap["node-2"].OsdMapping["0.06bf4d7c-9603-41a4-b250-284ecf3ecb2f.__stray"].DeviceMapping["/dev/vdc"]
				info.Path = "/dev/disk/by-id/virtio-ffe08946-7614-4f69-b"
				info.Rotational = false
				newInfo.CleanupMap["node-2"].OsdMapping["0.06bf4d7c-9603-41a4-b250-284ecf3ecb2f.__stray"].DeviceMapping["/dev/vdc"] = info
				return newInfo
			}(),
		},
		{
			name:             "present nodes in request - device filters, present in spec",
			hostsFromCluster: unitinputs.CephOsdTreeOutput,
			osdsMetadata:     unitinputs.CephOsdMetadataOutputNoStray,
			osdInfo:          unitinputs.CephOsdInfoOutputNoStray,
			taskConfig: getTaskConfig(map[string]lcmv1alpha1.NodeCleanUpSpec{
				"node-1": {CleanupByDevice: []lcmv1alpha1.DeviceCleanupSpec{{Device: "/dev/disk/by-path/pci-0000:00:0f.0"}}},
				"node-2": {CleanupByOsd: []lcmv1alpha1.OsdCleanupSpec{{ID: 0}}},
			}, cephClusterFilteredDevices, nil),
			nodeList: nodesListLabeledAvailable,
			nodeOsdReport: map[string]*lcmcommon.DiskDaemonReport{
				"node-1": &unitinputs.DiskDaemonReportOkNode1,
				"node-2": &unitinputs.DiskDaemonReportOkNode2,
			},
			expectedRemoveInfo: &lcmv1alpha1.TaskRemoveInfo{
				CleanupMap: map[string]lcmv1alpha1.HostMapping{},
				Issues:     []string{},
				Warnings: []string{
					"[node 'node-1'] device '/dev/disk/by-path/pci-0000:00:0f.0' is marked for clean up, but present in spec as '/dev/vd[fe]'",
					"[node 'node-2'] osd with id '0' is associated with block device 'vdb', which is present in spec, can't cleanup, skipping",
				},
			},
		},
		{
			name:             "present nodes in request - device filters, some osds ready to remove",
			hostsFromCluster: unitinputs.CephOsdTreeOutput,
			osdsMetadata:     unitinputs.CephOsdMetadataOutputNoStray,
			osdInfo:          unitinputs.CephOsdInfoOutputNoStray,
			taskConfig: getTaskConfig(map[string]lcmv1alpha1.NodeCleanUpSpec{
				"node-1": {CleanupByDevice: []lcmv1alpha1.DeviceCleanupSpec{{Device: "vdb"}}},
				"node-2": {CleanupByOsd: []lcmv1alpha1.OsdCleanupSpec{{ID: 4}, {ID: 5}}},
			}, cephClusterFilteredDevices, nil),
			nodeList: nodesListLabeledAvailable,
			nodeOsdReport: map[string]*lcmcommon.DiskDaemonReport{
				"node-1": &unitinputs.DiskDaemonReportOkNode1,
				"node-2": &unitinputs.DiskDaemonReportOkNode2,
			},
			expectedRemoveInfo: &lcmv1alpha1.TaskRemoveInfo{
				CleanupMap: map[string]lcmv1alpha1.HostMapping{
					"node-1": {
						OsdMapping: map[string]lcmv1alpha1.OsdMapping{
							"30": unitinputs.AdaptOsdMapping("node-1", "30",
								map[string]bool{"inCrush": true}, map[string]map[string]bool{"/dev/vda": {}, "/dev/vdb": {"zap": true}}),
						},
					},
					"node-2": unitinputs.DevNotInSpecRemoveMap.CleanupMap["node-2"],
				},
				Issues: []string{},
				Warnings: []string{
					"[node 'node-1'] found osd db partition '/dev/vda14' for osd '30', which is created not by rook, skipping disk/partition zap",
					"[node 'node-1'] found physical osd db partition '/dev/vda14' for osd '30'",
				},
			},
		},
		{
			name:             "present nodes in request - osd and device marked to skip cleanup",
			hostsFromCluster: unitinputs.CephOsdTreeOutput,
			osdsMetadata:     unitinputs.CephOsdMetadataOutputNoStray,
			osdInfo:          unitinputs.CephOsdInfoOutputNoStray,
			taskConfig: getTaskConfig(map[string]lcmv1alpha1.NodeCleanUpSpec{
				"node-1": {CleanupByDevice: []lcmv1alpha1.DeviceCleanupSpec{{Device: "vdb", SkipDeviceCleanup: true}}},
				"node-2": {CleanupByOsd: []lcmv1alpha1.OsdCleanupSpec{{ID: 4}, {ID: 5, SkipDeviceCleanup: true}}},
			}, cephClusterFilteredDevices, nil),
			nodeList: nodesListLabeledAvailable,
			nodeOsdReport: map[string]*lcmcommon.DiskDaemonReport{
				"node-1": &unitinputs.DiskDaemonReportOkNode1,
				"node-2": &unitinputs.DiskDaemonReportOkNode2,
			},
			expectedRemoveInfo: unitinputs.SkipCleanupJobRemoveMap,
		},
		{
			name:             "present nodes in request - spec analysis skipped, lcm skipped",
			hostsFromCluster: unitinputs.CephOsdTreeOutput,
			osdsMetadata:     unitinputs.CephOsdMetadataOutputNoStray,
			osdInfo:          unitinputs.CephOsdInfoOutputNoStray,
			taskConfig: getTaskConfig(
				map[string]lcmv1alpha1.NodeCleanUpSpec{"node-1": {CompleteCleanup: true}},
				&unitinputs.ReefCephClusterHasHealthIssues,
				map[string]lcmv1alpha1.DaemonStatus{"node-1": {Status: lcmv1alpha1.DaemonStateSkipped}},
			),
			nodeList: nodesListLabeledAvailable,
			expectedRemoveInfo: &lcmv1alpha1.TaskRemoveInfo{
				CleanupMap: map[string]lcmv1alpha1.HostMapping{},
				Issues:     []string{},
				Warnings:   []string{"[node 'node-1'] spec analyse status skipped, skipping lcm actions"},
			},
		},
	}
	oldCmdFunc := lcmcommon.RunPodCommand
	oldRetries := retriesForFailedCommand
	retriesForFailedCommand = 1
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			lcmConfigData := map[string]string{}
			if test.removeAllLVMs {
				lcmConfigData["TASK_ALLOW_REMOVE_MANUALLY_CREATED_LVMS"] = "true"
			}
			c := fakeCephReconcileConfig(&test.taskConfig, lcmConfigData)
			faketestclients.FakeReaction(c.api.Kubeclientset.CoreV1(), "list", []string{"pods"}, map[string]runtime.Object{"pods": unitinputs.ToolBoxAndDiskDaemonPodsList}, nil)

			lcmcommon.RunPodCommand = func(e lcmcommon.ExecConfig) (string, string, error) {
				switch e.Command {
				case "ceph osd tree -f json":
					return test.hostsFromCluster, "", nil
				case "ceph osd metadata -f json":
					return test.osdsMetadata, "", nil
				case "ceph osd info -f json":
					return test.osdInfo, "", nil
				}
				return "", "", errors.New("unexpected command")
			}

			clusterHostList, err := c.getOsdHostsFromCluster()
			assert.Nil(t, err)
			var clusterOsdsMetadata []lcmcommon.OsdMetadataInfo
			err = lcmcommon.RunAndParseCephToolboxCLI(c.context, c.api.Kubeclientset, c.api.Config, c.taskConfig.task.Namespace, "ceph osd metadata -f json", &clusterOsdsMetadata)
			assert.Nil(t, err)
			var clusterOsdsInfo []lcmcommon.OsdInfo
			err = lcmcommon.RunAndParseCephToolboxCLI(c.context, c.api.Kubeclientset, c.api.Config, c.taskConfig.task.Namespace, "ceph osd info -f json", &clusterOsdsInfo)
			assert.Nil(t, err)

			lcmcommon.RunPodCommand = func(e lcmcommon.ExecConfig) (string, string, error) {
				if e.Command == "pelagia-disk-daemon --osd-report --port 9999" {
					if report, present := test.nodeOsdReport[e.Nodename]; present {
						output, _ := json.Marshal(report)
						return string(output), "", nil
					}
					return "{||}", "", nil
				}
				return "", "", errors.New("unexpected command")
			}

			removeInfo := c.getOsdsForCleanup(clusterHostList, clusterOsdsMetadata, clusterOsdsInfo, test.nodeList.Items)
			assert.Equal(t, test.expectedRemoveInfo, removeInfo)
		})
	}
	lcmcommon.RunPodCommand = oldCmdFunc
	retriesForFailedCommand = oldRetries
}
