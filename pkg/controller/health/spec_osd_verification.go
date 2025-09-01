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

package health

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	cephv1 "github.com/rook/rook/pkg/apis/ceph.rook.io/v1"

	lcmv1alpha1 "github.com/Mirantis/pelagia/pkg/apis/ceph.pelagia.lcm/v1alpha1"
	lcmcommon "github.com/Mirantis/pelagia/pkg/common"
)

const (
	maxParallelThreads = 5
)

var (
	timeRetrySleep = time.Second * 5
)

type nodeDetails map[string]osdDetails

type osdDetails struct {
	ClusterFSID      string
	DeviceName       string
	DeviceByID       string
	DeviceByPath     string
	BlockPartition   string
	MetaDeviceName   string
	MetaDeviceByID   string
	MetaDeviceByPath string
	MetaPartition    string
	UUID             string
	Up               bool
	In               bool
}

func (c *cephDeploymentHealthConfig) getOsdClusterDetails() (map[string]nodeDetails, error) {
	var osdsMetadataInfo []lcmcommon.OsdMetadataInfo
	cmd := "ceph osd metadata -f json"
	err := lcmcommon.RunAndParseCephToolboxCLI(c.context, c.api.Kubeclientset, c.api.Config, c.lcmConfig.RookNamespace, cmd, &osdsMetadataInfo)
	if err != nil {
		c.log.Error().Err(err).Msg("")
		return nil, err
	}
	var osdsInfo []lcmcommon.OsdInfo
	cmd = "ceph osd info -f json"
	err = lcmcommon.RunAndParseCephToolboxCLI(c.context, c.api.Kubeclientset, c.api.Config, c.lcmConfig.RookNamespace, cmd, &osdsInfo)
	if err != nil {
		c.log.Error().Err(err).Msg("")
		return nil, err
	}

	devicesMap := map[string]nodeDetails{}
	for idx, osdMetadataInfo := range osdsMetadataInfo {
		name := fmt.Sprintf("osd.%d", osdMetadataInfo.OsdID)

		var up, in bool
		osdUUID := ""
		// in most cases metadata(osdsMetadataInfo) list and osdsInfo list will be sorted by osdID, and
		// we can use the same index in both of lists, but in case of any inconsistent we have
		// to find corresponding osdID in osdsInfo list
		if len(osdsMetadataInfo) == len(osdsInfo) && osdMetadataInfo.OsdID == osdsInfo[idx].OsdID {
			up = osdsInfo[idx].Up != 0
			in = osdsInfo[idx].In != 0
			osdUUID = osdsInfo[idx].UUID
		} else {
			for _, osdInfo := range osdsInfo {
				if osdMetadataInfo.OsdID == osdInfo.OsdID {
					up = osdInfo.Up != 0
					in = osdInfo.In != 0
					osdUUID = osdInfo.UUID
					break
				}
			}
		}

		deviceName := strings.Split(osdMetadataInfo.BluestoreDevices, ",")[0]
		metaDeviceName := strings.Split(osdMetadataInfo.MetadataDevices, ",")[0]

		var id, metaID string
		for _, deviceID := range strings.Split(osdMetadataInfo.DeviceIDs, ",") {
			if strings.HasPrefix(deviceID, deviceName) {
				splitID := strings.Split(deviceID, "=")
				id = splitID[len(splitID)-1]
			} else if strings.HasPrefix(deviceID, metaDeviceName) {
				splitID := strings.Split(deviceID, "=")
				metaID = splitID[len(splitID)-1]
			}
		}

		var path, metaPath string
		for _, devicePath := range strings.Split(osdMetadataInfo.DevicePathes, ",") {
			if strings.HasPrefix(devicePath, deviceName) {
				splitPath := strings.Split(devicePath, "=")
				path = splitPath[len(splitPath)-1]
			} else if strings.HasPrefix(devicePath, metaDeviceName) {
				splitPath := strings.Split(devicePath, "=")
				metaPath = splitPath[len(splitPath)-1]
			}
		}

		if osdMetadataInfo.Hostname == "" {
			osdMetadataInfo.Hostname = lcmcommon.StrayOsdNodeMarker
		}

		osdDetails := osdDetails{
			ClusterFSID:      c.healthConfig.cephCluster.Status.CephStatus.FSID,
			DeviceName:       deviceName,
			DeviceByID:       id,
			DeviceByPath:     path,
			BlockPartition:   osdMetadataInfo.BluestorePartition,
			MetaDeviceName:   metaDeviceName,
			MetaDeviceByID:   metaID,
			MetaDeviceByPath: metaPath,
			MetaPartition:    osdMetadataInfo.MetadataPartition,
			Up:               up,
			In:               in,
			UUID:             osdUUID,
		}
		if devicesMap[osdMetadataInfo.Hostname] == nil {
			devicesMap[osdMetadataInfo.Hostname] = nodeDetails{name: osdDetails}
		} else {
			devicesMap[osdMetadataInfo.Hostname][name] = osdDetails
		}
	}
	return devicesMap, nil
}

