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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

var ServicesListEmpty = corev1.ServiceList{Items: []corev1.Service{}}
var ServicesListRgwExternal = corev1.ServiceList{Items: []corev1.Service{RgwExternalService}}

var RgwExternalServiceGenerated = corev1.Service{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "rook-ceph-rgw-rgw-store-external",
		Namespace: "rook-ceph",
		Labels: map[string]string{
			"app":               "rook-ceph-rgw",
			"rook_object_store": "rgw-store",
			"external_access":   "rgw",
		},
	},
	Spec: corev1.ServiceSpec{
		Ports: []corev1.ServicePort{
			{
				Name:     "http",
				Port:     80,
				Protocol: "TCP",
				TargetPort: intstr.IntOrString{
					Type:   intstr.Int,
					IntVal: 80,
				},
			},
			{
				Name:     "https",
				Port:     443,
				Protocol: "TCP",
				TargetPort: intstr.IntOrString{
					Type:   intstr.Int,
					IntVal: 8443,
				},
			},
		},
		Type:            "LoadBalancer",
		SessionAffinity: "None",
		Selector: map[string]string{
			"app":               "rook-ceph-rgw",
			"rook_cluster":      "rook-ceph",
			"rook_object_store": "rgw-store",
		},
	},
}

var RgwExternalService = func() corev1.Service {
	svc := RgwExternalServiceGenerated.DeepCopy()
	svc.Status = corev1.ServiceStatus{
		LoadBalancer: corev1.LoadBalancerStatus{
			Ingress: []corev1.LoadBalancerIngress{
				{
					IP: "192.168.100.150",
				},
			},
		},
	}
	return *svc
}()
