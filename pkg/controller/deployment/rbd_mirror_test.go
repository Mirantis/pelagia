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
	"testing"

	"github.com/pkg/errors"
	cephv1 "github.com/rook/rook/pkg/apis/ceph.rook.io/v1"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"

	cephlcmv1alpha1 "github.com/Mirantis/pelagia/pkg/apis/ceph.pelagia.lcm/v1alpha1"
	faketestclients "github.com/Mirantis/pelagia/test/unit/clients"
	unitinputs "github.com/Mirantis/pelagia/test/unit/inputs"
)

func TestEnsureRBDMirror(t *testing.T) {
	tests := []struct {
		name              string
		cephDpl           *cephlcmv1alpha1.CephDeployment
		inputResources    map[string]runtime.Object
		apiErrors         map[string]error
		stateChanged      bool
		expectedResources map[string]runtime.Object
		expectedError     string
	}{
		{
			name:    "no rbd specified - nothing to cleanup",
			cephDpl: &unitinputs.BaseCephDeployment,
			inputResources: map[string]runtime.Object{
				"secrets":        unitinputs.SecretsListEmpty.DeepCopy(),
				"cephrbdmirrors": unitinputs.CephRBDMirrorsEmpty.DeepCopy(),
			},
		},
		{
			name:    "no rbd specified - cleanup",
			cephDpl: &unitinputs.BaseCephDeployment,
			inputResources: map[string]runtime.Object{
				"secrets":        unitinputs.CephRBDSecretList.DeepCopy(),
				"cephrbdmirrors": unitinputs.CephRBDMirrorsList.DeepCopy(),
			},
			expectedResources: map[string]runtime.Object{
				"secrets":        &unitinputs.SecretsListEmpty,
				"cephrbdmirrors": &unitinputs.CephRBDMirrorsEmpty,
			},
			stateChanged: true,
		},
		{
			name:    "no rbd specified - cleanup failed",
			cephDpl: &unitinputs.BaseCephDeployment,
			inputResources: map[string]runtime.Object{
				"secrets":        unitinputs.CephRBDSecretList.DeepCopy(),
				"cephrbdmirrors": unitinputs.CephRBDMirrorsList.DeepCopy(),
			},
			apiErrors:     map[string]error{"delete-cephrbdmirrors": errors.New("cephrbdmirrors delete failed")},
			expectedError: "failed to cleanup CephRBDMirrors: failed to remove CephRBDMirroring: cephrbdmirrors delete failed",
		},
		{
			name:           "failed to list rbd",
			cephDpl:        &unitinputs.CephDeployEnsureRbdMirror,
			inputResources: map[string]runtime.Object{},
			expectedError:  "failed to get list of CephRBDMirrors: failed to list cephrbdmirrors",
		},
		{
			name:    "create a new rbd",
			cephDpl: &unitinputs.CephDeployEnsureRbdMirror,
			inputResources: map[string]runtime.Object{
				"secrets":        unitinputs.SecretsListEmpty.DeepCopy(),
				"cephrbdmirrors": unitinputs.CephRBDMirrorsEmpty.DeepCopy(),
			},
			expectedResources: map[string]runtime.Object{
				"secrets":        &unitinputs.CephRBDSecretList,
				"cephrbdmirrors": &unitinputs.CephRBDMirrorsList,
			},
			stateChanged: true,
		},
		{
			name:    "create a new rbd failed",
			cephDpl: &unitinputs.CephDeployEnsureRbdMirror,
			inputResources: map[string]runtime.Object{
				"secrets":        unitinputs.SecretsListEmpty.DeepCopy(),
				"cephrbdmirrors": unitinputs.CephRBDMirrorsEmpty.DeepCopy(),
			},
			apiErrors: map[string]error{"create-cephrbdmirrors": errors.New("cephrbdmirrors create failed")},
			expectedResources: map[string]runtime.Object{
				"secrets":        &unitinputs.CephRBDSecretList,
				"cephrbdmirrors": &unitinputs.CephRBDMirrorsEmpty,
			},
			expectedError: "failed to create cephcluster CephRBDMirror: cephrbdmirrors create failed",
		},
		{
			name:    "many rbd specifed, cleanup extra rbd failed",
			cephDpl: &unitinputs.CephDeployEnsureRbdMirror,
			inputResources: map[string]runtime.Object{
				"secrets": unitinputs.CephRBDSecretList.DeepCopy(),
				"cephrbdmirrors": &cephv1.CephRBDMirrorList{
					Items: []cephv1.CephRBDMirror{
						unitinputs.CephRBDMirror,
						func() cephv1.CephRBDMirror {
							mirror := unitinputs.CephRBDMirror.DeepCopy()
							mirror.Name = "another-name"
							return *mirror
						}(),
					},
				},
			},
			apiErrors:     map[string]error{"delete-cephrbdmirrors": errors.New("cephrbdmirrors delete failed")},
			expectedError: "failed to remove unspecified another-name CephRBDMirror: cephrbdmirrors delete failed",
		},
		{
			name:    "many rbd specifed, cleanup extra rbd and actual is not ready",
			cephDpl: &unitinputs.CephDeployEnsureRbdMirror,
			inputResources: map[string]runtime.Object{
				"secrets": unitinputs.CephRBDSecretList.DeepCopy(),
				"cephrbdmirrors": &cephv1.CephRBDMirrorList{
					Items: []cephv1.CephRBDMirror{
						unitinputs.CephRBDMirror,
						func() cephv1.CephRBDMirror {
							mirror := unitinputs.CephRBDMirror.DeepCopy()
							mirror.Name = "another-name"
							return *mirror
						}(),
					},
				},
			},
			expectedResources: map[string]runtime.Object{
				"secrets":        &unitinputs.CephRBDSecretList,
				"cephrbdmirrors": &unitinputs.CephRBDMirrorsList,
			},
			expectedError: "resource RBDMirror is not ready, status is not available, waiting",
		},
		{
			name:    "failed to check secrets",
			cephDpl: &unitinputs.CephDeployEnsureRbdMirror,
			inputResources: map[string]runtime.Object{
				"secrets":        unitinputs.SecretsListEmpty.DeepCopy(),
				"cephrbdmirrors": &unitinputs.CephRBDMirrorsList,
			},
			apiErrors:     map[string]error{"get-secrets": errors.New("failed to get secret")},
			expectedError: "failed to get rbd-mirror-token-mirror1-pool-1 secret: failed to get secret",
		},
		{
			name: "update secrets",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				mc := unitinputs.CephDeployEnsureRbdMirror.DeepCopy()
				peer := mc.Spec.RBDMirror.Peers[0]
				peer.Token = "another-token"
				mc.Spec.RBDMirror.Peers[0] = peer
				return mc
			}(),
			inputResources: map[string]runtime.Object{
				"secrets":        unitinputs.CephRBDSecretList.DeepCopy(),
				"cephrbdmirrors": unitinputs.CephRBDMirrorsListReady.DeepCopy(),
			},
			expectedResources: map[string]runtime.Object{
				"secrets": &corev1.SecretList{
					Items: []corev1.Secret{
						func() corev1.Secret {
							secret := unitinputs.CephRBDMirrorSecret1.DeepCopy()
							secret.Data["token"] = []byte("another-token")
							return *secret
						}(),
						func() corev1.Secret {
							secret := unitinputs.CephRBDMirrorSecret2.DeepCopy()
							secret.Data["token"] = []byte("another-token")
							return *secret
						}(),
					},
				},
			},
			stateChanged: true,
		},
		{
			name: "update rbd",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				mc := unitinputs.CephDeployEnsureRbdMirror.DeepCopy()
				mc.Spec.RBDMirror.Count = 2
				return mc
			}(),
			inputResources: map[string]runtime.Object{
				"secrets":        unitinputs.CephRBDSecretList.DeepCopy(),
				"cephrbdmirrors": unitinputs.CephRBDMirrorsListReady.DeepCopy(),
			},
			expectedResources: map[string]runtime.Object{
				"cephrbdmirrors": &cephv1.CephRBDMirrorList{Items: []cephv1.CephRBDMirror{unitinputs.CephRBDMirrorUpdatedReady}},
			},
			stateChanged: true,
		},
		{
			name: "failed to update rbd",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				mc := unitinputs.CephDeployEnsureRbdMirror.DeepCopy()
				mc.Spec.RBDMirror.Count = 2
				return mc
			}(),
			inputResources: map[string]runtime.Object{
				"secrets":        unitinputs.CephRBDSecretList.DeepCopy(),
				"cephrbdmirrors": unitinputs.CephRBDMirrorsListReady.DeepCopy(),
			},
			apiErrors: map[string]error{
				"update-cephrbdmirrors": errors.New("cephrbdmirrors update failed"),
			},
			expectedError: "failed to update CephRBDMirror: cephrbdmirrors update failed",
		},
		{
			name:    "nothing to do",
			cephDpl: &unitinputs.CephDeployEnsureRbdMirror,
			inputResources: map[string]runtime.Object{
				"secrets":        unitinputs.CephRBDSecretList.DeepCopy(),
				"cephrbdmirrors": unitinputs.CephRBDMirrorsListReady.DeepCopy(),
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := fakeDeploymentConfig(&deployConfig{cephDpl: test.cephDpl}, nil)
			faketestclients.FakeReaction(c.api.Kubeclientset.CoreV1(), "list", []string{"secrets"}, test.inputResources, nil)
			faketestclients.FakeReaction(c.api.Kubeclientset.CoreV1(), "get", []string{"secrets"}, test.inputResources, test.apiErrors)
			faketestclients.FakeReaction(c.api.Kubeclientset.CoreV1(), "create", []string{"secrets"}, test.inputResources, test.apiErrors)
			faketestclients.FakeReaction(c.api.Kubeclientset.CoreV1(), "update", []string{"secrets"}, test.inputResources, test.apiErrors)
			faketestclients.FakeReaction(c.api.Kubeclientset.CoreV1(), "delete-collection", []string{"secrets"}, test.inputResources, test.apiErrors)
			faketestclients.FakeReaction(c.api.Rookclientset, "list", []string{"cephrbdmirrors"}, test.inputResources, test.apiErrors)
			faketestclients.FakeReaction(c.api.Rookclientset, "create", []string{"cephrbdmirrors"}, test.inputResources, test.apiErrors)
			faketestclients.FakeReaction(c.api.Rookclientset, "update", []string{"cephrbdmirrors"}, test.inputResources, test.apiErrors)
			faketestclients.FakeReaction(c.api.Rookclientset, "delete", []string{"cephrbdmirrors"}, test.inputResources, test.apiErrors)
			test.expectedResources = faketestclients.PrepareExpectedResources(test.inputResources, test.expectedResources)

			changed, err := c.ensureRBDMirroring()
			if test.expectedError != "" {
				assert.NotNil(t, err)
				assert.Equal(t, test.expectedError, err.Error())
			} else {
				assert.Nil(t, err)
			}
			assert.Equal(t, test.expectedResources, test.inputResources)
			assert.Equal(t, test.stateChanged, changed)
			faketestclients.CleanupFakeClientReactions(c.api.Kubeclientset.CoreV1())
			faketestclients.CleanupFakeClientReactions(c.api.Rookclientset)
		})
	}
}

