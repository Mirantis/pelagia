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

import lcmcommon "github.com/Mirantis/pelagia/pkg/common"

var LsblkReportFromNode2 = `{
   "blockdevices": [
      {"name": "/dev/loop0", "kname": "/dev/loop0", "maj:min": "7:0", "rota": true, "type": "loop", "pkname": null, "serial": null},
      {"name": "/dev/loop1", "kname": "/dev/loop1", "maj:min": "7:1", "rota": true, "type": "loop", "pkname": null, "serial": null},
      {"name": "/dev/loop2", "kname": "/dev/loop2", "maj:min": "7:2", "rota": true, "type": "loop", "pkname": null, "serial": null},
      {"name": "/dev/vda", "kname": "/dev/vda", "maj:min": "8:0", "rota": true, "type": "disk", "pkname": null, "serial": "908acd1c-2e38-4f10-9",
         "children": [
            {"name":"/dev/vda1", "kname":"/dev/vda1", "maj:min":"8:1", "fstype":"ext4", "rota": true, "type":"part", "pkname":"/dev/vda", "serial": null}
         ]
      },
      {"name": "/dev/vdb", "kname": "/dev/vdb", "maj:min": "8:16", "rota": true, "type": "disk", "pkname": null, "serial": "b4eaf39c-b561-4269-1",
         "children": [
            {"name": "/dev/mapper/ceph--cf7c8b53--27c7--4cfc--94de--6ad4c7d9f92d-osd--block--af39b794--e1c6--41c0--8997--d6b6c631b8f2", "kname": "/dev/dm-0", "maj:min": "253:0", "rota": true, "type": "lvm", "pkname": "/dev/vdb", "fstype": "ceph_bluestore", "serial": null}
         ]
      },
      {"name": "/dev/vdc", "kname": "/dev/vdc", "maj:min": "8:32", "rota": true, "type": "disk", "pkname": null, "serial": "ffe08946-7614-4f69-b",
         "children": [
            {"name": "/dev/mapper/ceph--c5628abe--ae41--4c3d--bdc6--ef86c54bf78c-osd--block--69481cd1--38b1--42fd--ac07--06bf4d7c0e19", "kname": "/dev/dm-6", "maj:min": "253:6", "rota": true, "type": "lvm", "pkname": "/dev/vdc", "fstype": "ceph_bluestore", "serial": null}
         ]
      },
      {"name": "/dev/vdd", "kname": "/dev/vdd", "maj:min": "8:48", "rota": false, "type": "disk", "pkname": null, "serial": "35a15532-8b56-4f83-9",
         "children": [
            {"name": "/dev/mapper/ceph--dada9f25--41b4--4c26--9a20--448ac01e1d06-osd--block--7d09cceb--4de0--478e--9d8d--bd09cb0c904e", "kname": "/dev/dm-2", "maj:min": "253:2", "rota": false, "type": "lvm", "pkname": "/dev/vdd", "fstype": "ceph_bluestore", "serial": null},
            {"name": "/dev/mapper/ceph--dada9f25--41b4--4c26--9a20--448ac01e1d06-osd--block--ad76cf53--5cb5--48fe--a39a--343734f5ccde", "kname": "/dev/dm-3", "maj:min": "253:3", "rota": false, "type": "lvm", "pkname": "/dev/vdd", "fstype": "ceph_bluestore", "serial": null}
         ]
      },
      {"name": "/dev/vde", "kname": "/dev/vde", "maj:min": "8:64", "rota": true, "type": "disk", "pkname": null, "serial": "8cbb9ce3-6fb4-4216-8",
         "children": [
            {"name": "/dev/mapper/ceph--0e03d5c6--d0e9--4f04--b9af--38d15e14369f-osd--block--61869d90--2c45--4f02--b7c3--96955f41e2ca", "kname": "/dev/dm-4", "maj:min": "253:4", "rota": true, "type": "lvm", "pkname": "/dev/vde", "fstype": "ceph_bluestore", "serial": null}
         ]
      },
      {"name": "/dev/vdf", "kname": "/dev/vdf", "maj:min": "8:80", "rota": true, "type": "disk", "pkname": null, "serial": "0089f849-3053-4b68-8",
         "children": [
            {"name":"/dev/mapper/data-extra", "kname":"/dev/dm-1", "maj:min":"253:1", "rota": true, "type":"lvm",  "pkname":"/dev/vdf", "serial": null}
         ]
      },
      {"name": "/dev/vdg", "kname": "/dev/vdg", "maj:min": "8:96", "rota": true, "type": "disk", "pkname": null, "serial": "5f7cc96a-8a07-4eaa-b",
         "children": [
            {"name":"/dev/mapper/data-extra", "kname":"/dev/dm-1", "maj:min":"253:1", "rota": true, "type":"lvm",  "pkname":"/dev/vdg", "serial": null}
         ]
      }
   ]
}`

