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
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/pkg/errors"
	cephv1 "github.com/rook/rook/pkg/apis/ceph.rook.io/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	cephlcmv1alpha1 "github.com/Mirantis/pelagia/pkg/apis/ceph.pelagia.lcm/v1alpha1"
)

func (c *cephDeploymentConfig) isMigrationRequired() bool {
	return c.deprecatedClusterParams() || len(c.cdConfig.cephDpl.Spec.Pools) > 0 ||
		(c.cdConfig.cephDpl.Spec.SharedFilesystem != nil && len(c.cdConfig.cephDpl.Spec.SharedFilesystem.OldCephFS) > 0) ||
		(c.cdConfig.cephDpl.Spec.ObjectStorage != nil && c.cdConfig.cephDpl.Spec.ObjectStorage.OldMultiSite != nil)
}

func (c *cephDeploymentConfig) deprecatedClusterParams() bool {
	required := c.cdConfig.cephDpl.Spec.DashboardEnabled != nil || c.cdConfig.cephDpl.Spec.DataDirHostPath != "" ||
		c.cdConfig.cephDpl.Spec.Network != nil || c.cdConfig.cephDpl.Spec.External != nil ||
		c.cdConfig.cephDpl.Spec.Mgr != nil || c.cdConfig.cephDpl.Spec.HealthCheck != nil

	// check that provided hyperconverge is really related to cluster params
	if c.cdConfig.cephDpl.Spec.HyperConverge != nil {
		if len(c.cdConfig.cephDpl.Spec.HyperConverge.Resources) > 0 {
			extraSvc := 0
			if _, ok := c.cdConfig.cephDpl.Spec.HyperConverge.Resources["rgw"]; ok {
				extraSvc++
			}
			if _, ok := c.cdConfig.cephDpl.Spec.HyperConverge.Resources["mds"]; ok {
				extraSvc++
			}
			required = required || len(c.cdConfig.cephDpl.Spec.HyperConverge.Resources) > extraSvc
		}

		if len(c.cdConfig.cephDpl.Spec.HyperConverge.Tolerations) > 0 {
			extraSvc := 0
			if _, ok := c.cdConfig.cephDpl.Spec.HyperConverge.Tolerations["rgw"]; ok {
				extraSvc++
			}
			if _, ok := c.cdConfig.cephDpl.Spec.HyperConverge.Tolerations["mds"]; ok {
				extraSvc++
			}
			required = required || len(c.cdConfig.cephDpl.Spec.HyperConverge.Tolerations) > extraSvc
		}
	}
	return required
}

var (
	msgTmpl    = "found deprecated field spec.%s, moving to spec.%s"
	errMsgTmpl = "found deprecated field spec.%s, but conflicts with spec.%s. Keep correct and remove not needed fields manually"
)

