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

	cephv1 "github.com/rook/rook/pkg/apis/ceph.rook.io/v1"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
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
			name: "failed to transform cephdeployment due dir path fieldconflict",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				depl := unitinputs.CephDeploymentDeprecated.DeepCopy()
				depl.Spec.Cluster = &cephlcmv1alpha1.CephCluster{
					ClusterSpec: cephv1.ClusterSpec{
						DataDirHostPath: "/var/lib/present",
					},
				}
				return depl
			}(),
			expectedError: "value from deprecated field spec.dataDirHostPath=/var/lib/custom-path is conflicting with spec.cluster.dataDirHostPath=/var/lib/present",
		},
		{
			name: "failed to transform cephdeployment due network field conflict",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				depl := unitinputs.CephDeploymentDeprecated.DeepCopy()
				depl.Spec.Cluster = &cephlcmv1alpha1.CephCluster{
					ClusterSpec: cephv1.ClusterSpec{
						Network: cephv1.NetworkSpec{
							AddressRanges: &cephv1.AddressRangesSpec{
								Public:  []cephv1.CIDR{cephv1.CIDR("192.168.0.0/16")},
								Cluster: []cephv1.CIDR{cephv1.CIDR("127.0.0.0/16")},
							},
						},
					},
				}
				return depl
			}(),
			expectedError: "networks from deprecated field spec.network are conflicting with spec.cluster.network.addressRanges",
		},
		{
			name: "failed to transform cephdeployment due resources field conflict",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				depl := unitinputs.CephDeploymentDeprecated.DeepCopy()
				depl.Spec.Cluster = &cephlcmv1alpha1.CephCluster{
					ClusterSpec: cephv1.ClusterSpec{
						Resources: cephv1.ResourceSpec{"mon": v1.ResourceRequirements{}},
					},
				}
				return depl
			}(),
			expectedError: "cluster resources from deprecated field spec.hyperconverge.resources are conflicting with spec.cluster.resources",
		},
		{
			name: "failed to transform cephdeployment due placement field conflict",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				depl := unitinputs.CephDeploymentDeprecated.DeepCopy()
				depl.Spec.Cluster = &cephlcmv1alpha1.CephCluster{
					ClusterSpec: cephv1.ClusterSpec{
						Placement: cephv1.PlacementSpec{"mon": cephv1.Placement{
							Tolerations: []v1.Toleration{
								{
									Key:      "test.kubernetes.io/testkey",
									Effect:   "Schedule",
									Operator: "Exists",
								},
							}}},
					},
				}
				return depl
			}(),
			expectedError: "placement tolerations from deprecated field spec.hyperconverge.tolerations[mon] are conflicting with spec.cluster.placement[mon].tolerations",
		},
		{
			name:            "transform regular cephdeployment",
			cephDpl:         unitinputs.CephDeploymentDeprecated.DeepCopy(),
			expectedCephDpl: &unitinputs.CephDeploymentMigrated,
			updated:         true,
		},
		{
			name: "transform regular cephdeployment with multinetworks",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				depl := unitinputs.CephDeploymentDeprecated.DeepCopy()
				depl.Spec.Network.ClusterNet = "192.168.0.0/24,192.168.1.0/24"
				depl.Spec.Network.PublicNet = "10.168.0.0/24,10.168.1.0/24"
				return depl
			}(),
			expectedCephDpl: func() *cephlcmv1alpha1.CephDeployment {
				depl := unitinputs.CephDeploymentMigrated.DeepCopy()
				depl.Spec.Cluster.Network.AddressRanges.Cluster = []cephv1.CIDR{cephv1.CIDR("192.168.0.0/24"), cephv1.CIDR("192.168.1.0/24")}
				depl.Spec.Cluster.Network.AddressRanges.Public = []cephv1.CIDR{cephv1.CIDR("10.168.0.0/24"), cephv1.CIDR("10.168.1.0/24")}
				return depl
			}(),
			updated: true,
		},
		{
			name:            "transform multus cephdeployment",
			cephDpl:         unitinputs.CephDeploymentMultusDeprecated.DeepCopy(),
			expectedCephDpl: &unitinputs.CephDeploymentMultusMigrated,
			updated:         true,
		},
		{
			name:            "transform external cephdeployment",
			cephDpl:         unitinputs.CephDeployExternalDeprecated.DeepCopy(),
			expectedCephDpl: &unitinputs.CephDeployExternalMigrated,
			updated:         true,
		},
		{
			name:            "no cephdeployment transform",
			cephDpl:         unitinputs.CephDeploymentMigrated.DeepCopy(),
			expectedCephDpl: &unitinputs.CephDeploymentMigrated,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := fakeDeploymentConfig(&deployConfig{cephDpl: test.cephDpl}, nil)
			inputResources := map[string]runtime.Object{"cephdeployments": &cephlcmv1alpha1.CephDeploymentList{Items: []cephlcmv1alpha1.CephDeployment{*test.cephDpl.DeepCopy()}}}
			expectedResources := map[string]runtime.Object{}
			if test.expectedCephDpl == nil {
				expectedResources["cephdeployments"] = &cephlcmv1alpha1.CephDeploymentList{Items: []cephlcmv1alpha1.CephDeployment{*test.cephDpl.DeepCopy()}}
			} else {
				expectedResources["cephdeployments"] = &cephlcmv1alpha1.CephDeploymentList{Items: []cephlcmv1alpha1.CephDeployment{*test.expectedCephDpl}}
			}
			faketestclients.FakeReaction(c.api.CephLcmclientset, "update", []string{"cephdeployments"}, inputResources, nil)

			updated, err := c.ensureDeprecatedFields()
			if test.expectedError == "" {
				assert.Nil(t, err)
			} else {
				assert.Equal(t, test.expectedError, err.Error())
			}
			assert.Equal(t, test.updated, updated)
			assert.Equal(t, expectedResources, inputResources)
			faketestclients.CleanupFakeClientReactions(c.api.CephLcmclientset)
		})
	}
}
