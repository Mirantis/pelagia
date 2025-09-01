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
	cephv1 "github.com/rook/rook/pkg/apis/ceph.rook.io/v1"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	faketestclients "github.com/Mirantis/pelagia/test/unit/clients"
	unitinputs "github.com/Mirantis/pelagia/test/unit/inputs"
)

func TestEnsureToolBox(t *testing.T) {
	tests := []struct {
		name              string
		infraConfig       infraConfig
		inputResources    map[string]runtime.Object
		apiErrors         map[string]error
		expectedResources map[string]runtime.Object
		expectedError     string
	}{
		{
			name: "failed to get generate deployment",
			inputResources: map[string]runtime.Object{
				"deployments": unitinputs.DeploymentListEmpty,
			},
			expectedError: "failed to generate toolbox deployment 'rook-ceph/pelagia-ceph-toolbox': failed to check 'rook-ceph/rook-ceph-operator' deployment: deployments \"rook-ceph-operator\" not found",
		},
		{
			name: "failed to get toolbox",
			inputResources: map[string]runtime.Object{
				"deployments": &appsv1.DeploymentList{
					Items: []appsv1.Deployment{*unitinputs.RookDeploymentLatestVersion},
				},
				"cephobjectstores": &unitinputs.CephObjectStoreListEmpty,
			},
			apiErrors: map[string]error{
				"get-deployments-pelagia-ceph-toolbox": errors.New("failed to get deployment"),
			},
			expectedError: "failed to check toolbox deployment 'rook-ceph/pelagia-ceph-toolbox: failed to get deployment",
		},
		{
			name: "create toolbox",
			inputResources: map[string]runtime.Object{
				"deployments": &appsv1.DeploymentList{
					Items: []appsv1.Deployment{*unitinputs.RookDeploymentLatestVersion},
				},
				"cephobjectstores": &unitinputs.CephObjectStoreListEmpty,
			},
			expectedResources: map[string]runtime.Object{
				"deployments": &appsv1.DeploymentList{
					Items: []appsv1.Deployment{*unitinputs.RookDeploymentLatestVersion, *unitinputs.ToolBoxDeploymentBase},
				},
			},
		},
		{
			name: "create toolbox failed",
			inputResources: map[string]runtime.Object{
				"deployments": &appsv1.DeploymentList{
					Items: []appsv1.Deployment{*unitinputs.RookDeploymentLatestVersion},
				},
				"cephobjectstores": &unitinputs.CephObjectStoreListEmpty,
			},
			apiErrors: map[string]error{
				"create-deployments-pelagia-ceph-toolbox": errors.New("failed to create deployment"),
			},
			expectedError: "failed to create toolbox deployment 'rook-ceph/pelagia-ceph-toolbox: failed to create deployment",
		},
		{
			name: "update toolbox failed",
			inputResources: map[string]runtime.Object{
				"deployments": &appsv1.DeploymentList{
					Items: []appsv1.Deployment{
						*unitinputs.RookDeploymentLatestVersion, *unitinputs.ToolBoxDeploymentWithRgwSecret.DeepCopy(),
					},
				},
				"cephobjectstores": &unitinputs.CephObjectStoreListEmpty,
			},
			apiErrors: map[string]error{
				"update-deployments-pelagia-ceph-toolbox": errors.New("failed to update deployment"),
			},
			expectedError: "failed to update toolbox deployment 'rook-ceph/pelagia-ceph-toolbox: failed to update deployment",
		},
		{
			name: "update toolbox",
			inputResources: map[string]runtime.Object{
				"deployments": &appsv1.DeploymentList{
					Items: []appsv1.Deployment{
						*unitinputs.RookDeploymentLatestVersion, *unitinputs.ToolBoxDeploymentWithRgwSecret.DeepCopy(),
					},
				},
				"cephobjectstores": &unitinputs.CephObjectStoreListEmpty,
			},
			expectedResources: map[string]runtime.Object{
				"deployments": &appsv1.DeploymentList{
					Items: []appsv1.Deployment{*unitinputs.RookDeploymentLatestVersion, *unitinputs.ToolBoxDeploymentBase},
				},
			},
		},
		{
			name: "update toolbox meta",
			inputResources: map[string]runtime.Object{
				"deployments": &appsv1.DeploymentList{
					Items: []appsv1.Deployment{
						*unitinputs.RookDeploymentLatestVersion,
						func() appsv1.Deployment {
							dp := unitinputs.ToolBoxDeploymentBase.DeepCopy()
							dp.Labels = nil
							dp.OwnerReferences = nil
							return *dp
						}(),
					},
				},
				"cephobjectstores": &unitinputs.CephObjectStoreListEmpty,
			},
			expectedResources: map[string]runtime.Object{
				"deployments": &appsv1.DeploymentList{
					Items: []appsv1.Deployment{*unitinputs.RookDeploymentLatestVersion, *unitinputs.ToolBoxDeploymentBase},
				},
			},
		},
		{
			name: "update toolbox with rgw secret update",
			inputResources: map[string]runtime.Object{
				"deployments": &appsv1.DeploymentList{
					Items: []appsv1.Deployment{
						*unitinputs.RookDeploymentLatestVersion,
						func() appsv1.Deployment {
							dp := unitinputs.ToolBoxDeploymentWithRgwSecret.DeepCopy()
							dp.Spec.Template.Annotations["rgw-ssl-certificate/sha256"] = "fake-sha"
							return *dp
						}(),
					},
				},
				"cephobjectstores": &unitinputs.CephObjectStoreListReady,
				"secrets":          &corev1.SecretList{Items: []corev1.Secret{unitinputs.RgwSSLCertSecret}},
			},
			expectedResources: map[string]runtime.Object{
				"deployments": &appsv1.DeploymentList{
					Items: []appsv1.Deployment{*unitinputs.RookDeploymentLatestVersion, *unitinputs.ToolBoxDeploymentWithRgwSecret},
				},
			},
		},
		{
			name: "nothing to do with toolbox",
			inputResources: map[string]runtime.Object{
				"deployments": &appsv1.DeploymentList{
					Items: []appsv1.Deployment{
						*unitinputs.RookDeploymentLatestVersion, *unitinputs.ToolBoxDeploymentBase.DeepCopy(),
					},
				},
				"cephobjectstores": &unitinputs.CephObjectStoreListEmpty,
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := fakeReconcileInfraConfig(&test.infraConfig, nil)
			faketestclients.FakeReaction(c.api.Rookclientset, "list", []string{"cephobjectstores"}, test.inputResources, nil)
			faketestclients.FakeReaction(c.api.Kubeclientset.AppsV1(), "get", []string{"deployments"}, test.inputResources, test.apiErrors)
			faketestclients.FakeReaction(c.api.Kubeclientset.AppsV1(), "create", []string{"deployments"}, test.inputResources, test.apiErrors)
			faketestclients.FakeReaction(c.api.Kubeclientset.AppsV1(), "update", []string{"deployments"}, test.inputResources, test.apiErrors)
			faketestclients.FakeReaction(c.api.Kubeclientset.CoreV1(), "get", []string{"secrets"}, test.inputResources, nil)
			test.expectedResources = faketestclients.PrepareExpectedResources(test.inputResources, test.expectedResources)

			err := c.ensureToolBox()
			if test.expectedError != "" {
				assert.NotNil(t, err)
				assert.Equal(t, test.expectedError, err.Error())
			} else {
				assert.Nil(t, err)
			}
			assert.Equal(t, test.expectedResources, test.inputResources)
			faketestclients.CleanupFakeClientReactions(c.api.Kubeclientset.AppsV1())
			faketestclients.CleanupFakeClientReactions(c.api.Kubeclientset.CoreV1())
			faketestclients.CleanupFakeClientReactions(c.api.Rookclientset)
		})
	}
}

