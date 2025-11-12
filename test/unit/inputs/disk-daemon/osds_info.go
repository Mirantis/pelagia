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
	lcmcommon "github.com/Mirantis/pelagia/pkg/common"
)

var OsdDevicesInfoNode1 = map[string][]lcmcommon.OsdDaemonInfo{
	"20": {
		{
			OsdUUID:     "vbsgs3a3-sdcv-casq-sd11-asd12dasczsf",
			ClusterFSID: "8668f062-3faa-358a-85f3-f80fe6c1e306",
			Devices: []lcmcommon.OsdDevice{
				{
					Name:       "/dev/vde",
					DeviceID:   "2926ff77-7491-4447-a",
					Rotational: true,
					DeviceSymlinks: []string{
						"/dev/disk/by-id/lvm-pv-uuid-nzJOk1-kLTM-ErxQ-0N4c-DpDU-0zhE-Q9hRJP",
						"/dev/disk/by-id/virtio-2926ff77-7491-4447-a",
						"/dev/disk/by-path/pci-0000:00:0f.0",
						"/dev/disk/by-path/virtio-pci-0000:00:0f.0",
					},
					RelatedPartition: "/dev/ceph-21312wds-sdfv-vs3f-scv3-sdfdsg23edaa/osd-block-vbsgs3a3-sdcv-casq-sd11-asd12dasczsf",
				},
				{
					Name:       "/dev/vdd",
					DeviceID:   "e8d89e2f-ffc6-4988-9",
					Rotational: true,
					DeviceSymlinks: []string{
						"/dev/disk/by-id/virtio-e8d89e2f-ffc6-4988-9",
						"/dev/disk/by-path/pci-0000:00:0e.0",
						"/dev/disk/by-path/virtio-pci-0000:00:0e.0",
					},
					RelatedPartition: "/dev/ceph-metadata/part-1",
				},
			},
			Partitions: []lcmcommon.OsdPartition{
				{
					Partition: "/dev/ceph-21312wds-sdfv-vs3f-scv3-sdfdsg23edaa/osd-block-vbsgs3a3-sdcv-casq-sd11-asd12dasczsf",
					PartitionSymlinks: []string{
						"/dev/disk/by-id/dm-name-ceph--21312wds--sdfv--vs3f--scv3--sdfdsg23edaa-osd--block--vbsgs3a3--sdcv--casq--sd11--asd12dasczsf",
						"/dev/disk/by-id/dm-uuid-LVM-oPXPcruZ1AK9dkZOsPR9ZW7PzVb9xtFrOnhN24VqDzKIOPBZLd60UQpS6PpCEzQs",
						"/dev/dm-5",
						"/dev/mapper/ceph--21312wds--sdfv--vs3f--scv3--sdfdsg23edaa-osd--block--vbsgs3a3--sdcv--casq--sd11--asd12dasczsf",
					},
					Type:   "block",
					Exists: true,
					Lvm:    true,
				},
				{
					Partition: "/dev/ceph-metadata/part-1",
					PartitionSymlinks: []string{
						"/dev/disk/by-id/dm-name-ceph--metadata-part--1",
						"/dev/disk/by-id/dm-uuid-LVM-4NjWWqNazXbV26cmzMqOasZgbTwEPpZaUW1YZSAnvx7CLqXUAIZ5UKlcZx8w8lWo",
						"/dev/dm-2",
						"/dev/mapper/ceph--metadata-part--1",
					},
					Type:   "db",
					Lvm:    true,
					Exists: true,
				},
			},
		},
	},
	"25": {
		{
			OsdUUID:     "d49fd9bf-d2dd-4c3d-824d-87f3f17ea44a",
			ClusterFSID: "8668f062-3faa-358a-85f3-f80fe6c1e306",
			Devices: []lcmcommon.OsdDevice{
				{
					Name:       "/dev/vdf",
					DeviceID:   "b7ea1c8c-89b8-4354-8",
					Rotational: true,
					DeviceSymlinks: []string{
						"/dev/disk/by-id/lvm-pv-uuid-fZ7Efo-X0nc-lAR3-lzik-MjMT-0rml-lZNf7b",
						"/dev/disk/by-id/virtio-b7ea1c8c-89b8-4354-8",
						"/dev/disk/by-path/pci-0000:00:10.0",
						"/dev/disk/by-path/virtio-pci-0000:00:10.0",
					},
					RelatedPartition: "/dev/ceph-2efce189-afb7-452f-bd32-c73b5017a0da/osd-block-d49fd9bf-d2dd-4c3d-824d-87f3f17ea44a",
				},
				{
					Name:       "/dev/vdd",
					DeviceID:   "e8d89e2f-ffc6-4988-9",
					Rotational: true,
					DeviceSymlinks: []string{
						"/dev/disk/by-id/virtio-e8d89e2f-ffc6-4988-9",
						"/dev/disk/by-path/pci-0000:00:0e.0",
						"/dev/disk/by-path/virtio-pci-0000:00:0e.0",
					},
					RelatedPartition: "/dev/ceph-metadata/part-2",
				},
			},
			Partitions: []lcmcommon.OsdPartition{
				{
					Partition: "/dev/ceph-2efce189-afb7-452f-bd32-c73b5017a0da/osd-block-d49fd9bf-d2dd-4c3d-824d-87f3f17ea44a",
					PartitionSymlinks: []string{
						"/dev/disk/by-id/dm-name-ceph--2efce189--afb7--452f--bd32--c73b5017a0da-osd--block--d49fd9bf--d2dd--4c3d--824d--87f3f17ea44a",
						"/dev/disk/by-id/dm-uuid-LVM-LMoz5X0a3VV3TO2rMomDqfh24zt91NaCiZmlePb5dTd9cws2kHF6Q28W96aUWWgJ",
						"/dev/dm-4",
						"/dev/mapper/ceph--2efce189--afb7--452f--bd32--c73b5017a0da-osd--block--d49fd9bf--d2dd--4c3d--824d--87f3f17ea44a",
					},
					Type:   "block",
					Exists: true,
					Lvm:    true,
				},
				{
					Partition: "/dev/ceph-metadata/part-2",
					PartitionSymlinks: []string{
						"/dev/disk/by-id/dm-name-ceph--metadata-part--2",
						"/dev/disk/by-id/dm-uuid-LVM-4NjWWqNazXbV26cmzMqOasZgbTwEPpZaH1waxWke4fbXEDXucEbZNeB4ZDfBeUrW",
						"/dev/dm-3",
						"/dev/mapper/ceph--metadata-part--2",
					},
					Type:   "db",
					Exists: true,
					Lvm:    true,
				},
			},
		},
	},
	"30": {
		{
			OsdUUID:     "f4edb5cd-fb1e-4620-9419-3f9a4fcecba5",
			ClusterFSID: "8668f062-3faa-358a-85f3-f80fe6c1e306",
			Devices: []lcmcommon.OsdDevice{
				{
					Name:       "/dev/vdb",
					DeviceID:   "996ea59f-7f47-4fac-b",
					Rotational: true,
					DeviceSymlinks: []string{
						"/dev/disk/by-id/lvm-pv-uuid-yd92Oj-9hBf-2w2n-IEjf-nBJ1-2dMk-kBeMZI",
						"/dev/disk/by-id/virtio-996ea59f-7f47-4fac-b",
						"/dev/disk/by-path/pci-0000:00:0a.0",
						"/dev/disk/by-path/virtio-pci-0000:00:0a.0",
					},
					RelatedPartition: "/dev/ceph-992bbd78-3d8e-4cc3-93dc-eae387309364/osd-block-f4edb5cd-fb1e-4620-9419-3f9a4fcecba5",
				},
				{
					Name:       "/dev/vda",
					DeviceID:   "8dad5ae9-ddf7-40bf-8",
					Rotational: true,
					DeviceSymlinks: []string{
						"/dev/disk/by-id/virtio-8dad5ae9-ddf7-40bf-8",
						"/dev/disk/by-path/pci-0000:00:09.0",
						"/dev/disk/by-path/virtio-pci-0000:00:09.0",
					},
					RelatedPartition: "/dev/vda14",
				},
			},
			Partitions: []lcmcommon.OsdPartition{
				{
					Partition: "/dev/ceph-992bbd78-3d8e-4cc3-93dc-eae387309364/osd-block-f4edb5cd-fb1e-4620-9419-3f9a4fcecba5",
					PartitionSymlinks: []string{
						"/dev/disk/by-id/dm-name-ceph--992bbd78--3d8e--4cc3--93dc--eae387309364-osd--block--f4edb5cd--fb1e--4620--9419--3f9a4fcecba5",
						"/dev/disk/by-id/dm-uuid-LVM-VjASpFzahZwHYS2XN4EblEfLAfVwAImtnWhRvxcC38bhRLCw9S8sCCR7JvTuSbco",
						"/dev/dm-1",
						"/dev/mapper/ceph--992bbd78--3d8e--4cc3--93dc--eae387309364-osd--block--f4edb5cd--fb1e--4620--9419--3f9a4fcecba5",
					},
					Type:   "block",
					Exists: true,
					Lvm:    true,
				},
				{
					Partition: "/dev/vda14",
					PartitionSymlinks: []string{
						"/dev/disk/by-partuuid/40dba738-2c45-4236-a681-75198bc111ae",
						"/dev/disk/by-path/pci-0000:00:09.0-part14",
						"/dev/disk/by-path/virtio-pci-0000:00:09.0-part14",
					},
					Exists: true,
					Type:   "db",
					Lvm:    false,
				},
			},
		},
	},
}

