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
	"sort"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	cephv1 "github.com/rook/rook/pkg/apis/ceph.rook.io/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	cephlcmv1alpha1 "github.com/Mirantis/pelagia/pkg/apis/ceph.pelagia.lcm/v1alpha1"
	lcmcommon "github.com/Mirantis/pelagia/pkg/common"
)

func (c *cephDeploymentConfig) validateSpec() cephlcmv1alpha1.CephDeploymentValidation {
	errMsgs := make([]string, 0)
	if !c.cdConfig.clusterSpec.External.Enable {
		if errs := validateNetworkSpec(c.cdConfig.clusterSpec.Network); len(errs) > 0 {
			c.log.Error().Msgf("failed to validate cluster network spec: %v", errs)
			errMsgs = append(errMsgs, errs...)
		}
		if err := c.validateClusterNodes(); err != nil {
			c.log.Error().Err(err).Msg("failed to validate provided nodes in cluster")
			errMsgs = append(errMsgs, err.Error())
		} else if errs := validateNodesSpec(c.cdConfig.cephDpl, c.cdConfig.nodesListExpanded); len(errs) > 0 {
			c.log.Error().Msgf("failed to validate nodes spec: %v", errs)
			errMsgs = append(errMsgs, errs...)
		}
		// TODO: keep rbdmirror as is, requires total rework
		if err := rbdPeersValidate(c.cdConfig.cephDpl); err != "" {
			c.log.Error().Msgf("failed to validate rbd mirror spec: %s", err)
			errMsgs = append(errMsgs, err)
		}
	}
	if errs := validatePoolsSpec(c.cdConfig.cephDpl, c.cdConfig.clusterSpec.External.Enable, len(c.cdConfig.nodesListExpanded) == 1); len(errs) > 0 {
		c.log.Error().Msgf("failed to validate block storage pools spec: %v", errs)
		errMsgs = append(errMsgs, errs...)
	}
	if errs := validateFilesystemSpec(c.cdConfig.cephDpl, c.cdConfig.nodesListExpanded, c.cdConfig.clusterSpec.External.Enable); len(errs) > 0 {
		c.log.Error().Msgf("failed to validate shared filesystem spec: %v", errs)
		errMsgs = append(errMsgs, errs...)
	}
	if errs := validateObjectStorageSpec(c.cdConfig.cephDpl, c.cdConfig.nodesListExpanded, c.cdConfig.clusterSpec.External.Enable); len(errs) > 0 {
		c.log.Error().Msgf("failed to validate object storage spec: %v", errs)
		errMsgs = append(errMsgs, errs...)
	}

	validationResult := cephlcmv1alpha1.CephDeploymentValidation{
		Result:                  cephlcmv1alpha1.ValidationSucceed,
		LastValidatedGeneration: c.cdConfig.cephDpl.Generation,
	}
	if len(errMsgs) > 0 {
		validationResult.Result = cephlcmv1alpha1.ValidationFailed
		validationResult.Messages = errMsgs
	}
	return validationResult
}

func validateNetworkSpec(clusterNetwork cephv1.NetworkSpec) []string {
	errMsgs := []string{}

	switch clusterNetwork.Provider {
	case "", "host", "multus":
		if clusterNetwork.AddressRanges == nil {
			errMsgs = append(errMsgs, "cluster network addressRanges parameter is not specified")
		} else {
			if len(clusterNetwork.AddressRanges.Public) == 0 {
				errMsgs = append(errMsgs, "cluster network addressRanges public parameter not specified")
			} else {
				for _, net := range clusterNetwork.AddressRanges.Public {
					if string(net) == "" || strings.HasPrefix(string(net), "0.0.0.0") {
						errMsgs = append(errMsgs, "cluster network address ranges public parameter should not be empty or contain range 0.0.0.0")
						break
					}
				}
			}
			if len(clusterNetwork.AddressRanges.Cluster) == 0 {
				errMsgs = append(errMsgs, "cluster network addressRanges cluster parameter not specified")
			} else {
				for _, net := range clusterNetwork.AddressRanges.Cluster {
					if string(net) == "" || strings.HasPrefix(string(net), "0.0.0.0") {
						errMsgs = append(errMsgs, "cluster network address ranges cluster parameter should not be empty or contain range 0.0.0.0")
						break
					}
				}
			}
		}
		if clusterNetwork.Provider == "multus" {
			if clusterNetwork.Selectors[cephv1.CephNetworkPublic] == "" || clusterNetwork.Selectors[cephv1.CephNetworkCluster] == "" {
				errMsgs = append(errMsgs, "cluster network public/cluster selector parameter(s) should not be empty for 'multus' provider")
			}
		}
	default:
		errMsgs = append(errMsgs, "cluster network provider parameter should be empty or equals 'host' or 'multus'")
	}
	return errMsgs
}

