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
	"context"
	"fmt"
	"testing"

	"github.com/pkg/errors"
	cephv1 "github.com/rook/rook/pkg/apis/ceph.rook.io/v1"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	lcmv1alpha1 "github.com/Mirantis/pelagia/pkg/apis/ceph.pelagia.lcm/v1alpha1"
	lcmcommon "github.com/Mirantis/pelagia/pkg/common"
	lcmconfig "github.com/Mirantis/pelagia/pkg/controller/config"
	faketestclients "github.com/Mirantis/pelagia/test/unit/clients"
	unitinputs "github.com/Mirantis/pelagia/test/unit/inputs"
	faketestscheme "github.com/Mirantis/pelagia/test/unit/scheme"
)

func FakeReconciler() *ReconcileCephOsdRemoveTask {
	return &ReconcileCephOsdRemoveTask{
		Config:        &rest.Config{},
		Client:        faketestclients.GetClient(nil),
		Kubeclientset: faketestclients.GetFakeKubeclient(),
		Rookclientset: faketestclients.GetFakeRookclient(),
		Lcmclientset:  faketestclients.GetFakeLcmclient(),
		Scheme:        faketestscheme.Scheme,
	}
}

func fakeCephReconcileConfig(tconfig *taskConfig, lcmConfigData map[string]string) *cephOsdRemoveConfig {
	lcmConfig := lcmconfig.ReadConfiguration(log.With().Str(lcmcommon.LoggerObjectField, "configmap").Logger(), lcmConfigData)
	sublog := log.With().Str(lcmcommon.LoggerObjectField, "cephosdremovetask 'lcm-namespace/osdremove-task'").Logger().Level(lcmConfig.TaskParams.LogLevel)
	tc := taskConfig{}
	if tconfig != nil {
		tc = *tconfig
	}
	return &cephOsdRemoveConfig{
		context:    context.TODO(),
		api:        FakeReconciler(),
		log:        &sublog,
		lcmConfig:  &lcmConfig,
		taskConfig: tc,
	}
}

