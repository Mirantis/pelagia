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
	tests := []struct {
		name            string
		cephDpl         *cephlcmv1alpha1.CephDeployment
		expectedCephDpl cephlcmv1alpha1.CephDeployment
		expectedError   string
		migrated        bool
	}{
		{
			name: "cant migrate deprecated fields due to conflicts",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				cephDeplConflicted := unitinputs.CephDeploymentDeprecated.DeepCopy()
				cephDeplConflicted.Spec.Cluster = unitinputs.CephDeploymentMigrated.Spec.Cluster.DeepCopy()
				cephDeplConflicted.Spec.Cluster.Raw = unitinputs.ConvertYamlToJSON(cephDeplConflicted.Spec.Cluster.Raw)
				return cephDeplConflicted
			}(),
			expectedCephDpl: func() cephlcmv1alpha1.CephDeployment {
				cephDeplConflicted := unitinputs.CephDeploymentDeprecated.DeepCopy()
				cephDeplConflicted.Spec.Cluster = unitinputs.CephDeploymentMigrated.Spec.Cluster.DeepCopy()
				return *cephDeplConflicted
			}(),
			expectedError: "found deprecated params which can't be automatically migrated: spec.dashboard,spec.dataDirHostPath,spec.healthCheck,spec.hyperconverge.resources,spec.hyperconverge.tolerations[all],spec.hyperconverge.tolerations[mgr],spec.hyperconverge.tolerations[mon],spec.hyperconverge.tolerations[osd],spec.mgr,spec.network",
		},
		{
			name:            "migrated deprecated fields",
			cephDpl:         unitinputs.CephDeploymentDeprecated.DeepCopy(),
			expectedCephDpl: unitinputs.CephDeploymentMigrated,
			migrated:        true,
		},
		{
			name:            "migrated multus deprecated fields",
			cephDpl:         unitinputs.CephDeploymentMultusDeprecated.DeepCopy(),
			expectedCephDpl: unitinputs.CephDeploymentMultusMigrated,
			migrated:        true,
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
			c := fakeDeploymentConfig(&deployConfig{cephDpl: test.cephDpl}, nil)
			inputResources := map[string]runtime.Object{"cephdeployments": &cephlcmv1alpha1.CephDeploymentList{Items: []cephlcmv1alpha1.CephDeployment{*test.cephDpl}}}
			expectedResources := map[string]runtime.Object{"cephdeployments": &cephlcmv1alpha1.CephDeploymentList{Items: []cephlcmv1alpha1.CephDeployment{test.expectedCephDpl}}}
			faketestclients.FakeReaction(c.api.CephLcmclientset, "update", []string{"cephdeployments"}, inputResources, nil)

			migrated, err := c.ensureDeprecatedFields(false)
			if test.expectedError == "" {
				assert.Nil(t, err)
			} else {
				assert.Equal(t, test.expectedError, err.Error())
			}
			assert.Equal(t, test.migrated, migrated)
			// convert spec to yaml for better UX with inputs
			cephDeploy := inputResources["cephdeployments"].(*cephlcmv1alpha1.CephDeploymentList).Items[0].Spec.Cluster.Raw
			inputResources["cephdeployments"].(*cephlcmv1alpha1.CephDeploymentList).Items[0].Spec.Cluster.Raw = unitinputs.ConvertJSONToYaml(cephDeploy)
			assert.Equal(t, expectedResources, inputResources)
			faketestclients.CleanupFakeClientReactions(c.api.CephLcmclientset)
		})
	}
}