var UdevadmReportFromNode2 = map[string]string{
	// mappers symlinks
	"/dev/mapper/ceph--cf7c8b53--27c7--4cfc--94de--6ad4c7d9f92d-osd--block--af39b794--e1c6--41c0--8997--d6b6c631b8f2": "/dev/disk/by-id/dm-uuid-LVM-6KkuftLkX8aUrCZNOp5tmQEbBDES1Vw2Ny8orN6geq0l6g9uoIRwxmCyyHViH1jWV /dev/disk/by-id/dm-name-ceph--cf7c8b53--27c7--4cfc--94de--6ad4c7d9f92d-osd--block--af39b794--e1c6--41c0--8997--d6b6c631b8f2",
	"/dev/mapper/ceph--dada9f25--41b4--4c26--9a20--448ac01e1d06-osd--block--7d09cceb--4de0--478e--9d8d--bd09cb0c904e": "/dev/disk/by-id/dm-uuid-LVM-tN7IFP8FqkBisGqDK0TyyYLl4RnBfXbdZtlcroq9NCpB5wzfkuq0yyypeXrJlL8sh /dev/disk/by-id/dm-name-ceph--dada9f25--41b4--4c26--9a20--448ac01e1d06-osd--block--7d09cceb--4de0--478e--9d8d--bd09cb0c904e",
	"/dev/mapper/ceph--c5628abe--ae41--4c3d--bdc6--ef86c54bf78c-osd--block--69481cd1--38b1--42fd--ac07--06bf4d7c0e19": "/dev/disk/by-id/dm-uuid-LVM-9slKo9AQFPaqO6SGUB32zefY9Kpd6hWSUrLCJIw96IhroutTDvuUekUcFZkkPeaHr /dev/disk/by-id/dm-name-ceph--c5628abe--ae41--4c3d--bdc6--ef86c54bf78c-osd--block--69481cd1--38b1--42fd--ac07--06bf4d7c0e19",
	"/dev/mapper/ceph--0e03d5c6--d0e9--4f04--b9af--38d15e14369f-osd--block--61869d90--2c45--4f02--b7c3--96955f41e2ca": "/dev/disk/by-id/dm-uuid-LVM-bo5jsBko0zj6Bcj6IWdhVGylUqzj6HWuAEwSZ452TlhWEymhVEdl9qK6m2CMjOyt2 /dev/disk/by-id/dm-name-ceph--0e03d5c6--d0e9--4f04--b9af--38d15e14369f-osd--block--61869d90--2c45--4f02--b7c3--96955f41e2ca",
	"/dev/mapper/ceph--dada9f25--41b4--4c26--9a20--448ac01e1d06-osd--block--ad76cf53--5cb5--48fe--a39a--343734f5ccde": "/dev/disk/by-id/dm-uuid-LVM-6DPn1o4v5MTKd3rjmRjKM2f4dyxVTnlTa3qt7mahREkp9uqXFAVuegyCmEeZgugBF /dev/disk/by-id/dm-name-ceph--dada9f25--41b4--4c26--9a20--448ac01e1d06-osd--block--ad76cf53--5cb5--48fe--a39a--343734f5ccde",
	// manually created volume
	"/dev/mapper/data-extra": "/dev/data/extra /dev/disk/by-id/dm-uuid-LVM-JdeJK2B0l37nzFkx06PJTxK3kk1pb4QMepESDhQ8qEKVsN11qSlbAWHsadL8JScW /dev/mapper/data-extra /dev/disk/by-id/dm-name-data-extra",
	// devs symlinks
	"/dev/vda":  "/dev/disk/by-id/virtio-908acd1c-2e38-4f10-9 /dev/disk/by-path/pci-0000:00:10.0 /dev/disk/by-path/virtio-pci-0000:00:10.0",
	"/dev/vda1": "/dev/disk/by-label/cloudimg-rootfs /dev/disk/by-path/virtio-pci-0000:00:10.0-part1 /dev/disk/by-partuuid/8a13887f-ec53-4427-b9ec-8291e6213b29 /dev/disk/by-path/pci-0000:00:10.0-part1 /dev/disk/by-uuid/f51a1ffe-1cc1-4274-a35e-7b6dd027ddc9",
	"/dev/vdb":  "/dev/disk/by-path/pci-0000:00:0a.0 /dev/disk/by-id/lvm-pv-uuid-8WtEUN-j4yZ-IrQb-viol-aRdu-EWd0-2rfAIz /dev/disk/by-path/virtio-pci-0000:00:0a.0 /dev/disk/by-id/virtio-b4eaf39c-b561-4269-1",
	"/dev/vdc":  "/dev/disk/by-path/pci-0000:00:0c.0 /dev/disk/by-path/virtio-pci-0000:00:0c.0 /dev/disk/by-id/lvm-pv-uuid-asfsdm-j4yZ-IrQb-viol-aRdu-EWd0-2rfAIz /dev/disk/by-id/virtio-ffe08946-7614-4f69-b",
	"/dev/vdd":  "/dev/disk/by-path/pci-0000:00:1e.0 /dev/disk/by-id/lvm-pv-uuid-3w90Bt-BAGQ-iawd-d7h7-yml2-kZZr-nqCNut /dev/disk/by-id/virtio-35a15532-8b56-4f83-9 /dev/disk/by-path/virtio-pci-0000:00:1e.0",
	"/dev/vde":  "/dev/disk/by-id/lvm-pv-uuid-ZEgqPM-4qjs-uGF5-pPfe-DwmN-korc-fxbLeO /dev/disk/by-path/virtio-pci-0000:00:0e.0 /dev/disk/by-path/pci-0000:00:0e.0 /dev/disk/by-id/virtio-8cbb9ce3-6fb4-4216-8",
	"/dev/vdf":  "/dev/disk/by-path/pci-0000:00:11.0 /dev/disk/by-path/virtio-pci-0000:00:11.0 /dev/disk/by-id/lvm-pv-uuid-JuVctY-xFTH-Sc8f-r1on-kD9p-R1pZ-DA2jdX /dev/disk/by-id/virtio-0089f849-3053-4b68-8",
	"/dev/vdg":  "/dev/disk/by-id/lvm-pv-uuid-Ji3Caj-mNrf-tiTx-Xi9K-JF5Z-MYjT-Cxok9j /dev/disk/by-path/virtio-pci-0000:00:12.0 /dev/disk/by-path/pci-0000:00:12.0 /dev/disk/by-id/virtio-5f7cc96a-8a07-4eaa-b",
}

