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
	"fmt"
	"reflect"
	"sort"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	lcmcommon "github.com/Mirantis/pelagia/pkg/common"
)

func (c *cephDeploymentConfig) ensureNetworkPolicy() (bool, error) {
	if c.lcmConfig.DeployParams.NetPolEnabled {
		c.log.Debug().Msg("ensure required network policies")
		portsMap := c.getPortsForPolicies()
		// sort keys, to have strong order
		keys := []string{}
		for app := range portsMap {
			keys = append(keys, app)
		}
		sort.Strings(keys)
		policyErr := 0
		updated := false
		for _, app := range keys {
			var changed bool
			var err error
			policyName := fmt.Sprintf("%s-policy", app)
			// create/update policy with ports or remove
			if len(portsMap[app]) > 0 {
				changed, err = c.manageNetworkPolicy(generateNetworkPolicy(policyName, c.lcmConfig.RookNamespace, app, portsMap[app]))
			} else {
				changed, err = c.deleteNetworkPolicy(policyName)
			}
			updated = updated || changed
			if err != nil {
				c.log.Error().Err(err).Msgf("failed to manage '%s/%s' networkpolicy", c.lcmConfig.RookNamespace, policyName)
				policyErr++
			}
		}
		if policyErr > 0 {
			return false, errors.New("failed to manage network policies")
		}
		return updated, nil
	}
	c.log.Debug().Msg("ensure network policies are not present")
	removed, err := c.cleanupNetworkPolicy()
	if err != nil {
		return false, errors.New("failed to clean up network policies")
	}
	return !removed, nil
}

func (c *cephDeploymentConfig) getPortsForPolicies() map[string][]networkingv1.NetworkPolicyPort {
	protocol := corev1.ProtocolTCP
	getPort := func(port int32) *intstr.IntOrString {
		v := intstr.FromInt32(port)
		return &v
	}
	getEndPort := func(port int32) *int32 { return &port }

	portsMap := map[string][]networkingv1.NetworkPolicyPort{
		"rook-ceph-mgr": {
			{
				Port:     getPort(int32(9283)),
				Protocol: &protocol,
			},
			{
				Port:     getPort(int32(6800)),
				Protocol: &protocol,
				EndPort:  getEndPort(int32(7300)),
			},
		},
		"rook-ceph-mon": {
			{
				Port:     getPort(int32(3300)),
				Protocol: &protocol,
			},
			{
				Port:     getPort(int32(6789)),
				Protocol: &protocol,
			},
		},
		"rook-ceph-osd": {
			{
				Port:     getPort(int32(6800)),
				Protocol: &protocol,
				EndPort:  getEndPort(int32(7300)),
			},
		},
	}
	if c.cdConfig.cephDpl.Spec.ObjectStorage != nil {
		portsMap["rook-ceph-rgw"] = []networkingv1.NetworkPolicyPort{
			{
				Port:     getPort(c.cdConfig.cephDpl.Spec.ObjectStorage.Rgw.Gateway.Port),
				Protocol: &protocol,
			},
			{
				Port:     getPort(c.cdConfig.cephDpl.Spec.ObjectStorage.Rgw.Gateway.SecurePort),
				Protocol: &protocol,
			},
		}
	} else {
		portsMap["rook-ceph-rgw"] = nil
	}
	if c.cdConfig.cephDpl.Spec.SharedFilesystem != nil && len(c.cdConfig.cephDpl.Spec.SharedFilesystem.CephFS) > 0 {
		portsMap["rook-ceph-mds"] = []networkingv1.NetworkPolicyPort{
			{
				Port:     getPort(int32(6800)),
				Protocol: &protocol,
				EndPort:  getEndPort(int32(7300)),
			},
		}
	} else {
		portsMap["rook-ceph-mds"] = nil
	}
	return portsMap
}

func generateNetworkPolicy(name, namespace, appName string, ports []networkingv1.NetworkPolicyPort) networkingv1.NetworkPolicy {
	return networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    map[string]string{rookNetworkPolicyLabel: "managed"},
		},
		Spec: networkingv1.NetworkPolicySpec{
			Ingress:     []networkingv1.NetworkPolicyIngressRule{{Ports: ports}},
			PodSelector: metav1.LabelSelector{MatchLabels: map[string]string{"app": appName}},
			PolicyTypes: []networkingv1.PolicyType{"Ingress"},
		},
	}
}