var OsdDevicesInfoNode1WithParted = func() map[string][]lcmcommon.OsdDaemonInfo {
	newMap := map[string][]lcmcommon.OsdDaemonInfo{}
	for osd, info := range OsdDevicesInfoNode1 {
		newMap[osd] = append([]lcmcommon.OsdDaemonInfo{}, info...)
		if osd == "20" || osd == "25" {
			newDevices := append([]lcmcommon.OsdDevice{}, info[0].Devices...)
			newDevices[1].PartedBy = "/dev/vdd1"
			newMap[osd][0].Devices = newDevices
		}
	}
	return newMap
}()

var OsdDevicesSomeLostInfoNode1 = map[string][]lcmcommon.OsdDaemonInfo{
	"20": {
		{
			OsdUUID:     "vbsgs3a3-sdcv-casq-sd11-asd12dasczsf",
			ClusterFSID: "8668f062-3faa-358a-85f3-f80fe6c1e306",
			Devices: []lcmcommon.OsdDevice{
				{
					Name:       "/dev/vde",
					DeviceID:   "2926ff77-7491-4447-a",
					Rotational: true,
					DeviceSymlinks: []string{
						"/dev/disk/by-id/lvm-pv-uuid-nzJOk1-kLTM-ErxQ-0N4c-DpDU-0zhE-Q9hRJP",
						"/dev/disk/by-id/virtio-2926ff77-7491-4447-a",
						"/dev/disk/by-path/pci-0000:00:0f.0",
						"/dev/disk/by-path/virtio-pci-0000:00:0f.0",
					},
					RelatedPartition: "/dev/ceph-21312wds-sdfv-vs3f-scv3-sdfdsg23edaa/osd-block-vbsgs3a3-sdcv-casq-sd11-asd12dasczsf",
				},
			},
			Partitions: []lcmcommon.OsdPartition{
				{
					Partition: "/dev/ceph-21312wds-sdfv-vs3f-scv3-sdfdsg23edaa/osd-block-vbsgs3a3-sdcv-casq-sd11-asd12dasczsf",
					PartitionSymlinks: []string{
						"/dev/disk/by-id/dm-name-ceph--21312wds--sdfv--vs3f--scv3--sdfdsg23edaa-osd--block--vbsgs3a3--sdcv--casq--sd11--asd12dasczsf",
						"/dev/disk/by-id/dm-uuid-LVM-oPXPcruZ1AK9dkZOsPR9ZW7PzVb9xtFrOnhN24VqDzKIOPBZLd60UQpS6PpCEzQs",
						"/dev/dm-5",
						"/dev/mapper/ceph--21312wds--sdfv--vs3f--scv3--sdfdsg23edaa-osd--block--vbsgs3a3--sdcv--casq--sd11--asd12dasczsf",
					},
					Type:   "block",
					Exists: true,
					Lvm:    true,
				},
				{
					Partition: "/dev/ceph-metadata/part-1",
					Type:      "db",
				},
			},
		},
	},
	"25": {
		{
			OsdUUID:     "d49fd9bf-d2dd-4c3d-824d-87f3f17ea44a",
			ClusterFSID: "8668f062-3faa-358a-85f3-f80fe6c1e306",
			Devices: []lcmcommon.OsdDevice{
				{
					Name:       "/dev/vdf",
					DeviceID:   "b7ea1c8c-89b8-4354-8",
					Rotational: true,
					DeviceSymlinks: []string{
						"/dev/disk/by-id/lvm-pv-uuid-fZ7Efo-X0nc-lAR3-lzik-MjMT-0rml-lZNf7b",
						"/dev/disk/by-id/virtio-b7ea1c8c-89b8-4354-8",
						"/dev/disk/by-path/pci-0000:00:10.0",
						"/dev/disk/by-path/virtio-pci-0000:00:10.0",
					},
					RelatedPartition: "/dev/ceph-2efce189-afb7-452f-bd32-c73b5017a0da/osd-block-d49fd9bf-d2dd-4c3d-824d-87f3f17ea44a",
				},
			},
			Partitions: []lcmcommon.OsdPartition{
				{
					Partition: "/dev/ceph-2efce189-afb7-452f-bd32-c73b5017a0da/osd-block-d49fd9bf-d2dd-4c3d-824d-87f3f17ea44a",
					PartitionSymlinks: []string{
						"/dev/disk/by-id/dm-name-ceph--2efce189--afb7--452f--bd32--c73b5017a0da-osd--block--d49fd9bf--d2dd--4c3d--824d--87f3f17ea44a",
						"/dev/disk/by-id/dm-uuid-LVM-LMoz5X0a3VV3TO2rMomDqfh24zt91NaCiZmlePb5dTd9cws2kHF6Q28W96aUWWgJ",
						"/dev/dm-4",
						"/dev/mapper/ceph--2efce189--afb7--452f--bd32--c73b5017a0da-osd--block--d49fd9bf--d2dd--4c3d--824d--87f3f17ea44a",
					},
					Type:   "block",
					Exists: true,
					Lvm:    true,
				},
				{
					Partition: "/dev/ceph-metadata/part-2",
					Type:      "db",
				},
			},
		},
	},
}