func (c *cephDeploymentConfig) validateClusterNodes() error {
	unknownNodes := make([]string, 0)
	allNodes, err := lcmcommon.GetNodeList(c.context, c.api.Kubeclientset, metav1.ListOptions{})
	if err != nil {
		return err
	}
	knownNodes := map[string]bool{}
	for _, node := range allNodes.Items {
		knownNodes[node.Name] = true
	}
	for _, cephDplNode := range c.cdConfig.nodesListExpanded {
		if !knownNodes[cephDplNode.Name] {
			unknownNodes = append(unknownNodes, cephDplNode.Name)
		}
	}
	if len(unknownNodes) > 0 {
		return errors.Errorf("found nodes present in spec, but not exist among k8s cluster nodes: %s", strings.Join(unknownNodes, ","))
	}
	return nil
}

func validateNodesSpec(cephDpl *cephlcmv1alpha1.CephDeployment, nodesListExpanded []cephlcmv1alpha1.CephDeploymentNode) []string {
	errMsgs := []string{}
	validCrushKeys := strings.Join(getCrushKeys(), ", ")
	for _, node := range cephDpl.Spec.Nodes {
		nodeType := "node"
		if node.NodesByLabel != "" || len(node.NodeGroup) > 0 {
			nodeType = "nodeGroup"
		}
		// field are not supported at all in favor of Ceph OSD LCM correct work
		if node.UseAllDevices != nil && *node.UseAllDevices {
			errMsg := fmt.Sprintf("found 'useAllDevices' field for nodes item %s '%s', which is not supported, remove field", nodeType, node.Name)
			errMsgs = append(errMsgs, errMsg)
			continue
		}
		// currently PVC based cluster is not supported
		if len(node.VolumeClaimTemplates) > 0 {
			errMsg := fmt.Sprintf("found 'volumeClaimTemplates' field for nodes item %s '%s', which is not supported, remove field", nodeType, node.Name)
			errMsgs = append(errMsgs, errMsg)
			continue
		}
		// check node crush topology
		for crush := range node.Crush {
			if _, ok := crushTopologyAllowedKeys[crush]; !ok {
				err := fmt.Sprintf("nodes item %s '%s' contains invalid crush topology key '%s'. Valid are: %v", nodeType, node.Name, crush, validCrushKeys)
				errMsgs = append(errMsgs, err)
			}
		}
		// check storage configs
		if lcmcommon.IsCephOsdNode(node.Node) {
			nodeDeviceClass := ""
			if node.Config != nil {
				if node.Config["deviceClass"] != "" {
					nodeDeviceClass = node.Config["deviceClass"]
					if err := validateDeviceClassName(node.Config["deviceClass"], cephDpl.Spec.ExtraOpts); err != nil {
						errMsg := fmt.Sprintf("nodes item %s '%s' config has %s", nodeType, node.Name, err.Error())
						errMsgs = append(errMsgs, errMsg)
						continue
					}
				}
				if node.Config["osdsPerDevice"] != "" {
					_, err := strconv.Atoi(node.Config["osdsPerDevice"])
					if err != nil {
						errMsg := fmt.Sprintf("failed to parse config parameter 'osdsPerDevice' from nodes item %s '%s': %s", nodeType, node.Name, err.Error())
						errMsgs = append(errMsgs, errMsg)
						continue
					}
				}
			}
			if len(node.Devices) > 0 {
				for _, device := range node.Devices {
					deviceClass := nodeDeviceClass
					deviceName := device.Name
					if device.FullPath != "" {
						deviceName = device.FullPath
					} else if device.Name == "" {
						errMsgs = append(errMsgs, fmt.Sprintf("nodes item %s '%s' has device without name or fullpath specified", nodeType, node.Name))
					}
					if device.Config != nil {
						if device.Config["deviceClass"] != "" {
							deviceClass = device.Config["deviceClass"]
							if err := validateDeviceClassName(deviceClass, cephDpl.Spec.ExtraOpts); err != nil {
								errMsg := fmt.Sprintf("device '%s' from nodes item %s '%s' has %s", deviceName, nodeType, node.Name, err.Error())
								errMsgs = append(errMsgs, errMsg)
							}
						}
						if device.Config["osdsPerDevice"] != "" {
							_, err := strconv.Atoi(device.Config["osdsPerDevice"])
							if err != nil {
								errMsg := fmt.Sprintf("failed to parse config parameter 'osdsPerDevice' for device '%s' from %s '%s': %s",
									deviceName, nodeType, node.Name, err.Error())
								errMsgs = append(errMsgs, errMsg)
							}
						}
					}
					if deviceClass == "" {
						errMsg := fmt.Sprintf("config parameter 'deviceClass' is not specified for device '%s' from nodes item %s '%s', but it is required",
							deviceName, nodeType, node.Name)
						errMsgs = append(errMsgs, errMsg)
					}
				}
			} else {
				// check device class is specified for deviceFilter and devicePathFilter
				if nodeDeviceClass == "" {
					errMsg := fmt.Sprintf("config parameter 'deviceClass' is not specified for nodes item %s '%s', but it is required", nodeType, node.Name)
					errMsgs = append(errMsgs, errMsg)
				}
			}
		}
	}
	monCount := 0
	mgrCount := 0
	for _, node := range nodesListExpanded {
		if lcmcommon.Contains(node.Roles, "mon") {
			monCount = monCount + 1
		}
		if lcmcommon.Contains(node.Roles, "mgr") {
			mgrCount = mgrCount + 1
		}
	}
	if monCount == 0 {
		errMsgs = append(errMsgs, "no nodes with 'mon' roles specified")
	} else if len(nodesListExpanded) >= 3 && monCount%2 == 0 {
		// skip check for PRODX-19248
		err := fmt.Sprintf("monitor nodes in spec (with roles 'mon') count is %d, but should be odd for a healthy quorum", monCount)
		errMsgs = append(errMsgs, err)
	}
	if mgrCount == 0 {
		errMsgs = append(errMsgs, "no nodes with 'mgr' roles specified, required at least one")
	}
	return errMsgs
}

