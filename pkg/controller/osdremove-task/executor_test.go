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
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	batch "k8s.io/api/batch/v1"
	"k8s.io/apimachinery/pkg/runtime"

	lcmv1alpha1 "github.com/Mirantis/pelagia/pkg/apis/ceph.pelagia.lcm/v1alpha1"
	lcmcommon "github.com/Mirantis/pelagia/pkg/common"
	faketestclients "github.com/Mirantis/pelagia/test/unit/clients"
	unitinputs "github.com/Mirantis/pelagia/test/unit/inputs"
)

func TestProcessOsdRemoveTask(t *testing.T) {
	infoWithRebalancingOsd := unitinputs.GetInfoWithStatus(unitinputs.FullNodesRemoveMap,
		map[string]*lcmv1alpha1.RemoveResult{
			"*":  {OsdRemoveStatus: &lcmv1alpha1.RemoveStatus{Status: lcmv1alpha1.RemovePending}},
			"25": {OsdRemoveStatus: &lcmv1alpha1.RemoveStatus{Status: lcmv1alpha1.RemoveWaitingRebalance, StartedAt: time.Now().Format(time.RFC3339)}},
		},
	)
	tests := []struct {
		name              string
		taskConfig        taskConfig
		expectedRemoveMap *lcmv1alpha1.TaskRemoveInfo
		cephCliOutput     map[string]string
		finished          bool
		requeueRequired   bool
		batchJob          *batch.Job
	}{
		{
			name: "processing - osds are not ready to move to rebalancing",
			taskConfig: taskConfig{
				task:        unitinputs.GetTaskForRemove(unitinputs.CephOsdRemoveTaskOnValidation, unitinputs.FullNodesRemoveMap.DeepCopy()),
				cephCluster: &unitinputs.ReefCephClusterReady,
			},
			cephCliOutput: map[string]string{
				"ceph osd info 0 --format json":  `{"osd":0, "up":1, "in":1}`,
				"ceph osd info 4 --format json":  `{"osd":4, "up":1, "in":1}`,
				"ceph osd info 5 --format json":  `{"osd":5, "up":1, "in":1}`,
				"ceph osd info 20 --format json": `{"osd":20, "up":1, "in":1}`,
				"ceph osd info 25 --format json": `{"osd":25, "up":1, "in":1}`,
				"ceph osd info 30 --format json": `{"osd":30, "up":1, "in":1}`,
			},
			expectedRemoveMap: unitinputs.GetInfoWithStatus(unitinputs.FullNodesRemoveMap,
				map[string]*lcmv1alpha1.RemoveResult{
					"*": {
						OsdRemoveStatus: &lcmv1alpha1.RemoveStatus{Status: lcmv1alpha1.RemovePending, StartedAt: "2025-04-14T14:30:40Z"},
					},
				},
			),
		},
		{
			name: "processing - osd 25 moved to rebalancing",
			taskConfig: taskConfig{
				task:        unitinputs.GetTaskForRemove(unitinputs.CephOsdRemoveTaskOnValidation, unitinputs.FullNodesRemoveMap.DeepCopy()),
				cephCluster: &unitinputs.ReefCephClusterReady,
			},
			cephCliOutput: map[string]string{
				"ceph osd info 0 --format json":      `{"osd":0, "up":1, "in":1}`,
				"ceph osd info 4 --format json":      `{"osd":4, "up":1, "in":1}`,
				"ceph osd info 5 --format json":      `{"osd":5, "up":1, "in":1}`,
				"ceph osd info 20 --format json":     `{"osd":20, "up":1, "in":1}`,
				"ceph osd info 25 --format json":     `{"osd":25, "up":1, "in":1}`,
				"ceph osd info 30 --format json":     `{"osd":30, "up":1, "in":1}`,
				"ceph osd ok-to-stop 25":             "",
				"ceph osd crush reweight osd.25 0.0": "",
			},
			expectedRemoveMap: unitinputs.GetInfoWithStatus(unitinputs.FullNodesRemoveMap,
				map[string]*lcmv1alpha1.RemoveResult{
					"*":  {OsdRemoveStatus: &lcmv1alpha1.RemoveStatus{Status: lcmv1alpha1.RemovePending}},
					"25": {OsdRemoveStatus: &lcmv1alpha1.RemoveStatus{Status: lcmv1alpha1.RemoveWaitingRebalance, StartedAt: "2025-04-14T14:30:41Z"}},
				},
			),
			requeueRequired: true,
		},
		{
			name: "processing - osd 25 is rebalancing",
			taskConfig: taskConfig{
				task:        unitinputs.GetTaskForRemove(unitinputs.CephOsdRemoveTaskOnValidation, infoWithRebalancingOsd),
				cephCluster: &unitinputs.ReefCephClusterReady,
			},
			cephCliOutput: map[string]string{
				"ceph pg ls-by-osd 25 --format json": `{"pg_stats": [ {"key": "value"} ]}`,
			},
			expectedRemoveMap: infoWithRebalancingOsd,
		},
		{
			name: "processing - osd 25 rebalance is finished",
			taskConfig: taskConfig{
				task:        unitinputs.GetTaskForRemove(unitinputs.CephOsdRemoveTaskOnValidation, infoWithRebalancingOsd),
				cephCluster: &unitinputs.ReefCephClusterReady,
			},
			cephCliOutput: map[string]string{
				"ceph pg ls-by-osd 25 --format json": `{"pg_stats": []}`,
			},
			expectedRemoveMap: func() *lcmv1alpha1.TaskRemoveInfo {
				info := infoWithRebalancingOsd.DeepCopy()
				info.CleanupMap["node-1"].OsdMapping["25"].RemoveStatus.OsdRemoveStatus.Status = lcmv1alpha1.RemoveInProgress
				return info
			}(),
			requeueRequired: true,
		},
		{
			name: "processing - osd 25 is removed and took osd 20 to remove",
			taskConfig: taskConfig{
				task: unitinputs.GetTaskForRemove(unitinputs.CephOsdRemoveTaskOnValidation, func() *lcmv1alpha1.TaskRemoveInfo {
					info := infoWithRebalancingOsd.DeepCopy()
					info.CleanupMap["node-1"].OsdMapping["25"].RemoveStatus.OsdRemoveStatus.Status = lcmv1alpha1.RemoveInProgress
					return info
				}()),
				cephCluster: &unitinputs.ReefCephClusterReady,
			},
			cephCliOutput: map[string]string{
				"ceph osd info 0 --format json":                    `{"osd":0, "up":1, "in":1}`,
				"ceph osd info 4 --format json":                    `{"osd":4, "up":1, "in":1}`,
				"ceph osd info 5 --format json":                    `{"osd":5, "up":1, "in":1}`,
				"ceph osd info 20 --format json":                   `{"osd":20, "up":1, "in":1}`,
				"ceph osd info 30 --format json":                   `{"osd":30, "up":1, "in":1}`,
				"ceph osd ok-to-stop 20":                           "",
				"ceph osd crush reweight osd.20 0.0":               "",
				"ceph osd purge 25 --force --yes-i-really-mean-it": "",
				"ceph auth del osd.25":                             "",
			},
			expectedRemoveMap: unitinputs.GetInfoWithStatus(unitinputs.FullNodesRemoveMap,
				map[string]*lcmv1alpha1.RemoveResult{
					"*":  {OsdRemoveStatus: &lcmv1alpha1.RemoveStatus{Status: lcmv1alpha1.RemovePending}},
					"20": {OsdRemoveStatus: &lcmv1alpha1.RemoveStatus{Status: lcmv1alpha1.RemoveWaitingRebalance, StartedAt: "2025-04-14T14:30:44Z"}},
					"25": {OsdRemoveStatus: &lcmv1alpha1.RemoveStatus{
						Status:     lcmv1alpha1.RemoveFinished,
						StartedAt:  infoWithRebalancingOsd.CleanupMap["node-1"].OsdMapping["25"].RemoveStatus.OsdRemoveStatus.StartedAt,
						FinishedAt: "2025-04-14T14:30:44Z",
					}},
				},
			),
			requeueRequired: true,
		},
		{
			name: "processing - skip running cleanup job for down host and for no volume info",
			taskConfig: taskConfig{
				task: unitinputs.GetTaskForRemove(unitinputs.CephOsdRemoveTaskOnValidation, func() *lcmv1alpha1.TaskRemoveInfo {
					info := infoWithRebalancingOsd.DeepCopy()
					info.CleanupMap["node-1"].OsdMapping["25"].RemoveStatus.OsdRemoveStatus.Status = lcmv1alpha1.RemoveFinished
					info.CleanupMap["node-2"].OsdMapping["0"].RemoveStatus.OsdRemoveStatus.Status = lcmv1alpha1.RemoveFinished
					host1 := info.CleanupMap["node-1"]
					host1.VolumesInfoMissed = true
					info.CleanupMap["node-1"] = host1
					host2 := info.CleanupMap["node-2"]
					host2.NodeIsDown = true
					info.CleanupMap["node-2"] = host2
					return info
				}()),
				cephCluster: &unitinputs.ReefCephClusterReady,
			},
			cephCliOutput: map[string]string{
				"ceph osd info 4 --format json":  `{"osd":4, "up":1, "in":1}`,
				"ceph osd info 5 --format json":  `{"osd":5, "up":1, "in":1}`,
				"ceph osd info 20 --format json": `{"osd":20, "up":1, "in":1}`,
				"ceph osd info 30 --format json": `{"osd":30, "up":1, "in":1}`,
			},
			expectedRemoveMap: func() *lcmv1alpha1.TaskRemoveInfo {
				info := infoWithRebalancingOsd.DeepCopy()
				info.CleanupMap["node-1"].OsdMapping["25"].RemoveStatus.OsdRemoveStatus.Status = lcmv1alpha1.RemoveFinished
				info.CleanupMap["node-2"].OsdMapping["0"].RemoveStatus.OsdRemoveStatus.Status = lcmv1alpha1.RemoveFinished
				info.CleanupMap["node-1"].OsdMapping["25"].RemoveStatus.DeviceCleanUpJob = &lcmv1alpha1.RemoveStatus{Status: lcmv1alpha1.RemoveSkipped}
				info.CleanupMap["node-1"].OsdMapping["25"].RemoveStatus.DeployRemoveStatus = &lcmv1alpha1.RemoveStatus{
					Status:     lcmv1alpha1.RemoveFinished,
					Name:       "rook-ceph-osd-25",
					StartedAt:  "2025-04-14T14:30:45Z",
					FinishedAt: "2025-04-14T14:30:45Z",
				}
				info.CleanupMap["node-2"].OsdMapping["0"].RemoveStatus.DeviceCleanUpJob = &lcmv1alpha1.RemoveStatus{Status: lcmv1alpha1.RemoveSkipped}
				info.CleanupMap["node-2"].OsdMapping["0"].RemoveStatus.DeployRemoveStatus = &lcmv1alpha1.RemoveStatus{
					Status:     lcmv1alpha1.RemoveFinished,
					Name:       "rook-ceph-osd-0",
					StartedAt:  "2025-04-14T14:30:45Z",
					FinishedAt: "2025-04-14T14:30:45Z",
				}
				host1 := info.CleanupMap["node-1"]
				host1.VolumesInfoMissed = true
				info.CleanupMap["node-1"] = host1
				host2 := info.CleanupMap["node-2"]
				host2.NodeIsDown = true
				info.CleanupMap["node-2"] = host2
				info.CleanupMap["node-1"].OsdMapping["20"].RemoveStatus.OsdRemoveStatus.StartedAt = "2025-04-14T14:30:45Z"
				info.CleanupMap["node-1"].OsdMapping["30"].RemoveStatus.OsdRemoveStatus.StartedAt = "2025-04-14T14:30:45Z"
				info.CleanupMap["node-2"].OsdMapping["4"].RemoveStatus.OsdRemoveStatus.StartedAt = "2025-04-14T14:30:45Z"
				info.CleanupMap["node-2"].OsdMapping["5"].RemoveStatus.OsdRemoveStatus.StartedAt = "2025-04-14T14:30:45Z"
				return info
			}(),
		},
		{
			name: "processing - osd 25 is failed to remove, cancel other pending and finish",
			taskConfig: taskConfig{
				task: unitinputs.GetTaskForRemove(unitinputs.CephOsdRemoveTaskOnValidation, func() *lcmv1alpha1.TaskRemoveInfo {
					info := infoWithRebalancingOsd.DeepCopy()
					info.CleanupMap["node-1"].OsdMapping["25"].RemoveStatus.OsdRemoveStatus.Status = lcmv1alpha1.RemoveFailed
					info.CleanupMap["node-1"].OsdMapping["25"].RemoveStatus.OsdRemoveStatus.Error = "timeouted"
					return info
				}()),
				cephCluster: &unitinputs.ReefCephClusterReady,
			},
			cephCliOutput: map[string]string{},
			expectedRemoveMap: func() *lcmv1alpha1.TaskRemoveInfo {
				info := unitinputs.GetInfoWithStatus(unitinputs.FullNodesRemoveMap,
					map[string]*lcmv1alpha1.RemoveResult{
						"*": {OsdRemoveStatus: &lcmv1alpha1.RemoveStatus{Status: lcmv1alpha1.RemovePending}},
						"25": {OsdRemoveStatus: &lcmv1alpha1.RemoveStatus{
							Status:    lcmv1alpha1.RemoveFailed,
							StartedAt: infoWithRebalancingOsd.CleanupMap["node-1"].OsdMapping["25"].RemoveStatus.OsdRemoveStatus.StartedAt,
							Error:     "timeouted",
						}},
					},
				)
				info.Issues = []string{"[node 'node-1'] failed to remove osd '25'"}
				return info
			}(),
			requeueRequired: true,
			finished:        true,
		},
		{
			name: "processing - osd 25 is failed to remove, cancel other pending, but let finish jobs and other removes",
			taskConfig: taskConfig{
				task: unitinputs.GetTaskForRemove(unitinputs.CephOsdRemoveTaskOnValidation, func() *lcmv1alpha1.TaskRemoveInfo {
					info := infoWithRebalancingOsd.DeepCopy()
					info.CleanupMap["node-1"].OsdMapping["25"].RemoveStatus.OsdRemoveStatus.Status = lcmv1alpha1.RemoveFailed
					info.CleanupMap["node-1"].OsdMapping["25"].RemoveStatus.OsdRemoveStatus.Error = "timeouted"
					info.CleanupMap["node-1"].OsdMapping["20"].RemoveStatus.OsdRemoveStatus.Status = lcmv1alpha1.RemoveInProgress
					info.CleanupMap["node-2"].OsdMapping["0"].RemoveStatus.OsdRemoveStatus.Status = lcmv1alpha1.RemoveFinished
					return info
				}()),
				cephCluster: &unitinputs.ReefCephClusterReady,
			},
			cephCliOutput: map[string]string{
				"ceph osd purge 20 --force --yes-i-really-mean-it": "",
				"ceph auth del osd.20":                             "",
			},
			expectedRemoveMap: func() *lcmv1alpha1.TaskRemoveInfo {
				info := unitinputs.GetInfoWithStatus(unitinputs.FullNodesRemoveMap,
					map[string]*lcmv1alpha1.RemoveResult{
						"*": {OsdRemoveStatus: &lcmv1alpha1.RemoveStatus{Status: lcmv1alpha1.RemovePending}},
						"0": {
							OsdRemoveStatus: &lcmv1alpha1.RemoveStatus{Status: lcmv1alpha1.RemoveFinished},
							DeviceCleanUpJob: &lcmv1alpha1.RemoveStatus{
								Status:    lcmv1alpha1.RemoveInProgress,
								StartedAt: "2025-04-14T14:30:47Z",
								Name:      "device-cleanup-job-node-2-0",
							},
						},
						"20": {
							OsdRemoveStatus: &lcmv1alpha1.RemoveStatus{
								Status:     lcmv1alpha1.RemoveFinished,
								FinishedAt: "2025-04-14T14:30:47Z",
							},
						},
						"25": {
							OsdRemoveStatus: &lcmv1alpha1.RemoveStatus{
								Status:    lcmv1alpha1.RemoveFailed,
								StartedAt: infoWithRebalancingOsd.CleanupMap["node-1"].OsdMapping["25"].RemoveStatus.OsdRemoveStatus.StartedAt,
								Error:     "timeouted",
							},
						},
					},
				)
				info.Issues = []string{"[node 'node-1'] failed to remove osd '25'"}
				return info
			}(),
		},
		{
			name: "processing - job is failing for osd 25 and no deploy remove",
			taskConfig: taskConfig{
				task: unitinputs.GetTaskForRemove(unitinputs.CephOsdRemoveTaskOnValidation, func() *lcmv1alpha1.TaskRemoveInfo {
					info := infoWithRebalancingOsd.DeepCopy()
					info.CleanupMap["node-1"].OsdMapping["25"].RemoveStatus.OsdRemoveStatus.Status = lcmv1alpha1.RemoveFinished
					info.CleanupMap["node-1"].OsdMapping["25"].RemoveStatus.DeviceCleanUpJob = &lcmv1alpha1.RemoveStatus{
						Status:    lcmv1alpha1.RemoveInProgress,
						StartedAt: "2025-04-14T14:30:47Z",
						Name:      "device-cleanup-job-node-1-25",
					}
					return info
				}()),
				cephCluster: &unitinputs.ReefCephClusterReady,
			},
			cephCliOutput: map[string]string{
				"ceph osd info 0 --format json":  `{"osd":0, "up":1, "in":1}`,
				"ceph osd info 4 --format json":  `{"osd":4, "up":1, "in":1}`,
				"ceph osd info 5 --format json":  `{"osd":5, "up":1, "in":1}`,
				"ceph osd info 20 --format json": `{"osd":20, "up":1, "in":1}`,
				"ceph osd info 30 --format json": `{"osd":30, "up":1, "in":1}`,
			},
			batchJob: unitinputs.GetCleanupJobOnlyStatus("device-cleanup-job-node-1-25", "lcm-namespace", 0, 1, 0),
			expectedRemoveMap: unitinputs.GetInfoWithStatus(unitinputs.FullNodesRemoveMap,
				map[string]*lcmv1alpha1.RemoveResult{
					"*": {
						OsdRemoveStatus: &lcmv1alpha1.RemoveStatus{
							Status:    lcmv1alpha1.RemovePending,
							StartedAt: "2025-04-14T14:30:48Z",
						},
					},
					"25": {
						OsdRemoveStatus: &lcmv1alpha1.RemoveStatus{
							Status:    lcmv1alpha1.RemoveFinished,
							StartedAt: infoWithRebalancingOsd.CleanupMap["node-1"].OsdMapping["25"].RemoveStatus.OsdRemoveStatus.StartedAt,
						},
						DeviceCleanUpJob: &lcmv1alpha1.RemoveStatus{
							Status:    lcmv1alpha1.RemoveFailed,
							StartedAt: "2025-04-14T14:30:47Z",
							Name:      "device-cleanup-job-node-1-25",
							Error:     "job failed, check logs",
						},
					},
				},
			),
		},
		{
			name: "processing - detected failed host remove, job is failed for osd 25, but proceed new osd",
			taskConfig: taskConfig{
				task: unitinputs.GetTaskForRemove(unitinputs.CephOsdRemoveTaskOnValidation, func() *lcmv1alpha1.TaskRemoveInfo {
					info := infoWithRebalancingOsd.DeepCopy()
					host2 := info.CleanupMap["node-2"]
					host2.HostRemoveStatus = &lcmv1alpha1.RemoveStatus{
						Status:    lcmv1alpha1.RemoveFailed,
						StartedAt: "2025-04-14T14:30:47Z",
						Error:     "host crush remove failed",
					}
					info.CleanupMap["node-2"] = host2
					info.CleanupMap["node-1"].OsdMapping["25"].RemoveStatus.OsdRemoveStatus.Status = lcmv1alpha1.RemoveFinished
					info.CleanupMap["node-1"].OsdMapping["25"].RemoveStatus.DeviceCleanUpJob = &lcmv1alpha1.RemoveStatus{
						Status:    lcmv1alpha1.RemoveFailed,
						StartedAt: "2025-04-14T14:30:47Z",
						Name:      "device-cleanup-job-node-1-25",
						Error:     "job failed, check logs",
					}
					return info
				}()),
				cephCluster: &unitinputs.ReefCephClusterReady,
			},
			cephCliOutput: map[string]string{
				"ceph osd info 0 --format json":     `{"osd":0, "up":1, "in":1}`,
				"ceph osd info 4 --format json":     `{"osd":4, "up":1, "in":1}`,
				"ceph osd info 5 --format json":     `{"osd":5, "up":1, "in":1}`,
				"ceph osd info 20 --format json":    `{"osd":20, "up":1, "in":1}`,
				"ceph osd info 30 --format json":    `{"osd":30, "up":1, "in":1}`,
				"ceph osd ok-to-stop 0":             "",
				"ceph osd crush reweight osd.0 0.0": "",
			},
			expectedRemoveMap: func() *lcmv1alpha1.TaskRemoveInfo {
				info := infoWithRebalancingOsd.DeepCopy()
				host2 := info.CleanupMap["node-2"]
				host2.HostRemoveStatus = &lcmv1alpha1.RemoveStatus{
					Status:    lcmv1alpha1.RemoveFailed,
					StartedAt: "2025-04-14T14:30:47Z",
					Error:     "host crush remove failed",
				}
				info.CleanupMap["node-2"] = host2
				info.CleanupMap["node-2"].OsdMapping["0"].RemoveStatus.OsdRemoveStatus = &lcmv1alpha1.RemoveStatus{
					Status:    lcmv1alpha1.RemoveWaitingRebalance,
					StartedAt: "2025-04-14T14:30:49Z",
				}
				info.CleanupMap["node-1"].OsdMapping["25"].RemoveStatus.OsdRemoveStatus.Status = lcmv1alpha1.RemoveFinished
				info.CleanupMap["node-1"].OsdMapping["25"].RemoveStatus.DeviceCleanUpJob = &lcmv1alpha1.RemoveStatus{
					Status:    lcmv1alpha1.RemoveFailed,
					StartedAt: "2025-04-14T14:30:47Z",
					Name:      "device-cleanup-job-node-1-25",
					Error:     "job failed, check logs",
				}
				info.Issues = []string{
					"[node 'node-1'] deployment 'rook-ceph-osd-25' is not removed, because job 'device-cleanup-job-node-1-25' is failed",
					"[node 'node-1'] disk cleanup job 'device-cleanup-job-node-1-25' has failed, clean up disk/partitions manually",
				}
				return info
			}(),
			requeueRequired: true,
		},
		{
			name: "processing - remove stray osds with partitions",
			taskConfig: taskConfig{
				task:        unitinputs.GetTaskForRemove(unitinputs.CephOsdRemoveTaskOnValidation, unitinputs.StrayOnNodeAndInCrushRemoveMap.DeepCopy()),
				cephCluster: &unitinputs.ReefCephClusterReady,
			},
			cephCliOutput: map[string]string{
				"ceph osd ls": "0\n9\n",
				"ceph osd purge 2 --force --yes-i-really-mean-it": "",
				"ceph auth del osd.2":                             "",
			},
			expectedRemoveMap: unitinputs.GetInfoWithStatus(unitinputs.StrayOnNodeAndInCrushRemoveMap,
				map[string]*lcmv1alpha1.RemoveResult{
					"0.06bf4d7c-9603-41a4-b250-284ecf3ecb2f.__stray": {
						OsdRemoveStatus: &lcmv1alpha1.RemoveStatus{Status: lcmv1alpha1.RemoveSkipped},
					},
					"2.61869d90-2c45-4f02-b7c3-96955f41e2ca.__stray": {
						OsdRemoveStatus: &lcmv1alpha1.RemoveStatus{
							Status:     lcmv1alpha1.RemoveFinished,
							FinishedAt: "2025-04-14T14:30:50Z",
						},
					},
				},
			),
			requeueRequired: true,
		},
		{
			name: "processing - clean stray osds from crush without partitions",
			taskConfig: taskConfig{
				task: unitinputs.GetTaskForRemove(unitinputs.CephOsdRemoveTaskOnValidation, unitinputs.GetInfoWithStatus(unitinputs.StrayOnlyInCrushRemoveMap,
					map[string]*lcmv1alpha1.RemoveResult{
						"2": {
							OsdRemoveStatus: &lcmv1alpha1.RemoveStatus{
								Status:     lcmv1alpha1.RemoveFinished,
								FinishedAt: "2025-04-14T14:30:51Z",
							},
						},
					},
				)),
				cephCluster: &unitinputs.ReefCephClusterReady,
			},
			expectedRemoveMap: unitinputs.GetInfoWithStatus(unitinputs.StrayOnlyInCrushRemoveMap,
				map[string]*lcmv1alpha1.RemoveResult{
					"2": {
						OsdRemoveStatus: &lcmv1alpha1.RemoveStatus{
							Status:     lcmv1alpha1.RemoveFinished,
							FinishedAt: "2025-04-14T14:30:51Z",
						},
						DeviceCleanUpJob: &lcmv1alpha1.RemoveStatus{
							Status: lcmv1alpha1.RemoveSkipped,
						},
						DeployRemoveStatus: &lcmv1alpha1.RemoveStatus{
							Status:     lcmv1alpha1.RemoveFinished,
							Name:       "rook-ceph-osd-2",
							StartedAt:  "2025-04-14T14:30:51Z",
							FinishedAt: "2025-04-14T14:30:51Z",
						},
					},
				},
			),
			requeueRequired: true,
		},
		{
			name: "processing - skip deploy remove for stray not in crush and remove for present in crush ",
			taskConfig: taskConfig{
				task: unitinputs.GetTaskForRemove(unitinputs.CephOsdRemoveTaskOnValidation, unitinputs.GetInfoWithStatus(unitinputs.StrayOnNodeAndInCrushRemoveMap,
					map[string]*lcmv1alpha1.RemoveResult{
						"0.06bf4d7c-9603-41a4-b250-284ecf3ecb2f.__stray": {
							OsdRemoveStatus:  &lcmv1alpha1.RemoveStatus{Status: lcmv1alpha1.RemoveSkipped},
							DeviceCleanUpJob: &lcmv1alpha1.RemoveStatus{Status: lcmv1alpha1.RemoveSkipped},
						},
						"2.61869d90-2c45-4f02-b7c3-96955f41e2ca.__stray": {
							OsdRemoveStatus: &lcmv1alpha1.RemoveStatus{
								Status:     lcmv1alpha1.RemoveFinished,
								FinishedAt: "2025-04-14T14:30:52Z",
							},
							DeviceCleanUpJob: &lcmv1alpha1.RemoveStatus{
								Status:    lcmv1alpha1.RemoveInProgress,
								Name:      "device-cleanup-job-node-2-2",
								StartedAt: "2025-04-14T14:10:52Z",
							},
						},
					},
				)),
				cephCluster: &unitinputs.ReefCephClusterReady,
			},
			batchJob: unitinputs.GetCleanupJobOnlyStatus("device-cleanup-job-node-2-2", "lcm-namespace", 0, 0, 1),
			expectedRemoveMap: unitinputs.GetInfoWithStatus(unitinputs.StrayOnNodeAndInCrushRemoveMap,
				map[string]*lcmv1alpha1.RemoveResult{
					"0.06bf4d7c-9603-41a4-b250-284ecf3ecb2f.__stray": {
						OsdRemoveStatus:    &lcmv1alpha1.RemoveStatus{Status: lcmv1alpha1.RemoveSkipped},
						DeviceCleanUpJob:   &lcmv1alpha1.RemoveStatus{Status: lcmv1alpha1.RemoveSkipped},
						DeployRemoveStatus: &lcmv1alpha1.RemoveStatus{Status: lcmv1alpha1.RemoveSkipped},
					},
					"2.61869d90-2c45-4f02-b7c3-96955f41e2ca.__stray": {
						OsdRemoveStatus: &lcmv1alpha1.RemoveStatus{
							Status:     lcmv1alpha1.RemoveFinished,
							FinishedAt: "2025-04-14T14:30:52Z",
						},
						DeviceCleanUpJob: &lcmv1alpha1.RemoveStatus{
							Status:     lcmv1alpha1.RemoveCompleted,
							Name:       "device-cleanup-job-node-2-2",
							StartedAt:  "2025-04-14T14:10:52Z",
							FinishedAt: "2025-04-14T14:30:52Z",
						},
						DeployRemoveStatus: &lcmv1alpha1.RemoveStatus{
							Status:     lcmv1alpha1.RemoveFinished,
							Name:       "rook-ceph-osd-2",
							StartedAt:  "2025-04-14T14:30:52Z",
							FinishedAt: "2025-04-14T14:30:52Z",
						},
					},
				},
			),
			requeueRequired: true,
		},
		{
			name: "processing - remove host from crush",
			taskConfig: taskConfig{
				task: unitinputs.GetTaskForRemove(unitinputs.CephOsdRemoveTaskOnValidation,
					&lcmv1alpha1.TaskRemoveInfo{CleanupMap: map[string]lcmv1alpha1.HostMapping{"node-1": {DropFromCrush: true}}}),
				cephCluster: &unitinputs.ReefCephClusterReady,
			},
			cephCliOutput: map[string]string{
				"ceph osd crush remove node-1": "",
			},
			expectedRemoveMap: &lcmv1alpha1.TaskRemoveInfo{
				CleanupMap: map[string]lcmv1alpha1.HostMapping{
					"node-1": {
						DropFromCrush: true,
						HostRemoveStatus: &lcmv1alpha1.RemoveStatus{
							Status:     lcmv1alpha1.RemoveFinished,
							StartedAt:  "2025-04-14T14:30:53Z",
							FinishedAt: "2025-04-14T14:30:53Z",
						},
					},
				}},
			requeueRequired: true,
		},
		{
			name: "processing - skip cleanup job for crush remove only",
			taskConfig: taskConfig{
				task: unitinputs.GetTaskForRemove(unitinputs.CephOsdRemoveTaskOnValidation,
					&lcmv1alpha1.TaskRemoveInfo{
						CleanupMap: map[string]lcmv1alpha1.HostMapping{
							"node-1": {
								DropFromCrush: true,
								OsdMapping: map[string]lcmv1alpha1.OsdMapping{
									"20": {
										RemoveStatus: &lcmv1alpha1.RemoveResult{
											OsdRemoveStatus: &lcmv1alpha1.RemoveStatus{Status: lcmv1alpha1.RemoveFinished},
										},
									},
								},
								HostRemoveStatus: &lcmv1alpha1.RemoveStatus{
									Status:     lcmv1alpha1.RemoveFinished,
									StartedAt:  "2025-04-14T14:30:53Z",
									FinishedAt: "2025-04-14T14:30:53Z",
								},
							},
						}}),
				cephCluster: &unitinputs.ReefCephClusterReady,
			},
			cephCliOutput: map[string]string{
				"ceph osd crush remove node-1": "",
			},
			expectedRemoveMap: &lcmv1alpha1.TaskRemoveInfo{
				Issues: []string{},
				CleanupMap: map[string]lcmv1alpha1.HostMapping{
					"node-1": {
						DropFromCrush: true,
						OsdMapping: map[string]lcmv1alpha1.OsdMapping{
							"20": {
								RemoveStatus: &lcmv1alpha1.RemoveResult{
									OsdRemoveStatus:  &lcmv1alpha1.RemoveStatus{Status: lcmv1alpha1.RemoveFinished},
									DeviceCleanUpJob: &lcmv1alpha1.RemoveStatus{Status: lcmv1alpha1.RemoveSkipped},
									DeployRemoveStatus: &lcmv1alpha1.RemoveStatus{
										Status:     lcmv1alpha1.RemoveFinished,
										Name:       "rook-ceph-osd-20",
										StartedAt:  "2025-04-14T14:30:54Z",
										FinishedAt: "2025-04-14T14:30:54Z",
									},
								},
							},
						},
						HostRemoveStatus: &lcmv1alpha1.RemoveStatus{
							Status:     lcmv1alpha1.RemoveFinished,
							StartedAt:  "2025-04-14T14:30:53Z",
							FinishedAt: "2025-04-14T14:30:53Z",
						},
					},
				}},
			requeueRequired: true,
		},
		{
			name: "processing - handle failed deploy remove and finish",
			taskConfig: taskConfig{
				task: unitinputs.GetTaskForRemove(unitinputs.CephOsdRemoveTaskOnValidation,
					&lcmv1alpha1.TaskRemoveInfo{
						CleanupMap: map[string]lcmv1alpha1.HostMapping{
							"node-1": {
								OsdMapping: map[string]lcmv1alpha1.OsdMapping{
									"20": {
										RemoveStatus: &lcmv1alpha1.RemoveResult{
											OsdRemoveStatus:    &lcmv1alpha1.RemoveStatus{Status: lcmv1alpha1.RemoveFinished},
											DeviceCleanUpJob:   &lcmv1alpha1.RemoveStatus{Status: lcmv1alpha1.RemoveCompleted},
											DeployRemoveStatus: &lcmv1alpha1.RemoveStatus{Status: lcmv1alpha1.RemoveFailed},
										},
									},
									"21": {
										RemoveStatus: &lcmv1alpha1.RemoveResult{
											OsdRemoveStatus:  &lcmv1alpha1.RemoveStatus{Status: lcmv1alpha1.RemoveFinished},
											DeviceCleanUpJob: &lcmv1alpha1.RemoveStatus{Status: lcmv1alpha1.RemoveFinished},
										},
									},
								},
								HostRemoveStatus: &lcmv1alpha1.RemoveStatus{
									Status:     lcmv1alpha1.RemoveFinished,
									StartedAt:  "2025-04-14T14:30:53Z",
									FinishedAt: "2025-04-14T14:30:53Z",
								},
							},
						}}),
				cephCluster: &unitinputs.ReefCephClusterReady,
			},
			expectedRemoveMap: &lcmv1alpha1.TaskRemoveInfo{
				Issues: []string{
					"[node 'node-1'] failed to remove deployment 'rook-ceph-osd-20'",
				},
				Warnings: []string{
					"unexpected remove status 'Removed' for device cleanup job '', deployment remove for osd '21' on node 'node-1' skipped",
				},
				CleanupMap: map[string]lcmv1alpha1.HostMapping{
					"node-1": {
						OsdMapping: map[string]lcmv1alpha1.OsdMapping{
							"20": {
								RemoveStatus: &lcmv1alpha1.RemoveResult{
									OsdRemoveStatus:    &lcmv1alpha1.RemoveStatus{Status: lcmv1alpha1.RemoveFinished},
									DeviceCleanUpJob:   &lcmv1alpha1.RemoveStatus{Status: lcmv1alpha1.RemoveCompleted},
									DeployRemoveStatus: &lcmv1alpha1.RemoveStatus{Status: lcmv1alpha1.RemoveFailed},
								},
							},
							"21": {
								RemoveStatus: &lcmv1alpha1.RemoveResult{
									OsdRemoveStatus:  &lcmv1alpha1.RemoveStatus{Status: lcmv1alpha1.RemoveFinished},
									DeviceCleanUpJob: &lcmv1alpha1.RemoveStatus{Status: lcmv1alpha1.RemoveFinished},
								},
							},
						},
						HostRemoveStatus: &lcmv1alpha1.RemoveStatus{
							Status:     lcmv1alpha1.RemoveFinished,
							StartedAt:  "2025-04-14T14:30:53Z",
							FinishedAt: "2025-04-14T14:30:53Z",
						},
					},
				}},
			requeueRequired: true,
			finished:        true,
		},
		{
			name: "processing - spec has some skip cleanup jobs",
			taskConfig: taskConfig{
				task: unitinputs.GetTaskForRemove(unitinputs.CephOsdRemoveTaskOnValidation, unitinputs.GetInfoWithStatus(unitinputs.SkipCleanupJobRemoveMap.DeepCopy(),
					map[string]*lcmv1alpha1.RemoveResult{
						"*": {OsdRemoveStatus: &lcmv1alpha1.RemoveStatus{Status: lcmv1alpha1.RemoveFinished}},
					})),
				cephCluster: &unitinputs.ReefCephClusterReady,
			},
			expectedRemoveMap: unitinputs.GetInfoWithStatus(unitinputs.SkipCleanupJobRemoveMap.DeepCopy(),
				map[string]*lcmv1alpha1.RemoveResult{
					"*": {
						OsdRemoveStatus:    &lcmv1alpha1.RemoveStatus{Status: lcmv1alpha1.RemoveFinished},
						DeviceCleanUpJob:   &lcmv1alpha1.RemoveStatus{Status: lcmv1alpha1.RemoveSkipped},
						DeployRemoveStatus: &lcmv1alpha1.RemoveStatus{Status: lcmv1alpha1.RemoveSkipped},
					},
					"4": {
						OsdRemoveStatus: &lcmv1alpha1.RemoveStatus{Status: lcmv1alpha1.RemoveFinished},
						DeviceCleanUpJob: &lcmv1alpha1.RemoveStatus{
							Status:    lcmv1alpha1.RemoveInProgress,
							Name:      "device-cleanup-job-node-2-4",
							StartedAt: "2025-04-14T14:30:56Z",
						},
					},
				}),
		},
		{
			name: "processing - continue postponed job",
			taskConfig: taskConfig{
				task: unitinputs.GetTaskForRemove(unitinputs.CephOsdRemoveTaskOnValidation, func() *lcmv1alpha1.TaskRemoveInfo {
					info := unitinputs.GetInfoWithStatus(unitinputs.FullNodesRemoveMap,
						map[string]*lcmv1alpha1.RemoveResult{
							"*": {
								OsdRemoveStatus:    &lcmv1alpha1.RemoveStatus{Status: lcmv1alpha1.RemoveFinished},
								DeviceCleanUpJob:   &lcmv1alpha1.RemoveStatus{Status: lcmv1alpha1.RemoveCompleted},
								DeployRemoveStatus: &lcmv1alpha1.RemoveStatus{Status: lcmv1alpha1.RemoveFinished},
							},
							"20": {
								OsdRemoveStatus:  &lcmv1alpha1.RemoveStatus{Status: lcmv1alpha1.RemoveFinished},
								DeviceCleanUpJob: &lcmv1alpha1.RemoveStatus{Status: lcmv1alpha1.RemovePending},
							},
							"4":  {OsdRemoveStatus: &lcmv1alpha1.RemoveStatus{Status: lcmv1alpha1.RemovePending}},
							"30": {OsdRemoveStatus: &lcmv1alpha1.RemoveStatus{Status: lcmv1alpha1.RemovePending}},
						},
					)
					return info
				}()),
				cephCluster: &unitinputs.ReefCephClusterReady,
			},
			cephCliOutput: map[string]string{
				"ceph osd info 4 --format json":  `{"osd":4, "up":1, "in":1}`,
				"ceph osd info 30 --format json": `{"osd":30, "up":1, "in":1}`,
			},
			expectedRemoveMap: unitinputs.GetInfoWithStatus(unitinputs.FullNodesRemoveMap,
				map[string]*lcmv1alpha1.RemoveResult{
					"*": {
						OsdRemoveStatus:    &lcmv1alpha1.RemoveStatus{Status: lcmv1alpha1.RemoveFinished},
						DeviceCleanUpJob:   &lcmv1alpha1.RemoveStatus{Status: lcmv1alpha1.RemoveCompleted},
						DeployRemoveStatus: &lcmv1alpha1.RemoveStatus{Status: lcmv1alpha1.RemoveFinished},
					},
					"20": {
						OsdRemoveStatus: &lcmv1alpha1.RemoveStatus{Status: lcmv1alpha1.RemoveFinished},
						DeviceCleanUpJob: &lcmv1alpha1.RemoveStatus{
							Status:    lcmv1alpha1.RemoveInProgress,
							Name:      "device-cleanup-job-node-1-20",
							StartedAt: "2025-04-14T14:30:57Z",
						},
					},
					"4": {OsdRemoveStatus: &lcmv1alpha1.RemoveStatus{
						Status:    lcmv1alpha1.RemovePending,
						StartedAt: "2025-04-14T14:30:57Z",
					}},
					"30": {OsdRemoveStatus: &lcmv1alpha1.RemoveStatus{
						Status:    lcmv1alpha1.RemovePending,
						StartedAt: "2025-04-14T14:30:57Z",
					}},
				},
			),
		},
	}
	oldRunCmd := lcmcommon.RunPodCommand
	oldRetryTimeout := commandRetryRunTimeout
	commandRetryRunTimeout = 0
	oldTimeFunc := lcmcommon.GetCurrentTimeString
	for idx, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := fakeCephReconcileConfig(&test.taskConfig, nil)
			faketestclients.FakeReaction(c.api.Kubeclientset.CoreV1(), "list", []string{"pods"}, map[string]runtime.Object{"pods": unitinputs.ToolBoxPodList}, nil)

			lcmcommon.GetCurrentTimeString = func() string {
				return time.Date(2025, 4, 14, 14, 30, 40+idx, 0, time.UTC).Format(time.RFC3339)
			}

			inputRes := map[string]runtime.Object{"jobs": &batch.JobList{}}
			if test.batchJob != nil {
				inputRes["jobs"] = &batch.JobList{Items: []batch.Job{*test.batchJob}}
			}
			faketestclients.FakeReaction(c.api.Kubeclientset.BatchV1(), "get", []string{"jobs"}, inputRes, nil)

			lcmcommon.RunPodCommand = func(e lcmcommon.ExecConfig) (string, string, error) {
				if output, ok := test.cephCliOutput[e.Command]; ok {
					return output, "", nil
				}
				return "", "", errors.New("run failed")
			}

			finished, result := c.processOsdRemoveTask()
			assert.Equal(t, test.expectedRemoveMap, result)
			assert.Equal(t, test.finished, finished)
			assert.Equal(t, test.requeueRequired, c.taskConfig.requeueNow)
			faketestclients.CleanupFakeClientReactions(c.api.Kubeclientset.BatchV1())
			faketestclients.CleanupFakeClientReactions(c.api.Kubeclientset.CoreV1())
		})
	}
	lcmcommon.RunPodCommand = oldRunCmd
	commandRetryRunTimeout = oldRetryTimeout
	lcmcommon.GetCurrentTimeString = oldTimeFunc
}

