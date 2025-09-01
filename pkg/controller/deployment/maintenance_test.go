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

	lcmcommon "github.com/Mirantis/pelagia/pkg/common"
	faketestclients "github.com/Mirantis/pelagia/test/unit/clients"
	unitinputs "github.com/Mirantis/pelagia/test/unit/inputs"
)

func TestCephUpgradeAllowed(t *testing.T) {
	tests := []struct {
		name           string
		osdplst        runtime.Object
		varError       bool
		upgradeAllowed bool
		expectedError  string
	}{
		{
			name:          "get envvar failed - fail",
			varError:      true,
			expectedError: "failed to get current cluster release: env variable 'CEPH_CONTROLLER_CLUSTER_RELEASE' is not set",
		},
		{
			name:          "failed to get osdpl list",
			osdplst:       unitinputs.GetOpenstackDeploymentStatusList("", "", false),
			expectedError: "failed to get openstackdeploymentstatus state and release: OpenstackDeploymentStatus required values in status.osdpl not found",
		},
		{
			name:           "no osdpl present - allow upgrade",
			upgradeAllowed: true,
		},
		{
			name:    "osdpl has different release - disallow upgrade",
			osdplst: unitinputs.GetOpenstackDeploymentStatusList("new", "APPLIED", true),
		},
		{
			name:    "osdpl has current release, but not ready - disallow upgrade",
			osdplst: unitinputs.GetOpenstackDeploymentStatusList("new", "ERROR", true),
		},
		{
			name:           "osdpl has current release, ready - allow upgrade",
			osdplst:        unitinputs.GetOpenstackDeploymentStatusList("cur", "APPLIED", true),
			upgradeAllowed: true,
		},
	}
	oldLookup := lcmcommon.LookupEnv
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := fakeDeploymentConfig(nil, nil)
			if test.osdplst != nil {
				c.api.Client = faketestclients.GetClient(faketestclients.GetClientBuilder().WithRuntimeObjects(test.osdplst))
			} else {
				c.api.Client = faketestclients.GetClient(nil)
			}

			lcmcommon.LookupEnv = func(key string) (string, bool) {
				if !test.varError {
					if key == "CEPH_CONTROLLER_CLUSTER_RELEASE" {
						return "cur", true
					}
				}
				return "", false
			}

			allowed, err := c.cephUpgradeAllowed()
			if test.expectedError != "" {
				assert.NotNil(t, err)
				assert.Equal(t, test.expectedError, err.Error())
			} else {
				assert.Nil(t, err)
			}
			assert.Equal(t, test.upgradeAllowed, allowed)
		})
	}
	lcmcommon.LookupEnv = oldLookup
}