func validatePoolsSpec(cephDpl *cephlcmv1alpha1.CephDeployment, externalCluster bool, singleNode bool) []string {
	if cephDpl.Spec.BlockStorage == nil || len(cephDpl.Spec.BlockStorage.Pools) == 0 {
		if externalCluster {
			return nil
		}
		return []string{"no block storage pools provided, required at least one"}
	}

	errMsgs := []string{}
	defaultFound := false
	for _, cephDplPool := range cephDpl.Spec.BlockStorage.Pools {
		if defaultFound && cephDplPool.StorageClassOpts.Default {
			errMsgs = append(errMsgs, "multiple default pools specified")
		}
		defaultFound = defaultFound || cephDplPool.StorageClassOpts.Default

		castedPool, _ := cephDplPool.GetSpec()
		if externalCluster {
			if castedPool.DeviceClass == "" {
				errMsgs = append(errMsgs, fmt.Sprintf("pool '%s' has no device class specified", cephDplPool.Name))
			}
			continue
		}

		if poolErrs := validatePoolSpec(castedPool, false, cephDplPool.Name, singleNode, cephDpl.Spec.ExtraOpts); len(poolErrs) > 0 {
			errMsgs = append(errMsgs, poolErrs...)
		}
		if cephDplPool.StorageClassOpts.ReclaimPolicy != "" && !lcmcommon.Contains(poolReclaimPolicies, cephDplPool.StorageClassOpts.ReclaimPolicy) {
			errMsgs = append(errMsgs, fmt.Sprintf("pool %s contains invalid reclaimPolicy '%s', valid are: %v",
				cephDplPool.Name, cephDplPool.StorageClassOpts.ReclaimPolicy, poolReclaimPolicies))
		}
	}
	if !externalCluster {
		if !defaultFound {
			errMsgs = append(errMsgs, "no default pool specified")
		}
		if errs := openstackPoolsValidate(cephDpl.Spec.BlockStorage.Pools); len(errs) > 0 {
			errMsgs = append(errMsgs, errs...)
		}
	}
	return errMsgs
}

