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
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var IngressesListEmpty = networkingv1.IngressList{Items: []networkingv1.Ingress{}}
var IngressesList = networkingv1.IngressList{Items: []networkingv1.Ingress{RgwIngress}}

var IngressPathType = networkingv1.PathTypeImplementationSpecific

var RgwIngress = networkingv1.Ingress{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "rook-ceph-rgw-rgw-store-ingress",
		Namespace: "rook-ceph",
		Labels: map[string]string{
			"ingress-type": "openstack-ingress-nginx-rgw",
			"cephdeployment.lcm.mirantis.com/ingress": "ceph-object-store-ingress",
			"app":               "rook-ceph-rgw",
			"rook_object_store": "rgw-store",
			"external_access":   "rgw",
		},
		Annotations: map[string]string{
			"nginx.ingress.kubernetes.io/proxy-body-size": "0",
			"nginx.ingress.kubernetes.io/rewrite-target":  "/",
			"nginx.ingress.kubernetes.io/upstream-vhost":  "rgw-store.example.com",
			"kubernetes.io/ingress.class":                 "openstack-ingress-nginx",
		},
	},
	Spec: networkingv1.IngressSpec{
		Rules: []networkingv1.IngressRule{
			{
				Host: "rgw-store.example.com",
				IngressRuleValue: networkingv1.IngressRuleValue{
					HTTP: &networkingv1.HTTPIngressRuleValue{
						Paths: []networkingv1.HTTPIngressPath{
							{
								Backend: networkingv1.IngressBackend{
									Service: &networkingv1.IngressServiceBackend{
										Name: "rook-ceph-rgw-rgw-store",
										Port: networkingv1.ServiceBackendPort{
											Name: "http",
										},
									},
								},
								Path:     "/",
								PathType: &IngressPathType,
							},
						},
					},
				},
			},
		},
		TLS: []networkingv1.IngressTLS{
			{
				Hosts:      []string{"rgw-store.example.com"},
				SecretName: "rgw-store-ingress-secret",
			},
		},
	},
}

var RgwOpenstackIngress = func(host string) *networkingv1.Ingress {
	ingress := RgwIngress.DeepCopy()
	ingress.Annotations["nginx.ingress.kubernetes.io/upstream-vhost"] = host
	ingress.Spec.Rules[0].Host = host
	ingress.Spec.TLS[0].Hosts = []string{host}
	return ingress
}
