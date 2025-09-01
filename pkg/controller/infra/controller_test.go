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
	"context"
	"testing"

	"github.com/pkg/errors"
	cephv1 "github.com/rook/rook/pkg/apis/ceph.rook.io/v1"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	lcmv1alpha1 "github.com/Mirantis/pelagia/pkg/apis/ceph.pelagia.lcm/v1alpha1"
	lcmcommon "github.com/Mirantis/pelagia/pkg/common"
	lcmconfig "github.com/Mirantis/pelagia/pkg/controller/config"
	faketestclients "github.com/Mirantis/pelagia/test/unit/clients"
	unitinputs "github.com/Mirantis/pelagia/test/unit/inputs"
	faketestscheme "github.com/Mirantis/pelagia/test/unit/scheme"
)

func FakeReconciler() *ReconcileLcmResources {
	return &ReconcileLcmResources{
		Client:        faketestclients.GetClient(nil),
		Kubeclientset: faketestclients.GetFakeKubeclient(),
		Rookclientset: faketestclients.GetFakeRookclient(),
		Lcmclientset:  faketestclients.GetFakeLcmclient(),
		Scheme:        faketestscheme.Scheme,
	}
}

func fakeReconcileInfraConfig(config *infraConfig, lcmConfigData map[string]string) *cephDeploymentInfraConfig {
	if lcmConfigData == nil {
		lcmConfigData = map[string]string{}
	}
	lcmConfigData["HEALTH_LOG_LEVEL"] = "TRACE"
	lcmConfig := lcmconfig.ReadConfiguration(log.With().Str(lcmcommon.LoggerObjectField, "configmap").Logger(), lcmConfigData)
	sublog := log.With().Str(lcmcommon.LoggerObjectField, "namespace 'lcm-namespace'").Logger().Level(lcmConfig.HealthParams.LogLevel)
	ic := infraConfig{}
	if config != nil {
		ic = *config
	}
	ic.name = unitinputs.LcmObjectMeta.Name
	ic.namespace = unitinputs.LcmObjectMeta.Namespace
	if len(ic.lcmOwnerRefs) == 0 {
		ic.lcmOwnerRefs = []metav1.OwnerReference{
			{
				APIVersion: "lcm.mirantis.com/v1alpha1",
				Kind:       "CephDeploymentHealth",
				Name:       unitinputs.LcmObjectMeta.Name,
			},
		}
	}
	if len(ic.cephOwnerRefs) == 0 {
		ic.cephOwnerRefs = []metav1.OwnerReference{
			{
				APIVersion: "ceph.rook.io/v1",
				Kind:       "CephCluster",
				Name:       unitinputs.LcmObjectMeta.Name,
			},
		}
	}
	return &cephDeploymentInfraConfig{
		context:     context.TODO(),
		api:         FakeReconciler(),
		log:         &sublog,
		lcmConfig:   &lcmConfig,
		infraConfig: ic,
	}
}

