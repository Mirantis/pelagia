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
	"strings"
	"testing"

	cephv1 "github.com/rook/rook/pkg/apis/ceph.rook.io/v1"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	cephlcmv1alpha1 "github.com/Mirantis/pelagia/pkg/apis/ceph.pelagia.lcm/v1alpha1"
	lcmcommon "github.com/Mirantis/pelagia/pkg/common"
	faketestclients "github.com/Mirantis/pelagia/test/unit/clients"
	unitinputs "github.com/Mirantis/pelagia/test/unit/inputs"
)

var multisiteResources = []string{"cephobjectstores", "cephobjectrealms", "cephobjectzonegroups", "cephobjectzones"}

func TestEnsureRgwMultiSite(t *testing.T) {
	multisiteNoResources := unitinputs.CephDeployMultisiteRgw.DeepCopy()
	multisiteNoResources.Spec.ObjectStorage.MultiSite.Realms = nil
	multisiteNoResources.Spec.ObjectStorage.MultiSite.Zones = nil
	multisiteNoResources.Spec.ObjectStorage.MultiSite.ZoneGroups = nil
	tests := []struct {
		name           string
		cephDpl        *cephlcmv1alpha1.CephDeployment
		stateChanged   bool
		inputResources map[string]runtime.Object
		apiErrors      map[string]error
		expectedError  string
	}{
		{
			name:           "failed to ensure realms",
			cephDpl:        multisiteNoResources,
			inputResources: map[string]runtime.Object{},
			expectedError:  "failed to ensure realms: failed to get list CephObjectRealms in 'rook-ceph' namespace: failed to list cephobjectrealms",
		},
		{
			name:    "failed to ensure zonegroups",
			cephDpl: multisiteNoResources,
			inputResources: map[string]runtime.Object{
				"cephobjectrealms":     &cephv1.CephObjectRealmList{},
				"cephobjectzonegroups": &cephv1.CephObjectZoneGroupList{},
			},
			expectedError: "failed to ensure zone groups: failed to get list CephObjectZones in 'rook-ceph' namespace: failed to list cephobjectzones",
		},
		{
			name:    "failed to ensure zones",
			cephDpl: multisiteNoResources,
			inputResources: map[string]runtime.Object{
				"cephobjectrealms":     &cephv1.CephObjectRealmList{},
				"cephobjectzonegroups": &cephv1.CephObjectZoneGroupList{},
				"cephobjectzones":      &cephv1.CephObjectZoneList{},
			},
			expectedError: "failed to ensure zones: failed to check zones in use: failed to list cephobjectstores",
		},
		{
			name:    "no changes",
			cephDpl: multisiteNoResources,
			inputResources: map[string]runtime.Object{
				"cephobjectrealms":     &cephv1.CephObjectRealmList{},
				"cephobjectzonegroups": &cephv1.CephObjectZoneGroupList{},
				"cephobjectzones":      &cephv1.CephObjectZoneList{},
				"cephobjectstores":     &cephv1.CephObjectStoreList{},
			},
		},
		{
			name:    "changes for realms",
			cephDpl: multisiteNoResources,
			inputResources: map[string]runtime.Object{
				"secrets": &corev1.SecretList{},
				"cephobjectrealms": &cephv1.CephObjectRealmList{
					Items: []cephv1.CephObjectRealm{*unitinputs.RgwMultisiteMasterPullRealm1.DeepCopy()},
				},
				"cephobjectzonegroups": &cephv1.CephObjectZoneGroupList{},
				"cephobjectzones":      &cephv1.CephObjectZoneList{},
				"cephobjectstores":     &cephv1.CephObjectStoreList{},
			},
			stateChanged: true,
		},
		{
			name:    "changes for zonegroups",
			cephDpl: multisiteNoResources,
			inputResources: map[string]runtime.Object{
				"cephobjectrealms": &cephv1.CephObjectRealmList{},
				"cephobjectzonegroups": &cephv1.CephObjectZoneGroupList{
					Items: []cephv1.CephObjectZoneGroup{*unitinputs.RgwMultisiteMasterZoneGroup1.DeepCopy()},
				},
				"cephobjectzones":  &cephv1.CephObjectZoneList{},
				"cephobjectstores": &cephv1.CephObjectStoreList{},
			},
			stateChanged: true,
		},
		{
			name:    "changes for zones",
			cephDpl: multisiteNoResources,
			inputResources: map[string]runtime.Object{
				"cephobjectrealms":     &cephv1.CephObjectRealmList{},
				"cephobjectzonegroups": &cephv1.CephObjectZoneGroupList{},
				"cephobjectzones": &cephv1.CephObjectZoneList{
					Items: []cephv1.CephObjectZone{*unitinputs.RgwMultisiteSecondaryZone1.DeepCopy()},
				},
				"cephobjectstores": &cephv1.CephObjectStoreList{},
			},
			stateChanged: true,
		},
		{
			name:    "changes for all objects",
			cephDpl: multisiteNoResources,
			inputResources: map[string]runtime.Object{
				"secrets": &corev1.SecretList{},
				"cephobjectrealms": &cephv1.CephObjectRealmList{
					Items: []cephv1.CephObjectRealm{*unitinputs.RgwMultisiteMasterPullRealm1.DeepCopy()},
				},
				"cephobjectzonegroups": &cephv1.CephObjectZoneGroupList{
					Items: []cephv1.CephObjectZoneGroup{*unitinputs.RgwMultisiteMasterZoneGroup1.DeepCopy()},
				},
				"cephobjectzones": &cephv1.CephObjectZoneList{
					Items: []cephv1.CephObjectZone{*unitinputs.RgwMultisiteSecondaryZone1.DeepCopy()},
				},
				"cephobjectstores": &cephv1.CephObjectStoreList{},
			},
			stateChanged: true,
		},
	}
	oldFunc := lcmcommon.RunPodCommandWithValidation
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := fakeDeploymentConfig(&deployConfig{cephDpl: test.cephDpl}, nil)
			lcmcommon.RunPodCommandWithValidation = func(e lcmcommon.ExecConfig) (string, string, error) {
				if test.apiErrors["cli"] != nil {
					return "", "", test.apiErrors["cli"]
				}
				switch e.Command {
				case "radosgw-admin realm list":
					return "{}", "", nil
				case "radosgw-admin zonegroup list":
					return "{}", "", nil
				}
				return "", "", errors.New("unexpected command call: " + e.Command)
			}
			faketestclients.FakeReaction(c.api.Kubeclientset.CoreV1(), "get", []string{"secrets"}, test.inputResources, nil)
			faketestclients.FakeReaction(c.api.Kubeclientset.CoreV1(), "delete", []string{"secrets"}, test.inputResources, nil)
			faketestclients.FakeReaction(c.api.Rookclientset, "list", multisiteResources, test.inputResources, nil)
			faketestclients.FakeReaction(c.api.Rookclientset, "delete", []string{"cephobjectrealms", "cephobjectzonegroups", "cephobjectzones"}, test.inputResources, nil)

			stateChanged, err := c.ensureRgwMultiSite()
			if test.expectedError != "" {
				assert.Equal(t, false, stateChanged)
				assert.NotNil(t, err)
				assert.Equal(t, test.expectedError, err.Error())
			} else {
				assert.Nil(t, err)
				assert.Equal(t, test.stateChanged, stateChanged)
			}

			faketestclients.CleanupFakeClientReactions(c.api.Kubeclientset.CoreV1())
			faketestclients.CleanupFakeClientReactions(c.api.Rookclientset)
		})
	}
	lcmcommon.RunPodCommandWithValidation = oldFunc
}