var DiskInfoReportLsblkFromNode2 = &lcmcommon.DiskDaemonDisksReport{
	BlockInfo: map[string]lcmcommon.BlockDeviceInfo{
		"/dev/mapper/ceph--cf7c8b53--27c7--4cfc--94de--6ad4c7d9f92d-osd--block--af39b794--e1c6--41c0--8997--d6b6c631b8f2": {
			Kname: "/dev/dm-0", Type: "lvm", Rotational: true, MajMin: "253:0", Parent: []string{"/dev/vdb"},
			Symlinks: []string{
				"/dev/disk/by-id/dm-name-ceph--cf7c8b53--27c7--4cfc--94de--6ad4c7d9f92d-osd--block--af39b794--e1c6--41c0--8997--d6b6c631b8f2",
				"/dev/disk/by-id/dm-uuid-LVM-6KkuftLkX8aUrCZNOp5tmQEbBDES1Vw2Ny8orN6geq0l6g9uoIRwxmCyyHViH1jWV",
			},
			Childrens: []string{},
		},
		"/dev/mapper/ceph--c5628abe--ae41--4c3d--bdc6--ef86c54bf78c-osd--block--69481cd1--38b1--42fd--ac07--06bf4d7c0e19": {
			Kname: "/dev/dm-6", Type: "lvm", Rotational: true, MajMin: "253:6", Parent: []string{"/dev/vdc"},
			Symlinks: []string{
				"/dev/disk/by-id/dm-name-ceph--c5628abe--ae41--4c3d--bdc6--ef86c54bf78c-osd--block--69481cd1--38b1--42fd--ac07--06bf4d7c0e19",
				"/dev/disk/by-id/dm-uuid-LVM-9slKo9AQFPaqO6SGUB32zefY9Kpd6hWSUrLCJIw96IhroutTDvuUekUcFZkkPeaHr",
			},
			Childrens: []string{},
		},
		"/dev/mapper/ceph--dada9f25--41b4--4c26--9a20--448ac01e1d06-osd--block--7d09cceb--4de0--478e--9d8d--bd09cb0c904e": {
			Kname: "/dev/dm-2", Type: "lvm", Rotational: false, MajMin: "253:2", Parent: []string{"/dev/vdd"},
			Symlinks: []string{
				"/dev/disk/by-id/dm-name-ceph--dada9f25--41b4--4c26--9a20--448ac01e1d06-osd--block--7d09cceb--4de0--478e--9d8d--bd09cb0c904e",
				"/dev/disk/by-id/dm-uuid-LVM-tN7IFP8FqkBisGqDK0TyyYLl4RnBfXbdZtlcroq9NCpB5wzfkuq0yyypeXrJlL8sh",
			},
			Childrens: []string{},
		},
		"/dev/mapper/ceph--dada9f25--41b4--4c26--9a20--448ac01e1d06-osd--block--ad76cf53--5cb5--48fe--a39a--343734f5ccde": {
			Kname: "/dev/dm-3", Type: "lvm", Rotational: false, MajMin: "253:3", Parent: []string{"/dev/vdd"},
			Symlinks: []string{
				"/dev/disk/by-id/dm-name-ceph--dada9f25--41b4--4c26--9a20--448ac01e1d06-osd--block--ad76cf53--5cb5--48fe--a39a--343734f5ccde",
				"/dev/disk/by-id/dm-uuid-LVM-6DPn1o4v5MTKd3rjmRjKM2f4dyxVTnlTa3qt7mahREkp9uqXFAVuegyCmEeZgugBF",
			},
			Childrens: []string{},
		},
		"/dev/mapper/ceph--0e03d5c6--d0e9--4f04--b9af--38d15e14369f-osd--block--61869d90--2c45--4f02--b7c3--96955f41e2ca": {
			Kname: "/dev/dm-4", Type: "lvm", Rotational: true, MajMin: "253:4", Parent: []string{"/dev/vde"},
			Symlinks: []string{
				"/dev/disk/by-id/dm-name-ceph--0e03d5c6--d0e9--4f04--b9af--38d15e14369f-osd--block--61869d90--2c45--4f02--b7c3--96955f41e2ca",
				"/dev/disk/by-id/dm-uuid-LVM-bo5jsBko0zj6Bcj6IWdhVGylUqzj6HWuAEwSZ452TlhWEymhVEdl9qK6m2CMjOyt2",
			},
			Childrens: []string{},
		},
		"/dev/mapper/data-extra": {
			Kname: "/dev/dm-1", Type: "lvm", Rotational: true, MajMin: "253:1", Parent: []string{"/dev/vdf", "/dev/vdg"},
			Symlinks: []string{
				"/dev/data/extra",
				"/dev/disk/by-id/dm-name-data-extra",
				"/dev/disk/by-id/dm-uuid-LVM-JdeJK2B0l37nzFkx06PJTxK3kk1pb4QMepESDhQ8qEKVsN11qSlbAWHsadL8JScW",
				"/dev/mapper/data-extra",
			},
			Childrens: []string{},
		},
		"/dev/vda": {
			Kname: "/dev/vda", Type: "disk", Rotational: true, MajMin: "8:0", Parent: []string{""}, Serial: "908acd1c-2e38-4f10-9",
			Symlinks: []string{
				"/dev/disk/by-id/virtio-908acd1c-2e38-4f10-9",
				"/dev/disk/by-path/pci-0000:00:10.0",
				"/dev/disk/by-path/virtio-pci-0000:00:10.0",
			},
			Childrens: []string{"/dev/vda1"},
		},
		"/dev/vda1": {
			Kname: "/dev/vda1", Type: "part", Rotational: true, MajMin: "8:1", Parent: []string{"/dev/vda"},
			Symlinks: []string{
				"/dev/disk/by-label/cloudimg-rootfs",
				"/dev/disk/by-partuuid/8a13887f-ec53-4427-b9ec-8291e6213b29",
				"/dev/disk/by-path/pci-0000:00:10.0-part1",
				"/dev/disk/by-path/virtio-pci-0000:00:10.0-part1",
				"/dev/disk/by-uuid/f51a1ffe-1cc1-4274-a35e-7b6dd027ddc9",
			},
			Childrens: []string{},
		},
		"/dev/vdb": {
			Kname: "/dev/vdb", Type: "disk", Rotational: true, MajMin: "8:16", Parent: []string{""}, Serial: "b4eaf39c-b561-4269-1",
			Symlinks: []string{
				"/dev/disk/by-id/lvm-pv-uuid-8WtEUN-j4yZ-IrQb-viol-aRdu-EWd0-2rfAIz",
				"/dev/disk/by-id/virtio-b4eaf39c-b561-4269-1",
				"/dev/disk/by-path/pci-0000:00:0a.0",
				"/dev/disk/by-path/virtio-pci-0000:00:0a.0",
			},
			Childrens: []string{"/dev/mapper/ceph--cf7c8b53--27c7--4cfc--94de--6ad4c7d9f92d-osd--block--af39b794--e1c6--41c0--8997--d6b6c631b8f2"},
		},
		"/dev/vdc": {
			Kname: "/dev/vdc", Type: "disk", Rotational: true, MajMin: "8:32", Parent: []string{""}, Serial: "ffe08946-7614-4f69-b",
			Symlinks: []string{
				"/dev/disk/by-id/lvm-pv-uuid-asfsdm-j4yZ-IrQb-viol-aRdu-EWd0-2rfAIz",
				"/dev/disk/by-id/virtio-ffe08946-7614-4f69-b",
				"/dev/disk/by-path/pci-0000:00:0c.0",
				"/dev/disk/by-path/virtio-pci-0000:00:0c.0",
			},
			Childrens: []string{"/dev/mapper/ceph--c5628abe--ae41--4c3d--bdc6--ef86c54bf78c-osd--block--69481cd1--38b1--42fd--ac07--06bf4d7c0e19"},
		},
		"/dev/vdd": {
			Kname: "/dev/vdd", Type: "disk", Rotational: false, MajMin: "8:48", Parent: []string{""}, Serial: "35a15532-8b56-4f83-9",
			Symlinks: []string{
				"/dev/disk/by-id/lvm-pv-uuid-3w90Bt-BAGQ-iawd-d7h7-yml2-kZZr-nqCNut",
				"/dev/disk/by-id/virtio-35a15532-8b56-4f83-9",
				"/dev/disk/by-path/pci-0000:00:1e.0",
				"/dev/disk/by-path/virtio-pci-0000:00:1e.0",
			},
			Childrens: []string{
				"/dev/mapper/ceph--dada9f25--41b4--4c26--9a20--448ac01e1d06-osd--block--7d09cceb--4de0--478e--9d8d--bd09cb0c904e",
				"/dev/mapper/ceph--dada9f25--41b4--4c26--9a20--448ac01e1d06-osd--block--ad76cf53--5cb5--48fe--a39a--343734f5ccde",
			},
		},
		"/dev/vde": {
			Kname: "/dev/vde", Type: "disk", Rotational: true, MajMin: "8:64", Parent: []string{""}, Serial: "8cbb9ce3-6fb4-4216-8",
			Symlinks: []string{
				"/dev/disk/by-id/lvm-pv-uuid-ZEgqPM-4qjs-uGF5-pPfe-DwmN-korc-fxbLeO",
				"/dev/disk/by-id/virtio-8cbb9ce3-6fb4-4216-8",
				"/dev/disk/by-path/pci-0000:00:0e.0",
				"/dev/disk/by-path/virtio-pci-0000:00:0e.0",
			},
			Childrens: []string{
				"/dev/mapper/ceph--0e03d5c6--d0e9--4f04--b9af--38d15e14369f-osd--block--61869d90--2c45--4f02--b7c3--96955f41e2ca",
			},
		},
		"/dev/vdf": {
			Kname: "/dev/vdf", Type: "disk", Rotational: true, MajMin: "8:80", Parent: []string{""}, Serial: "0089f849-3053-4b68-8",
			Symlinks: []string{
				"/dev/disk/by-id/lvm-pv-uuid-JuVctY-xFTH-Sc8f-r1on-kD9p-R1pZ-DA2jdX",
				"/dev/disk/by-id/virtio-0089f849-3053-4b68-8",
				"/dev/disk/by-path/pci-0000:00:11.0",
				"/dev/disk/by-path/virtio-pci-0000:00:11.0",
			},
			Childrens: []string{"/dev/mapper/data-extra"},
		},
		"/dev/vdg": {
			Kname: "/dev/vdg", Type: "disk", Rotational: true, MajMin: "8:96", Parent: []string{""}, Serial: "5f7cc96a-8a07-4eaa-b",
			Symlinks: []string{
				"/dev/disk/by-id/lvm-pv-uuid-Ji3Caj-mNrf-tiTx-Xi9K-JF5Z-MYjT-Cxok9j",
				"/dev/disk/by-id/virtio-5f7cc96a-8a07-4eaa-b",
				"/dev/disk/by-path/pci-0000:00:12.0",
				"/dev/disk/by-path/virtio-pci-0000:00:12.0",
			},
			Childrens: []string{"/dev/mapper/data-extra"},
		},
	},
	Aliases: map[string]string{
		// lvm partition pathes used in ceph-volumes
		"/dev/ceph-cf7c8b53-27c7-4cfc-94de-6ad4c7d9f92d/osd-block-af39b794-e1c6-41c0-8997-d6b6c631b8f2": "/dev/mapper/ceph--cf7c8b53--27c7--4cfc--94de--6ad4c7d9f92d-osd--block--af39b794--e1c6--41c0--8997--d6b6c631b8f2",
		"/dev/ceph-dada9f25-41b4-4c26-9a20-448ac01e1d06/osd-block-7d09cceb-4de0-478e-9d8d-bd09cb0c904e": "/dev/mapper/ceph--dada9f25--41b4--4c26--9a20--448ac01e1d06-osd--block--7d09cceb--4de0--478e--9d8d--bd09cb0c904e",
		"/dev/ceph-c5628abe-ae41-4c3d-bdc6-ef86c54bf78c/osd-block-69481cd1-38b1-42fd-ac07-06bf4d7c0e19": "/dev/mapper/ceph--c5628abe--ae41--4c3d--bdc6--ef86c54bf78c-osd--block--69481cd1--38b1--42fd--ac07--06bf4d7c0e19",
		"/dev/ceph-0e03d5c6-d0e9-4f04-b9af-38d15e14369f/osd-block-61869d90-2c45-4f02-b7c3-96955f41e2ca": "/dev/mapper/ceph--0e03d5c6--d0e9--4f04--b9af--38d15e14369f-osd--block--61869d90--2c45--4f02--b7c3--96955f41e2ca",
		"/dev/ceph-dada9f25-41b4-4c26-9a20-448ac01e1d06/osd-block-ad76cf53-5cb5-48fe-a39a-343734f5ccde": "/dev/mapper/ceph--dada9f25--41b4--4c26--9a20--448ac01e1d06-osd--block--ad76cf53--5cb5--48fe--a39a--343734f5ccde",
		"/dev/data/extra": "/dev/mapper/data-extra",
		// disk path by-id
		"/dev/disk/by-id/dm-name-ceph--cf7c8b53--27c7--4cfc--94de--6ad4c7d9f92d-osd--block--af39b794--e1c6--41c0--8997--d6b6c631b8f2": "/dev/mapper/ceph--cf7c8b53--27c7--4cfc--94de--6ad4c7d9f92d-osd--block--af39b794--e1c6--41c0--8997--d6b6c631b8f2",
		"/dev/disk/by-id/dm-name-ceph--dada9f25--41b4--4c26--9a20--448ac01e1d06-osd--block--7d09cceb--4de0--478e--9d8d--bd09cb0c904e": "/dev/mapper/ceph--dada9f25--41b4--4c26--9a20--448ac01e1d06-osd--block--7d09cceb--4de0--478e--9d8d--bd09cb0c904e",
		"/dev/disk/by-id/dm-name-ceph--c5628abe--ae41--4c3d--bdc6--ef86c54bf78c-osd--block--69481cd1--38b1--42fd--ac07--06bf4d7c0e19": "/dev/mapper/ceph--c5628abe--ae41--4c3d--bdc6--ef86c54bf78c-osd--block--69481cd1--38b1--42fd--ac07--06bf4d7c0e19",
		"/dev/disk/by-id/dm-name-ceph--0e03d5c6--d0e9--4f04--b9af--38d15e14369f-osd--block--61869d90--2c45--4f02--b7c3--96955f41e2ca": "/dev/mapper/ceph--0e03d5c6--d0e9--4f04--b9af--38d15e14369f-osd--block--61869d90--2c45--4f02--b7c3--96955f41e2ca",
		"/dev/disk/by-id/dm-name-ceph--dada9f25--41b4--4c26--9a20--448ac01e1d06-osd--block--ad76cf53--5cb5--48fe--a39a--343734f5ccde": "/dev/mapper/ceph--dada9f25--41b4--4c26--9a20--448ac01e1d06-osd--block--ad76cf53--5cb5--48fe--a39a--343734f5ccde",
		"/dev/disk/by-id/dm-name-data-extra": "/dev/mapper/data-extra",
		"/dev/disk/by-id/dm-uuid-LVM-6KkuftLkX8aUrCZNOp5tmQEbBDES1Vw2Ny8orN6geq0l6g9uoIRwxmCyyHViH1jWV": "/dev/mapper/ceph--cf7c8b53--27c7--4cfc--94de--6ad4c7d9f92d-osd--block--af39b794--e1c6--41c0--8997--d6b6c631b8f2",
		"/dev/disk/by-id/dm-uuid-LVM-tN7IFP8FqkBisGqDK0TyyYLl4RnBfXbdZtlcroq9NCpB5wzfkuq0yyypeXrJlL8sh": "/dev/mapper/ceph--dada9f25--41b4--4c26--9a20--448ac01e1d06-osd--block--7d09cceb--4de0--478e--9d8d--bd09cb0c904e",
		"/dev/disk/by-id/dm-uuid-LVM-9slKo9AQFPaqO6SGUB32zefY9Kpd6hWSUrLCJIw96IhroutTDvuUekUcFZkkPeaHr": "/dev/mapper/ceph--c5628abe--ae41--4c3d--bdc6--ef86c54bf78c-osd--block--69481cd1--38b1--42fd--ac07--06bf4d7c0e19",
		"/dev/disk/by-id/dm-uuid-LVM-bo5jsBko0zj6Bcj6IWdhVGylUqzj6HWuAEwSZ452TlhWEymhVEdl9qK6m2CMjOyt2": "/dev/mapper/ceph--0e03d5c6--d0e9--4f04--b9af--38d15e14369f-osd--block--61869d90--2c45--4f02--b7c3--96955f41e2ca",
		"/dev/disk/by-id/dm-uuid-LVM-6DPn1o4v5MTKd3rjmRjKM2f4dyxVTnlTa3qt7mahREkp9uqXFAVuegyCmEeZgugBF": "/dev/mapper/ceph--dada9f25--41b4--4c26--9a20--448ac01e1d06-osd--block--ad76cf53--5cb5--48fe--a39a--343734f5ccde",
		"/dev/disk/by-id/dm-uuid-LVM-JdeJK2B0l37nzFkx06PJTxK3kk1pb4QMepESDhQ8qEKVsN11qSlbAWHsadL8JScW":  "/dev/mapper/data-extra",
		"/dev/disk/by-id/lvm-pv-uuid-8WtEUN-j4yZ-IrQb-viol-aRdu-EWd0-2rfAIz":                            "/dev/vdb",
		"/dev/disk/by-id/lvm-pv-uuid-asfsdm-j4yZ-IrQb-viol-aRdu-EWd0-2rfAIz":                            "/dev/vdc",
		"/dev/disk/by-id/lvm-pv-uuid-3w90Bt-BAGQ-iawd-d7h7-yml2-kZZr-nqCNut":                            "/dev/vdd",
		"/dev/disk/by-id/lvm-pv-uuid-ZEgqPM-4qjs-uGF5-pPfe-DwmN-korc-fxbLeO":                            "/dev/vde",
		"/dev/disk/by-id/lvm-pv-uuid-JuVctY-xFTH-Sc8f-r1on-kD9p-R1pZ-DA2jdX":                            "/dev/vdf",
		"/dev/disk/by-id/lvm-pv-uuid-Ji3Caj-mNrf-tiTx-Xi9K-JF5Z-MYjT-Cxok9j":                            "/dev/vdg",
		"/dev/disk/by-id/virtio-908acd1c-2e38-4f10-9":                                                   "/dev/vda",
		"/dev/disk/by-id/virtio-b4eaf39c-b561-4269-1":                                                   "/dev/vdb",
		"/dev/disk/by-id/virtio-ffe08946-7614-4f69-b":                                                   "/dev/vdc",
		"/dev/disk/by-id/virtio-35a15532-8b56-4f83-9":                                                   "/dev/vdd",
		"/dev/disk/by-id/virtio-8cbb9ce3-6fb4-4216-8":                                                   "/dev/vde",
		"/dev/disk/by-id/virtio-0089f849-3053-4b68-8":                                                   "/dev/vdf",
		"/dev/disk/by-id/virtio-5f7cc96a-8a07-4eaa-b":                                                   "/dev/vdg",
		// disk path by label
		"/dev/disk/by-label/cloudimg-rootfs": "/dev/vda1",
		// disk path by part uuid
		"/dev/disk/by-partuuid/8a13887f-ec53-4427-b9ec-8291e6213b29": "/dev/vda1",
		// disk path by path
		"/dev/disk/by-path/pci-0000:00:10.0":              "/dev/vda",
		"/dev/disk/by-path/pci-0000:00:10.0-part1":        "/dev/vda1",
		"/dev/disk/by-path/pci-0000:00:0a.0":              "/dev/vdb",
		"/dev/disk/by-path/pci-0000:00:0c.0":              "/dev/vdc",
		"/dev/disk/by-path/pci-0000:00:1e.0":              "/dev/vdd",
		"/dev/disk/by-path/pci-0000:00:0e.0":              "/dev/vde",
		"/dev/disk/by-path/pci-0000:00:11.0":              "/dev/vdf",
		"/dev/disk/by-path/pci-0000:00:12.0":              "/dev/vdg",
		"/dev/disk/by-path/virtio-pci-0000:00:10.0":       "/dev/vda",
		"/dev/disk/by-path/virtio-pci-0000:00:10.0-part1": "/dev/vda1",
		"/dev/disk/by-path/virtio-pci-0000:00:0a.0":       "/dev/vdb",
		"/dev/disk/by-path/virtio-pci-0000:00:0c.0":       "/dev/vdc",
		"/dev/disk/by-path/virtio-pci-0000:00:1e.0":       "/dev/vdd",
		"/dev/disk/by-path/virtio-pci-0000:00:0e.0":       "/dev/vde",
		"/dev/disk/by-path/virtio-pci-0000:00:11.0":       "/dev/vdf",
		"/dev/disk/by-path/virtio-pci-0000:00:12.0":       "/dev/vdg",
		// disk path by uuid
		"/dev/disk/by-uuid/f51a1ffe-1cc1-4274-a35e-7b6dd027ddc9": "/dev/vda1",
		// device mapper aliases
		"/dev/dm-0": "/dev/mapper/ceph--cf7c8b53--27c7--4cfc--94de--6ad4c7d9f92d-osd--block--af39b794--e1c6--41c0--8997--d6b6c631b8f2",
		"/dev/dm-1": "/dev/mapper/data-extra",
		"/dev/dm-2": "/dev/mapper/ceph--dada9f25--41b4--4c26--9a20--448ac01e1d06-osd--block--7d09cceb--4de0--478e--9d8d--bd09cb0c904e",
		"/dev/dm-3": "/dev/mapper/ceph--dada9f25--41b4--4c26--9a20--448ac01e1d06-osd--block--ad76cf53--5cb5--48fe--a39a--343734f5ccde",
		"/dev/dm-6": "/dev/mapper/ceph--c5628abe--ae41--4c3d--bdc6--ef86c54bf78c-osd--block--69481cd1--38b1--42fd--ac07--06bf4d7c0e19",
		"/dev/dm-4": "/dev/mapper/ceph--0e03d5c6--d0e9--4f04--b9af--38d15e14369f-osd--block--61869d90--2c45--4f02--b7c3--96955f41e2ca",
		"/dev/mapper/ceph--cf7c8b53--27c7--4cfc--94de--6ad4c7d9f92d-osd--block--af39b794--e1c6--41c0--8997--d6b6c631b8f2": "/dev/mapper/ceph--cf7c8b53--27c7--4cfc--94de--6ad4c7d9f92d-osd--block--af39b794--e1c6--41c0--8997--d6b6c631b8f2",
		"/dev/mapper/ceph--dada9f25--41b4--4c26--9a20--448ac01e1d06-osd--block--7d09cceb--4de0--478e--9d8d--bd09cb0c904e": "/dev/mapper/ceph--dada9f25--41b4--4c26--9a20--448ac01e1d06-osd--block--7d09cceb--4de0--478e--9d8d--bd09cb0c904e",
		"/dev/mapper/ceph--c5628abe--ae41--4c3d--bdc6--ef86c54bf78c-osd--block--69481cd1--38b1--42fd--ac07--06bf4d7c0e19": "/dev/mapper/ceph--c5628abe--ae41--4c3d--bdc6--ef86c54bf78c-osd--block--69481cd1--38b1--42fd--ac07--06bf4d7c0e19",
		"/dev/mapper/ceph--0e03d5c6--d0e9--4f04--b9af--38d15e14369f-osd--block--61869d90--2c45--4f02--b7c3--96955f41e2ca": "/dev/mapper/ceph--0e03d5c6--d0e9--4f04--b9af--38d15e14369f-osd--block--61869d90--2c45--4f02--b7c3--96955f41e2ca",
		"/dev/mapper/ceph--dada9f25--41b4--4c26--9a20--448ac01e1d06-osd--block--ad76cf53--5cb5--48fe--a39a--343734f5ccde": "/dev/mapper/ceph--dada9f25--41b4--4c26--9a20--448ac01e1d06-osd--block--ad76cf53--5cb5--48fe--a39a--343734f5ccde",
		"/dev/mapper/data-extra": "/dev/mapper/data-extra",
		// by dev name
		"/dev/vda":  "/dev/vda",
		"/dev/vda1": "/dev/vda1",
		"/dev/vdb":  "/dev/vdb",
		"/dev/vdc":  "/dev/vdc",
		"/dev/vdd":  "/dev/vdd",
		"/dev/vde":  "/dev/vde",
		"/dev/vdf":  "/dev/vdf",
		"/dev/vdg":  "/dev/vdg",
	},
}

