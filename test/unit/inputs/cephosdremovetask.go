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

package input

import (
	"fmt"
	"strconv"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	lcmv1alpha1 "github.com/Mirantis/pelagia/pkg/apis/ceph.pelagia.lcm/v1alpha1"
)

func getObjectMeta(resourceVersion string) metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Name:              "osdremove-task",
		Namespace:         LcmObjectMeta.Namespace,
		CreationTimestamp: metav1.Time{Time: time.Date(2025, 4, 7, 14, 30, 45, 0, time.Local)},
		ResourceVersion:   resourceVersion,
	}
}

var CephOsdRemoveTaskBase = lcmv1alpha1.CephOsdRemoveTask{
	ObjectMeta: getObjectMeta("0"),
	TypeMeta: metav1.TypeMeta{
		APIVersion: "lcm.mirantis.com/v1alpha1",
		Kind:       "CephOsdRemoveTask",
	},
}
var CephOsdRemoveTaskOld = lcmv1alpha1.CephOsdRemoveTask{
	ObjectMeta: metav1.ObjectMeta{
		Name:              "old-osdremove-task",
		Namespace:         LcmObjectMeta.Namespace,
		CreationTimestamp: metav1.Time{Time: time.Date(2025, 4, 6, 14, 30, 45, 0, time.Local)},
	},
}
var CephOsdRemoveTaskOldCompleted = lcmv1alpha1.CephOsdRemoveTask{
	ObjectMeta: metav1.ObjectMeta{
		Name:              "old-completed-osdremove-task",
		Namespace:         LcmObjectMeta.Namespace,
		CreationTimestamp: metav1.Time{Time: time.Date(2025, 4, 6, 13, 30, 45, 0, time.Local)},
	},
	Status: &lcmv1alpha1.CephOsdRemoveTaskStatus{
		Phase: lcmv1alpha1.TaskPhaseCompleted,
	},
}

func GetAbortedTask(taskToAbort lcmv1alpha1.CephOsdRemoveTask, time, reason string) *lcmv1alpha1.CephOsdRemoveTask {
	abortedTask := taskToAbort.DeepCopy()
	abortedTask.Status.Phase = lcmv1alpha1.TaskPhaseAborted
	abortedTask.Status.PhaseInfo = reason
	abortedTask.Status.Messages = append(abortedTask.Status.Messages, reason)
	abortedTask.Status.Conditions = append(abortedTask.Status.Conditions, lcmv1alpha1.CephOsdRemoveTaskCondition{
		Phase:     lcmv1alpha1.TaskPhaseAborted,
		Timestamp: time,
	})
	ver, _ := strconv.Atoi(taskToAbort.ResourceVersion)
	abortedTask.ResourceVersion = fmt.Sprintf("%d", ver+1)
	return abortedTask
}

var initMessages = []string{"initiated"}
var initConditions = []lcmv1alpha1.CephOsdRemoveTaskCondition{
	{
		Phase:     lcmv1alpha1.TaskPhasePending,
		Timestamp: "test-time-3",
	},
}

var CephOsdRemoveTaskInited = lcmv1alpha1.CephOsdRemoveTask{
	ObjectMeta: getObjectMeta("1"),
	TypeMeta: metav1.TypeMeta{
		APIVersion: "lcm.mirantis.com/v1alpha1",
		Kind:       "CephOsdRemoveTask",
	},
	Status: &lcmv1alpha1.CephOsdRemoveTaskStatus{
		Phase:      lcmv1alpha1.TaskPhasePending,
		PhaseInfo:  "initializing",
		Messages:   initMessages,
		Conditions: initConditions,
	},
}

var CephOsdRemoveTaskFullInited = lcmv1alpha1.CephOsdRemoveTask{
	ObjectMeta: metav1.ObjectMeta{
		Name:              "osdremove-task",
		Namespace:         LcmObjectMeta.Namespace,
		CreationTimestamp: metav1.Time{Time: time.Date(2025, 4, 7, 14, 30, 45, 0, time.Local)},
		ResourceVersion:   "1",
		OwnerReferences: []metav1.OwnerReference{
			{
				APIVersion: "lcm.mirantis.com/v1alpha1",
				Kind:       "CephDeploymentHealth",
				Name:       LcmObjectMeta.Name,
			},
		},
	},
	TypeMeta: metav1.TypeMeta{
		APIVersion: "lcm.mirantis.com/v1alpha1",
		Kind:       "CephOsdRemoveTask",
	},
	Status: &lcmv1alpha1.CephOsdRemoveTaskStatus{
		Phase:      lcmv1alpha1.TaskPhasePending,
		Messages:   initMessages,
		Conditions: initConditions,
	},
}

