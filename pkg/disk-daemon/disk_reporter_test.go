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
	"strings"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"

	lcmcommon "github.com/Mirantis/pelagia/pkg/common"
	lcmdiskdaemoninput "github.com/Mirantis/pelagia/test/unit/inputs/disk-daemon"
)

func TestProcessLsblkInfo(t *testing.T) {
	tests := []struct {
		name           string
		lsblkOutput    string
		udevadmOutput  map[string]string
		cmdError       string
		expectedReport *lcmcommon.DiskDaemonDisksReport
		expectedLvms   map[string][]string
		expectedError  string
	}{
		{
			name:          "failed to get lsblk info",
			cmdError:      "lsblk",
			expectedError: "failed to get lsblk info: failed to execute lsblk",
		},
		{
			name:          "failed to parse lsblk info",
			lsblkOutput:   "{||}",
			expectedError: "failed to get lsblk info: unable to unmarshal lsblk output: invalid character '|' looking for beginning of object key string",
		},
		{
			name:          "emtpy output",
			lsblkOutput:   "{}",
			expectedError: "no blockdevices found for 'lsblk' output",
		},
		{
			name:          "failed to get udevadm info for some device",
			lsblkOutput:   lcmdiskdaemoninput.LsblkReportFromNode1,
			cmdError:      "udevadm",
			expectedError: "failed to prepare block info: failed to get udevadm info for device '/dev/vda1': failed to execute udevadm",
		},
		{
			name:           "node-1 disk report is ok",
			lsblkOutput:    lcmdiskdaemoninput.LsblkReportFromNode1,
			udevadmOutput:  lcmdiskdaemoninput.UdevadmReportFromNode1,
			expectedLvms:   lcmdiskdaemoninput.FoundLvmsFromNode1,
			expectedReport: lcmdiskdaemoninput.DiskInfoReportLsblkFromNode1,
		},
		{
			name:           "node-2 disk report is ok",
			lsblkOutput:    lcmdiskdaemoninput.LsblkReportFromNode2,
			udevadmOutput:  lcmdiskdaemoninput.UdevadmReportFromNode2,
			expectedLvms:   lcmdiskdaemoninput.FoundLvmsFromNode2,
			expectedReport: lcmdiskdaemoninput.DiskInfoReportLsblkFromNode2,
		},
	}

	oldCmd := runShellCmd
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			runShellCmd = func(command string) (string, string, error) {
				if command == "lsblk -J -p -O" {
					if test.cmdError == "lsblk" {
						return "", "some error happened during lsblk run", errors.New("failed to execute lsblk")
					}
					return test.lsblkOutput, "", nil
				} else if strings.HasPrefix(command, "udevadm info -r --query=symlink") {
					if test.cmdError == "udevadm" {
						return "", "some error happened during udevadm run", errors.New("failed to execute udevadm")
					}
					cmdargs := strings.Split(command, " ")
					dev := cmdargs[len(cmdargs)-1]
					if output, present := test.udevadmOutput[dev]; present {
						return output, "", nil
					}
					return "", "failed to get udevadm for device", errors.Errorf("device '%s' not found", dev)
				}
				return "", "", errors.New("unknown command")
			}

			report, foundLvms, err := processLsblkInfo()
			if test.expectedError != "" {
				assert.NotNil(t, err)
				assert.Equal(t, test.expectedError, err.Error())
				assert.Nil(t, report)
			} else {
				assert.Nil(t, err)
				assert.Equal(t, test.expectedReport, report)
				assert.Equal(t, test.expectedLvms, foundLvms)
			}
		})
	}
	runShellCmd = oldCmd
}