func TestEnsureRealms(t *testing.T) {
	tests := []struct {
		name           string
		cephDpl        *cephlcmv1alpha1.CephDeployment
		stateChanged   bool
		inputResources map[string]runtime.Object
		expectedRealms *cephv1.CephObjectRealmList
		apiErrors      map[string]error
		expectedError  string
	}{
		{
			name:           "failed to list realms",
			cephDpl:        unitinputs.CephDeployMultisiteRgw.DeepCopy(),
			inputResources: map[string]runtime.Object{},
			expectedError:  "failed to get list CephObjectRealms in 'rook-ceph' namespace: failed to list cephobjectrealms",
		},
		{
			name:    "failed to list zonegroups",
			cephDpl: unitinputs.CephDeployMultisiteRgw.DeepCopy(),
			inputResources: map[string]runtime.Object{
				"cephobjectrealms": &cephv1.CephObjectRealmList{},
			},
			expectedRealms: &cephv1.CephObjectRealmList{},
			expectedError:  "failed to get list CephObjectZoneGroups in 'rook-ceph' namespace: failed to list cephobjectzonegroups",
		},
		{
			name:    "nothing to do - realms are aligned",
			cephDpl: unitinputs.CephDeployMultisiteRgw.DeepCopy(),
			inputResources: map[string]runtime.Object{
				"secrets": &corev1.SecretList{
					Items: []corev1.Secret{*unitinputs.MultisiteRealmSecret.DeepCopy()},
				},
				"cephobjectrealms": &cephv1.CephObjectRealmList{
					Items: []cephv1.CephObjectRealm{*unitinputs.RgwMultisiteMasterPullRealm1.DeepCopy()},
				},
				"cephobjectzonegroups": &cephv1.CephObjectZoneGroupList{},
			},
			expectedRealms: &cephv1.CephObjectRealmList{
				Items: []cephv1.CephObjectRealm{unitinputs.RgwMultisiteMasterPullRealm1},
			},
		},
		{
			name: "nothing to do - no realms in spec",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				mc := unitinputs.CephDeployMultisiteRgw.DeepCopy()
				mc.Spec.ObjectStorage.MultiSite.Realms = nil
				return mc
			}(),
			inputResources: map[string]runtime.Object{
				"cephobjectzonegroups": &cephv1.CephObjectZoneGroupList{},
				"cephobjectrealms":     &cephv1.CephObjectRealmList{},
			},
			expectedRealms: &cephv1.CephObjectRealmList{},
		},
		{
			name:    "failed to create secret for realm",
			cephDpl: unitinputs.CephDeployMultisiteRgw.DeepCopy(),
			inputResources: map[string]runtime.Object{
				"secrets":              &corev1.SecretList{},
				"cephobjectzonegroups": &cephv1.CephObjectZoneGroupList{},
				"cephobjectrealms":     &cephv1.CephObjectRealmList{},
			},
			expectedRealms: &cephv1.CephObjectRealmList{},
			apiErrors: map[string]error{
				"create-secrets": errors.New("failed to create secret"),
			},
			expectedError: "failed to create Secret 'rook-ceph/realm1-keys': failed to create secret",
		},
		{
			name:    "failed to create realm",
			cephDpl: unitinputs.CephDeployMultisiteRgw.DeepCopy(),
			inputResources: map[string]runtime.Object{
				"secrets":              &corev1.SecretList{},
				"cephobjectzonegroups": &cephv1.CephObjectZoneGroupList{},
				"cephobjectrealms":     &cephv1.CephObjectRealmList{},
			},
			expectedRealms: &cephv1.CephObjectRealmList{},
			apiErrors: map[string]error{
				"create-cephobjectrealms": errors.New("failed to create realm"),
			},
			expectedError: "failed to create CephObjectRealm 'rook-ceph/realm1': failed to create realm",
		},
		{
			name:    "create pull realm ok",
			cephDpl: unitinputs.CephDeployMultisiteRgw.DeepCopy(),
			inputResources: map[string]runtime.Object{
				"secrets":              &corev1.SecretList{},
				"cephobjectzonegroups": &cephv1.CephObjectZoneGroupList{},
				"cephobjectrealms":     &cephv1.CephObjectRealmList{},
			},
			expectedRealms: &cephv1.CephObjectRealmList{
				Items: []cephv1.CephObjectRealm{unitinputs.RgwMultisiteMasterPullRealm1},
			},
			stateChanged: true,
		},
		{
			name: "create master realm ok",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				mc := unitinputs.CephDeployMultisiteRgw.DeepCopy()
				realm := mc.Spec.ObjectStorage.MultiSite.Realms[0]
				realm.Pull = nil
				mc.Spec.ObjectStorage.MultiSite.Realms[0] = realm
				return mc
			}(),
			inputResources: map[string]runtime.Object{
				"cephobjectzonegroups": &cephv1.CephObjectZoneGroupList{},
				"cephobjectrealms":     &cephv1.CephObjectRealmList{},
			},
			expectedRealms: &cephv1.CephObjectRealmList{
				Items: []cephv1.CephObjectRealm{
					func() cephv1.CephObjectRealm {
						realm := unitinputs.RgwMultisiteMasterRealm1.DeepCopy()
						realm.Spec.DefaultRealm = true
						return *realm
					}(),
				},
			},
			stateChanged: true,
		},
		{
			name:    "failed to check secret",
			cephDpl: unitinputs.CephDeployMultisiteRgw.DeepCopy(),
			inputResources: map[string]runtime.Object{
				"secrets": &corev1.SecretList{},
				"cephobjectrealms": &cephv1.CephObjectRealmList{
					Items: []cephv1.CephObjectRealm{*unitinputs.RgwMultisiteMasterPullRealm1.DeepCopy()},
				},
				"cephobjectzonegroups": &cephv1.CephObjectZoneGroupList{},
			},
			apiErrors: map[string]error{
				"get-secrets": errors.New("failed to get secret"),
			},
			expectedRealms: &cephv1.CephObjectRealmList{
				Items: []cephv1.CephObjectRealm{unitinputs.RgwMultisiteMasterPullRealm1},
			},
			expectedError: "failed to get secret 'rook-ceph/realm1-keys' for CephObjectRealm 'realm1': failed to get secret",
		},
		{
			name:    "create realm secret, but realm is present already",
			cephDpl: unitinputs.CephDeployMultisiteRgw.DeepCopy(),
			inputResources: map[string]runtime.Object{
				"secrets": &corev1.SecretList{},
				"cephobjectrealms": &cephv1.CephObjectRealmList{
					Items: []cephv1.CephObjectRealm{*unitinputs.RgwMultisiteMasterPullRealm1.DeepCopy()},
				},
				"cephobjectzonegroups": &cephv1.CephObjectZoneGroupList{},
			},
			expectedRealms: &cephv1.CephObjectRealmList{
				Items: []cephv1.CephObjectRealm{unitinputs.RgwMultisiteMasterPullRealm1},
			},
		},
		{
			name:    "create realm secret failed, but realm is present already",
			cephDpl: unitinputs.CephDeployMultisiteRgw.DeepCopy(),
			inputResources: map[string]runtime.Object{
				"secrets": &corev1.SecretList{},
				"cephobjectrealms": &cephv1.CephObjectRealmList{
					Items: []cephv1.CephObjectRealm{*unitinputs.RgwMultisiteMasterPullRealm1.DeepCopy()},
				},
				"cephobjectzonegroups": &cephv1.CephObjectZoneGroupList{},
			},
			expectedRealms: &cephv1.CephObjectRealmList{
				Items: []cephv1.CephObjectRealm{unitinputs.RgwMultisiteMasterPullRealm1},
			},
			apiErrors: map[string]error{
				"create-secrets": errors.New("failed to create secret"),
			},
			expectedError: "failed to create Secret 'rook-ceph/realm1-keys': failed to create secret",
		},
		{
			name:    "update realm ok, but secret realm update failed",
			cephDpl: unitinputs.CephDeployMultisiteRgw.DeepCopy(),
			inputResources: map[string]runtime.Object{
				"secrets": &corev1.SecretList{
					Items: []corev1.Secret{
						func() corev1.Secret {
							secret := unitinputs.MultisiteRealmSecret.DeepCopy()
							secret.Data["secret-key"] = []byte("wrongkey")
							return *secret
						}(),
					},
				},
				"cephobjectrealms": &cephv1.CephObjectRealmList{
					Items: []cephv1.CephObjectRealm{*unitinputs.RgwMultisiteMasterRealm1.DeepCopy()},
				},
				"cephobjectzonegroups": &cephv1.CephObjectZoneGroupList{},
			},
			apiErrors: map[string]error{
				"update-secrets": errors.New("failed to update secret"),
			},
			expectedRealms: &cephv1.CephObjectRealmList{
				Items: []cephv1.CephObjectRealm{unitinputs.RgwMultisiteMasterPullRealm1},
			},
			expectedError: "failed to update Secret 'rook-ceph/realm1-keys': failed to update secret",
		},
		{
			name:    "update realm failed",
			cephDpl: unitinputs.CephDeployMultisiteRgw.DeepCopy(),
			inputResources: map[string]runtime.Object{
				"secrets": &corev1.SecretList{
					Items: []corev1.Secret{*unitinputs.MultisiteRealmSecret.DeepCopy()},
				},
				"cephobjectrealms": &cephv1.CephObjectRealmList{
					Items: []cephv1.CephObjectRealm{*unitinputs.RgwMultisiteMasterRealm1.DeepCopy()},
				},
				"cephobjectzonegroups": &cephv1.CephObjectZoneGroupList{},
			},
			apiErrors: map[string]error{
				"update-cephobjectrealms": errors.New("failed to update realm"),
			},
			expectedRealms: &cephv1.CephObjectRealmList{
				Items: []cephv1.CephObjectRealm{unitinputs.RgwMultisiteMasterRealm1},
			},
			expectedError: "failed to update CephObjectRealm 'rook-ceph/realm1': failed to update realm",
		},
		{
			name: "update master realm ok",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				mc := unitinputs.CephDeployMultisiteRgw.DeepCopy()
				realm := mc.Spec.ObjectStorage.MultiSite.Realms[0]
				realm.Pull = nil
				mc.Spec.ObjectStorage.MultiSite.Realms[0] = realm
				return mc
			}(),
			inputResources: map[string]runtime.Object{
				"cephobjectzonegroups": &cephv1.CephObjectZoneGroupList{},
				"cephobjectrealms": &cephv1.CephObjectRealmList{
					Items: []cephv1.CephObjectRealm{*unitinputs.RgwMultisiteMasterPullRealm1.DeepCopy()},
				},
			},
			expectedRealms: &cephv1.CephObjectRealmList{
				Items: []cephv1.CephObjectRealm{*unitinputs.RgwMultisiteMasterRealm1.DeepCopy()},
			},
			stateChanged: true,
		},
		{
			name: "delete secret for realm failed",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				mc := unitinputs.CephDeployMultisiteRgw.DeepCopy()
				mc.Spec.ObjectStorage.MultiSite.Realms = nil
				return mc
			}(),
			inputResources: map[string]runtime.Object{
				"secrets": &corev1.SecretList{
					Items: []corev1.Secret{*unitinputs.MultisiteRealmSecret.DeepCopy()},
				},
				"cephobjectrealms": &cephv1.CephObjectRealmList{
					Items: []cephv1.CephObjectRealm{*unitinputs.RgwMultisiteMasterPullRealm1.DeepCopy()},
				},
				"cephobjectzonegroups": &cephv1.CephObjectZoneGroupList{},
			},
			apiErrors: map[string]error{
				"delete-secrets": errors.New("failed to delete secret"),
			},
			expectedRealms: &cephv1.CephObjectRealmList{
				Items: []cephv1.CephObjectRealm{unitinputs.RgwMultisiteMasterPullRealm1},
			},
			expectedError: "failed to delete Secret 'rook-ceph/realm1-keys' for CephObjectRealm 'rook-ceph/realm1': failed to delete secret",
		},
		{
			name: "delete realm failed",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				mc := unitinputs.CephDeployMultisiteRgw.DeepCopy()
				mc.Spec.ObjectStorage.MultiSite.Realms = nil
				return mc
			}(),
			inputResources: map[string]runtime.Object{
				"secrets": &corev1.SecretList{
					Items: []corev1.Secret{*unitinputs.MultisiteRealmSecret.DeepCopy()},
				},
				"cephobjectrealms": &cephv1.CephObjectRealmList{
					Items: []cephv1.CephObjectRealm{*unitinputs.RgwMultisiteMasterPullRealm1.DeepCopy()},
				},
				"cephobjectzonegroups": &cephv1.CephObjectZoneGroupList{},
			},
			apiErrors: map[string]error{
				"delete-cephobjectrealms": errors.New("failed to delete realm"),
			},
			expectedRealms: &cephv1.CephObjectRealmList{
				Items: []cephv1.CephObjectRealm{unitinputs.RgwMultisiteMasterPullRealm1},
			},
			expectedError: "failed to delete CephObjectRealm 'rook-ceph/realm1': failed to delete realm",
		},
		{
			name: "delete realm ok",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				mc := unitinputs.CephDeployMultisiteRgw.DeepCopy()
				mc.Spec.ObjectStorage.MultiSite.Realms = nil
				return mc
			}(),
			inputResources: map[string]runtime.Object{
				"secrets": &corev1.SecretList{
					Items: []corev1.Secret{*unitinputs.MultisiteRealmSecret.DeepCopy()},
				},
				"cephobjectrealms": &cephv1.CephObjectRealmList{
					Items: []cephv1.CephObjectRealm{*unitinputs.RgwMultisiteMasterPullRealm1.DeepCopy()},
				},
				"cephobjectzonegroups": &cephv1.CephObjectZoneGroupList{},
			},
			expectedRealms: &cephv1.CephObjectRealmList{
				Items: []cephv1.CephObjectRealm{},
			},
			stateChanged: true,
		},
		{
			name: "delete realm skipped - in use",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				mc := unitinputs.CephDeployMultisiteRgw.DeepCopy()
				mc.Spec.ObjectStorage.MultiSite.Realms = nil
				return mc
			}(),
			inputResources: map[string]runtime.Object{
				"secrets": &corev1.SecretList{
					Items: []corev1.Secret{*unitinputs.MultisiteRealmSecret.DeepCopy()},
				},
				"cephobjectrealms": &cephv1.CephObjectRealmList{
					Items: []cephv1.CephObjectRealm{*unitinputs.RgwMultisiteMasterPullRealm1.DeepCopy()},
				},
				"cephobjectzonegroups": &cephv1.CephObjectZoneGroupList{
					Items: []cephv1.CephObjectZoneGroup{*unitinputs.RgwMultisiteMasterZoneGroup1.DeepCopy()},
				},
			},
			expectedRealms: &cephv1.CephObjectRealmList{
				Items: []cephv1.CephObjectRealm{unitinputs.RgwMultisiteMasterPullRealm1},
			},
		},
	}
	oldFunc := lcmcommon.RunPodCommandWithValidation
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := fakeDeploymentConfig(&deployConfig{cephDpl: test.cephDpl}, nil)
			lcmcommon.RunPodCommandWithValidation = func(e lcmcommon.ExecConfig) (string, string, error) {
				if e.Command == "radosgw-admin realm list" {
					return "{}", "", nil
				}
				return "", "", errors.New("unexpected command call: " + e.Command)
			}
			faketestclients.FakeReaction(c.api.Kubeclientset.CoreV1(), "get", []string{"secrets"}, test.inputResources, test.apiErrors)
			faketestclients.FakeReaction(c.api.Kubeclientset.CoreV1(), "create", []string{"secrets"}, test.inputResources, test.apiErrors)
			faketestclients.FakeReaction(c.api.Kubeclientset.CoreV1(), "update", []string{"secrets"}, test.inputResources, test.apiErrors)
			faketestclients.FakeReaction(c.api.Kubeclientset.CoreV1(), "delete", []string{"secrets"}, test.inputResources, test.apiErrors)
			faketestclients.FakeReaction(c.api.Rookclientset, "list", []string{"cephobjectrealms", "cephobjectzonegroups"}, test.inputResources, nil)
			faketestclients.FakeReaction(c.api.Rookclientset, "create", []string{"cephobjectrealms"}, test.inputResources, test.apiErrors)
			faketestclients.FakeReaction(c.api.Rookclientset, "update", []string{"cephobjectrealms"}, test.inputResources, test.apiErrors)
			faketestclients.FakeReaction(c.api.Rookclientset, "delete", []string{"cephobjectrealms"}, test.inputResources, test.apiErrors)

			stateChanged, err := c.ensureRealms()
			if test.expectedError != "" {
				assert.Equal(t, false, stateChanged)
				assert.NotNil(t, err)
				assert.Equal(t, test.expectedError, err.Error())
			} else {
				assert.Nil(t, err)
				assert.Equal(t, test.stateChanged, stateChanged)
			}
			if test.inputResources["cephobjectrealms"] != nil {
				assert.Equal(t, test.expectedRealms, test.inputResources["cephobjectrealms"].(*cephv1.CephObjectRealmList))
			}

			faketestclients.CleanupFakeClientReactions(c.api.Kubeclientset.CoreV1())
			faketestclients.CleanupFakeClientReactions(c.api.Rookclientset)
		})
	}
	lcmcommon.RunPodCommandWithValidation = oldFunc
}