var OsdDevicesInfoNode2 = map[string][]lcmcommon.OsdDaemonInfo{
	"0": {
		{
			OsdUUID:     "69481cd1-38b1-42fd-ac07-06bf4d7c0e19",
			ClusterFSID: "8668f062-3faa-358a-85f3-f80fe6c1e306",
			Devices: []lcmcommon.OsdDevice{
				{
					Name:       "/dev/vdb",
					DeviceID:   "b4eaf39c-b561-4269-1",
					Rotational: true,
					DeviceSymlinks: []string{
						"/dev/disk/by-id/lvm-pv-uuid-8WtEUN-j4yZ-IrQb-viol-aRdu-EWd0-2rfAIz",
						"/dev/disk/by-id/virtio-b4eaf39c-b561-4269-1",
						"/dev/disk/by-path/pci-0000:00:0a.0",
						"/dev/disk/by-path/virtio-pci-0000:00:0a.0",
					},
					RelatedPartition: "/dev/ceph-cf7c8b53-27c7-4cfc-94de-6ad4c7d9f92d/osd-block-af39b794-e1c6-41c0-8997-d6b6c631b8f2",
				},
			},
			Partitions: []lcmcommon.OsdPartition{
				{
					Partition: "/dev/ceph-cf7c8b53-27c7-4cfc-94de-6ad4c7d9f92d/osd-block-af39b794-e1c6-41c0-8997-d6b6c631b8f2",
					PartitionSymlinks: []string{
						"/dev/disk/by-id/dm-name-ceph--cf7c8b53--27c7--4cfc--94de--6ad4c7d9f92d-osd--block--af39b794--e1c6--41c0--8997--d6b6c631b8f2",
						"/dev/disk/by-id/dm-uuid-LVM-6KkuftLkX8aUrCZNOp5tmQEbBDES1Vw2Ny8orN6geq0l6g9uoIRwxmCyyHViH1jWV",
						"/dev/dm-0",
						"/dev/mapper/ceph--cf7c8b53--27c7--4cfc--94de--6ad4c7d9f92d-osd--block--af39b794--e1c6--41c0--8997--d6b6c631b8f2",
					},
					Type:   "block",
					Exists: true,
					Lvm:    true,
				},
			},
		},
	},
	"0-stray": {
		{
			OsdUUID:     "06bf4d7c-9603-41a4-b250-284ecf3ecb2f",
			ClusterFSID: "8668f062-0lsk-358a-1gt4-f80fe6c1e306",
			Devices: []lcmcommon.OsdDevice{
				{
					Name:       "/dev/vdc",
					DeviceID:   "ffe08946-7614-4f69-b",
					Rotational: true,
					DeviceSymlinks: []string{
						"/dev/disk/by-id/lvm-pv-uuid-asfsdm-j4yZ-IrQb-viol-aRdu-EWd0-2rfAIz",
						"/dev/disk/by-id/virtio-ffe08946-7614-4f69-b",
						"/dev/disk/by-path/pci-0000:00:0c.0",
						"/dev/disk/by-path/virtio-pci-0000:00:0c.0",
					},
					RelatedPartition: "/dev/ceph-c5628abe-ae41-4c3d-bdc6-ef86c54bf78c/osd-block-69481cd1-38b1-42fd-ac07-06bf4d7c0e19",
				},
			},
			Partitions: []lcmcommon.OsdPartition{
				{
					Partition: "/dev/ceph-c5628abe-ae41-4c3d-bdc6-ef86c54bf78c/osd-block-69481cd1-38b1-42fd-ac07-06bf4d7c0e19",
					PartitionSymlinks: []string{
						"/dev/disk/by-id/dm-name-ceph--c5628abe--ae41--4c3d--bdc6--ef86c54bf78c-osd--block--69481cd1--38b1--42fd--ac07--06bf4d7c0e19",
						"/dev/disk/by-id/dm-uuid-LVM-9slKo9AQFPaqO6SGUB32zefY9Kpd6hWSUrLCJIw96IhroutTDvuUekUcFZkkPeaHr",
						"/dev/dm-6",
						"/dev/mapper/ceph--c5628abe--ae41--4c3d--bdc6--ef86c54bf78c-osd--block--69481cd1--38b1--42fd--ac07--06bf4d7c0e19",
					},
					Type:   "block",
					Exists: true,
					Lvm:    true,
				},
			},
		},
	},
	"0-stray-nvme": {
		{
			OsdUUID:     "06bf4d7c-9603-41a4-b250-284ecf3ecb2f",
			ClusterFSID: "8668f062-0lsk-358a-1gt4-f80fe6c1e306",
			Devices: []lcmcommon.OsdDevice{
				{
					Name:       "/dev/vdc",
					DeviceID:   "ffe08946-7614-4f69-b",
					Rotational: false,
					DeviceSymlinks: []string{
						"/dev/disk/by-id/lvm-pv-uuid-asfsdm-j4yZ-IrQb-viol-aRdu-EWd0-2rfAIz",
						"/dev/disk/by-id/virtio-ffe08946-7614-4f69-b",
					},
					RelatedPartition: "/dev/ceph-c5628abe-ae41-4c3d-bdc6-ef86c54bf78c/osd-block-69481cd1-38b1-42fd-ac07-06bf4d7c0e19",
				},
			},
			Partitions: []lcmcommon.OsdPartition{
				{
					Partition: "/dev/ceph-c5628abe-ae41-4c3d-bdc6-ef86c54bf78c/osd-block-69481cd1-38b1-42fd-ac07-06bf4d7c0e19",
					PartitionSymlinks: []string{
						"/dev/disk/by-id/dm-name-ceph--c5628abe--ae41--4c3d--bdc6--ef86c54bf78c-osd--block--69481cd1--38b1--42fd--ac07--06bf4d7c0e19",
						"/dev/disk/by-id/dm-uuid-LVM-9slKo9AQFPaqO6SGUB32zefY9Kpd6hWSUrLCJIw96IhroutTDvuUekUcFZkkPeaHr",
						"/dev/dm-6",
						"/dev/mapper/ceph--c5628abe--ae41--4c3d--bdc6--ef86c54bf78c-osd--block--69481cd1--38b1--42fd--ac07--06bf4d7c0e19",
					},
					Type:   "block",
					Exists: true,
					Lvm:    true,
				},
			},
		},
	},
	"2": {
		{
			OsdUUID:     "61869d90-2c45-4f02-b7c3-96955f41e2ca",
			ClusterFSID: "8668f062-3faa-358a-85f3-f80fe6c1e306",
			Devices: []lcmcommon.OsdDevice{
				{
					Name:       "/dev/vde",
					DeviceID:   "8cbb9ce3-6fb4-4216-8",
					Rotational: true,
					DeviceSymlinks: []string{
						"/dev/disk/by-id/lvm-pv-uuid-ZEgqPM-4qjs-uGF5-pPfe-DwmN-korc-fxbLeO",
						"/dev/disk/by-id/virtio-8cbb9ce3-6fb4-4216-8",
						"/dev/disk/by-path/pci-0000:00:0e.0",
						"/dev/disk/by-path/virtio-pci-0000:00:0e.0",
					},
					RelatedPartition: "/dev/ceph-0e03d5c6-d0e9-4f04-b9af-38d15e14369f/osd-block-61869d90-2c45-4f02-b7c3-96955f41e2ca",
				},
			},
			Partitions: []lcmcommon.OsdPartition{
				{
					Partition: "/dev/ceph-0e03d5c6-d0e9-4f04-b9af-38d15e14369f/osd-block-61869d90-2c45-4f02-b7c3-96955f41e2ca",
					PartitionSymlinks: []string{
						"/dev/disk/by-id/dm-name-ceph--0e03d5c6--d0e9--4f04--b9af--38d15e14369f-osd--block--61869d90--2c45--4f02--b7c3--96955f41e2ca",
						"/dev/disk/by-id/dm-uuid-LVM-bo5jsBko0zj6Bcj6IWdhVGylUqzj6HWuAEwSZ452TlhWEymhVEdl9qK6m2CMjOyt2",
						"/dev/dm-4",
						"/dev/mapper/ceph--0e03d5c6--d0e9--4f04--b9af--38d15e14369f-osd--block--61869d90--2c45--4f02--b7c3--96955f41e2ca",
					},
					Type:   "block",
					Exists: true,
					Lvm:    true,
				},
			},
		},
	},
	"4": {
		{
			OsdUUID:     "ad76cf53-5cb5-48fe-a39a-343734f5ccde",
			ClusterFSID: "8668f062-3faa-358a-85f3-f80fe6c1e306",
			Devices: []lcmcommon.OsdDevice{
				{
					Name:       "/dev/vdd",
					DeviceID:   "35a15532-8b56-4f83-9",
					Rotational: false,
					DeviceSymlinks: []string{
						"/dev/disk/by-id/lvm-pv-uuid-3w90Bt-BAGQ-iawd-d7h7-yml2-kZZr-nqCNut",
						"/dev/disk/by-id/virtio-35a15532-8b56-4f83-9",
						"/dev/disk/by-path/pci-0000:00:1e.0",
						"/dev/disk/by-path/virtio-pci-0000:00:1e.0",
					},
					RelatedPartition: "/dev/ceph-dada9f25-41b4-4c26-9a20-448ac01e1d06/osd-block-ad76cf53-5cb5-48fe-a39a-343734f5ccde",
				},
			},
			Partitions: []lcmcommon.OsdPartition{
				{
					Partition: "/dev/ceph-dada9f25-41b4-4c26-9a20-448ac01e1d06/osd-block-ad76cf53-5cb5-48fe-a39a-343734f5ccde",
					PartitionSymlinks: []string{
						"/dev/disk/by-id/dm-name-ceph--dada9f25--41b4--4c26--9a20--448ac01e1d06-osd--block--ad76cf53--5cb5--48fe--a39a--343734f5ccde",
						"/dev/disk/by-id/dm-uuid-LVM-6DPn1o4v5MTKd3rjmRjKM2f4dyxVTnlTa3qt7mahREkp9uqXFAVuegyCmEeZgugBF",
						"/dev/dm-3",
						"/dev/mapper/ceph--dada9f25--41b4--4c26--9a20--448ac01e1d06-osd--block--ad76cf53--5cb5--48fe--a39a--343734f5ccde",
					},
					Type:   "block",
					Exists: true,
					Lvm:    true,
				},
			},
		},
	},
	"5": {
		{
			OsdUUID:     "af39b794-e1c6-41c0-8997-d6b6c631b8f2",
			ClusterFSID: "8668f062-3faa-358a-85f3-f80fe6c1e306",
			Devices: []lcmcommon.OsdDevice{
				{
					Name:       "/dev/vdd",
					DeviceID:   "35a15532-8b56-4f83-9",
					Rotational: false,
					DeviceSymlinks: []string{
						"/dev/disk/by-id/lvm-pv-uuid-3w90Bt-BAGQ-iawd-d7h7-yml2-kZZr-nqCNut",
						"/dev/disk/by-id/virtio-35a15532-8b56-4f83-9",
						"/dev/disk/by-path/pci-0000:00:1e.0",
						"/dev/disk/by-path/virtio-pci-0000:00:1e.0",
					},
					RelatedPartition: "/dev/ceph-dada9f25-41b4-4c26-9a20-448ac01e1d06/osd-block-7d09cceb-4de0-478e-9d8d-bd09cb0c904e",
				},
			},
			Partitions: []lcmcommon.OsdPartition{
				{
					Partition: "/dev/ceph-dada9f25-41b4-4c26-9a20-448ac01e1d06/osd-block-7d09cceb-4de0-478e-9d8d-bd09cb0c904e",
					PartitionSymlinks: []string{
						"/dev/disk/by-id/dm-name-ceph--dada9f25--41b4--4c26--9a20--448ac01e1d06-osd--block--7d09cceb--4de0--478e--9d8d--bd09cb0c904e",
						"/dev/disk/by-id/dm-uuid-LVM-tN7IFP8FqkBisGqDK0TyyYLl4RnBfXbdZtlcroq9NCpB5wzfkuq0yyypeXrJlL8sh",
						"/dev/dm-2",
						"/dev/mapper/ceph--dada9f25--41b4--4c26--9a20--448ac01e1d06-osd--block--7d09cceb--4de0--478e--9d8d--bd09cb0c904e",
					},
					Type:   "block",
					Exists: true,
					Lvm:    true,
				},
			},
		},
	},
}