func TestRemoveStray(t *testing.T) {
	taskConfigForTest := taskConfig{task: unitinputs.CephOsdRemoveTaskProcessing, cephCluster: &unitinputs.ReefCephClusterReady}
	var1 := int32(1)
	var0 := int32(0)
	osdDeploy := unitinputs.GetDeployment("rook-ceph-osd-0", "rook-ceph", map[string]string{"app": "rook-ceph-osd"}, &var1)
	deployList := &appsv1.DeploymentList{Items: []appsv1.Deployment{*osdDeploy}}

	tests := []struct {
		name             string
		cliOutput        map[string]string
		scaleError       bool
		expectedReplicas *int32
		currentStatus    *lcmv1alpha1.RemoveStatus
		expectedStatus   *lcmv1alpha1.RemoveStatus
	}{
		{
			name: "failed to check osds",
			currentStatus: &lcmv1alpha1.RemoveStatus{
				Status: lcmv1alpha1.RemoveStray,
			},
			expectedReplicas: &var1,
			expectedStatus: &lcmv1alpha1.RemoveStatus{
				Status: lcmv1alpha1.RemoveFailed,
				Error:  "Retries (5/5) exceeded: failed to run command 'ceph osd ls': command failed",
			},
		},
		{
			name: "osd deployment scale failed",
			cliOutput: map[string]string{
				"ceph osd ls": "5\n",
			},
			scaleError: true,
			currentStatus: &lcmv1alpha1.RemoveStatus{
				Status: lcmv1alpha1.RemoveStray,
			},
			expectedReplicas: &var1,
			expectedStatus: &lcmv1alpha1.RemoveStatus{
				Status:    lcmv1alpha1.RemoveFailed,
				Error:     "Retries (5/5) exceeded: failed to scale osd deployment",
				StartedAt: "time-1",
			},
		},
		{
			name: "osd keyring drop failed",
			cliOutput: map[string]string{
				"ceph osd ls": "5\n",
			},
			currentStatus: &lcmv1alpha1.RemoveStatus{
				Status: lcmv1alpha1.RemoveStray,
			},
			expectedReplicas: &var0,
			expectedStatus: &lcmv1alpha1.RemoveStatus{
				Status:    lcmv1alpha1.RemoveFailed,
				Error:     "Retries (5/5) exceeded: failed to run command 'ceph auth del osd.0': command failed",
				StartedAt: "time-2",
			},
		},
		{
			name: "stray osd cleanup completed",
			cliOutput: map[string]string{
				"ceph osd ls":         "5\n",
				"ceph auth del osd.0": "",
			},
			currentStatus: &lcmv1alpha1.RemoveStatus{
				Status: lcmv1alpha1.RemoveStray,
			},
			expectedReplicas: &var0,
			expectedStatus: &lcmv1alpha1.RemoveStatus{
				Status:     lcmv1alpha1.RemoveFinished,
				StartedAt:  "time-3",
				FinishedAt: "time-3",
			},
		},
		{
			name: "stray osd cleanup skipped",
			cliOutput: map[string]string{
				"ceph osd ls": "0\n",
			},
			currentStatus: &lcmv1alpha1.RemoveStatus{
				Status: lcmv1alpha1.RemoveStray,
			},
			expectedReplicas: &var1,
			expectedStatus: &lcmv1alpha1.RemoveStatus{
				Status: lcmv1alpha1.RemoveSkipped,
			},
		},
	}
	oldRunCmd := lcmcommon.RunPodCommand
	oldValue := commandRetryRunTimeout
	commandRetryRunTimeout = 0
	oldTimeFunc := lcmcommon.GetCurrentTimeString
	for idx, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := fakeCephReconcileConfig(&taskConfigForTest, nil)
			inputRes := map[string]runtime.Object{"deployments": deployList.DeepCopy()}
			apiErrors := map[string]error{}
			if test.scaleError {
				apiErrors = map[string]error{"update-deployments": errors.New("failed to scale osd deployment")}
			}
			faketestclients.FakeReaction(c.api.Kubeclientset.AppsV1(), "update", []string{"deployments"}, inputRes, apiErrors)
			faketestclients.FakeReaction(c.api.Kubeclientset.CoreV1(), "list", []string{"pods"}, map[string]runtime.Object{"pods": unitinputs.ToolBoxPodList}, nil)

			lcmcommon.GetCurrentTimeString = func() string {
				return fmt.Sprintf("time-%d", idx)
			}

			lcmcommon.RunPodCommand = func(e lcmcommon.ExecConfig) (string, string, error) {
				if output, ok := test.cliOutput[e.Command]; ok {
					return output, "", nil
				}
				return "", "", errors.New("command failed")
			}

			status := c.removeStray("0.06bf4d7c-9603-41a4-b250-284ecf3ecb2f.__stray", test.currentStatus)
			assert.Equal(t, test.expectedStatus, status)
			if test.expectedReplicas != nil {
				assert.Equal(t, test.expectedReplicas, inputRes["deployments"].(*appsv1.DeploymentList).Items[0].Spec.Replicas)
			}
			faketestclients.CleanupFakeClientReactions(c.api.Kubeclientset.AppsV1())
			faketestclients.CleanupFakeClientReactions(c.api.Kubeclientset.CoreV1())
		})
	}
	commandRetryRunTimeout = oldValue
	lcmcommon.RunPodCommand = oldRunCmd
	lcmcommon.GetCurrentTimeString = oldTimeFunc
}