func (c *cephDeploymentConfig) ensureDeprecatedFields(skip bool) (bool, error) {
	// check do we need migration at all, before proceed to avoid not needed casts
	// TODO: force skip for now from controller to avoid huge diff
	if skip || !c.isMigrationRequired() {
		return false, nil
	}

	extraPlacement := cephv1.PlacementSpec{}
	extraResources := cephv1.ResourceSpec{}
	// since currently all specified under one section, but for cluster and extra svc need to be separated
	if c.cdConfig.cephDpl.Spec.HyperConverge != nil {
		if len(c.cdConfig.cephDpl.Spec.HyperConverge.Resources) > 0 {
			if rgw, ok := c.cdConfig.cephDpl.Spec.HyperConverge.Resources["rgw"]; ok {
				if c.cdConfig.cephDpl.Spec.ObjectStorage != nil {
					extraResources["rgw"] = rgw
				} else {
					c.log.Warn().Msg("found deprecated field spec.hyperconverge.resources['rgw'], but no spec.objectStorage present, will be removed")
				}
				delete(c.cdConfig.cephDpl.Spec.HyperConverge.Resources, "rgw")
			}
			if mds, ok := c.cdConfig.cephDpl.Spec.HyperConverge.Resources["mds"]; ok {
				if c.cdConfig.cephDpl.Spec.SharedFilesystem != nil && len(c.cdConfig.cephDpl.Spec.SharedFilesystem.OldCephFS) > 0 {
					extraResources["mds"] = mds
				} else {
					c.log.Warn().Msg("found deprecated field spec.hyperconverge.resources['mds'], but no spec.sharedFilesystem.cephFS present, will be removed")
				}
				delete(c.cdConfig.cephDpl.Spec.HyperConverge.Resources, "mds")
			}
		}
		if len(c.cdConfig.cephDpl.Spec.HyperConverge.Tolerations) > 0 {
			if rgw, ok := c.cdConfig.cephDpl.Spec.HyperConverge.Tolerations["rgw"]; ok && len(rgw.Rules) > 0 {
				if c.cdConfig.cephDpl.Spec.ObjectStorage != nil {
					extraPlacement["rgw"] = cephv1.Placement{
						Tolerations: rgw.Rules,
					}
				} else {
					c.log.Warn().Msg("found deprecated field spec.hyperconverge.tolerations['rgw'], but no spec.objectStorage present, will be removed")
				}
				delete(c.cdConfig.cephDpl.Spec.HyperConverge.Tolerations, "rgw")
			}
			if mds, ok := c.cdConfig.cephDpl.Spec.HyperConverge.Tolerations["mds"]; ok && len(mds.Rules) > 0 {
				if c.cdConfig.cephDpl.Spec.SharedFilesystem != nil && len(c.cdConfig.cephDpl.Spec.SharedFilesystem.OldCephFS) > 0 {
					extraPlacement["mds"] = cephv1.Placement{
						Tolerations: mds.Rules,
					}
				} else {
					c.log.Warn().Msg("found deprecated field spec.hyperconverge.tolerations['mds'], but no spec.sharedFilesystem.cephFS present, will be removed")
				}
				delete(c.cdConfig.cephDpl.Spec.HyperConverge.Tolerations, "mds")
			}
		}
	}

	// Working with fields as map[string]interface to avoid putting
	// in spec extra default fields from Rook structures, which has no
	// pointers and keep our spec is small as possible.
	//
	paramsCantMigrate := []string{}
	clusterData, clusterParamsCantMigrate, err := c.convertClusterRelatedParams()
	if err != nil {
		return false, errors.Wrapf(err, "failed check deprecated cluster params")
	}
	paramsCantMigrate = append(paramsCantMigrate, clusterParamsCantMigrate...)

	if len(c.cdConfig.cephDpl.Spec.Pools) > 0 {
		if c.cdConfig.cephDpl.Spec.BlockStorage != nil && len(c.cdConfig.cephDpl.Spec.BlockStorage.Pools) > 0 {
			c.log.Error().Msgf(errMsgTmpl, "pools", "blockStorage.pools")
			paramsCantMigrate = append(paramsCantMigrate, "spec.pools")
		} else {
			if c.cdConfig.cephDpl.Spec.BlockStorage == nil {
				c.cdConfig.cephDpl.Spec.BlockStorage = &cephlcmv1alpha1.CephBlockStorage{}
			}
			c.log.Warn().Msgf(msgTmpl, "pools", "blockStorage.pools")
			newPools, err := c.convertPoolsParams()
			if err != nil {
				return false, errors.Wrap(err, "failed to migrate deprecated pools section")
			}
			c.cdConfig.cephDpl.Spec.BlockStorage.Pools = newPools
			c.cdConfig.cephDpl.Spec.Pools = nil
		}
	}

	if c.cdConfig.cephDpl.Spec.SharedFilesystem != nil && len(c.cdConfig.cephDpl.Spec.SharedFilesystem.OldCephFS) > 0 {
		if len(c.cdConfig.cephDpl.Spec.SharedFilesystem.Filesystems) > 0 {
			c.log.Error().Msgf(errMsgTmpl, "sharedFilesystem.cephFS", "sharedFilesystem.cephFilesystems")
			paramsCantMigrate = append(paramsCantMigrate, "spec.sharedFilesystem.cephFS")
		} else {
			c.log.Warn().Msgf(msgTmpl, "sharedFilesystem.cephFS", "sharedFilesystem.cephFilesystems")
			newFs, err := c.convertCephFsParams(extraPlacement, extraResources)
			if err != nil {
				return false, errors.Wrap(err, "failed to migrate deprecated ceph filesystems section")
			}
			c.cdConfig.cephDpl.Spec.SharedFilesystem.Filesystems = newFs
			c.cdConfig.cephDpl.Spec.SharedFilesystem.OldCephFS = nil
		}
	}

	if c.cdConfig.cephDpl.Spec.ObjectStorage != nil {
		if c.cdConfig.cephDpl.Spec.ObjectStorage.OldMultiSite != nil {
			canMove := true
			if len(c.cdConfig.cephDpl.Spec.ObjectStorage.OldMultiSite.Realms) > 0 && len(c.cdConfig.cephDpl.Spec.ObjectStorage.Realms) > 0 {
				c.log.Error().Msgf(errMsgTmpl, "objectStorage.multiSite.realms", "objectStorage.realms")
				paramsCantMigrate = append(paramsCantMigrate, "spec.objectStorage.multiSite.realms")
				canMove = false
			}
			if len(c.cdConfig.cephDpl.Spec.ObjectStorage.OldMultiSite.ZoneGroups) > 0 && len(c.cdConfig.cephDpl.Spec.ObjectStorage.Zonegroups) > 0 {
				c.log.Error().Msgf(errMsgTmpl, "objectStorage.multiSite.zoneGroups", "objectStorage.zoneGroups")
				paramsCantMigrate = append(paramsCantMigrate, "spec.objectStorage.multiSite.zoneGroups")
				canMove = false
			}
			if len(c.cdConfig.cephDpl.Spec.ObjectStorage.OldMultiSite.Zones) > 0 && len(c.cdConfig.cephDpl.Spec.ObjectStorage.Zones) > 0 {
				c.log.Error().Msgf(errMsgTmpl, "objectStorage.multiSite.zones", "objectStorage.zones")
				paramsCantMigrate = append(paramsCantMigrate, "spec.objectStorage.multiSite.zones")
				canMove = false
			}
			if canMove {
				c.log.Warn().Msgf(msgTmpl, "objectStorage.multiSite.realms", "objectStorage.realms")
				c.log.Warn().Msgf(msgTmpl, "objectStorage.multiSite.zoneGroups", "objectStorage.zoneGroups")
				c.log.Warn().Msgf(msgTmpl, "objectStorage.multiSite.zones", "objectStorage.zones")
				realms, zonegroups, zones, err := c.convertMultisiteParams()
				if err != nil {
					return false, errors.Wrap(err, "failed to migrate deprecated objectstore multisite section")
				}
				c.cdConfig.cephDpl.Spec.ObjectStorage.Realms = realms
				c.cdConfig.cephDpl.Spec.ObjectStorage.Zonegroups = zonegroups
				c.cdConfig.cephDpl.Spec.ObjectStorage.Zones = zones
				c.cdConfig.cephDpl.Spec.ObjectStorage.OldMultiSite = nil
			}
		}
	}

	if len(paramsCantMigrate) > 0 {
		return false, errors.Errorf("found deprecated params which can't be automatically migrated: [ %s ]", strings.Join(paramsCantMigrate, " "))
	}

	if len(clusterData) > 0 {
		newCluster := &cephlcmv1alpha1.CephCluster{}
		err = cephlcmv1alpha1.SetRawSpec(&newCluster.RawExtension, clusterData, &cephv1.ClusterSpec{})
		if err != nil {
			return false, errors.Wrap(err, "failed to migrate deprecated cluster sections")
		}
		c.cdConfig.cephDpl.Spec.Cluster = newCluster
	}

	c.log.Info().Msgf("removing deprecated params from CephDeployment %s/%s spec", c.cdConfig.cephDpl.Namespace, c.cdConfig.cephDpl.Name)
	_, err = c.api.CephLcmclientset.LcmV1alpha1().CephDeployments(c.cdConfig.cephDpl.Namespace).Update(c.context, c.cdConfig.cephDpl, metav1.UpdateOptions{})
	if err != nil {
		return false, errors.Wrapf(err, "failed to update CephDeployment spec")
	}
	return true, nil
}

