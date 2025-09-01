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
	"fmt"
	"testing"
	"time"

	"github.com/pkg/errors"
	cephv1 "github.com/rook/rook/pkg/apis/ceph.rook.io/v1"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"

	lcmv1alpha1 "github.com/Mirantis/pelagia/pkg/apis/ceph.pelagia.lcm/v1alpha1"
	lcmcommon "github.com/Mirantis/pelagia/pkg/common"
	faketestclients "github.com/Mirantis/pelagia/test/unit/clients"
	unitinputs "github.com/Mirantis/pelagia/test/unit/inputs"
)

func TestHandleTask(t *testing.T) {
	validationConfig := taskConfig{
		task:                  unitinputs.CephOsdRemoveTaskOnValidation.DeepCopy(),
		cephCluster:           &unitinputs.ReefCephClusterReady,
		cephHealthOsdAnalysis: unitinputs.OsdSpecAnalysisOk,
	}
	nodesListLabeledAvailable := unitinputs.GetNodesList(
		[]unitinputs.NodeAttrs{{Name: "node-1", Labeled: true}, {Name: "node-2", Labeled: true}})

	tests := []struct {
		name           string
		taskConfig     taskConfig
		cmdOutputs     map[string]string
		nodesList      *v1.NodeList
		deploymentList *appsv1.DeploymentList
		nodeOsdsReport map[string]*lcmcommon.DiskDaemonReport
		requeueNow     bool
		expectedStatus *lcmv1alpha1.CephOsdRemoveTaskStatus
	}{
		{
			name: "incorrect task status conditions",
			taskConfig: taskConfig{
				task: func() *lcmv1alpha1.CephOsdRemoveTask {
					task := unitinputs.CephOsdRemoveTaskFullInited.DeepCopy()
					task.Status.Conditions = nil
					return task
				}(),
				cephCluster: &unitinputs.ReefCephClusterReady,
			},
			expectedStatus: &lcmv1alpha1.CephOsdRemoveTaskStatus{
				Phase:     lcmv1alpha1.TaskPhaseAborted,
				PhaseInfo: "status conditions section unexpectedly missed, task should be re-created",
				Messages: []string{
					"initiated",
					"status conditions section unexpectedly missed, task should be re-created",
				},
				Conditions: []lcmv1alpha1.CephOsdRemoveTaskCondition{
					{
						Phase:     lcmv1alpha1.TaskPhaseAborted,
						Timestamp: "time-0",
					},
				},
			},
		},
		{
			name: "move task to validating state",
			taskConfig: taskConfig{
				task:        unitinputs.CephOsdRemoveTaskFullInited.DeepCopy(),
				cephCluster: &unitinputs.ReefCephClusterReady,
			},
			requeueNow:     true,
			expectedStatus: unitinputs.CephOsdRemoveTaskOnValidation.Status,
		},
		{
			name:       "task validation failed",
			taskConfig: validationConfig,
			expectedStatus: func() *lcmv1alpha1.CephOsdRemoveTaskStatus {
				status := unitinputs.CephOsdRemoveTaskOnValidation.Status.DeepCopy()
				status.Phase = lcmv1alpha1.TaskPhaseValidationFailed
				status.PhaseInfo = "validation failed"
				status.Messages = append(status.Messages, "cephosdremovetask moved to 'ValidationFailed' phase: validation failed")
				status.Conditions = append(status.Conditions, lcmv1alpha1.CephOsdRemoveTaskCondition{
					Phase:     lcmv1alpha1.TaskPhaseValidationFailed,
					Timestamp: "time-2",
					CephClusterSpecVersion: &lcmv1alpha1.CephClusterSpecVersion{
						Generation: 4,
					},
				})
				status.RemoveInfo = &lcmv1alpha1.TaskRemoveInfo{
					Issues: []string{
						"failed to get ceph cluster nodes list: failed to run command 'ceph osd tree -f json': command failed",
					},
				}
				return status
			}(),
		},
		{
			name: "task validation postponed, waiting for spec analyse",
			taskConfig: taskConfig{
				task:                  unitinputs.CephOsdRemoveTaskOnValidation.DeepCopy(),
				cephCluster:           &unitinputs.ReefCephClusterReady,
				cephHealthOsdAnalysis: &lcmv1alpha1.OsdSpecAnalysisState{CephClusterSpecGeneration: &[]int64{3}[0]},
			},
			expectedStatus: unitinputs.CephOsdRemoveTaskOnValidation.Status,
		},
		{
			name:       "task validation completed and nothing to remove",
			taskConfig: validationConfig,
			cmdOutputs: map[string]string{
				"ceph osd tree -f json":     unitinputs.CephOsdTreeOutput,
				"ceph osd info -f json":     unitinputs.CephOsdInfoOutputNoStray,
				"ceph osd metadata -f json": unitinputs.CephOsdMetadataOutputNoStray,
			},
			nodesList: &nodesListLabeledAvailable,
			nodeOsdsReport: map[string]*lcmcommon.DiskDaemonReport{
				"node-1": &unitinputs.DiskDaemonReportOkNode1,
				"node-2": &unitinputs.DiskDaemonReportOkNode2,
			},
			expectedStatus: func() *lcmv1alpha1.CephOsdRemoveTaskStatus {
				status := unitinputs.CephOsdRemoveTaskOnValidation.Status.DeepCopy()
				status.Phase = lcmv1alpha1.TaskPhaseCompleted
				status.PhaseInfo = "validation completed, nothing to remove"
				status.Messages = append(status.Messages, "cephosdremovetask moved to 'Completed' phase: validation completed, nothing to remove")
				status.RemoveInfo = unitinputs.EmptyRemoveMap
				status.Conditions = append(status.Conditions, lcmv1alpha1.CephOsdRemoveTaskCondition{
					Phase:     lcmv1alpha1.TaskPhaseCompleted,
					Timestamp: "time-4",
					CephClusterSpecVersion: &lcmv1alpha1.CephClusterSpecVersion{
						Generation: 4,
					},
				})
				return status
			}(),
		},
		{
			name:       "task validation completed and moved to approve waiting",
			taskConfig: validationConfig,
			cmdOutputs: map[string]string{
				"ceph osd tree -f json":     unitinputs.CephOsdTreeOutput,
				"ceph osd info -f json":     unitinputs.CephOsdInfoOutput,
				"ceph osd metadata -f json": unitinputs.CephOsdMetadataOutput,
			},
			nodesList: &nodesListLabeledAvailable,
			nodeOsdsReport: map[string]*lcmcommon.DiskDaemonReport{
				"node-1": &unitinputs.DiskDaemonReportOkNode1,
				"node-2": &unitinputs.DiskDaemonReportOkNode2,
			},
			expectedStatus: unitinputs.CephOsdRemoveTaskOnApproveWaiting.Status,
		},
		{
			name: "task validation completed and pre-approved",
			taskConfig: taskConfig{
				task: func() *lcmv1alpha1.CephOsdRemoveTask {
					removeTask := unitinputs.CephOsdRemoveTaskOnValidation.DeepCopy()
					removeTask.Spec = &lcmv1alpha1.CephOsdRemoveTaskSpec{Approve: true}
					return removeTask
				}(),
				cephCluster:           &unitinputs.ReefCephClusterReady,
				cephHealthOsdAnalysis: unitinputs.OsdSpecAnalysisOk,
			},
			cmdOutputs: map[string]string{
				"ceph osd tree -f json":     unitinputs.CephOsdTreeOutput,
				"ceph osd info -f json":     unitinputs.CephOsdInfoOutput,
				"ceph osd metadata -f json": unitinputs.CephOsdMetadataOutput,
			},
			nodesList: &nodesListLabeledAvailable,
			nodeOsdsReport: map[string]*lcmcommon.DiskDaemonReport{
				"node-1": &unitinputs.DiskDaemonReportOkNode1,
				"node-2": &unitinputs.DiskDaemonReportOkNode2,
			},
			expectedStatus: func() *lcmv1alpha1.CephOsdRemoveTaskStatus {
				status := unitinputs.CephOsdRemoveTaskOnValidation.Status.DeepCopy()
				status.RemoveInfo = unitinputs.CephOsdRemoveTaskOnApproveWaiting.Status.RemoveInfo
				status.Phase = lcmv1alpha1.TaskPhaseWaitingOperator
				status.Messages = append(status.Messages, "cephosdremovetask moved to 'WaitingOperator' phase: validation completed, approve pre-set")
				status.Conditions = append(status.Conditions, lcmv1alpha1.CephOsdRemoveTaskCondition{
					Phase:     lcmv1alpha1.TaskPhaseWaitingOperator,
					Timestamp: "time-6",
					CephClusterSpecVersion: &lcmv1alpha1.CephClusterSpecVersion{
						Generation: 4,
					},
				})
				status.PhaseInfo = "validation completed, approve pre-set"
				return status
			}(),
			requeueNow: true,
		},
		{
			name: "task waiting for approve",
			taskConfig: taskConfig{
				task:                  unitinputs.CephOsdRemoveTaskOnApproveWaiting.DeepCopy(),
				cephCluster:           &unitinputs.ReefCephClusterReady,
				cephHealthOsdAnalysis: unitinputs.OsdSpecAnalysisOk,
			},
			expectedStatus: unitinputs.CephOsdRemoveTaskOnApproveWaiting.Status,
		},
		{
			name: "task move to validating after changes in cephCluster and task",
			taskConfig: taskConfig{
				task: func() *lcmv1alpha1.CephOsdRemoveTask {
					taskNew := unitinputs.CephOsdRemoveTaskOnApproveWaiting.DeepCopy()
					taskNew.Spec = &lcmv1alpha1.CephOsdRemoveTaskSpec{
						Nodes: map[string]lcmv1alpha1.NodeCleanUpSpec{
							"test-node": {CompleteCleanup: true},
						},
					}
					return taskNew
				}(),
				cephCluster: func() *cephv1.CephCluster {
					cluster := unitinputs.ReefCephClusterReady.DeepCopy()
					cluster.Generation = 10
					return cluster
				}(),
				cephHealthOsdAnalysis: unitinputs.OsdSpecAnalysisOk,
			},
			requeueNow: true,
			expectedStatus: func() *lcmv1alpha1.CephOsdRemoveTaskStatus {
				status := unitinputs.CephOsdRemoveTaskOnApproveWaiting.Status.DeepCopy()
				status.Phase = lcmv1alpha1.TaskPhaseValidating
				status.PhaseInfo = "revalidation triggered"
				status.Messages = append(status.Messages, "cephosdremovetask moved to 'Validating' phase: revalidation triggered")
				status.RemoveInfo = nil
				status.Conditions = append(status.Conditions, lcmv1alpha1.CephOsdRemoveTaskCondition{
					Phase:     lcmv1alpha1.TaskPhaseValidating,
					Timestamp: "time-8",
					CephClusterSpecVersion: &lcmv1alpha1.CephClusterSpecVersion{
						Generation: 10,
					},
					Nodes: map[string]lcmv1alpha1.NodeCleanUpSpec{
						"test-node": {CompleteCleanup: true},
					},
				})
				return status
			}(),
		},
		{
			name: "task got approved moving to waiting operator",
			taskConfig: taskConfig{
				task: func() *lcmv1alpha1.CephOsdRemoveTask {
					taskNew := unitinputs.CephOsdRemoveTaskOnApproveWaiting.DeepCopy()
					taskNew.Spec = &lcmv1alpha1.CephOsdRemoveTaskSpec{Approve: true}
					return taskNew
				}(),
				cephCluster:           &unitinputs.ReefCephClusterReady,
				cephHealthOsdAnalysis: unitinputs.OsdSpecAnalysisOk,
			},
			expectedStatus: unitinputs.CephOsdRemoveTaskOnApproved.Status,
			requeueNow:     true,
		},
		{
			name: "move task to aborted state due to spec updates after approve",
			taskConfig: taskConfig{
				task: func() *lcmv1alpha1.CephOsdRemoveTask {
					taskNew := unitinputs.CephOsdRemoveTaskOnApproved.DeepCopy()
					taskNew.Spec = &lcmv1alpha1.CephOsdRemoveTaskSpec{
						Nodes: map[string]lcmv1alpha1.NodeCleanUpSpec{
							"test-node": {CompleteCleanup: true},
						},
					}
					return taskNew
				}(),
				cephCluster: func() *cephv1.CephCluster {
					cluster := unitinputs.ReefCephClusterReady.DeepCopy()
					cluster.Generation = 10
					return cluster
				}(),
				cephHealthOsdAnalysis: unitinputs.OsdSpecAnalysisOk,
			},
			expectedStatus: func() *lcmv1alpha1.CephOsdRemoveTaskStatus {
				status := unitinputs.CephOsdRemoveTaskOnApproved.Status.DeepCopy()
				status.Phase = lcmv1alpha1.TaskPhaseAborted
				status.PhaseInfo = "detected inappropriate spec changes after receiving approval"
				status.Messages = append(status.Messages, "cephosdremovetask moved to 'Aborted' phase: detected inappropriate spec changes after receiving approval")
				status.RemoveInfo = nil
				status.Conditions = append(status.Conditions, lcmv1alpha1.CephOsdRemoveTaskCondition{
					Phase:     lcmv1alpha1.TaskPhaseAborted,
					Timestamp: "time-10",
					CephClusterSpecVersion: &lcmv1alpha1.CephClusterSpecVersion{
						Generation: 10,
					},
					Nodes: map[string]lcmv1alpha1.NodeCleanUpSpec{
						"test-node": {CompleteCleanup: true},
					},
				})
				return status
			}(),
		},
		{
			name: "waiting rook operator stooped before processing",
			taskConfig: taskConfig{
				task:                  unitinputs.CephOsdRemoveTaskOnApproved.DeepCopy(),
				cephCluster:           &unitinputs.ReefCephClusterReady,
				cephHealthOsdAnalysis: unitinputs.OsdSpecAnalysisOk,
			},
			deploymentList: unitinputs.DeploymentList,
			expectedStatus: unitinputs.CephOsdRemoveTaskOnApproved.Status,
		},
		{
			name: "move task to processing state failed",
			taskConfig: taskConfig{
				task:                  unitinputs.CephOsdRemoveTaskOnApproved.DeepCopy(),
				cephCluster:           &unitinputs.ReefCephClusterReady,
				cephHealthOsdAnalysis: unitinputs.OsdSpecAnalysisOk,
			},
			deploymentList: &appsv1.DeploymentList{},
			expectedStatus: unitinputs.CephOsdRemoveTaskOnApproved.Status,
		},
		{
			name: "move task to processing state",
			taskConfig: taskConfig{
				task:                  unitinputs.CephOsdRemoveTaskOnApproved.DeepCopy(),
				cephCluster:           &unitinputs.ReefCephClusterReady,
				cephHealthOsdAnalysis: unitinputs.OsdSpecAnalysisOk,
			},
			deploymentList: &appsv1.DeploymentList{Items: []appsv1.Deployment{*unitinputs.RookDeploymentNotScaled}},
			expectedStatus: unitinputs.CephOsdRemoveTaskProcessing.Status,
			requeueNow:     true,
		},
		{
			name: "task failed and finished",
			taskConfig: taskConfig{
				task: func() *lcmv1alpha1.CephOsdRemoveTask {
					newTask := unitinputs.CephOsdRemoveTaskProcessing.DeepCopy()
					newTask.Status.RemoveInfo = unitinputs.GetInfoWithStatus(unitinputs.StrayOnlyInCrushRemoveMap,
						map[string]*lcmv1alpha1.RemoveResult{
							"2": {OsdRemoveStatus: &lcmv1alpha1.RemoveStatus{Status: lcmv1alpha1.RemoveFailed}},
						},
					)
					return newTask
				}(),
				cephCluster:           &unitinputs.ReefCephClusterReady,
				cephHealthOsdAnalysis: unitinputs.OsdSpecAnalysisOk,
			},
			expectedStatus: func() *lcmv1alpha1.CephOsdRemoveTaskStatus {
				status := unitinputs.CephOsdRemoveTaskProcessing.Status.DeepCopy()
				status.Phase = lcmv1alpha1.TaskPhaseFailed
				status.PhaseInfo = "osd remove failed"
				status.Messages = append(status.Messages, "cephosdremovetask moved to 'Failed' phase: osd remove failed")
				status.RemoveInfo = unitinputs.GetInfoWithStatus(unitinputs.StrayOnlyInCrushRemoveMap,
					map[string]*lcmv1alpha1.RemoveResult{
						"2": {OsdRemoveStatus: &lcmv1alpha1.RemoveStatus{Status: lcmv1alpha1.RemoveFailed}},
					},
				)
				status.RemoveInfo.Issues = []string{"[node '__stray'] failed to remove osd '2'"}
				status.Conditions = append(status.Conditions, lcmv1alpha1.CephOsdRemoveTaskCondition{
					Phase:     lcmv1alpha1.TaskPhaseFailed,
					Timestamp: "time-14",
					CephClusterSpecVersion: &lcmv1alpha1.CephClusterSpecVersion{
						Generation: 4,
					},
				})
				return status
			}(),
			requeueNow: true,
		},
		{
			name: "waiting task finished",
			taskConfig: taskConfig{
				task:                  unitinputs.CephOsdRemoveTaskProcessing.DeepCopy(),
				cephCluster:           &unitinputs.ReefCephClusterReady,
				cephHealthOsdAnalysis: unitinputs.OsdSpecAnalysisOk,
			},
			cmdOutputs: map[string]string{
				"ceph osd purge 2 --force --yes-i-really-mean-it": "",
				"ceph auth del osd.2":                             "",
			},
			expectedStatus: func() *lcmv1alpha1.CephOsdRemoveTaskStatus {
				status := unitinputs.CephOsdRemoveTaskProcessing.Status.DeepCopy()
				status.RemoveInfo = unitinputs.GetInfoWithStatus(unitinputs.StrayOnlyInCrushRemoveMap,
					map[string]*lcmv1alpha1.RemoveResult{
						"2": {
							OsdRemoveStatus: &lcmv1alpha1.RemoveStatus{Status: lcmv1alpha1.RemoveFinished, FinishedAt: "time-15"},
						},
					})
				return status
			}(),
			requeueNow: true,
		},
		{
			name: "task finished with warnings",
			taskConfig: taskConfig{
				task: func() *lcmv1alpha1.CephOsdRemoveTask {
					newTask := unitinputs.CephOsdRemoveTaskProcessing.DeepCopy()
					newTask.Status.RemoveInfo = unitinputs.GetInfoWithStatus(unitinputs.StrayOnlyInCrushRemoveMap,
						map[string]*lcmv1alpha1.RemoveResult{
							"2": {
								OsdRemoveStatus:    &lcmv1alpha1.RemoveStatus{Status: lcmv1alpha1.RemoveFinished, FinishedAt: "time-13"},
								DeviceCleanUpJob:   &lcmv1alpha1.RemoveStatus{Status: lcmv1alpha1.RemoveSkipped},
								DeployRemoveStatus: &lcmv1alpha1.RemoveStatus{Status: lcmv1alpha1.RemoveFinished},
							},
						})
					return newTask
				}(),
				cephCluster:           &unitinputs.ReefCephClusterReady,
				cephHealthOsdAnalysis: unitinputs.OsdSpecAnalysisOk,
			},
			expectedStatus: unitinputs.CephOsdRemoveTaskCompletedWithWarnings.Status,
			requeueNow:     true,
		},
		{
			name: "task finished",
			taskConfig: taskConfig{
				task: func() *lcmv1alpha1.CephOsdRemoveTask {
					newTask := unitinputs.CephOsdRemoveTaskProcessing.DeepCopy()
					newTask.Status.RemoveInfo = unitinputs.NodesRemoveFullFinishedStatus.DeepCopy()
					newTask.Status.RemoveInfo.Warnings = nil
					return newTask
				}(),
				cephCluster:           &unitinputs.ReefCephClusterReady,
				cephHealthOsdAnalysis: unitinputs.OsdSpecAnalysisOk,
			},
			expectedStatus: unitinputs.CephOsdRemoveTaskCompleted.Status,
			requeueNow:     true,
		},
	}

	oldTimeFunc := lcmcommon.GetCurrentTimeString
	oldRunCmd := lcmcommon.RunPodCommandWithValidation
	oldRetries := retriesForFailedCommand
	retriesForFailedCommand = 1
	for idx, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := fakeCephReconcileConfig(&test.taskConfig, nil)
			inputResources := map[string]runtime.Object{}
			if test.nodesList != nil {
				inputResources["nodes"] = test.nodesList
			}
			if test.deploymentList != nil {
				inputResources["deployments"] = test.deploymentList
			}
			faketestclients.FakeReaction(c.api.Kubeclientset.CoreV1(), "list", []string{"nodes"}, inputResources, nil)
			faketestclients.FakeReaction(c.api.Kubeclientset.AppsV1(), "get", []string{"deployments"}, inputResources, nil)

			lcmcommon.RunPodCommandWithValidation = func(e lcmcommon.ExecConfig) (string, string, error) {
				if e.Command == "pelagia-disk-daemon --osd-report --port 9999" {
					if report, present := test.nodeOsdsReport[e.Nodename]; present {
						output, _ := json.Marshal(report)
						return string(output), "", nil
					}
					return "{||}", "", nil
				} else if res, ok := test.cmdOutputs[e.Command]; ok {
					return res, "", nil
				}
				return "", "", errors.New("command failed")
			}

			lcmcommon.GetCurrentTimeString = func() string {
				return fmt.Sprintf("time-%d", idx)
			}

			newStatus := c.handleTask()
			assert.Equal(t, test.expectedStatus, newStatus)
			assert.Equal(t, test.requeueNow, c.taskConfig.requeueNow)
			faketestclients.CleanupFakeClientReactions(c.api.Kubeclientset.CoreV1())
			faketestclients.CleanupFakeClientReactions(c.api.Kubeclientset.AppsV1())
		})
	}
	lcmcommon.GetCurrentTimeString = oldTimeFunc
	lcmcommon.RunPodCommandWithValidation = oldRunCmd
	retriesForFailedCommand = oldRetries
}