func TestRemoveFromCrush(t *testing.T) {
	taskConfigForTest := taskConfig{task: unitinputs.CephOsdRemoveTaskProcessing, cephCluster: &unitinputs.ReefCephClusterReady}
	var1 := int32(1)
	var0 := int32(0)
	osdDeploy := unitinputs.GetDeployment("rook-ceph-osd-5", "rook-ceph", map[string]string{"app": "rook-ceph-osd"}, &var1)
	deployList := &appsv1.DeploymentList{Items: []appsv1.Deployment{*osdDeploy}}

	tests := []struct {
		name             string
		cliOutput        map[string]string
		scaleError       bool
		currentStatus    *lcmv1alpha1.RemoveStatus
		expectedReplicas *int32
		expectedStatus   *lcmv1alpha1.RemoveStatus
	}{
		{
			name:       "osd deployment scale failed",
			scaleError: true,
			currentStatus: &lcmv1alpha1.RemoveStatus{
				Status: lcmv1alpha1.RemoveInProgress,
			},
			expectedReplicas: &var1,
			expectedStatus: &lcmv1alpha1.RemoveStatus{
				Status: lcmv1alpha1.RemoveFailed,
				Error:  "Retries (5/5) exceeded: failed to scale osd deployment",
			},
		},
		{
			name: "failed to remove osd from crush map",
			currentStatus: &lcmv1alpha1.RemoveStatus{
				Status: lcmv1alpha1.RemoveInProgress,
			},
			expectedReplicas: &var0,
			expectedStatus: &lcmv1alpha1.RemoveStatus{
				Status: lcmv1alpha1.RemoveFailed,
				Error:  "Retries (5/5) exceeded: failed to run command 'ceph osd purge 5 --force --yes-i-really-mean-it': command failed",
			},
		},
		{
			name: "failed to drop osd keyring",
			cliOutput: map[string]string{
				"ceph osd purge 5 --force --yes-i-really-mean-it": "",
			},
			currentStatus: &lcmv1alpha1.RemoveStatus{
				Status: lcmv1alpha1.RemoveInProgress,
			},
			expectedReplicas: &var0,
			expectedStatus: &lcmv1alpha1.RemoveStatus{
				Status: lcmv1alpha1.RemoveFailed,
				Error:  "Retries (5/5) exceeded: failed to run command 'ceph auth del osd.5': command failed",
			},
		},
		{
			name: "remove is completed",
			cliOutput: map[string]string{
				"ceph osd purge 5 --force --yes-i-really-mean-it": "",
				"ceph auth del osd.5":                             "",
			},
			currentStatus: &lcmv1alpha1.RemoveStatus{
				Status:    lcmv1alpha1.RemoveInProgress,
				StartedAt: "start-time",
			},
			expectedReplicas: &var0,
			expectedStatus: &lcmv1alpha1.RemoveStatus{
				Status:     lcmv1alpha1.RemoveFinished,
				StartedAt:  "start-time",
				FinishedAt: "finish-time",
			},
		},
	}
	oldRunCmd := lcmcommon.RunPodCommand
	oldValue := commandRetryRunTimeout
	commandRetryRunTimeout = 0
	oldTimeFunc := lcmcommon.GetCurrentTimeString

	lcmcommon.GetCurrentTimeString = func() string {
		return "finish-time"
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := fakeCephReconcileConfig(&taskConfigForTest, nil)
			inputRes := map[string]runtime.Object{"deployments": deployList.DeepCopy()}
			apiErrors := map[string]error{}
			if test.scaleError {
				apiErrors = map[string]error{"update-deployments": errors.New("failed to scale osd deployment")}
			}
			faketestclients.FakeReaction(c.api.Kubeclientset.AppsV1(), "update", []string{"deployments"}, inputRes, apiErrors)
			faketestclients.FakeReaction(c.api.Kubeclientset.CoreV1(), "list", []string{"pods"}, map[string]runtime.Object{"pods": unitinputs.ToolBoxPodList}, nil)

			lcmcommon.RunPodCommand = func(e lcmcommon.ExecConfig) (string, string, error) {
				if output, ok := test.cliOutput[e.Command]; ok {
					return output, "", nil
				}
				return "", "", errors.New("command failed")
			}

			status := c.removeFromCrush("5", test.currentStatus)
			assert.Equal(t, test.expectedStatus, status)
			if test.expectedReplicas != nil {
				assert.Equal(t, test.expectedReplicas, inputRes["deployments"].(*appsv1.DeploymentList).Items[0].Spec.Replicas)
			}
			faketestclients.CleanupFakeClientReactions(c.api.Kubeclientset.AppsV1())
			faketestclients.CleanupFakeClientReactions(c.api.Kubeclientset.CoreV1())
		})
	}
	commandRetryRunTimeout = oldValue
	lcmcommon.RunPodCommand = oldRunCmd
	lcmcommon.GetCurrentTimeString = oldTimeFunc
}

