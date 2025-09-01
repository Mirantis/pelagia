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
	"fmt"

	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

var (
	mgrPort       = intstr.FromInt(9283)
	monPort       = intstr.FromInt(3300)
	monPort2      = intstr.FromInt(6789)
	rgwPort       = intstr.FromInt(80)
	rgwSecurePort = intstr.FromInt(8443)

	startPort = intstr.FromInt(6800)
	endPort   = int32(7300)

	netpolProtocol = corev1.ProtocolTCP
)

func GetNetworkPolicy(appName string, ports []networkingv1.NetworkPolicyPort) networkingv1.NetworkPolicy {
	return networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-policy", appName),
			Namespace: "rook-ceph",
			Labels:    map[string]string{"cephdeployment.lcm.mirantis.com/networkpolicy": "managed"},
		},
		Spec: networkingv1.NetworkPolicySpec{
			Ingress:     []networkingv1.NetworkPolicyIngressRule{{Ports: ports}},
			PodSelector: metav1.LabelSelector{MatchLabels: map[string]string{"app": appName}},
			PolicyTypes: []networkingv1.PolicyType{"Ingress"},
		},
	}
}

var NetworkPolicyMgr = GetNetworkPolicy("rook-ceph-mgr", []networkingv1.NetworkPolicyPort{
	{
		Port: &mgrPort, Protocol: &netpolProtocol,
	},
	{
		Port: &startPort, Protocol: &netpolProtocol, EndPort: &endPort,
	},
})

var NetworkPolicyMon = GetNetworkPolicy("rook-ceph-mon", []networkingv1.NetworkPolicyPort{
	{
		Port: &monPort, Protocol: &netpolProtocol,
	},
	{
		Port: &monPort2, Protocol: &netpolProtocol,
	},
})

var NetworkPolicyOsd = GetNetworkPolicy("rook-ceph-osd", []networkingv1.NetworkPolicyPort{
	{
		Port: &startPort, Protocol: &netpolProtocol, EndPort: &endPort,
	},
})

var NetworkPolicyRgw = GetNetworkPolicy("rook-ceph-rgw", []networkingv1.NetworkPolicyPort{
	{
		Port: &rgwPort, Protocol: &netpolProtocol,
	},
	{
		Port: &rgwSecurePort, Protocol: &netpolProtocol,
	},
})

var NetworkPolicyMds = GetNetworkPolicy("rook-ceph-mds", []networkingv1.NetworkPolicyPort{
	{
		Port: &startPort, Protocol: &netpolProtocol, EndPort: &endPort,
	},
})