func (c *cephDeploymentHealthConfig) getSpecAnalysisStatus() (*lcmv1alpha1.OsdSpecAnalysisState, []string) {
	if c.healthConfig.cephCluster.Spec.External.Enable {
		return nil, nil
	}
	if lcmcommon.Contains(c.lcmConfig.HealthParams.ChecksSkip, specAnalysisCheck) {
		c.log.Debug().Msgf("skipping cephcluster spec OSD analysis check, set '%s' to skip through lcm config settings", specAnalysisCheck)
		return nil, nil
	}
	osdClusterDetails, err := c.getOsdClusterDetails()
	if err != nil {
		c.log.Error().Err(err).Msg("")
		return nil, []string{"failed to get osd cluster info"}
	}

	issues := []string{}
	diskDaemonState, numberReady := c.getDaemonSetStatus(c.healthConfig.namespace, lcmcommon.PelagiaDiskDaemon)
	newStatus := &lcmv1alpha1.OsdSpecAnalysisState{DiskDaemon: diskDaemonState}
	if len(newStatus.DiskDaemon.Issues) > 0 {
		issues = append(issues, newStatus.DiskDaemon.Issues...)
	}
	if numberReady == 0 {
		return newStatus, issues
	}

	newStatus.CephClusterSpecGeneration = &c.healthConfig.cephCluster.Generation
	if len(c.healthConfig.cephCluster.Spec.Storage.Nodes) == 0 {
		c.log.Debug().Msg("skipping cephcluster storage spec analyse, since no nodes specified")
		return newStatus, issues
	}
	specStatus, specIssues := c.prepareSpecAnalysis(osdClusterDetails)
	if len(specIssues) > 0 {
		issues = append(issues, specIssues...)
	}
	newStatus.SpecAnalysis = specStatus

	return newStatus, issues
}