func TestReconcile(t *testing.T) {
	r := FakeReconciler()
	requeueRes := reconcile.Result{RequeueAfter: requeueAfterInterval}
	request := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      unitinputs.LcmObjectMeta.Name,
			Namespace: unitinputs.LcmObjectMeta.Namespace,
		},
	}

	tests := []struct {
		name              string
		infraConfig       infraConfig
		inputResources    map[string]runtime.Object
		apiErrors         map[string]error
		envVarMissed      bool
		expectedResult    reconcile.Result
		expectedError     string
		expectedResources map[string]runtime.Object
	}{
		{
			name: "cephdeploymenthealth is removed, nothing to do",
			inputResources: map[string]runtime.Object{
				"cephdeploymenthealths": &lcmv1alpha1.CephDeploymentHealthList{},
				"deployments":           unitinputs.DeploymentListEmpty.DeepCopy(),
			},
			expectedResult: reconcile.Result{},
		},
		{
			name: "failed to get cephdeploymenthealth",
			inputResources: map[string]runtime.Object{
				"cephdeploymenthealths": &lcmv1alpha1.CephDeploymentHealthList{
					Items: []lcmv1alpha1.CephDeploymentHealth{unitinputs.CephDeploymentHealth},
				},
			},
			apiErrors: map[string]error{
				"get-cephdeploymenthealths": errors.New("failed to get object"),
			},
			expectedResult: requeueRes,
			expectedError:  "failed to get object",
		},
		{
			name: "required env var is not set",
			inputResources: map[string]runtime.Object{
				"cephdeploymenthealths": &lcmv1alpha1.CephDeploymentHealthList{
					Items: []lcmv1alpha1.CephDeploymentHealth{unitinputs.CephDeploymentHealth},
				},
			},
			envVarMissed:   true,
			expectedResult: requeueRes,
		},
		{
			name: "cephcluster failed to get",
			inputResources: map[string]runtime.Object{
				"cephdeploymenthealths": &lcmv1alpha1.CephDeploymentHealthList{
					Items: []lcmv1alpha1.CephDeploymentHealth{unitinputs.CephDeploymentHealth},
				},
				"cephclusters": &unitinputs.CephClusterListEmpty,
			},
			expectedResult: requeueRes,
		},
		{
			name: "cephcluster is not ready and reconcile failed",
			inputResources: map[string]runtime.Object{
				"cephdeploymenthealths": &lcmv1alpha1.CephDeploymentHealthList{
					Items: []lcmv1alpha1.CephDeploymentHealth{unitinputs.CephDeploymentHealth},
				},
				"cephclusters":     &unitinputs.CephClusterListNotReady,
				"cephobjectstores": &unitinputs.CephObjectStoreListEmpty,
				"deployments": &appsv1.DeploymentList{
					Items: []appsv1.Deployment{*unitinputs.RookDeploymentLatestVersion},
				},
				"daemonsets": unitinputs.DaemonSetListEmpty.DeepCopy(),
			},
			apiErrors: map[string]error{
				"get-deployments-rook-ceph-operator": errors.New("failed to get deploy"),
			},
			expectedResult: requeueRes,
		},
		{
			name: "reconcile base",
			inputResources: map[string]runtime.Object{
				"cephdeploymenthealths": &lcmv1alpha1.CephDeploymentHealthList{
					Items: []lcmv1alpha1.CephDeploymentHealth{unitinputs.CephDeploymentHealth},
				},
				"cephclusters":     &unitinputs.CephClusterListReady,
				"cephobjectstores": &unitinputs.CephObjectStoreListEmpty,
				"deployments": &appsv1.DeploymentList{
					Items: []appsv1.Deployment{*unitinputs.RookDeploymentLatestVersion},
				},
				"daemonsets": unitinputs.DaemonSetListEmpty.DeepCopy(),
			},
			expectedResult: requeueRes,
			expectedResources: map[string]runtime.Object{
				"deployments": &appsv1.DeploymentList{
					Items: []appsv1.Deployment{
						*unitinputs.RookDeploymentLatestVersion, *unitinputs.ToolBoxDeploymentBase,
					},
				},
				"daemonsets": &appsv1.DaemonSetList{
					Items: []appsv1.DaemonSet{*unitinputs.DiskDaemonDaemonset.DeepCopy()},
				},
			},
		},
		{
			name: "reconcile with rgw and osd tolerations",
			inputResources: map[string]runtime.Object{
				"cephdeploymenthealths": &lcmv1alpha1.CephDeploymentHealthList{
					Items: []lcmv1alpha1.CephDeploymentHealth{unitinputs.CephDeploymentHealth},
				},
				"cephclusters": &cephv1.CephClusterList{
					Items: []cephv1.CephCluster{
						func() cephv1.CephCluster {
							cluster := unitinputs.ReefCephClusterReady.DeepCopy()
							cluster.Spec.Placement = cephv1.PlacementSpec{
								cephv1.KeyOSD: cephv1.Placement{
									Tolerations: []corev1.Toleration{
										{
											Key:      "test.kubernetes.io/testkey",
											Effect:   "Schedule",
											Operator: "Exists",
										},
									},
								},
							}
							return *cluster
						}(),
					},
				},
				"cephobjectstores": &unitinputs.CephObjectStoreListReady,
				"deployments": &appsv1.DeploymentList{
					Items: []appsv1.Deployment{*unitinputs.RookDeploymentLatestVersion},
				},
				"daemonsets": unitinputs.DaemonSetListEmpty.DeepCopy(),
				"secrets":    &corev1.SecretList{Items: []corev1.Secret{unitinputs.RgwSSLCertSecret}},
			},
			expectedResult: requeueRes,
			expectedResources: map[string]runtime.Object{
				"deployments": &appsv1.DeploymentList{
					Items: []appsv1.Deployment{
						*unitinputs.RookDeploymentLatestVersion, *unitinputs.ToolBoxDeploymentWithRgwSecret,
					},
				},
				"daemonsets": &appsv1.DaemonSetList{
					Items: []appsv1.DaemonSet{*unitinputs.DiskDaemonDaemonsetWithOsdTolerations.DeepCopy()},
				},
			},
		},
		{
			name: "reconcile external ok",
			inputResources: map[string]runtime.Object{
				"cephdeploymenthealths": &lcmv1alpha1.CephDeploymentHealthList{
					Items: []lcmv1alpha1.CephDeploymentHealth{unitinputs.CephDeploymentHealth},
				},
				"cephclusters": &unitinputs.CephClusterListExternal,
				"deployments": &appsv1.DeploymentList{
					Items: []appsv1.Deployment{*unitinputs.RookDeploymentLatestVersion},
				},
				"cephobjectstores": &unitinputs.CephObjectStoreListEmpty,
				"daemonsets":       unitinputs.DaemonSetListEmpty.DeepCopy(),
			},
			expectedResult: requeueRes,
			expectedResources: map[string]runtime.Object{
				"deployments": &appsv1.DeploymentList{
					Items: []appsv1.Deployment{
						*unitinputs.RookDeploymentLatestVersion, *unitinputs.ToolBoxDeploymentExternal,
					},
				},
				"daemonsets": unitinputs.DaemonSetListEmpty.DeepCopy(),
			},
		},
	}
	oldVar := lcmcommon.LookupEnv
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			lcmcommon.LookupEnv = func(_ string) (string, bool) {
				if test.envVarMissed {
					return "", false
				}
				return "some-registry/lcm-controller:v1", true
			}
			test.expectedResources = faketestclients.PrepareExpectedResources(test.inputResources, test.expectedResources)

			faketestclients.FakeReaction(r.Rookclientset, "list", []string{"cephobjectstores"}, test.inputResources, nil)
			faketestclients.FakeReaction(r.Lcmclientset, "get", []string{"cephdeploymenthealths"}, test.inputResources, test.apiErrors)
			faketestclients.FakeReaction(r.Rookclientset, "get", []string{"cephclusters"}, test.inputResources, test.apiErrors)
			faketestclients.FakeReaction(r.Kubeclientset.AppsV1(), "get", []string{"deployments", "daemonsets"}, test.inputResources, test.apiErrors)
			faketestclients.FakeReaction(r.Kubeclientset.AppsV1(), "create", []string{"deployments", "daemonsets"}, test.inputResources, test.apiErrors)
			faketestclients.FakeReaction(r.Kubeclientset.CoreV1(), "get", []string{"secrets"}, test.inputResources, test.apiErrors)

			res, err := r.Reconcile(context.TODO(), request)
			if test.expectedError != "" {
				assert.NotNil(t, err)
				assert.Equal(t, test.expectedError, err.Error())
			} else {
				assert.Nil(t, err)
			}
			assert.Equal(t, test.expectedResult, res)
			assert.Equal(t, test.expectedResources, test.inputResources)

			faketestclients.CleanupFakeClientReactions(r.Lcmclientset)
			faketestclients.CleanupFakeClientReactions(r.Kubeclientset.AppsV1())
			faketestclients.CleanupFakeClientReactions(r.Kubeclientset.CoreV1())
			faketestclients.CleanupFakeClientReactions(r.Rookclientset)
		})
	}
	lcmcommon.LookupEnv = oldVar
}