func TestTaskReconcile(t *testing.T) {
	noRequeue := reconcile.Result{}
	immidiateRequeue := reconcile.Result{Requeue: true}
	resInterval := reconcile.Result{RequeueAfter: requeueAfterInterval}
	r := FakeReconciler()
	request := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Namespace: unitinputs.LcmObjectMeta.Namespace,
			Name:      "osdremove-task",
		},
	}

	tests := []struct {
		name                 string
		inputResources       map[string]runtime.Object
		apiErrors            map[string]error
		compareWithLcmClient bool
		expectedTask         *lcmv1alpha1.CephOsdRemoveTask
		expectedErr          string
		expectedResult       reconcile.Result
	}{
		{
			name: "cephtask - not found",
			inputResources: map[string]runtime.Object{
				"cephosdremovetasks": &lcmv1alpha1.CephOsdRemoveTaskList{},
			},
			expectedResult: noRequeue,
		},
		{
			name: "cephtask - failed to get",
			inputResources: map[string]runtime.Object{
				"cephosdremovetasks": &lcmv1alpha1.CephOsdRemoveTaskList{},
			},
			apiErrors: map[string]error{
				"get-cephosdremovetasks": errors.New("failed to get task"),
			},
			expectedErr:    "failed to get task",
			expectedResult: resInterval,
		},
		{
			name: "cephtask - init failed",
			inputResources: map[string]runtime.Object{
				"cephosdremovetasks": &lcmv1alpha1.CephOsdRemoveTaskList{
					Items: []lcmv1alpha1.CephOsdRemoveTask{*unitinputs.CephOsdRemoveTaskBase.DeepCopy()},
				},
			},
			apiErrors:      map[string]error{"status": errors.New("failed to update status")},
			expectedResult: resInterval,
		},
		{
			name: "cephtask - inited",
			inputResources: map[string]runtime.Object{
				"cephosdremovetasks": &lcmv1alpha1.CephOsdRemoveTaskList{
					Items: []lcmv1alpha1.CephOsdRemoveTask{*unitinputs.CephOsdRemoveTaskBase.DeepCopy()},
				},
			},
			expectedTask:   &unitinputs.CephOsdRemoveTaskInited,
			expectedResult: immidiateRequeue,
		},
		{
			name: "cephtask - fail to list cephdeploymenthealths",
			inputResources: map[string]runtime.Object{
				"cephosdremovetasks": &lcmv1alpha1.CephOsdRemoveTaskList{
					Items: []lcmv1alpha1.CephOsdRemoveTask{*unitinputs.CephOsdRemoveTaskInited.DeepCopy()},
				},
			},
			expectedTask:   &unitinputs.CephOsdRemoveTaskInited,
			expectedResult: resInterval,
		},
		{
			name: "cephtask - no cephdeploymenthealths, failed to remove stale",
			inputResources: map[string]runtime.Object{
				"cephdeploymenthealths": &lcmv1alpha1.CephDeploymentHealthList{},
				"cephosdremovetasks": &lcmv1alpha1.CephOsdRemoveTaskList{
					Items: []lcmv1alpha1.CephOsdRemoveTask{*unitinputs.CephOsdRemoveTaskInited.DeepCopy()},
				},
			},
			expectedTask: &unitinputs.CephOsdRemoveTaskInited,
			apiErrors: map[string]error{
				"delete-cephosdremovetasks": errors.New("failed to remove task"),
			},
			compareWithLcmClient: true,
			expectedResult:       resInterval,
		},
		{
			name: "cephtask - no cephdeploymenthealths, remove stale",
			inputResources: map[string]runtime.Object{
				"cephdeploymenthealths": &lcmv1alpha1.CephDeploymentHealthList{},
				"cephosdremovetasks": &lcmv1alpha1.CephOsdRemoveTaskList{
					Items: []lcmv1alpha1.CephOsdRemoveTask{*unitinputs.CephOsdRemoveTaskInited.DeepCopy()},
				},
			},
			compareWithLcmClient: true,
			expectedResult:       noRequeue,
		},
		{
			name: "cephtask - few cephdeploymenthealths, abort failed",
			inputResources: map[string]runtime.Object{
				"cephdeploymenthealths": &lcmv1alpha1.CephDeploymentHealthList{
					Items: []lcmv1alpha1.CephDeploymentHealth{unitinputs.CephDeploymentHealth, unitinputs.CephDeploymentHealth},
				},
				"cephosdremovetasks": &lcmv1alpha1.CephOsdRemoveTaskList{
					Items: []lcmv1alpha1.CephOsdRemoveTask{*unitinputs.CephOsdRemoveTaskInited.DeepCopy()},
				},
			},
			apiErrors:      map[string]error{"status": errors.New("failed to update status")},
			expectedResult: resInterval,
		},
		{
			name: "cephtask - few cephdeploymenthealths, aborted",
			inputResources: map[string]runtime.Object{
				"cephdeploymenthealths": &lcmv1alpha1.CephDeploymentHealthList{
					Items: []lcmv1alpha1.CephDeploymentHealth{unitinputs.CephDeploymentHealth, unitinputs.CephDeploymentHealth},
				},
				"cephosdremovetasks": &lcmv1alpha1.CephOsdRemoveTaskList{
					Items: []lcmv1alpha1.CephOsdRemoveTask{*unitinputs.CephOsdRemoveTaskInited.DeepCopy()},
				},
			},
			expectedTask:   unitinputs.GetAbortedTask(unitinputs.CephOsdRemoveTaskInited, "test-time-8", "multiple CephDeploymentHealth objects found in namespace"),
			expectedResult: noRequeue,
		},
		{
			name: "cephtask - ownerRefs update failed",
			inputResources: map[string]runtime.Object{
				"cephdeploymenthealths": &lcmv1alpha1.CephDeploymentHealthList{
					Items: []lcmv1alpha1.CephDeploymentHealth{unitinputs.CephDeploymentHealth},
				},
				"cephosdremovetasks": &lcmv1alpha1.CephOsdRemoveTaskList{
					Items: []lcmv1alpha1.CephOsdRemoveTask{*unitinputs.CephOsdRemoveTaskInited.DeepCopy()},
				},
			},
			apiErrors: map[string]error{
				"update-cephosdremovetasks": errors.New("failed to update task"),
			},
			compareWithLcmClient: true,
			expectedTask:         &unitinputs.CephOsdRemoveTaskInited,
			expectedResult:       resInterval,
		},
		{
			name: "cephtask - ownerRefs updated",
			inputResources: map[string]runtime.Object{
				"cephdeploymenthealths": &lcmv1alpha1.CephDeploymentHealthList{
					Items: []lcmv1alpha1.CephDeploymentHealth{unitinputs.CephDeploymentHealth},
				},
				"cephosdremovetasks": &lcmv1alpha1.CephOsdRemoveTaskList{
					Items: []lcmv1alpha1.CephOsdRemoveTask{*unitinputs.CephOsdRemoveTaskInited.DeepCopy()},
				},
			},
			compareWithLcmClient: true,
			expectedTask:         &unitinputs.CephOsdRemoveTaskFullInited,
			expectedResult:       immidiateRequeue,
		},
		{
			name: "cephtask - failed to get cephcluster",
			inputResources: map[string]runtime.Object{
				"cephdeploymenthealths": &lcmv1alpha1.CephDeploymentHealthList{
					Items: []lcmv1alpha1.CephDeploymentHealth{unitinputs.CephDeploymentHealth},
				},
				"cephosdremovetasks": &lcmv1alpha1.CephOsdRemoveTaskList{
					Items: []lcmv1alpha1.CephOsdRemoveTask{*unitinputs.CephOsdRemoveTaskFullInited.DeepCopy()},
				},
			},
			compareWithLcmClient: true,
			expectedTask:         &unitinputs.CephOsdRemoveTaskFullInited,
			expectedResult:       resInterval,
		},
		{
			name: "cephtask - lcm skipped for external cluster, abort failed",
			inputResources: map[string]runtime.Object{
				"cephdeploymenthealths": &lcmv1alpha1.CephDeploymentHealthList{
					Items: []lcmv1alpha1.CephDeploymentHealth{unitinputs.CephDeploymentHealth},
				},
				"cephosdremovetasks": &lcmv1alpha1.CephOsdRemoveTaskList{
					Items: []lcmv1alpha1.CephOsdRemoveTask{*unitinputs.CephOsdRemoveTaskFullInited.DeepCopy()},
				},
				"cephclusters": &unitinputs.CephClusterListExternal,
			},
			apiErrors:      map[string]error{"status": errors.New("status update failed")},
			expectedResult: resInterval,
		},
		{
			name: "cephtask - lcm skipped for external cluster",
			inputResources: map[string]runtime.Object{
				"cephdeploymenthealths": &lcmv1alpha1.CephDeploymentHealthList{
					Items: []lcmv1alpha1.CephDeploymentHealth{unitinputs.CephDeploymentHealth},
				},
				"cephosdremovetasks": &lcmv1alpha1.CephOsdRemoveTaskList{
					Items: []lcmv1alpha1.CephOsdRemoveTask{*unitinputs.CephOsdRemoveTaskFullInited.DeepCopy()},
				},
				"cephclusters": &unitinputs.CephClusterListExternal,
			},
			expectedTask:   unitinputs.GetAbortedTask(unitinputs.CephOsdRemoveTaskFullInited, "test-time-13", "detected external CephCluster configuration"),
			expectedResult: noRequeue,
		},
		{
			name: "cephtask - cephcluster has no ceph status and fsid yet",
			inputResources: map[string]runtime.Object{
				"cephdeploymenthealths": &lcmv1alpha1.CephDeploymentHealthList{
					Items: []lcmv1alpha1.CephDeploymentHealth{unitinputs.CephDeploymentHealth},
				},
				"cephosdremovetasks": &lcmv1alpha1.CephOsdRemoveTaskList{
					Items: []lcmv1alpha1.CephOsdRemoveTask{*unitinputs.CephOsdRemoveTaskFullInited.DeepCopy()},
				},
				"cephclusters": &cephv1.CephClusterList{Items: []cephv1.CephCluster{unitinputs.BuildBaseCephCluster(unitinputs.ReefCephClusterReady.Name, unitinputs.ReefCephClusterReady.Namespace)}},
			},
			expectedTask: func() *lcmv1alpha1.CephOsdRemoveTask {
				req := unitinputs.CephOsdRemoveTaskFullInited.DeepCopy()
				req.ResourceVersion = "2"
				req.Status.PhaseInfo = "CephCluster is not deployed yet, no fsid provided"
				return req
			}(),
			expectedResult: resInterval,
		},
		{
			name: "cephtask - cephdeploymenthealth has no healthreport status",
			inputResources: map[string]runtime.Object{
				"cephdeploymenthealths": &lcmv1alpha1.CephDeploymentHealthList{
					Items: []lcmv1alpha1.CephDeploymentHealth{unitinputs.CephDeploymentHealth},
				},
				"cephosdremovetasks": &lcmv1alpha1.CephOsdRemoveTaskList{
					Items: []lcmv1alpha1.CephOsdRemoveTask{*unitinputs.CephOsdRemoveTaskFullInited.DeepCopy()},
				},
				"cephclusters": &unitinputs.CephClusterListReady,
			},
			expectedTask: func() *lcmv1alpha1.CephOsdRemoveTask {
				req := unitinputs.CephOsdRemoveTaskFullInited.DeepCopy()
				req.ResourceVersion = "2"
				req.Status.PhaseInfo = "related CephDeploymentHealth has no CephCluster status yet"
				return req
			}(),
			expectedResult: resInterval,
		},
		{
			name: "cephtask - cephdeploymenthealth has no osd analysis status",
			inputResources: map[string]runtime.Object{
				"cephdeploymenthealths": &lcmv1alpha1.CephDeploymentHealthList{
					Items: []lcmv1alpha1.CephDeploymentHealth{unitinputs.CephDeploymentHealthStatusNotOk},
				},
				"cephosdremovetasks": &lcmv1alpha1.CephOsdRemoveTaskList{
					Items: []lcmv1alpha1.CephOsdRemoveTask{*unitinputs.CephOsdRemoveTaskFullInited.DeepCopy()},
				},
				"cephclusters": &unitinputs.CephClusterListReady,
			},
			expectedTask: func() *lcmv1alpha1.CephOsdRemoveTask {
				req := unitinputs.CephOsdRemoveTaskFullInited.DeepCopy()
				req.ResourceVersion = "2"
				req.Status.PhaseInfo = "related CephDeploymentHealth has no CephCluster osd storage analysis yet"
				return req
			}(),
			expectedResult: resInterval,
		},
		{
			name: "cephtask - no task handling, waiting for another oldest",
			inputResources: map[string]runtime.Object{
				"cephdeploymenthealths": &lcmv1alpha1.CephDeploymentHealthList{
					Items: []lcmv1alpha1.CephDeploymentHealth{unitinputs.CephDeploymentHealthStatusOk},
				},
				"cephosdremovetasks": &lcmv1alpha1.CephOsdRemoveTaskList{
					Items: []lcmv1alpha1.CephOsdRemoveTask{
						*unitinputs.CephOsdRemoveTaskFullInited.DeepCopy(),
						*unitinputs.CephOsdRemoveTaskOld.DeepCopy(),
					},
				},
				"cephclusters": &unitinputs.CephClusterListReady,
			},
			expectedTask: func() *lcmv1alpha1.CephOsdRemoveTask {
				req := unitinputs.CephOsdRemoveTaskFullInited.DeepCopy()
				req.ResourceVersion = "2"
				req.Status.PhaseInfo = "waiting for older CephOsdRemoveTask completion"
				return req
			}(),
			expectedResult: resInterval,
		},
		{
			name: "cephtask - start task handling, requeue without interval",
			inputResources: map[string]runtime.Object{
				"cephdeploymenthealths": &lcmv1alpha1.CephDeploymentHealthList{
					Items: []lcmv1alpha1.CephDeploymentHealth{unitinputs.CephDeploymentHealthStatusOk},
				},
				"cephosdremovetasks": &lcmv1alpha1.CephOsdRemoveTaskList{
					Items: []lcmv1alpha1.CephOsdRemoveTask{
						*unitinputs.CephOsdRemoveTaskFullInited.DeepCopy(),
					},
				},
				"cephclusters": &unitinputs.CephClusterListReady,
			},
			expectedTask: func() *lcmv1alpha1.CephOsdRemoveTask {
				req := unitinputs.CephOsdRemoveTaskOnValidation.DeepCopy()
				req.ResourceVersion = "2"
				req.Status.Conditions[1].Timestamp = "test-time-18"
				return req
			}(),
			expectedResult: immidiateRequeue,
		},
		{
			name: "cephtask - start task handling, failed to update status",
			inputResources: map[string]runtime.Object{
				"cephdeploymenthealths": &lcmv1alpha1.CephDeploymentHealthList{
					Items: []lcmv1alpha1.CephDeploymentHealth{unitinputs.CephDeploymentHealthStatusOk},
				},
				"cephosdremovetasks": &lcmv1alpha1.CephOsdRemoveTaskList{
					Items: []lcmv1alpha1.CephOsdRemoveTask{
						*unitinputs.CephOsdRemoveTaskFullInited.DeepCopy(),
					},
				},
				"cephclusters": &unitinputs.CephClusterListReady,
			},
			apiErrors:      map[string]error{"status": errors.New("status update failed")},
			expectedTask:   &unitinputs.CephOsdRemoveTaskFullInited,
			expectedResult: resInterval,
		},
		{
			name: "cephtask - continue task handling, in progress, no status update",
			inputResources: map[string]runtime.Object{
				"cephdeploymenthealths": &lcmv1alpha1.CephDeploymentHealthList{
					Items: []lcmv1alpha1.CephDeploymentHealth{unitinputs.CephDeploymentHealthStatusOk},
				},
				"cephosdremovetasks": &lcmv1alpha1.CephOsdRemoveTaskList{
					Items: []lcmv1alpha1.CephOsdRemoveTask{
						*unitinputs.CephOsdRemoveTaskOnApproveWaiting.DeepCopy(),
					},
				},
				"cephclusters": &unitinputs.CephClusterListReady,
			},
			apiErrors: map[string]error{"status": errors.New("unexpected update failed")},
			expectedTask: func() *lcmv1alpha1.CephOsdRemoveTask {
				task := unitinputs.CephOsdRemoveTaskOnApproveWaiting.DeepCopy()
				task.Status.RemoveInfo.Issues = nil
				return task
			}(),
			expectedResult: resInterval,
		},
		{
			name: "cephtask - finished task handling",
			inputResources: map[string]runtime.Object{
				"cephdeploymenthealths": &lcmv1alpha1.CephDeploymentHealthList{
					Items: []lcmv1alpha1.CephDeploymentHealth{unitinputs.CephDeploymentHealthStatusOk},
				},
				"cephosdremovetasks": &lcmv1alpha1.CephOsdRemoveTaskList{
					Items: []lcmv1alpha1.CephOsdRemoveTask{
						func() lcmv1alpha1.CephOsdRemoveTask {
							newTask := unitinputs.CephOsdRemoveTaskProcessing.DeepCopy()
							newTask.Status.RemoveInfo = unitinputs.NodesRemoveFullFinishedStatus.DeepCopy()
							newTask.Status.RemoveInfo.Warnings = nil
							return *newTask
						}(),
					},
				},
				"cephclusters": &unitinputs.CephClusterListReady,
			},
			expectedTask: func() *lcmv1alpha1.CephOsdRemoveTask {
				task := unitinputs.CephOsdRemoveTaskCompleted.DeepCopy()
				task.ResourceVersion = "2"
				task.Status.RemoveInfo.Issues = nil
				task.Status.Conditions[len(task.Status.Conditions)-1].Timestamp = "test-time-21"
				return task
			}(),
			expectedResult: noRequeue,
		},
	}
	oldCurrentTime := lcmcommon.GetCurrentTimeString
	for idx, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			faketestclients.FakeReaction(r.Lcmclientset, "list", []string{"cephdeploymenthealths", "cephosdremovetasks"}, test.inputResources, nil)
			faketestclients.FakeReaction(r.Lcmclientset, "get", []string{"cephosdremovetasks"}, test.inputResources, test.apiErrors)
			faketestclients.FakeReaction(r.Lcmclientset, "update", []string{"cephosdremovetasks"}, test.inputResources, test.apiErrors)
			faketestclients.FakeReaction(r.Lcmclientset, "delete", []string{"cephosdremovetasks"}, test.inputResources, test.apiErrors)
			faketestclients.FakeReaction(r.Rookclientset, "get", []string{"cephclusters"}, test.inputResources, nil)

			if test.inputResources["cephosdremovetasks"] != nil && test.apiErrors["status"] == nil && test.expectedTask != nil {
				list := test.inputResources["cephosdremovetasks"].(*lcmv1alpha1.CephOsdRemoveTaskList)
				cb := faketestclients.GetClientBuilder()
				for _, req := range list.Items {
					cb.WithStatusSubresource(req.DeepCopy()).WithObjects(req.DeepCopy())
				}
				r.Client = faketestclients.GetClient(cb)
			} else {
				r.Client = faketestclients.GetClient(nil)
			}

			lcmcommon.GetCurrentTimeString = func() string {
				return fmt.Sprintf("test-time-%d", idx)
			}

			ctx := context.TODO()
			result, err := r.Reconcile(ctx, request)
			if test.expectedErr != "" {
				assert.NotNil(t, err)
				assert.Equal(t, test.expectedErr, err.Error())
			} else {
				assert.Nil(t, err)
			}
			assert.Equal(t, test.expectedResult, result)

			// check request status is properly updated
			cephTask := &lcmv1alpha1.CephOsdRemoveTask{}
			err = r.Client.Get(ctx, request.NamespacedName, cephTask)
			if test.compareWithLcmClient {
				// check request itself is properly updated
				cephTask, err = r.Lcmclientset.LcmV1alpha1().CephOsdRemoveTasks(request.Namespace).Get(ctx, request.Name, metav1.GetOptions{})
			}
			if (test.apiErrors != nil && test.apiErrors["status"] != nil) || test.expectedTask == nil {
				assert.NotNil(t, err)
				errMsg := "cephosdremovetasks.lcm.mirantis.com \"osdremove-task\" not found"
				if test.compareWithLcmClient {
					errMsg = "cephosdremovetasks \"osdremove-task\" not found"
				}
				assert.Equal(t, errMsg, err.Error())
			} else {
				assert.Nil(t, err)
				assert.Equal(t, test.expectedTask, cephTask)
			}
			// clean reactions
			faketestclients.CleanupFakeClientReactions(r.Lcmclientset)
			faketestclients.CleanupFakeClientReactions(r.Rookclientset)
		})
	}
	lcmcommon.GetCurrentTimeString = oldCurrentTime
}

func TestGetOldestCephOsdRemoveTask(t *testing.T) {
	tests := []struct {
		name         string
		cephTasks    []lcmv1alpha1.CephOsdRemoveTask
		expectedName string
	}{
		{
			name:      "empty request list",
			cephTasks: []lcmv1alpha1.CephOsdRemoveTask{},
		},
		{
			name: "single item in request list",
			cephTasks: []lcmv1alpha1.CephOsdRemoveTask{
				unitinputs.CephOsdRemoveTaskInited,
			},
			expectedName: "osdremove-task",
		},
		{
			name: "multiple items in request list",
			cephTasks: []lcmv1alpha1.CephOsdRemoveTask{
				unitinputs.CephOsdRemoveTaskInited,
				unitinputs.CephOsdRemoveTaskOldCompleted,
				unitinputs.CephOsdRemoveTaskOld,
			},
			expectedName: "old-osdremove-task",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			oldestName := getOldestCephOsdRemoveTaskName(test.cephTasks)
			assert.Equal(t, test.expectedName, oldestName)
		})
	}
}