func TestCheckLvmCached(t *testing.T) {
	tests := []struct {
		name          string
		lvmLvsOutput  string
		cmdError      string
		activationLvm bool
		foundLvms     map[string][]string
		expectedError string
	}{
		{
			name:          "failed to get lvs lvm info",
			cmdError:      "lvm lvs",
			expectedError: "failed to execute lvm lvs",
		},
		{
			name:          "failed to parse lvs info",
			lvmLvsOutput:  "{||}",
			expectedError: "unable to unmarshal lvm lvs output: invalid character '|' looking for beginning of object key string",
		},
		{
			name:          "emtpy lvm lvs output requires activation",
			lvmLvsOutput:  "{}",
			foundLvms:     lcmdiskdaemoninput.FoundLvmsFromNode1,
			activationLvm: true,
		},
		{
			name:          "failed to activate lvms",
			lvmLvsOutput:  "{}",
			foundLvms:     lcmdiskdaemoninput.FoundLvmsFromNode1,
			cmdError:      "pvscan",
			activationLvm: true,
			expectedError: "failed to cache volumes: failed to execute pvscan",
		},
		{
			name:          "lvm lvs output - activation required",
			lvmLvsOutput:  lcmdiskdaemoninput.LvmLvsReportFromNode1,
			activationLvm: true,
			foundLvms: map[string][]string{
				"/dev/mapper/new-partition": {"/dev/new-dev"},
			},
		},
		{
			name:         "lvm lvs output - no activation",
			lvmLvsOutput: lcmdiskdaemoninput.LvmLvsReportFromNode1,
			foundLvms:    lcmdiskdaemoninput.FoundLvmsFromNode1,
		},
	}

	activated := false
	oldCmd := runShellCmd
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			activated = false

			runShellCmd = func(command string) (string, string, error) {
				if command == "lvm lvs --reportformat json -o lv_dm_path" {
					if test.cmdError == "lvm lvs" {
						return "", "some error happened during lvm lvs run", errors.New("failed to execute lvm lvs")
					}
					return test.lvmLvsOutput, "", nil
				} else if strings.HasPrefix(command, "pvscan --cache /dev") {
					if !test.activationLvm {
						return "", "", errors.New("unexpected lvm activation")
					}
					activated = true
					if test.cmdError == "pvscan" {
						return "", "some error happened during pvscan run", errors.New("failed to execute pvscan")
					}
					devArg := strings.Split(command, " ")[2]
					devInList := false
					for _, disks := range test.foundLvms {
						if lcmcommon.Contains(disks, devArg) {
							devInList = true
							break
						}
					}
					if !devInList {
						return "", "unexpected device", errors.New("unexpected device wants to recache")
					}
					return "", "", nil
				}
				return "", "", errors.New("unknown command")
			}

			err := checkLvmCached(test.foundLvms)
			if test.expectedError != "" {
				assert.NotNil(t, err)
				assert.Equal(t, test.expectedError, err.Error())
			} else {
				assert.Nil(t, err)
			}
			assert.Equal(t, test.activationLvm, activated)
		})
	}
	runShellCmd = oldCmd
}