func TestCheckRebalance(t *testing.T) {
	taskConfigForTest := taskConfig{task: unitinputs.CephOsdRemoveTaskProcessing, cephCluster: &unitinputs.ReefCephClusterReady}
	rebalanceStatusNoTimestamp := &lcmv1alpha1.RemoveStatus{Status: lcmv1alpha1.RemoveWaitingRebalance}
	rebalanceStatus := &lcmv1alpha1.RemoveStatus{
		Status:    lcmv1alpha1.RemoveWaitingRebalance,
		StartedAt: time.Now().Format(time.RFC3339),
	}
	pgPresent := `{"pg_stats": [ {"key": "value"} ]}`

	tests := []struct {
		name           string
		cliOutput      string
		currentStatus  *lcmv1alpha1.RemoveStatus
		expectedStatus *lcmv1alpha1.RemoveStatus
	}{
		{
			name:          "failed to check pgs",
			currentStatus: rebalanceStatusNoTimestamp.DeepCopy(),
			expectedStatus: &lcmv1alpha1.RemoveStatus{
				Status: lcmv1alpha1.RemoveFailed,
				Error:  "Retries (5/5) exceeded: failed to run command 'ceph pg ls-by-osd 5 --format json': run failed",
			},
		},
		{
			name:           "pgs is present for osd, failed to check start timestamp",
			cliOutput:      pgPresent,
			currentStatus:  rebalanceStatusNoTimestamp.DeepCopy(),
			expectedStatus: rebalanceStatusNoTimestamp,
		},
		{
			name:           "pgs is present for osd, rebalance is not finished",
			cliOutput:      pgPresent,
			currentStatus:  rebalanceStatus.DeepCopy(),
			expectedStatus: rebalanceStatus,
		},
		{
			name:      "pgs is present for osd, rebalance is timeouted",
			cliOutput: pgPresent,
			currentStatus: &lcmv1alpha1.RemoveStatus{
				Status:    lcmv1alpha1.RemoveWaitingRebalance,
				StartedAt: "2021-08-15T14:30:41Z",
			},
			expectedStatus: &lcmv1alpha1.RemoveStatus{
				Status:    lcmv1alpha1.RemoveFailed,
				Error:     "timeout (30m0s) reached for waiting pg rebalance",
				StartedAt: "2021-08-15T14:30:41Z",
			},
		},
		{
			name:          "rebalance is not finished",
			cliOutput:     `{"pg_stats":[]}`,
			currentStatus: rebalanceStatus.DeepCopy(),
			expectedStatus: &lcmv1alpha1.RemoveStatus{
				Status:    lcmv1alpha1.RemoveInProgress,
				StartedAt: rebalanceStatus.StartedAt,
			},
		},
	}
	oldRunCmd := lcmcommon.RunPodCommand
	oldValue := commandRetryRunTimeout
	commandRetryRunTimeout = 0
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := fakeCephReconcileConfig(&taskConfigForTest, nil)
			faketestclients.FakeReaction(c.api.Kubeclientset.CoreV1(), "list", []string{"pods"}, map[string]runtime.Object{"pods": unitinputs.ToolBoxPodList}, nil)

			lcmcommon.RunPodCommand = func(_ lcmcommon.ExecConfig) (string, string, error) {
				if test.cliOutput != "" {
					return test.cliOutput, "", nil
				}
				return "", "", errors.New("run failed")
			}

			status := c.checkRebalance("5", test.currentStatus)
			assert.Equal(t, test.expectedStatus, status)
		})
	}
	commandRetryRunTimeout = oldValue
	lcmcommon.RunPodCommand = oldRunCmd
}