func TestEnsureZoneGroups(t *testing.T) {
	tests := []struct {
		name               string
		cephDpl            *cephlcmv1alpha1.CephDeployment
		stateChanged       bool
		inputResources     map[string]runtime.Object
		expectedZoneGroups *cephv1.CephObjectZoneGroupList
		apiErrors          map[string]error
		expectedError      string
	}{
		{
			name:           "failed to list zonegroups",
			cephDpl:        unitinputs.CephDeployMultisiteRgw.DeepCopy(),
			inputResources: map[string]runtime.Object{},
			expectedError:  "failed to get list CephObjectZoneGroups in 'rook-ceph' namespace: failed to list cephobjectzonegroups",
		},
		{
			name:    "failed to list zones",
			cephDpl: unitinputs.CephDeployMultisiteRgw.DeepCopy(),
			inputResources: map[string]runtime.Object{
				"cephobjectzonegroups": &cephv1.CephObjectZoneGroupList{},
			},
			expectedZoneGroups: &cephv1.CephObjectZoneGroupList{},
			expectedError:      "failed to get list CephObjectZones in 'rook-ceph' namespace: failed to list cephobjectzones",
		},
		{
			name:    "nothing to do - zonegroups are aligned",
			cephDpl: unitinputs.CephDeployMultisiteRgw.DeepCopy(),
			inputResources: map[string]runtime.Object{
				"cephobjectzonegroups": &cephv1.CephObjectZoneGroupList{
					Items: []cephv1.CephObjectZoneGroup{*unitinputs.RgwMultisiteMasterZoneGroup1.DeepCopy()},
				},
				"cephobjectzones": &cephv1.CephObjectZoneList{},
			},
			expectedZoneGroups: &cephv1.CephObjectZoneGroupList{
				Items: []cephv1.CephObjectZoneGroup{unitinputs.RgwMultisiteMasterZoneGroup1},
			},
		},
		{
			name: "nothing to do - no zonegroups in spec",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				mc := unitinputs.CephDeployMultisiteRgw.DeepCopy()
				mc.Spec.ObjectStorage.MultiSite.ZoneGroups = nil
				return mc
			}(),
			inputResources: map[string]runtime.Object{
				"cephobjectzonegroups": &cephv1.CephObjectZoneGroupList{},
				"cephobjectzones":      &cephv1.CephObjectZoneList{},
			},
			expectedZoneGroups: &cephv1.CephObjectZoneGroupList{},
		},
		{
			name:    "failed to create zonegroup",
			cephDpl: unitinputs.CephDeployMultisiteRgw.DeepCopy(),
			inputResources: map[string]runtime.Object{
				"cephobjectzones":      &cephv1.CephObjectZoneList{},
				"cephobjectzonegroups": &cephv1.CephObjectZoneGroupList{},
			},
			expectedZoneGroups: &cephv1.CephObjectZoneGroupList{},
			apiErrors: map[string]error{
				"create-cephobjectzonegroups": errors.New("failed to create zonegroup"),
			},
			expectedError: "failed to create CephObjectZoneGroup 'rook-ceph/zonegroup1': failed to create zonegroup",
		},
		{
			name:    "create zonegroup ok",
			cephDpl: unitinputs.CephDeployMultisiteRgw.DeepCopy(),
			inputResources: map[string]runtime.Object{
				"cephobjectzones":      &cephv1.CephObjectZoneList{},
				"cephobjectzonegroups": &cephv1.CephObjectZoneGroupList{},
			},
			expectedZoneGroups: &cephv1.CephObjectZoneGroupList{
				Items: []cephv1.CephObjectZoneGroup{unitinputs.RgwMultisiteMasterZoneGroup1},
			},
			stateChanged: true,
		},
		{
			name: "delete zonegroup failed",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				mc := unitinputs.CephDeployMultisiteRgw.DeepCopy()
				mc.Spec.ObjectStorage.MultiSite.ZoneGroups = nil
				return mc
			}(),
			inputResources: map[string]runtime.Object{
				"cephobjectzonegroups": &cephv1.CephObjectZoneGroupList{
					Items: []cephv1.CephObjectZoneGroup{*unitinputs.RgwMultisiteMasterZoneGroup1.DeepCopy()},
				},
				"cephobjectzones": &cephv1.CephObjectZoneList{},
			},
			expectedZoneGroups: &cephv1.CephObjectZoneGroupList{
				Items: []cephv1.CephObjectZoneGroup{unitinputs.RgwMultisiteMasterZoneGroup1},
			},
			apiErrors: map[string]error{
				"delete-cephobjectzonegroups": errors.New("failed to delete zonegroup"),
			},
			expectedError: "failed to delete CephObjectZoneGroup 'rook-ceph/zonegroup1': failed to delete zonegroup",
		},
		{
			name: "delete zonegroup ok",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				mc := unitinputs.CephDeployMultisiteRgw.DeepCopy()
				mc.Spec.ObjectStorage.MultiSite.ZoneGroups = nil
				return mc
			}(),
			inputResources: map[string]runtime.Object{
				"cephobjectzonegroups": &cephv1.CephObjectZoneGroupList{
					Items: []cephv1.CephObjectZoneGroup{*unitinputs.RgwMultisiteMasterZoneGroup1.DeepCopy()},
				},
				"cephobjectzones": &cephv1.CephObjectZoneList{},
			},
			expectedZoneGroups: &cephv1.CephObjectZoneGroupList{Items: []cephv1.CephObjectZoneGroup{}},
			stateChanged:       true,
		},
		{
			name: "delete zone skipped",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				mc := unitinputs.CephDeployMultisiteRgw.DeepCopy()
				mc.Spec.ObjectStorage.MultiSite.ZoneGroups = nil
				return mc
			}(),
			inputResources: map[string]runtime.Object{
				"cephobjectzonegroups": &cephv1.CephObjectZoneGroupList{
					Items: []cephv1.CephObjectZoneGroup{*unitinputs.RgwMultisiteMasterZoneGroup1.DeepCopy()},
				},
				"cephobjectzones": &cephv1.CephObjectZoneList{
					Items: []cephv1.CephObjectZone{*unitinputs.RgwMultisiteSecondaryZone1.DeepCopy()},
				},
			},
			expectedZoneGroups: &cephv1.CephObjectZoneGroupList{
				Items: []cephv1.CephObjectZoneGroup{unitinputs.RgwMultisiteMasterZoneGroup1},
			},
		},
	}
	oldFunc := lcmcommon.RunPodCommandWithValidation
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := fakeDeploymentConfig(&deployConfig{cephDpl: test.cephDpl}, nil)
			lcmcommon.RunPodCommandWithValidation = func(e lcmcommon.ExecConfig) (string, string, error) {
				if e.Command == "radosgw-admin zonegroup list" {
					return "{}", "", nil
				}
				return "", "", errors.New("unexpected command call: " + e.Command)
			}
			faketestclients.FakeReaction(c.api.Rookclientset, "list", []string{"cephobjectzonegroups", "cephobjectzones"}, test.inputResources, nil)
			faketestclients.FakeReaction(c.api.Rookclientset, "create", []string{"cephobjectzonegroups"}, test.inputResources, test.apiErrors)
			faketestclients.FakeReaction(c.api.Rookclientset, "delete", []string{"cephobjectzonegroups"}, test.inputResources, test.apiErrors)

			stateChanged, err := c.ensureZoneGroups()
			if test.expectedError != "" {
				assert.Equal(t, false, stateChanged)
				assert.NotNil(t, err)
				assert.Equal(t, test.expectedError, err.Error())
			} else {
				assert.Nil(t, err)
				assert.Equal(t, test.stateChanged, stateChanged)
			}
			if test.inputResources["cephobjectzonegroups"] != nil {
				assert.Equal(t, test.expectedZoneGroups, test.inputResources["cephobjectzonegroups"].(*cephv1.CephObjectZoneGroupList))
			}

			faketestclients.CleanupFakeClientReactions(c.api.Rookclientset)
		})
	}
	lcmcommon.RunPodCommandWithValidation = oldFunc
}

