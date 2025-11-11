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
	"reflect"
	"sort"
	"strings"

	"github.com/pkg/errors"

	lcmcommon "github.com/Mirantis/pelagia/pkg/common"
)

func (d *diskDaemon) checkDisks() (bool, error) {
	newDisksReport, foundLvms, err := processLsblkInfo()
	if err != nil {
		err = errors.Wrap(err, "daemon is failed to prepare block device report")
		log.Error().Err(err).Msg("")
		return false, err
	}
	changed := false
	// if block info is change, check that lvm partitions are up to date and update runtime block info
	if d.data.runtime.disksReport == nil || !reflect.DeepEqual(d.data.runtime.disksReport.BlockInfo, newDisksReport.BlockInfo) {
		log.Info().Msg("Daemon's disks info report is updating")
		if !reflect.DeepEqual(d.data.runtime.knownLvms, foundLvms) {
			if d.data.runtime.knownLvms != nil {
				log.Info().Msg("Found diff in lvm partitions, checking lvm table")
				lcmcommon.ShowObjectDiff(log, d.data.runtime.knownLvms, foundLvms)
			}
			err = checkLvmCached(foundLvms)
			if err != nil {
				err = errors.Wrap(err, "daemon is failed to check cached logical volumes")
				log.Error().Err(err).Msg("")
				return false, err
			}
			d.data.runtime.knownLvms = foundLvms
		}
		if d.data.runtime.disksReport != nil {
			// if some report is already prepared - set only fields affected by block info
			lcmcommon.ShowObjectDiff(log, d.data.runtime.disksReport.BlockInfo, newDisksReport.BlockInfo)
			d.data.runtime.disksReport.BlockInfo = newDisksReport.BlockInfo
			d.data.runtime.disksReport.Aliases = newDisksReport.Aliases
		} else {
			d.data.runtime.disksReport = newDisksReport
		}
		changed = true
	}
	newVolumesReport, err := getCephVolumeLvmList()
	if err != nil {
		err = errors.Wrap(err, "daemon is failed to prepare ceph volumes info report")
		log.Error().Err(err).Msg("")
		return false, err
	}
	if !reflect.DeepEqual(d.data.runtime.volumesReport, newVolumesReport) {
		log.Info().Msg("Daemon's ceph volumes info report is updating")
		lcmcommon.ShowObjectDiff(log, d.data.runtime.volumesReport, newVolumesReport)
		changed = true
		d.data.runtime.volumesReport = newVolumesReport
		d.data.runtime.disksReport.DiskToOsd = d.diskToOsdMap()
	}
	return changed, nil
}

func (d *diskDaemon) diskToOsdMap() map[string][]string {
	if len(d.data.runtime.volumesReport) == 0 {
		return nil
	}
	diskToOsd := map[string]map[string]bool{}
	for osdID, volumes := range d.data.runtime.volumesReport {
		for _, volume := range volumes {
			var devs []string
			if len(volume.Devices) > 0 {
				for _, volDevice := range volume.Devices {
					devs = append(devs, findDisks(d.data.runtime.disksReport.Aliases[volDevice], d.data.runtime.disksReport.BlockInfo)...)
				}
			} else {
				devs = findDisks(d.data.runtime.disksReport.Aliases[volume.Path], d.data.runtime.disksReport.BlockInfo)
			}
			for _, dev := range devs {
				if _, present := diskToOsd[dev]; present {
					diskToOsd[dev][osdID] = true
				} else {
					diskToOsd[dev] = map[string]bool{osdID: true}
				}
			}
		}
	}
	diskToOsdMap := map[string][]string{}
	for dev, mapping := range diskToOsd {
		ids := []string{}
		for osdID := range mapping {
			ids = append(ids, osdID)
		}
		sort.Strings(ids)
		diskToOsdMap[dev] = ids
	}
	return diskToOsdMap
}

func findDisks(name string, blockInfo map[string]lcmcommon.BlockDeviceInfo) []string {
	if blockInfo[name].Type == "disk" {
		return []string{name}
	}
	if len(blockInfo[name].Parent) > 1 {
		devs := []string{}
		for _, name := range blockInfo[name].Parent {
			devs = append(devs, findDisks(name, blockInfo)...)
		}
		return devs
	} else if len(blockInfo[name].Parent) == 0 {
		log.Warn().Msgf("'%s' type of '%s' has no parent", name, blockInfo[name].Type)
		return []string{}
	}
	return findDisks(blockInfo[name].Parent[0], blockInfo)
}