func TestTryMoveOsdOut(t *testing.T) {
	taskConfigForTest := taskConfig{task: unitinputs.CephOsdRemoveTaskProcessing, cephCluster: &unitinputs.ReefCephClusterReady}

	tests := []struct {
		name           string
		cliOutput      map[string]string
		currentStatus  *lcmv1alpha1.RemoveStatus
		expectedStatus *lcmv1alpha1.RemoveStatus
	}{
		{
			name: "failed to get osd info",
			expectedStatus: &lcmv1alpha1.RemoveStatus{
				Status: lcmv1alpha1.RemoveFailed,
				Error:  "Retries (5/5) exceeded: failed to run command 'ceph osd info 5 --format json': run failed",
			},
		},
		{
			name: "osd is not ok to stop",
			cliOutput: map[string]string{
				"ceph osd info 5 --format json": `{"osd":5, "up":1, "in":1}`,
			},
			expectedStatus: &lcmv1alpha1.RemoveStatus{
				Status:    lcmv1alpha1.RemovePending,
				StartedAt: "2021-08-15T14:30:41Z",
			},
		},
		{
			name: "osd is not ok to stop and incorrect timestamp",
			cliOutput: map[string]string{
				"ceph osd info 5 --format json": `{"osd":5, "up":1, "in":1}`,
			},
			currentStatus: &lcmv1alpha1.RemoveStatus{
				Status:    lcmv1alpha1.RemovePending,
				StartedAt: "current-time-1",
			},
			expectedStatus: &lcmv1alpha1.RemoveStatus{
				Status:    lcmv1alpha1.RemovePending,
				StartedAt: "current-time-1",
			},
		},
		{
			name: "osd is not ok to stop and timeout exceeded",
			cliOutput: map[string]string{
				"ceph osd info 5 --format json": `{"osd":5, "up":1, "in":1}`,
			},
			currentStatus: &lcmv1alpha1.RemoveStatus{
				Status:    lcmv1alpha1.RemovePending,
				StartedAt: "2021-08-15T14:00:05Z",
			},
			expectedStatus: &lcmv1alpha1.RemoveStatus{
				Status: lcmv1alpha1.RemoveFailed,
				Error:  "timeout (30m0s) reached for waiting ok-to-stop on osd '5'",
			},
		},
		{
			name: "failed to reweight osd to 0",
			cliOutput: map[string]string{
				"ceph osd info 5 --format json": `{"osd":5, "up":1, "in":1}`,
				"ceph osd ok-to-stop 5":         "ok",
			},
			expectedStatus: &lcmv1alpha1.RemoveStatus{
				Status: lcmv1alpha1.RemoveFailed,
				Error:  "Retries (5/5) exceeded: failed to run command 'ceph osd crush reweight osd.5 0.0': run failed",
			},
		},
		{
			name: "osd in and reweighted, wait rebalancing",
			cliOutput: map[string]string{
				"ceph osd info 5 --format json":     `{"osd":5, "up":1, "in":1}`,
				"ceph osd ok-to-stop 5":             "ok",
				"ceph osd crush reweight osd.5 0.0": "",
			},
			expectedStatus: &lcmv1alpha1.RemoveStatus{
				Status:    lcmv1alpha1.RemoveWaitingRebalance,
				StartedAt: "2021-08-15T14:30:45Z",
			},
		},
		{
			name: "osd not in and up, wait rebalancing",
			cliOutput: map[string]string{
				"ceph osd info 5 --format json": `{"osd":5, "up":1, "in":0}`,
			},
			expectedStatus: &lcmv1alpha1.RemoveStatus{
				Status:    lcmv1alpha1.RemoveWaitingRebalance,
				StartedAt: "2021-08-15T14:30:46Z",
			},
		},
		{
			name: "osd in and not up, ready to remove",
			cliOutput: map[string]string{
				"ceph osd info 5 --format json":     `{"osd":5, "up":0, "in":1}`,
				"ceph osd crush reweight osd.5 0.0": "",
			},
			expectedStatus: &lcmv1alpha1.RemoveStatus{
				Status:    lcmv1alpha1.RemoveInProgress,
				StartedAt: "2021-08-15T14:30:47Z",
			},
		},
		{
			name: "osd not in and not up, ready to remove",
			cliOutput: map[string]string{
				"ceph osd info 5 --format json": `{"osd":5, "up":0, "in":0}`,
			},
			expectedStatus: &lcmv1alpha1.RemoveStatus{
				Status:    lcmv1alpha1.RemoveInProgress,
				StartedAt: "2021-08-15T14:30:48Z",
			},
		},
	}
	oldRunCmd := lcmcommon.RunPodCommand
	oldTimeFunc := lcmcommon.GetCurrentTimeString
	oldValue := commandRetryRunTimeout
	commandRetryRunTimeout = 0
	for idx, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := fakeCephReconcileConfig(&taskConfigForTest, nil)
			faketestclients.FakeReaction(c.api.Kubeclientset.CoreV1(), "list", []string{"pods"}, map[string]runtime.Object{"pods": unitinputs.ToolBoxPodList}, nil)

			lcmcommon.GetCurrentTimeString = func() string {
				return time.Date(2021, 8, 15, 14, 30, 40+idx, 0, time.UTC).Format(time.RFC3339)
			}

			lcmcommon.RunPodCommand = func(e lcmcommon.ExecConfig) (string, string, error) {
				if output, ok := test.cliOutput[e.Command]; ok {
					return output, "", nil
				}
				return "", "", errors.New("run failed")
			}

			status := c.tryToMoveOsdOut("5", test.currentStatus)
			assert.Equal(t, test.expectedStatus, status)
		})
	}
	commandRetryRunTimeout = oldValue
	lcmcommon.GetCurrentTimeString = oldTimeFunc
	lcmcommon.RunPodCommand = oldRunCmd
}