var DiskInfoReportCephVolumeFromNode2 = &lcmcommon.DiskDaemonDisksReport{
	BlockInfo: DiskInfoReportLsblkFromNode2.BlockInfo,
	Aliases:   DiskInfoReportLsblkFromNode2.Aliases,
	DiskToOsd: map[string][]string{
		"/dev/vdb": {"0"},
		"/dev/vdc": {"0"},
		"/dev/vdd": {"4", "5"},
		"/dev/vde": {"2"},
	},
}

var CephVolumeLvmReportFromNode2 = `{
    "0": [
        {
            "devices": [
                "/dev/vdb"
            ],
            "lv_name": "osd-block-af39b794-e1c6-41c0-8997-d6b6c631b8f2",
            "lv_path": "/dev/ceph-cf7c8b53-27c7-4cfc-94de-6ad4c7d9f92d/osd-block-af39b794-e1c6-41c0-8997-d6b6c631b8f2",
            "lv_size": "53682896896",
            "lv_tags": "ceph.block_device=/dev/ceph-cf7c8b53-27c7-4cfc-94de-6ad4c7d9f92d/osd-block-af39b794-e1c6-41c0-8997-d6b6c631b8f2,ceph.block_uuid=4obO0P-Qjzj-Fste-SCmY-oQva-5Gkk-YQX8fb,ceph.cephx_lockbox_secret=,ceph.cluster_fsid=8668f062-3faa-358a-85f3-f80fe6c1e306,ceph.cluster_name=ceph,ceph.crush_device_class=hdd,ceph.encrypted=0,ceph.osd_fsid=69481cd1-38b1-42fd-ac07-06bf4d7c0e19,ceph.osd_id=0,ceph.osdspec_affinity=,ceph.type=block,ceph.vdo=0",
            "lv_uuid": "4obO0P-Qjzj-Fste-SCmY-oQva-5Gkk-YQX8fb",
            "name": "osd-block-af39b794-e1c6-41c0-8997-d6b6c631b8f2",
            "path": "/dev/ceph-cf7c8b53-27c7-4cfc-94de-6ad4c7d9f92d/osd-block-af39b794-e1c6-41c0-8997-d6b6c631b8f2",
            "tags": {
                "ceph.block_device": "/dev/ceph-cf7c8b53-27c7-4cfc-94de-6ad4c7d9f92d/osd-block-af39b794-e1c6-41c0-8997-d6b6c631b8f2",
                "ceph.block_uuid": "4obO0P-Qjzj-Fste-SCmY-oQva-5Gkk-YQX8fb",
                "ceph.cephx_lockbox_secret": "",
                "ceph.cluster_fsid": "8668f062-3faa-358a-85f3-f80fe6c1e306",
                "ceph.cluster_name": "ceph",
                "ceph.crush_device_class": "hdd",
                "ceph.encrypted": "0",
                "ceph.osd_fsid": "69481cd1-38b1-42fd-ac07-06bf4d7c0e19",
                "ceph.osd_id": "0",
                "ceph.osdspec_affinity": "",
                "ceph.type": "block",
                "ceph.vdo": "0"
            },
            "type": "block",
            "vg_name": "ceph-cf7c8b53-27c7-4cfc-94de-6ad4c7d9f92d"
        },
        {
            "devices": [
                "/dev/vdc"
            ],
            "lv_name": "osd-block-69481cd1-38b1-42fd-ac07-06bf4d7c0e19",
            "lv_path": "/dev/ceph-c5628abe-ae41-4c3d-bdc6-ef86c54bf78c/osd-block-69481cd1-38b1-42fd-ac07-06bf4d7c0e19",
            "lv_size": "26839351296",
            "lv_tags": "ceph.block_device=/dev/ceph-c5628abe-ae41-4c3d-bdc6-ef86c54bf78/osd-block-69481cd1-38b1-42fd-ac07-06bf4d7c0e19,ceph.block_uuid=XUSOFJ-c8YT-qhbL-3SEY-6Le5-Pe0D-s4Pcj2,ceph.cephx_lockbox_secret=,ceph.cluster_fsid=8668f062-3faa-358a-85f3-f80fe6c1e306,ceph.cluster_name=ceph,ceph.crush_device_class=hdd,ceph.encrypted=0,ceph.osd_fsid=06bf4d7c-9603-41a4-b250-284ecf3ecb2f,ceph.osd_id=0,ceph.osdspec_affinity=,ceph.type=block,ceph.vdo=0",
            "lv_uuid": "XUSOFJ-c8YT-qhbL-3SEY-6Le5-Pe0D-s4Pcj2",
            "name": "osd-block-69481cd1-38b1-42fd-ac07-06bf4d7c0e19",
            "path": "/dev/ceph-c5628abe-ae41-4c3d-bdc6-ef86c54bf78c/osd-block-69481cd1-38b1-42fd-ac07-06bf4d7c0e19",
            "tags": {
                "ceph.block_device": "/dev/ceph-c5628abe-ae41-4c3d-bdc6-ef86c54bf78c/osd-block-69481cd1-38b1-42fd-ac07-06bf4d7c0e19",
                "ceph.block_uuid": "XUSOFJ-c8YT-qhbL-3SEY-6Le5-Pe0D-s4Pcj2",
                "ceph.cephx_lockbox_secret": "",
                "ceph.cluster_fsid": "8668f062-0lsk-358a-1gt4-f80fe6c1e306",
                "ceph.cluster_name": "ceph",
                "ceph.crush_device_class": "hdd",
                "ceph.encrypted": "0",
                "ceph.osd_fsid": "06bf4d7c-9603-41a4-b250-284ecf3ecb2f",
                "ceph.osd_id": "0",
                "ceph.osdspec_affinity": "",
                "ceph.type": "block",
                "ceph.vdo": "0"
            },
            "type": "block",
            "vg_name": "ceph-c5628abe-ae41-4c3d-bdc6-ef86c54bf78c"
        }
    ],
    "2": [
        {
            "devices": [
                "/dev/vde"
            ],
            "lv_name": "osd-block-61869d90-2c45-4f02-b7c3-96955f41e2ca",
            "lv_path": "/dev/ceph-0e03d5c6-d0e9-4f04-b9af-38d15e14369f/osd-block-61869d90-2c45-4f02-b7c3-96955f41e2ca",
            "lv_size": "53682896896",
            "lv_tags": "ceph.block_device=/dev/ceph-0e03d5c6-d0e9-4f04-b9af-38d15e14369f/osd-block-61869d90-2c45-4f02-b7c3-96955f41e2ca,ceph.block_uuid=IBtepf-Y49c-5WTc-kZRO-ZMc5-Pswf-BpFriS,ceph.cephx_lockbox_secret=,ceph.cluster_fsid=8668f062-3faa-358a-85f3-f80fe6c1e306,ceph.cluster_name=ceph,ceph.crush_device_class=hdd,ceph.encrypted=0,ceph.osd_fsid=61869d90-2c45-4f02-b7c3-96955f41e2ca,ceph.osd_id=2,ceph.osdspec_affinity=,ceph.type=block,ceph.vdo=0",
            "lv_uuid": "IBtepf-Y49c-5WTc-kZRO-ZMc5-Pswf-BpFriS",
            "name": "osd-block-61869d90-2c45-4f02-b7c3-96955f41e2ca",
            "path": "/dev/ceph-0e03d5c6-d0e9-4f04-b9af-38d15e14369f/osd-block-61869d90-2c45-4f02-b7c3-96955f41e2ca",
            "tags": {
                "ceph.block_device": "/dev/ceph-0e03d5c6-d0e9-4f04-b9af-38d15e14369f/osd-block-61869d90-2c45-4f02-b7c3-96955f41e2ca",
                "ceph.block_uuid": "IBtepf-Y49c-5WTc-kZRO-ZMc5-Pswf-BpFriS",
                "ceph.cephx_lockbox_secret": "",
                "ceph.cluster_fsid": "8668f062-3faa-358a-85f3-f80fe6c1e306",
                "ceph.cluster_name": "ceph",
                "ceph.crush_device_class": "hdd",
                "ceph.encrypted": "0",
                "ceph.osd_fsid": "61869d90-2c45-4f02-b7c3-96955f41e2ca",
                "ceph.osd_id": "2",
                "ceph.osdspec_affinity": "",
                "ceph.type": "block",
                "ceph.vdo": "0"
            },
            "type": "block",
            "vg_name": "ceph-0e03d5c6-d0e9-4f04-b9af-38d15e14369f"
        }
    ],
    "4": [
        {
            "devices": [
                "/dev/vdd"
            ],
            "lv_name": "osd-block-ad76cf53-5cb5-48fe-a39a-343734f5ccde",
            "lv_path": "/dev/ceph-dada9f25-41b4-4c26-9a20-448ac01e1d06/osd-block-ad76cf53-5cb5-48fe-a39a-343734f5ccde",
            "lv_size": "26839351296",
            "lv_tags": "ceph.block_device=/dev/ceph-dada9f25-41b4-4c26-9a20-448ac01e1d06/osd-block-ad76cf53-5cb5-48fe-a39a-343734f5ccde,ceph.block_uuid=XUSOFJ-c8YT-qhbL-3SEY-6Le5-Pe0D-s4Pcj2,ceph.cephx_lockbox_secret=,ceph.cluster_fsid=8668f062-3faa-358a-85f3-f80fe6c1e306,ceph.cluster_name=ceph,ceph.crush_device_class=ssd,ceph.encrypted=0,ceph.osd_fsid=ad76cf53-5cb5-48fe-a39a-343734f5ccde,ceph.osd_id=4,ceph.osdspec_affinity=,ceph.type=block,ceph.vdo=0",
            "lv_uuid": "XUSOFJ-c8YT-qhbL-3SEY-6Le5-Pe0D-s4Pcj2",
            "name": "osd-block-ad76cf53-5cb5-48fe-a39a-343734f5ccde",
            "path": "/dev/ceph-dada9f25-41b4-4c26-9a20-448ac01e1d06/osd-block-ad76cf53-5cb5-48fe-a39a-343734f5ccde",
            "tags": {
                "ceph.block_device": "/dev/ceph-dada9f25-41b4-4c26-9a20-448ac01e1d06/osd-block-ad76cf53-5cb5-48fe-a39a-343734f5ccde",
                "ceph.block_uuid": "XUSOFJ-c8YT-qhbL-3SEY-6Le5-Pe0D-s4Pcj2",
                "ceph.cephx_lockbox_secret": "",
                "ceph.cluster_fsid": "8668f062-3faa-358a-85f3-f80fe6c1e306",
                "ceph.cluster_name": "ceph",
                "ceph.crush_device_class": "ssd",
                "ceph.encrypted": "0",
                "ceph.osd_fsid": "ad76cf53-5cb5-48fe-a39a-343734f5ccde",
                "ceph.osd_id": "4",
                "ceph.osdspec_affinity": "",
                "ceph.type": "block",
                "ceph.vdo": "0"
            },
            "type": "block",
            "vg_name": "ceph-dada9f25-41b4-4c26-9a20-448ac01e1d06"
        }
    ],
    "5": [
        {
            "devices": [
                "/dev/vdd"
            ],
            "lv_name": "osd-block-7d09cceb-4de0-478e-9d8d-bd09cb0c904e",
            "lv_path": "/dev/ceph-dada9f25-41b4-4c26-9a20-448ac01e1d06/osd-block-7d09cceb-4de0-478e-9d8d-bd09cb0c904e",
            "lv_size": "32208060416",
            "lv_tags": "ceph.block_device=/dev/ceph-dada9f25-41b4-4c26-9a20-448ac01e1d06/osd-block-7d09cceb-4de0-478e-9d8d-bd09cb0c904e,ceph.block_uuid=qByj19-Ti9D-S9N1-5e3M-9MJF-k7hR-ahl5G3,ceph.cephx_lockbox_secret=,ceph.cluster_fsid=8668f062-3faa-358a-85f3-f80fe6c1e306,ceph.cluster_name=ceph,ceph.crush_device_class=ssd,ceph.encrypted=0,ceph.osd_fsid=af39b794-e1c6-41c0-8997-d6b6c631b8f2,ceph.osd_id=5,ceph.osdspec_affinity=,ceph.type=block,ceph.vdo=0",
            "lv_uuid": "qByj19-Ti9D-S9N1-5e3M-9MJF-k7hR-ahl5G3",
            "name": "osd-block-7d09cceb-4de0-478e-9d8d-bd09cb0c904e",
            "path": "/dev/ceph-dada9f25-41b4-4c26-9a20-448ac01e1d06/osd-block-7d09cceb-4de0-478e-9d8d-bd09cb0c904e",
            "tags": {
                "ceph.block_device": "/dev/ceph-dada9f25-41b4-4c26-9a20-448ac01e1d06/osd-block-7d09cceb-4de0-478e-9d8d-bd09cb0c904e",
                "ceph.block_uuid": "qByj19-Ti9D-S9N1-5e3M-9MJF-k7hR-ahl5G3",
                "ceph.cephx_lockbox_secret": "",
                "ceph.cluster_fsid": "8668f062-3faa-358a-85f3-f80fe6c1e306",
                "ceph.cluster_name": "ceph",
                "ceph.crush_device_class": "sdd",
                "ceph.encrypted": "0",
                "ceph.osd_fsid": "af39b794-e1c6-41c0-8997-d6b6c631b8f2",
                "ceph.osd_id": "5",
                "ceph.osdspec_affinity": "",
                "ceph.type": "block",
                "ceph.vdo": "0"
            },
            "type": "block",
            "vg_name": "ceph-dada9f25-41b4-4c26-9a20-448ac01e1d06"
        }
    ]
}`