func (c *cephDeploymentHealthConfig) prepareSpecAnalysis(osdClusterInfo map[string]nodeDetails) (map[string]lcmv1alpha1.DaemonStatus, []string) {
	var issues []string
	var wg sync.WaitGroup
	// count threads to keep them no more than maxParallelThreads
	parallelThreadsCount := struct {
		mu    sync.RWMutex
		count int
	}{
		count: 0,
	}
	// since issues detected in parallel threads save them safely
	statusThreads := struct {
		mu            sync.RWMutex
		daemonsStatus map[string]lcmv1alpha1.DaemonStatus
		issues        []string
	}{
		daemonsStatus: map[string]lcmv1alpha1.DaemonStatus{},
		issues:        []string{},
	}
	// update thread's count with lock to avoid data race
	disposeThreads := func(delta int) bool {
		parallelThreadsCount.mu.Lock()
		defer parallelThreadsCount.mu.Unlock()

		// if we want run new thread - be sure that we really can
		if delta > 0 {
			if parallelThreadsCount.count == maxParallelThreads {
				return false
			}
		}
		parallelThreadsCount.count += delta
		return true
	}
	// update thread's issues with lock to avoid data race
	updateStatus := func(nodeName string, status lcmv1alpha1.DaemonStatus, nodeIssue string) {
		statusThreads.mu.Lock()
		defer statusThreads.mu.Unlock()
		statusThreads.daemonsStatus[nodeName] = status
		if nodeIssue != "" {
			statusThreads.issues = append(statusThreads.issues, nodeIssue)
		}
	}
	for _, node := range c.healthConfig.cephCluster.Spec.Storage.Nodes {
		// should not happen, but since name field is optional
		// double check and skip if any found
		if node.Name == "" {
			c.log.Warn().Msg("found node without name field in cephcluster storage nodes spec")
			continue
		}
		for !disposeThreads(1) {
			time.Sleep(time.Second * 1)
		}
		wg.Add(1)
		go func() {
			defer func() {
				disposeThreads(-1)
				wg.Done()
			}()
			var issue string
			status, extraFound := c.getNodeAnalyseStatus(c.healthConfig.namespace, node, osdClusterInfo)
			if len(status.Issues) > 0 {
				issue = fmt.Sprintf("node '%s' has failed spec analyse", node.Name)
			}
			if extraFound {
				issues = append(issues, fmt.Sprintf("node '%s' has running osd(s), not described in spec", node.Name))
			}
			updateStatus(node.Name, status, issue)
		}()
	}
	wg.Wait()
	if len(statusThreads.issues) > 0 {
		issues = append(issues, statusThreads.issues...)
		sort.Strings(issues)
	}
	return statusThreads.daemonsStatus, issues
}

func (c *cephDeploymentHealthConfig) getNodeAnalyseStatus(namespace string, node cephv1.Node, osdClusterInfo map[string]nodeDetails) (lcmv1alpha1.DaemonStatus, bool) {
	knode, err := lcmcommon.GetNode(c.context, c.api.Kubeclientset, node.Name)
	if err != nil {
		c.log.Error().Err(err).Msg("")
		return lcmv1alpha1.DaemonStatus{
			Status: lcmv1alpha1.DaemonStateFailed,
			Issues: []string{fmt.Sprintf("failed to get node '%s' info", node.Name)},
		}, false
	}
	if !lcmcommon.IsNodeWithDiskDaemon(*knode, c.lcmConfig.DiskDaemonPlacementLabel) {
		c.log.Warn().Msgf("node '%s', present in cluster spec, has missed disk daemon label '%s'", node.Name, c.lcmConfig.DiskDaemonPlacementLabel)
		return lcmv1alpha1.DaemonStatus{
			Status:   lcmv1alpha1.DaemonStateSkipped,
			Messages: []string{"disk daemon is not running for node (missed daemon label), spec analysis skipped"},
		}, false
	}
	if ok, reason := lcmcommon.IsNodeAvailable(*knode); !ok {
		c.log.Warn().Msg(reason)
		return lcmv1alpha1.DaemonStatus{
			Status: lcmv1alpha1.DaemonStateFailed,
			Issues: []string{reason},
		}, false
	}
	notReadyRetries := 3
	var diskDaemonReport lcmcommon.DiskDaemonReport
	for {
		cmd := fmt.Sprintf("%s --full-report --port %d", lcmcommon.PelagiaDiskDaemon, c.lcmConfig.DiskDaemonPort)
		err := lcmcommon.RunAndParseDiskDaemonCLI(c.context, c.api.Kubeclientset, c.api.Config, namespace, node.Name, cmd, &diskDaemonReport)
		if err != nil {
			c.log.Error().Err(err).Msg("")
			return lcmv1alpha1.DaemonStatus{
				Status: lcmv1alpha1.DaemonStateFailed,
				Issues: []string{fmt.Sprintf("failed to run '%s' command to get disk report from %s", cmd, lcmcommon.PelagiaDiskDaemon)},
			}, false
		}
		if diskDaemonReport.State == lcmcommon.DiskDaemonStateFailed {
			c.log.Error().Msgf("disk report for node '%s' is failed: %v", node.Name, diskDaemonReport.Issues)
			return lcmv1alpha1.DaemonStatus{Status: lcmv1alpha1.DaemonStateFailed, Issues: []string{"disk report is failed"}}, false
		}
		if diskDaemonReport.State != lcmcommon.DiskDaemonStateOk {
			if notReadyRetries > 0 {
				c.log.Warn().Msgf("waiting disk report for node '%s', not ready, waiting", node.Name)
				time.Sleep(timeRetrySleep)
				notReadyRetries--
				continue
			}
			c.log.Error().Msgf("disk report for node '%s', not ready", node.Name)
			return lcmv1alpha1.DaemonStatus{Status: lcmv1alpha1.DaemonStateFailed, Issues: []string{"disk report is not ready"}}, false
		}
		break
	}
	// skip PVC based nodes and nodes with full device usage
	if node.UseAllDevices != nil && *node.UseAllDevices {
		return lcmv1alpha1.DaemonStatus{
			Status:   lcmv1alpha1.DaemonStateSkipped,
			Messages: []string{"used 'useAllDevices' flag for node definition, spec analysis skipped"},
		}, false
	}
	if len(node.VolumeClaimTemplates) > 0 {
		return lcmv1alpha1.DaemonStatus{
			Status:   lcmv1alpha1.DaemonStateSkipped,
			Messages: []string{"pvc based node, spec analysis skipped"},
		}, false
	}

	nodeSpecStatus := lcmv1alpha1.DaemonStatus{Status: lcmv1alpha1.DaemonStateOk}
	analyseIssues, analyseWarnings, extraFound := runSpecAnalysis(node, &diskDaemonReport, osdClusterInfo[node.Name])
	if len(analyseIssues) > 0 {
		c.log.Error().Msgf("found problem(s) with nodespec device configuration for node '%s': %v", node.Name, analyseIssues)
		nodeSpecStatus.Status = lcmv1alpha1.DaemonStateFailed
		nodeSpecStatus.Issues = analyseIssues
	}
	if len(analyseWarnings) > 0 {
		c.log.Warn().Msgf("found configuration deviation for device(s) on node '%s': %v", node.Name, analyseWarnings)
		nodeSpecStatus.Messages = analyseWarnings
	}
	return nodeSpecStatus, extraFound
}