func TestEnsureZones(t *testing.T) {
	correctZone := unitinputs.RgwMultisiteSecondaryZone1.DeepCopy()
	correctZone.Spec.DataPool.CrushRoot = "default"
	correctZone.Spec.MetadataPool.CrushRoot = "default"
	objectStore := unitinputs.CephObjectStoreWithZone.DeepCopy()
	objectStore.Spec.Zone.Name = "secondary-zone1"
	zoneWithExtSvcEndpoint := correctZone.DeepCopy()
	zoneWithExtSvcEndpoint.Spec.CustomEndpoints = []string{"http://192.168.100.150:80"}

	tests := []struct {
		name           string
		cephDpl        *cephlcmv1alpha1.CephDeployment
		stateChanged   bool
		inputResources map[string]runtime.Object
		expectedZones  *cephv1.CephObjectZoneList
		apiErrors      map[string]error
		expectedError  string
	}{
		{
			name:           "failed to list zones",
			cephDpl:        unitinputs.CephDeployMultisiteRgw.DeepCopy(),
			inputResources: map[string]runtime.Object{},
			expectedError:  "failed to get list CephObjectZones in 'rook-ceph' namespace: failed to list cephobjectzones",
		},
		{
			name:    "failed to list rgw",
			cephDpl: unitinputs.CephDeployMultisiteRgw.DeepCopy(),
			inputResources: map[string]runtime.Object{
				"cephobjectzones": &cephv1.CephObjectZoneList{},
			},
			expectedZones: &cephv1.CephObjectZoneList{},
			expectedError: "failed to check zones in use: failed to list cephobjectstores",
		},
		{
			name:    "nothing to do - zones are aligned",
			cephDpl: unitinputs.CephDeployMultisiteRgw.DeepCopy(),
			inputResources: map[string]runtime.Object{
				"cephobjectzones": &cephv1.CephObjectZoneList{
					Items: []cephv1.CephObjectZone{*correctZone.DeepCopy()},
				},
				"cephobjectstores": &cephv1.CephObjectStoreList{},
			},
			expectedZones: &cephv1.CephObjectZoneList{
				Items: []cephv1.CephObjectZone{*correctZone.DeepCopy()},
			},
		},
		{
			name: "nothing to do - no zones in spec",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				mc := unitinputs.CephDeployMultisiteRgw.DeepCopy()
				mc.Spec.ObjectStorage.MultiSite.Zones = nil
				return mc
			}(),
			inputResources: map[string]runtime.Object{
				"cephobjectzones":  &cephv1.CephObjectZoneList{},
				"cephobjectstores": &cephv1.CephObjectStoreList{},
			},
			expectedZones: &cephv1.CephObjectZoneList{},
		},
		{
			name:    "failed to create zone",
			cephDpl: unitinputs.CephDeployMultisiteRgw.DeepCopy(),
			inputResources: map[string]runtime.Object{
				"cephobjectzones":  &cephv1.CephObjectZoneList{},
				"cephobjectstores": &cephv1.CephObjectStoreList{},
			},
			expectedZones: &cephv1.CephObjectZoneList{},
			apiErrors: map[string]error{
				"create-cephobjectzones": errors.New("failed to create zone"),
			},
			expectedError: "failed to create CephObjectZone 'rook-ceph/secondary-zone1': failed to create zone",
		},
		{
			name:    "create zone ok",
			cephDpl: unitinputs.CephDeployMultisiteRgw.DeepCopy(),
			inputResources: map[string]runtime.Object{
				"cephobjectzones":  &cephv1.CephObjectZoneList{},
				"cephobjectstores": &cephv1.CephObjectStoreList{},
			},
			expectedZones: &cephv1.CephObjectZoneList{
				Items: []cephv1.CephObjectZone{*correctZone.DeepCopy()},
			},
			stateChanged: true,
		},
		{
			name:    "update zone failed",
			cephDpl: unitinputs.CephDeployMultisiteRgw.DeepCopy(),
			inputResources: map[string]runtime.Object{
				"cephobjectzones": &cephv1.CephObjectZoneList{
					Items: []cephv1.CephObjectZone{*unitinputs.RgwMultisiteSecondaryZone1.DeepCopy()},
				},
				"cephobjectstores": &cephv1.CephObjectStoreList{},
			},
			expectedZones: &cephv1.CephObjectZoneList{
				Items: []cephv1.CephObjectZone{*unitinputs.RgwMultisiteSecondaryZone1.DeepCopy()},
			},
			apiErrors: map[string]error{
				"update-cephobjectzones": errors.New("failed to update zone"),
			},
			expectedError: "failed to update CephObjectZone 'rook-ceph/secondary-zone1': failed to update zone",
		},
		{
			name:    "update zone ok",
			cephDpl: unitinputs.CephDeployMultisiteRgw.DeepCopy(),
			inputResources: map[string]runtime.Object{
				"cephobjectzones": &cephv1.CephObjectZoneList{
					Items: []cephv1.CephObjectZone{*unitinputs.RgwMultisiteSecondaryZone1.DeepCopy()},
				},
				"cephobjectstores": &cephv1.CephObjectStoreList{},
			},
			expectedZones: &cephv1.CephObjectZoneList{
				Items: []cephv1.CephObjectZone{*correctZone.DeepCopy()},
			},
			stateChanged: true,
		},
		{
			name: "delete zone failed",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				mc := unitinputs.CephDeployMultisiteRgw.DeepCopy()
				mc.Spec.ObjectStorage.MultiSite.Zones = nil
				return mc
			}(),
			inputResources: map[string]runtime.Object{
				"cephobjectzones": &cephv1.CephObjectZoneList{
					Items: []cephv1.CephObjectZone{*unitinputs.RgwMultisiteMasterZone1.DeepCopy()},
				},
				"cephobjectstores": &cephv1.CephObjectStoreList{},
			},
			expectedZones: &cephv1.CephObjectZoneList{
				Items: []cephv1.CephObjectZone{*unitinputs.RgwMultisiteMasterZone1.DeepCopy()},
			},
			apiErrors: map[string]error{
				"delete-cephobjectzones": errors.New("failed to delete zone"),
			},
			expectedError: "failed to delete CephObjectZone 'rook-ceph/zone1': failed to delete zone",
		},
		{
			name: "delete zone ok",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				mc := unitinputs.CephDeployMultisiteRgw.DeepCopy()
				mc.Spec.ObjectStorage.MultiSite.Zones = nil
				return mc
			}(),
			inputResources: map[string]runtime.Object{
				"cephobjectzones": &cephv1.CephObjectZoneList{
					Items: []cephv1.CephObjectZone{*unitinputs.RgwMultisiteMasterZone1.DeepCopy()},
				},
				"cephobjectstores": &cephv1.CephObjectStoreList{},
			},
			expectedZones: &cephv1.CephObjectZoneList{Items: []cephv1.CephObjectZone{}},
			stateChanged:  true,
		},
		{
			name: "delete zone skipped",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				mc := unitinputs.CephDeployMultisiteRgw.DeepCopy()
				mc.Spec.ObjectStorage.MultiSite.Zones = nil
				return mc
			}(),
			inputResources: map[string]runtime.Object{
				"cephobjectzones": &cephv1.CephObjectZoneList{
					Items: []cephv1.CephObjectZone{*unitinputs.RgwMultisiteMasterZone1.DeepCopy()},
				},
				"cephobjectstores": &cephv1.CephObjectStoreList{
					Items: []cephv1.CephObjectStore{*unitinputs.CephObjectStoreWithZone.DeepCopy()},
				},
			},
			expectedZones: &cephv1.CephObjectZoneList{
				Items: []cephv1.CephObjectZone{*unitinputs.RgwMultisiteMasterZone1.DeepCopy()},
			},
		},
		{
			name: "nothing to do - no endpoints, but old ingress present",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				mc := unitinputs.CephDeployMultisiteRgw.DeepCopy()
				mc.Spec.IngressConfig = unitinputs.CephIngressConfig.DeepCopy()
				return mc
			}(),
			inputResources: map[string]runtime.Object{
				"cephobjectzones": &cephv1.CephObjectZoneList{
					Items: []cephv1.CephObjectZone{*correctZone.DeepCopy()},
				},
				"cephobjectstores": &cephv1.CephObjectStoreList{
					Items: []cephv1.CephObjectStore{*objectStore},
				},
			},
			expectedZones: &cephv1.CephObjectZoneList{
				Items: []cephv1.CephObjectZone{*correctZone.DeepCopy()},
			},
		},
		{
			name: "nothing to do - no endpoints, but ingress config present",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				mc := unitinputs.CephDeployMultisiteRgw.DeepCopy()
				mc.Spec.IngressConfig = unitinputs.CephIngressConfig.DeepCopy()
				return mc
			}(),
			inputResources: map[string]runtime.Object{
				"cephobjectzones": &cephv1.CephObjectZoneList{
					Items: []cephv1.CephObjectZone{*correctZone.DeepCopy()},
				},
				"cephobjectstores": &cephv1.CephObjectStoreList{
					Items: []cephv1.CephObjectStore{*objectStore},
				},
			},
			expectedZones: &cephv1.CephObjectZoneList{
				Items: []cephv1.CephObjectZone{*correctZone.DeepCopy()},
			},
		},
		{
			name:    "failed to get external svc - no endpoints",
			cephDpl: unitinputs.CephDeployMultisiteRgw.DeepCopy(),
			inputResources: map[string]runtime.Object{
				"cephobjectzones": &cephv1.CephObjectZoneList{
					Items: []cephv1.CephObjectZone{*correctZone.DeepCopy()},
				},
				"cephobjectstores": &cephv1.CephObjectStoreList{
					Items: []cephv1.CephObjectStore{*objectStore},
				},
				"services": &corev1.ServiceList{},
			},
			expectedZones: &cephv1.CephObjectZoneList{
				Items: []cephv1.CephObjectZone{*correctZone.DeepCopy()},
			},
			apiErrors: map[string]error{
				"get-services": errors.New("failed to get external service"),
			},
			expectedError: "failed to get ip of external service: failed to get external service",
		},
		{
			name:    "nothing to do - no endpoints, no rgw external service",
			cephDpl: unitinputs.CephDeployMultisiteRgw.DeepCopy(),
			inputResources: map[string]runtime.Object{
				"cephobjectzones": &cephv1.CephObjectZoneList{
					Items: []cephv1.CephObjectZone{*correctZone.DeepCopy()},
				},
				"cephobjectstores": &cephv1.CephObjectStoreList{
					Items: []cephv1.CephObjectStore{*objectStore},
				},
				"services": &corev1.ServiceList{},
			},
			expectedZones: &cephv1.CephObjectZoneList{
				Items: []cephv1.CephObjectZone{*correctZone.DeepCopy()},
			},
		},
		{
			name:    "update zone - no endpoints, using rgw external service ip and http port",
			cephDpl: unitinputs.CephDeployMultisiteRgw.DeepCopy(),
			inputResources: map[string]runtime.Object{
				"cephobjectzones": &cephv1.CephObjectZoneList{
					Items: []cephv1.CephObjectZone{*correctZone.DeepCopy()},
				},
				"cephobjectstores": &cephv1.CephObjectStoreList{
					Items: []cephv1.CephObjectStore{*objectStore},
				},
				"services": &corev1.ServiceList{
					Items: []corev1.Service{unitinputs.RgwExternalService},
				},
			},
			stateChanged: true,
			expectedZones: &cephv1.CephObjectZoneList{
				Items: []cephv1.CephObjectZone{*zoneWithExtSvcEndpoint},
			},
		},
		{
			name: "update zone - endpoints specified, replace default endpoint",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				mc := unitinputs.CephDeployMultisiteRgw.DeepCopy()
				zone := mc.Spec.ObjectStorage.MultiSite.Zones[0]
				zone.EndpointsForZone = []string{"https://custom-endpoint"}
				mc.Spec.ObjectStorage.MultiSite.Zones[0] = zone
				return mc
			}(),
			inputResources: map[string]runtime.Object{
				"cephobjectzones": &cephv1.CephObjectZoneList{
					Items: []cephv1.CephObjectZone{*zoneWithExtSvcEndpoint.DeepCopy()},
				},
				"cephobjectstores": &cephv1.CephObjectStoreList{
					Items: []cephv1.CephObjectStore{*objectStore},
				},
			},
			stateChanged: true,
			expectedZones: &cephv1.CephObjectZoneList{
				Items: []cephv1.CephObjectZone{
					func() cephv1.CephObjectZone {
						zoneWithEndpoint := correctZone.DeepCopy()
						zoneWithEndpoint.Spec.CustomEndpoints = []string{"https://custom-endpoint"}
						return *zoneWithEndpoint
					}(),
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := fakeDeploymentConfig(&deployConfig{cephDpl: test.cephDpl}, nil)
			faketestclients.FakeReaction(c.api.Rookclientset, "list", []string{"cephobjectstores", "cephobjectzones"}, test.inputResources, nil)
			faketestclients.FakeReaction(c.api.Rookclientset, "create", []string{"cephobjectzones"}, test.inputResources, test.apiErrors)
			faketestclients.FakeReaction(c.api.Rookclientset, "update", []string{"cephobjectzones"}, test.inputResources, test.apiErrors)
			faketestclients.FakeReaction(c.api.Rookclientset, "delete", []string{"cephobjectzones"}, test.inputResources, test.apiErrors)
			faketestclients.FakeReaction(c.api.Kubeclientset.CoreV1(), "get", []string{"services"}, test.inputResources, test.apiErrors)

			stateChanged, err := c.ensureZones()
			if test.expectedError != "" {
				assert.Equal(t, false, stateChanged)
				assert.NotNil(t, err)
				assert.Equal(t, test.expectedError, err.Error())
			} else {
				assert.Nil(t, err)
				assert.Equal(t, test.stateChanged, stateChanged)
			}
			if test.inputResources["cephobjectzones"] != nil {
				assert.Equal(t, test.expectedZones, test.inputResources["cephobjectzones"].(*cephv1.CephObjectZoneList))
			}

			faketestclients.CleanupFakeClientReactions(c.api.Kubeclientset.CoreV1())
			faketestclients.CleanupFakeClientReactions(c.api.Rookclientset)
		})
	}
}