func TestEnsureRBDSecrets(t *testing.T) {
	tests := []struct {
		name           string
		cephDpl        *cephlcmv1alpha1.CephDeployment
		presentSecrets *corev1.SecretList
		finalSecrets   *corev1.SecretList
		apiErrors      map[string]error
		stateChanged   bool
		expectedError  string
	}{
		{
			name: "no peers speciied - cleanup secrets",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				mc := unitinputs.CephDeployEnsureRbdMirror.DeepCopy()
				mc.Spec.RBDMirror.Peers = nil
				return mc
			}(),
			presentSecrets: unitinputs.CephRBDSecretList.DeepCopy(),
			finalSecrets:   &unitinputs.SecretsListEmpty,
			stateChanged:   true,
		},
		{
			name: "no peers speciied - cleanup secrets failed",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				mc := unitinputs.CephDeployEnsureRbdMirror.DeepCopy()
				mc.Spec.RBDMirror.Peers = nil
				return mc
			}(),
			presentSecrets: unitinputs.CephRBDSecretList.DeepCopy(),
			apiErrors:      map[string]error{"delete-collection-secrets": errors.New("secret delete failed")},
			expectedError:  "failed to remove rbd secrets: secret delete failed",
		},
		{
			name:           "create new secrets",
			cephDpl:        &unitinputs.CephDeployEnsureRbdMirror,
			presentSecrets: unitinputs.SecretsListEmpty.DeepCopy(),
			finalSecrets:   &unitinputs.CephRBDSecretList,
			stateChanged:   true,
		},
		{
			name:           "create new secrets failed",
			cephDpl:        &unitinputs.CephDeployEnsureRbdMirror,
			presentSecrets: unitinputs.SecretsListEmpty.DeepCopy(),
			apiErrors:      map[string]error{"create-secrets": errors.New("secret create failed")},
			expectedError:  "failed to create rbd-mirror-token-mirror1-pool-1 secret: secret create failed",
		},
		{
			name: "update present secrets",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				mc := unitinputs.CephDeployEnsureRbdMirror.DeepCopy()
				peer := mc.Spec.RBDMirror.Peers[0]
				peer.Token = "another-token"
				mc.Spec.RBDMirror.Peers[0] = peer
				return mc
			}(),
			presentSecrets: &corev1.SecretList{
				Items: []corev1.Secret{*unitinputs.CephRBDMirrorSecret1.DeepCopy(), *unitinputs.CephRBDMirrorSecret2.DeepCopy()},
			},
			finalSecrets: &corev1.SecretList{Items: []corev1.Secret{
				func() corev1.Secret {
					secret := unitinputs.CephRBDMirrorSecret1.DeepCopy()
					secret.Data["token"] = []byte("another-token")
					return *secret
				}(),
				func() corev1.Secret {
					secret := unitinputs.CephRBDMirrorSecret2.DeepCopy()
					secret.Data["token"] = []byte("another-token")
					return *secret
				}(),
			}},
			stateChanged: true,
		},
		{
			name: "update present secrets failed",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				mc := unitinputs.CephDeployEnsureRbdMirror.DeepCopy()
				peer := mc.Spec.RBDMirror.Peers[0]
				peer.Token = "another-token"
				mc.Spec.RBDMirror.Peers[0] = peer
				return mc
			}(),
			presentSecrets: unitinputs.CephRBDSecretList.DeepCopy(),
			apiErrors:      map[string]error{"update-secrets": errors.New("secret update failed")},
			expectedError:  "failed to update rbd-mirror-token-mirror1-pool-1 secret: secret update failed",
		},
		{
			name:           "nothing to do",
			cephDpl:        &unitinputs.CephDeployEnsureRbdMirror,
			presentSecrets: unitinputs.CephRBDSecretList.DeepCopy(),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := fakeDeploymentConfig(&deployConfig{cephDpl: test.cephDpl}, nil)
			inputResources := map[string]runtime.Object{"secrets": test.presentSecrets}
			if test.finalSecrets == nil {
				test.finalSecrets = test.presentSecrets.DeepCopy()
			}
			faketestclients.FakeReaction(c.api.Kubeclientset.CoreV1(), "list", []string{"secrets"}, inputResources, nil)
			faketestclients.FakeReaction(c.api.Kubeclientset.CoreV1(), "get", []string{"secrets"}, inputResources, test.apiErrors)
			faketestclients.FakeReaction(c.api.Kubeclientset.CoreV1(), "create", []string{"secrets"}, inputResources, test.apiErrors)
			faketestclients.FakeReaction(c.api.Kubeclientset.CoreV1(), "update", []string{"secrets"}, inputResources, test.apiErrors)
			faketestclients.FakeReaction(c.api.Kubeclientset.CoreV1(), "delete-collection", []string{"secrets"}, inputResources, test.apiErrors)

			changed, err := c.ensureRBDSecrets()
			if test.expectedError != "" {
				assert.NotNil(t, err)
				assert.Equal(t, test.expectedError, err.Error())
			} else {
				assert.Nil(t, err)
			}
			assert.Equal(t, map[string]runtime.Object{"secrets": test.finalSecrets}, inputResources)
			assert.Equal(t, test.stateChanged, changed)
			faketestclients.CleanupFakeClientReactions(c.api.Kubeclientset.CoreV1())
		})
	}
}