var CephOsdRemoveTaskOnValidation = func() *lcmv1alpha1.CephOsdRemoveTask {
	task := CephOsdRemoveTaskFullInited.DeepCopy()
	task.Status.Phase = lcmv1alpha1.TaskPhaseValidating
	task.Status.Messages = append(task.Status.Messages, "cephosdremovetask moved to 'Validating' phase: validation")
	task.Status.Conditions = append(task.Status.Conditions, lcmv1alpha1.CephOsdRemoveTaskCondition{
		Phase:     lcmv1alpha1.TaskPhaseValidating,
		Timestamp: "time-1",
		CephClusterSpecVersion: &lcmv1alpha1.CephClusterSpecVersion{
			Generation: 4,
		},
	})
	task.Status.PhaseInfo = "validation"
	return task
}()

var CephOsdRemoveTaskOnApproveWaiting = func() *lcmv1alpha1.CephOsdRemoveTask {
	task := CephOsdRemoveTaskOnValidation.DeepCopy()
	task.Status.Phase = lcmv1alpha1.TaskPhaseApproveWaiting
	task.Status.Messages = append(task.Status.Messages, "cephosdremovetask moved to 'ApproveWaiting' phase: validation completed, waiting approve")
	task.Status.Conditions = append(task.Status.Conditions, lcmv1alpha1.CephOsdRemoveTaskCondition{
		Phase:     lcmv1alpha1.TaskPhaseApproveWaiting,
		Timestamp: "time-5",
		CephClusterSpecVersion: &lcmv1alpha1.CephClusterSpecVersion{
			Generation: 4,
		},
	})
	task.Status.PhaseInfo = "validation completed, waiting approve"
	task.Status.RemoveInfo = StrayOnlyInCrushRemoveMap.DeepCopy()
	return task
}()

var CephOsdRemoveTaskOnApproved = func() *lcmv1alpha1.CephOsdRemoveTask {
	task := CephOsdRemoveTaskOnApproveWaiting.DeepCopy()
	task.Spec = &lcmv1alpha1.CephOsdRemoveTaskSpec{Approve: true}
	task.Status.Phase = lcmv1alpha1.TaskPhaseWaitingOperator
	task.Status.Messages = append(task.Status.Messages, "cephosdremovetask moved to 'WaitingOperator' phase: approve received, wait rook-operator stop")
	task.Status.Conditions = append(task.Status.Conditions, lcmv1alpha1.CephOsdRemoveTaskCondition{
		Phase:     lcmv1alpha1.TaskPhaseWaitingOperator,
		Timestamp: "time-9",
		CephClusterSpecVersion: &lcmv1alpha1.CephClusterSpecVersion{
			Generation: 4,
		},
	})
	task.Status.PhaseInfo = "approve received, wait rook-operator stop"
	return task
}()

var CephOsdRemoveTaskProcessing = func() *lcmv1alpha1.CephOsdRemoveTask {
	task := CephOsdRemoveTaskOnApproved.DeepCopy()
	task.Spec = &lcmv1alpha1.CephOsdRemoveTaskSpec{Approve: true}
	task.Status.Phase = lcmv1alpha1.TaskPhaseProcessing
	task.Status.Messages = append(task.Status.Messages, "cephosdremovetask moved to 'Processing' phase: processing")
	task.Status.Conditions = append(task.Status.Conditions, lcmv1alpha1.CephOsdRemoveTaskCondition{
		Phase:     lcmv1alpha1.TaskPhaseProcessing,
		Timestamp: "time-13",
		CephClusterSpecVersion: &lcmv1alpha1.CephClusterSpecVersion{
			Generation: 4,
		},
	})
	task.Status.PhaseInfo = "processing"
	return task
}()

var CephOsdRemoveTaskCompletedWithWarnings = func() *lcmv1alpha1.CephOsdRemoveTask {
	task := CephOsdRemoveTaskProcessing.DeepCopy()
	task.Spec = &lcmv1alpha1.CephOsdRemoveTaskSpec{Approve: true}
	task.Status.Phase = lcmv1alpha1.TaskPhaseCompletedWithWarnings
	task.Status.Messages = append(task.Status.Messages, "cephosdremovetask moved to 'CompletedWithWarnings' phase: osd remove completed")
	task.Status.Conditions = append(task.Status.Conditions, lcmv1alpha1.CephOsdRemoveTaskCondition{
		Phase:     lcmv1alpha1.TaskPhaseCompletedWithWarnings,
		Timestamp: "time-16",
		CephClusterSpecVersion: &lcmv1alpha1.CephClusterSpecVersion{
			Generation: 4,
		},
	})
	task.Status.PhaseInfo = "osd remove completed"
	task.Status.RemoveInfo = GetInfoWithStatus(StrayOnlyInCrushRemoveMap,
		map[string]*lcmv1alpha1.RemoveResult{
			"2": {
				OsdRemoveStatus:    &lcmv1alpha1.RemoveStatus{Status: lcmv1alpha1.RemoveFinished, FinishedAt: "time-13"},
				DeviceCleanUpJob:   &lcmv1alpha1.RemoveStatus{Status: lcmv1alpha1.RemoveSkipped},
				DeployRemoveStatus: &lcmv1alpha1.RemoveStatus{Status: lcmv1alpha1.RemoveFinished},
			},
		})
	return task
}()