func processLsblkInfo() (*lcmcommon.DiskDaemonDisksReport, map[string][]string, error) {
	lsblkInfo, err := getLsblk()
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to get lsblk info")
	}
	if len(lsblkInfo.Blockdevices) == 0 {
		return nil, nil, errors.New("no blockdevices found for 'lsblk' output")
	}
	report := &lcmcommon.DiskDaemonDisksReport{
		BlockInfo: map[string]lcmcommon.BlockDeviceInfo{},
		Aliases:   map[string]string{},
	}
	foundLvms := map[string][]string{}
	err = loopOverDevices(lsblkInfo.Blockdevices, foundLvms, report)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to prepare block info")
	}
	return report, foundLvms, nil
}

func loopOverDevices(blockDevices []DeviceLsblk, foundLvms map[string][]string, report *lcmcommon.DiskDaemonDisksReport) error {
	for _, blockDevice := range blockDevices {
		if blockDevice.Type == "loop" {
			// always skip loop devices
			continue
		}
		devsLoopErr := loopOverDevices(blockDevice.Childrens, foundLvms, report)
		if devsLoopErr != nil {
			return devsLoopErr
		}
		symlinksStr, err := getUdevadmInfo(blockDevice.Name, "symlink")
		if err != nil {
			return errors.Wrapf(err, "failed to get udevadm info for device '%s'", blockDevice.Name)
		}
		symlinks := strings.Split(strings.TrimSuffix(symlinksStr, "\n"), " ")
		for _, symlink := range symlinks {
			report.Aliases[symlink] = blockDevice.Name
		}
		// sort to avoid not needed diff between reconciles
		sort.Strings(symlinks)
		// kname may be different, for example for /dev/mapper/ it will be /dev/dm-x
		report.Aliases[blockDevice.Kname] = blockDevice.Name
		// put same name for quick access for any alias/name to real one
		report.Aliases[blockDevice.Name] = blockDevice.Name
		if blockDevice.Type == "lvm" {
			// need to convert data path from /dev/mapper/ format to usual /dev format, see example in tests
			symlinkForMapper := strings.ReplaceAll(blockDevice.Name, "/mapper/", "/")
			symlinkForMapper = strings.ReplaceAll(symlinkForMapper, "-", "/")
			symlinkForMapper = strings.ReplaceAll(symlinkForMapper, "//", "-")
			report.Aliases[symlinkForMapper] = blockDevice.Name
			if lvDisks, present := foundLvms[blockDevice.Name]; present {
				if !lcmcommon.Contains(lvDisks, blockDevice.Parent) {
					foundLvms[blockDevice.Name] = append(lvDisks, blockDevice.Parent)
				}
			} else {
				foundLvms[blockDevice.Name] = []string{blockDevice.Parent}
			}
		}
		childrenNames := []string{}
		for _, children := range blockDevice.Childrens {
			childrenNames = append(childrenNames, children.Name)
		}
		if presentInfo, ok := report.BlockInfo[blockDevice.Name]; ok {
			if !lcmcommon.Contains(presentInfo.Parent, blockDevice.Parent) {
				// need to check it because some lvms, raid can be on top of multiple disks/partitions
				presentInfo.Parent = append(presentInfo.Parent, blockDevice.Parent)
				report.BlockInfo[blockDevice.Name] = presentInfo
			}
		} else {
			report.BlockInfo[blockDevice.Name] = lcmcommon.BlockDeviceInfo{
				Kname:      blockDevice.Kname,
				Serial:     blockDevice.Serial,
				Type:       blockDevice.Type,
				Rotational: blockDevice.Rotational,
				MajMin:     blockDevice.MajMin,
				Parent:     []string{blockDevice.Parent},
				Symlinks:   symlinks,
				Childrens:  childrenNames,
			}
		}
	}
	return nil
}

func checkLvmCached(newLvms map[string][]string) error {
	activatedLvs, err := getActiveLvs()
	if err != nil {
		return err
	}
	notActivated := map[string]bool{}
	for lvm, disks := range newLvms {
		if !lcmcommon.Contains(activatedLvs, lvm) {
			for _, disk := range disks {
				notActivated[disk] = true
			}
			log.Info().Msgf("Found logical volume '%s' on disk(s) '%s', which is not cached in daemon container",
				lvm, strings.Join(disks, ","))
		}
	}
	if len(notActivated) == 0 {
		return nil
	}
	// since we have same lvm config as for node, we should not have any
	// different lvms setup detected there, but anyway we will scan only needed pvs
	return cacheLvms(notActivated)
}