func (c *cephDeploymentConfig) convertClusterRelatedParams() ([]byte, []string, error) {
	clusterParams := map[string]interface{}{}
	paramsCantMigrate := []string{}

	// in case if not a cluster section passed as deprecated, no need to read it
	if !c.deprecatedClusterParams() {
		return nil, nil, nil
	}

	if c.cdConfig.cephDpl.Spec.Cluster != nil {
		err := cephlcmv1alpha1.DecodeRawToStruct(c.cdConfig.cephDpl.Spec.Cluster.Raw, &clusterParams)
		if err != nil {
			return nil, nil, errors.Wrap(err, "failed decode cluster field in spec")
		}
	}

	if c.cdConfig.cephDpl.Spec.DashboardEnabled != nil {
		if _, ok := clusterParams["dashboard"]; ok {
			c.log.Error().Msgf(errMsgTmpl, "dashboard", "cluster.dashboard")
			paramsCantMigrate = append(paramsCantMigrate, "spec.dashboard")
		} else {
			if *c.cdConfig.cephDpl.Spec.DashboardEnabled {
				c.log.Warn().Msgf(msgTmpl, "dashboard", "cluster.dashboard")
				clusterParams["dashboard"] = map[string]interface{}{
					"enabled": true,
				}
			}
			c.cdConfig.cephDpl.Spec.DashboardEnabled = nil
		}
	}

	if c.cdConfig.cephDpl.Spec.DataDirHostPath != "" {
		if _, ok := clusterParams["dataDirHostPath"]; ok {
			c.log.Error().Msgf(errMsgTmpl, "dataDirHostPath", "cluster.dataDirHostPath")
			paramsCantMigrate = append(paramsCantMigrate, "spec.dataDirHostPath")
		} else {
			c.log.Warn().Msgf(msgTmpl, "dataDirHostPath", "cluster.dataDirHostPath")
			clusterParams["dataDirHostPath"] = c.cdConfig.cephDpl.Spec.DataDirHostPath
			c.cdConfig.cephDpl.Spec.DataDirHostPath = ""
		}
	}

	external := false
	if c.cdConfig.cephDpl.Spec.External != nil {
		if _, ok := clusterParams["external"]; ok {
			c.log.Error().Msgf(errMsgTmpl, "external", "cluster.external")
			paramsCantMigrate = append(paramsCantMigrate, "spec.external")
		} else {
			c.log.Warn().Msgf(msgTmpl, "external", "cluster.external")
			if *c.cdConfig.cephDpl.Spec.External {
				clusterParams["external"] = map[string]interface{}{
					"enable": true,
				}
				external = true
			}
			c.cdConfig.cephDpl.Spec.External = nil
		}
	}

	if c.cdConfig.cephDpl.Spec.Network != nil {
		if _, ok := clusterParams["network"]; ok {
			c.log.Error().Msgf(errMsgTmpl, "network", "cluster.network")
			paramsCantMigrate = append(paramsCantMigrate, "spec.network")
		} else {
			if external {
				c.log.Warn().Msg("found deprecated field spec.network, which is not required for external setup, will be removed")
			} else {
				c.log.Warn().Msgf(msgTmpl, "network", "cluster.network")
				network := map[string]interface{}{
					"addressRanges": map[string]interface{}{
						"public":  strings.Split(c.cdConfig.cephDpl.Spec.Network.PublicNet, ","),
						"cluster": strings.Split(c.cdConfig.cephDpl.Spec.Network.ClusterNet, ","),
					},
				}
				if c.cdConfig.cephDpl.Spec.Network.Provider != "" {
					network["provider"] = c.cdConfig.cephDpl.Spec.Network.Provider
				}
				if c.cdConfig.cephDpl.Spec.Network.Selector != nil {
					network["selectors"] = c.cdConfig.cephDpl.Spec.Network.Selector
				}
				clusterParams["network"] = network
			}
			c.cdConfig.cephDpl.Spec.Network = nil
		}
	}

	if c.cdConfig.cephDpl.Spec.Mgr != nil {
		if _, ok := clusterParams["mgr"]; ok {
			c.log.Error().Msgf(errMsgTmpl, "mgr", "cluster.mgr")
			paramsCantMigrate = append(paramsCantMigrate, "spec.mgr")
		} else {
			c.log.Warn().Msgf(msgTmpl, "mgr", "cluster.mgr")
			clusterParams["mgr"] = map[string]interface{}{
				"modules": c.cdConfig.cephDpl.Spec.Mgr.MgrModules,
			}
			c.cdConfig.cephDpl.Spec.Mgr = nil
		}
	}

	if c.cdConfig.cephDpl.Spec.HealthCheck != nil {
		if _, ok := clusterParams["healthCheck"]; ok {
			c.log.Error().Msgf(errMsgTmpl, "healthCheck", "cluster.healthCheck")
			paramsCantMigrate = append(paramsCantMigrate, "spec.healthCheck")
		} else {
			c.log.Warn().Msgf(msgTmpl, "healthCheck", "cluster.healthCheck")
			clusterParams["healthCheck"] = c.cdConfig.cephDpl.Spec.HealthCheck
			c.cdConfig.cephDpl.Spec.HealthCheck = nil
		}
	}

	if c.cdConfig.cephDpl.Spec.HyperConverge != nil {
		if len(c.cdConfig.cephDpl.Spec.HyperConverge.Resources) > 0 {
			if _, ok := clusterParams["resources"]; ok {
				c.log.Error().Msgf(errMsgTmpl, "hyperconverge.resources", "cluster.resources")
				paramsCantMigrate = append(paramsCantMigrate, "spec.hyperconverge.resources")
			} else {
				c.log.Warn().Msgf(msgTmpl, "hyperconverge.resources", "cluster.resources")
				clusterParams["resources"] = c.cdConfig.cephDpl.Spec.HyperConverge.Resources
			}
		}

		if len(c.cdConfig.cephDpl.Spec.HyperConverge.Tolerations) > 0 {
			for key, tol := range c.cdConfig.cephDpl.Spec.HyperConverge.Tolerations {
				oldSection := fmt.Sprintf("hyperconverge.tolerations[%s]", key)
				newSection := fmt.Sprintf("cluster.placement[%s].tolerations", key)
				if p, pok := clusterParams["placement"]; pok {
					placement := p.(map[string]interface{})
					if d, dok := placement[key]; dok {
						daemon := d.(map[string]interface{})
						if _, tok := daemon["tolerations"]; tok {
							c.log.Error().Msgf(errMsgTmpl, oldSection, newSection)
							paramsCantMigrate = append(paramsCantMigrate, "spec."+oldSection)
							continue
						}
						daemon["tolerations"] = tol.Rules
						placement[key] = daemon
						clusterParams["placement"] = placement
					} else {
						placement[key] = map[string]interface{}{
							"tolerations": tol.Rules,
						}
						clusterParams["placement"] = placement
					}
				} else {
					clusterParams["placement"] = map[string]interface{}{
						key: map[string]interface{}{
							"tolerations": tol.Rules,
						},
					}
				}
				c.log.Warn().Msgf(msgTmpl, oldSection, newSection)
			}
		}
		c.cdConfig.cephDpl.Spec.HyperConverge = nil
	}

	data, err := json.Marshal(clusterParams)
	if err != nil {
		return nil, nil, err
	}
	sort.Strings(paramsCantMigrate)

	return data, paramsCantMigrate, nil
}