func TestValidateRequest(t *testing.T) {
	baseTaskConfig := taskConfig{
		task:        unitinputs.CephOsdRemoveTaskOnValidation,
		cephCluster: &unitinputs.ReefCephClusterReady,
		cephHealthOsdAnalysis: &lcmv1alpha1.OsdSpecAnalysisState{
			SpecAnalysis: unitinputs.OsdStorageSpecAnalysisOk,
		},
	}
	nodesListLabeledAvailable := unitinputs.GetNodesList(
		[]unitinputs.NodeAttrs{{Name: "node-1", Labeled: true}, {Name: "node-2", Labeled: true}})
	tests := []struct {
		name               string
		taskConfig         taskConfig
		expectedRemoveInfo *lcmv1alpha1.TaskRemoveInfo
		osdTreeFromCluster string
		osdMetadata        string
		osdInfo            string
		nodeOsdsReport     map[string]*lcmcommon.DiskDaemonReport
		nodeList           *v1.NodeList
	}{
		{
			name:       "fail to get cluster hosts",
			taskConfig: baseTaskConfig,
			expectedRemoveInfo: &lcmv1alpha1.TaskRemoveInfo{
				Issues: []string{"failed to get ceph cluster nodes list: failed to run command 'ceph osd tree -f json': command failed"},
			},
		},
		{
			name:               "fail to get osd metadata info",
			taskConfig:         baseTaskConfig,
			osdTreeFromCluster: unitinputs.CephOsdTreeOutput,
			expectedRemoveInfo: &lcmv1alpha1.TaskRemoveInfo{
				Issues: []string{"failed to get ceph osd metadata info: failed to run command 'ceph osd metadata -f json': command failed"},
			},
		},
		{
			name:               "fail to get node lists",
			taskConfig:         baseTaskConfig,
			osdTreeFromCluster: unitinputs.CephOsdTreeOutput,
			osdMetadata:        unitinputs.CephOsdMetadataOutput,
			expectedRemoveInfo: &lcmv1alpha1.TaskRemoveInfo{
				Issues: []string{"failed to get k8s nodes list: failed to list nodes"},
			},
		},
		{
			name:               "fail to get osd info",
			taskConfig:         baseTaskConfig,
			osdTreeFromCluster: unitinputs.CephOsdTreeOutput,
			osdMetadata:        unitinputs.CephOsdMetadataOutput,
			nodeList:           &nodesListLabeledAvailable,
			expectedRemoveInfo: &lcmv1alpha1.TaskRemoveInfo{
				Issues: []string{"failed to get osds info: failed to run command 'ceph osd info -f json': command failed"},
			},
		},
		{
			name: "validation failed",
			taskConfig: func() taskConfig {
				newConf := baseTaskConfig
				newConf.task = unitinputs.CephOsdRemoveTaskOnValidation.DeepCopy()
				newConf.task.Spec = &lcmv1alpha1.CephOsdRemoveTaskSpec{
					Nodes: unitinputs.RequestRemoveByOsdID,
				}
				return newConf
			}(),
			osdTreeFromCluster: unitinputs.CephOsdTreeOutput,
			osdMetadata:        unitinputs.CephOsdMetadataOutputNoStray,
			nodeList:           &nodesListLabeledAvailable,
			osdInfo:            unitinputs.CephOsdInfoOutputNoStray,
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
			name:               "validation completed and nothing to remove",
			taskConfig:         baseTaskConfig,
			osdTreeFromCluster: unitinputs.CephOsdTreeOutput,
			osdMetadata:        unitinputs.CephOsdMetadataOutputNoStray,
			nodeList:           &nodesListLabeledAvailable,
			osdInfo:            unitinputs.CephOsdInfoOutputNoStray,
			nodeOsdsReport: map[string]*lcmcommon.DiskDaemonReport{
				"node-1": &unitinputs.DiskDaemonReportOkNode1,
				"node-2": &unitinputs.DiskDaemonReportOkNode2,
			},
			expectedRemoveInfo: unitinputs.EmptyRemoveMap,
		},
		{
			name:               "validation completed and found to remove",
			taskConfig:         baseTaskConfig,
			osdTreeFromCluster: unitinputs.CephOsdTreeOutput,
			osdMetadata:        unitinputs.CephOsdMetadataOutput,
			nodeList:           &nodesListLabeledAvailable,
			osdInfo:            unitinputs.CephOsdInfoOutput,
			nodeOsdsReport: map[string]*lcmcommon.DiskDaemonReport{
				"node-1": &unitinputs.DiskDaemonReportOkNode1,
				"node-2": &unitinputs.DiskDaemonReportOkNode2,
			},
			expectedRemoveInfo: unitinputs.StrayOnlyInCrushRemoveMap,
		},
	}

	oldRunCmd := lcmcommon.RunPodCommand
	oldRetries := retriesForFailedCommand
	retriesForFailedCommand = 1
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := fakeCephReconcileConfig(&test.taskConfig, nil)
			inputResources := map[string]runtime.Object{"pods": unitinputs.ToolBoxAndDiskDaemonPodsList}
			if test.nodeList != nil {
				inputResources["nodes"] = test.nodeList
			}
			faketestclients.FakeReaction(c.api.Kubeclientset.CoreV1(), "list", []string{"nodes", "pods"}, inputResources, nil)

			lcmcommon.RunPodCommand = func(e lcmcommon.ExecConfig) (string, string, error) {
				switch e.Command {
				case "ceph osd tree -f json":
					if test.osdTreeFromCluster != "" {
						return test.osdTreeFromCluster, "", nil
					}
				case "ceph osd metadata -f json":
					if test.osdMetadata != "" {
						return test.osdMetadata, "", nil
					}
				case "ceph osd info -f json":
					if test.osdInfo != "" {
						return test.osdInfo, "", nil
					}
				default:
					if e.Command == "pelagia-disk-daemon --osd-report --port 9999" {
						if report, present := test.nodeOsdsReport[e.Nodename]; present {
							output, _ := json.Marshal(report)
							return string(output), "", nil
						}
						return "{||}", "", nil
					}
				}
				return "", "", errors.New("command failed")
			}

			result := c.validateTask()
			assert.Equal(t, test.expectedRemoveInfo, result)
			faketestclients.CleanupFakeClientReactions(c.api.Kubeclientset.CoreV1())
		})
	}
	lcmcommon.RunPodCommand = oldRunCmd
	retriesForFailedCommand = oldRetries
}

