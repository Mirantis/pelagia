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
	"github.com/stretchr/testify/assert"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/runtime"

	cephlcmv1alpha1 "github.com/Mirantis/pelagia/pkg/apis/ceph.pelagia.lcm/v1alpha1"
	faketestclients "github.com/Mirantis/pelagia/test/unit/clients"
	unitinputs "github.com/Mirantis/pelagia/test/unit/inputs"
)

func TestEnsureNetworkPolicy(t *testing.T) {
	tests := []struct {
		name                 string
		cephDpl              *cephlcmv1alpha1.CephDeployment
		networkPolicyEnabled string
		inputResources       map[string]runtime.Object
		apiErrors            map[string]error
		expectedResources    map[string]runtime.Object
		expectedChange       bool
		expectedError        string
	}{
		{
			name:    "ensure network policy - failed to create resources",
			cephDpl: unitinputs.BaseCephDeployment.DeepCopy(),
			inputResources: map[string]runtime.Object{
				"networkpolicies": &networkingv1.NetworkPolicyList{},
			},
			apiErrors:            map[string]error{"create-networkpolicies": errors.New("failed to create")},
			networkPolicyEnabled: "true",
			expectedError:        "failed to manage network policies",
		},
		{
			name:    "ensure network policy - create base network policies",
			cephDpl: unitinputs.BaseCephDeployment.DeepCopy(),
			inputResources: map[string]runtime.Object{
				"networkpolicies": &networkingv1.NetworkPolicyList{},
			},
			expectedResources: map[string]runtime.Object{
				"networkpolicies": &networkingv1.NetworkPolicyList{
					Items: []networkingv1.NetworkPolicy{
						unitinputs.NetworkPolicyMgr, unitinputs.NetworkPolicyMon, unitinputs.NetworkPolicyOsd,
					},
				},
			},
			expectedChange:       true,
			networkPolicyEnabled: "true",
		},
		{
			name:    "ensure network policy - create all network policies",
			cephDpl: unitinputs.CephDeployNonMosk.DeepCopy(),
			inputResources: map[string]runtime.Object{
				"networkpolicies": &networkingv1.NetworkPolicyList{},
			},
			expectedResources: map[string]runtime.Object{
				"networkpolicies": &networkingv1.NetworkPolicyList{
					Items: []networkingv1.NetworkPolicy{
						unitinputs.NetworkPolicyMds, unitinputs.NetworkPolicyMgr, unitinputs.NetworkPolicyMon, unitinputs.NetworkPolicyOsd, unitinputs.NetworkPolicyRgw,
					},
				},
			},
			expectedChange:       true,
			networkPolicyEnabled: "true",
		},
		{
			name:    "ensure network policy - update all network policies",
			cephDpl: unitinputs.CephDeployNonMosk.DeepCopy(),
			inputResources: map[string]runtime.Object{
				"networkpolicies": &networkingv1.NetworkPolicyList{
					Items: []networkingv1.NetworkPolicy{
						unitinputs.GetNetworkPolicy("rook-ceph-mgr", nil),
						unitinputs.GetNetworkPolicy("rook-ceph-mon", nil),
						unitinputs.GetNetworkPolicy("rook-ceph-osd", nil),
						unitinputs.GetNetworkPolicy("rook-ceph-rgw", nil),
						unitinputs.GetNetworkPolicy("rook-ceph-mds", nil),
					},
				},
			},
			expectedResources: map[string]runtime.Object{
				"networkpolicies": &networkingv1.NetworkPolicyList{
					Items: []networkingv1.NetworkPolicy{
						unitinputs.NetworkPolicyMgr, unitinputs.NetworkPolicyMon, unitinputs.NetworkPolicyOsd, unitinputs.NetworkPolicyRgw, unitinputs.NetworkPolicyMds,
					},
				},
			},
			expectedChange:       true,
			networkPolicyEnabled: "true",
		},
		{
			name:    "ensure network policy - remove not required",
			cephDpl: unitinputs.BaseCephDeployment.DeepCopy(),
			inputResources: map[string]runtime.Object{
				"networkpolicies": &networkingv1.NetworkPolicyList{
					Items: []networkingv1.NetworkPolicy{
						unitinputs.NetworkPolicyMgr, unitinputs.NetworkPolicyMon, unitinputs.NetworkPolicyOsd, unitinputs.NetworkPolicyRgw, unitinputs.NetworkPolicyMds,
					},
				},
			},
			expectedResources: map[string]runtime.Object{
				"networkpolicies": &networkingv1.NetworkPolicyList{
					Items: []networkingv1.NetworkPolicy{
						unitinputs.NetworkPolicyMgr, unitinputs.NetworkPolicyMon, unitinputs.NetworkPolicyOsd,
					},
				},
			},
			expectedChange:       true,
			networkPolicyEnabled: "true",
		},
		{
			name:    "ensure network policy - nothing to do",
			cephDpl: unitinputs.CephDeployNonMosk.DeepCopy(),
			inputResources: map[string]runtime.Object{
				"networkpolicies": &networkingv1.NetworkPolicyList{
					Items: []networkingv1.NetworkPolicy{
						unitinputs.NetworkPolicyMgr, unitinputs.NetworkPolicyMon, unitinputs.NetworkPolicyOsd, unitinputs.NetworkPolicyRgw, unitinputs.NetworkPolicyMds,
					},
				},
			},
			networkPolicyEnabled: "true",
		},
		{
			name:    "ensure network policy - cleanup failed",
			cephDpl: unitinputs.BaseCephDeployment.DeepCopy(),
			inputResources: map[string]runtime.Object{
				"networkpolicies": &networkingv1.NetworkPolicyList{Items: []networkingv1.NetworkPolicy{
					unitinputs.NetworkPolicyMgr, unitinputs.NetworkPolicyMon, unitinputs.NetworkPolicyOsd, unitinputs.NetworkPolicyRgw, unitinputs.NetworkPolicyMds,
				}},
			},
			apiErrors: map[string]error{
				"delete-networkpolicies": errors.New("failed to delete netpol"),
			},
			expectedError: "failed to clean up network policies",
		},
		{
			name:    "ensure network policy - cleanup ok",
			cephDpl: unitinputs.CephDeployNonMosk.DeepCopy(),
			inputResources: map[string]runtime.Object{
				"networkpolicies": &networkingv1.NetworkPolicyList{
					Items: []networkingv1.NetworkPolicy{
						unitinputs.NetworkPolicyMgr, unitinputs.NetworkPolicyMon, unitinputs.NetworkPolicyOsd, unitinputs.NetworkPolicyRgw, unitinputs.NetworkPolicyMds,
					},
				},
			},
			expectedResources: map[string]runtime.Object{
				"networkpolicies": &networkingv1.NetworkPolicyList{Items: []networkingv1.NetworkPolicy{}},
			},
			expectedChange: true,
		},
		{
			name:    "ensure network policy - no netpol present",
			cephDpl: unitinputs.CephDeployNonMosk.DeepCopy(),
			inputResources: map[string]runtime.Object{
				"networkpolicies": &networkingv1.NetworkPolicyList{},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := fakeDeploymentConfig(&deployConfig{cephDpl: test.cephDpl}, map[string]string{"DEPLOYMENT_NETPOL_ENABLED": test.networkPolicyEnabled})
			faketestclients.FakeReaction(c.api.Kubeclientset.NetworkingV1(), "list", []string{"networkpolicies"}, test.inputResources, nil)
			faketestclients.FakeReaction(c.api.Kubeclientset.NetworkingV1(), "get", []string{"networkpolicies"}, test.inputResources, test.apiErrors)
			faketestclients.FakeReaction(c.api.Kubeclientset.NetworkingV1(), "create", []string{"networkpolicies"}, test.inputResources, test.apiErrors)
			faketestclients.FakeReaction(c.api.Kubeclientset.NetworkingV1(), "update", []string{"networkpolicies"}, test.inputResources, test.apiErrors)
			faketestclients.FakeReaction(c.api.Kubeclientset.NetworkingV1(), "delete", []string{"networkpolicies"}, test.inputResources, test.apiErrors)
			test.expectedResources = faketestclients.PrepareExpectedResources(test.inputResources, test.expectedResources)

			changed, err := c.ensureNetworkPolicy()
			if test.expectedError != "" {
				assert.Equal(t, false, changed)
				assert.NotNil(t, err)
				assert.Equal(t, test.expectedError, err.Error())
			} else {
				assert.Equal(t, test.expectedChange, changed)
				assert.Nil(t, err)
			}
			assert.Equal(t, test.expectedResources, test.inputResources)
			faketestclients.CleanupFakeClientReactions(c.api.Kubeclientset.NetworkingV1())
		})
	}
}