func TestDeleteMultiSite(t *testing.T) {
	tests := []struct {
		name           string
		cleanupDone    bool
		inputResources map[string]runtime.Object
		apiErrors      map[string]error
		expectedError  string
	}{
		{
			name:           "failed to list zones",
			inputResources: map[string]runtime.Object{},
			expectedError:  "failed to get list CephObjectZones in 'rook-ceph' namespace: failed to list cephobjectzones",
		},
		{
			name: "failed to list zonegroups",
			inputResources: map[string]runtime.Object{
				"cephobjectzones": &cephv1.CephObjectZoneList{},
			},
			expectedError: "failed to get list CephObjectZoneGroups in 'rook-ceph' namespace: failed to list cephobjectzonegroups",
		},
		{
			name: "failed to list realms",
			inputResources: map[string]runtime.Object{
				"cephobjectzones":      &cephv1.CephObjectZoneList{},
				"cephobjectzonegroups": &cephv1.CephObjectZoneGroupList{},
			},
			expectedError: "failed to get list CephObjectRealms in 'rook-ceph' namespace: failed to list cephobjectrealms",
		},
		{
			name: "failed to list rgw",
			inputResources: map[string]runtime.Object{
				"cephobjectzones":      &cephv1.CephObjectZoneList{},
				"cephobjectzonegroups": &cephv1.CephObjectZoneGroupList{},
				"cephobjectrealms":     &cephv1.CephObjectRealmList{},
			},
			expectedError: "failed to check zones in use: failed to list cephobjectstores",
		},
		{
			name: "multisite objects cant be removed - all in use",
			inputResources: map[string]runtime.Object{
				"cephobjectzones": &cephv1.CephObjectZoneList{
					Items: []cephv1.CephObjectZone{*unitinputs.RgwMultisiteMasterZone1.DeepCopy()},
				},
				"cephobjectzonegroups": &cephv1.CephObjectZoneGroupList{
					Items: []cephv1.CephObjectZoneGroup{*unitinputs.RgwMultisiteMasterZoneGroup1.DeepCopy()},
				},
				"cephobjectrealms": &cephv1.CephObjectRealmList{
					Items: []cephv1.CephObjectRealm{*unitinputs.RgwMultisiteMasterRealm1.DeepCopy()},
				},
				"cephobjectstores": &cephv1.CephObjectStoreList{
					Items: []cephv1.CephObjectStore{*unitinputs.CephObjectStoreWithZone.DeepCopy()},
				},
			},
			apiErrors: map[string]error{
				"delete-cephobjectzones":      errors.New("unexpected delete"),
				"delete-cephobjectzonegroups": errors.New("unexpected delete"),
				"delete-cephobjectrealms":     errors.New("unexpected delete"),
				"delete-secrets":              errors.New("unexpected delete"),
			},
		},
		{
			name: "multisite zone removed",
			inputResources: map[string]runtime.Object{
				"cephobjectzones": &cephv1.CephObjectZoneList{
					Items: []cephv1.CephObjectZone{*unitinputs.RgwMultisiteMasterZone1.DeepCopy()},
				},
				"cephobjectzonegroups": &cephv1.CephObjectZoneGroupList{
					Items: []cephv1.CephObjectZoneGroup{*unitinputs.RgwMultisiteMasterZoneGroup1.DeepCopy()},
				},
				"cephobjectrealms": &cephv1.CephObjectRealmList{
					Items: []cephv1.CephObjectRealm{*unitinputs.RgwMultisiteMasterRealm1.DeepCopy()},
				},
				"cephobjectstores": &cephv1.CephObjectStoreList{
					Items: []cephv1.CephObjectStore{*unitinputs.CephObjectStoreBase.DeepCopy()},
				},
			},
			apiErrors: map[string]error{
				"delete-cephobjectzonegroups": errors.New("unexpected delete"),
				"delete-cephobjectrealms":     errors.New("unexpected delete"),
				"delete-secrets":              errors.New("unexpected delete"),
			},
		},
		{
			name: "multisite zonegroup removed",
			inputResources: map[string]runtime.Object{
				"cephobjectzones": &cephv1.CephObjectZoneList{},
				"cephobjectzonegroups": &cephv1.CephObjectZoneGroupList{
					Items: []cephv1.CephObjectZoneGroup{*unitinputs.RgwMultisiteMasterZoneGroup1.DeepCopy()},
				},
				"cephobjectrealms": &cephv1.CephObjectRealmList{
					Items: []cephv1.CephObjectRealm{*unitinputs.RgwMultisiteMasterRealm1.DeepCopy()},
				},
				"cephobjectstores": &cephv1.CephObjectStoreList{
					Items: []cephv1.CephObjectStore{*unitinputs.CephObjectStoreBase.DeepCopy()},
				},
			},
			apiErrors: map[string]error{
				"delete-cephobjectrealms": errors.New("unexpected delete"),
				"delete-secrets":          errors.New("unexpected delete"),
			},
		},
		{
			name: "multisite realm removed",
			inputResources: map[string]runtime.Object{
				"secrets": &corev1.SecretList{
					Items: []corev1.Secret{*unitinputs.MultisiteRealmSecret.DeepCopy()},
				},
				"cephobjectzones":      &cephv1.CephObjectZoneList{},
				"cephobjectzonegroups": &cephv1.CephObjectZoneGroupList{},
				"cephobjectrealms": &cephv1.CephObjectRealmList{
					Items: []cephv1.CephObjectRealm{*unitinputs.RgwMultisiteMasterRealm1.DeepCopy()},
				},
				"cephobjectstores": &cephv1.CephObjectStoreList{
					Items: []cephv1.CephObjectStore{*unitinputs.CephObjectStoreBase.DeepCopy()},
				},
			},
		},
		{
			name: "multisite removed not used objects",
			inputResources: map[string]runtime.Object{
				"secrets": &corev1.SecretList{},
				"cephobjectzones": &cephv1.CephObjectZoneList{
					Items: []cephv1.CephObjectZone{
						*unitinputs.RgwMultisiteMasterZone1.DeepCopy(),
						*unitinputs.RgwMultisiteSecondaryZone1.DeepCopy(),
					},
				},
				"cephobjectzonegroups": &cephv1.CephObjectZoneGroupList{
					Items: []cephv1.CephObjectZoneGroup{
						*unitinputs.RgwMultisiteMasterZoneGroup1.DeepCopy(),
						{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "fake-zonegroup1",
								Namespace: "rook-ceph",
							},
							Spec: cephv1.ObjectZoneGroupSpec{
								Realm: "fake-realm1",
							},
						},
					},
				},
				"cephobjectrealms": &cephv1.CephObjectRealmList{
					Items: []cephv1.CephObjectRealm{
						*unitinputs.RgwMultisiteMasterRealm1.DeepCopy(),
						{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "fake-realm",
								Namespace: "rook-ceph",
							},
						},
					},
				},
				"cephobjectstores": &cephv1.CephObjectStoreList{
					Items: []cephv1.CephObjectStore{*unitinputs.CephObjectStoreWithZone.DeepCopy()},
				},
			},
			apiErrors: map[string]error{
				"delete-cephobjectzones-zone1":           errors.New("unexpected delete"),
				"delete-cephobjectzonegroups-zonegroup1": errors.New("unexpected delete"),
				"delete-cephobjectrealms-realm1":         errors.New("unexpected delete"),
				"delete-secrets-realm1-keys":             errors.New("unexpected delete"),
			},
		},
		{
			name: "multisite failed to removed not used objects",
			inputResources: map[string]runtime.Object{
				"secrets": &corev1.SecretList{},
				"cephobjectzones": &cephv1.CephObjectZoneList{
					Items: []cephv1.CephObjectZone{
						*unitinputs.RgwMultisiteMasterZone1.DeepCopy(),
						*unitinputs.RgwMultisiteSecondaryZone1.DeepCopy(),
					},
				},
				"cephobjectzonegroups": &cephv1.CephObjectZoneGroupList{
					Items: []cephv1.CephObjectZoneGroup{
						*unitinputs.RgwMultisiteMasterZoneGroup1.DeepCopy(),
						{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "fake-zonegroup1",
								Namespace: "rook-ceph",
							},
							Spec: cephv1.ObjectZoneGroupSpec{
								Realm: "fake-realm1",
							},
						},
					},
				},
				"cephobjectrealms": &cephv1.CephObjectRealmList{
					Items: []cephv1.CephObjectRealm{
						*unitinputs.RgwMultisiteMasterRealm1.DeepCopy(),
						{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "fake-realm",
								Namespace: "rook-ceph",
							},
						},
					},
				},
				"cephobjectstores": &cephv1.CephObjectStoreList{
					Items: []cephv1.CephObjectStore{*unitinputs.CephObjectStoreWithZone.DeepCopy()},
				},
			},
			apiErrors: map[string]error{
				"delete-cephobjectzones":      errors.New("failed to delete zone"),
				"delete-cephobjectzonegroups": errors.New("failed to delete zonegroup"),
				"delete-cephobjectrealms":     errors.New("failed to delete realm"),
				"delete-secrets":              errors.New("failed to delete secret"),
			},
			expectedError: "failed to cleanup multisite: failed to delete CephObjectZone 'rook-ceph/secondary-zone1': failed to delete zone, failed to delete CephObjectZoneGroup 'rook-ceph/fake-zonegroup1': failed to delete zonegroup, failed to delete Secret 'rook-ceph/fake-realm-keys' for CephObjectRealm 'rook-ceph/fake-realm': failed to delete secret",
		},
		{
			name: "multisite objects not in use, remove all",
			inputResources: map[string]runtime.Object{
				"secrets":              &corev1.SecretList{},
				"cephobjectzones":      &cephv1.CephObjectZoneList{},
				"cephobjectzonegroups": &cephv1.CephObjectZoneGroupList{},
				"cephobjectrealms":     &cephv1.CephObjectRealmList{},
				"cephobjectstores": &cephv1.CephObjectStoreList{
					Items: []cephv1.CephObjectStore{*unitinputs.CephObjectStoreBase.DeepCopy()},
				},
			},
			cleanupDone: true,
		},
	}
	oldFunc := lcmcommon.RunPodCommandWithValidation
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := fakeDeploymentConfig(nil, nil)
			lcmcommon.RunPodCommandWithValidation = func(e lcmcommon.ExecConfig) (string, string, error) {
				switch e.Command {
				case "radosgw-admin realm list":
					return "{}", "", nil
				case "radosgw-admin zonegroup list":
					return "{}", "", nil
				}
				return "", "", errors.New("unexpected command call: " + e.Command)
			}
			faketestclients.FakeReaction(c.api.Kubeclientset.CoreV1(), "delete", []string{"secrets"}, test.inputResources, test.apiErrors)
			faketestclients.FakeReaction(c.api.Rookclientset, "list", multisiteResources, test.inputResources, nil)
			faketestclients.FakeReaction(c.api.Rookclientset, "delete", multisiteResources, test.inputResources, test.apiErrors)

			cleanupDone, err := c.deleteMultiSite()
			if test.expectedError != "" {
				assert.Equal(t, false, cleanupDone)
				assert.NotNil(t, err)
				assert.Equal(t, test.expectedError, err.Error())
			} else {
				assert.Nil(t, err)
				assert.Equal(t, test.cleanupDone, cleanupDone)
			}

			faketestclients.CleanupFakeClientReactions(c.api.Kubeclientset.CoreV1())
			faketestclients.CleanupFakeClientReactions(c.api.Rookclientset)
		})
	}
	lcmcommon.RunPodCommandWithValidation = oldFunc
}