func openstackPoolsValidate(specPools []cephlcmv1alpha1.CephPool) []string {
	openstackRoles := map[string]int{
		"images":  0,
		"vms":     0,
		"backup":  0,
		"volumes": 0,
	}
	// count specified roles for pools
	for _, pool := range specPools {
		if _, ok := openstackRoles[pool.Role]; ok {
			openstackRoles[pool.Role]++
		}
	}
	extraRolesSpecified := []string{}
	rolesNotSpecified := []string{}
	totalCount := 0
	for role, count := range openstackRoles {
		totalCount += count
		if count > 1 && role != "volumes" {
			extraRolesSpecified = append(extraRolesSpecified, role)
		}
		if count == 0 {
			rolesNotSpecified = append(rolesNotSpecified, role)
		}
	}
	// no openstack pools
	if totalCount == 0 {
		return nil
	}
	errMsgs := []string{}
	if len(rolesNotSpecified) > 0 {
		sort.Strings(rolesNotSpecified)
		errMsgs = append(errMsgs, fmt.Sprintf("found pools with Openstack roles, but missed pools with next roles: %v - required to be specified for Openstack", rolesNotSpecified))
	}
	if len(extraRolesSpecified) > 0 {
		sort.Strings(extraRolesSpecified)
		errMsgs = append(errMsgs, fmt.Sprintf("found pools with Openstack roles, but pools with roles %v allowed to be specified only once", extraRolesSpecified))
	}
	return errMsgs
}

func validateFilesystemSpec(cephDpl *cephlcmv1alpha1.CephDeployment, nodesListExpanded []cephlcmv1alpha1.CephDeploymentNode, externalCluster bool) []string {
	if cephDpl.Spec.SharedFilesystem == nil || len(cephDpl.Spec.SharedFilesystem.Filesystems) == 0 {
		return nil
	}

	singleNode := len(nodesListExpanded) == 1
	fsErrors := make([]string, 0)
	mdsCount := 0
	for _, node := range nodesListExpanded {
		if lcmcommon.Contains(node.Roles, "mds") {
			mdsCount = mdsCount + 1
		}
	}
	for _, cephFSSpec := range cephDpl.Spec.SharedFilesystem.Filesystems {
		cephFsSpecCasted, _ := cephFSSpec.GetSpec()
		if len(cephFsSpecCasted.DataPools) == 0 {
			fsErrors = append(fsErrors, fmt.Sprintf("cephfs '%s' has no datapools specified, requires at least one", cephFSSpec.Name))
			continue
		}
		if externalCluster {
			continue
		}

		metapoolName := fmt.Sprintf("cephfs '%s' metadata", cephFSSpec.Name)
		if metaIssues := validatePoolSpec(cephFsSpecCasted.MetadataPool.PoolSpec, true, metapoolName, singleNode, cephDpl.Spec.ExtraOpts); len(metaIssues) > 0 {
			fsErrors = append(fsErrors, metaIssues...)
		}

		for idx, dataPool := range cephFsSpecCasted.DataPools {
			datapoolName := fmt.Sprintf("cephfs '%s' data %s", cephFSSpec.Name, dataPool.Name)
			if idx == 0 {
				if dataPool.Replicated.Size == 0 {
					msg := fmt.Sprintf("%s will be used as default and must use replication only", datapoolName)
					fsErrors = append(fsErrors, msg)
				}
				continue
			}
			if poolIssues := validatePoolSpec(dataPool.PoolSpec, false, datapoolName, singleNode, cephDpl.Spec.ExtraOpts); len(poolIssues) > 0 {
				fsErrors = append(fsErrors, poolIssues...)
			}
		}

		if int(cephFsSpecCasted.MetadataServer.ActiveCount) > mdsCount {
			fsErrors = append(fsErrors, fmt.Sprintf("not enough 'mds' roles specified in nodes spec, cephfs %s requires at least %d, found %d",
				cephFSSpec.Name, cephFsSpecCasted.MetadataServer.ActiveCount, mdsCount))
		}
	}
	return fsErrors
}

func rbdPeersValidate(cephDpl *cephlcmv1alpha1.CephDeployment) string {
	// Currently (Ceph Octopus release) only a single peer is supported where a peer represents a Ceph cluster.
	if cephDpl.Spec.RBDMirror != nil && len(cephDpl.Spec.RBDMirror.Peers) > 1 {
		return "multiple RBD Peers aren't supported yet"
	}
	return ""
}