func TestDeleteRBDMirroring(t *testing.T) {
	cephDpl := &unitinputs.BaseCephDeployment
	initialResources := map[string]runtime.Object{
		"secrets":        unitinputs.CephRBDSecretList.DeepCopy(),
		"cephrbdmirrors": unitinputs.CephRBDMirrorsList.DeepCopy(),
	}
	tests := []struct {
		name           string
		inputResources map[string]runtime.Object
		apiErrors      map[string]error
		removed        bool
		expectedError  string
	}{
		{
			name:           "cleanup rbd failed",
			inputResources: initialResources,
			apiErrors:      map[string]error{"delete-cephrbdmirrors": errors.New("cephrbdmirrors failed to delete")},
			expectedError:  "failed to remove CephRBDMirroring: cephrbdmirrors failed to delete",
		},
		{
			name:           "cleanup secrets failed",
			inputResources: initialResources,
			apiErrors:      map[string]error{"delete-collection-secrets": errors.New("secret failed to delete")},
			expectedError:  "failed to remove rbd secrets: secret failed to delete",
		},
		{
			name:           "cleanup rbd in progress",
			inputResources: initialResources,
		},
		{
			name:           "cleanup rbd is finished",
			inputResources: initialResources,
			removed:        true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := fakeDeploymentConfig(&deployConfig{cephDpl: cephDpl}, nil)
			faketestclients.FakeReaction(c.api.Rookclientset, "delete", []string{"cephrbdmirrors"}, test.inputResources, test.apiErrors)
			faketestclients.FakeReaction(c.api.Kubeclientset.CoreV1(), "list", []string{"secrets"}, test.inputResources, nil)
			faketestclients.FakeReaction(c.api.Kubeclientset.CoreV1(), "delete-collection", []string{"secrets"}, test.inputResources, test.apiErrors)

			done, err := c.deleteRBDMirroring()
			if test.expectedError != "" {
				assert.NotNil(t, err)
				assert.Equal(t, test.expectedError, err.Error())
			} else {
				assert.Nil(t, err)
			}
			assert.Equal(t, test.removed, done)
			faketestclients.CleanupFakeClientReactions(c.api.Kubeclientset.CoreV1())
			faketestclients.CleanupFakeClientReactions(c.api.Rookclientset)
		})
	}
}
