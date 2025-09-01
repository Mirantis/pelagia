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

package diskdaemon

import (
	"fmt"
	"sort"

	lcmcommon "github.com/Mirantis/pelagia/pkg/common"
)

func (d *diskDaemon) checkOsds() []string {
	log.Info().Msg("Preparing osd disk's usage report")
	osdsReport := map[string][]lcmcommon.OsdDaemonInfo{}
	issues := []string{}
	warnings := []string{}
	for osd, osdsVolumesInfo := range d.data.runtime.volumesReport {
		osdInfo := []lcmcommon.OsdDaemonInfo{}
		pathToIdxMap := map[string]int{}
		for _, osdVolumeInfo := range osdsVolumesInfo {
			var devName string
			if len(osdVolumeInfo.Devices) > 1 {
				issue := fmt.Sprintf("multidisk setup detected for osd '%s', partition '%s', which is not supported", osd, osdVolumeInfo.Path)
				log.Error().Msg(issue)
				issues = append(issues, issue)
				break
			} else if len(osdVolumeInfo.Devices) == 0 {
				// support physical parts only for metadata
				if osdVolumeInfo.Type != "db" {
					issue := fmt.Sprintf("found physical osd %s partition '%s' for osd '%s', which is not supported", osdVolumeInfo.Type, osdVolumeInfo.Path, osd)
					log.Error().Msg(issue)
					issues = append(issues, issue)
					break
				}
				name, err := lcmcommon.FindDiskName(osdVolumeInfo.Path, d.data.runtime.disksReport)
				if err != nil {
					issue := fmt.Sprintf("for osd '%s', partition '%s' %s", osd, osdVolumeInfo.Path, err.Error())
					log.Error().Msg(issue)
					issues = append(issues, issue)
					break
				}
				devName = name
			} else {
				devName = osdVolumeInfo.Devices[0]
			}
			devSymlinks := d.data.runtime.disksReport.BlockInfo[devName].Symlinks
			sort.Strings(devSymlinks)
			osdDevice := lcmcommon.OsdDevice{
				Name:             devName,
				DeviceID:         d.data.runtime.disksReport.BlockInfo[devName].Serial,
				DeviceSymlinks:   devSymlinks,
				Rotational:       d.data.runtime.disksReport.BlockInfo[devName].Rotational,
				RelatedPartition: osdVolumeInfo.Path,
			}
			partitionSymlinks := []string{}
			// get block name in case if partition specified not in block dev format
			// like for lvm partition /dev/ceph/blabla - is not block dev, just simlink to /dev/mapper/ceph-blabla
			blockName, blockNamePresent := d.data.runtime.disksReport.Aliases[osdVolumeInfo.Path]
			if blockNamePresent && d.data.runtime.disksReport.BlockInfo[blockName].Type != "disk" {
				partitionSymlinks = d.data.runtime.disksReport.BlockInfo[blockName].Symlinks
				if blockName != osdVolumeInfo.Path {
					partitionSymlinks = append(partitionSymlinks, blockName)
				}
				if d.data.runtime.disksReport.BlockInfo[blockName].Kname != blockName {
					partitionSymlinks = append(partitionSymlinks, d.data.runtime.disksReport.BlockInfo[blockName].Kname)
				}
			}
			sort.Strings(partitionSymlinks)
			osdPartition := lcmcommon.OsdPartition{
				Partition:         osdVolumeInfo.Path,
				PartitionSymlinks: partitionSymlinks,
				Type:              osdVolumeInfo.Type,
				Exists:            blockNamePresent,
				Lvm:               osdVolumeInfo.LvPath != "",
			}
			if !osdPartition.Lvm {
				warning := fmt.Sprintf("found physical osd %s partition '%s' for osd '%s'", osdPartition.Type, osdPartition.Partition, osd)
				log.Warn().Msg(warning)
				warnings = append(warnings, warning)
			}

			var relatedPartition *lcmcommon.OsdPartition
			var info lcmcommon.OsdDaemonInfo
			var existIdx int
			var present bool
			switch osdVolumeInfo.Type {
			case "db":
				if osdVolumeInfo.Tags.BlockDevice != "" {
					if existIdx, present = pathToIdxMap[osdVolumeInfo.Tags.BlockDevice]; present {
						info = osdInfo[existIdx]
					} else {
						pathToIdxMap[osdVolumeInfo.Tags.BlockDevice] = pathToIdxMap[osdVolumeInfo.Path]
						relatedPartition = &lcmcommon.OsdPartition{
							Partition: osdVolumeInfo.Tags.BlockDevice,
							Type:      "block",
						}
					}
				} else if existIdx, present = pathToIdxMap[osdVolumeInfo.Path]; present {
					info = osdInfo[existIdx]
				}
			case "block":
				if osdVolumeInfo.Tags.DBDevice != "" {
					if existIdx, present = pathToIdxMap[osdVolumeInfo.Tags.DBDevice]; present {
						info = osdInfo[existIdx]
					} else {
						pathToIdxMap[osdVolumeInfo.Tags.DBDevice] = pathToIdxMap[osdVolumeInfo.Path]
						relatedPartition = &lcmcommon.OsdPartition{
							Partition: osdVolumeInfo.Tags.DBDevice,
							Type:      "db",
						}
					}
				} else if existIdx, present = pathToIdxMap[osdVolumeInfo.Path]; present {
					info = osdInfo[existIdx]
				}
			}

			// since still supported case with phys partitions for metadata devices, handle it with extra checks:
			// 1) current partition is legacy and found related block partition
			// 2) current partition is block and found legacy related db partition
			curLegacyPart := !osdPartition.Lvm && osdPartition.Type == "db" && (osdVolumeInfo.Tags.OsdFSID == "" || osdVolumeInfo.Tags.ClusterFSID == "")
			legacyPartFound := false
			if !curLegacyPart && (info.OsdUUID == "" || info.ClusterFSID == "") {
				for _, part := range info.Partitions {
					if !part.Lvm && part.Type == "db" {
						legacyPartFound = true
						break
					}
				}
			}
			// since we may found same partition but for completely another osd check fsid
			relatedPartitionOk := (osdVolumeInfo.Tags.OsdFSID == info.OsdUUID && osdVolumeInfo.Tags.ClusterFSID == info.ClusterFSID) || curLegacyPart || legacyPartFound
			if present && relatedPartitionOk {
				// if we found previously and fill for - update current part info
				foundPart := false
				for idx, part := range info.Partitions {
					if part.Partition == osdPartition.Partition {
						info.Partitions[idx] = osdPartition
						foundPart = true
						break
					}
				}
				if !foundPart {
					info.Partitions = append(info.Partitions, osdPartition)
				}
				info.Devices = append(info.Devices, osdDevice)
				if info.ClusterFSID == "" {
					info.ClusterFSID = osdVolumeInfo.Tags.ClusterFSID
				}
				if info.OsdUUID == "" {
					info.OsdUUID = osdVolumeInfo.Tags.OsdFSID
				}
				osdInfo[existIdx] = info
			} else {
				info = lcmcommon.OsdDaemonInfo{
					OsdUUID:     osdVolumeInfo.Tags.OsdFSID,
					ClusterFSID: osdVolumeInfo.Tags.ClusterFSID,
					Devices:     []lcmcommon.OsdDevice{osdDevice},
					Partitions:  []lcmcommon.OsdPartition{osdPartition},
				}
				if relatedPartition != nil {
					info.Partitions = append(info.Partitions, *relatedPartition)
				}
				pathToIdxMap[osdVolumeInfo.Path] = len(osdInfo)
				osdInfo = append(osdInfo, info)
			}
		}
		osdsReport[osd] = osdInfo
	}
	sort.Strings(warnings)
	sort.Strings(issues)
	newOsdReport := &lcmcommon.DiskDaemonOsdsReport{
		Warnings: warnings,
		Osds:     osdsReport,
	}
	if d.data.runtime.osdsReport != nil {
		lcmcommon.ShowObjectDiff(log, d.data.runtime.osdsReport, newOsdReport)
	}
	d.data.runtime.osdsReport = newOsdReport
	log.Info().Msg("Osd disk's usage report is prepared")
	return issues
}