func TestManageNetworkPolicy(t *testing.T) {
	tests := []struct {
		name              string
		policy            networkingv1.NetworkPolicy
		inputResources    map[string]runtime.Object
		expectedResources map[string]runtime.Object
		apiErrors         map[string]error
		expectedChange    bool
		expectedError     string
	}{
		{
			name:   "ensure rgw network policy - netpol get failed",
			policy: unitinputs.NetworkPolicyMgr,
			inputResources: map[string]runtime.Object{
				"networkpolicies": &networkingv1.NetworkPolicyList{},
			},
			apiErrors: map[string]error{
				"get-networkpolicies": errors.New("failed to get netpol"),
			},
			expectedError: "failed to check 'rook-ceph/rook-ceph-mgr-policy' networkpolicy: failed to get netpol",
		},
		{
			name:   "ensure rgw network policy - netpol create failed",
			policy: unitinputs.NetworkPolicyMgr,
			inputResources: map[string]runtime.Object{
				"networkpolicies": &networkingv1.NetworkPolicyList{},
			},
			apiErrors: map[string]error{
				"create-networkpolicies": errors.New("failed to create netpol"),
			},
			expectedError: "failed to create 'rook-ceph/rook-ceph-mgr-policy' networkpolicy: failed to create netpol",
		},
		{
			name:   "ensure rgw network policy - netpol created",
			policy: unitinputs.NetworkPolicyMgr,
			inputResources: map[string]runtime.Object{
				"networkpolicies": &networkingv1.NetworkPolicyList{},
			},
			expectedResources: map[string]runtime.Object{
				"networkpolicies": &networkingv1.NetworkPolicyList{Items: []networkingv1.NetworkPolicy{unitinputs.NetworkPolicyMgr}},
			},
			expectedChange: true,
		},
		{
			name:   "ensure rgw network policy - netpol update failed",
			policy: unitinputs.NetworkPolicyMgr,
			inputResources: map[string]runtime.Object{
				"networkpolicies": &networkingv1.NetworkPolicyList{Items: []networkingv1.NetworkPolicy{unitinputs.GetNetworkPolicy("rook-ceph-mgr", nil)}},
			},
			apiErrors: map[string]error{
				"update-networkpolicies": errors.New("failed to update netpol"),
			},
			expectedError: "failed to update 'rook-ceph/rook-ceph-mgr-policy' networkpolicy: failed to update netpol",
		},
		{
			name:   "ensure rgw network policy - netpol updated ports",
			policy: unitinputs.NetworkPolicyMgr,
			inputResources: map[string]runtime.Object{
				"networkpolicies": &networkingv1.NetworkPolicyList{Items: []networkingv1.NetworkPolicy{unitinputs.GetNetworkPolicy("rook-ceph-mgr", nil)}},
			},
			expectedResources: map[string]runtime.Object{
				"networkpolicies": &networkingv1.NetworkPolicyList{Items: []networkingv1.NetworkPolicy{unitinputs.NetworkPolicyMgr}},
			},
			expectedChange: true,
		},
		{
			name:   "ensure rgw network policy - netpol updated label",
			policy: unitinputs.NetworkPolicyMgr,
			inputResources: map[string]runtime.Object{
				"networkpolicies": &networkingv1.NetworkPolicyList{Items: []networkingv1.NetworkPolicy{
					func() networkingv1.NetworkPolicy {
						p := unitinputs.GetNetworkPolicy("rook-ceph-mgr", nil)
						p.Labels = nil
						return p
					}(),
				}},
			},
			expectedResources: map[string]runtime.Object{
				"networkpolicies": &networkingv1.NetworkPolicyList{Items: []networkingv1.NetworkPolicy{unitinputs.NetworkPolicyMgr}},
			},
			expectedChange: true,
		},
		{
			name:   "ensure rgw network policy - nothing to do",
			policy: unitinputs.NetworkPolicyMgr,
			inputResources: map[string]runtime.Object{
				"networkpolicies": &networkingv1.NetworkPolicyList{Items: []networkingv1.NetworkPolicy{unitinputs.NetworkPolicyMgr}},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := fakeDeploymentConfig(nil, nil)
			faketestclients.FakeReaction(c.api.Kubeclientset.NetworkingV1(), "get", []string{"networkpolicies"}, test.inputResources, test.apiErrors)
			faketestclients.FakeReaction(c.api.Kubeclientset.NetworkingV1(), "create", []string{"networkpolicies"}, test.inputResources, test.apiErrors)
			faketestclients.FakeReaction(c.api.Kubeclientset.NetworkingV1(), "update", []string{"networkpolicies"}, test.inputResources, test.apiErrors)
			faketestclients.FakeReaction(c.api.Kubeclientset.NetworkingV1(), "delete", []string{"networkpolicies"}, test.inputResources, test.apiErrors)
			test.expectedResources = faketestclients.PrepareExpectedResources(test.inputResources, test.expectedResources)

			changed, err := c.manageNetworkPolicy(test.policy)
			if test.expectedError != "" {
				assert.Equal(t, false, changed)
				assert.NotNil(t, err)
				assert.Equal(t, test.expectedError, err.Error())
			} else {
				assert.Equal(t, test.expectedChange, changed)
				assert.Nil(t, err)
			}
			assert.Equal(t, test.expectedResources, test.inputResources)
			faketestclients.CleanupFakeClientReactions(c.api.Kubeclientset.NetworkingV1())
		})
	}
}

