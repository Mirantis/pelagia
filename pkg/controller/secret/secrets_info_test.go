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

package secret

import (
	"testing"

	cephv1 "github.com/rook/rook/pkg/apis/ceph.rook.io/v1"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"

	cephlcmv1alpha1 "github.com/Mirantis/pelagia/pkg/apis/ceph.pelagia.lcm/v1alpha1"
	faketestclients "github.com/Mirantis/pelagia/test/unit/clients"
	unitinputs "github.com/Mirantis/pelagia/test/unit/inputs"
)

func TestGetSecretsStatusInfo(t *testing.T) {
	cephClientNotReady := unitinputs.TestCephClientNotReady.DeepCopy()
	cephClientNotReady.Name = "someclient"
	rgwUserNotReady := unitinputs.RgwUserWithStatus(unitinputs.GetCephRgwUser("test-user-2", "rook-ceph", "rgw-store"), "NotReady")
	rgwUserNotReady.Status.Info = nil
	rgwUserReady := unitinputs.RgwUserWithStatus(unitinputs.GetCephRgwUser("test-user", "rook-ceph", "rgw-store"), "Ready")
	tests := []struct {
		name                string
		cephDpl             *cephlcmv1alpha1.CephDeployment
		inputResources      map[string]runtime.Object
		expectedSecretsInfo *cephlcmv1alpha1.CephDeploymentSecretsInfo
		expectedIssues      []string
	}{
		{
			name:    "secrets status no issues",
			cephDpl: &unitinputs.CephDeployNonMoskForSecret,
			inputResources: map[string]runtime.Object{
				"secrets":              &corev1.SecretList{Items: []corev1.Secret{unitinputs.CephAdminKeyringSecret}},
				"cephclients":          &cephv1.CephClientList{Items: []cephv1.CephClient{unitinputs.TestCephClientReady}},
				"cephobjectstoreusers": &cephv1.CephObjectStoreUserList{Items: []cephv1.CephObjectStoreUser{*rgwUserReady}},
			},
			expectedSecretsInfo: unitinputs.CephSecretReadySecretsInfo,
			expectedIssues:      []string{},
		},
		{
			name:    "secrets status - failed to list users and clients",
			cephDpl: &unitinputs.CephDeployNonMoskForSecret,
			inputResources: map[string]runtime.Object{
				"secrets": &unitinputs.SecretsListEmpty,
			},
			expectedSecretsInfo: unitinputs.CephSecretNotReady.Status.SecretsInfo,
			expectedIssues: []string{
				"admin keyring secret is not available: secrets \"rook-ceph-admin-keyring\" not found",
				"failed to list ceph clients: failed to list cephclients",
				"failed to list ceph clients: failed to list cephobjectstoreusers",
			},
		},
		{
			name:    "secrets status - some users or clients not ready",
			cephDpl: &unitinputs.CephDeployNonMoskForSecret,
			inputResources: map[string]runtime.Object{
				"secrets": &corev1.SecretList{Items: []corev1.Secret{unitinputs.CephAdminKeyringSecret}},
				"cephclients": &cephv1.CephClientList{
					Items: []cephv1.CephClient{unitinputs.TestCephClientReady, *cephClientNotReady},
				},
				"cephobjectstoreusers": &cephv1.CephObjectStoreUserList{
					Items: []cephv1.CephObjectStoreUser{*rgwUserNotReady, *rgwUserReady},
				},
			},
			expectedSecretsInfo: unitinputs.CephSecretReadySecretsInfo,
			expectedIssues:      []string{"client someclient secret is not ready", "rgw user test-user-2 secret is not ready"},
		},
		{
			name: "external cluster - secrets status no issues",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				mc := unitinputs.CephDeployNonMoskForSecret.DeepCopy()
				mc.Spec.External = true
				return mc
			}(),
			inputResources: map[string]runtime.Object{
				"cephclients":          &cephv1.CephClientList{Items: []cephv1.CephClient{unitinputs.TestCephClientReady}},
				"cephobjectstoreusers": &cephv1.CephObjectStoreUserList{Items: []cephv1.CephObjectStoreUser{*rgwUserReady}},
			},
			expectedSecretsInfo: unitinputs.CephExternalSecretReadySecretsInfo,
			expectedIssues:      []string{},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := fakeSecretConfig(&secretsConfig{cephDpl: test.cephDpl})
			faketestclients.FakeReaction(c.api.Kubeclientset.CoreV1(), "get", []string{"secrets"}, test.inputResources, nil)
			faketestclients.FakeReaction(c.api.Rookclientset, "list", []string{"cephclients", "cephobjectstoreusers"}, test.inputResources, nil)

			secretsInfo, issues := c.getSecretsStatusInfo()
			assert.Equal(t, test.expectedSecretsInfo, secretsInfo)
			assert.Equal(t, test.expectedIssues, issues)

			faketestclients.CleanupFakeClientReactions(c.api.Kubeclientset.CoreV1())
			faketestclients.CleanupFakeClientReactions(c.api.Rookclientset)
		})
	}
}