var cephVolumeNode1 = map[string][]OsdVolumeInfo{
	"20": {
		{
			Devices: []string{"/dev/vde"},
			LvPath:  "/dev/ceph-21312wds-sdfv-vs3f-scv3-sdfdsg23edaa/osd-block-vbsgs3a3-sdcv-casq-sd11-asd12dasczsf",
			Path:    "/dev/ceph-21312wds-sdfv-vs3f-scv3-sdfdsg23edaa/osd-block-vbsgs3a3-sdcv-casq-sd11-asd12dasczsf",
			Type:    "block",
			Tags: OsdVolumeTags{
				BlockDevice: "/dev/ceph-21312wds-sdfv-vs3f-scv3-sdfdsg23edaa/osd-block-vbsgs3a3-sdcv-casq-sd11-asd12dasczsf",
				ClusterFSID: "8668f062-3faa-358a-85f3-f80fe6c1e306",
				DBDevice:    "/dev/ceph-metadata/part-1",
				OsdFSID:     "vbsgs3a3-sdcv-casq-sd11-asd12dasczsf",
			},
		},
		{
			Devices: []string{"/dev/vdd"},
			LvPath:  "/dev/ceph-metadata/part-1",
			Path:    "/dev/ceph-metadata/part-1",
			Tags: OsdVolumeTags{
				BlockDevice: "/dev/ceph-21312wds-sdfv-vs3f-scv3-sdfdsg23edaa/osd-block-vbsgs3a3-sdcv-casq-sd11-asd12dasczsf",
				ClusterFSID: "8668f062-3faa-358a-85f3-f80fe6c1e306",
				DBDevice:    "/dev/ceph-metadata/part-1",
				OsdFSID:     "vbsgs3a3-sdcv-casq-sd11-asd12dasczsf",
			},
			Type: "db",
		},
	},
	"25": {
		{
			Devices: []string{"/dev/vdf"},
			LvPath:  "/dev/ceph-2efce189-afb7-452f-bd32-c73b5017a0da/osd-block-d49fd9bf-d2dd-4c3d-824d-87f3f17ea44a",
			Path:    "/dev/ceph-2efce189-afb7-452f-bd32-c73b5017a0da/osd-block-d49fd9bf-d2dd-4c3d-824d-87f3f17ea44a",
			Tags: OsdVolumeTags{
				BlockDevice: "/dev/ceph-2efce189-afb7-452f-bd32-c73b5017a0da/osd-block-d49fd9bf-d2dd-4c3d-824d-87f3f17ea44a",
				ClusterFSID: "8668f062-3faa-358a-85f3-f80fe6c1e306",
				DBDevice:    "/dev/ceph-metadata/part-2",
				OsdFSID:     "d49fd9bf-d2dd-4c3d-824d-87f3f17ea44a",
			},
			Type: "block",
		},
		{
			Devices: []string{"/dev/vdd"},
			LvPath:  "/dev/ceph-metadata/part-2",
			Path:    "/dev/ceph-metadata/part-2",
			Tags: OsdVolumeTags{
				BlockDevice: "/dev/ceph-2efce189-afb7-452f-bd32-c73b5017a0da/osd-block-d49fd9bf-d2dd-4c3d-824d-87f3f17ea44a",
				ClusterFSID: "8668f062-3faa-358a-85f3-f80fe6c1e306",
				DBDevice:    "/dev/ceph-metadata/part-2",
				OsdFSID:     "d49fd9bf-d2dd-4c3d-824d-87f3f17ea44a",
			},
			Type: "db",
		},
	},
	"30": {
		{
			Devices: []string{"/dev/vdb"},
			LvPath:  "/dev/ceph-992bbd78-3d8e-4cc3-93dc-eae387309364/osd-block-f4edb5cd-fb1e-4620-9419-3f9a4fcecba5",
			Path:    "/dev/ceph-992bbd78-3d8e-4cc3-93dc-eae387309364/osd-block-f4edb5cd-fb1e-4620-9419-3f9a4fcecba5",
			Tags: OsdVolumeTags{
				BlockDevice: "/dev/ceph-6e6365e0-b9ae-478b-a43c-644074506aae/osd-block-635a6fd8-dcad-4601-84a6-2150f2eef8c8",
				ClusterFSID: "8668f062-3faa-358a-85f3-f80fe6c1e306",
				DBDevice:    "/dev/vda14",
				OsdFSID:     "f4edb5cd-fb1e-4620-9419-3f9a4fcecba5",
			},
			Type: "block",
		},
		{
			Path: "/dev/vda14",
			Type: "db",
		},
	},
}