func TestGetJobData(t *testing.T) {
	c := fakeCephReconcileConfig(nil, nil)
	devMappingZapProhibited := unitinputs.FullNodesRemoveMap.DeepCopy().CleanupMap["node-2"].OsdMapping["4"].DeviceMapping
	devInfo := devMappingZapProhibited["/dev/vdd"]
	devInfo.Zap = false
	devMappingZapProhibited["/dev/vdd"] = devInfo

	tests := []struct {
		name               string
		osd                string
		osdMapping         map[string]lcmv1alpha1.OsdMapping
		parallelDetected   bool
		expectedDeviceData map[string]lcmv1alpha1.DeviceInfo
	}{
		{
			name:               "single osd device",
			osd:                "20",
			osdMapping:         unitinputs.GetInfoWithStatus(unitinputs.DevNotInSpecRemoveMap, map[string]*lcmv1alpha1.RemoveResult{"20": nil}).CleanupMap["node-1"].OsdMapping,
			expectedDeviceData: unitinputs.DevNotInSpecRemoveMap.CleanupMap["node-1"].OsdMapping["20"].DeviceMapping,
		},
		{
			name: "job for other device is completed",
			osd:  "4",
			osdMapping: unitinputs.GetInfoWithStatus(unitinputs.FullNodesRemoveMap, map[string]*lcmv1alpha1.RemoveResult{
				"*": nil, "5": {DeviceCleanUpJob: &lcmv1alpha1.RemoveStatus{Status: lcmv1alpha1.RemoveCompleted}}}).CleanupMap["node-2"].OsdMapping,
			expectedDeviceData: unitinputs.FullNodesRemoveMap.CleanupMap["node-2"].OsdMapping["4"].DeviceMapping,
		},
		{
			name: "job for other device is in progress",
			osd:  "4",
			osdMapping: unitinputs.GetInfoWithStatus(unitinputs.FullNodesRemoveMap, map[string]*lcmv1alpha1.RemoveResult{
				"*": nil, "5": {DeviceCleanUpJob: &lcmv1alpha1.RemoveStatus{Status: lcmv1alpha1.RemoveInProgress}}}).CleanupMap["node-2"].OsdMapping,
			parallelDetected:   true,
			expectedDeviceData: unitinputs.FullNodesRemoveMap.CleanupMap["node-2"].OsdMapping["4"].DeviceMapping,
		},
		{
			name:               "device in use for different osd which is not removed, abort zap for current",
			osd:                "4",
			osdMapping:         unitinputs.NodesRemoveMapEmptyRemoveStatus.DeepCopy().CleanupMap["node-2"].OsdMapping,
			expectedDeviceData: devMappingZapProhibited,
		},
		{
			name: "device in use for different osd which is going to skip cleanup, abort zap for current",
			osd:  "4",
			osdMapping: unitinputs.GetInfoWithStatus(unitinputs.SkipCleanupJobRemoveMap, map[string]*lcmv1alpha1.RemoveResult{
				"*": nil}).CleanupMap["node-2"].OsdMapping,
			expectedDeviceData: devMappingZapProhibited,
		},
		{
			name: "device in use for different osd which has failed job, abort zap for current",
			osd:  "4",
			osdMapping: unitinputs.GetInfoWithStatus(unitinputs.FullNodesRemoveMap, map[string]*lcmv1alpha1.RemoveResult{
				"*": nil, "5": {DeviceCleanUpJob: &lcmv1alpha1.RemoveStatus{Status: lcmv1alpha1.RemoveFailed}}}).CleanupMap["node-2"].OsdMapping,
			expectedDeviceData: devMappingZapProhibited,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			parallelDetected, deviceData := c.getJobData(test.osd, test.osdMapping)
			assert.Equal(t, test.expectedDeviceData, deviceData)
			assert.Equal(t, test.parallelDetected, parallelDetected)
		})
	}
}

