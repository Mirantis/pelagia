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

package input

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	gatewayapi "sigs.k8s.io/gateway-api/apis/v1"

	lcmcommon "github.com/Mirantis/pelagia/pkg/common"
)

var HTTPRoutesListEmpty = gatewayapi.HTTPRouteList{}

var HTTPRoutesListDefaultMosk = gatewayapi.HTTPRouteList{
	Items: []gatewayapi.HTTPRoute{DefaultMoskHTTPRoute},
}

var HTTPRoutesListDefaultBase = gatewayapi.HTTPRouteList{
	Items: []gatewayapi.HTTPRoute{DefaultBaseHTTPRoute},
}

var HTTPRoutesListDefaultBaseReady = gatewayapi.HTTPRouteList{
	Items: []gatewayapi.HTTPRoute{DefaultBaseHTTPRouteReady},
}

var DefaultMoskHTTPRoute = gatewayapi.HTTPRoute{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "rgw-store-openstack-route",
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
		Hostnames: []gatewayapi.Hostname{gatewayapi.Hostname("rgw-store.openstack.com")},
		CommonRouteSpec: gatewayapi.CommonRouteSpec{
			ParentRefs: []gatewayapi.ParentReference{
				{
					Name:      gatewayapi.ObjectName("app-gateway"),
					Namespace: lcmcommon.PtrTo(gatewayapi.Namespace("openstack")),
					Group:     lcmcommon.PtrTo(gatewayapi.Group(gatewayapi.GroupName)),
					Kind:      lcmcommon.PtrTo(gatewayapi.Kind("Gateway")),
				},
			},
		},
		Rules: []gatewayapi.HTTPRouteRule{
			{
				Name: lcmcommon.PtrTo(gatewayapi.SectionName("default")),
				Matches: []gatewayapi.HTTPRouteMatch{
					{
						Path: &gatewayapi.HTTPPathMatch{
							Type:  lcmcommon.PtrTo(gatewayapi.PathMatchPathPrefix),
							Value: lcmcommon.PtrTo("/"),
						},
					},
				},
				BackendRefs: []gatewayapi.HTTPBackendRef{
					{
						BackendRef: gatewayapi.BackendRef{
							BackendObjectReference: gatewayapi.BackendObjectReference{
								Group: lcmcommon.PtrTo(gatewayapi.Group("")),
								Kind:  lcmcommon.PtrTo(gatewayapi.Kind("Service")),
								Name:  gatewayapi.ObjectName("rook-ceph-rgw-rgw-store"),
								Port:  lcmcommon.PtrTo(int32(80)),
							},
							Weight: lcmcommon.PtrTo(int32(1)),
						},
					},
				},
			},
		},
	},
}

var DefaultBaseHTTPRoute = func() gatewayapi.HTTPRoute {
	route := DefaultMoskHTTPRoute.DeepCopy()
	route.Name = "rgw-route"
	route.Spec.Hostnames[0] = gatewayapi.Hostname("rgw-store.example.com")
	return *route
}()

var DefaultBaseHTTPRouteReady = func() gatewayapi.HTTPRoute {
	route := DefaultBaseHTTPRoute.DeepCopy()
	route.Status = gatewayapi.HTTPRouteStatus{
		RouteStatus: gatewayapi.RouteStatus{
			Parents: []gatewayapi.RouteParentStatus{
				{
					Conditions: []metav1.Condition{
						{
							Reason: "Accepted",
						},
					},
				},
			},
		},
	}
	return *route
}()
