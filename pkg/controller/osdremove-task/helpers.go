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
	"strings"

	lcmv1alpha1 "github.com/Mirantis/pelagia/pkg/apis/ceph.pelagia.lcm/v1alpha1"
	lcmcommon "github.com/Mirantis/pelagia/pkg/common"
)

// func to check provided disk is in provided osd daemon info
func checkDisksOrPartition(osdDaemonInfo lcmcommon.OsdDaemonInfo, lookForDev string) bool {
	for _, dev := range osdDaemonInfo.Devices {
		if lcmcommon.PathDevPrepended(lookForDev) == dev.Name {
			return true
		}
		if lcmcommon.Contains(dev.DeviceSymlinks, lookForDev) {
			return true
		}
	}
	for _, part := range osdDaemonInfo.Partitions {
		if lookForDev == part.Partition {
			return true
		}
		if lcmcommon.Contains(part.PartitionSymlinks, lookForDev) {
			return true
		}
	}
	return false
}

// func for filling device's info from ceph osd metadata
func fillDevicesInfoFromMetadata(osdMetadataInfo lcmcommon.OsdMetadataInfo) map[string]lcmv1alpha1.DeviceInfo {
	devices := map[string]lcmv1alpha1.DeviceInfo{}
	// since we always has only 1 device - but to avoid any unexpected situations
	blockDevice := lcmcommon.PathDevPrepended(strings.Split(osdMetadataInfo.BluestoreDevices, ",")[0])
	devices[blockDevice] = lcmv1alpha1.DeviceInfo{
		Rotational: osdMetadataInfo.BluestoreDeviceType == "hdd",
		Type:       "block",
		Partition:  osdMetadataInfo.BluestorePartition,
	}
	if osdMetadataInfo.MetadataDiskUsed == "1" {
		dbDevice := lcmcommon.PathDevPrepended(strings.Split(osdMetadataInfo.MetadataDevices, ",")[0])
		devices[dbDevice] = lcmv1alpha1.DeviceInfo{
			Rotational: osdMetadataInfo.MetadataDeviceType == "hdd",
			Type:       "db",
			Partition:  osdMetadataInfo.MetadataPartition,
		}
	}

	fillDevicePathOrID := func(devices map[string]lcmv1alpha1.DeviceInfo, devArray []string, byId bool) {
		for _, dev := range devArray {
			// check just in case osd metadata output problems or updates to avoid null pointer
			devSplit := strings.Split(dev, "=")
			if len(devSplit) == 2 {
				devName := lcmcommon.PathDevPrepended(devSplit[0])
				// if devices contain unexpectedly some other device do not use it (like wal device)
				if info, ok := devices[devName]; ok {
					if byId {
						info.ID = devSplit[1]
					} else {
						info.Path = devSplit[1]
					}
					devices[devName] = info
				}
			}
		}
	}

	// by default osd metadata show device pathes like: "vdd=/dev/disk/by-path/pci-0000:00:0d.0"
	fillDevicePathOrID(devices, strings.Split(osdMetadataInfo.DevicePathes, ","), false)
	// by default osd metadata show device ids like: "vdd=95ac6e89-2f7e-4427-9"
	fillDevicePathOrID(devices, strings.Split(osdMetadataInfo.DeviceIDs, ","), true)

	return devices
}

type mappingConfig struct {
	nodeInSpec              bool
	nodeAvailable           bool
	host                    string
	osdsToClean             []int
	osdsMetadataFromCluster []lcmcommon.OsdMetadataInfo
	osdMetadataMap          map[int]int
	osdUUIDMap              map[string]string
	clusterFSID             string
	clusterNamespace        string
	clusterHostDir          string
}

// func for filling osd mapping info from ceph osd metadata, when disk daemon is not available
func fillOsdMappingFromOsdMeta(c mappingConfig) (map[string]lcmv1alpha1.OsdMapping, []string) {
	warnings := []string{}
	osdMapping := map[string]lcmv1alpha1.OsdMapping{}
	if c.nodeAvailable {
		warnings = append(warnings, fmt.Sprintf("[node '%s'] node is available, but has no disk daemon running, device cleanup jobs will be skipped", c.host))
	} else {
		if c.nodeInSpec {
			warnings = append(warnings, fmt.Sprintf("[node '%s'] node is not available, but present in spec, device cleanup jobs will be skipped", c.host))
		} else {
			warnings = append(warnings, fmt.Sprintf("[node '%s'] node is not available, device cleanup jobs will be skipped", c.host))
		}
	}
	for _, osd := range c.osdsToClean {
		if osdMetaIdx, present := c.osdMetadataMap[osd]; present {
			deviceMapping := fillDevicesInfoFromMetadata(c.osdsMetadataFromCluster[osdMetaIdx])
			if !c.nodeAvailable {
				for dev, mapping := range deviceMapping {
					mapping.Alive = false
					deviceMapping[dev] = mapping
				}
			}
			osdMapping[fmt.Sprint(osd)] = lcmv1alpha1.OsdMapping{
				UUID:          c.osdUUIDMap[fmt.Sprint(osd)],
				ClusterFSID:   c.clusterFSID,
				InCrushMap:    true,
				DeviceMapping: deviceMapping,
				HostDirectory: fmt.Sprintf("%s/%s/%s_%s", c.clusterHostDir, c.clusterNamespace, c.clusterFSID, c.osdUUIDMap[fmt.Sprint(osd)]),
			}
		} else {
			warnings = append(warnings, fmt.Sprintf("[node '%s'] node has no osd id '%d', skipping", c.host, osd))
		}
	}
	return osdMapping, warnings
}

// func for getting actual warnings, affecting osd mapping
func getReportWarningsInNodeFormat(hostMapping lcmv1alpha1.HostMapping, host string, reportWarnings []string) []string {
	warnings := []string{}
	if hostMapping.CompleteCleanup || hostMapping.DropFromCrush {
		for _, warning := range reportWarnings {
			warnings = append(warnings, fmt.Sprintf("[node '%s'] %s", host, warning))
		}
	} else {
		for osd := range hostMapping.OsdMapping {
			osdID := osd
			if isStrayOsdID(osd) {
				osdID = strings.Split(osd, ".")[0]
			}
			for _, warning := range reportWarnings {
				// compare with disk daemon warning format
				if strings.Contains(warning, fmt.Sprintf("osd '%s'", osdID)) {
					warnings = append(warnings, fmt.Sprintf("[node '%s'] %s", host, warning))
				}
			}
		}
	}
	return warnings
}

func isStrayOsdID(osdID string) bool {
	return strings.HasSuffix(osdID, lcmcommon.StrayOsdNodeMarker)
}