func validateObjectStorageSpec(cephDpl *cephlcmv1alpha1.CephDeployment, nodesListExpanded []cephlcmv1alpha1.CephDeploymentNode, external bool) []string {
	issues := []string{}
	singleClusterNode := len(nodesListExpanded) == 1

	if cephDpl.Spec.ObjectStorage != nil {
		monNodesCount := int32(0)
		rgwNodesCount := int32(0)
		knownZones := map[string]bool{}
		if external {
			if len(cephDpl.Spec.ObjectStorage.Realms) > 0 {
				issues = append(issues, "cluster is external, rgw realms can't be created")
			}
			if len(cephDpl.Spec.ObjectStorage.Zonegroups) > 0 {
				issues = append(issues, "cluster is external, rgw zonegroups can't be created")
			}
			if len(cephDpl.Spec.ObjectStorage.Zones) > 0 {
				issues = append(issues, "cluster is external, rgw zones can't be created")
			}
		} else {
			knownRealms := map[string]bool{}
			knownZonegroups := map[string]bool{}
			// TODO (degorenko): limit realms,zones,zonegroups to only 1 per cluster for now
			if len(cephDpl.Spec.ObjectStorage.Realms) > 1 {
				issues = append(issues, "more than one realm specified, but currently supported only one realm per cluster")
			}
			if len(cephDpl.Spec.ObjectStorage.Zonegroups) > 1 {
				issues = append(issues, "more than one zonegroup specified, but currently supported only one zonegroup per cluster")
			}
			if len(cephDpl.Spec.ObjectStorage.Zones) > 1 {
				issues = append(issues, "more than one zone specified, but currently supported only one zone per cluster")
			}
			for _, realm := range cephDpl.Spec.ObjectStorage.Realms {
				knownRealms[realm.Name] = true
			}
			for _, zoneGroup := range cephDpl.Spec.ObjectStorage.Zonegroups {
				knownZonegroups[zoneGroup.Name] = true
				zoneGroupCasted, _ := zoneGroup.GetSpec()
				if _, ok := knownRealms[zoneGroupCasted.Realm]; !ok {
					issues = append(issues, fmt.Sprintf("zonegroup '%s' has specified realm '%s', which is not specified in spec", zoneGroup.Name, zoneGroupCasted.Realm))
				}
			}
			for _, zone := range cephDpl.Spec.ObjectStorage.Zones {
				knownZones[zone.Name] = true
				zoneCasted, _ := zone.GetSpec()
				if _, ok := knownZonegroups[zoneCasted.ZoneGroup]; !ok {
					issues = append(issues, fmt.Sprintf("zone '%s' has specified zonegroup '%s', which is not specified in spec", zone.Name, zoneCasted.ZoneGroup))
				}
				t := "zone '%s' %s"
				issues = append(issues, validatePoolSpec(zoneCasted.MetadataPool, true, fmt.Sprintf(t, zone.Name, "metadata"), singleClusterNode, cephDpl.Spec.ExtraOpts)...)
				issues = append(issues, validatePoolSpec(zoneCasted.DataPool, false, fmt.Sprintf(t, zone.Name, "data"), singleClusterNode, cephDpl.Spec.ExtraOpts)...)
			}
			for _, node := range nodesListExpanded {
				if lcmcommon.Contains(node.Roles, "mon") {
					monNodesCount = monNodesCount + 1
				}
				if lcmcommon.Contains(node.Roles, "rgw") {
					rgwNodesCount = rgwNodesCount + 1
				}
			}
		}

		for _, rgw := range cephDpl.Spec.ObjectStorage.Rgws {
			rgwCasted, _ := rgw.GetSpec()
			if rgwCasted.Gateway.Port == 0 {
				issues = append(issues, fmt.Sprintf("rgw '%s' has no port specified", rgw.Name))
			}
			if external {
				if rgwCasted.MetadataPool.Replicated.Size != 0 || rgwCasted.MetadataPool.ErasureCoded.DataChunks != 0 || rgwCasted.MetadataPool.ErasureCoded.CodingChunks != 0 ||
					rgwCasted.DataPool.Replicated.Size != 0 || rgwCasted.DataPool.ErasureCoded.DataChunks != 0 || rgwCasted.DataPool.ErasureCoded.CodingChunks != 0 || rgwCasted.Zone.Name != "" {
					issues = append(issues, fmt.Sprintf("cluster is external, rgw '%s' pools (metadata and data) specification is not allowed", rgw.Name))
				}
				if len(rgwCasted.Gateway.ExternalRgwEndpoints) == 0 {
					issues = append(issues, fmt.Sprintf("external endpoints for rgw '%s' are not provided", rgw.Name))
				}
			} else {
				if rgwCasted.Zone.Name != "" {
					if !knownZones[rgwCasted.Zone.Name] {
						issues = append(issues, fmt.Sprintf("incorrect rgw '%s' configuration, specified zone '%s' is not found", rgw.Name, rgwCasted.Zone.Name))
					}
				} else {
					issues = append(issues, validatePoolSpec(rgwCasted.MetadataPool, true, fmt.Sprintf("rgw '%s' metadata", rgw.Name), singleClusterNode, cephDpl.Spec.ExtraOpts)...)
					issues = append(issues, validatePoolSpec(rgwCasted.DataPool, false, fmt.Sprintf("rgw '%s' data", rgw.Name), singleClusterNode, cephDpl.Spec.ExtraOpts)...)
				}
				if (rgwNodesCount > 0 && rgwCasted.Gateway.Instances > rgwNodesCount) ||
					(rgwNodesCount == 0 && rgwCasted.Gateway.Instances > monNodesCount) {
					issues = append(issues, fmt.Sprintf("not enough 'rgw' roles specified in nodes spec, rgw '%s' requires at least %d, found %d", rgw.Name, rgwCasted.Gateway.Instances, rgwNodesCount+monNodesCount))
				}
			}
		}

		for _, user := range cephDpl.Spec.ObjectStorage.Users {
			userCasted, _ := user.GetSpec()
			if userCasted.Store == "" {
				issues = append(issues, fmt.Sprintf("object store user '%s' has no related rgw set ('store' field)", user.Name))
			} else {
				found := false
				for _, rgw := range cephDpl.Spec.ObjectStorage.Rgws {
					if rgw.Name == userCasted.Store {
						found = true
						break
					}
				}
				if !found {
					issues = append(issues, fmt.Sprintf("object store user '%s' has unknown rgw set ('store' field)", user.Name))
				}
			}
		}
	}
	return issues
}

