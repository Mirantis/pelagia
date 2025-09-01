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

package deployment

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	cephlcmv1alpha1 "github.com/Mirantis/pelagia/pkg/apis/ceph.pelagia.lcm/v1alpha1"
	faketestclients "github.com/Mirantis/pelagia/test/unit/clients"
	unitinputs "github.com/Mirantis/pelagia/test/unit/inputs"
)

func TestAddExternalResources(t *testing.T) {
	cephOwnerRefs := []metav1.OwnerReference{
		{
			APIVersion: "ceph.rook.io/v1",
			Kind:       "CephCluster",
			Name:       "fake",
		},
	}
	tests := []struct {
		name              string
		cephDpl           *cephlcmv1alpha1.CephDeployment
		ownerRefs         []metav1.OwnerReference
		inputResources    map[string]runtime.Object
		apiErrors         map[string]error
		stateChanged      bool
		expectedResources map[string]runtime.Object
		expectedError     string
	}{
		{
			name:    "external connection secret is not found",
			cephDpl: &unitinputs.CephDeployExternal,
			inputResources: map[string]runtime.Object{
				"secrets": &corev1.SecretList{},
			},
			expectedError: "failed to get secret 'lcm-namespace/pelagia-external-connection' with external connection info: secrets \"pelagia-external-connection\" not found",
		},
		{
			name:    "external connection secret has empty connection parameter",
			cephDpl: &unitinputs.CephDeployExternal,
			inputResources: map[string]runtime.Object{
				"secrets": &corev1.SecretList{Items: []corev1.Secret{unitinputs.GetExternalConnectionSecret(nil)}},
			},
			expectedError: "required for connection to external cluster parameters ('connection' field) is not specified in secret 'lcm-namespace/pelagia-external-connection'",
		},
		{
			name:    "external connection secret has wrong connection parameter value",
			cephDpl: &unitinputs.CephDeployExternal,
			inputResources: map[string]runtime.Object{
				"secrets": &corev1.SecretList{Items: []corev1.Secret{unitinputs.GetExternalConnectionSecret([]byte("{||}"))}},
			},
			expectedError: "failed to parse external connection string from secret 'lcm-namespace/pelagia-external-connection': invalid character '|' looking for beginning of object key string",
		},
		{
			name:    "external connection with admin user, create mon endpoints map failed",
			cephDpl: unitinputs.CephDeployExternal.DeepCopy(),
			inputResources: map[string]runtime.Object{
				"configmaps": &corev1.ConfigMapList{},
				"secrets": &corev1.SecretList{
					Items: []corev1.Secret{unitinputs.ExternalConnectionSecretWithAdmin},
				},
			},
			apiErrors: map[string]error{
				"create-configmaps": errors.New("failed to create config map"),
			},
			expectedError: "failed to manage config map for external cluster: failed to create rook-ceph/rook-ceph-mon-endpoints config map: failed to create config map",
		},
		{
			name:    "external connection with admin user, created",
			cephDpl: unitinputs.CephDeployExternal.DeepCopy(),
			inputResources: map[string]runtime.Object{
				"configmaps": &corev1.ConfigMapList{
					Items: []corev1.ConfigMap{},
				},
				"secrets": &corev1.SecretList{
					Items: []corev1.Secret{unitinputs.ExternalConnectionSecretWithAdmin},
				},
			},
			expectedResources: map[string]runtime.Object{
				"configmaps": &corev1.ConfigMapList{
					Items: []corev1.ConfigMap{*unitinputs.RookCephMonEndpointsExternal.DeepCopy()},
				},
				"secrets": &corev1.SecretList{
					Items: []corev1.Secret{unitinputs.ExternalConnectionSecretWithAdmin, *unitinputs.RookCephMonSecret.DeepCopy()},
				},
			},
			stateChanged: true,
		},
		{
			name:    "external connection with admin user, nothing to do",
			cephDpl: unitinputs.CephDeployExternal.DeepCopy(),
			inputResources: map[string]runtime.Object{
				"configmaps": &corev1.ConfigMapList{
					Items: []corev1.ConfigMap{*unitinputs.RookCephMonEndpointsExternal.DeepCopy()},
				},
				"secrets": &corev1.SecretList{
					Items: []corev1.Secret{unitinputs.ExternalConnectionSecretWithAdmin, *unitinputs.RookCephMonSecret.DeepCopy()},
				},
			},
		},
		{
			name:    "external connection with admin user, external rgw, failed to create rgw ops secret",
			cephDpl: unitinputs.CephDeployExternalRgw.DeepCopy(),
			inputResources: map[string]runtime.Object{
				"configmaps": &corev1.ConfigMapList{
					Items: []corev1.ConfigMap{},
				},
				"secrets": &corev1.SecretList{
					Items: []corev1.Secret{unitinputs.ExternalConnectionSecretWithAdminAndRgw},
				},
			},
			expectedResources: map[string]runtime.Object{
				"configmaps": &corev1.ConfigMapList{
					Items: []corev1.ConfigMap{*unitinputs.RookCephMonEndpointsExternal.DeepCopy()},
				},
				"secrets": &corev1.SecretList{
					Items: []corev1.Secret{unitinputs.ExternalConnectionSecretWithAdminAndRgw, *unitinputs.RookCephMonSecret.DeepCopy()},
				},
			},
			apiErrors: map[string]error{
				"get-secrets-rgw-admin-ops-user": errors.New("failed to get rgw ops secret secret"),
			},
			expectedError: "failed to manage secrets for external cluster: failed to get rgw ops secret secret",
		},
		{
			name:    "external connection with admin user, external rgw, created",
			cephDpl: unitinputs.CephDeployExternalRgw.DeepCopy(),
			inputResources: map[string]runtime.Object{
				"configmaps": &corev1.ConfigMapList{
					Items: []corev1.ConfigMap{},
				},
				"secrets": &corev1.SecretList{
					Items: []corev1.Secret{unitinputs.ExternalConnectionSecretWithAdminAndRgw},
				},
			},
			expectedResources: map[string]runtime.Object{
				"configmaps": &corev1.ConfigMapList{
					Items: []corev1.ConfigMap{*unitinputs.RookCephMonEndpointsExternal.DeepCopy()},
				},
				"secrets": &corev1.SecretList{
					Items: []corev1.Secret{unitinputs.ExternalConnectionSecretWithAdminAndRgw, *unitinputs.RookCephMonSecret.DeepCopy(), *unitinputs.RookCephRgwAdminSecret.DeepCopy()},
				},
			},
			stateChanged: true,
		},
		{
			name:    "external connection with admin user, external rgw, no changes",
			cephDpl: unitinputs.CephDeployExternalRgw.DeepCopy(),
			inputResources: map[string]runtime.Object{
				"configmaps": &corev1.ConfigMapList{
					Items: []corev1.ConfigMap{*unitinputs.RookCephMonEndpointsExternal.DeepCopy()},
				},
				"secrets": &corev1.SecretList{
					Items: []corev1.Secret{unitinputs.ExternalConnectionSecretWithAdminAndRgw, *unitinputs.RookCephMonSecret.DeepCopy(), *unitinputs.RookCephRgwAdminSecret.DeepCopy()},
				},
			},
		},
		{
			name:    "external connection with non admin user, failed to create secrets",
			cephDpl: unitinputs.CephDeployExternalCephFS.DeepCopy(),
			inputResources: map[string]runtime.Object{
				"configmaps": &corev1.ConfigMapList{
					Items: []corev1.ConfigMap{*unitinputs.RookCephMonEndpointsExternal.DeepCopy()},
				},
				"secrets": &corev1.SecretList{Items: []corev1.Secret{unitinputs.ExternalConnectionSecretNonAdmin}},
			},
			apiErrors: map[string]error{
				"create-secrets-rook-csi-rbd-node":           errors.New("failed to create csi rbd node secret"),
				"create-secrets-rook-csi-rbd-provisioner":    errors.New("failed to create csi rbd provisioner secret"),
				"create-secrets-rook-csi-cephfs-node":        errors.New("failed to create csi cephfs node secret"),
				"create-secrets-rook-csi-cephfs-provisioner": errors.New("failed to create csi cephfs provisioner secret"),
				"create-secrets-rook-ceph-mon":               errors.New("failed to create ceph mon secret"),
			},
			expectedError: "failed to manage secrets for external cluster: failed to create ceph mon secret, failed to create csi rbd node secret, failed to create csi rbd provisioner secret, failed to create csi cephfs node secret, failed to create csi cephfs provisioner secret",
		},
		{
			name:    "external connection with non admin user, created secrets",
			cephDpl: unitinputs.CephDeployExternalCephFS.DeepCopy(),
			inputResources: map[string]runtime.Object{
				"configmaps": &corev1.ConfigMapList{
					Items: []corev1.ConfigMap{*unitinputs.RookCephMonEndpointsExternal.DeepCopy()},
				},
				"secrets": &corev1.SecretList{Items: []corev1.Secret{unitinputs.ExternalConnectionSecretNonAdmin}},
			},
			expectedResources: map[string]runtime.Object{
				"configmaps": &corev1.ConfigMapList{
					Items: []corev1.ConfigMap{*unitinputs.RookCephMonEndpointsExternal.DeepCopy()},
				},
				"secrets": &corev1.SecretList{
					Items: []corev1.Secret{
						unitinputs.ExternalConnectionSecretNonAdmin,
						*unitinputs.RookCephMonSecretNonAdmin.DeepCopy(),
						*unitinputs.CSIRBDNodeSecret.DeepCopy(), *unitinputs.CSIRBDProvisionerSecret.DeepCopy(),
						*unitinputs.CSICephFSNodeSecret.DeepCopy(), *unitinputs.CSICephFSProvisionerSecret.DeepCopy(),
					},
				},
			},
			stateChanged: true,
		},
		{
			name:      "external connection with non admin user, failed to update secrets",
			cephDpl:   unitinputs.CephDeployExternalCephFS.DeepCopy(),
			ownerRefs: cephOwnerRefs,
			inputResources: map[string]runtime.Object{
				"configmaps": &corev1.ConfigMapList{
					Items: []corev1.ConfigMap{*unitinputs.RookCephMonEndpointsExternal.DeepCopy()},
				},
				"secrets": &corev1.SecretList{
					Items: []corev1.Secret{
						unitinputs.ExternalConnectionSecretNonAdmin,
						*unitinputs.RookCephMonSecretNonAdmin.DeepCopy(),
						*unitinputs.CSIRBDNodeSecret.DeepCopy(), *unitinputs.CSIRBDProvisionerSecret.DeepCopy(),
						*unitinputs.CSICephFSNodeSecret.DeepCopy(), *unitinputs.CSICephFSProvisionerSecret.DeepCopy(),
					},
				},
			},
			expectedResources: map[string]runtime.Object{
				"configmaps": &corev1.ConfigMapList{
					Items: []corev1.ConfigMap{
						func() corev1.ConfigMap {
							cmUpdated := unitinputs.RookCephMonEndpointsExternal.DeepCopy()
							cmUpdated.OwnerReferences = cephOwnerRefs
							return *cmUpdated
						}(),
					},
				},
			},
			apiErrors: map[string]error{
				"update-secrets-rook-csi-rbd-node":           errors.New("failed to update csi rbd node secret"),
				"update-secrets-rook-csi-rbd-provisioner":    errors.New("failed to update csi rbd provisioner secret"),
				"update-secrets-rook-csi-cephfs-node":        errors.New("failed to update csi cephfs node secret"),
				"update-secrets-rook-csi-cephfs-provisioner": errors.New("failed to update csi cephfs provisioner secret"),
				"update-secrets-rook-ceph-mon":               errors.New("failed to update ceph mon secret"),
			},
			expectedError: "failed to manage secrets for external cluster: failed to update ceph mon secret, failed to update csi rbd node secret, failed to update csi rbd provisioner secret, failed to update csi cephfs node secret, failed to update csi cephfs provisioner secret",
		},
		{
			name:    "external connection with non admin user, nothing to do",
			cephDpl: unitinputs.CephDeployExternalCephFS.DeepCopy(),
			inputResources: map[string]runtime.Object{
				"configmaps": &corev1.ConfigMapList{
					Items: []corev1.ConfigMap{*unitinputs.RookCephMonEndpointsExternal.DeepCopy()},
				},
				"secrets": &corev1.SecretList{
					Items: []corev1.Secret{
						unitinputs.ExternalConnectionSecretNonAdmin,
						*unitinputs.RookCephMonSecretNonAdmin.DeepCopy(),
						*unitinputs.CSIRBDNodeSecret.DeepCopy(), *unitinputs.CSIRBDProvisionerSecret.DeepCopy(),
						*unitinputs.CSICephFSNodeSecret.DeepCopy(), *unitinputs.CSICephFSProvisionerSecret.DeepCopy(),
					},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := fakeDeploymentConfig(&deployConfig{cephDpl: test.cephDpl}, nil)

			faketestclients.FakeReaction(c.api.Kubeclientset.CoreV1(), "get", []string{"configmaps", "secrets"}, test.inputResources, test.apiErrors)
			faketestclients.FakeReaction(c.api.Kubeclientset.CoreV1(), "create", []string{"configmaps", "secrets"}, test.inputResources, test.apiErrors)
			faketestclients.FakeReaction(c.api.Kubeclientset.CoreV1(), "update", []string{"configmaps", "secrets"}, test.inputResources, test.apiErrors)
			test.expectedResources = faketestclients.PrepareExpectedResources(test.inputResources, test.expectedResources)

			changed, err := c.addExternalResources(test.ownerRefs)
			if test.expectedError != "" {
				assert.NotNil(t, err)
				assert.Equal(t, test.expectedError, err.Error())
			} else {
				assert.Nil(t, err)
			}
			assert.Equal(t, test.stateChanged, changed)
			assert.Equal(t, test.expectedResources, test.inputResources)
			faketestclients.CleanupFakeClientReactions(c.api.Kubeclientset.CoreV1())
		})
	}
}

func TestManageSecrets(t *testing.T) {
	tests := []struct {
		name           string
		secrets        []*corev1.Secret
		inputResources map[string]runtime.Object
		apiErrors      map[string]error
		stateChanged   bool
		expectedError  string
	}{
		{
			name: "failed to manage some secrets",
			secrets: []*corev1.Secret{
				&unitinputs.RookCephMonSecret, &unitinputs.CSIRBDNodeSecret, &unitinputs.CSIRBDProvisionerSecret,
			},
			inputResources: map[string]runtime.Object{
				"secrets": &corev1.SecretList{
					Items: []corev1.Secret{*unitinputs.RookCephMonSecretNonAdmin.DeepCopy()},
				},
			},
			apiErrors: map[string]error{
				"update-secrets-rook-ceph-mon":            errors.New("failed to update rook-ceph-mon secret"),
				"get-secrets-rook-csi-rbd-node":           errors.New("failed to get rook-csi-rbd-node secret"),
				"create-secrets-rook-csi-rbd-provisioner": errors.New("failed to create rook-csi-rbd-provisioner secret"),
			},
			expectedError: "failed to manage secrets for external cluster: failed to update rook-ceph-mon secret, failed to get rook-csi-rbd-node secret, failed to create rook-csi-rbd-provisioner secret",
		},
		{
			name: "successfully manage some secrets",
			secrets: []*corev1.Secret{
				&unitinputs.RookCephMonSecret, &unitinputs.CSIRBDNodeSecret, &unitinputs.CSIRBDProvisionerSecret,
			},
			inputResources: map[string]runtime.Object{
				"secrets": &corev1.SecretList{
					Items: []corev1.Secret{*unitinputs.RookCephMonSecretNonAdmin.DeepCopy(), *unitinputs.CSIRBDNodeSecret.DeepCopy()},
				},
			},
			stateChanged: true,
		},
		{
			name: "nothing to change",
			secrets: []*corev1.Secret{
				&unitinputs.CSIRBDNodeSecret, &unitinputs.CSIRBDProvisionerSecret,
			},
			inputResources: map[string]runtime.Object{
				"secrets": &corev1.SecretList{
					Items: []corev1.Secret{*unitinputs.CSIRBDProvisionerSecret.DeepCopy(), *unitinputs.CSIRBDNodeSecret.DeepCopy()},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := fakeDeploymentConfig(nil, nil)
			faketestclients.FakeReaction(c.api.Kubeclientset.CoreV1(), "get", []string{"secrets"}, test.inputResources, test.apiErrors)
			faketestclients.FakeReaction(c.api.Kubeclientset.CoreV1(), "create", []string{"secrets"}, test.inputResources, test.apiErrors)
			faketestclients.FakeReaction(c.api.Kubeclientset.CoreV1(), "update", []string{"secrets"}, test.inputResources, test.apiErrors)

			changed, err := c.manageSecrets(test.secrets)
			if test.expectedError != "" {
				assert.NotNil(t, err)
				assert.Equal(t, test.expectedError, err.Error())
			} else {
				assert.Nil(t, err)
			}
			assert.Equal(t, test.stateChanged, changed)
			faketestclients.CleanupFakeClientReactions(c.api.Kubeclientset.CoreV1())
		})
	}
}

func TestManageConfigMap(t *testing.T) {
	tests := []struct {
		name             string
		configMap        *corev1.ConfigMap
		presentConfigMap *corev1.ConfigMap
		apiErrors        map[string]error
		changed          bool
		expectedError    string
	}{
		{
			name:          "failed to get config map",
			configMap:     &unitinputs.RookCephMonEndpointsExternal,
			apiErrors:     map[string]error{"get-configmaps": errors.New("failed to get config map")},
			expectedError: "failed to get rook-ceph/rook-ceph-mon-endpoints config map: failed to get config map",
		},
		{
			name:          "failed to create config map",
			configMap:     &unitinputs.RookCephMonEndpointsExternal,
			apiErrors:     map[string]error{"create-configmaps": errors.New("failed to create config map")},
			expectedError: "failed to create rook-ceph/rook-ceph-mon-endpoints config map: failed to create config map",
		},
		{
			name: "failed to update config map",
			configMap: func() *corev1.ConfigMap {
				configMap := unitinputs.RookCephMonEndpointsExternal.DeepCopy()
				// changed mon endpoints to initiate update
				configMap.Data["data"] = "cmn01=10.0.0.4:6969,cmn02=10.0.0.2:6969,cmn03=10.0.0.3:6969"
				return configMap
			}(),
			presentConfigMap: unitinputs.RookCephMonEndpointsExternal.DeepCopy(),
			apiErrors:        map[string]error{"update-configmaps": errors.New("failed to update config map")},
			expectedError:    "failed to update rook-ceph/rook-ceph-mon-endpoints config map: failed to update config map",
		},
		{
			name:      "successfully create config map",
			configMap: &unitinputs.RookCephMonEndpointsExternal,
			changed:   true,
		},
		{
			name: "successfully update config map data",
			configMap: func() *corev1.ConfigMap {
				configMap := unitinputs.RookCephMonEndpointsExternal.DeepCopy()
				// changed mon endpoints to initiate update
				configMap.Data["data"] = "cmn01=10.0.0.4:6969,cmn02=10.0.0.2:6969,cmn03=10.0.0.3:6969"
				return configMap
			}(),
			presentConfigMap: unitinputs.RookCephMonEndpointsExternal.DeepCopy(),
			changed:          true,
		},
		{
			name: "successfully update config map owner refs",
			configMap: func() *corev1.ConfigMap {
				configMap := unitinputs.RookCephMonEndpointsExternal.DeepCopy()
				configMap.OwnerReferences = []metav1.OwnerReference{
					{
						APIVersion: "ceph.rook.io/v1",
						Kind:       "CephCluster",
						Name:       "fake",
					},
				}
				return configMap
			}(),
			presentConfigMap: unitinputs.RookCephMonEndpointsExternal.DeepCopy(),
			changed:          true,
		},
		{
			name:             "no config map update",
			configMap:        unitinputs.RookCephMonEndpointsExternal.DeepCopy(),
			presentConfigMap: unitinputs.RookCephMonEndpointsExternal.DeepCopy(),
			apiErrors:        map[string]error{"update-configmaps": errors.New("unexpected update configmap")},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := fakeDeploymentConfig(nil, nil)
			inputResources := map[string]runtime.Object{"configmaps": &corev1.ConfigMapList{}}
			if test.presentConfigMap != nil {
				inputResources["configmaps"] = &corev1.ConfigMapList{Items: []corev1.ConfigMap{*test.presentConfigMap}}
			}
			faketestclients.FakeReaction(c.api.Kubeclientset.CoreV1(), "get", []string{"configmaps"}, inputResources, test.apiErrors)
			faketestclients.FakeReaction(c.api.Kubeclientset.CoreV1(), "create", []string{"configmaps"}, inputResources, test.apiErrors)
			faketestclients.FakeReaction(c.api.Kubeclientset.CoreV1(), "update", []string{"configmaps"}, inputResources, test.apiErrors)

			changed, err := c.manageConfigMap(test.configMap)
			if test.expectedError != "" {
				assert.NotNil(t, err)
				assert.Equal(t, test.expectedError, err.Error())
			} else {
				assert.Nil(t, err)
			}
			assert.Equal(t, test.changed, changed)
			faketestclients.CleanupFakeClientReactions(c.api.Kubeclientset.CoreV1())
		})
	}
}