func (c *cephDeploymentConfig) cleanupNetworkPolicy() (bool, error) {
	listOptions := metav1.ListOptions{LabelSelector: fmt.Sprintf("%s=managed", rookNetworkPolicyLabel)}
	policies, err := c.api.Kubeclientset.NetworkingV1().NetworkPolicies(c.lcmConfig.RookNamespace).List(c.context, listOptions)
	if err != nil {
		c.log.Error().Err(err).Msgf("failed to check networkpolicies in '%s' namespace", c.lcmConfig.RookNamespace)
		return false, err
	}
	removed := true
	policyErr := 0
	for _, policy := range policies.Items {
		changed, err := c.deleteNetworkPolicy(policy.Name)
		if err != nil {
			c.log.Error().Err(err).Msgf("failed to remove networkpolicy '%s/%s'", c.lcmConfig.RookNamespace, policy.Name)
			policyErr++
		}
		removed = removed && !changed
	}
	if policyErr > 0 {
		return false, errors.New("networkpolicies remove failed")
	}
	return removed, nil
}

func (c *cephDeploymentConfig) manageNetworkPolicy(policy networkingv1.NetworkPolicy) (bool, error) {
	netPol, netPolErr := c.api.Kubeclientset.NetworkingV1().NetworkPolicies(c.lcmConfig.RookNamespace).Get(c.context, policy.Name, metav1.GetOptions{})
	if netPolErr != nil {
		if !apierrors.IsNotFound(netPolErr) {
			c.log.Error().Err(netPolErr).Msg("")
			return false, errors.Wrapf(netPolErr, "failed to check '%s/%s' networkpolicy", c.lcmConfig.RookNamespace, policy.Name)
		}
		c.log.Info().Msgf("creating network policy %s/%s", c.lcmConfig.RookNamespace, policy.Name)
		_, err := c.api.Kubeclientset.NetworkingV1().NetworkPolicies(c.lcmConfig.RookNamespace).Create(c.context, &policy, metav1.CreateOptions{})
		if err != nil {
			c.log.Error().Err(err).Msg("")
			return false, errors.Wrapf(err, "failed to create '%s/%s' networkpolicy", c.lcmConfig.RookNamespace, policy.Name)
		}
		return true, nil
	}
	if !reflect.DeepEqual(netPol.Spec, policy.Spec) || netPol.Labels[rookNetworkPolicyLabel] != policy.Labels[rookNetworkPolicyLabel] {
		c.log.Info().Msgf("updating network policy %s/%s", c.lcmConfig.RookNamespace, policy.Name)
		if netPol.Labels == nil {
			netPol.Labels = map[string]string{}
		}
		if netPol.Labels[rookNetworkPolicyLabel] != policy.Labels[rookNetworkPolicyLabel] {
			netPol.Labels[rookNetworkPolicyLabel] = policy.Labels[rookNetworkPolicyLabel]
		}
		lcmcommon.ShowObjectDiff(*c.log, netPol.Spec, policy.Spec)
		netPol.Spec = policy.Spec
		_, err := c.api.Kubeclientset.NetworkingV1().NetworkPolicies(c.lcmConfig.RookNamespace).Update(c.context, netPol, metav1.UpdateOptions{})
		if err != nil {
			c.log.Error().Err(err).Msg("")
			return false, errors.Wrapf(err, "failed to update '%s/%s' networkpolicy", c.lcmConfig.RookNamespace, policy.Name)
		}
		return true, nil
	}
	return false, nil
}

func (c *cephDeploymentConfig) deleteNetworkPolicy(policyName string) (bool, error) {
	err := c.api.Kubeclientset.NetworkingV1().NetworkPolicies(c.lcmConfig.RookNamespace).Delete(c.context, policyName, metav1.DeleteOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return false, nil
		}
		c.log.Error().Err(err).Msg("")
		return false, errors.Wrapf(err, "failed to remove rgw network policy %s/%s", c.lcmConfig.RookNamespace, policyName)
	}
	c.log.Info().Msgf("removed network policy %s/%s", c.lcmConfig.RookNamespace, policyName)
	return true, nil
}