func TestGenerateToolBox(t *testing.T) {
	deployList := &appsv1.DeploymentList{
		Items: []appsv1.Deployment{*unitinputs.RookDeploymentLatestVersion.DeepCopy()},
	}
	tests := []struct {
		name           string
		infraConfig    infraConfig
		inputResources map[string]runtime.Object
		expectedDeploy *appsv1.Deployment
		expectedError  string
	}{
		{
			name: "failed to get rook deployment",
			inputResources: map[string]runtime.Object{
				"deployments": unitinputs.DeploymentListEmpty,
			},
			expectedError: "failed to check 'rook-ceph/rook-ceph-operator' deployment: deployments \"rook-ceph-operator\" not found",
		},
		{
			name: "failed to get objectstores",
			inputResources: map[string]runtime.Object{
				"deployments": deployList,
			},
			expectedError: "failed to check cephobjectstores: failed to list cephobjectstores",
		},
		{
			name: "generate toolbox for external",
			infraConfig: infraConfig{
				externalCeph: true,
				lcmOwnerRefs: []metav1.OwnerReference{
					{
						APIVersion: "lcm.mirantis.com/v1alpha1",
						Kind:       "CephDeploymentHealth",
						Name:       "cephcluster",
					},
				},
			},
			inputResources: map[string]runtime.Object{
				"deployments":      deployList,
				"cephobjectstores": &unitinputs.CephObjectStoreListEmpty,
			},
			expectedDeploy: unitinputs.ToolBoxDeploymentExternal,
		},
		{
			name: "generate toolbox no objectstores",
			inputResources: map[string]runtime.Object{
				"deployments":      deployList,
				"cephobjectstores": &unitinputs.CephObjectStoreListEmpty,
			},
			expectedDeploy: unitinputs.ToolBoxDeploymentBase,
		},
		{
			name: "generate toolbox, objectstore, failed to find objectstore secret",
			inputResources: map[string]runtime.Object{
				"deployments":      deployList,
				"cephobjectstores": &unitinputs.CephObjectStoreListReady,
				"secrets":          &unitinputs.SecretsListEmpty,
			},
			expectedError: "failed to get secret 'rook-ceph/rgw-ssl-certificate' with cabundle for CephObjectStore 'rook-ceph/rgw-store': secrets \"rgw-ssl-certificate\" not found",
		},
		{
			name: "generate toolbox one objectstore",
			inputResources: map[string]runtime.Object{
				"deployments":      deployList,
				"cephobjectstores": &unitinputs.CephObjectStoreListReady,
				"secrets":          &corev1.SecretList{Items: []corev1.Secret{unitinputs.RgwSSLCertSecret}},
			},
			expectedDeploy: unitinputs.ToolBoxDeploymentWithRgwSecret,
		},
		{
			name: "generate toolbox multiple objectstores with secret",
			inputResources: map[string]runtime.Object{
				"deployments": deployList,
				"cephobjectstores": &cephv1.CephObjectStoreList{
					Items: []cephv1.CephObjectStore{
						unitinputs.CephObjectStoreReady,
						func() cephv1.CephObjectStore {
							store := unitinputs.CephObjectStoreReady.DeepCopy()
							store.Name = "different-name"
							store.Spec.Gateway.CaBundleRef = "another-cabundle"
							return *store
						}(),
					},
				},
				"secrets": &corev1.SecretList{
					Items: []corev1.Secret{
						unitinputs.RgwSSLCertSecret,
						func() corev1.Secret {
							newsecret := unitinputs.RgwSSLCertSecret.DeepCopy()
							newsecret.Name = "another-cabundle"
							newsecret.Data["cabundle"] = []byte("another-cabundle-for-objectstore")
							return *newsecret
						}(),
					},
				},
			},
			expectedDeploy: func() *appsv1.Deployment {
				deploy := unitinputs.ToolBoxDeploymentWithRgwSecret.DeepCopy()
				deploy.Spec.Template.Annotations["another-cabundle/sha256"] = "edb1f52c6686336991e2d03200964fb5a2a6f312a863cc4fe13312f22382b92c"
				deploy.Spec.Template.Spec.Volumes[len(deploy.Spec.Template.Spec.Volumes)-1].Secret = nil
				deploy.Spec.Template.Spec.Volumes[len(deploy.Spec.Template.Spec.Volumes)-1].Projected = &corev1.ProjectedVolumeSource{
					Sources: []corev1.VolumeProjection{
						{
							Secret: &corev1.SecretProjection{
								LocalObjectReference: corev1.LocalObjectReference{Name: "rgw-ssl-certificate"},
								Items: []corev1.KeyToPath{
									{
										Key:  "cabundle",
										Path: "rgw-ssl-certificate.crt",
										Mode: &[]int32{256}[0],
									},
								},
							},
						},
						{
							Secret: &corev1.SecretProjection{
								LocalObjectReference: corev1.LocalObjectReference{Name: "another-cabundle"},
								Items: []corev1.KeyToPath{
									{
										Key:  "cabundle",
										Path: "another-cabundle.crt",
										Mode: &[]int32{256}[0],
									},
								},
							},
						},
					},
				}
				return deploy
			}(),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := fakeReconcileInfraConfig(&test.infraConfig, nil)
			faketestclients.FakeReaction(c.api.Rookclientset, "list", []string{"cephobjectstores"}, test.inputResources, nil)
			faketestclients.FakeReaction(c.api.Kubeclientset.AppsV1(), "get", []string{"deployments"}, test.inputResources, nil)
			faketestclients.FakeReaction(c.api.Kubeclientset.CoreV1(), "get", []string{"secrets"}, test.inputResources, nil)

			deploy, err := c.generateToolBox()
			if test.expectedError != "" {
				assert.NotNil(t, err)
				assert.Equal(t, test.expectedError, err.Error())
			} else {
				assert.Nil(t, err)
			}
			assert.Equal(t, test.expectedDeploy, deploy)
			faketestclients.CleanupFakeClientReactions(c.api.Kubeclientset.AppsV1())
			faketestclients.CleanupFakeClientReactions(c.api.Kubeclientset.CoreV1())
			faketestclients.CleanupFakeClientReactions(c.api.Rookclientset)
		})
	}
}
