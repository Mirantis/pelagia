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
	"strconv"
	"strings"

	"github.com/pkg/errors"
	cephv1 "github.com/rook/rook/pkg/apis/ceph.rook.io/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	cephlcmv1alpha1 "github.com/Mirantis/pelagia/pkg/apis/ceph.pelagia.lcm/v1alpha1"
	lcmcommon "github.com/Mirantis/pelagia/pkg/common"
)

func (c *cephDeploymentConfig) validate() cephlcmv1alpha1.CephDeploymentValidation {
	errMsgs := make([]string, 0)
	defaultFound := false
	if c.cdConfig.cephDpl.Spec.BlockStorage != nil {
		for _, cephDplPool := range c.cdConfig.cephDpl.Spec.BlockStorage.Pools {
			if defaultFound && cephDplPool.StorageClassOpts.Default {
				err := "CephDeployment has multiple default pools specified"
				c.log.Error().Msg(err)
				errMsgs = append(errMsgs, err)
			}
			defaultFound = defaultFound || cephDplPool.StorageClassOpts.Default
			castedPool, _ := cephDplPool.GetSpec()
			if err := validateDeviceClassName(castedPool.DeviceClass, c.cdConfig.cephDpl.Spec.ExtraOpts); err != nil {
				err := fmt.Sprintf("CephDeployment pool %s has %s", cephDplPool.Name, err.Error())
				c.log.Error().Msg(err)
				errMsgs = append(errMsgs, err)
			}
			if !c.cdConfig.clusterSpec.External.Enable && ((castedPool.ErasureCoded.CodingChunks == 0 && castedPool.ErasureCoded.DataChunks == 0 && castedPool.Replicated.Size == 0) ||
				(castedPool.ErasureCoded.CodingChunks != 0 && castedPool.ErasureCoded.DataChunks != 0 && castedPool.Replicated.Size != 0)) {
				err := fmt.Sprintf("CephDeployment pool %s spec should contain either replicated or erasureCoded spec", cephDplPool.Name)
				c.log.Error().Msg(err)
				errMsgs = append(errMsgs, err)
			}
			if cephDplPool.StorageClassOpts.ReclaimPolicy != "" && !lcmcommon.Contains([]string{"Retain", "Delete"}, cephDplPool.StorageClassOpts.ReclaimPolicy) {
				err := fmt.Sprintf("CephDeployment pool %s spec contains invalid reclaimPolicy '%s', valid are: %v", cephDplPool.Name, cephDplPool.StorageClassOpts.ReclaimPolicy, []string{"Retain", "Delete"})
				c.log.Error().Msg(err)
				errMsgs = append(errMsgs, err)
			}
			if castedPool.FailureDomain == "osd" && len(c.cdConfig.nodesListExpanded) > 1 {
				err := fmt.Sprintf("CephDeployment pool %s spec contains prohibited 'osd' failureDomain", cephDplPool.Name)
				c.log.Error().Msg(err)
				errMsgs = append(errMsgs, err)
			}
		}
	}
	// do not fail for external case - may only CephFS be specified for usage
	if !defaultFound && !c.cdConfig.clusterSpec.External.Enable {
		err := "CephDeployment has no default pool specified"
		c.log.Error().Msg(err)
		errMsgs = append(errMsgs, err)
	}
	if !c.cdConfig.clusterSpec.External.Enable {
		for _, node := range c.cdConfig.cephDpl.Spec.Nodes {
			if node.UseAllDevices != nil && *node.UseAllDevices {
				errMsg := fmt.Sprintf("detected using 'useAllDevices' for '%s' node item, which is not supported", node.Name)
				c.log.Error().Msg(errMsg)
				errMsgs = append(errMsgs, errMsg)
				continue
			}
			nodeType := "node"
			if node.NodesByLabel != "" || len(node.NodeGroup) > 0 {
				nodeType = "nodeGroup"
			}
			nodeDeviceClass := ""
			if node.Config != nil {
				if node.Config["deviceClass"] != "" {
					nodeDeviceClass = node.Config["deviceClass"]
					if err := validateDeviceClassName(node.Config["deviceClass"], c.cdConfig.cephDpl.Spec.ExtraOpts); err != nil {
						errMsg := fmt.Sprintf("%s config '%s' has %s", nodeType, node.Name, err.Error())
						c.log.Error().Msg(errMsg)
						errMsgs = append(errMsgs, errMsg)
					}
				}
				if node.Config["osdsPerDevice"] != "" {
					_, err := strconv.Atoi(node.Config["osdsPerDevice"])
					if err != nil {
						errMsg := fmt.Sprintf("failed to parse config parameter 'osdsPerDevice' for %s '%s': %s", nodeType, node.Name, err.Error())
						c.log.Error().Msg(errMsg)
						errMsgs = append(errMsgs, errMsg)
					}
				}
			}
			if lcmcommon.IsCephOsdNode(node.Node) {
				if len(node.Devices) > 0 {
					for _, device := range node.Devices {
						deviceClass := ""
						if device.Config != nil {
							if device.Config["deviceClass"] != "" {
								deviceClass = device.Config["deviceClass"]
							}
							if device.Config["osdsPerDevice"] != "" {
								_, err := strconv.Atoi(device.Config["osdsPerDevice"])
								if err != nil {
									errMsg := fmt.Sprintf("failed to parse config parameter 'osdsPerDevice' for device '%s' from %s '%s': %s",
										device.Name, nodeType, node.Name, err.Error())
									c.log.Error().Msg(errMsg)
									errMsgs = append(errMsgs, errMsg)
								}
							}
						}
						// out of device config check because deviceClass must have param - or on node level,
						// or on device level - if set only on node level skip check for device
						if deviceClass == "" && nodeDeviceClass != "" {
							continue
						}
						if err := validateDeviceClassName(deviceClass, c.cdConfig.cephDpl.Spec.ExtraOpts); err != nil {
							deviceName := device.Name
							if device.FullPath != "" {
								deviceName = device.FullPath
							}
							errMsg := fmt.Sprintf("device '%s' on %s '%s' has %s", deviceName, nodeType, node.Name, err.Error())
							c.log.Error().Msg(errMsg)
							errMsgs = append(errMsgs, errMsg)
						}
					}
				} else {
					if nodeDeviceClass == "" {
						errMsg := fmt.Sprintf("deviceClass is not specified for '%s' node item, but it is required", node.Name)
						c.log.Error().Msg(errMsg)
						errMsgs = append(errMsgs, errMsg)
					}
				}
			}
			for crush := range node.Crush {
				if _, ok := crushTopologyAllowedKeys[crush]; !ok {
					err := fmt.Sprintf("CephDeployment node spec for node '%s' contains invalid crush topology key '%s'. Valid are: %v", node.Name, crush, strings.Join(getCrushKeys(), ", "))
					c.log.Error().Msg(err)
					errMsgs = append(errMsgs, err)
				}
			}
		}
		monCount := 0
		mgrCount := 0
		for _, node := range c.cdConfig.nodesListExpanded {
			if lcmcommon.Contains(node.Roles, "mon") {
				monCount = monCount + 1
			}
			if lcmcommon.Contains(node.Roles, "mgr") {
				mgrCount = mgrCount + 1
			}
		}
		// skip check for PRODX-19248
		if len(c.cdConfig.nodesListExpanded) >= 3 && monCount%2 == 0 {
			err := fmt.Sprintf("CephDeployment monitors (roles 'mon') count %d is even, but should be odd for a healthy quorum", monCount)
			c.log.Error().Msg(err)
			errMsgs = append(errMsgs, err)
		}
		if mgrCount == 0 {
			err := "no 'mgr' roles specified, required at least one"
			c.log.Error().Msg(err)
			errMsgs = append(errMsgs, err)
		}
		if err := openstackPoolsValidate(c.cdConfig.cephDpl); err != nil {
			c.log.Error().Err(err).Msg("")
			errMsgs = append(errMsgs, err.Error())
		}
		if err := c.cephDeploymentNodesValidate(); err != nil {
			c.log.Error().Err(err).Msg("")
			errMsgs = append(errMsgs, err.Error())
		}
		if err := validateObjectStorage(c.cdConfig.cephDpl, c.cdConfig.nodesListExpanded, c.cdConfig.clusterSpec.External.Enable); err != nil {
			c.log.Error().Err(err).Msg("")
			errMsgs = append(errMsgs, err.Error())
		}
		switch c.cdConfig.clusterSpec.Network.Provider {
		case "", "host", "multus":
			if c.cdConfig.clusterSpec.Network.AddressRanges == nil {
				err := errors.New("network addressRanges parameter is not specified")
				c.log.Error().Err(err).Msg("")
				errMsgs = append(errMsgs, err.Error())
			} else {
				if len(c.cdConfig.clusterSpec.Network.AddressRanges.Public) == 0 {
					err := errors.New("network addressRanges public parameter is empty")
					c.log.Error().Err(err).Msg("")
					errMsgs = append(errMsgs, err.Error())
				} else {
					for _, net := range c.cdConfig.clusterSpec.Network.AddressRanges.Public {
						if string(net) == "" || strings.HasPrefix(string(net), "0.0.0.0") {
							err := errors.New("network address ranges public parameter should not be empty or contain range 0.0.0.0")
							c.log.Error().Err(err).Msg("")
							errMsgs = append(errMsgs, err.Error())
							break
						}
					}
				}
				if len(c.cdConfig.clusterSpec.Network.AddressRanges.Cluster) == 0 {
					err := errors.New("network addressRanges cluster parameter is empty")
					c.log.Error().Err(err).Msg("")
					errMsgs = append(errMsgs, err.Error())
				} else {
					for _, net := range c.cdConfig.clusterSpec.Network.AddressRanges.Cluster {
						if string(net) == "" || strings.HasPrefix(string(net), "0.0.0.0") {
							err := errors.New("network address ranges cluster parameter should not be empty or contain range 0.0.0.0")
							c.log.Error().Err(err).Msg("")
							errMsgs = append(errMsgs, err.Error())
							break
						}
					}
				}
			}
			if c.cdConfig.clusterSpec.Network.Provider == "multus" {
				if c.cdConfig.clusterSpec.Network.Selectors[cephv1.CephNetworkPublic] == "" || c.cdConfig.clusterSpec.Network.Selectors[cephv1.CephNetworkCluster] == "" {
					err := errors.New("network.selector public and/or cluster parameters should not be empty for provider 'multus'")
					errMsgs = append(errMsgs, err.Error())
				}
			}
		default:
			err := errors.New("network provider parameter should be empty or equals 'host' or 'multus'")
			errMsgs = append(errMsgs, err.Error())
		}
	}
	if err := rbdPeersValidate(c.cdConfig.cephDpl); err != nil {
		c.log.Error().Err(err).Msg("")
		errMsgs = append(errMsgs, err.Error())
	}
	if errs := cephSharedFilesystemValidate(c.cdConfig.cephDpl, c.lcmConfig.RookNamespace, c.cdConfig.nodesListExpanded, c.cdConfig.clusterSpec.External.Enable); len(errs) > 0 {
		c.log.Error().Msgf("errors during shared filesystem settings validation: %v", errs)
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

func cephSharedFilesystemValidate(cephDpl *cephlcmv1alpha1.CephDeployment, rookNamespace string, nodesListExpanded []cephlcmv1alpha1.CephDeploymentNode, external bool) []string {
	fsErrors := make([]string, 0)
	if cephDpl.Spec.SharedFilesystem != nil {
		for _, cephFSSpec := range cephDpl.Spec.SharedFilesystem.Filesystems {
			cephSpecCasted, _ := cephFSSpec.GetSpec()
			if cephSpecCasted.MetadataPool.Replicated.Size == 0 {
				msg := fmt.Sprintf("metadataPool for CephFS %s/%s must use replication only", rookNamespace, cephFSSpec.Name)
				fsErrors = append(fsErrors, msg)
			}
			if len(cephSpecCasted.DataPools) == 0 {
				msg := fmt.Sprintf("dataPools sections for CephFS %s/%s has no data pools defined", rookNamespace, cephFSSpec.Name)
				fsErrors = append(fsErrors, msg)
				continue
			}
			// for cephfs allowed do not specify deviceClass at all
			if err := validateDeviceClassName(cephSpecCasted.MetadataPool.DeviceClass, cephDpl.Spec.ExtraOpts); err != nil {
				msg := fmt.Sprintf("metadataPool for CephFS %s/%s has %s", rookNamespace, cephFSSpec.Name, err.Error())
				fsErrors = append(fsErrors, msg)
			}
			if cephSpecCasted.MetadataPool.FailureDomain == "osd" && len(nodesListExpanded) > 1 {
				msg := fmt.Sprintf("metadataPool for CephFS %s/%s contains prohibited 'osd' failureDomain", rookNamespace, cephFSSpec.Name)
				fsErrors = append(fsErrors, msg)
			}
			for idx, dataPool := range cephSpecCasted.DataPools {
				if err := validateDeviceClassName(dataPool.DeviceClass, cephDpl.Spec.ExtraOpts); err != nil {
					msg := fmt.Sprintf("dataPool %s for CephFS %s/%s has %s", dataPool.Name, rookNamespace, cephFSSpec.Name, err.Error())
					fsErrors = append(fsErrors, msg)
				}
				if dataPool.FailureDomain == "osd" && len(nodesListExpanded) > 1 {
					msg := fmt.Sprintf("dataPool %s for CephFS %s/%s contains prohibited 'osd' failureDomain", dataPool.Name, rookNamespace, cephFSSpec.Name)
					fsErrors = append(fsErrors, msg)
				}
				if idx == 0 {
					if dataPool.Replicated.Size == 0 {
						msg := fmt.Sprintf("dataPool %s will be used as default for CephFS %s/%s and must use replication only", dataPool.Name, rookNamespace, cephFSSpec.Name)
						fsErrors = append(fsErrors, msg)
					}
					continue
				}
				if (dataPool.ErasureCoded.CodingChunks == 0 && dataPool.ErasureCoded.DataChunks == 0) && dataPool.Replicated.Size == 0 {
					msg := fmt.Sprintf("dataPool %s for CephFS %s/%s has no neither replication or erasureCoded sections specified", dataPool.Name, rookNamespace, cephFSSpec.Name)
					fsErrors = append(fsErrors, msg)
				}
			}
			// do not count mds roles for external cluster
			if !external {
				mdsCount := 0
				for _, node := range nodesListExpanded {
					if lcmcommon.Contains(node.Roles, "mds") {
						mdsCount = mdsCount + 1
					}
				}
				if int(cephSpecCasted.MetadataServer.ActiveCount) > mdsCount {
					fsErrors = append(fsErrors, fmt.Sprintf("not enough 'mds' roles specified in nodes spec, CephFS %s/%s requires at least %d",
						rookNamespace, cephFSSpec.Name, cephSpecCasted.MetadataServer.ActiveCount))
				}
			}
		}
	}
	return fsErrors
}

func openstackPoolsValidate(cephDpl *cephlcmv1alpha1.CephDeployment) error {
	expectedRoles := []string{"images", "vms", "backup", "volumes"}
	foundRoles := map[string]int{
		"images":  0,
		"vms":     0,
		"backup":  0,
		"volumes": 0,
	}
	anyRolesFound := false
	extraRolesSpecified := []string{}
	if cephDpl.Spec.BlockStorage != nil {
		for _, pool := range cephDpl.Spec.BlockStorage.Pools {
			if lcmcommon.Contains(expectedRoles, pool.Role) {
				anyRolesFound = true
				foundRoles[pool.Role]++
				if foundRoles[pool.Role] > 1 && pool.Role != "volumes" {
					extraRolesSpecified = append(extraRolesSpecified, pool.Role)
				}
			}
		}
	}
	if len(extraRolesSpecified) > 0 {
		return errors.Errorf("Detected incorrent number of OpenStack Pools with roles: %v - allowed to be specified only once", extraRolesSpecified)
	}
	if !anyRolesFound {
		return nil
	}
	rolesNotSpecified := []string{}
	for role, roleCount := range foundRoles {
		if roleCount == 0 {
			rolesNotSpecified = append(rolesNotSpecified, role)
		}
	}
	if len(rolesNotSpecified) > 0 {
		return errors.Errorf("Not all Openstack required pools was found: missed %v. Or it should not be Openstack pools at all", rolesNotSpecified)
	}
	return nil
}

func rbdPeersValidate(cephDpl *cephlcmv1alpha1.CephDeployment) error {
	// Currently (Ceph Octopus release) only a single peer is supported where a peer represents a Ceph cluster.
	if cephDpl.Spec.RBDMirror != nil && len(cephDpl.Spec.RBDMirror.Peers) > 1 {
		return errors.Errorf("Multiple RBD Peers aren't supported yet")
	}
	return nil
}

func (c *cephDeploymentConfig) cephDeploymentNodesValidate() error {
	unknownNodes := make([]string, 0)
	allNodes, err := lcmcommon.GetNodeList(c.context, c.api.Kubeclientset, metav1.ListOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to get node list")
	}
CephDeploymentNodesLoop:
	for _, cephDplNode := range c.cdConfig.nodesListExpanded {
		for _, node := range allNodes.Items {
			if cephDplNode.Name == node.Name {
				continue CephDeploymentNodesLoop
			}
		}
		unknownNodes = append(unknownNodes, cephDplNode.Name)
	}
	if len(unknownNodes) > 0 {
		return errors.Errorf("The following nodes are present in CephDeployment spec but not present in k8s cluster node list: %s", strings.Join(unknownNodes, ","))
	}
	return nil
}

func validateObjectStorage(cephDpl *cephlcmv1alpha1.CephDeployment, nodesListExpanded []cephlcmv1alpha1.CephDeploymentNode, external bool) error {
	issues := []string{}

	if cephDpl.Spec.ObjectStorage != nil {
		rgwInstancesDesired := int32(0)
		for _, rgw := range cephDpl.Spec.ObjectStorage.Rgws {
			rgwCasted, _ := rgw.GetSpec()
			if external {
				if rgwCasted.MetadataPool.Replicated.Size != 0 || rgwCasted.MetadataPool.ErasureCoded.DataChunks != 0 || rgwCasted.MetadataPool.ErasureCoded.CodingChunks != 0 ||
					rgwCasted.DataPool.Replicated.Size != 0 || rgwCasted.DataPool.ErasureCoded.DataChunks != 0 || rgwCasted.DataPool.ErasureCoded.CodingChunks != 0 {
					issues = append(issues, "rgw in external mode, pools (metadata and data) specification is not allowed")
				}
			} else {
				// find max instances required
				if rgwCasted.Gateway.Instances > rgwInstancesDesired {
					rgwInstancesDesired = rgwCasted.Gateway.Instances
				}
				if rgwCasted.Zone.Name != "" {
					zoneFound := false
					// TODO (degorenko): limit realms,zones,zonegroups to only 1 per cluster for now
					if len(cephDpl.Spec.ObjectStorage.Zones) > 1 {
						issues = append(issues, "more than one zone specified, but currently supported only one zone per cluster")
					}
					if len(cephDpl.Spec.ObjectStorage.Zonegroups) > 1 {
						issues = append(issues, "more than one zonegroup specified, but currently supported only one zonegroup per cluster")
					}
					if len(cephDpl.Spec.ObjectStorage.Realms) > 1 {
						issues = append(issues, "more than one realm specified, but currently supported only one realm per cluster")
					}
					for _, zone := range cephDpl.Spec.ObjectStorage.Zones {
						if zone.Name == rgwCasted.Zone.Name {
							zoneFound = true
							zoneCasted, _ := zone.GetSpec()
							zonegroupFound := false
							for _, zoneGroup := range cephDpl.Spec.ObjectStorage.Zonegroups {
								if zoneGroup.Name == zoneCasted.ZoneGroup {
									zonegroupFound = true
									zoneGroupCasted, _ := zoneGroup.GetSpec()
									realmFound := false
									for _, realm := range cephDpl.Spec.ObjectStorage.Realms {
										if realm.Name == zoneGroupCasted.Realm {
											realmFound = true
											break
										}
									}
									if !realmFound {
										issues = append(issues, fmt.Sprintf("incorrect zonegroup configuration, specified realm '%s' is not found", zoneGroupCasted.Realm))
									}
									break
								}
							}
							if !zonegroupFound {
								issues = append(issues, fmt.Sprintf("incorrect zone configuration, specified zonegroup '%s' is not found", zoneCasted.ZoneGroup))
							} else {
								t := "zone '%s' %s"
								issues = append(issues, validatePoolSpec(zoneCasted.MetadataPool, true, fmt.Sprintf(t, zone.Name, "metadata"), len(nodesListExpanded), cephDpl.Spec.ExtraOpts)...)
								issues = append(issues, validatePoolSpec(zoneCasted.DataPool, false, fmt.Sprintf(t, zone.Name, "data"), len(nodesListExpanded), cephDpl.Spec.ExtraOpts)...)
							}
							break
						}
					}
					if !zoneFound {
						issues = append(issues, fmt.Sprintf("incorrect rgw configuration, specified zone '%s' is not found", rgwCasted.Zone.Name))
					}
				} else {
					issues = append(issues, validatePoolSpec(rgwCasted.MetadataPool, true, "rgw metadata", len(nodesListExpanded), cephDpl.Spec.ExtraOpts)...)
					issues = append(issues, validatePoolSpec(rgwCasted.DataPool, false, "rgw data", len(nodesListExpanded), cephDpl.Spec.ExtraOpts)...)
				}
			}
		}
		if !external {
			monCount := int32(0)
			rgwCount := int32(0)
			for _, node := range nodesListExpanded {
				if lcmcommon.Contains(node.Roles, "mon") {
					monCount = monCount + 1
				}
				if lcmcommon.Contains(node.Roles, "rgw") {
					rgwCount = rgwCount + 1
				}
			}
			if (rgwCount > 0 && rgwInstancesDesired > rgwCount) ||
				(rgwCount == 0 && rgwInstancesDesired > monCount) {
				return fmt.Errorf("not enough 'rgw' roles specified in nodes spec, ObjectStorage section requires at least %d", rgwInstancesDesired)
			}
		}
	}
	if len(issues) > 0 {
		return fmt.Errorf("ObjectStorage section is incorrect: %s", strings.Join(issues, ","))
	}
	return nil
}

func validatePoolSpec(spec cephv1.PoolSpec, metapool bool, poolName string, nodesCount int, extraOpts *cephlcmv1alpha1.CephDeploymentExtraOpts) []string {
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
			issues = append(issues, fmt.Sprintf("erasureCoded %s pool needs dataChunks set to at least 1", poolName))
		}
	}

	if err := validateDeviceClassName(spec.DeviceClass, extraOpts); err != nil {
		issues = append(issues, fmt.Sprintf("%s pool has %s", poolName, err.Error()))
	}
	if spec.FailureDomain == "osd" && nodesCount > 1 {
		issues = append(issues, fmt.Sprintf("%s pool contains prohibited 'osd' failureDomain", poolName))
	}

	return issues
}
