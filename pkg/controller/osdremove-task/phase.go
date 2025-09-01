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

	lcmv1alpha1 "github.com/Mirantis/pelagia/pkg/apis/ceph.pelagia.lcm/v1alpha1"
	lcmcommon "github.com/Mirantis/pelagia/pkg/common"
)

func prepareAbortStatus(cephTaskStatus *lcmv1alpha1.CephOsdRemoveTaskStatus, reason string) *lcmv1alpha1.CephOsdRemoveTaskStatus {
	newStatus := cephTaskStatus.DeepCopy()
	newStatus.Phase = lcmv1alpha1.TaskPhaseAborted
	newStatus.PhaseInfo = reason
	newStatus.Messages = append(newStatus.Messages, reason)
	newStatus.Conditions = append(cephTaskStatus.Conditions, lcmv1alpha1.CephOsdRemoveTaskCondition{
		Phase:     lcmv1alpha1.TaskPhaseAborted,
		Timestamp: lcmcommon.GetCurrentTimeString(),
	})
	return newStatus
}

func prepareInitStatus(cephTask *lcmv1alpha1.CephOsdRemoveTask) *lcmv1alpha1.CephOsdRemoveTaskStatus {
	status := &lcmv1alpha1.CephOsdRemoveTaskStatus{
		Phase:    lcmv1alpha1.TaskPhasePending,
		Messages: []string{"initiated"},
		Conditions: []lcmv1alpha1.CephOsdRemoveTaskCondition{
			{
				Phase:     lcmv1alpha1.TaskPhasePending,
				Timestamp: lcmcommon.GetCurrentTimeString(),
			},
		},
	}
	status.PhaseInfo = "initializing"
	if cephTask.Spec != nil && len(cephTask.Spec.Nodes) > 0 {
		status.Conditions[0].Nodes = cephTask.Spec.Nodes
	}
	return status
}

func (t taskConfig) moveTaskPhase(newPhase lcmv1alpha1.TaskPhase, reason string, removeInfo *lcmv1alpha1.TaskRemoveInfo) *lcmv1alpha1.CephOsdRemoveTaskStatus {
	newStatus := t.task.Status.DeepCopy()
	newStatus.Phase = newPhase
	newStatus.PhaseInfo = reason
	newStatus.Messages = append(newStatus.Messages, fmt.Sprintf("cephosdremovetask moved to '%s' phase: %s", newPhase, reason))
	newStatus.RemoveInfo = removeInfo
	newCondition := lcmv1alpha1.CephOsdRemoveTaskCondition{
		Phase:     newPhase,
		Timestamp: lcmcommon.GetCurrentTimeString(),
		CephClusterSpecVersion: &lcmv1alpha1.CephClusterSpecVersion{
			Generation:      t.cephCluster.Generation,
			ResourceVersion: t.cephCluster.ResourceVersion,
		},
	}
	if t.task.Spec != nil {
		newCondition.Nodes = t.task.Spec.Nodes
	}
	newStatus.Conditions = append(newStatus.Conditions, newCondition)
	return newStatus
}
