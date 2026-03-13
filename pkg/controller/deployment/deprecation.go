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

	"github.com/pkg/errors"
	cephv1 "github.com/rook/rook/pkg/apis/ceph.rook.io/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	cephlcmv1alpha1 "github.com/Mirantis/pelagia/pkg/apis/ceph.pelagia.lcm/v1alpha1"
)

func (c *cephDeploymentConfig) ensureDeprecatedFields() error {
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
			c.log.Error().Msgf("%s, remove deprecated field or set correct path manually", errMsg)
			return errors.New(errMsg)
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
		c.log.Warn().Msg("found deprecated field spec.network, moving to spec.cluster.network")
		// check we dont have network conflict to avoid daemon crashes
		if c.cdConfig.cephDpl.Spec.Cluster.Network.AddressRanges != nil {
			errMsg := "networks from deprecated field spec.network are conflicting with spec.cluster.network.addressRanges"
			c.log.Error().Msgf("%s, remove deprecated field or set correct networks manually", errMsg)
			return errors.New(errMsg)
		}

		c.cdConfig.cephDpl.Spec.Cluster.Network = cephv1.NetworkSpec{
			Provider:  cephv1.NetworkProviderType(c.cdConfig.cephDpl.Spec.Network.Provider),
			Selectors: c.cdConfig.cephDpl.Spec.Network.Selector,
			AddressRanges: &cephv1.AddressRangesSpec{
				Public:  []cephv1.CIDR{cephv1.CIDR(c.cdConfig.cephDpl.Spec.Network.PublicNet)},
				Cluster: []cephv1.CIDR{cephv1.CIDR(c.cdConfig.cephDpl.Spec.Network.ClusterNet)},
			},
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

	if len(c.cdConfig.cephDpl.Spec.Pools) > 0 {
		c.log.Warn().Msg("found deprecated field spec.pools, moving to spec.blockStorage.pools")
		// let user figure out which pools are incorrect or copy manually if both specified
		if c.cdConfig.cephDpl.Spec.BlockStorage != nil && len(c.cdConfig.cephDpl.Spec.BlockStorage.Pools) > 0 {
			errMsg := "pools from deprecated field spec.pools are conflicting with spec.blockStorage.pools"
			c.log.Error().Msgf("%s, remove deprecated field or set correct pools manually", errMsg)
			return errors.New(errMsg)
		}

		if c.cdConfig.cephDpl.Spec.BlockStorage == nil {
			c.cdConfig.cephDpl.Spec.BlockStorage = &cephlcmv1alpha1.CephBlockStorage{}
		}
		newPools := make([]cephlcmv1alpha1.CephPool, len(c.cdConfig.cephDpl.Spec.Pools))
		for idx, pool := range c.cdConfig.cephDpl.Spec.Pools {
			newPools[idx] = cephlcmv1alpha1.CephPool{
				Name:             pool.Name,
				UseAsFullName:    pool.UseAsFullName,
				Role:             pool.Role,
				PreserveOnDelete: pool.PreserveOnDelete,
				StorageClassOpts: pool.StorageClassOpts,
				PoolSpec: cephv1.PoolSpec{
					FailureDomain:      pool.FailureDomain,
					CrushRoot:          pool.CrushRoot,
					DeviceClass:        pool.DeviceClass,
					Parameters:         pool.Parameters,
					EnableCrushUpdates: pool.EnableCrushUpdates,
				},
			}
			if pool.Mirroring != nil {
				newPools[idx].Mirroring = cephv1.MirroringSpec{
					Enabled: true,
					Mode:    pool.Mirroring.Mode,
				}
			}
			if pool.Replicated != nil {
				newPools[idx].Replicated = cephv1.ReplicatedSpec{
					Size: pool.Replicated.Size,
				}
				if pool.Replicated.TargetSizeRatio != 0 {
					newPools[idx].Replicated.TargetSizeRatio = pool.Replicated.TargetSizeRatio
				}
			}
			if pool.ErasureCoded != nil {
				newPools[idx].ErasureCoded = cephv1.ErasureCodedSpec{
					CodingChunks: pool.ErasureCoded.CodingChunks,
					DataChunks:   pool.ErasureCoded.DataChunks,
					Algorithm:    pool.ErasureCoded.Algorithm,
				}
			}
		}
		c.cdConfig.cephDpl.Spec.BlockStorage.Pools = newPools
		c.cdConfig.cephDpl.Spec.Pools = nil
		updateRequired = true
	}

	if c.cdConfig.cephDpl.Spec.SharedFilesystem != nil && len(c.cdConfig.cephDpl.Spec.SharedFilesystem.CephFS) > 0 {
		// let user figure out which cephfs are incorrect or copy manually if both specified
		if len(c.cdConfig.cephDpl.Spec.SharedFilesystem.CephFs) > 0 {
			errMsg := "cephfs from deprecated field spec.sharedFilesystem.cephFS are conflicting with spec.sharedFilesystem.cephFilesystems"
			c.log.Error().Msgf("%s, remove deprecated field or set correct cephfs manually", errMsg)
			return errors.New(errMsg)
		}

		c.cdConfig.cephDpl.Spec.SharedFilesystem.CephFs = make([]cephlcmv1alpha1.CephFilesystem, len(c.cdConfig.cephDpl.Spec.SharedFilesystem.CephFS))
		c.log.Warn().Msg("found deprecated field spec.sharedFilesystem.cephFS, moving to spec.sharedFilesystem.cephFilesystems")
		for idx, cephfs := range c.cdConfig.cephDpl.Spec.SharedFilesystem.CephFS {
			c.cdConfig.cephDpl.Spec.SharedFilesystem.CephFs[idx] = cephlcmv1alpha1.CephFilesystem{
				Name: cephfs.Name,
				FilesystemSpec: cephv1.FilesystemSpec{
					PreserveFilesystemOnDelete: cephfs.PreserveFilesystemOnDelete,
					MetadataServer: cephv1.MetadataServerSpec{
						ActiveCount:   cephfs.MetadataServer.ActiveCount,
						ActiveStandby: cephfs.MetadataServer.ActiveStandby,
					},
					MetadataPool: cephv1.NamedPoolSpec{
						PoolSpec: cephv1.PoolSpec{
							FailureDomain:      cephfs.MetadataPool.FailureDomain,
							CrushRoot:          cephfs.MetadataPool.CrushRoot,
							DeviceClass:        cephfs.MetadataPool.DeviceClass,
							Parameters:         cephfs.MetadataPool.Parameters,
							EnableCrushUpdates: cephfs.MetadataPool.EnableCrushUpdates,
							Replicated: cephv1.ReplicatedSpec{
								Size: cephfs.MetadataPool.Replicated.Size,
							},
						},
					},
				},
			}
			newDataPools := make([]cephv1.NamedPoolSpec, len(cephfs.DataPools))
			for jdx, dataPool := range cephfs.DataPools {
				newDataPools[jdx] = cephv1.NamedPoolSpec{
					Name: dataPool.Name,
					PoolSpec: cephv1.PoolSpec{
						FailureDomain:      dataPool.FailureDomain,
						CrushRoot:          dataPool.CrushRoot,
						DeviceClass:        dataPool.DeviceClass,
						Parameters:         dataPool.Parameters,
						EnableCrushUpdates: dataPool.EnableCrushUpdates,
					},
				}
				if dataPool.Replicated != nil {
					newDataPools[idx].Replicated = cephv1.ReplicatedSpec{
						Size: dataPool.Replicated.Size,
					}
				}
				if dataPool.ErasureCoded != nil {
					newDataPools[idx].ErasureCoded = cephv1.ErasureCodedSpec{
						CodingChunks: dataPool.ErasureCoded.CodingChunks,
						DataChunks:   dataPool.ErasureCoded.DataChunks,
						Algorithm:    dataPool.ErasureCoded.Algorithm,
					}
				}
			}
			c.cdConfig.cephDpl.Spec.SharedFilesystem.CephFs[idx].DataPools = newDataPools
			if c.cdConfig.cephDpl.Spec.HyperConverge != nil {
				if c.cdConfig.cephDpl.Spec.HyperConverge.Resources != nil {
					if res, ok := c.cdConfig.cephDpl.Spec.HyperConverge.Resources["mds"]; ok {
						c.cdConfig.cephDpl.Spec.SharedFilesystem.CephFs[idx].MetadataServer.Resources = res
						delete(c.cdConfig.cephDpl.Spec.HyperConverge.Resources, "mds")
					}
				}
				if c.cdConfig.cephDpl.Spec.HyperConverge.Tolerations != nil {
					if tol, ok := c.cdConfig.cephDpl.Spec.HyperConverge.Tolerations["mds"]; ok {
						c.cdConfig.cephDpl.Spec.SharedFilesystem.CephFs[idx].MetadataServer.Placement.Tolerations = tol.Rules
						delete(c.cdConfig.cephDpl.Spec.HyperConverge.Tolerations, "mds")
					}
				}
			}
			if cephfs.MetadataServer.Resources != nil {
				c.cdConfig.cephDpl.Spec.SharedFilesystem.CephFs[idx].MetadataServer.Resources = *cephfs.MetadataServer.Resources
			}
			if cephfs.MetadataServer.HealthCheck != nil {
				c.cdConfig.cephDpl.Spec.SharedFilesystem.CephFs[idx].MetadataServer.LivenessProbe = cephfs.MetadataServer.HealthCheck.LivenessProbe
				c.cdConfig.cephDpl.Spec.SharedFilesystem.CephFs[idx].MetadataServer.StartupProbe = cephfs.MetadataServer.HealthCheck.StartupProbe
			}
		}
		c.cdConfig.cephDpl.Spec.SharedFilesystem.CephFS = nil
		updateRequired = true
	}

	// TODO: move hyperconverge
	if updateRequired {
		c.log.Info().Msgf("rework spec fields for CephDeployment %s/%s", c.cdConfig.cephDpl.Namespace, c.cdConfig.cephDpl.Name)
		_, err := c.api.CephLcmclientset.LcmV1alpha1().CephDeployments(c.cdConfig.cephDpl.Namespace).Update(c.context, c.cdConfig.cephDpl, metav1.UpdateOptions{})
		if err != nil {
			return errors.Wrapf(err, "failed to update CephDeployment spec")
		}
	}
	return nil
}
