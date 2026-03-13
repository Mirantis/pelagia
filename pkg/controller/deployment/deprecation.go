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
	"strings"

	"github.com/pkg/errors"
	cephv1 "github.com/rook/rook/pkg/apis/ceph.rook.io/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	cephlcmv1alpha1 "github.com/Mirantis/pelagia/pkg/apis/ceph.pelagia.lcm/v1alpha1"
)

func (c *cephDeploymentConfig) ensureDeprecatedFields() (bool, error) {
	updateRequired := false
	if c.cdConfig.cephDpl.Spec.Cluster == nil {
		c.log.Warn().Msg("section spec.cluster is not specified, adding to spec")
		c.cdConfig.cephDpl.Spec.Cluster = &cephlcmv1alpha1.CephCluster{
			ClusterSpec: cephv1.ClusterSpec{},
		}
		updateRequired = true
	}

	if c.cdConfig.cephDpl.Spec.DashboardEnabled != nil {
		c.log.Warn().Msg("found deprecated field spec.dashboard, moving to spec.cluster.dashboard.enabled")
		c.cdConfig.cephDpl.Spec.Cluster.Dashboard = cephv1.DashboardSpec{
			Enabled: *c.cdConfig.cephDpl.Spec.DashboardEnabled,
		}
		c.cdConfig.cephDpl.Spec.DashboardEnabled = nil
		updateRequired = true
	}

	if c.cdConfig.cephDpl.Spec.DataDirHostPath != "" {
		c.log.Warn().Msg("found deprecated field spec.dataDirHostPath, moving to spec.cluster.dataDirHostPath")
		// check backward compatibility and user error since field is sensitive
		if c.cdConfig.cephDpl.Spec.Cluster.DataDirHostPath != "" &&
			c.cdConfig.cephDpl.Spec.Cluster.DataDirHostPath != c.cdConfig.cephDpl.Spec.DataDirHostPath {
			errMsg := fmt.Sprintf("value from deprecated field spec.dataDirHostPath=%s is conflicting with spec.cluster.dataDirHostPath=%s",
				c.cdConfig.cephDpl.Spec.DataDirHostPath, c.cdConfig.cephDpl.Spec.Cluster.DataDirHostPath)
			c.log.Error().Msgf("%s, remove deprecated field and set correct path manually", errMsg)
			return false, errors.New(errMsg)
		}

		c.cdConfig.cephDpl.Spec.Cluster.DataDirHostPath = c.cdConfig.cephDpl.Spec.DataDirHostPath
		c.cdConfig.cephDpl.Spec.DataDirHostPath = ""
		updateRequired = true
	}

	if c.cdConfig.cephDpl.Spec.External != nil {
		c.log.Warn().Msg("found deprecated field spec.external, moving to spec.cluster.external.enabled")
		if *c.cdConfig.cephDpl.Spec.External {
			c.cdConfig.cephDpl.Spec.Cluster.External = cephv1.ExternalSpec{Enable: true}
		}
		c.cdConfig.cephDpl.Spec.External = nil
		updateRequired = true
	}

	if c.cdConfig.cephDpl.Spec.Mgr != nil {
		c.log.Warn().Msg("found deprecated field spec.mgr, moving to spec.cluster.mgr")
		if len(c.cdConfig.cephDpl.Spec.Mgr.MgrModules) > 0 {
			cephMgrModules := make([]cephv1.Module, len(c.cdConfig.cephDpl.Spec.Mgr.MgrModules))
			for idx, module := range c.cdConfig.cephDpl.Spec.Mgr.MgrModules {
				cephMgrModules[idx] = cephv1.Module{
					Name:    module.Name,
					Enabled: module.Enabled,
				}
				if module.Settings != nil && module.Settings.BalancerMode != "" {
					cephMgrModules[idx].Settings = cephv1.ModuleSettings{
						BalancerMode: module.Settings.BalancerMode,
					}
				}
			}
			c.cdConfig.cephDpl.Spec.Cluster.Mgr = cephv1.MgrSpec{
				Modules: cephMgrModules,
			}
		}
		c.cdConfig.cephDpl.Spec.Mgr = nil
		updateRequired = true
	}

	if c.cdConfig.cephDpl.Spec.Network != nil {
		if c.cdConfig.cephDpl.Spec.Cluster.External.Enable {
			c.log.Warn().Msg("found deprecated field spec.network, which is not required for external setup anymore")
		} else {
			c.log.Warn().Msg("found deprecated field spec.network, moving to spec.cluster.network")
			// check we dont have network conflict to avoid daemon crashes
			if c.cdConfig.cephDpl.Spec.Cluster.Network.AddressRanges != nil {
				errMsg := "networks from deprecated field spec.network are conflicting with spec.cluster.network.addressRanges"
				c.log.Error().Msgf("%s, remove deprecated field and set correct networks manually", errMsg)
				return false, errors.New(errMsg)
			}

			c.cdConfig.cephDpl.Spec.Cluster.Network = cephv1.NetworkSpec{
				Provider:      cephv1.NetworkProviderType(c.cdConfig.cephDpl.Spec.Network.Provider),
				Selectors:     c.cdConfig.cephDpl.Spec.Network.Selector,
				AddressRanges: &cephv1.AddressRangesSpec{},
			}
			pubRanges := []cephv1.CIDR{}
			for _, pubNet := range strings.Split(c.cdConfig.cephDpl.Spec.Network.PublicNet, ",") {
				pubRanges = append(pubRanges, cephv1.CIDR(pubNet))
			}
			c.cdConfig.cephDpl.Spec.Cluster.Network.AddressRanges.Public = pubRanges
			netRanges := []cephv1.CIDR{}
			for _, clusterNet := range strings.Split(c.cdConfig.cephDpl.Spec.Network.ClusterNet, ",") {
				netRanges = append(netRanges, cephv1.CIDR(clusterNet))
			}
			c.cdConfig.cephDpl.Spec.Cluster.Network.AddressRanges.Cluster = netRanges
		}
		c.cdConfig.cephDpl.Spec.Network = nil
		updateRequired = true
	}

	if c.cdConfig.cephDpl.Spec.HealthCheck != nil {
		c.log.Warn().Msg("found deprecated field spec.healthCheck, moving to spec.cluster.healthCheck")
		c.cdConfig.cephDpl.Spec.Cluster.HealthCheck = cephv1.CephClusterHealthCheckSpec{
			DaemonHealth:  c.cdConfig.cephDpl.Spec.HealthCheck.DaemonHealth,
			LivenessProbe: c.cdConfig.cephDpl.Spec.HealthCheck.LivenessProbe,
			StartupProbe:  c.cdConfig.cephDpl.Spec.HealthCheck.StartupProbe,
		}
		c.cdConfig.cephDpl.Spec.HealthCheck = nil
		updateRequired = true
	}

	// TODO: process Pools section later
	// TODO: process CephFS section later
	// TODO: process ObjectStore section later

	if c.cdConfig.cephDpl.Spec.HyperConverge != nil {
		if len(c.cdConfig.cephDpl.Spec.HyperConverge.Resources) > 0 {
			if len(c.cdConfig.cephDpl.Spec.Cluster.Resources) > 0 {
				errMsg := "cluster resources from deprecated field spec.hyperconverge.resources are conflicting with spec.cluster.resources"
				c.log.Error().Msgf("%s, remove deprecated field and set correct resources manually", errMsg)
				return false, errors.New(errMsg)
			}
			c.cdConfig.cephDpl.Spec.Cluster.Resources = c.cdConfig.cephDpl.Spec.HyperConverge.Resources
		}
		if len(c.cdConfig.cephDpl.Spec.HyperConverge.Tolerations) > 0 {
			if c.cdConfig.cephDpl.Spec.Cluster.Placement == nil {
				c.cdConfig.cephDpl.Spec.Cluster.Placement = cephv1.PlacementSpec{}
			}
			for key, tol := range c.cdConfig.cephDpl.Spec.HyperConverge.Tolerations {
				if placement, ok := c.cdConfig.cephDpl.Spec.Cluster.Placement[cephv1.KeyType(key)]; ok {
					if len(placement.Tolerations) > 0 {
						errMsg := fmt.Sprintf("placement tolerations from deprecated field spec.hyperconverge.tolerations[%s] are conflicting with spec.cluster.placement[%s].tolerations",
							key, key)
						c.log.Error().Msgf("%s, remove deprecated field and set correct tolerations manually", errMsg)
						return false, errors.New(errMsg)
					}
					placement.Tolerations = tol.Rules
					c.cdConfig.cephDpl.Spec.Cluster.Placement[cephv1.KeyType(key)] = placement
				} else {
					c.cdConfig.cephDpl.Spec.Cluster.Placement[cephv1.KeyType(key)] = cephv1.Placement{Tolerations: tol.Rules}
				}
			}
		}
		c.cdConfig.cephDpl.Spec.HyperConverge = nil
		updateRequired = true
	}

	if updateRequired {
		c.log.Info().Msgf("rework spec fields, updating CephDeployment %s/%s", c.cdConfig.cephDpl.Namespace, c.cdConfig.cephDpl.Name)
		_, err := c.api.CephLcmclientset.LcmV1alpha1().CephDeployments(c.cdConfig.cephDpl.Namespace).Update(c.context, c.cdConfig.cephDpl, metav1.UpdateOptions{})
		if err != nil {
			return false, errors.Wrapf(err, "failed to update CephDeployment spec")
		}
	}
	return updateRequired, nil
}
