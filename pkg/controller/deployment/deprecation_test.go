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

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/runtime"

	cephlcmv1alpha1 "github.com/Mirantis/pelagia/pkg/apis/ceph.pelagia.lcm/v1alpha1"
	faketestclients "github.com/Mirantis/pelagia/test/unit/clients"
	unitinputs "github.com/Mirantis/pelagia/test/unit/inputs"
)

func TestEnsureDeprecatedFields(t *testing.T) {
	cephDeplConflicted := unitinputs.CephDeploymentDeprecated.DeepCopy()
	cephDeplConflicted.Spec.Cluster = unitinputs.CephDeploymentMigrated.Spec.Cluster.DeepCopy()
	cephDeplConflicted.Spec.BlockStorage = unitinputs.CephDeploymentMigrated.Spec.BlockStorage.DeepCopy()
	cephDeplConflicted.Spec.SharedFilesystem.Filesystems = unitinputs.CephDeploymentMigrated.Spec.SharedFilesystem.DeepCopy().Filesystems
	cephDeplConflicted.Spec.ObjectStorage = unitinputs.CephDeploymentMigrated.Spec.ObjectStorage.DeepCopy()
	cephDeplConflicted.Spec.ObjectStorage.OldRgw = unitinputs.CephDeploymentDeprecated.Spec.ObjectStorage.OldRgw.DeepCopy()

	cephDeplMultisiteConflicted := unitinputs.CephDeploymentMultisiteMigrated.DeepCopy()
	cephDeplMultisiteConflicted.Spec.ObjectStorage.OldMultiSite = unitinputs.CephDeploymentMultisiteDeprecated.Spec.ObjectStorage.OldMultiSite.DeepCopy()

	tests := []struct {
		name            string
		cephDpl         *cephlcmv1alpha1.CephDeployment
		expectedCephDpl cephlcmv1alpha1.CephDeployment
		expectedError   string
		migrated        bool
	}{
		{
			name:            "cant migrate deprecated fields due to conflicts",
			cephDpl:         cephDeplConflicted.DeepCopy(),
			expectedCephDpl: *cephDeplConflicted,
			expectedError:   "found deprecated params which can't be automatically migrated: [ spec.dashboard spec.dataDirHostPath spec.healthCheck spec.hyperconverge.resources spec.hyperconverge.tolerations[all] spec.hyperconverge.tolerations[mgr] spec.hyperconverge.tolerations[mon] spec.hyperconverge.tolerations[osd] spec.mgr spec.network spec.pools spec.sharedFilesystem.cephFS spec.objectStorage.rgw.objectUsers spec.objectStorage.rgw ]",
		},
		{
			name:            "cant migrate deprecated multisite fields due to conflicts",
			cephDpl:         cephDeplMultisiteConflicted.DeepCopy(),
			expectedCephDpl: *cephDeplMultisiteConflicted,
			expectedError:   "found deprecated params which can't be automatically migrated: [ spec.objectStorage.multiSite.realms spec.objectStorage.multiSite.zoneGroups spec.objectStorage.multiSite.zones ]",
		},
		{
			name:            "migrated non-mosk deprecated fields",
			cephDpl:         unitinputs.CephDeploymentDeprecated.DeepCopy(),
			expectedCephDpl: unitinputs.CephDeploymentMigrated,
			migrated:        true,
		},
		{
			name: "migrated mosk deprecated fields",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				cephDpl := unitinputs.CephDeployMosk.DeepCopy()
				cephDpl.Spec.ObjectStorage = unitinputs.CephDeploymentDeprecated.Spec.ObjectStorage.DeepCopy()
				return cephDpl
			}(),
			expectedCephDpl: func() cephlcmv1alpha1.CephDeployment {
				cephDpl := unitinputs.CephDeployMosk.DeepCopy()
				cephDpl.Spec.ObjectStorage = unitinputs.CephDeploymentMigrated.Spec.ObjectStorage.DeepCopy()
				cephDpl.Spec.ObjectStorage.Rgws[0].UsedByRockoon = true
				cephDpl.Spec.ObjectStorage.Rgws[0].ServedByIngress = true
				cephDpl.Spec.ObjectStorage.Rgws[0].Spec.Raw = []byte(`{"dataPool":{"deviceClass":"hdd","erasureCoded":{"codingChunks":1,"dataChunks":2}},"gateway":{"instances":2,"port":80,"securePort":8443,"sslCertificateRef":"rgw-ssl-certificate"},"metadataPool":{"replicated":{"size":3},"deviceClass":"hdd"},"preservePoolsOnDelete":false}`)
				return *cephDpl
			}(),
			migrated: true,
		},
		{
			name:            "migrated multus deprecated fields",
			cephDpl:         unitinputs.CephDeploymentMultusDeprecated.DeepCopy(),
			expectedCephDpl: unitinputs.CephDeploymentMultusMigrated,
			migrated:        true,
		},
		{
			name:            "migrated deprecated multisite fields",
			cephDpl:         unitinputs.CephDeploymentMultisiteDeprecated.DeepCopy(),
			expectedCephDpl: unitinputs.CephDeploymentMultisiteMigrated,
			migrated:        true,
		},
		{
			name: "migrated deprecated multisite with pull realm fields",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				cdpl := unitinputs.CephDeploymentMultisiteDeprecated.DeepCopy()
				cdpl.Spec.ObjectStorage.OldMultiSite.Realms[0].Pull = &cephlcmv1alpha1.CephRGWRealmPull{
					Endpoint:  "http://custom",
					AccessKey: "accesskey",
					SecretKey: "secretkey",
				}
				cdpl.Spec.ObjectStorage.OldRgw.Gateway.SplitDaemonForMultisiteTrafficSync = true
				return cdpl
			}(),
			expectedCephDpl: func() cephlcmv1alpha1.CephDeployment {
				cdpl := unitinputs.CephDeploymentMultisiteMigrated.DeepCopy()
				cdpl.Spec.ObjectStorage.Realms[0].Spec.Raw = []byte(`{"defaultRealm":false,"pull":{"endpoint":"http://custom"}}`)
				syncRgw := cephlcmv1alpha1.CephObjectStore{
					Name:             "rgw-store-sync",
					AuxiliaryService: true,
					Spec:             runtime.RawExtension{Raw: []byte(`{"gateway":{"disableMultisiteSyncTraffic":false,"instances":1,"port":8380},"zone":{"name":"zone1"}}`)},
				}
				cdpl.Spec.ObjectStorage.Rgws[0].Spec.Raw = []byte(`{"gateway":{"disableMultisiteSyncTraffic":true,"instances":2,"port":80,"securePort":8443},"zone":{"name":"zone1"}}`)
				cdpl.Spec.ObjectStorage.Rgws = append(cdpl.Spec.ObjectStorage.Rgws, syncRgw)
				return *cdpl
			}(),
			migrated: true,
		},
		{
			name:            "migrated external deprecated fields",
			cephDpl:         unitinputs.CephDeployExternalDeprecated.DeepCopy(),
			expectedCephDpl: unitinputs.CephDeployExternalMigrated,
			migrated:        true,
		},
		{
			name:            "no transform",
			cephDpl:         unitinputs.CephDeploymentMigrated.DeepCopy(),
			expectedCephDpl: unitinputs.CephDeploymentMigrated,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := fakeDeploymentConfig(&deployConfig{cephDpl: test.cephDpl.DeepCopy()}, nil)
			inputResources := map[string]runtime.Object{"cephdeployments": &cephlcmv1alpha1.CephDeploymentList{Items: []cephlcmv1alpha1.CephDeployment{*test.cephDpl}}}
			expectedResources := map[string]runtime.Object{"cephdeployments": &cephlcmv1alpha1.CephDeploymentList{Items: []cephlcmv1alpha1.CephDeployment{test.expectedCephDpl}}}
			faketestclients.FakeReaction(c.api.CephLcmclientset, "update", []string{"cephdeployments"}, inputResources, nil)

			migrated, err := c.ensureDeprecatedFields()
			if test.expectedError == "" {
				assert.Nil(t, err)
			} else {
				assert.Equal(t, test.expectedError, err.Error())
			}
			assert.Equal(t, test.migrated, migrated)
			assert.Equal(t, expectedResources, inputResources)
			faketestclients.CleanupFakeClientReactions(c.api.CephLcmclientset)
		})
	}
}