func TestDeleteNetworkPolicy(t *testing.T) {
	tests := []struct {
		name              string
		inputResources    map[string]runtime.Object
		expectedResources map[string]runtime.Object
		apiErrors         map[string]error
		expectedError     string
		removed           bool
	}{
		{
			name:           "delete network policy - failed to check",
			inputResources: map[string]runtime.Object{},
			expectedError:  "failed to list networkpolicies",
		},
		{
			name: "delete network policy - cleanup failed",
			inputResources: map[string]runtime.Object{
				"networkpolicies": &networkingv1.NetworkPolicyList{
					Items: []networkingv1.NetworkPolicy{
						unitinputs.NetworkPolicyMgr, unitinputs.NetworkPolicyMon, unitinputs.NetworkPolicyOsd, unitinputs.NetworkPolicyRgw, unitinputs.NetworkPolicyMds,
					},
				},
			},
			apiErrors: map[string]error{
				"delete-networkpolicies": errors.New("failed to delete netpol"),
			},
			expectedError: "networkpolicies remove failed",
		},
		{
			name: "delete network policy - cleanup ok",
			inputResources: map[string]runtime.Object{
				"networkpolicies": &networkingv1.NetworkPolicyList{
					Items: []networkingv1.NetworkPolicy{
						unitinputs.NetworkPolicyMgr, unitinputs.NetworkPolicyMon, unitinputs.NetworkPolicyOsd, unitinputs.NetworkPolicyRgw, unitinputs.NetworkPolicyMds,
						func() networkingv1.NetworkPolicy {
							p := unitinputs.GetNetworkPolicy("custom-policy", nil)
							p.Labels = nil
							return p
						}(),
					},
				},
			},
			expectedResources: map[string]runtime.Object{
				"networkpolicies": &networkingv1.NetworkPolicyList{Items: []networkingv1.NetworkPolicy{
					func() networkingv1.NetworkPolicy {
						p := unitinputs.GetNetworkPolicy("custom-policy", nil)
						p.Labels = nil
						return p
					}(),
				}},
			},
		},
		{
			name: "delete network policy - nothing to cleanup",
			inputResources: map[string]runtime.Object{
				"networkpolicies": &networkingv1.NetworkPolicyList{Items: []networkingv1.NetworkPolicy{}},
			},
			removed: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := fakeDeploymentConfig(nil, nil)
			faketestclients.FakeReaction(c.api.Kubeclientset.NetworkingV1(), "list", []string{"networkpolicies"}, test.inputResources, nil)
			faketestclients.FakeReaction(c.api.Kubeclientset.NetworkingV1(), "delete", []string{"networkpolicies"}, test.inputResources, test.apiErrors)
			test.expectedResources = faketestclients.PrepareExpectedResources(test.inputResources, test.expectedResources)

			removed, err := c.cleanupNetworkPolicy()
			assert.Equal(t, test.removed, removed)
			if test.expectedError != "" {
				assert.NotNil(t, err)
				assert.Equal(t, test.expectedError, err.Error())
			} else {
				assert.Nil(t, err)
			}
			assert.Equal(t, test.expectedResources, test.inputResources)
			faketestclients.CleanupFakeClientReactions(c.api.Kubeclientset.NetworkingV1())
		})
	}
}
