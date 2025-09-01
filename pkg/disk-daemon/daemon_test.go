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
	unitinputs "github.com/Mirantis/pelagia/test/unit/inputs"
	lcmdiskdaemoninput "github.com/Mirantis/pelagia/test/unit/inputs/disk-daemon"
)

func TestPrepareReport(t *testing.T) {
	newDaemon := diskDaemon{
		data: initDaemonData(),
	}
	tests := []struct {
		name             string
		lsblkOutput      string
		udevadmOutput    map[string]string
		cephVolumeOutput string
		lvmLvsOutput     string
		expectedReport   lcmcommon.DiskDaemonReport
	}{
		{
			name:             "daemon report prepared",
			lsblkOutput:      lcmdiskdaemoninput.LsblkReportFromNode1,
			udevadmOutput:    lcmdiskdaemoninput.UdevadmReportFromNode1,
			cephVolumeOutput: lcmdiskdaemoninput.CephVolumeLvmReportFromNode1,
			lvmLvsOutput:     lcmdiskdaemoninput.LvmLvsReportFromNode1,
			expectedReport:   unitinputs.DiskDaemonReportOkNode1,
		},
		{
			name:        "daemon report failed to prepare disk report",
			lsblkOutput: "{}",
			expectedReport: lcmcommon.DiskDaemonReport{
				State: lcmcommon.DiskDaemonStateFailed,
				Issues: []string{
					"daemon is failed to prepare block device report: no blockdevices found for 'lsblk' output",
				},
			},
		},
		{
			name:             "daemon report not changed since last success",
			lsblkOutput:      lcmdiskdaemoninput.LsblkReportFromNode1,
			udevadmOutput:    lcmdiskdaemoninput.UdevadmReportFromNode1,
			cephVolumeOutput: lcmdiskdaemoninput.CephVolumeLvmReportFromNode1,
			lvmLvsOutput:     lcmdiskdaemoninput.LvmLvsReportFromNode1,
			expectedReport:   unitinputs.DiskDaemonReportOkNode1,
		},
		{
			name:          "daemon report failed for osd report",
			lsblkOutput:   lcmdiskdaemoninput.LsblkReportFromNode1,
			udevadmOutput: lcmdiskdaemoninput.UdevadmReportFromNode1,
			cephVolumeOutput: `{
    "2": [
        {
            "path": "/dev/vda14",
            "tags": {
                "PARTUUID": "40dba738-2c45-4236-a681-75198bc111ae"
            },
            "type": "block"
        }
    ]
}`,
			lvmLvsOutput: lcmdiskdaemoninput.LvmLvsReportFromNode1,
			expectedReport: lcmcommon.DiskDaemonReport{
				State: lcmcommon.DiskDaemonStateFailed,
				Issues: []string{
					"found physical osd block partition '/dev/vda14' for osd '2', which is not supported",
				},
				DisksReport: &lcmcommon.DiskDaemonDisksReport{
					BlockInfo: lcmdiskdaemoninput.DiskInfoReportLsblkFromNode1.BlockInfo,
					Aliases:   lcmdiskdaemoninput.DiskInfoReportLsblkFromNode1.Aliases,
					DiskToOsd: map[string][]string{
						"/dev/vda": {"2"},
					},
				},
				OsdsReport: &lcmcommon.DiskDaemonOsdsReport{
					Warnings: []string{},
					Osds: map[string][]lcmcommon.OsdDaemonInfo{
						"2": {},
					},
				},
			},
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

			newDaemon.prepareReport()
			assert.Equal(t, test.expectedReport, newDaemon.data.report.node)
		})
	}
	runShellCmd = oldCmd
}
