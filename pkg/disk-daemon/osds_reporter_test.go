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

	lcmcommon "github.com/Mirantis/pelagia/pkg/common"
	unitinputs "github.com/Mirantis/pelagia/test/unit/inputs"
	lcmdiskdaemoninput "github.com/Mirantis/pelagia/test/unit/inputs/disk-daemon"
)

func TestPrepareOsdReport(t *testing.T) {
	newDaemon := diskDaemon{
		data: initDaemonData(),
	}

	cephVolumeNode1Lost := map[string][]OsdVolumeInfo{
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

	cephVolumeNode2 := map[string][]OsdVolumeInfo{
		"0": {
			{
				Devices: []string{"/dev/vdb"},
				LvPath:  "/dev/ceph-cf7c8b53-27c7-4cfc-94de-6ad4c7d9f92d/osd-block-af39b794-e1c6-41c0-8997-d6b6c631b8f2",
				Path:    "/dev/ceph-cf7c8b53-27c7-4cfc-94de-6ad4c7d9f92d/osd-block-af39b794-e1c6-41c0-8997-d6b6c631b8f2",
				Tags: OsdVolumeTags{
					ClusterFSID: "8668f062-3faa-358a-85f3-f80fe6c1e306",
					BlockDevice: "/dev/ceph-cf7c8b53-27c7-4cfc-94de-6ad4c7d9f92d/osd-block-af39b794-e1c6-41c0-8997-d6b6c631b8f2",
					OsdFSID:     "69481cd1-38b1-42fd-ac07-06bf4d7c0e19",
				},
				Type: "block",
			},
			{
				Devices: []string{"/dev/vdc"},
				LvPath:  "/dev/ceph-c5628abe-ae41-4c3d-bdc6-ef86c54bf78c/osd-block-69481cd1-38b1-42fd-ac07-06bf4d7c0e19",
				Path:    "/dev/ceph-c5628abe-ae41-4c3d-bdc6-ef86c54bf78c/osd-block-69481cd1-38b1-42fd-ac07-06bf4d7c0e19",
				Tags: OsdVolumeTags{
					ClusterFSID: "8668f062-0lsk-358a-1gt4-f80fe6c1e306",
					OsdFSID:     "06bf4d7c-9603-41a4-b250-284ecf3ecb2f",
					BlockDevice: "/dev/ceph-c5628abe-ae41-4c3d-bdc6-ef86c54bf78c/osd-block-69481cd1-38b1-42fd-ac07-06bf4d7c0e19",
				},
				Type: "block",
			},
		},
		"2": {
			{
				Devices: []string{"/dev/vde"},
				LvPath:  "/dev/ceph-0e03d5c6-d0e9-4f04-b9af-38d15e14369f/osd-block-61869d90-2c45-4f02-b7c3-96955f41e2ca",
				Path:    "/dev/ceph-0e03d5c6-d0e9-4f04-b9af-38d15e14369f/osd-block-61869d90-2c45-4f02-b7c3-96955f41e2ca",
				Tags: OsdVolumeTags{
					ClusterFSID: "8668f062-3faa-358a-85f3-f80fe6c1e306",
					OsdFSID:     "61869d90-2c45-4f02-b7c3-96955f41e2ca",
					BlockDevice: "/dev/ceph-0e03d5c6-d0e9-4f04-b9af-38d15e14369f/osd-block-61869d90-2c45-4f02-b7c3-96955f41e2ca",
				},
				Type: "block",
			},
		},
		"4": {
			{
				Devices: []string{"/dev/vdd"},
				LvPath:  "/dev/ceph-dada9f25-41b4-4c26-9a20-448ac01e1d06/osd-block-ad76cf53-5cb5-48fe-a39a-343734f5ccde",
				Path:    "/dev/ceph-dada9f25-41b4-4c26-9a20-448ac01e1d06/osd-block-ad76cf53-5cb5-48fe-a39a-343734f5ccde",
				Tags: OsdVolumeTags{
					ClusterFSID: "8668f062-3faa-358a-85f3-f80fe6c1e306",
					OsdFSID:     "ad76cf53-5cb5-48fe-a39a-343734f5ccde",
					BlockDevice: "/dev/ceph-dada9f25-41b4-4c26-9a20-448ac01e1d06/osd-block-ad76cf53-5cb5-48fe-a39a-343734f5ccde",
				},
				Type: "block",
			},
		},
		"5": {
			{
				Devices: []string{"/dev/vdd"},
				LvPath:  "/dev/ceph-dada9f25-41b4-4c26-9a20-448ac01e1d06/osd-block-7d09cceb-4de0-478e-9d8d-bd09cb0c904e",
				Path:    "/dev/ceph-dada9f25-41b4-4c26-9a20-448ac01e1d06/osd-block-7d09cceb-4de0-478e-9d8d-bd09cb0c904e",
				Tags: OsdVolumeTags{
					ClusterFSID: "8668f062-3faa-358a-85f3-f80fe6c1e306",
					OsdFSID:     "af39b794-e1c6-41c0-8997-d6b6c631b8f2",
					BlockDevice: "/dev/ceph-dada9f25-41b4-4c26-9a20-448ac01e1d06/osd-block-7d09cceb-4de0-478e-9d8d-bd09cb0c904e",
				},
				Type: "block",
			},
		},
	}

	cephVolumeNode1Issues := map[string][]OsdVolumeInfo{
		"0": {
			{
				Devices: []string{"/dev/vdb", "/dev/vdx"},
				LvPath:  "/dev/ceph-cf7c8b53-27c7-4cfc-94de-6ad4c7d9f92d/osd-block-af39b794-e1c6-41c0-8997-d6b6c631b8f2",
				Path:    "/dev/ceph-cf7c8b53-27c7-4cfc-94de-6ad4c7d9f92d/osd-block-af39b794-e1c6-41c0-8997-d6b6c631b8f2",
				Tags: OsdVolumeTags{
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
		volumesReport      map[string][]OsdVolumeInfo
		expectedOsdsReport *lcmcommon.DiskDaemonOsdsReport
		expectedIssues     []string
	}{
		{
			name:               "regular osd report",
			disksReport:        lcmdiskdaemoninput.DiskInfoReportCephVolumeFromNode1,
			volumesReport:      cephVolumeNode1,
			expectedOsdsReport: unitinputs.DiskDaemonReportOkNode1WithParted.OsdsReport,
			expectedIssues:     []string{},
		},
		{
			name:               "osd report - some devices with osd lost",
			disksReport:        lcmdiskdaemoninput.DiskInfoReportCephVolumeSomeOsdLostFromNode1,
			volumesReport:      cephVolumeNode1Lost,
			expectedOsdsReport: unitinputs.DiskDaemonReportOkNode1SomeDevLost.OsdsReport,
			expectedIssues:     []string{},
		},
		{
			name:               "osd report with stray, but it is unknown in report",
			disksReport:        lcmdiskdaemoninput.DiskInfoReportCephVolumeFromNode2,
			volumesReport:      cephVolumeNode2,
			expectedOsdsReport: unitinputs.DiskDaemonNodeReportWithStrayOkNode2(true).OsdsReport,
			expectedIssues:     []string{},
		},
		{
			name:          "osd report has issues",
			disksReport:   lcmdiskdaemoninput.DiskInfoReportCephVolumeFromNode1,
			volumesReport: cephVolumeNode1Issues,
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
				"found physical osd block partition '/dev/vda14' for osd '2', which is not supported",
				"multidisk setup detected for osd '0', partition '/dev/ceph-cf7c8b53-27c7-4cfc-94de-6ad4c7d9f92d/osd-block-af39b794-e1c6-41c0-8997-d6b6c631b8f2', which is not supported",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			newDaemon.data.runtime.disksReport = test.disksReport
			newDaemon.data.runtime.volumesReport = test.volumesReport

			issues := newDaemon.checkOsds()
			assert.Equal(t, test.expectedIssues, issues)
			assert.Equal(t, test.expectedOsdsReport, newDaemon.data.runtime.osdsReport)
		})
	}
}