func validatePoolSpec(spec cephv1.PoolSpec, metapool bool, poolName string, singleNode bool, extraOpts *cephlcmv1alpha1.CephDeploymentExtraOpts) []string {
	if metapool {
		if spec.Replicated.Size == 0 || (spec.ErasureCoded.DataChunks != 0 || spec.ErasureCoded.CodingChunks != 0) {
			return []string{fmt.Sprintf("%s pool must be only replicated", poolName)}
		}
	}

	if (spec.Replicated.Size == 0 && spec.ErasureCoded.DataChunks == 0 && spec.ErasureCoded.CodingChunks == 0) ||
		(spec.Replicated.Size > 0 && (spec.ErasureCoded.DataChunks > 0 || spec.ErasureCoded.CodingChunks > 0)) {
		return []string{fmt.Sprintf("%s pool should be either replicated or erasureCoded", poolName)}
	}

	issues := []string{}
	if spec.ErasureCoded.DataChunks > 0 || spec.ErasureCoded.CodingChunks > 0 {
		if spec.ErasureCoded.DataChunks < 2 {
			issues = append(issues, fmt.Sprintf("erasureCoded %s pool needs dataChunks set to at least 2", poolName))
		}
		if spec.ErasureCoded.CodingChunks < 1 {
			issues = append(issues, fmt.Sprintf("erasureCoded %s pool needs codingChunks set to at least 1", poolName))
		}
	}

	if err := validateDeviceClassName(spec.DeviceClass, extraOpts); err != nil {
		issues = append(issues, fmt.Sprintf("%s pool has %s", poolName, err.Error()))
	}
	if spec.FailureDomain == "osd" && !singleNode {
		issues = append(issues, fmt.Sprintf("%s pool contains prohibited 'osd' failureDomain", poolName))
	}
	return issues
}

func validateDeviceClassName(deviceClass string, extraOpts *cephlcmv1alpha1.CephDeploymentExtraOpts) error {
	customDeviceClasses := make([]string, 0)
	if extraOpts != nil && len(extraOpts.CustomDeviceClasses) > 0 {
		customDeviceClasses = extraOpts.CustomDeviceClasses
	}
	validNames := append([]string{"hdd", "nvme", "ssd"}, customDeviceClasses...)
	if deviceClass == "" {
		return fmt.Errorf("no deviceClass specified (default valid options are: %v, either specify custom classes)", validNames)
	}
	for _, className := range validNames {
		if className == deviceClass {
			return nil
		}
	}
	return fmt.Errorf("unknown deviceClass '%s' (default valid options are: %v, either specify custom classes)", deviceClass, validNames)
}