var CephOsdRemoveTaskCompleted = func() *lcmv1alpha1.CephOsdRemoveTask {
	task := CephOsdRemoveTaskProcessing.DeepCopy()
	task.Spec = &lcmv1alpha1.CephOsdRemoveTaskSpec{Approve: true}
	task.Status.Phase = lcmv1alpha1.TaskPhaseCompleted
	task.Status.Messages = append(task.Status.Messages, "cephosdremovetask moved to 'Completed' phase: osd remove completed")
	task.Status.Conditions = append(task.Status.Conditions, lcmv1alpha1.CephOsdRemoveTaskCondition{
		Phase:     lcmv1alpha1.TaskPhaseCompleted,
		Timestamp: "time-17",
		CephClusterSpecVersion: &lcmv1alpha1.CephClusterSpecVersion{
			Generation: 4,
		},
	})
	task.Status.PhaseInfo = "osd remove completed"
	task.Status.RemoveInfo = NodesRemoveFullFinishedStatus.DeepCopy()
	task.Status.RemoveInfo.Warnings = nil
	return task
}()

var CephOsdRemoveTaskFailed = func() *lcmv1alpha1.CephOsdRemoveTask {
	task := CephOsdRemoveTaskProcessing.DeepCopy()
	task.Spec = &lcmv1alpha1.CephOsdRemoveTaskSpec{Approve: true}
	task.Status = CephOsdRemoveTaskProcessing.Status.DeepCopy()
	task.Status.Phase = lcmv1alpha1.TaskPhaseFailed
	task.Status.PhaseInfo = "osd remove failed"
	task.Status.Messages = append(task.Status.Messages, "cephosdremovetask moved to 'Failed' phase: osd remove failed")
	task.Status.RemoveInfo = GetInfoWithStatus(StrayOnlyInCrushRemoveMap,
		map[string]*lcmv1alpha1.RemoveResult{
			"2": {OsdRemoveStatus: &lcmv1alpha1.RemoveStatus{Status: lcmv1alpha1.RemoveFailed}},
		},
	)
	task.Status.RemoveInfo.Issues = []string{"[node '__stray'] failed to remove osd '2'"}
	task.Status.Conditions = append(task.Status.Conditions, lcmv1alpha1.CephOsdRemoveTaskCondition{
		Phase:     lcmv1alpha1.TaskPhaseFailed,
		Timestamp: "time-18",
		CephClusterSpecVersion: &lcmv1alpha1.CephClusterSpecVersion{
			Generation: 4,
		},
	})
	return task
}()

var CephOsdRemoveTaskListEmpty = &lcmv1alpha1.CephOsdRemoveTaskList{}

func GetTaskList(tasks ...lcmv1alpha1.CephOsdRemoveTask) *lcmv1alpha1.CephOsdRemoveTaskList {
	newList := &lcmv1alpha1.CephOsdRemoveTaskList{}
	newList.Items = append(newList.Items, tasks...)
	return newList
}

func GetTaskForRemove(sourceTask *lcmv1alpha1.CephOsdRemoveTask, removeInfo *lcmv1alpha1.TaskRemoveInfo) *lcmv1alpha1.CephOsdRemoveTask {
	newTask := sourceTask.DeepCopy()
	newTask.Status.RemoveInfo = removeInfo
	return newTask
}

var RequestRemoveByOsdID = map[string]lcmv1alpha1.NodeCleanUpSpec{
	"node-1": {
		CleanupByOsd: []lcmv1alpha1.OsdCleanupSpec{
			{ID: 20}, {ID: 30},
		},
	},
	"node-2": {
		CleanupByOsd: []lcmv1alpha1.OsdCleanupSpec{
			{ID: 4}, {ID: 88},
		},
	},
}

var RequestRemoveByDevice = map[string]lcmv1alpha1.NodeCleanUpSpec{
	"node-1": {
		CleanupByDevice: []lcmv1alpha1.DeviceCleanupSpec{
			{
				Device: "/dev/disk/by-path/virtio-pci-0000:00:0f.0",
			},
		},
	},
	"node-2": {
		CleanupByDevice: []lcmv1alpha1.DeviceCleanupSpec{
			{
				Device: "vdd",
			},
		},
	},
}

var RequestRemoveFullNodeRemove = map[string]lcmv1alpha1.NodeCleanUpSpec{
	"node-1": {
		CompleteCleanup: true,
	},
	"node-2": {
		DropFromCrush: true,
	},
}
