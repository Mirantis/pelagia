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

package input

import (
	"encoding/json"

	lcmcommon "github.com/Mirantis/pelagia/pkg/common"
	lcmdiskdaemoninput "github.com/Mirantis/pelagia/test/unit/inputs/disk-daemon"
)

var CephDiskDaemonDiskReportStringNode1 = GetDiskDaemonReportToString(&DiskDaemonReportOkNode1)
var CephDiskDaemonDiskReportStringNode2 = GetDiskDaemonReportToString(DiskDaemonNodeReportWithStrayOkNode2(true))

func GetDiskDaemonReportToString(report *lcmcommon.DiskDaemonReport) string {
	strReport, _ := json.Marshal(report)
	return string(strReport)
}

var DiskDaemonReportOkNode1 = lcmcommon.DiskDaemonReport{
	State:       lcmcommon.DiskDaemonStateOk,
	DisksReport: lcmdiskdaemoninput.DiskInfoReportCephVolumeFromNode1,
	OsdsReport: &lcmcommon.DiskDaemonOsdsReport{
		Warnings: []string{
			"found physical osd db partition '/dev/vda14' for osd '30'",
		},
		Osds: lcmdiskdaemoninput.OsdDevicesInfoNode1,
	},
}

var DiskDaemonReportOkNode1SomeDevLost = lcmcommon.DiskDaemonReport{
	State:       lcmcommon.DiskDaemonStateOk,
	DisksReport: lcmdiskdaemoninput.DiskInfoReportCephVolumeSomeOsdLostFromNode1,
	OsdsReport: &lcmcommon.DiskDaemonOsdsReport{
		Warnings: []string{
			"found physical osd db partition '/dev/vda14' for osd '30'",
		},
		Osds: map[string][]lcmcommon.OsdDaemonInfo{
			"20": lcmdiskdaemoninput.OsdDevicesSomeLostInfoNode1["20"],
			"25": lcmdiskdaemoninput.OsdDevicesSomeLostInfoNode1["25"],
			"30": lcmdiskdaemoninput.OsdDevicesInfoNode1["30"],
		},
	},
}

var DiskDaemonReportOkNode2 = lcmcommon.DiskDaemonReport{
	State:       lcmcommon.DiskDaemonStateOk,
	DisksReport: lcmdiskdaemoninput.DiskInfoReportCephVolumeFromNode2,
	OsdsReport: &lcmcommon.DiskDaemonOsdsReport{
		Warnings: []string{},
		Osds: map[string][]lcmcommon.OsdDaemonInfo{
			"0": lcmdiskdaemoninput.OsdDevicesInfoNode2["0"],
			"4": lcmdiskdaemoninput.OsdDevicesInfoNode2["4"],
			"5": lcmdiskdaemoninput.OsdDevicesInfoNode2["5"],
		},
	},
}

var DiskDaemonReportOkNode2SomeDevLost = lcmcommon.DiskDaemonReport{
	State: lcmcommon.DiskDaemonStateOk,
	OsdsReport: &lcmcommon.DiskDaemonOsdsReport{
		Warnings: []string{},
		Osds: map[string][]lcmcommon.OsdDaemonInfo{
			"0": lcmdiskdaemoninput.OsdDevicesInfoNode2["0"],
		},
	},
}

func DiskDaemonNodeReportWithStrayOkNode2(inCrush bool) *lcmcommon.DiskDaemonReport {
	report := &lcmcommon.DiskDaemonReport{
		State:       lcmcommon.DiskDaemonStateOk,
		DisksReport: lcmdiskdaemoninput.DiskInfoReportCephVolumeFromNode2,
		OsdsReport: &lcmcommon.DiskDaemonOsdsReport{
			Warnings: []string{},
			Osds: map[string][]lcmcommon.OsdDaemonInfo{
				"0": append(lcmdiskdaemoninput.OsdDevicesInfoNode2["0"], lcmdiskdaemoninput.OsdDevicesInfoNode2["0-stray"]...),
				"4": lcmdiskdaemoninput.OsdDevicesInfoNode2["4"],
				"5": lcmdiskdaemoninput.OsdDevicesInfoNode2["5"],
			},
		},
	}
	if inCrush {
		report.OsdsReport.Osds["2"] = lcmdiskdaemoninput.OsdDevicesInfoNode2["2"]
	}
	return report
}
