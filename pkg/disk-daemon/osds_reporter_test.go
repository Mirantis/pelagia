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
	"testing"

	"github.com/stretchr/testify/assert"

	lcmcommon "github.com/Mirantis/pelagia/v3/pkg/common"
	unitinputs "github.com/Mirantis/pelagia/v3/test/unit/inputs"
	lcmdiskdaemoninput "github.com/Mirantis/pelagia/v3/test/unit/inputs/disk-daemon"
)

func TestPrepareOsdReport(t *testing.T) {
	newDaemon := diskDaemon{
		nodeName: "test-node",
		data:     initDaemonData(),
	}

	cephVolumeLvmNode1Lost := map[string][]OsdVolumeLvmInfo{
		"20": {
			{
				Devices: []string{"/dev/vde"},
				LvPath:  "/dev/ceph-21312wds-sdfv-vs3f-scv3-sdfdsg23edaa/osd-block-vbsgs3a3-sdcv-casq-sd11-asd12dasczsf",
				Path:    "/dev/ceph-21312wds-sdfv-vs3f-scv3-sdfdsg23edaa/osd-block-vbsgs3a3-sdcv-casq-sd11-asd12dasczsf",
				Type:    "block",
				Tags: OsdVolumeLvmTags{
					BlockDevice: "/dev/ceph-21312wds-sdfv-vs3f-scv3-sdfdsg23edaa/osd-block-vbsgs3a3-sdcv-casq-sd11-asd12dasczsf",
					ClusterFSID: "8668f062-3faa-358a-85f3-f80fe6c1e306",
					DBDevice:    "/dev/ceph-metadata/part-1",
					OsdFSID:     "vbsgs3a3-sdcv-casq-sd11-asd12dasczsf",
				},
			},
		},
		"25": {
			{
				Devices: []string{"/dev/vdf"},
				LvPath:  "/dev/ceph-2efce189-afb7-452f-bd32-c73b5017a0da/osd-block-d49fd9bf-d2dd-4c3d-824d-87f3f17ea44a",
				Path:    "/dev/ceph-2efce189-afb7-452f-bd32-c73b5017a0da/osd-block-d49fd9bf-d2dd-4c3d-824d-87f3f17ea44a",
				Tags: OsdVolumeLvmTags{
					BlockDevice: "/dev/ceph-2efce189-afb7-452f-bd32-c73b5017a0da/osd-block-d49fd9bf-d2dd-4c3d-824d-87f3f17ea44a",
					ClusterFSID: "8668f062-3faa-358a-85f3-f80fe6c1e306",
					DBDevice:    "/dev/ceph-metadata/part-2",
					OsdFSID:     "d49fd9bf-d2dd-4c3d-824d-87f3f17ea44a",
				},
				Type: "block",
			},
		},
		"30": {
			{
				Devices: []string{"/dev/vdb"},
				LvPath:  "/dev/ceph-992bbd78-3d8e-4cc3-93dc-eae387309364/osd-block-f4edb5cd-fb1e-4620-9419-3f9a4fcecba5",
				Path:    "/dev/ceph-992bbd78-3d8e-4cc3-93dc-eae387309364/osd-block-f4edb5cd-fb1e-4620-9419-3f9a4fcecba5",
				Tags: OsdVolumeLvmTags{
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

	cephVolumeLvmNode1Issues := map[string][]OsdVolumeLvmInfo{
		"0": {
			{
				Devices: []string{"/dev/vdb", "/dev/vdx"},
				LvPath:  "/dev/ceph-cf7c8b53-27c7-4cfc-94de-6ad4c7d9f92d/osd-block-af39b794-e1c6-41c0-8997-d6b6c631b8f2",
				Path:    "/dev/ceph-cf7c8b53-27c7-4cfc-94de-6ad4c7d9f92d/osd-block-af39b794-e1c6-41c0-8997-d6b6c631b8f2",
				Tags: OsdVolumeLvmTags{
					ClusterFSID: "8668f062-3faa-358a-85f3-f80fe6c1e306",
					BlockDevice: "/dev/ceph-cf7c8b53-27c7-4cfc-94de-6ad4c7d9f92d/osd-block-af39b794-e1c6-41c0-8997-d6b6c631b8f2",
					OsdFSID:     "69481cd1-38b1-42fd-ac07-06bf4d7c0e19",
				},
				Type: "block",
			},
		},
		"2": {
			{
				Path: "/dev/vda14",
				Type: "block",
			},
		},
		"4": {
			{
				Path: "/dev/md127",
				Type: "db",
			},
		},
	}

	tests := []struct {
		name               string
		disksReport        *lcmcommon.DiskDaemonDisksReport
		volumesLvmReport   map[string][]OsdVolumeLvmInfo
		volumesRawReport   map[string]OsdVolumeRawInfo
		expectedOsdsReport *lcmcommon.DiskDaemonOsdsReport
		expectedIssues     []string
	}{
		{
			name:               "regular osd report",
			disksReport:        lcmdiskdaemoninput.DiskInfoReportCephVolumeFromNode1,
			volumesLvmReport:   cephVolumeLvmNode1,
			expectedOsdsReport: unitinputs.DiskDaemonReportOkNode1WithParted.OsdsReport,
			expectedIssues:     []string{},
		},
		{
			name:               "osd report - some devices with osd lost",
			disksReport:        lcmdiskdaemoninput.DiskInfoReportCephVolumeSomeOsdLostFromNode1,
			volumesLvmReport:   cephVolumeLvmNode1Lost,
			expectedOsdsReport: unitinputs.DiskDaemonReportOkNode1SomeDevLost.OsdsReport,
			expectedIssues:     []string{},
		},
		{
			name:               "osd report with stray, but it is unknown in report",
			disksReport:        lcmdiskdaemoninput.DiskInfoReportCephVolumeFromNode2,
			volumesLvmReport:   cephVolumeLvmNode2,
			volumesRawReport:   cephVolumeRawNode2,
			expectedOsdsReport: unitinputs.DiskDaemonNodeReportWithStrayOkNode2(true).OsdsReport,
			expectedIssues:     []string{},
		},
		{
			name:             "osd report has issues",
			disksReport:      lcmdiskdaemoninput.DiskInfoReportCephVolumeFromNode1,
			volumesLvmReport: cephVolumeLvmNode1Issues,
			volumesRawReport: map[string]OsdVolumeRawInfo{
				"3e847863-45b6-41c3-a467-ba5f88094f9c": {
					ClusterFSID: "8668f062-3faa-358a-85f3-f80fe6c1e306",
					Device:      "/dev/vda14",
					OsdID:       9,
					OsdFSID:     "3e847863-45b6-41c3-a467-ba5f88094f9c",
				},
			},
			expectedOsdsReport: &lcmcommon.DiskDaemonOsdsReport{
				Warnings: []string{},
				Osds: map[string][]lcmcommon.OsdDaemonInfo{
					"0": {},
					"2": {},
					"4": {},
				},
			},
			expectedIssues: []string{
				"for osd '4', partition '/dev/md127' detected multidisk setup, which is not supported: /dev/vda15,/dev/vdh1",
				"for osd '9', raw device '/dev/vda14' is not a full disk (likely physical partition), only full disk raw osds are supported",
				"found physical osd block partition '/dev/vda14' for osd '2', which is not supported",
				"multidisk setup detected for osd '0', partition '/dev/ceph-cf7c8b53-27c7-4cfc-94de-6ad4c7d9f92d/osd-block-af39b794-e1c6-41c0-8997-d6b6c631b8f2', which is not supported",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			newDaemon.data.runtime.disksReport = test.disksReport
			newDaemon.data.runtime.volumesLvmReport = test.volumesLvmReport
			newDaemon.data.runtime.volumesRawReport = test.volumesRawReport

			issues := newDaemon.checkOsds()
			assert.Equal(t, test.expectedIssues, issues)
			assert.Equal(t, test.expectedOsdsReport, newDaemon.data.runtime.osdsReport)
		})
	}
}
