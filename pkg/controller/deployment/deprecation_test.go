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
		expectedCephDpl *cephlcmv1alpha1.CephDeployment
		expectedError   string
		updated         bool
	}{
		{
			name:            "no transform",
			cephDpl:         unitinputs.BaseCephDeployment.DeepCopy(),
			expectedCephDpl: &unitinputs.BaseCephDeployment,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := fakeDeploymentConfig(nil, nil)
			inputResources := map[string]runtime.Object{"cephdeployments": &cephlcmv1alpha1.CephDeploymentList{Items: []cephlcmv1alpha1.CephDeployment{*test.cephDpl}}}
			expectedResources := map[string]runtime.Object{"cephdeployments": &cephlcmv1alpha1.CephDeploymentList{Items: []cephlcmv1alpha1.CephDeployment{*test.expectedCephDpl}}}
			faketestclients.FakeReaction(c.api.CephLcmclientset, "update", []string{"cephdeployments"}, inputResources, nil)

			err := c.ensureDeprecatedFields()
			if test.expectedError == "" {
				assert.Nil(t, err)
			} else {
				assert.Equal(t, test.expectedError, err.Error())
			}
			assert.Equal(t, expectedResources, inputResources)
			faketestclients.CleanupFakeClientReactions(c.api.CephLcmclientset)
		})
	}
}
