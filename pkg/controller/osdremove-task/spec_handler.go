/*
Copyright 2025 The Mirantis Authors.

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

package osdremove

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	cephv1 "github.com/rook/rook/pkg/apis/ceph.rook.io/v1"
	v1 "k8s.io/api/core/v1"

	lcmv1alpha1 "github.com/Mirantis/pelagia/pkg/apis/ceph.pelagia.lcm/v1alpha1"
	lcmcommon "github.com/Mirantis/pelagia/pkg/common"
)

type osdHostInfo struct {
	inSpec    bool
	specIdx   int
	inCrush   bool
	crushOsds []int
	labeled   bool
	available bool
}

func (c *cephOsdRemoveConfig) getOsdsForCleanup(hostsFromCluster map[string][]int, osdsMetadata []lcmcommon.OsdMetadataInfo,
	osdInfo []lcmcommon.OsdInfo, nodesList []v1.Node) *lcmv1alpha1.TaskRemoveInfo {
	osdsHosts := map[string]osdHostInfo{}
	var nodesFromTaskSpec map[string]lcmv1alpha1.NodeCleanUpSpec
	if c.taskConfig.task.Spec != nil && len(c.taskConfig.task.Spec.Nodes) > 0 {
		nodesFromTaskSpec = c.taskConfig.task.Spec.Nodes
	}

	pickHost := func(hostname string) bool {
		// avoid error cases, when hostname is empty in spec
		if hostname == "" {
			return false
		}
		if len(nodesFromTaskSpec) > 0 {
			_, present := nodesFromTaskSpec[hostname]
			return present
		}
		return true
	}

	for host, osdsOnHost := range hostsFromCluster {
		if pickHost(host) {
			osdsHosts[host] = osdHostInfo{
				crushOsds: osdsOnHost,
				inCrush:   true,
			}
		}
	}
	for idx, node := range c.taskConfig.cephCluster.Spec.Storage.Nodes {
		if pickHost(node.Name) {
			if info, present := osdsHosts[node.Name]; present {
				info.inSpec = true
				info.specIdx = idx
				osdsHosts[node.Name] = info
			} else {
				osdsHosts[node.Name] = osdHostInfo{
					inSpec:  true,
					specIdx: idx,
				}
			}
		}
	}
	// check k8s as last step, because now we have expected hosts from ceph/spec side
	// fill now with hosts with osd role label
	for _, node := range nodesList {
		if pickHost(node.Name) {
			labeled := lcmcommon.IsNodeWithDiskDaemon(node, c.lcmConfig.DiskDaemonPlacementLabel)
			hostInfo, present := osdsHosts[node.Name]
			if present || labeled {
				hostAvailable, reason := lcmcommon.IsNodeAvailable(node)
				if reason != "" {
					c.log.Warn().Msg(reason)
				}
				if present {
					hostInfo.labeled = labeled
					hostInfo.available = hostAvailable
					osdsHosts[node.Name] = hostInfo
				} else {
					osdsHosts[node.Name] = osdHostInfo{
						labeled:   labeled,
						available: hostAvailable,
					}
				}
			}
		}
	}
	if len(nodesFromTaskSpec) > 0 {
		return c.verifyOsdsForCleanUp(nodesFromTaskSpec, c.taskConfig.cephCluster.Spec.Storage.Nodes, osdsHosts, osdsMetadata, osdInfo)
	}
	return c.findOsdsForCleanUp(c.taskConfig.cephCluster.Spec.Storage.Nodes, osdsHosts, osdsMetadata, osdInfo)
}

func (c *cephOsdRemoveConfig) verifyOsdsForCleanUp(nodesFromTask map[string]lcmv1alpha1.NodeCleanUpSpec, nodesList []cephv1.Node,
	osdsHosts map[string]osdHostInfo, osdsMetadataFromCluster []lcmcommon.OsdMetadataInfo, osdInfo []lcmcommon.OsdInfo) *lcmv1alpha1.TaskRemoveInfo {
	newResult := &lcmv1alpha1.TaskRemoveInfo{
		CleanupMap: map[string]lcmv1alpha1.HostMapping{},
		Issues:     []string{},
		Warnings:   []string{},
	}
	osdMetadataMap := map[int]int{}
	strayOsdsNoHost := map[string]bool{}
	for idx, osdMetadataInfo := range osdsMetadataFromCluster {
		osdMetadataMap[osdMetadataInfo.OsdID] = idx
		if osdMetadataInfo.Hostname == "" {
			strayOsdsNoHost[fmt.Sprint(osdMetadataInfo.OsdID)] = true
		}
	}
	osdUUIDMap := map[string]string{}
	for _, osd := range osdInfo {
		osdUUIDMap[fmt.Sprint(osd.OsdID)] = osd.UUID
	}
	dataDirHostPath := lcmcommon.DefaultDataDirHostPath
	if c.taskConfig.cephCluster.Spec.DataDirHostPath != "" {
		dataDirHostPath = c.taskConfig.cephCluster.Spec.DataDirHostPath
	}
	specAnalyse := c.taskConfig.cephHealthOsdAnalysis.SpecAnalysis
	for host, hostSpec := range nodesFromTask {
		// special case for remove strays osds only from crush map will be processed later
		if host == lcmcommon.StrayOsdNodeMarker {
			continue
		}
		// check that spec analyse completed
		if osdsHosts[host].inSpec && osdsHosts[host].labeled && osdsHosts[host].available {
			if state, present := specAnalyse[host]; present {
				if len(state.Issues) > 0 {
					newResult.Issues = append(newResult.Issues, fmt.Sprintf("[node '%s'] spec analyse status has failed, resolve it first", host))
					continue
				}
				if state.Status == lcmv1alpha1.DaemonStateSkipped {
					newResult.Warnings = append(newResult.Warnings, fmt.Sprintf("[node '%s'] spec analyse status skipped, skipping lcm actions", host))
					continue
				}
			} else {
				newResult.Issues = append(newResult.Issues, fmt.Sprintf("[node '%s'] spec analyse status is not available yet", host))
				continue
			}
		}
		// check stray mode first and process it to avoid any futher occasional stray processing
		if hostSpec.CleanupStrayPartitions {
			if osdsHosts[host].available {
				if !osdsHosts[host].labeled {
					newResult.Warnings = append(newResult.Warnings,
						fmt.Sprintf("[node '%s'] node is available, but has no disk daemon running, cleanup stray paritions is not possible, use by id or complete remove, skipping", host))
					continue
				}
			} else {
				newResult.Warnings = append(newResult.Warnings, fmt.Sprintf("[node '%s'] node is not available, cleanup stray paritions is not possible, skipping", host))
				continue
			}
		}
		if !osdsHosts[host].inCrush {
			// allow to cleanup host, which is not in crush only if it is up and available,
			// labeled (role in spec or osd pods on it) and stray cleanup checked. Otherwise - ignore
			if !hostSpec.CleanupStrayPartitions {
				newResult.Warnings = append(newResult.Warnings, fmt.Sprintf("[node '%s'] node is not present in Ceph cluster crush map, skipping", host))
				continue
			}
		}
		// if host in spec and used complete cleanup - do not allow it, no matter available host or not
		if osdsHosts[host].inSpec && (hostSpec.CompleteCleanup || hostSpec.DropFromCrush) {
			newResult.Warnings = append(newResult.Warnings, fmt.Sprintf("[node '%s'] node is present in spec, complete host remove from crush map is not possible, skipping", host))
			continue
		}
		hostMapping := lcmv1alpha1.HostMapping{
			CompleteCleanup: hostSpec.CompleteCleanup,
			DropFromCrush:   hostSpec.DropFromCrush,
			NodeIsDown:      !osdsHosts[host].available,
		}
		// next block is for cases, when host not labeled or not available, so we can't
		// get info from disk daemon and need to take info from ceph osd meta if possible
		if hostMapping.NodeIsDown || !osdsHosts[host].labeled {
			if osdsHosts[host].available {
				warning := fmt.Sprintf("[node '%s'] node is available", host)
				if osdsHosts[host].inSpec {
					newResult.Warnings = append(newResult.Warnings, fmt.Sprintf("%s, but present in spec, has no disk daemon running, skipping", warning))
					continue
				} else if len(hostSpec.CleanupByDevice) > 0 {
					newResult.Warnings = append(newResult.Warnings, fmt.Sprintf("%s, but has no disk daemon running, cleanup by device is not possible, use by id or complete remove, skipping", warning))
					continue
				}
			} else {
				if len(hostSpec.CleanupByDevice) > 0 {
					newResult.Warnings = append(newResult.Warnings, fmt.Sprintf("[node '%s'] node is not available, cleanup by device is not possible, skipping", host))
					continue
				}
			}
			var osdMap map[string]lcmv1alpha1.OsdMapping
			var runtimeWarnings []string
			mappingConfig := mappingConfig{
				nodeAvailable:           osdsHosts[host].available,
				host:                    host,
				osdsMetadataFromCluster: osdsMetadataFromCluster,
				osdMetadataMap:          osdMetadataMap,
				osdUUIDMap:              osdUUIDMap,
				clusterFSID:             c.taskConfig.cephCluster.Status.CephStatus.FSID,
				clusterNamespace:        c.taskConfig.cephCluster.Namespace,
				clusterHostDir:          dataDirHostPath,
			}
			if hostSpec.CompleteCleanup || hostSpec.DropFromCrush {
				mappingConfig.nodeInSpec = false
				mappingConfig.osdsToClean = osdsHosts[host].crushOsds
				osdMap, runtimeWarnings = fillOsdMappingFromOsdMeta(mappingConfig)
			} else {
				ids := make([]int, len(hostSpec.CleanupByOsd))
				for idx := range hostSpec.CleanupByOsd {
					ids[idx] = hostSpec.CleanupByOsd[idx].ID
				}
				mappingConfig.nodeInSpec = osdsHosts[host].inSpec
				mappingConfig.osdsToClean = ids
				osdMap, runtimeWarnings = fillOsdMappingFromOsdMeta(mappingConfig)
			}
			if len(osdMap) > 0 || hostSpec.CompleteCleanup || hostSpec.DropFromCrush {
				hostMapping.OsdMapping = osdMap
				if osdsHosts[host].available {
					hostMapping.VolumesInfoMissed = true
				}
				newResult.CleanupMap[host] = hostMapping
			}
			newResult.Warnings = append(newResult.Warnings, runtimeWarnings...)
			continue
		}
		// now we have situation: host is available plus labeled plus in crush or not in crush with stray mode
		// so use available disk daemon info
		osdsReport, issues := c.tryToGetNodeOsdsReportOrIssues(host)
		if len(issues) > 0 {
			newResult.Issues = append(newResult.Issues, issues...)
			continue
		}
		osdMapping := map[string]lcmv1alpha1.OsdMapping{}
		lockDeviceZap := map[string]bool{}
		var usedDevicesInSpec usedDevices
		if osdsHosts[host].inSpec {
			usedDevicesInSpec = getListUsedDevices(nodesList[osdsHosts[host].specIdx])
		}
		// string to int map for osd ids to access meta from report
		knownHostOsds := map[string]int{}
		if osdsHosts[host].inCrush {
			for _, osd := range osdsHosts[host].crushOsds {
				knownHostOsds[fmt.Sprint(osd)] = osd
			}
		}

		checkFromCrush := func(osd string, osdInfo lcmcommon.OsdDaemonInfo) bool {
			if osdInt, osdPresent := knownHostOsds[osd]; osdPresent {
				if osdInfo.OsdUUID != "" && osdInfo.ClusterFSID != "" {
					return osdInfo.OsdUUID == osdUUIDMap[osd] && osdInfo.ClusterFSID == c.taskConfig.cephCluster.Status.CephStatus.FSID
				}
				// handle migration case, when physical partition is allowed for meta (e.g. migration from nautilus, /dev/sdf4)
				// in that case it may not have uuid/fsid and block device is lost (otherwise uuid/fsid will be set from disk-daemon side)
				// and in case of physical partition - it will always match partition from ceph osd meta
				metadataDevicesInfo := fillDevicesInfoFromMetadata(osdsMetadataFromCluster[osdMetadataMap[osdInt]])
				for _, info := range metadataDevicesInfo {
					if info.Type == "db" {
						for _, part := range osdInfo.Partitions {
							if info.Partition == part.Partition && part.Type == "db" {
								return true
							}
						}
					}
				}
			}
			return false
		}

		if hostSpec.CompleteCleanup || hostSpec.DropFromCrush || hostSpec.CleanupStrayPartitions {
			if !hostSpec.CleanupStrayPartitions {
				// get all meta info about all known host's osds
				for osdStr, osdInt := range knownHostOsds {
					osdMapping[osdStr] = lcmv1alpha1.OsdMapping{
						UUID:          osdUUIDMap[osdStr],
						ClusterFSID:   c.taskConfig.cephCluster.Status.CephStatus.FSID,
						InCrushMap:    true,
						HostDirectory: fmt.Sprintf("%s/%s/%s_%s", dataDirHostPath, c.taskConfig.cephCluster.Namespace, c.taskConfig.cephCluster.Status.CephStatus.FSID, osdUUIDMap[osdStr]),
						DeviceMapping: fillDevicesInfoFromMetadata(osdsMetadataFromCluster[osdMetadataMap[osdInt]]),
					}
				}
			}
			for osd, osdsInfo := range osdsReport.Osds {
				for _, osdInfo := range osdsInfo {
					osdInCrush := checkFromCrush(osd, osdInfo)
					if hostSpec.CleanupStrayPartitions && osdInCrush {
						if osdsHosts[host].inSpec {
							if _, _, inspec := inSpec(osdInfo, usedDevicesInSpec, ""); inspec {
								for _, dev := range osdInfo.Devices {
									lockDeviceZap[dev.Name] = true
								}
							}
						}
						continue
					}
					// since stray osd with that id can be present in crush, but not related to host osds
					strayInCrush := strayOsdsNoHost[osd] && osdInfo.OsdUUID == osdUUIDMap[osd]
					if osdInfo.ClusterFSID == "" && (strayInCrush || osdInCrush) {
						osdInfo.ClusterFSID = c.taskConfig.cephCluster.Status.CephStatus.FSID
					}
					if osdInfo.OsdUUID == "" && (strayInCrush || osdInCrush) {
						osdInfo.OsdUUID = osdUUIDMap[osd]
					}
					osdKey := osd
					// if found as stray in crush or not found in crush at all - let operator know
					if !osdInCrush {
						osdKey = fmt.Sprintf("%s.%s.%s", osd, osdInfo.OsdUUID, lcmcommon.StrayOsdNodeMarker)
						if osdInfo.OsdUUID == "" {
							osdKey = fmt.Sprintf("%s.%s", osd, lcmcommon.StrayOsdNodeMarker)
						}
						newResult.Warnings = append(newResult.Warnings, fmt.Sprintf("[node '%s'] found partition with stray osd uuid '%s', id '%s', will be cleaned up", host, osdInfo.OsdUUID, osd))
						// stray may have same osd id as in crush, but different uuid
						if strayInCrush || (!osdInCrush && osdInfo.OsdUUID == osdUUIDMap[osd]) {
							strayOsdsNoHost[osd] = false
						}
					}
					devsMap := getDevsInfoFromDaemonInfo(osdInfo)
					if presentMapping, present := osdMapping[osdKey]; present {
						for dev, devMapping := range devsMap {
							presentMapping.DeviceMapping[dev] = devMapping
						}
						osdMapping[osdKey] = presentMapping
					} else {
						hostDirectory := ""
						if osdInfo.ClusterFSID != "" && osdInfo.OsdUUID != "" {
							hostDirectory = fmt.Sprintf("%s/%s/%s_%s", dataDirHostPath, c.taskConfig.cephCluster.Namespace, osdInfo.ClusterFSID, osdInfo.OsdUUID)
						}
						osdMapping[osdKey] = lcmv1alpha1.OsdMapping{
							UUID:          osdInfo.OsdUUID,
							ClusterFSID:   osdInfo.ClusterFSID,
							HostDirectory: hostDirectory,
							InCrushMap:    strayInCrush || osdInCrush,
							DeviceMapping: devsMap,
						}
					}
				}
			}
		} else if len(hostSpec.CleanupByOsd) > 0 {
			for _, osdToRemove := range hostSpec.CleanupByOsd {
				osdStr := fmt.Sprint(osdToRemove.ID)
				if _, onHostPresent := knownHostOsds[osdStr]; onHostPresent {
					removeAllowed := true
					// prepare device info from osd metadata first
					devicesInfo := fillDevicesInfoFromMetadata(osdsMetadataFromCluster[osdMetadataMap[osdToRemove.ID]])
					for osd, osdsInfo := range osdsReport.Osds {
						for _, osdInfo := range osdsInfo {
							if osdStr != osd {
								foundInRemoveList := false
								for _, osdItem := range hostSpec.CleanupByOsd {
									if fmt.Sprint(osdItem.ID) == osd {
										foundInRemoveList = true
										break
									}
								}
								if !foundInRemoveList {
									for _, dev := range osdInfo.Devices {
										// lock even stray partition, but it has osd ID different from specified in request
										lockDeviceZap[dev.Name] = true
									}
								}
								continue
							}
							// check that our current info is belongs to in crush osd or it unknown stray
							if !checkFromCrush(osd, osdInfo) {
								for _, dev := range osdInfo.Devices {
									lockDeviceZap[dev.Name] = true
								}
								continue
							}
							if osdsHosts[host].inSpec {
								if inSpecName, devType, inspec := inSpec(osdInfo, usedDevicesInSpec, ""); inspec {
									for _, dev := range osdInfo.Devices {
										if devType == "block" {
											lockDeviceZap[dev.Name] = true
										} else {
											for _, part := range osdInfo.Partitions {
												if part.Partition == dev.RelatedPartition && part.Type == devType {
													lockDeviceZap[dev.Name] = true
													break
												}
											}
											break
										}
									}
									if devType == "block" {
										removeAllowed = false
										newResult.Warnings = append(newResult.Warnings,
											fmt.Sprintf("[node '%s'] osd with id '%d' is associated with block device '%s', which is present in spec, can't cleanup, skipping",
												host, osdToRemove.ID, inSpecName))
										continue
									}
									newResult.Warnings = append(newResult.Warnings,
										fmt.Sprintf("[node '%s'] osd with id '%d' is associated with metadata device '%s', which is present in spec, disk zap will be skipped",
											host, osdToRemove.ID, inSpecName))
								}
							}
							for dev, mapping := range getDevsInfoFromDaemonInfo(osdInfo) {
								devicesInfo[dev] = mapping
							}
						}
					}
					if removeAllowed {
						if osdToRemove.SkipDeviceCleanup {
							newResult.Warnings = append(newResult.Warnings,
								fmt.Sprintf("[node '%s'] osd with id '%d' has 'skip device clean up' flag set in spec. Related deployment should be removed manually as well",
									host, osdToRemove.ID))
						}
						osdMapping[osdStr] = lcmv1alpha1.OsdMapping{
							UUID:                 osdUUIDMap[osdStr],
							ClusterFSID:          c.taskConfig.cephCluster.Status.CephStatus.FSID,
							InCrushMap:           true,
							HostDirectory:        fmt.Sprintf("%s/%s/%s_%s", dataDirHostPath, c.taskConfig.cephCluster.Namespace, c.taskConfig.cephCluster.Status.CephStatus.FSID, osdUUIDMap[osdStr]),
							SkipDeviceCleanupJob: osdToRemove.SkipDeviceCleanup,
							DeviceMapping:        devicesInfo,
						}
					}
				} else {
					newResult.Warnings = append(newResult.Warnings, fmt.Sprintf("[node '%s'] osd with id '%d' is not found on a node, skipping", host, osdToRemove.ID))
				}
			}
		} else if len(hostSpec.CleanupByDevice) > 0 {
			denyDevsToRemove := []string{}
		DevLoop:
			for _, devConfig := range hostSpec.CleanupByDevice {
				devToRemove := devConfig.Device
				found := false
				for osd, osdsInfo := range osdsReport.Osds {
					for _, osdInfo := range osdsInfo {
						if !checkDisksOrPartition(osdInfo, devToRemove) {
							// not our dev to remove - check if it is in spec or not
							if _, _, inspec := inSpec(osdInfo, usedDevicesInSpec, ""); inspec {
								for _, dev := range osdInfo.Devices {
									lockDeviceZap[dev.Name] = true
								}
							}
							continue
						}
						if osdsHosts[host].inSpec {
							if deviceInSpec, _, inspec := inSpec(osdInfo, usedDevicesInSpec, devToRemove); inspec {
								for _, dev := range osdInfo.Devices {
									lockDeviceZap[dev.Name] = true
									// deny remove all related osds to that disks as well
									denyDevsToRemove = append(denyDevsToRemove, dev.Name)
								}
								newResult.Warnings = append(newResult.Warnings,
									fmt.Sprintf("[node '%s'] device '%s' is marked for clean up, but present in spec as '%s'", host, devToRemove, deviceInSpec))
								continue DevLoop
							}
						}
						found = true
						osdInCrush := checkFromCrush(osd, osdInfo)
						// since stray osd with that id can be present in crush, but not related to host osds
						strayInCrush := strayOsdsNoHost[osd] && osdInfo.OsdUUID == osdUUIDMap[osd]
						if osdInfo.ClusterFSID == "" && (strayInCrush || osdInCrush) {
							osdInfo.ClusterFSID = c.taskConfig.cephCluster.Status.CephStatus.FSID
						}
						if osdInfo.OsdUUID == "" && (strayInCrush || osdInCrush) {
							osdInfo.OsdUUID = osdUUIDMap[osd]
						}
						osdKey := osd
						devsMap := getDevsInfoFromDaemonInfo(osdInfo)
						// if found as stray in crush or not found in crush at all - let operator know
						if !osdInCrush {
							osdKey = fmt.Sprintf("%s.%s.%s", osd, osdInfo.OsdUUID, lcmcommon.StrayOsdNodeMarker)
							if osdInfo.OsdUUID == "" {
								osdKey = fmt.Sprintf("%s.%s", osd, lcmcommon.StrayOsdNodeMarker)
							}
							newResult.Warnings = append(newResult.Warnings,
								fmt.Sprintf("[node '%s'] found partition with stray osd uuid '%s', id '%s', will be cleaned up", host, osdInfo.OsdUUID, osd))
							// stray may have same osd id as in crush, but different uuid
							if strayInCrush || (!osdInCrush && osdInfo.OsdUUID == osdUUIDMap[osd]) {
								strayOsdsNoHost[osd] = false
							}
						} else {
							devicesInfo := fillDevicesInfoFromMetadata(osdsMetadataFromCluster[osdMetadataMap[knownHostOsds[osd]]])
							for dev, info := range devicesInfo {
								if _, present := devsMap[dev]; !present {
									devsMap[dev] = info
								}
							}
						}
						if devConfig.SkipDeviceCleanup && osdInCrush {
							newResult.Warnings = append(newResult.Warnings,
								fmt.Sprintf("[node '%s'] device '%s' has set 'skip device clean up' flag set in spec. Related osd deployment (osd id '%s') should be removed manually as well",
									host, devToRemove, osd))
						}
						if presentMapping, present := osdMapping[osdKey]; present {
							for dev, devMapping := range devsMap {
								presentMapping.DeviceMapping[dev] = devMapping
							}
							osdMapping[osdKey] = presentMapping
						} else {
							hostDirectory := ""
							if osdInfo.ClusterFSID != "" && osdInfo.OsdUUID != "" {
								hostDirectory = fmt.Sprintf("%s/%s/%s_%s", dataDirHostPath, c.taskConfig.cephCluster.Namespace, osdInfo.ClusterFSID, osdInfo.OsdUUID)
							}
							osdMapping[osdKey] = lcmv1alpha1.OsdMapping{
								UUID:                 osdInfo.OsdUUID,
								ClusterFSID:          osdInfo.ClusterFSID,
								HostDirectory:        hostDirectory,
								InCrushMap:           strayInCrush || osdInCrush,
								SkipDeviceCleanupJob: devConfig.SkipDeviceCleanup,
								DeviceMapping:        devsMap,
							}
						}
					}
				}
				if !found {
					newResult.Warnings = append(newResult.Warnings,
						fmt.Sprintf("[node '%s'] device '%s' is not found on a node or has no osd partitions to cleanup, skipping", host, devToRemove))
				}
			}
			for _, denyDev := range denyDevsToRemove {
				for osd, mapping := range osdMapping {
					if _, present := mapping.DeviceMapping[denyDev]; present {
						delete(osdMapping, osd)
					}
				}
			}
		}
		if len(osdMapping) > 0 || hostSpec.CompleteCleanup || hostSpec.DropFromCrush {
			if !hostSpec.DropFromCrush {
				if warn := checkDeviceZapping(host, osdMapping, lockDeviceZap, c.lcmConfig.TaskParams.AllowToRemoveManuallyCreatedLVM); len(warn) > 0 {
					newResult.Warnings = append(newResult.Warnings, warn...)
				}
			}
			hostMapping.OsdMapping = osdMapping
			// get only need warnings, related to picked osds
			warnings := getReportWarningsInNodeFormat(hostMapping, host, osdsReport.Warnings)
			newResult.Warnings = append(newResult.Warnings, warnings...)
			newResult.CleanupMap[host] = hostMapping
		}
	}
	// process stray osds, also we already may found some stray during prev steps
	if strayRemove, present := nodesFromTask[lcmcommon.StrayOsdNodeMarker]; present {
		if len(strayRemove.CleanupByOsd) == 0 {
			warning := fmt.Sprintf("[%s] stray which are present in crush map is possible to remove only be osd id", lcmcommon.StrayOsdNodeMarker)
			newResult.Warnings = append(newResult.Warnings, warning)
		} else {
			strayOsdMapping := map[string]lcmv1alpha1.OsdMapping{}
			for _, strayOsd := range strayRemove.CleanupByOsd {
				osdID := fmt.Sprintf("%d", strayOsd.ID)
				if needToRemove, present := strayOsdsNoHost[osdID]; present {
					if needToRemove {
						strayOsdMapping[osdID] = lcmv1alpha1.OsdMapping{
							UUID:        osdUUIDMap[osdID],
							ClusterFSID: c.taskConfig.cephCluster.Status.CephStatus.FSID,
							InCrushMap:  true,
						}
					}
				} else {
					warning := fmt.Sprintf("[%s] stray osd with id '%d' is not found in crush map", lcmcommon.StrayOsdNodeMarker, strayOsd.ID)
					newResult.Warnings = append(newResult.Warnings, warning)
				}
			}
			if len(strayOsdMapping) > 0 {
				newResult.CleanupMap[lcmcommon.StrayOsdNodeMarker] = lcmv1alpha1.HostMapping{
					OsdMapping: strayOsdMapping,
				}
			}
		}
	}
	sort.Strings(newResult.Issues)
	sort.Strings(newResult.Warnings)
	return newResult
}

func (c *cephOsdRemoveConfig) findOsdsForCleanUp(nodesList []cephv1.Node, osdsHosts map[string]osdHostInfo, osdsMetadataFromCluster []lcmcommon.OsdMetadataInfo,
	osdInfo []lcmcommon.OsdInfo) *lcmv1alpha1.TaskRemoveInfo {
	newResult := &lcmv1alpha1.TaskRemoveInfo{
		CleanupMap: map[string]lcmv1alpha1.HostMapping{},
		Issues:     []string{},
		Warnings:   []string{},
	}
	osdMetadataMap := map[int]int{}
	strayOsdsNoHost := map[string]bool{}
	for idx, osdMetadataInfo := range osdsMetadataFromCluster {
		osdMetadataMap[osdMetadataInfo.OsdID] = idx
		if osdMetadataInfo.Hostname == "" {
			strayOsdsNoHost[fmt.Sprint(osdMetadataInfo.OsdID)] = true
		}
	}
	osdUUIDMap := map[string]string{}
	for _, osd := range osdInfo {
		osdUUIDMap[fmt.Sprint(osd.OsdID)] = osd.UUID
	}
	dataDirHostPath := lcmcommon.DefaultDataDirHostPath
	if c.taskConfig.cephCluster.Spec.DataDirHostPath != "" {
		dataDirHostPath = c.taskConfig.cephCluster.Spec.DataDirHostPath
	}
	specAnalyse := c.taskConfig.cephHealthOsdAnalysis.SpecAnalysis
	for host, hostInfo := range osdsHosts {
		hostMapping := lcmv1alpha1.HostMapping{
			CompleteCleanup: !hostInfo.inSpec,
			NodeIsDown:      !hostInfo.available,
		}
		// main case, when node is available, labeled - we can get info from daemon
		if hostInfo.available && hostInfo.labeled {
			var usedDevicesInSpec usedDevices
			var lockDeviceZap map[string]bool
			if hostInfo.inSpec {
				if state, present := specAnalyse[host]; present {
					if len(state.Issues) > 0 {
						newResult.Issues = append(newResult.Issues, fmt.Sprintf("[node '%s'] spec analyse status has failed, resolve it first", host))
						continue
					}
					if state.Status == lcmv1alpha1.DaemonStateSkipped {
						newResult.Warnings = append(newResult.Warnings, fmt.Sprintf("[node '%s'] spec analyse status skipped, skipping lcm actions", host))
						continue
					}
				} else {
					newResult.Issues = append(newResult.Issues, fmt.Sprintf("[node '%s'] spec analyse status is not available yet", host))
					continue
				}
				usedDevicesInSpec = getListUsedDevices(nodesList[hostInfo.specIdx])
				lockDeviceZap = map[string]bool{}
			}
			// try to get node report or issues
			osdsReport, issues := c.tryToGetNodeOsdsReportOrIssues(host)
			if len(issues) > 0 {
				newResult.Issues = append(newResult.Issues, issues...)
				continue
			}
			// string to int map for osd ids to access meta from report
			// and control verified osds
			knownHostOsdsToProcess := map[string]int{}
			for _, osd := range hostInfo.crushOsds {
				knownHostOsdsToProcess[fmt.Sprint(osd)] = osd
			}

			checkFromCrush := func(osd string, osdInfo lcmcommon.OsdDaemonInfo) bool {
				if osdInt, osdPresent := knownHostOsdsToProcess[osd]; osdPresent {
					if osdInfo.OsdUUID != "" && osdInfo.ClusterFSID != "" {
						return osdInfo.OsdUUID == osdUUIDMap[osd] && osdInfo.ClusterFSID == c.taskConfig.cephCluster.Status.CephStatus.FSID
					}
					// handle migration case, when physical partition is allowed for meta (e.g. migration from nautilus, /dev/sdf4)
					// in that case it may not have uuid/fsid and block device is lost (otherwise uuid/fsid will be set from disk-daemon side)
					// and in case of physical partition - it will always match partition from ceph osd meta
					metadataDevicesInfo := fillDevicesInfoFromMetadata(osdsMetadataFromCluster[osdMetadataMap[osdInt]])
					for _, info := range metadataDevicesInfo {
						if info.Type == "db" {
							for _, part := range osdInfo.Partitions {
								if info.Partition == part.Partition && part.Type == "db" {
									return true
								}
							}
						}
					}
				}
				return false
			}
			// possible cases to process are next:
			// * host in spec, in crush - complete check in spec and in crush all found volumes info, remove from node and cluster, if found to remove
			// * host not in spec, in crush - remove host and its osd from cluster, cleanup all osd volumes on a node if found
			// * host not in spec, not in crush - cleanup all osd volumes on a node if found, from cluster remove will be skipped
			// * host in spec, not in crush - spec analyser should have issues, remove host from spec if need to cleanup
			osdMapping := map[string]lcmv1alpha1.OsdMapping{}
			for osd, osdsInfo := range osdsReport.Osds {
				deviceInfoForOsdFromMeta := map[string]lcmv1alpha1.DeviceInfo{}
				expectedOsdDevices := []string{}
				if hostInfo.inCrush {
					if osdIDInt, presentOnHost := knownHostOsdsToProcess[osd]; presentOnHost {
						// start with info from osd metadata and correlate it with on node info
						// that particular found volume info is not for stray
						// also if some volume info is not available we will have info from meta (like block/db volume part lost)
						deviceInfoForOsdFromMeta = fillDevicesInfoFromMetadata(osdsMetadataFromCluster[osdMetadataMap[osdIDInt]])
						expectedOsdDevices = make([]string, 0, len(deviceInfoForOsdFromMeta))
						for dev := range deviceInfoForOsdFromMeta {
							expectedOsdDevices = append(expectedOsdDevices, dev)
						}
					}
				}
				// will be denied only if block dev/part found in spec
				// if some devices lost in a system and still in spec, spec analyser will raise issue
				allowToRemove := len(deviceInfoForOsdFromMeta) > 0
				for _, osdInfo := range osdsInfo {
					fromCrushAndHost := checkFromCrush(osd, osdInfo)
					// if in spec and crush - need to check is device in spec or not - otherwise just remove
					if fromCrushAndHost && hostInfo.inSpec {
						// in the same time can't be used disk and any partition from that disk
						// in that case spec analyser will raise issue, so or used by disk or by part
						if _, devType, inspec := inSpec(osdInfo, usedDevicesInSpec, ""); inspec {
							for _, dev := range osdInfo.Devices {
								if devType == "block" {
									lockDeviceZap[dev.Name] = true
								} else {
									for _, part := range osdInfo.Partitions {
										if part.Partition == dev.RelatedPartition && part.Type == devType {
											lockDeviceZap[dev.Name] = true
											break
										}
									}
									break
								}
							}
							// osd remove not allowed only if block dev found in spec
							// otherwise not allowed only disk zap
							if devType == "block" {
								allowToRemove = false
							}
						}
						if allowToRemove {
							devsMap := getDevsInfoFromDaemonInfo(osdInfo)
							for dev, mapping := range devsMap {
								deviceInfoForOsdFromMeta[dev] = mapping
							}
						}
						continue
					}
					// since stray osd with that id can be present in crush, but not related to host osds
					inCrushAndStray := strayOsdsNoHost[osd] && osdInfo.OsdUUID == osdUUIDMap[osd]
					// w/a for legacy partitions
					if osdInfo.ClusterFSID == "" && (inCrushAndStray || fromCrushAndHost) {
						osdInfo.ClusterFSID = c.taskConfig.cephCluster.Status.CephStatus.FSID
					}
					if osdInfo.OsdUUID == "" && (inCrushAndStray || fromCrushAndHost) {
						osdInfo.OsdUUID = osdUUIDMap[osd]
					}
					osdKey := osd
					// if found as stray in crush or not found in crush at all
					if inCrushAndStray || !fromCrushAndHost {
						warning := fmt.Sprintf("[node '%s'] found partition with stray osd uuid '%s', id '%s', will be cleaned up", host, osdInfo.OsdUUID, osd)
						newResult.Warnings = append(newResult.Warnings, warning)
						osdKey = fmt.Sprintf("%s.%s.%s", osd, osdInfo.OsdUUID, lcmcommon.StrayOsdNodeMarker)
						if osdInfo.OsdUUID == "" {
							osdKey = fmt.Sprintf("%s.%s", osd, lcmcommon.StrayOsdNodeMarker)
						}
						delete(strayOsdsNoHost, osd)
					}
					// case when osd is not in spec or not in crush
					devsMap := getDevsInfoFromDaemonInfo(osdInfo)
					if presentMapping, present := osdMapping[osdKey]; present {
						for dev, devMapping := range devsMap {
							presentMapping.DeviceMapping[dev] = devMapping
						}
						osdMapping[osdKey] = presentMapping
					} else {
						hostDirectory := ""
						if osdInfo.ClusterFSID != "" && osdInfo.OsdUUID != "" {
							hostDirectory = fmt.Sprintf("%s/%s/%s_%s", dataDirHostPath, c.taskConfig.cephCluster.Namespace, osdInfo.ClusterFSID, osdInfo.OsdUUID)
						}
						var devicesMap map[string]lcmv1alpha1.DeviceInfo
						if fromCrushAndHost {
							for dev, devMapping := range devsMap {
								deviceInfoForOsdFromMeta[dev] = devMapping
							}
							devicesMap = deviceInfoForOsdFromMeta
						} else {
							devicesMap = devsMap
						}
						osdMapping[osdKey] = lcmv1alpha1.OsdMapping{
							UUID:          osdInfo.OsdUUID,
							ClusterFSID:   osdInfo.ClusterFSID,
							HostDirectory: hostDirectory,
							InCrushMap:    inCrushAndStray || fromCrushAndHost,
							DeviceMapping: devicesMap,
						}
					}
				}
				// after we check all volumes for osd and found/not found devices in spec
				// finally we may decide remove or not, affects only case when node in spec
				if hostInfo.inSpec && allowToRemove {
					osdMapping[osd] = lcmv1alpha1.OsdMapping{
						UUID:          osdUUIDMap[osd],
						ClusterFSID:   c.taskConfig.cephCluster.Status.CephStatus.FSID,
						InCrushMap:    true,
						HostDirectory: fmt.Sprintf("%s/%s/%s_%s", dataDirHostPath, c.taskConfig.cephCluster.Namespace, c.taskConfig.cephCluster.Status.CephStatus.FSID, osdUUIDMap[osd]),
						DeviceMapping: deviceInfoForOsdFromMeta,
					}
				}
				// verify that we found on a node all expected osd devices, which should be removed
				// and compare against osd metadata info
				if hostInfo.inCrush && len(expectedOsdDevices) > 0 {
					if !allowToRemove {
						delete(knownHostOsdsToProcess, osd)
						continue
					}
					if mapping, present := osdMapping[osd]; present {
						updated := false
						for _, expectedDev := range expectedOsdDevices {
							if _, devPresent := mapping.DeviceMapping[expectedDev]; !devPresent {
								updated = true
								mapping.DeviceMapping[expectedDev] = deviceInfoForOsdFromMeta[expectedDev]
							}
						}
						if updated {
							osdMapping[osd] = mapping
						}
						delete(knownHostOsdsToProcess, osd)
					}
				}
			}
			// check all osds which are not reflected on host, but belongs to current host
			// that means they not in spec and can be removed because lost disk and down
			for osdString, osdInt := range knownHostOsdsToProcess {
				deviceInfoForOsdFromMeta := fillDevicesInfoFromMetadata(osdsMetadataFromCluster[osdMetadataMap[osdInt]])
				osdMapping[osdString] = lcmv1alpha1.OsdMapping{
					UUID:          osdUUIDMap[osdString],
					ClusterFSID:   c.taskConfig.cephCluster.Status.CephStatus.FSID,
					InCrushMap:    true,
					HostDirectory: fmt.Sprintf("%s/%s/%s_%s", dataDirHostPath, c.taskConfig.cephCluster.Namespace, c.taskConfig.cephCluster.Status.CephStatus.FSID, osdUUIDMap[osdString]),
					DeviceMapping: deviceInfoForOsdFromMeta,
				}
			}
			if warn := checkDeviceZapping(host, osdMapping, lockDeviceZap, c.lcmConfig.TaskParams.AllowToRemoveManuallyCreatedLVM); len(warn) > 0 {
				newResult.Warnings = append(newResult.Warnings, warn...)
			}
			if len(osdMapping) > 0 {
				hostMapping.OsdMapping = osdMapping
				// get only need warnings, related to picked osds
				hostWarnings := getReportWarningsInNodeFormat(hostMapping, host, osdsReport.Warnings)
				newResult.Warnings = append(newResult.Warnings, hostWarnings...)
			}
		} else {
			// if node is specified in spec - skip, no matter in crush or not - operator should remove from spec first
			var warning string
			if hostInfo.inSpec {
				if hostInfo.available {
					warning = fmt.Sprintf("[node '%s'] node is available and present in spec, but has no disk daemon running, unable to run auto cleanup, skipping", host)
				} else {
					warning = fmt.Sprintf("[node '%s'] node is present in spec, but is not available, unable to auto detect osds to remove, please specify manually", host)
				}
				newResult.Warnings = append(newResult.Warnings, warning)
				continue
			}
			// case when no info available from daemon, but host is in crush, we can get info from crush
			// all other cases - just remove stray osds from crush
			if hostInfo.inCrush {
				hostMapping.VolumesInfoMissed = hostInfo.available
				mappingConfig := mappingConfig{
					nodeInSpec:              hostInfo.inSpec,
					nodeAvailable:           hostInfo.available,
					host:                    host,
					osdsToClean:             hostInfo.crushOsds,
					osdsMetadataFromCluster: osdsMetadataFromCluster,
					osdMetadataMap:          osdMetadataMap,
					osdUUIDMap:              osdUUIDMap,
					clusterFSID:             c.taskConfig.cephCluster.Status.CephStatus.FSID,
					clusterNamespace:        c.taskConfig.cephCluster.Namespace,
					clusterHostDir:          dataDirHostPath,
				}
				osdMapping, warnings := fillOsdMappingFromOsdMeta(mappingConfig)
				hostMapping.OsdMapping = osdMapping
				newResult.Warnings = append(newResult.Warnings, warnings...)
			}
		}
		// if we found something to remove - add to remove map
		// or just remove host from crush map if empty
		if len(hostMapping.OsdMapping) > 0 || (hostInfo.inCrush && hostMapping.CompleteCleanup) {
			newResult.CleanupMap[host] = hostMapping
		}
	}
	// remove all other stray osds which have no any partitions, but have some deployments and osd in crush
	strayOsdMapping := map[string]lcmv1alpha1.OsdMapping{}
	for strayOsd := range strayOsdsNoHost {
		strayOsdMapping[strayOsd] = lcmv1alpha1.OsdMapping{
			UUID:        osdUUIDMap[strayOsd],
			ClusterFSID: c.taskConfig.cephCluster.Status.CephStatus.FSID,
			InCrushMap:  true,
		}
	}
	if len(strayOsdMapping) > 0 {
		newResult.Warnings = append(newResult.Warnings,
			"[stray] detected stray osds, but impossible to determine related host/device (probably disk(s) removed or host(s) down), device cleanup jobs will be skipped")
		newResult.CleanupMap[lcmcommon.StrayOsdNodeMarker] = lcmv1alpha1.HostMapping{OsdMapping: strayOsdMapping}
	}
	// if found issues, do not unset any other info, request will fail, but
	// user will have some understanding on what is may be done if specify some particular node
	sort.Strings(newResult.Issues)
	sort.Strings(newResult.Warnings)
	return newResult
}

func getDevsInfoFromDaemonInfo(osdDaemonInfo lcmcommon.OsdDaemonInfo) map[string]lcmv1alpha1.DeviceInfo {
	newInfo := map[string]lcmv1alpha1.DeviceInfo{}
	for _, dev := range osdDaemonInfo.Devices {
		devPath := dev.Name
		for _, symlink := range dev.DeviceSymlinks {
			// use by by-path as high priority
			if strings.HasPrefix(symlink, "/dev/disk/by-path/") {
				devPath = symlink
				break
			}
			// udevadm returns sometime symlinks like /dev/disk/by-id/lvm-pv-uuid-KARLw7-BWqc-hBBP-31GT-wM6P-x0Z5-9VffsY
			if strings.HasPrefix(symlink, "/dev/disk/by-id/lvm") {
				continue
			}
			devPath = symlink
		}
		usageType := ""
		for _, part := range osdDaemonInfo.Partitions {
			if part.Partition == dev.RelatedPartition {
				usageType = part.Type
				break
			}
		}
		deviceInfo := lcmv1alpha1.DeviceInfo{
			ID:         dev.DeviceID,
			Rotational: dev.Rotational,
			Path:       devPath,
			Partition:  dev.RelatedPartition,
			Type:       usageType,
			Alive:      true,
		}
		newInfo[dev.Name] = deviceInfo
	}
	return newInfo
}

type usedDevices struct {
	DeviceFilter     string
	DevicePathFilter string
	Devices          map[string]string
}

// convert to target device or mapper name
func getListUsedDevices(node cephv1.Node) usedDevices {
	devices := usedDevices{
		Devices: map[string]string{},
	}
	if dev, ok := node.Config["metadataDevice"]; ok {
		devices.Devices[lcmcommon.PathDevPrepended(dev)] = "db"
	}
	if len(node.Devices) == 0 {
		if node.DeviceFilter != "" {
			devices.DeviceFilter = node.DeviceFilter
		}
		if node.DevicePathFilter != "" {
			devices.DevicePathFilter = node.DevicePathFilter
		}
		return devices
	}
	for _, device := range node.Devices {
		if device.Name != "" {
			devices.Devices[lcmcommon.PathDevPrepended(device.Name)] = "block"
		} else if device.FullPath != "" {
			devices.Devices[device.FullPath] = "block"
		}
		if device.Config["metadataDevice"] != "" {
			devices.Devices[lcmcommon.PathDevPrepended(device.Config["metadataDevice"])] = "db"
		}
	}
	return devices
}

// func to find in spec dev or partition for provided device info
func inSpec(osdDaemonInfo lcmcommon.OsdDaemonInfo, specDevs usedDevices, lookForDev string) (string, string, bool) {
	foundInSpec := ""
	devMapping := map[string][]string{}
	partMapping := map[string][]string{}
	for _, dev := range osdDaemonInfo.Devices {
		for _, part := range osdDaemonInfo.Partitions {
			if part.Partition == dev.RelatedPartition {
				devNames := append([]string{dev.Name, lcmcommon.PathDevPrepended(dev.Name)}, dev.DeviceSymlinks...)
				partNames := append([]string{part.Partition}, part.PartitionSymlinks...)
				if lookForDev == "" || lcmcommon.Contains(devNames, lcmcommon.PathDevPrepended(lookForDev)) || lcmcommon.Contains(partNames, lcmcommon.PathDevPrepended(lookForDev)) {
					devMapping[part.Type] = devNames
					partMapping[part.Type] = partNames
				}
				break
			}
		}
	}
	// if no block partitions, check only meta, otherwise check for block
	devsTypeToCheck := []string{"block", "db"}
	if devNames, blockPresent := devMapping["block"]; blockPresent {
		if specDevs.DeviceFilter != "" || specDevs.DevicePathFilter != "" {
			filter := ""
			if specDevs.DeviceFilter != "" {
				filter = specDevs.DeviceFilter
			} else {
				filter = specDevs.DevicePathFilter
			}
			filterRegexp := regexp.MustCompile(filter)
			for _, name := range devNames {
				if specDevs.DeviceFilter != "" {
					name, _ = strings.CutPrefix(name, "/dev/")
				}
				if filterRegexp.MatchString(name) {
					return filter, "block", true
				}
			}
			// since it is possible to pass non-lvm partition, need to check exact partitions
			for _, partName := range partMapping["block"] {
				if filterRegexp.MatchString(partName) {
					return filter, "block", true
				}
			}
			devsTypeToCheck = []string{"db"}
		}
	}
	for _, devType := range devsTypeToCheck {
		for _, dev := range devMapping[devType] {
			if _, present := specDevs.Devices[dev]; present {
				foundInSpec = dev
				break
			}
		}
		for _, partition := range partMapping[devType] {
			if partType, present := specDevs.Devices[partition]; present && partType == devType {
				foundInSpec = partition
				break
			}
		}
		if foundInSpec != "" {
			return foundInSpec, devType, true
		}
	}
	return "", "", false
}

func checkDeviceZapping(host string, osdMapping map[string]lcmv1alpha1.OsdMapping, lockedDevices map[string]bool, allowToRemoveAllLvms bool) []string {
	warn := []string{}
	for osd, mapping := range osdMapping {
		if mapping.SkipDeviceCleanupJob {
			continue
		}
		for device, devMapping := range mapping.DeviceMapping {
			if !devMapping.Alive {
				continue
			}
			allowLvmDrop := isLvmRookMade(devMapping.Partition) || allowToRemoveAllLvms
			if !allowLvmDrop {
				warn = append(warn, fmt.Sprintf("[node '%s'] found osd %s partition '%s' for osd '%s', which is created not by rook, skipping disk/partition zap",
					host, devMapping.Type, devMapping.Partition, osd))
			}
			if lockedDevices[device] {
				devMapping.Zap = false
			} else {
				devMapping.Zap = allowLvmDrop
			}
			osdMapping[osd].DeviceMapping[device] = devMapping
		}
	}
	return warn
}

func isLvmRookMade(lvmPartition string) bool {
	parts := strings.Split(lvmPartition, "/")
	return strings.HasPrefix(parts[len(parts)-1], lcmcommon.RookLVMarker)
}