func TestHandleJobRun(t *testing.T) {
	taskConfigForTest := taskConfig{task: unitinputs.CephOsdRemoveTaskProcessing, cephCluster: &unitinputs.ReefCephClusterReady}
	statusForCreatedJob := unitinputs.GetInfoWithStatus(
		unitinputs.FullNodesRemoveMap, map[string]*lcmv1alpha1.RemoveResult{"4": {
			DeviceCleanUpJob: &lcmv1alpha1.RemoveStatus{
				Name:      "device-cleanup-job-node-2-4",
				Status:    lcmv1alpha1.RemoveInProgress,
				StartedAt: "current-time-4",
			},
		}})

	tests := []struct {
		name           string
		taskConfig     taskConfig
		osd            string
		host           string
		apiError       string
		removeInfo     *lcmv1alpha1.TaskRemoveInfo
		batchJob       *batch.Job
		expectedError  error
		expectedResult *lcmv1alpha1.RemoveStatus
	}{
		{
			name:       "job create delayed",
			taskConfig: taskConfigForTest,
			osd:        "4",
			host:       "node-2",
			removeInfo: unitinputs.GetInfoWithStatus(unitinputs.FullNodesRemoveMap, map[string]*lcmv1alpha1.RemoveResult{
				"*": nil, "5": {DeviceCleanUpJob: &lcmv1alpha1.RemoveStatus{Status: lcmv1alpha1.RemoveInProgress}}}),
			expectedResult: &lcmv1alpha1.RemoveStatus{Status: lcmv1alpha1.RemovePending},
		},
		{
			name:       "job create skipped",
			taskConfig: taskConfigForTest,
			osd:        "4",
			host:       "node-2",
			removeInfo: func() *lcmv1alpha1.TaskRemoveInfo {
				info := unitinputs.NodesRemoveMapEmptyRemoveStatus.DeepCopy()
				mapping := info.CleanupMap["node-2"].OsdMapping["4"]
				mapping.DeviceMapping = nil
				info.CleanupMap["node-2"].OsdMapping["4"] = mapping
				return info
			}(),
			expectedResult: &lcmv1alpha1.RemoveStatus{Status: lcmv1alpha1.RemoveSkipped},
		},
		{
			name:       "job failed to create with not full device info",
			taskConfig: taskConfigForTest,
			osd:        "2",
			host:       "node-2",
			removeInfo: &lcmv1alpha1.TaskRemoveInfo{
				CleanupMap: map[string]lcmv1alpha1.HostMapping{
					"node-2": {
						OsdMapping: map[string]lcmv1alpha1.OsdMapping{
							"2": {
								DeviceMapping: map[string]lcmv1alpha1.DeviceInfo{"/dev/vdc": {}},
								RemoveStatus:  &lcmv1alpha1.RemoveResult{},
							},
						},
					},
				},
			},
			expectedResult: &lcmv1alpha1.RemoveStatus{
				Name:   "device-cleanup-job-node-2-2",
				Status: lcmv1alpha1.RemoveFailed,
				Error:  "failed to run job: partition or device path missed in provided info",
			},
		},
		{
			name:       "job create - create failed",
			taskConfig: taskConfigForTest,
			osd:        "4",
			host:       "node-2",
			removeInfo: unitinputs.NodesRemoveMapEmptyRemoveStatus.DeepCopy(),
			apiError:   "create-jobs",
			expectedResult: &lcmv1alpha1.RemoveStatus{
				Name:   "device-cleanup-job-node-2-4",
				Status: lcmv1alpha1.RemoveFailed,
				Error:  "failed to run job: Retries (5/5) exceeded: failed to create-jobs",
			},
		},
		{
			name:           "job created",
			taskConfig:     taskConfigForTest,
			osd:            "4",
			host:           "node-2",
			removeInfo:     unitinputs.NodesRemoveMapEmptyRemoveStatus.DeepCopy(),
			expectedResult: statusForCreatedJob.CleanupMap["node-2"].OsdMapping["4"].RemoveStatus.DeviceCleanUpJob,
		},
		{
			name:       "job exists - cant check job",
			taskConfig: taskConfigForTest,
			osd:        "4",
			host:       "node-2",
			removeInfo: statusForCreatedJob.DeepCopy(),
			apiError:   "get-jobs",
			expectedResult: &lcmv1alpha1.RemoveStatus{
				Name:      "device-cleanup-job-node-2-4",
				Status:    lcmv1alpha1.RemoveFailed,
				StartedAt: "current-time-4",
				Error:     "failed to get job info: Retries (5/5) exceeded: failed to get-jobs",
			},
		},
		{
			name:           "job exists - initialized and not started",
			taskConfig:     taskConfigForTest,
			osd:            "4",
			host:           "node-2",
			removeInfo:     statusForCreatedJob.DeepCopy(),
			batchJob:       unitinputs.GetCleanupJobOnlyStatus("device-cleanup-job-node-2-4", "lcm-namespace", 0, 0, 0),
			expectedResult: statusForCreatedJob.CleanupMap["node-2"].OsdMapping["4"].RemoveStatus.DeviceCleanUpJob,
		},
		{
			name:       "job exists - completed",
			taskConfig: taskConfigForTest,
			osd:        "4",
			host:       "node-2",
			removeInfo: statusForCreatedJob.DeepCopy(),
			batchJob:   unitinputs.GetCleanupJobOnlyStatus("device-cleanup-job-node-2-4", "lcm-namespace", 0, 0, 1),
			expectedResult: &lcmv1alpha1.RemoveStatus{
				Name:       "device-cleanup-job-node-2-4",
				Status:     lcmv1alpha1.RemoveCompleted,
				StartedAt:  "current-time-4",
				FinishedAt: "current-time-7",
			},
		},
		{
			name:       "job exists - failed",
			taskConfig: taskConfigForTest,
			osd:        "4",
			host:       "node-2",
			removeInfo: statusForCreatedJob.DeepCopy(),
			batchJob:   unitinputs.GetCleanupJobOnlyStatus("device-cleanup-job-node-2-4", "lcm-namespace", 0, 1, 0),
			expectedResult: &lcmv1alpha1.RemoveStatus{
				Name:      "device-cleanup-job-node-2-4",
				Status:    lcmv1alpha1.RemoveFailed,
				StartedAt: "current-time-4",
				Error:     "job failed, check logs",
			},
		},
		{
			name:           "job exists - in progress",
			taskConfig:     taskConfigForTest,
			osd:            "4",
			host:           "node-2",
			removeInfo:     statusForCreatedJob.DeepCopy(),
			batchJob:       unitinputs.GetCleanupJobOnlyStatus("device-cleanup-job-node-2-4", "lcm-namespace", 1, 0, 0),
			expectedResult: statusForCreatedJob.CleanupMap["node-2"].OsdMapping["4"].RemoveStatus.DeviceCleanUpJob,
		},
		{
			name:       "job keeps delayed",
			taskConfig: taskConfigForTest,
			osd:        "4",
			host:       "node-2",
			removeInfo: unitinputs.GetInfoWithStatus(unitinputs.FullNodesRemoveMap, map[string]*lcmv1alpha1.RemoveResult{
				"*": nil,
				"4": {DeviceCleanUpJob: &lcmv1alpha1.RemoveStatus{Status: lcmv1alpha1.RemovePending}},
				"5": {DeviceCleanUpJob: &lcmv1alpha1.RemoveStatus{Status: lcmv1alpha1.RemoveInProgress}},
			}),
			expectedResult: &lcmv1alpha1.RemoveStatus{Status: lcmv1alpha1.RemovePending},
		},
		{
			name:       "job created after delay",
			taskConfig: taskConfigForTest,
			osd:        "4",
			host:       "node-2",
			removeInfo: unitinputs.GetInfoWithStatus(unitinputs.FullNodesRemoveMap, map[string]*lcmv1alpha1.RemoveResult{
				"*": nil,
				"4": {DeviceCleanUpJob: &lcmv1alpha1.RemoveStatus{Status: lcmv1alpha1.RemovePending}},
				"5": {DeviceCleanUpJob: &lcmv1alpha1.RemoveStatus{Status: lcmv1alpha1.RemoveCompleted}},
			}),
			expectedResult: &lcmv1alpha1.RemoveStatus{
				Name:      "device-cleanup-job-node-2-4",
				Status:    lcmv1alpha1.RemoveInProgress,
				StartedAt: "current-time-11",
			},
		},
	}
	oldRetryTimeout := commandRetryRunTimeout
	commandRetryRunTimeout = 0
	oldFunc := lcmcommon.GetCurrentTimeString
	for idx, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			lcmcommon.GetCurrentTimeString = func() string {
				return fmt.Sprintf("current-time-%d", idx)
			}

			c := fakeCephReconcileConfig(&test.taskConfig, nil)

			inputRes := map[string]runtime.Object{"jobs": &batch.JobList{}}
			if test.batchJob != nil {
				inputRes["jobs"] = &batch.JobList{Items: []batch.Job{*test.batchJob}}
			}
			apiErrors := map[string]error{}
			if test.apiError != "" {
				apiErrors[test.apiError] = errors.New("failed to " + test.apiError)
			}
			faketestclients.FakeReaction(c.api.Kubeclientset.BatchV1(), "get", []string{"jobs"}, inputRes, apiErrors)
			faketestclients.FakeReaction(c.api.Kubeclientset.BatchV1(), "create", []string{"jobs"}, inputRes, apiErrors)

			result := c.handleJobRun(test.osd, test.host, test.removeInfo.CleanupMap[test.host].OsdMapping)
			assert.Equal(t, test.expectedResult, result)
			faketestclients.CleanupFakeClientReactions(c.api.Kubeclientset.BatchV1())
		})
	}
	lcmcommon.GetCurrentTimeString = oldFunc
	commandRetryRunTimeout = oldRetryTimeout
}
