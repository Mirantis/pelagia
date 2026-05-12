/*
Copyright 2026 Mirantis IT.

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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	gatewayapi "sigs.k8s.io/gateway-api/apis/v1"

	cephlcmv1alpha1 "github.com/Mirantis/pelagia/pkg/apis/ceph.pelagia.lcm/v1alpha1"
	lcmcommon "github.com/Mirantis/pelagia/pkg/common"
	faketestclients "github.com/Mirantis/pelagia/test/unit/clients"
	unitinputs "github.com/Mirantis/pelagia/test/unit/inputs"
)

func TestGenerateHTTPRoute(t *testing.T) {
	tests := []struct {
		name          string
		cephDpl       *cephlcmv1alpha1.CephDeployment
		httpRoute     cephlcmv1alpha1.CephDeploymentHTTPRoute
		expectedRoute gatewayapi.HTTPRoute
	}{
		{
			name:    "generate default httproute for rockoon",
			cephDpl: &unitinputs.CephDeployMosk,
			httpRoute: cephlcmv1alpha1.CephDeploymentHTTPRoute{
				Name:            "rgw-store-openstack-route",
				ObjectStoreName: "rgw-store",
				Spec: runtime.RawExtension{
					Raw: []byte(`{"hostnames": ["rgw-store.openstack.com"]}`),
				},
			},
			expectedRoute: unitinputs.DefaultMoskHTTPRoute,
		},
		{
			name:    "generate custom httproute",
			cephDpl: &unitinputs.CephDeployNonMosk,
			httpRoute: cephlcmv1alpha1.CephDeploymentHTTPRoute{
				Name:            "rgw-store-extra-route",
				ObjectStoreName: "rgw-store",
				Spec: runtime.RawExtension{
					Raw: unitinputs.ConvertStructToRaw(
						gatewayapi.HTTPRouteSpec{
							Hostnames: []gatewayapi.Hostname{gatewayapi.Hostname("rgw-store.example.com")},
							CommonRouteSpec: gatewayapi.CommonRouteSpec{
								ParentRefs: []gatewayapi.ParentReference{{Kind: lcmcommon.PtrTo(gatewayapi.Kind("Gateway2"))}},
							},
							Rules: []gatewayapi.HTTPRouteRule{
								{
									Matches: []gatewayapi.HTTPRouteMatch{
										{Path: &gatewayapi.HTTPPathMatch{Value: lcmcommon.PtrTo("/extra")}},
									},
									BackendRefs: []gatewayapi.HTTPBackendRef{
										{
											BackendRef: gatewayapi.BackendRef{
												BackendObjectReference: gatewayapi.BackendObjectReference{
													Kind: lcmcommon.PtrTo(gatewayapi.Kind("Service")),
													Name: gatewayapi.ObjectName("rgw-store-extra"),
													Port: lcmcommon.PtrTo(int32(8080)),
												},
											},
										},
									},
								},
							},
						},
					),
				},
			},
			expectedRoute: gatewayapi.HTTPRoute{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "rgw-store-extra-route",
					Namespace: "rook-ceph",
					Labels: map[string]string{
						"app":                          "rook-ceph-rgw",
						"external_access":              "rgw",
						"rook_object_store":            "rgw-store",
						"app.kubernetes.io/created-by": "pelagia-deployment-controller",
						"app.kubernetes.io/managed-by": "pelagia-deployment-controller",
						"app.kubernetes.io/part-of":    "ceph.pelagia.lcm",
					},
				},
				Spec: gatewayapi.HTTPRouteSpec{
					Hostnames: []gatewayapi.Hostname{gatewayapi.Hostname("rgw-store.example.com")},
					CommonRouteSpec: gatewayapi.CommonRouteSpec{
						ParentRefs: []gatewayapi.ParentReference{
							{
								Kind: lcmcommon.PtrTo(gatewayapi.Kind("Gateway2")),
							},
						},
					},
					Rules: []gatewayapi.HTTPRouteRule{
						{
							Matches: []gatewayapi.HTTPRouteMatch{
								{
									Path: &gatewayapi.HTTPPathMatch{Value: lcmcommon.PtrTo("/extra")},
								},
							},
							BackendRefs: []gatewayapi.HTTPBackendRef{
								{
									BackendRef: gatewayapi.BackendRef{
										BackendObjectReference: gatewayapi.BackendObjectReference{
											Kind: lcmcommon.PtrTo(gatewayapi.Kind("Service")),
											Name: gatewayapi.ObjectName("rgw-store-extra"),
											Port: lcmcommon.PtrTo(int32(8080)),
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := fakeDeploymentConfig(&deployConfig{cephDpl: test.cephDpl}, nil)

			httpRoute := c.generateHTTPRoute(test.httpRoute)
			assert.Equal(t, test.expectedRoute, httpRoute)
		})
	}
}

func TestEnsureGatewayHTTPRoutes(t *testing.T) {
	tests := []struct {
		name              string
		cephDpl           *cephlcmv1alpha1.CephDeployment
		extraLcmConfig    map[string]string
		inputResources    map[string]runtime.Object
		expectedResources map[string]runtime.Object
		apiErrors         map[string]error
		stateChanged      bool
		expectedError     string
	}{
		{
			name:          "failed to list httproutes",
			expectedError: "failed to list rgw gateway httproutes to ensure rgw httproutes: failed to list httproutes",
		},
		{
			name:    "nothing to do, no httroutes present and no expected",
			cephDpl: &unitinputs.CephDeployNonMosk,
			inputResources: map[string]runtime.Object{
				"httproutes": &gatewayapi.HTTPRouteList{},
			},
		},
		{
			name:    "default mosk expected, but openstack shared namespace is not set",
			cephDpl: &unitinputs.CephDeployMosk,
			extraLcmConfig: map[string]string{
				"DEPLOYMENT_OPENSTACK_CEPH_SHARED_NAMESPACE": "",
			},
			inputResources: map[string]runtime.Object{
				"httproutes": &gatewayapi.HTTPRouteList{},
				"secrets":    &corev1.SecretList{},
			},
			expectedError: "CephRGW object storage 'rgw-store' has specified for Openstack usage, but Pelagia lcmconfig has no var 'DEPLOYMENT_OPENSTACK_CEPH_SHARED_NAMESPACE' set",
		},
		{
			name:    "default mosk expected, but mosk secret is not present yet",
			cephDpl: &unitinputs.CephDeployMosk,
			inputResources: map[string]runtime.Object{
				"httproutes": &gatewayapi.HTTPRouteList{},
				"secrets":    &corev1.SecretList{},
			},
		},
		{
			name:    "default mosk expected, but mosk secret failed to get",
			cephDpl: &unitinputs.CephDeployMosk,
			inputResources: map[string]runtime.Object{
				"httproutes": &gatewayapi.HTTPRouteList{},
				"secrets":    &corev1.SecretList{},
			},
			apiErrors: map[string]error{
				"get-secrets": errors.New("failed to get secret"),
			},
			expectedError: "failed to ensure default rgw gateway httproute",
		},
		{
			name:    "default mosk httproute created",
			cephDpl: &unitinputs.CephDeployMosk,
			inputResources: map[string]runtime.Object{
				"httproutes": &gatewayapi.HTTPRouteList{},
				"secrets":    &corev1.SecretList{Items: []corev1.Secret{unitinputs.OpenstackRgwCredsSecret}},
			},
			expectedResources: map[string]runtime.Object{
				"httproutes": &gatewayapi.HTTPRouteList{Items: []gatewayapi.HTTPRoute{unitinputs.DefaultMoskHTTPRoute}},
			},
			stateChanged: true,
		},
		{
			name:    "default mosk httproute exists and labels updated",
			cephDpl: &unitinputs.CephDeployMosk,
			inputResources: map[string]runtime.Object{
				"httproutes": &gatewayapi.HTTPRouteList{Items: []gatewayapi.HTTPRoute{
					func() gatewayapi.HTTPRoute {
						route := unitinputs.DefaultMoskHTTPRoute.DeepCopy()
						route.Labels = nil
						return *route
					}(),
				}},
				"secrets": &corev1.SecretList{Items: []corev1.Secret{unitinputs.OpenstackRgwCredsSecret}},
			},
			expectedResources: map[string]runtime.Object{
				"httproutes": &gatewayapi.HTTPRouteList{Items: []gatewayapi.HTTPRoute{unitinputs.DefaultMoskHTTPRoute}},
			},
			stateChanged: true,
		},
		{
			name:    "default mosk httproute exists and no changes",
			cephDpl: &unitinputs.CephDeployMosk,
			inputResources: map[string]runtime.Object{
				"httproutes": &gatewayapi.HTTPRouteList{Items: []gatewayapi.HTTPRoute{unitinputs.DefaultMoskHTTPRoute}},
				"secrets":    &corev1.SecretList{Items: []corev1.Secret{unitinputs.OpenstackRgwCredsSecret}},
			},
		},
		{
			name:    "default mosk httproute removed",
			cephDpl: &unitinputs.CephDeployNonMosk,
			inputResources: map[string]runtime.Object{
				"httproutes": &gatewayapi.HTTPRouteList{Items: []gatewayapi.HTTPRoute{unitinputs.DefaultMoskHTTPRoute}},
			},
			expectedResources: map[string]runtime.Object{
				"httproutes": &gatewayapi.HTTPRouteList{Items: []gatewayapi.HTTPRoute{}},
			},
			stateChanged: true,
		},
		{
			name:    "default mosk httproute remove failed",
			cephDpl: &unitinputs.CephDeployNonMosk,
			inputResources: map[string]runtime.Object{
				"httproutes": &gatewayapi.HTTPRouteList{Items: []gatewayapi.HTTPRoute{unitinputs.DefaultMoskHTTPRoute}},
			},
			apiErrors: map[string]error{
				"delete-httproutes": errors.New("failed to delete httproute"),
			},
			expectedError: "failed to ensure rgw gateway httproute(s)",
		},
		{
			name:    "create httproute from spec",
			cephDpl: &unitinputs.CephDeployNonMoskWithGatewayRoute,
			inputResources: map[string]runtime.Object{
				"httproutes": &gatewayapi.HTTPRouteList{},
			},
			expectedResources: map[string]runtime.Object{
				"httproutes": &gatewayapi.HTTPRouteList{Items: []gatewayapi.HTTPRoute{unitinputs.DefaultBaseHTTPRoute}},
			},
			stateChanged: true,
		},
		{
			name:    "create httproute from spec failed",
			cephDpl: &unitinputs.CephDeployNonMoskWithGatewayRoute,
			inputResources: map[string]runtime.Object{
				"httproutes": &gatewayapi.HTTPRouteList{},
			},
			apiErrors: map[string]error{
				"create-httproutes": errors.New("failed to create httproute"),
			},
			expectedError: "failed to ensure rgw gateway httproute(s)",
		},
		{
			name:    "update httproute from spec",
			cephDpl: &unitinputs.CephDeployNonMoskWithGatewayRoute,
			inputResources: map[string]runtime.Object{
				"httproutes": &gatewayapi.HTTPRouteList{Items: []gatewayapi.HTTPRoute{
					func() gatewayapi.HTTPRoute {
						route := unitinputs.DefaultBaseHTTPRoute.DeepCopy()
						route.Spec.ParentRefs = nil
						return *route
					}(),
				}},
			},
			expectedResources: map[string]runtime.Object{
				"httproutes": &gatewayapi.HTTPRouteList{Items: []gatewayapi.HTTPRoute{unitinputs.DefaultBaseHTTPRoute}},
			},
			stateChanged: true,
		},
		{
			name:    "update httproute from spec failed",
			cephDpl: &unitinputs.CephDeployNonMoskWithGatewayRoute,
			inputResources: map[string]runtime.Object{
				"httproutes": &gatewayapi.HTTPRouteList{Items: []gatewayapi.HTTPRoute{
					func() gatewayapi.HTTPRoute {
						route := unitinputs.DefaultBaseHTTPRoute.DeepCopy()
						route.Spec.ParentRefs = nil
						return *route
					}(),
				}},
			},
			apiErrors: map[string]error{
				"update-httproutes": errors.New("failed to update httproute"),
			},
			expectedError: "failed to ensure rgw gateway httproute(s)",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := fakeDeploymentConfig(&deployConfig{cephDpl: test.cephDpl}, test.extraLcmConfig)

			faketestclients.FakeReaction(c.api.Gatewayclientset, "list", []string{"httproutes"}, test.inputResources, test.apiErrors)
			faketestclients.FakeReaction(c.api.Gatewayclientset, "create", []string{"httproutes"}, test.inputResources, test.apiErrors)
			faketestclients.FakeReaction(c.api.Gatewayclientset, "update", []string{"httproutes"}, test.inputResources, test.apiErrors)
			faketestclients.FakeReaction(c.api.Gatewayclientset, "delete", []string{"httproutes"}, test.inputResources, test.apiErrors)
			faketestclients.FakeReaction(c.api.Kubeclientset.CoreV1(), "get", []string{"secrets"}, test.inputResources, test.apiErrors)
			test.expectedResources = faketestclients.PrepareExpectedResources(test.inputResources, test.expectedResources)

			changed, err := c.ensureGatewayHTTPRoutes()
			if test.expectedError == "" {
				assert.Nil(t, err)
			} else {
				assert.NotNil(t, err)
				assert.Equal(t, test.expectedError, err.Error())
			}
			assert.Equal(t, test.expectedResources, test.inputResources)
			assert.Equal(t, test.stateChanged, changed)

			faketestclients.CleanupFakeClientReactions(c.api.Gatewayclientset)
			faketestclients.CleanupFakeClientReactions(c.api.Kubeclientset.CoreV1())
		})
	}
}
