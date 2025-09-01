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

package infra

import (
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	lcmv1alpha1 "github.com/Mirantis/pelagia/pkg/apis/ceph.pelagia.lcm/v1alpha1"
	faketestclients "github.com/Mirantis/pelagia/test/unit/clients"
	unitinputs "github.com/Mirantis/pelagia/test/unit/inputs"
)

func TestCheckRookOperatorReplicas(t *testing.T) {
	deployList := &appsv1.DeploymentList{
		Items: []appsv1.Deployment{*unitinputs.RookDeploymentLatestVersion.DeepCopy()},
	}
	rookDeploy := unitinputs.RookDeploymentNotScaled.DeepCopy()
	deployListWithScaleDown := &appsv1.DeploymentList{
		Items: []appsv1.Deployment{*rookDeploy},
	}
	failedTask := unitinputs.CephOsdRemoveTaskProcessing.DeepCopy()
	failedTask.Status.Phase = lcmv1alpha1.TaskPhaseFailed
	failedTaskResolved := failedTask.DeepCopy()
	failedTaskResolved.Spec = &lcmv1alpha1.CephOsdRemoveTaskSpec{Resolved: true}
	var0 := int32(0)
	var1 := int32(1)

	tests := []struct {
		name           string
		infraConfig    infraConfig
		inputResources map[string]runtime.Object
		replicas       *int32
		apiErrors      map[string]error
		expectedError  string
	}{
		{
			name: "failed to get rook operator deployment",
			inputResources: map[string]runtime.Object{
				"deployments": unitinputs.DeploymentListEmpty,
			},
			expectedError: "failed to check replicas for rook operator 'rook-ceph/rook-ceph-operator': deployments \"rook-ceph-operator\" not found",
		},
		{
			name:        "check replicas for external, nothing to do",
			infraConfig: infraConfig{externalCeph: true},
			inputResources: map[string]runtime.Object{
				"deployments":                deployList,
				"cephdeploymentmaintenances": unitinputs.CephDeploymentMaintenanceListIdle,
			},
			replicas:  &var1,
			apiErrors: map[string]error{"update-deployments": errors.New("unexpected update")},
		},
		{
			name:        "check replicas for external, scale down maintenance failed",
			infraConfig: infraConfig{externalCeph: true},
			inputResources: map[string]runtime.Object{
				"deployments": deployList,
				"cephdeploymentmaintenances": &lcmv1alpha1.CephDeploymentMaintenanceList{
					Items: []lcmv1alpha1.CephDeploymentMaintenance{unitinputs.CephDeploymentMaintenanceActing}},
			},
			replicas:      &var1,
			apiErrors:     map[string]error{"update-deployments-rook-ceph-operator": errors.New("failed to scale")},
			expectedError: "failed to scale rook operator 'rook-ceph/rook-ceph-operator': failed to scale",
		},
		{
			name: "failed to check cephosdremovetasks",
			inputResources: map[string]runtime.Object{
				"deployments":                deployList,
				"cephdeploymentmaintenances": unitinputs.CephDeploymentMaintenanceListIdle,
			},
			expectedError: "failed to check CephOsdRemoveTasks: failed to list cephosdremovetasks",
		},
		{
			name: "check replicas, upscale back failed",
			inputResources: map[string]runtime.Object{
				"deployments":                deployListWithScaleDown,
				"cephosdremovetasks":         unitinputs.CephOsdRemoveTaskListEmpty,
				"cephdeploymentmaintenances": unitinputs.CephDeploymentMaintenanceListIdle,
			},
			replicas:      &var0,
			apiErrors:     map[string]error{"update-deployments-rook-ceph-operator": errors.New("failed to scale")},
			expectedError: "failed to scale rook operator 'rook-ceph/rook-ceph-operator': failed to scale",
		},
		{
			name: "check replicas, nothing to do",
			inputResources: map[string]runtime.Object{
				"deployments":                deployList,
				"cephosdremovetasks":         unitinputs.CephOsdRemoveTaskListEmpty,
				"cephdeploymentmaintenances": unitinputs.CephDeploymentMaintenanceListIdle,
			},
			replicas: &var1,
		},
		{
			name: "check replicas, found waiting task, scale failed",
			inputResources: map[string]runtime.Object{
				"deployments":                deployList,
				"cephosdremovetasks":         unitinputs.GetTaskList(*unitinputs.CephOsdRemoveTaskOnApproved),
				"cephdeploymentmaintenances": unitinputs.CephDeploymentMaintenanceListIdle,
			},
			replicas:      &var1,
			apiErrors:     map[string]error{"update-deployments-rook-ceph-operator": errors.New("failed to scale")},
			expectedError: "failed to scale rook operator 'rook-ceph/rook-ceph-operator': failed to scale",
		},
		{
			name: "check replicas, found waiting task, scaledown",
			inputResources: map[string]runtime.Object{
				"deployments":                deployList.DeepCopy(),
				"cephosdremovetasks":         unitinputs.GetTaskList(*unitinputs.CephOsdRemoveTaskOnApproved),
				"cephdeploymentmaintenances": unitinputs.CephDeploymentMaintenanceListIdle,
			},
			replicas: &var0,
		},
		{
			name: "check replicas, found processing task, nothing to do",
			inputResources: map[string]runtime.Object{
				"deployments":                deployListWithScaleDown,
				"cephosdremovetasks":         unitinputs.GetTaskList(*unitinputs.CephOsdRemoveTaskProcessing),
				"cephdeploymentmaintenances": unitinputs.CephDeploymentMaintenanceListIdle,
			},
			replicas: &var0,
		},
		{
			name: "check replicas, found failed task, but not resolved, nothing to do",
			inputResources: map[string]runtime.Object{
				"deployments":                deployListWithScaleDown,
				"cephosdremovetasks":         unitinputs.GetTaskList(*failedTask),
				"cephdeploymentmaintenances": unitinputs.CephDeploymentMaintenanceListIdle,
			},
			replicas: &var0,
		},
		{
			name: "check replicas, found failed task, but resolved, upscale failed",
			inputResources: map[string]runtime.Object{
				"deployments":                deployListWithScaleDown,
				"cephosdremovetasks":         unitinputs.GetTaskList(*failedTaskResolved),
				"cephdeploymentmaintenances": unitinputs.CephDeploymentMaintenanceListIdle,
			},
			apiErrors:     map[string]error{"update-deployments-rook-ceph-operator": errors.New("failed to scale")},
			expectedError: "failed to scale rook operator 'rook-ceph/rook-ceph-operator': failed to scale",
			replicas:      &var0,
		},
		{
			name: "check replicas, found failed task, but resolved, upscale failed",
			inputResources: map[string]runtime.Object{
				"deployments":                deployListWithScaleDown.DeepCopy(),
				"cephosdremovetasks":         unitinputs.GetTaskList(*failedTaskResolved),
				"cephdeploymentmaintenances": unitinputs.CephDeploymentMaintenanceListIdle,
			},
			replicas: &var1,
		},
		{
			name: "check replicas, found tasks, nothing to do",
			inputResources: map[string]runtime.Object{
				"deployments":                deployList,
				"cephosdremovetasks":         unitinputs.GetTaskList(*failedTaskResolved, *unitinputs.CephOsdRemoveTaskOnApproveWaiting, unitinputs.CephOsdRemoveTaskBase),
				"cephdeploymentmaintenances": unitinputs.CephDeploymentMaintenanceListIdle,
			},
			replicas: &var1,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := fakeReconcileInfraConfig(&test.infraConfig, nil)
			faketestclients.FakeReaction(c.api.Lcmclientset, "list", []string{"cephosdremovetasks"}, test.inputResources, nil)
			faketestclients.FakeReaction(c.api.Lcmclientset, "get", []string{"cephdeploymentmaintenances"}, test.inputResources, nil)
			faketestclients.FakeReaction(c.api.Kubeclientset.AppsV1(), "get", []string{"deployments"}, test.inputResources, nil)
			faketestclients.FakeReaction(c.api.Kubeclientset.AppsV1(), "update", []string{"deployments"}, test.inputResources, test.apiErrors)

			err := c.checkRookOperatorReplicas()
			if test.expectedError != "" {
				assert.NotNil(t, err)
				assert.Equal(t, test.expectedError, err.Error())
			} else {
				assert.Nil(t, err)
			}

			if test.inputResources != nil && test.inputResources["deployments"] != nil && test.replicas != nil {
				rookOperator, err := c.api.Kubeclientset.AppsV1().Deployments(c.lcmConfig.RookNamespace).Get(c.context, "rook-ceph-operator", metav1.GetOptions{})
				assert.Nil(t, err)
				currentReplicas := rookOperator.Spec.Replicas
				if currentReplicas == nil {
					currentReplicas = &var1
				}
				assert.Equal(t, test.replicas, currentReplicas)
			}

			faketestclients.CleanupFakeClientReactions(c.api.Lcmclientset)
			faketestclients.CleanupFakeClientReactions(c.api.Kubeclientset.AppsV1())
		})
	}
}