func (c *cephDeploymentConfig) convertPoolsParams() ([]cephlcmv1alpha1.CephPool, error) {
	newPools := make([]cephlcmv1alpha1.CephPool, len(c.cdConfig.cephDpl.Spec.Pools))
	for idx, oldPool := range c.cdConfig.cephDpl.Spec.Pools {
		newPool := cephlcmv1alpha1.CephPool{
			Name:             oldPool.Name,
			UseAsFullName:    oldPool.UseAsFullName,
			Role:             oldPool.Role,
			PreserveOnDelete: oldPool.PreserveOnDelete,
			StorageClassOpts: oldPool.StorageClassOpts,
		}
		poolData, err := json.Marshal(oldPool.CephPoolSpec)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to convert to JSON pool %s", oldPool.Name)
		}
		err = cephlcmv1alpha1.SetRawSpec(&newPool.PoolSpec, []byte(poolData), &cephv1.PoolSpec{})
		if err != nil {
			return nil, errors.Wrapf(err, "failed to migrate pool %s from deprecated section", oldPool.Name)
		}
		newPools[idx] = newPool
	}
	return newPools, nil
}

func (c *cephDeploymentConfig) convertCephFsParams(hyperConvergePlacement cephv1.PlacementSpec, hyperConvergeResources cephv1.ResourceSpec) ([]cephlcmv1alpha1.CephFilesystem, error) {
	newFs := make([]cephlcmv1alpha1.CephFilesystem, len(c.cdConfig.cephDpl.Spec.SharedFilesystem.OldCephFS))
	for idx, oldCephFs := range c.cdConfig.cephDpl.Spec.SharedFilesystem.OldCephFS {
		metadataServerParams := map[string]interface{}{
			"activeCount":   oldCephFs.MetadataServer.ActiveCount,
			"activeStandby": oldCephFs.MetadataServer.ActiveStandby,
		}
		if tol, ok := hyperConvergePlacement["mds"]; ok {
			metadataServerParams["placement"] = map[string]interface{}{
				"tolerations": tol.Tolerations,
			}
		}
		if res, ok := hyperConvergeResources["mds"]; ok {
			metadataServerParams["resources"] = res
		}
		if oldCephFs.MetadataServer.Resources != nil {
			metadataServerParams["resources"] = *oldCephFs.MetadataServer.Resources
		}
		if oldCephFs.MetadataServer.HealthCheck != nil {
			if oldCephFs.MetadataServer.HealthCheck.LivenessProbe != nil {
				metadataServerParams["livenessProbe"] = oldCephFs.MetadataServer.HealthCheck.LivenessProbe
			}
			if oldCephFs.MetadataServer.HealthCheck.StartupProbe != nil {
				metadataServerParams["startupProbe"] = oldCephFs.MetadataServer.HealthCheck.StartupProbe
			}
		}
		cephFsSpec := map[string]interface{}{
			"preserveFilesystemOnDelete": oldCephFs.PreserveFilesystemOnDelete,
			"metadataPool":               oldCephFs.MetadataPool,
			"dataPools":                  oldCephFs.DataPools,
			"metadataServer":             metadataServerParams,
		}
		fsData, err := json.Marshal(cephFsSpec)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to convert to JSON cephfilesystem %s", oldCephFs.Name)
		}
		cephFilesystem := cephlcmv1alpha1.CephFilesystem{Name: oldCephFs.Name}
		err = cephlcmv1alpha1.SetRawSpec(&cephFilesystem.FsSpec, []byte(fsData), &cephv1.FilesystemSpec{})
		if err != nil {
			return nil, errors.Wrapf(err, "failed to migrate cephfilesystem %s from deprecated section", oldCephFs.Name)
		}
		newFs[idx] = cephFilesystem
	}
	return newFs, nil
}