func TestProcessRequest(t *testing.T) {
	removeInfoFinished := unitinputs.NodesRemoveMapOsdFinishedStatus.DeepCopy()
	host1 := removeInfoFinished.CleanupMap["node-1"]
	host1.HostRemoveStatus = &lcmv1alpha1.RemoveStatus{
		Status:     lcmv1alpha1.RemoveFinished,
		StartedAt:  time.Now().Format(time.RFC3339),
		FinishedAt: time.Now().Format(time.RFC3339),
	}
	removeInfoFinished.CleanupMap["node-1"] = host1
	host2 := removeInfoFinished.CleanupMap["node-2"]
	host2.HostRemoveStatus = &lcmv1alpha1.RemoveStatus{
		Status:     lcmv1alpha1.RemoveFinished,
		StartedAt:  time.Now().Format(time.RFC3339),
		FinishedAt: time.Now().Format(time.RFC3339),
	}
	removeInfoFinished.CleanupMap["node-2"] = host2

	tests := []struct {
		name               string
		taskConfig         taskConfig
		finished           bool
		expectedRemoveInfo *lcmv1alpha1.TaskRemoveInfo
	}{
		{
			name: "empty status in request",
			taskConfig: taskConfig{
				task:        unitinputs.CephOsdRemoveTaskOnValidation,
				cephCluster: &unitinputs.ReefCephClusterReady,
			},
			finished:           true,
			expectedRemoveInfo: &lcmv1alpha1.TaskRemoveInfo{Issues: []string{"empty remove info, aborting"}},
		},
		{
			name: "processing is not finished",
			taskConfig: taskConfig{
				task:        unitinputs.GetTaskForRemove(unitinputs.CephOsdRemoveTaskProcessing, unitinputs.NodesRemoveMapOsdFinishedStatus.DeepCopy()),
				cephCluster: &unitinputs.ReefCephClusterReady,
			},
			expectedRemoveInfo: removeInfoFinished,
		},
		{
			name: "processing is finished",
			taskConfig: taskConfig{
				task:        unitinputs.GetTaskForRemove(unitinputs.CephOsdRemoveTaskProcessing, removeInfoFinished.DeepCopy()),
				cephCluster: &unitinputs.ReefCephClusterReady,
			},
			finished:           true,
			expectedRemoveInfo: removeInfoFinished,
		},
	}
	oldFunc := lcmcommon.RunPodCommandWithValidation
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := fakeCephReconcileConfig(&test.taskConfig, nil)

			lcmcommon.RunPodCommandWithValidation = func(_ lcmcommon.ExecConfig) (string, string, error) {
				return "", "", nil
			}

			finished, removeInfo := c.processTask()
			assert.Equal(t, test.expectedRemoveInfo, removeInfo)
			assert.Equal(t, test.finished, finished)
		})
	}
	lcmcommon.RunPodCommandWithValidation = oldFunc
}