var LvmLvsReportFromNode2 = `{
      "report": [
          {
              "lv": [
                  {"lv_dm_path":"/dev/mapper/data-extra"},
                  {"lv_dm_path":"/dev/mapper/ceph--cf7c8b53--27c7--4cfc--94de--6ad4c7d9f92d-osd--block--af39b794--e1c6--41c0--8997--d6b6c631b8f2"},
                  {"lv_dm_path":"/dev/mapper/ceph--dada9f25--41b4--4c26--9a20--448ac01e1d06-osd--block--7d09cceb--4de0--478e--9d8d--bd09cb0c904e"},
                  {"lv_dm_path":"/dev/mapper/ceph--c5628abe--ae41--4c3d--bdc6--ef86c54bf78c-osd--block--69481cd1--38b1--42fd--ac07--06bf4d7c0e19"},
                  {"lv_dm_path":"/dev/mapper/ceph--0e03d5c6--d0e9--4f04--b9af--38d15e14369f-osd--block--61869d90--2c45--4f02--b7c3--96955f41e2ca"},
                  {"lv_dm_path":"/dev/mapper/ceph--dada9f25--41b4--4c26--9a20--448ac01e1d06-osd--block--ad76cf53--5cb5--48fe--a39a--343734f5ccde"}
              ]
          }
      ]
  }
`

var FoundLvmsFromNode2 = map[string][]string{
	"/dev/mapper/ceph--cf7c8b53--27c7--4cfc--94de--6ad4c7d9f92d-osd--block--af39b794--e1c6--41c0--8997--d6b6c631b8f2": {"/dev/vdb"},
	"/dev/mapper/ceph--c5628abe--ae41--4c3d--bdc6--ef86c54bf78c-osd--block--69481cd1--38b1--42fd--ac07--06bf4d7c0e19": {"/dev/vdc"},
	"/dev/mapper/ceph--dada9f25--41b4--4c26--9a20--448ac01e1d06-osd--block--7d09cceb--4de0--478e--9d8d--bd09cb0c904e": {"/dev/vdd"},
	"/dev/mapper/ceph--dada9f25--41b4--4c26--9a20--448ac01e1d06-osd--block--ad76cf53--5cb5--48fe--a39a--343734f5ccde": {"/dev/vdd"},
	"/dev/mapper/ceph--0e03d5c6--d0e9--4f04--b9af--38d15e14369f-osd--block--61869d90--2c45--4f02--b7c3--96955f41e2ca": {"/dev/vde"},
	"/dev/mapper/data-extra": {"/dev/vdf", "/dev/vdg"},
}