func (c *cephDeploymentConfig) convertMultisiteParams() ([]cephlcmv1alpha1.CephObjectRealm, []cephlcmv1alpha1.CephObjectZonegroup, []cephlcmv1alpha1.CephObjectZone, error) {
	newRealms := make([]cephlcmv1alpha1.CephObjectRealm, len(c.cdConfig.cephDpl.Spec.ObjectStorage.OldMultiSite.Realms))
	for idx, realm := range c.cdConfig.cephDpl.Spec.ObjectStorage.OldMultiSite.Realms {
		newRealm := cephlcmv1alpha1.CephObjectRealm{Name: realm.Name}
		realmSpec := map[string]interface{}{
			"defaultRealm": realm.DefaultRealm,
		}
		if realm.Pull != nil {
			msg := "found deprecated parameters spec.objectStorage.multiSite[0].pullEndpoint.accessKey and spec.objectStorage.multiSite[0].pullEndpoint.secretKey, which contains user creds, removing from spec"
			c.log.Warn().Msg(msg)
			realmSpec["pull"] = map[string]interface{}{
				"endpoint": realm.Pull.Endpoint,
			}
		}
		realmData, err := json.Marshal(realmSpec)
		if err != nil {
			return nil, nil, nil, errors.Wrapf(err, "failed to convert to JSON realm %s", realm.Name)
		}
		err = cephlcmv1alpha1.SetRawSpec(&newRealm.Spec, realmData, &cephv1.ObjectRealmSpec{})
		if err != nil {
			return nil, nil, nil, errors.Wrap(err, "failed to migrate deprecated realm section")
		}
		newRealms[idx] = newRealm
	}

	newZonegroups := make([]cephlcmv1alpha1.CephObjectZonegroup, len(c.cdConfig.cephDpl.Spec.ObjectStorage.OldMultiSite.ZoneGroups))
	for idx, zonegroup := range c.cdConfig.cephDpl.Spec.ObjectStorage.OldMultiSite.ZoneGroups {
		newZonegroup := cephlcmv1alpha1.CephObjectZonegroup{Name: zonegroup.Name}
		zonegroupData, err := json.Marshal(cephv1.ObjectZoneGroupSpec{Realm: zonegroup.Realm})
		if err != nil {
			return nil, nil, nil, errors.Wrapf(err, "failed to convert to JSON zonegroup %s", zonegroup.Name)
		}
		err = cephlcmv1alpha1.SetRawSpec(&newZonegroup.Spec, zonegroupData, nil)
		if err != nil {
			return nil, nil, nil, errors.Wrap(err, "failed to migrate deprecated zoneGroups section")
		}
		newZonegroups[idx] = newZonegroup
	}

	newZones := make([]cephlcmv1alpha1.CephObjectZone, len(c.cdConfig.cephDpl.Spec.ObjectStorage.OldMultiSite.Zones))
	for idx, zone := range c.cdConfig.cephDpl.Spec.ObjectStorage.OldMultiSite.Zones {
		newZone := cephlcmv1alpha1.CephObjectZone{Name: zone.Name}
		zoneSpec := map[string]interface{}{
			"zoneGroup":    zone.ZoneGroup,
			"metadataPool": zone.MetadataPool,
			"dataPool":     zone.DataPool,
		}
		if len(zone.EndpointsForZone) > 0 {
			zoneSpec["customEndpoints"] = zone.EndpointsForZone
		}
		zoneData, err := json.Marshal(zoneSpec)
		if err != nil {
			return nil, nil, nil, errors.Wrapf(err, "failed to convert to JSON zone %s", newZone.Name)
		}
		err = cephlcmv1alpha1.SetRawSpec(&newZone.Spec, zoneData, &cephv1.ObjectZoneSpec{})
		if err != nil {
			return nil, nil, nil, errors.Wrap(err, "failed to migrate deprecated zone section")
		}
		newZones[idx] = newZone
	}

	return newRealms, newZonegroups, newZones, nil
}