func TestDeleteRealm(t *testing.T) {
	realmOutput := `{
    "default_info": "a6038b94-fcea-46fe-8ef5-33af7f9eed80",
    "realms": [
        "fake-realm"
    ]
}`
	tests := []struct {
		name          string
		realm         cephv1.CephObjectRealm
		cmdOutput     string
		apiErrors     map[string]error
		expectedError string
	}{
		{
			name:  "failed to remove realm secret",
			realm: cephv1.CephObjectRealm{ObjectMeta: metav1.ObjectMeta{Name: "fake-realm", Namespace: "rook-ceph"}},
			apiErrors: map[string]error{
				"delete-secrets": errors.New("failed to remove secrets"),
			},
			expectedError: "failed to delete Secret 'rook-ceph/fake-realm-keys' for CephObjectRealm 'rook-ceph/fake-realm': failed to remove secrets",
		},
		{
			name:  "failed to check realms list via cli",
			realm: cephv1.CephObjectRealm{ObjectMeta: metav1.ObjectMeta{Name: "fake-realm", Namespace: "rook-ceph"}},
			apiErrors: map[string]error{
				"list-cmd": errors.New("failed to list"),
			},
			expectedError: "failed to check realm list: failed to run command 'radosgw-admin realm list': failed to list",
		},
		{
			name:      "failed to remove realm via cli",
			realm:     cephv1.CephObjectRealm{ObjectMeta: metav1.ObjectMeta{Name: "fake-realm", Namespace: "rook-ceph"}},
			cmdOutput: realmOutput,
			apiErrors: map[string]error{
				"delete-cmd": errors.New("failed to delete"),
			},
			expectedError: "failed to remove realm 'fake-realm': failed to run command 'radosgw-admin realm rm --rgw-realm=fake-realm': failed to delete",
		},
		{
			name:      "failed to remove realm via kubeclient",
			realm:     cephv1.CephObjectRealm{ObjectMeta: metav1.ObjectMeta{Name: "fake-realm", Namespace: "rook-ceph"}},
			cmdOutput: realmOutput,
			apiErrors: map[string]error{
				"delete-cephobjectrealms": errors.New("failed to delete"),
			},
			expectedError: "failed to delete CephObjectRealm 'rook-ceph/fake-realm': failed to delete",
		},
		{
			name:      "clean up is done",
			realm:     cephv1.CephObjectRealm{ObjectMeta: metav1.ObjectMeta{Name: "fake-realm", Namespace: "rook-ceph"}},
			cmdOutput: "{\"realms\":[]}",
		},
	}
	oldFunc := lcmcommon.RunPodCommandWithValidation
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := fakeDeploymentConfig(nil, nil)
			lcmcommon.RunPodCommandWithValidation = func(e lcmcommon.ExecConfig) (string, string, error) {
				if strings.HasPrefix(e.Command, "radosgw-admin realm rm --rgw-realm=") {
					return "", "", test.apiErrors["delete-cmd"]
				} else if e.Command == "radosgw-admin realm list" {
					return test.cmdOutput, "", test.apiErrors["list-cmd"]
				}
				return "", "", errors.New("unexpected command call: " + e.Command)
			}

			inputResources := map[string]runtime.Object{
				"secrets":          &corev1.SecretList{},
				"cephobjectrealms": &cephv1.CephObjectRealmList{Items: []cephv1.CephObjectRealm{test.realm}},
			}
			faketestclients.FakeReaction(c.api.Kubeclientset.CoreV1(), "delete", []string{"secrets"}, inputResources, test.apiErrors)
			faketestclients.FakeReaction(c.api.Rookclientset, "delete", []string{"cephobjectrealms"}, inputResources, test.apiErrors)

			err := c.deleteRealm(test.realm.Name)
			if test.expectedError != "" {
				assert.NotNil(t, err)
				assert.Equal(t, test.expectedError, err.Error())
			} else {
				assert.Nil(t, err)
			}
			faketestclients.CleanupFakeClientReactions(c.api.Kubeclientset.CoreV1())
			faketestclients.CleanupFakeClientReactions(c.api.Rookclientset)
		})
	}
	lcmcommon.RunPodCommandWithValidation = oldFunc
}

