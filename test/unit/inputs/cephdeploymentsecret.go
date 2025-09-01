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
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	cephlcmv1alpha1 "github.com/Mirantis/pelagia/pkg/apis/ceph.pelagia.lcm/v1alpha1"
)

var EmptyCephSecret = &cephlcmv1alpha1.CephDeploymentSecret{
	ObjectMeta: LcmObjectMeta,
}

func GetNewCephSecret() *cephlcmv1alpha1.CephDeploymentSecret {
	secret := EmptyCephSecret.DeepCopy()
	secret.OwnerReferences = []metav1.OwnerReference{
		{
			APIVersion: "lcm.mirantis.com/v1alpha1",
			Kind:       "CephDeployment",
			Name:       LcmObjectMeta.Name,
		},
	}
	return secret
}

var CephSecretReady = &cephlcmv1alpha1.CephDeploymentSecret{
	ObjectMeta: GetNewCephSecret().ObjectMeta,
	Status: &cephlcmv1alpha1.CephDeploymentSecretStatus{
		State: cephlcmv1alpha1.HealthStateOk,
		SecretsInfo: &cephlcmv1alpha1.CephDeploymentSecretsInfo{
			ClientSecrets: []cephlcmv1alpha1.CephDeploymentSecretInfo{
				{
					ObjectName:      "client.admin",
					SecretName:      "rook-ceph-admin-keyring",
					SecretNamespace: "rook-ceph",
				},
			},
		},
		LastSecretCheck:  time.Date(2021, 8, 15, 14, 30, 52, 0, time.Local).Format(time.RFC3339),
		LastSecretUpdate: time.Date(2021, 8, 15, 14, 30, 52, 0, time.Local).Format(time.RFC3339),
	},
}

var CephSecretNotReady = &cephlcmv1alpha1.CephDeploymentSecret{
	ObjectMeta: GetNewCephSecret().ObjectMeta,
	Status: &cephlcmv1alpha1.CephDeploymentSecretStatus{
		State:            cephlcmv1alpha1.HealthStateFailed,
		SecretsInfo:      &cephlcmv1alpha1.CephDeploymentSecretsInfo{},
		LastSecretUpdate: time.Date(2021, 8, 15, 14, 30, 53, 0, time.Local).Format(time.RFC3339),
		LastSecretCheck:  time.Date(2021, 8, 15, 14, 30, 53, 0, time.Local).Format(time.RFC3339),
		Messages:         []string{"admin keyring secret is not available: secrets \"rook-ceph-admin-keyring\" not found"},
	},
}

var CephSecretReadySecretsInfo = &cephlcmv1alpha1.CephDeploymentSecretsInfo{
	ClientSecrets: []cephlcmv1alpha1.CephDeploymentSecretInfo{
		{
			ObjectName:      "client.admin",
			SecretName:      "rook-ceph-admin-keyring",
			SecretNamespace: "rook-ceph",
		},
		{
			ObjectName:      "client.test",
			SecretName:      "rook-ceph-client-test",
			SecretNamespace: "rook-ceph",
		},
	},
	RgwUserSecrets: []cephlcmv1alpha1.CephDeploymentSecretInfo{
		{
			ObjectName:      "test-user",
			SecretName:      "rgw-metrics-user-secret",
			SecretNamespace: "rook-ceph",
		},
	},
}

var CephExternalSecretReadySecretsInfo = &cephlcmv1alpha1.CephDeploymentSecretsInfo{
	ClientSecrets: []cephlcmv1alpha1.CephDeploymentSecretInfo{
		{
			ObjectName:      "client.test",
			SecretName:      "rook-ceph-client-test",
			SecretNamespace: "rook-ceph",
		},
	},
	RgwUserSecrets: []cephlcmv1alpha1.CephDeploymentSecretInfo{
		{
			ObjectName:      "test-user",
			SecretName:      "rgw-metrics-user-secret",
			SecretNamespace: "rook-ceph",
		},
	},
}