func findDisksForFilter(filter string, byName bool, disksInfo map[string]lcmcommon.BlockDeviceInfo) []string {
	devicesFound := []string{}
	filterRegexp := regexp.MustCompile(filter)
	for dev, devInfo := range disksInfo {
		// skip LVM partitions
		if strings.HasPrefix(dev, "/dev/mapper/") {
			continue
		}
		if byName {
			devName, _ := strings.CutPrefix(dev, "/dev/")
			if filterRegexp.MatchString(devName) {
				devicesFound = append(devicesFound, dev)
			}
			continue
		}
		if filterRegexp.MatchString(dev) {
			devicesFound = append(devicesFound, dev)
			continue
		}
		for _, symlink := range devInfo.Symlinks {
			if filterRegexp.MatchString(symlink) {
				devicesFound = append(devicesFound, dev)
				break
			}
		}
	}
	return devicesFound
}

type devForAnalysis struct {
	Dev            string
	FullDiskUsage  bool
	MetadataDevice string
	OsdsPerDevice  int
	Filtered       bool
	SpecifiedAs    string
}

func runSpecAnalysis(node cephv1.Node, diskDaemonReport *lcmcommon.DiskDaemonReport, nodeOsdClusterInfo nodeDetails) ([]string, []string, bool) {
	issues := []string{}
	osdsReflectedBySpec := map[string]bool{}
	nodeConfigMetadataDevice := ""
	nodeConfigOsdsPerDevice := 1
	if v, ok := node.Config["metadataDevice"]; ok {
		nodeConfigMetadataDevice = v
	}
	if v, ok := node.Config["osdsPerDevice"]; ok {
		// covered by spec validation, no reason to check error again
		nodeConfigOsdsPerDevice, _ = strconv.Atoi(v)
	}
	devicesToAnalyse := []devForAnalysis{}
	// check first that all specified devices in spec are exist
	if len(node.Devices) == 0 && (node.DeviceFilter != "" || node.DevicePathFilter != "") {
		if nodeConfigMetadataDevice != "" {
			if _, present := diskDaemonReport.DisksReport.Aliases[lcmcommon.PathDevPrepended(nodeConfigMetadataDevice)]; !present {
				issues = append(issues, fmt.Sprintf("specified metadata device '%s' is not found on a node", nodeConfigMetadataDevice))
				return issues, nil, false
			}
		}
		var devicesFound []string
		byFilter := ""
		if node.DeviceFilter != "" {
			byFilter = node.DeviceFilter
			devicesFound = findDisksForFilter(node.DeviceFilter, true, diskDaemonReport.DisksReport.BlockInfo)
		} else {
			byFilter = node.DevicePathFilter
			devicesFound = findDisksForFilter(node.DevicePathFilter, false, diskDaemonReport.DisksReport.BlockInfo)
		}
		if len(devicesFound) == 0 {
			issues = append(issues, fmt.Sprintf("no devices found for device filter '%s'", byFilter))
		} else {
			if nodeConfigMetadataDevice != "" {
				if regexp.MustCompile(byFilter).MatchString(nodeConfigMetadataDevice) {
					issues = append(issues, fmt.Sprintf("specified metadata device matches device filter '%s'", byFilter))
					return issues, nil, false
				}
			}
			for _, dev := range devicesFound {
				usedDev := diskDaemonReport.DisksReport.Aliases[lcmcommon.PathDevPrepended(dev)]
				devicesToAnalyse = append(devicesToAnalyse, devForAnalysis{
					Dev:            dev,
					FullDiskUsage:  diskDaemonReport.DisksReport.BlockInfo[usedDev].Type == "disk",
					MetadataDevice: nodeConfigMetadataDevice,
					OsdsPerDevice:  nodeConfigOsdsPerDevice,
					Filtered:       true,
					SpecifiedAs:    byFilter,
				})
			}
		}
	} else {
		devAlreadyUsed := map[string]string{}
		metaDevs := map[string]string{}
		for _, cephDev := range node.Devices {
			deviceForAnalysis := devForAnalysis{
				MetadataDevice: nodeConfigMetadataDevice,
				OsdsPerDevice:  nodeConfigOsdsPerDevice,
			}
			if cephDev.FullPath != "" {
				deviceForAnalysis.SpecifiedAs = cephDev.FullPath
			} else {
				deviceForAnalysis.SpecifiedAs = cephDev.Name
			}
			// check simple case, when device is not found on a node or possibly, device name is changed
			diskName, err := lcmcommon.FindDiskName(deviceForAnalysis.SpecifiedAs, diskDaemonReport.DisksReport)
			if err != nil {
				issues = append(issues, fmt.Sprintf("failed to check device '%s' specified in spec: %s", deviceForAnalysis.SpecifiedAs, err.Error()))
				continue
			}
			deviceForAnalysis.Dev = diskName
			if v, ok := cephDev.Config["metadataDevice"]; ok {
				deviceForAnalysis.MetadataDevice = v
			}
			if v, ok := cephDev.Config["osdsPerDevice"]; ok {
				// covered by spec validation, no reason to check error again
				osdsPerDevice, _ := strconv.Atoi(v)
				deviceForAnalysis.OsdsPerDevice = osdsPerDevice
			}
			// check that there is no duplication in spec
			usedDev := diskDaemonReport.DisksReport.Aliases[lcmcommon.PathDevPrepended(deviceForAnalysis.SpecifiedAs)]
			if duplicateFor, ok := devAlreadyUsed[usedDev]; ok {
				msg := fmt.Sprintf("spec device '%s' is duplication usage for item with device '%s' (devices matched by id)",
					deviceForAnalysis.SpecifiedAs, duplicateFor)
				issues = append(issues, msg)
				continue
			}
			if deviceForAnalysis.MetadataDevice != "" {
				metaDev, present := diskDaemonReport.DisksReport.Aliases[lcmcommon.PathDevPrepended(deviceForAnalysis.MetadataDevice)]
				if !present {
					// case when meta specified in node config, not in device config
					if lcmcommon.PathDevPrepended(deviceForAnalysis.MetadataDevice) == lcmcommon.PathDevPrepended(nodeConfigMetadataDevice) {
						issues = append(issues, fmt.Sprintf("specified metadata device '%s' is not found on a node", nodeConfigMetadataDevice))
					} else {
						issues = append(issues, fmt.Sprintf("metadata device '%s' specified for device '%s' is not found on a node",
							deviceForAnalysis.MetadataDevice, deviceForAnalysis.SpecifiedAs))
					}
					continue
				}
				if diskDaemonReport.DisksReport.BlockInfo[metaDev].Type != "disk" && metaDevs[deviceForAnalysis.MetadataDevice] != "" {
					issues = append(issues, fmt.Sprintf("spec device '%s' has metadata device '%s', which is duplication use as meta for device '%s'",
						deviceForAnalysis.SpecifiedAs, deviceForAnalysis.MetadataDevice, metaDevs[deviceForAnalysis.MetadataDevice]))
				}
				metaDevs[deviceForAnalysis.MetadataDevice] = deviceForAnalysis.SpecifiedAs
			}
			deviceForAnalysis.FullDiskUsage = diskDaemonReport.DisksReport.BlockInfo[usedDev].Type == "disk"
			devAlreadyUsed[usedDev] = deviceForAnalysis.SpecifiedAs
			devicesToAnalyse = append(devicesToAnalyse, deviceForAnalysis)
		}
		// check that devices specified for meta are not used for block
		for meta, usedFor := range metaDevs {
			if dev, present := devAlreadyUsed[diskDaemonReport.DisksReport.Aliases[meta]]; present {
				issues = append(issues, fmt.Sprintf("specified as metadata device '%s' (for device '%s') and is used as block device '%s'",
					meta, usedFor, dev))
			}
		}
	}
	// return validation errors if any before checking osd partitions
	if len(issues) > 0 {
		sort.Strings(issues)
		return issues, nil, false
	}
	for _, devToAnalyse := range devicesToAnalyse {
		usedInSpecAs := fmt.Sprintf("device '%s'", devToAnalyse.SpecifiedAs)
		if devToAnalyse.Filtered {
			usedInSpecAs = fmt.Sprintf("device '%s' filtered by '%s'", devToAnalyse.Dev, devToAnalyse.SpecifiedAs)
		}
		osdsOnDevice := 0
		for _, osd := range diskDaemonReport.DisksReport.DiskToOsd[devToAnalyse.Dev] {
			checkedParts := map[string]bool{}
			for _, osdInfo := range diskDaemonReport.OsdsReport.Osds[osd] {
				if osdInfoCluster, present := nodeOsdClusterInfo[fmt.Sprintf("osd.%s", osd)]; present {
					if osdInfo.ClusterFSID == osdInfoCluster.ClusterFSID && osdInfo.OsdUUID == osdInfoCluster.UUID {
						metaFound := false
						for _, osdDev := range osdInfo.Devices {
							lookForBlock := false
							lookForMeta := false
							if osdDev.Name == devToAnalyse.Dev {
								lookForBlock = true
							}
							if devToAnalyse.MetadataDevice != "" {
								lookForMeta = true
							}
							for _, osdPart := range osdInfo.Partitions {
								if osdDev.RelatedPartition != osdPart.Partition {
									continue
								}
								if !devToAnalyse.FullDiskUsage {
									if devToAnalyse.Filtered {
										if !regexp.MustCompile(devToAnalyse.SpecifiedAs).MatchString(osdPart.Partition) {
											continue
										}
										skip := true
										for _, partSym := range osdPart.PartitionSymlinks {
											if regexp.MustCompile(devToAnalyse.SpecifiedAs).MatchString(partSym) {
												skip = false
											}
										}
										if skip {
											continue
										}
									} else if diskDaemonReport.DisksReport.Aliases[osdPart.Partition] != diskDaemonReport.DisksReport.Aliases[devToAnalyse.SpecifiedAs] {
										continue
									}
								}
								switch osdPart.Type {
								case "block":
									if lookForBlock {
										osdsOnDevice++
										osdsReflectedBySpec[osd] = true
										checkedParts[osdPart.Partition] = true
									}
								case "db":
									if lookForBlock {
										issues = append(issues, fmt.Sprintf("%s is specified as block device, but contains db partition '%s'", usedInSpecAs, osdPart.Partition))
									} else if lookForMeta {
										metaDevAliased := diskDaemonReport.DisksReport.Aliases[lcmcommon.PathDevPrepended(devToAnalyse.MetadataDevice)]
										if metaDevAliased == diskDaemonReport.DisksReport.Aliases[lcmcommon.PathDevPrepended(osdPart.Partition)] ||
											metaDevAliased == diskDaemonReport.DisksReport.Aliases[lcmcommon.PathDevPrepended(osdDev.Name)] {
											metaFound = true
											checkedParts[osdPart.Partition] = true
										} else {
											issues = append(issues, fmt.Sprintf("%s has unknown db partition '%s', while expected '%s' (osd %s)",
												usedInSpecAs, osdPart.Partition, devToAnalyse.MetadataDevice, osd))
										}
									} else {
										issues = append(issues, fmt.Sprintf("%s has no specified metadata device, but found related db partition '%s' (osd %s)", usedInSpecAs, osdPart.Partition, osd))
									}
								}
								if !devToAnalyse.FullDiskUsage {
									break
								}
							}
						}
						if devToAnalyse.MetadataDevice != "" && !metaFound {
							issues = append(issues, fmt.Sprintf("metadata device '%s' is not found for osd '%s' for %s", devToAnalyse.MetadataDevice, osd, usedInSpecAs))
						}
					} else {
						// exclude strays osd partitions with same osd id but on different devices
						for _, osdDev := range osdInfo.Devices {
							if osdDev.Name == devToAnalyse.Dev {
								for _, part := range osdInfo.Partitions {
									// if assumed full disk usage - all on it unexpected. Otherwise, if partition specified to use - other just stray
									if devToAnalyse.FullDiskUsage {
										issues = append(issues, fmt.Sprintf("unknow osd '%s' %s partition '%s' is found on %s", osd, part.Type, part.Partition, usedInSpecAs))
									}
								}
							}
						}
					}
				}
			}
		}
		if osdsOnDevice != devToAnalyse.OsdsPerDevice {
			msg := fmt.Sprintf("%s should have %d osd(s), but actually found %d", usedInSpecAs, devToAnalyse.OsdsPerDevice, osdsOnDevice)
			issues = append(issues, msg)
			continue
		}
	}
	// if found issues with alignment - do not check strays
	if len(issues) > 0 {
		sort.Strings(issues)
		return issues, nil, false
	}
	// highlight unknown osds for devices on node, which are not in spec
	warnings := []string{}
	extraFound := false
	for osd, osdVolumes := range diskDaemonReport.OsdsReport.Osds {
		for _, osdVolume := range osdVolumes {
			knownOsd := false
			presentInSpec := osdsReflectedBySpec[osd]
			if info, present := nodeOsdClusterInfo[fmt.Sprintf("osd.%s", osd)]; present {
				if info.UUID == osdVolume.OsdUUID {
					if presentInSpec {
						continue
					}
					knownOsd = true
				}
			}
			if !knownOsd || !presentInSpec {
				for _, dev := range osdVolume.Devices {
					for _, part := range osdVolume.Partitions {
						if dev.RelatedPartition != part.Partition {
							continue
						}
						msg := fmt.Sprintf("found ceph %s partition '%s', belongs to osd '%s'", part.Type, part.Partition, osd)
						if osdVolume.OsdUUID != "" {
							msg = fmt.Sprintf("%s (osd fsid '%s')", msg, osdVolume.OsdUUID)
						}
						if len(osdVolume.Devices) > 0 {
							msg = fmt.Sprintf("%s, placed on '%s' device", msg, dev.Name)
						}
						if knownOsd {
							warnings = append(warnings, fmt.Sprintf("%s, which is not reflected in spec", msg))
							extraFound = true
						} else {
							warnings = append(warnings, fmt.Sprintf("%s, which seems to be stray, can be cleaned up", msg))
						}
						break
					}
				}
			}
		}
	}
	sort.Strings(warnings)
	return nil, warnings, extraFound
}