func TestDeleteZoneGroup(t *testing.T) {
	zonegroupOutput := `{
    "default_info": "83f9d079-f175-4dbf-8e70-60c6ac1a9ca9",
    "zonegroups": [
        "fake-zonegroup1"
    ]
}`
	tests := []struct {
		name          string
		zonegroup     cephv1.CephObjectZoneGroup
		cmdOutput     string
		apiErrors     map[string]error
		expectedError string
	}{
		{
			name:      "failed to check zonegroups list via cli",
			zonegroup: cephv1.CephObjectZoneGroup{ObjectMeta: metav1.ObjectMeta{Name: "fake-zonegroup1", Namespace: "rook-ceph"}},
			apiErrors: map[string]error{
				"list-cmd": errors.New("failed to list"),
			},
			expectedError: "failed to check zonegroup list: failed to run command 'radosgw-admin zonegroup list': failed to list",
		},
		{
			name:      "failed to remove zonegroup via cli",
			zonegroup: cephv1.CephObjectZoneGroup{ObjectMeta: metav1.ObjectMeta{Name: "fake-zonegroup1", Namespace: "rook-ceph"}},
			cmdOutput: zonegroupOutput,
			apiErrors: map[string]error{
				"delete-cmd": errors.New("failed to delete"),
			},
			expectedError: "failed to remove zonegroup 'fake-zonegroup1': failed to run command 'radosgw-admin zonegroup delete --rgw-zonegroup=fake-zonegroup1': failed to delete",
		},
		{
			name:      "failed to remove zonegroup via kubeclient",
			zonegroup: cephv1.CephObjectZoneGroup{ObjectMeta: metav1.ObjectMeta{Name: "fake-zonegroup1", Namespace: "rook-ceph"}},
			cmdOutput: zonegroupOutput,
			apiErrors: map[string]error{
				"delete-cephobjectzonegroups": errors.New("failed to delete"),
			},
			expectedError: "failed to delete CephObjectZoneGroup 'rook-ceph/fake-zonegroup1': failed to delete",
		},
		{
			name:      "clean up is done",
			zonegroup: cephv1.CephObjectZoneGroup{ObjectMeta: metav1.ObjectMeta{Name: "fake-zonegroup1", Namespace: "rook-ceph"}},
			cmdOutput: "{\"zonegroups\":[]}",
		},
	}
	oldFunc := lcmcommon.RunPodCommandWithValidation
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := fakeDeploymentConfig(nil, nil)
			lcmcommon.RunPodCommandWithValidation = func(e lcmcommon.ExecConfig) (string, string, error) {
				if strings.HasPrefix(e.Command, "radosgw-admin zonegroup delete --rgw-zonegroup=") {
					return "", "", test.apiErrors["delete-cmd"]
				} else if e.Command == "radosgw-admin zonegroup list" {
					return test.cmdOutput, "", test.apiErrors["list-cmd"]
				}
				return "", "", errors.New("unexpected command call: " + e.Command)
			}

			inputResources := map[string]runtime.Object{
				"cephobjectzonegroups": &cephv1.CephObjectZoneGroupList{Items: []cephv1.CephObjectZoneGroup{test.zonegroup}},
			}
			faketestclients.FakeReaction(c.api.Rookclientset, "delete", []string{"cephobjectzonegroups"}, inputResources, test.apiErrors)

			err := c.deleteZoneGroup(test.zonegroup.Name)
			if test.expectedError != "" {
				assert.NotNil(t, err)
				assert.Equal(t, test.expectedError, err.Error())
			} else {
				assert.Nil(t, err)
			}
			faketestclients.CleanupFakeClientReactions(c.api.Rookclientset)
		})
	}
	lcmcommon.RunPodCommandWithValidation = oldFunc
}