func TestCheckDisks(t *testing.T) {
	newDaemon := diskDaemon{
		data: initDaemonData(),
	}

	tests := []struct {
		name               string
		lsblkOutput        string
		udevadmOutput      map[string]string
		cephVolumeOutput   string
		lvmLvsOutput       string
		disksChanged       bool
		expectedDiskReport *lcmcommon.DiskDaemonDisksReport
		expectedCephVolume map[string][]OsdVolumeInfo
		expectedCachedLvms map[string][]string
		expectedError      string
		stateChanged       bool
	}{
		{
			name:               "check failed with lsblk error",
			lsblkOutput:        "{}",
			expectedCachedLvms: map[string][]string{},
			expectedCephVolume: map[string][]OsdVolumeInfo{},
			expectedError:      "daemon is failed to prepare block device report: no blockdevices found for 'lsblk' output",
		},
		{
			name:               "check report created - no ceph volume report",
			lsblkOutput:        lcmdiskdaemoninput.LsblkReportFromNode1,
			udevadmOutput:      lcmdiskdaemoninput.UdevadmReportFromNode1,
			lvmLvsOutput:       "{}",
			cephVolumeOutput:   "{}",
			expectedDiskReport: lcmdiskdaemoninput.DiskInfoReportLsblkFromNode1,
			expectedCachedLvms: lcmdiskdaemoninput.FoundLvmsFromNode1,
			expectedCephVolume: map[string][]OsdVolumeInfo{},
			stateChanged:       true,
		},
		{
			name:               "check failed to update with lvs error",
			lsblkOutput:        lcmdiskdaemoninput.LsblkReportFromNode1,
			udevadmOutput:      lcmdiskdaemoninput.UdevadmReportFromNode1,
			disksChanged:       true,
			expectedDiskReport: &lcmcommon.DiskDaemonDisksReport{},
			expectedCachedLvms: map[string][]string{},
			expectedCephVolume: map[string][]OsdVolumeInfo{},
			expectedError:      "daemon is failed to check cached logical volumes: unable to unmarshal lvm lvs output: unexpected end of JSON input",
		},
		{
			name:               "check failed to update with ceph-volume error",
			lsblkOutput:        lcmdiskdaemoninput.LsblkReportFromNode1,
			udevadmOutput:      lcmdiskdaemoninput.UdevadmReportFromNode1,
			lvmLvsOutput:       lcmdiskdaemoninput.LvmLvsReportFromNode1,
			expectedDiskReport: lcmdiskdaemoninput.DiskInfoReportLsblkFromNode1,
			expectedCachedLvms: lcmdiskdaemoninput.FoundLvmsFromNode1,
			expectedCephVolume: map[string][]OsdVolumeInfo{},
			expectedError:      "daemon is failed to prepare ceph volumes info report: failed to parse output for command 'ceph-volume lvm list --format json': unexpected end of JSON input",
		},
		{
			name:               "check has update full info",
			lsblkOutput:        lcmdiskdaemoninput.LsblkReportFromNode1,
			udevadmOutput:      lcmdiskdaemoninput.UdevadmReportFromNode1,
			cephVolumeOutput:   lcmdiskdaemoninput.CephVolumeLvmReportFromNode1,
			lvmLvsOutput:       lcmdiskdaemoninput.LvmLvsReportFromNode1,
			expectedDiskReport: lcmdiskdaemoninput.DiskInfoReportCephVolumeFromNode1,
			expectedCachedLvms: lcmdiskdaemoninput.FoundLvmsFromNode1,
			expectedCephVolume: cephVolumeNode1,
			stateChanged:       true,
		},
		{
			name:               "check has no changes",
			lsblkOutput:        lcmdiskdaemoninput.LsblkReportFromNode1,
			udevadmOutput:      lcmdiskdaemoninput.UdevadmReportFromNode1,
			cephVolumeOutput:   lcmdiskdaemoninput.CephVolumeLvmReportFromNode1,
			lvmLvsOutput:       lcmdiskdaemoninput.LvmLvsReportFromNode1,
			expectedDiskReport: lcmdiskdaemoninput.DiskInfoReportCephVolumeFromNode1,
			expectedCachedLvms: lcmdiskdaemoninput.FoundLvmsFromNode1,
			expectedCephVolume: cephVolumeNode1,
		},
		{
			name:               "check has ceph volumes changes",
			lsblkOutput:        lcmdiskdaemoninput.LsblkReportFromNode1,
			udevadmOutput:      lcmdiskdaemoninput.UdevadmReportFromNode1,
			cephVolumeOutput:   "{}",
			expectedDiskReport: lcmdiskdaemoninput.DiskInfoReportLsblkFromNode1,
			expectedCachedLvms: lcmdiskdaemoninput.FoundLvmsFromNode1,
			expectedCephVolume: map[string][]OsdVolumeInfo{},
			stateChanged:       true,
		},
	}

	oldCmd := runShellCmd
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			runShellCmd = func(command string) (string, string, error) {
				if command == "lsblk -J -p -O" {
					return test.lsblkOutput, "", nil
				} else if strings.HasPrefix(command, "udevadm info -r --query=symlink") {
					cmdargs := strings.Split(command, " ")
					dev := cmdargs[len(cmdargs)-1]
					if output, present := test.udevadmOutput[dev]; present {
						return output, "", nil
					}
					return "", "failed to get udevadm for device", errors.Errorf("device '%s' not found", dev)
				} else if command == "lvm lvs --reportformat json -o lv_dm_path" {
					return test.lvmLvsOutput, "", nil
				} else if strings.HasPrefix(command, "pvscan --cache") {
					return "", "", nil
				} else if command == "ceph-volume lvm list --format json" {
					return test.cephVolumeOutput, "", nil
				}
				return "", "", errors.New("unknown command")
			}

			if test.disksChanged {
				newDaemon.data.runtime.knownLvms = map[string][]string{}
				newDaemon.data.runtime.disksReport = &lcmcommon.DiskDaemonDisksReport{}
			}

			changed, err := newDaemon.checkDisks()
			assert.Equal(t, test.stateChanged, changed)
			if test.expectedError != "" {
				assert.Equal(t, test.expectedError, err.Error())
			}
			assert.Equal(t, test.expectedDiskReport, newDaemon.data.runtime.disksReport)
			assert.Equal(t, test.expectedCephVolume, newDaemon.data.runtime.volumesReport)
			assert.Equal(t, test.expectedCachedLvms, newDaemon.data.runtime.knownLvms)
		})
	}
	runShellCmd = oldCmd
}
