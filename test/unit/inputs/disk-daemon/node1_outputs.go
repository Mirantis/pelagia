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

var LsblkReportFromNode1 = `{
   "blockdevices": [
      {"name": "/dev/loop0", "kname": "/dev/loop0", "maj:min": "7:0", "rota": true, "type": "loop", "pkname": null, "serial": null},
      {"name": "/dev/loop1", "kname": "/dev/loop1", "maj:min": "7:1", "rota": true, "type": "loop", "pkname": null, "serial": null},
      {"name": "/dev/loop2", "kname": "/dev/loop2", "maj:min": "7:2", "rota": true, "type": "loop", "pkname": null, "serial": null},
      {"name": "/dev/loop3", "kname": "/dev/loop3", "maj:min": "7:3", "rota": true, "type": "loop", "pkname": null, "serial": null},
      {"name": "/dev/loop4", "kname": "/dev/loop4", "maj:min": "7:4", "rota": true, "type": "loop", "pkname": null, "serial": null},
      {"name": "/dev/loop5", "kname": "/dev/loop5", "maj:min": "7:5", "rota": true, "type": "loop", "pkname": null, "serial": null},
      {"name": "/dev/loop6", "kname": "/dev/loop6", "maj:min": "7:6", "rota": true, "type": "loop", "pkname": null, "serial": null},
      {"name": "/dev/loop10", "kname": "/dev/loop10", "maj:min": "7:10", "rota": true, "type": "loop", "pkname": null, "serial": null},
      {"name": "/dev/vda", "kname": "/dev/vda", "maj:min": "8:0", "rota": true, "type": "disk", "pkname": null, "serial": "8dad5ae9-ddf7-40bf-8",
         "children": [
            {"name": "/dev/vda1", "kname": "/dev/vda1", "maj:min": "8:1", "rota": true, "type": "part", "pkname": "/dev/vda", "serial": null},
            {"name": "/dev/vda14", "kname": "/dev/vda14", "maj:min": "8:14", "rota": true, "type": "part", "pkname": "/dev/vda", "serial": null},
            {"name": "/dev/vda15", "kname": "/dev/vda15", "maj:min": "8:15", "rota": true, "type": "part", "pkname": "/dev/vda", "serial": null,
               "children": [
                  {"name": "/dev/md127", "kname": "/dev/md127", "maj:min": "9:127", "rota": true, "type": "part", "pkname": "/dev/vda15", "serial": null,
                     "children": [
                        {"name": "/dev/mapper/vg_root-lv_root", "kname": "/dev/dm-0", "maj:min": "252:0", "rota": true, "type": "lvm", "pkname": "/dev/md127", "serial": null}
                     ]
                  }
               ]
            }
         ]
      },
      {"name": "/dev/vdb", "kname": "/dev/vdb", "maj:min": "8:16", "rota": true, "type": "disk", "pkname": null, "serial": "996ea59f-7f47-4fac-b",
         "children": [
            {"name": "/dev/mapper/ceph--992bbd78--3d8e--4cc3--93dc--eae387309364-osd--block--f4edb5cd--fb1e--4620--9419--3f9a4fcecba5", "kname": "/dev/dm-1", "maj:min": "253:1", "rota": true, "type": "lvm", "pkname": "/dev/vdb", "fstype": "ceph_bluestore", "serial": null}
         ]
      },
      {"name": "/dev/vdc", "kname": "/dev/vdc", "maj:min": "8:32", "rota": true, "type": "disk", "pkname": null, "serial": null},
      {"name": "/dev/vdd", "kname": "/dev/vdd", "maj:min": "8:48", "rota": true, "type": "disk", "pkname": null, "serial": "e8d89e2f-ffc6-4988-9",
         "children": [
            {"name": "/dev/vdd1", "kname": "/dev/vdd1", "maj:min": "8:49", "rota": true, "type": "part", "pkname": "/dev/vdd", "serial": null,
         	   "children": [
	               {"name": "/dev/mapper/ceph--metadata-part--1", "kname": "/dev/dm-2", "maj:min": "253:2", "rota": true, "type": "lvm", "pkname": "/dev/vdd1", "fstype": "ceph_bluestore", "serial": null},
	               {"name": "/dev/mapper/ceph--metadata-part--2", "kname": "/dev/dm-3", "maj:min": "253:3", "rota": true, "type": "lvm", "pkname": "/dev/vdd1", "fstype": "ceph_bluestore", "serial": null}
	            ]
	         }
         ]
      },
      {"name": "/dev/vde", "kname": "/dev/vde", "maj:min": "8:112", "rota": true, "type": "disk", "pkname": null, "serial": "2926ff77-7491-4447-a",
         "children": [
            {"name": "/dev/mapper/ceph--21312wds--sdfv--vs3f--scv3--sdfdsg23edaa-osd--block--vbsgs3a3--sdcv--casq--sd11--asd12dasczsf", "kname": "/dev/dm-5", "maj:min": "253:6", "rota": true, "type": "lvm", "pkname": "/dev/vde", "fstype": "ceph_bluestore", "serial": null}
         ]
      },
      {"name": "/dev/vdf", "kname": "/dev/vdf", "maj:min": "8:80", "rota": true, "type": "disk", "pkname": null, "serial": "b7ea1c8c-89b8-4354-8",
         "children": [
            {"name": "/dev/mapper/ceph--2efce189--afb7--452f--bd32--c73b5017a0da-osd--block--d49fd9bf--d2dd--4c3d--824d--87f3f17ea44a", "kname": "/dev/dm-4", "maj:min": "253:5", "rota": true, "type": "lvm", "pkname": "/dev/vdf", "fstype": "ceph_bluestore", "serial": null}
         ]
      },
      {"name": "/dev/vdh", "kname": "/dev/vdh", "maj:min": "8:96", "rota": true, "type": "disk", "pkname": null, "serial": "cf77cbec-ca01-45d9-a",
         "children": [
            {"name": "/dev/vdh1", "kname": "/dev/vdh1", "maj:min": "8:97", "rota": true, "type": "part", "pkname": "/dev/vdh", "serial": null,
               "children": [
                  {"name": "/dev/md127", "kname": "/dev/md127", "maj:min": "9:127", "rota": true, "type": "part", "pkname": "/dev/vdh1", "serial": null,
                     "children": [
                        {"name": "/dev/mapper/vg_root-lv_root", "kname": "/dev/dm-0", "maj:min": "252:0", "rota": true, "type": "lvm", "pkname": "/dev/md127", "serial": null}
                     ]
                  }
               ]
            }
         ]
      }
   ]
}`

var UdevadmReportFromNode1 = map[string]string{
	// mappers symlinks
	"/dev/mapper/ceph--21312wds--sdfv--vs3f--scv3--sdfdsg23edaa-osd--block--vbsgs3a3--sdcv--casq--sd11--asd12dasczsf": "/dev/disk/by-id/dm-uuid-LVM-oPXPcruZ1AK9dkZOsPR9ZW7PzVb9xtFrOnhN24VqDzKIOPBZLd60UQpS6PpCEzQs /dev/disk/by-id/dm-name-ceph--21312wds--sdfv--vs3f--scv3--sdfdsg23edaa-osd--block--vbsgs3a3--sdcv--casq--sd11--asd12dasczsf",
	"/dev/mapper/ceph--2efce189--afb7--452f--bd32--c73b5017a0da-osd--block--d49fd9bf--d2dd--4c3d--824d--87f3f17ea44a": "/dev/disk/by-id/dm-uuid-LVM-LMoz5X0a3VV3TO2rMomDqfh24zt91NaCiZmlePb5dTd9cws2kHF6Q28W96aUWWgJ /dev/disk/by-id/dm-name-ceph--2efce189--afb7--452f--bd32--c73b5017a0da-osd--block--d49fd9bf--d2dd--4c3d--824d--87f3f17ea44a",
	"/dev/mapper/ceph--992bbd78--3d8e--4cc3--93dc--eae387309364-osd--block--f4edb5cd--fb1e--4620--9419--3f9a4fcecba5": "/dev/disk/by-id/dm-uuid-LVM-VjASpFzahZwHYS2XN4EblEfLAfVwAImtnWhRvxcC38bhRLCw9S8sCCR7JvTuSbco /dev/disk/by-id/dm-name-ceph--992bbd78--3d8e--4cc3--93dc--eae387309364-osd--block--f4edb5cd--fb1e--4620--9419--3f9a4fcecba5",
	"/dev/mapper/ceph--metadata-part--1": "/dev/disk/by-id/dm-name-ceph--metadata-part--1 /dev/disk/by-id/dm-uuid-LVM-4NjWWqNazXbV26cmzMqOasZgbTwEPpZaUW1YZSAnvx7CLqXUAIZ5UKlcZx8w8lWo",
	"/dev/mapper/ceph--metadata-part--2": "/dev/disk/by-id/dm-name-ceph--metadata-part--2 /dev/disk/by-id/dm-uuid-LVM-4NjWWqNazXbV26cmzMqOasZgbTwEPpZaH1waxWke4fbXEDXucEbZNeB4ZDfBeUrW",
	"/dev/mapper/vg_root-lv_root":        "/dev/disk/by-id/dm-name-vg_root-lv_root /dev/disk/by-id/dm-uuid-LVM-hVhQGAaFSKQ12ENRZABVk0nXCAos3JGulscWc97Kr4AQJNmIG0CbWYNy7fSDiVCe /dev/disk/by-uuid/a3387596-cb0c-4b14-ae91-b6992b767f50",
	"/dev/md127":                         "/dev/disk/by-id/lvm-pv-uuid-K7zgwt-1AY8-QxFu-ltxQ-lCnI-XMe0-91XwfL /dev/disk/by-id/md-name-any:md_root /dev/disk/by-id/md-uuid-2fd11014:2ffb43fd:06d3979e:b213232b /dev/md/md_root",
	// devs symlinks
	"/dev/vda":   "/dev/disk/by-path/pci-0000:00:09.0 /dev/disk/by-path/virtio-pci-0000:00:09.0 /dev/disk/by-id/virtio-8dad5ae9-ddf7-40bf-8",
	"/dev/vda1":  "/dev/disk/by-label/cloudimg-rootfs /dev/disk/by-path/virtio-pci-0000:00:09.0-part1 /dev/disk/by-path/pci-0000:00:09.0-part1 /dev/disk/by-partuuid/8a13887f-ec53-4427-b9ec-8291e6213b29 /dev/disk/by-uuid/f51a1ffe-1cc1-4274-a35e-7b6dd027ddc9",
	"/dev/vda14": "/dev/disk/by-partuuid/40dba738-2c45-4236-a681-75198bc111ae /dev/disk/by-path/pci-0000:00:09.0-part14 /dev/disk/by-path/virtio-pci-0000:00:09.0-part14",
	"/dev/vda15": "/dev/disk/by-path/pci-0000:00:09.0-part15 /dev/disk/by-label/UEFI /dev/disk/by-uuid/A82C-5E66 /dev/disk/by-path/virtio-pci-0000:00:09.0-part15 /dev/disk/by-partuuid/ef825b91-d4cc-47b3-bf54-99c78546a9c4",
	"/dev/vdb":   "/dev/disk/by-id/lvm-pv-uuid-yd92Oj-9hBf-2w2n-IEjf-nBJ1-2dMk-kBeMZI /dev/disk/by-path/pci-0000:00:0a.0 /dev/disk/by-path/virtio-pci-0000:00:0a.0 /dev/disk/by-id/virtio-996ea59f-7f47-4fac-b",
	"/dev/vdc":   "/dev/disk/by-uuid/BA42-906E /dev/disk/by-path/pci-0000:00:0b.0 /dev/disk/by-path/virtio-pci-0000:00:0b.0 /dev/disk/by-label/config-2",
	"/dev/vdd":   "/dev/disk/by-path/virtio-pci-0000:00:0e.0 /dev/disk/by-id/virtio-e8d89e2f-ffc6-4988-9 /dev/disk/by-path/pci-0000:00:0e.0",
	"/dev/vdd1":  "/dev/disk/by-path/virtio-pci-0000:00:0e.0-part1 /dev/disk/by-id/virtio-e8d89e2f-ffc6-4988-9-part1 /dev/disk/by-path/pci-0000:00:0e.0-part1 /dev/disk/by-id/lvm-pv-uuid-7nUuVo-Zpzv-Tqze-5rtG-Y8f0-HdvQ-m6WXIU",
	"/dev/vde":   "/dev/disk/by-id/lvm-pv-uuid-nzJOk1-kLTM-ErxQ-0N4c-DpDU-0zhE-Q9hRJP /dev/disk/by-id/virtio-2926ff77-7491-4447-a /dev/disk/by-path/pci-0000:00:0f.0 /dev/disk/by-path/virtio-pci-0000:00:0f.0",
	"/dev/vdf":   "/dev/disk/by-path/pci-0000:00:10.0 /dev/disk/by-id/lvm-pv-uuid-fZ7Efo-X0nc-lAR3-lzik-MjMT-0rml-lZNf7b /dev/disk/by-path/virtio-pci-0000:00:10.0 /dev/disk/by-id/virtio-b7ea1c8c-89b8-4354-8",
	"/dev/vdh":   "/dev/disk/by-path/pci-0000:00:11.0 /dev/disk/by-id/lvm-pv-uuid-gN4hiQ-gqT4-V19I-kvfA-fHWf-YIsh-gPFLTB /dev/disk/by-id/virtio-cf77cbec-ca01-45d9-a /dev/disk/by-path/virtio-pci-0000:00:11.0",
	"/dev/vdh1":  "/dev/disk/by-path/virtio-pci-0000:00:11.0-part1 /dev/disk/by-path/pci-0000:00:11.0-part1 /dev/disk/by-partuuid/cf77cbec-ec53-4427-45d9-8291e6213b29 /dev/disk/by-uuid/8a13887f-1cc1-4427-ec53-7b6dd027ddc9",
}

var DiskInfoReportLsblkFromNode1 = &lcmcommon.DiskDaemonDisksReport{
	BlockInfo: map[string]lcmcommon.BlockDeviceInfo{
		"/dev/mapper/ceph--2efce189--afb7--452f--bd32--c73b5017a0da-osd--block--d49fd9bf--d2dd--4c3d--824d--87f3f17ea44a": {
			Kname: "/dev/dm-4", Type: "lvm", Rotational: true, MajMin: "253:5", Parent: []string{"/dev/vdf"},
			Symlinks: []string{
				"/dev/disk/by-id/dm-name-ceph--2efce189--afb7--452f--bd32--c73b5017a0da-osd--block--d49fd9bf--d2dd--4c3d--824d--87f3f17ea44a",
				"/dev/disk/by-id/dm-uuid-LVM-LMoz5X0a3VV3TO2rMomDqfh24zt91NaCiZmlePb5dTd9cws2kHF6Q28W96aUWWgJ",
			},
			Childrens: []string{},
		},
		"/dev/mapper/ceph--992bbd78--3d8e--4cc3--93dc--eae387309364-osd--block--f4edb5cd--fb1e--4620--9419--3f9a4fcecba5": {
			Kname: "/dev/dm-1", Type: "lvm", Rotational: true, MajMin: "253:1", Parent: []string{"/dev/vdb"},
			Symlinks: []string{
				"/dev/disk/by-id/dm-name-ceph--992bbd78--3d8e--4cc3--93dc--eae387309364-osd--block--f4edb5cd--fb1e--4620--9419--3f9a4fcecba5",
				"/dev/disk/by-id/dm-uuid-LVM-VjASpFzahZwHYS2XN4EblEfLAfVwAImtnWhRvxcC38bhRLCw9S8sCCR7JvTuSbco",
			},
			Childrens: []string{},
		},
		"/dev/mapper/ceph--21312wds--sdfv--vs3f--scv3--sdfdsg23edaa-osd--block--vbsgs3a3--sdcv--casq--sd11--asd12dasczsf": {
			Kname: "/dev/dm-5", Type: "lvm", Rotational: true, MajMin: "253:6", Parent: []string{"/dev/vde"},
			Symlinks: []string{
				"/dev/disk/by-id/dm-name-ceph--21312wds--sdfv--vs3f--scv3--sdfdsg23edaa-osd--block--vbsgs3a3--sdcv--casq--sd11--asd12dasczsf",
				"/dev/disk/by-id/dm-uuid-LVM-oPXPcruZ1AK9dkZOsPR9ZW7PzVb9xtFrOnhN24VqDzKIOPBZLd60UQpS6PpCEzQs",
			},
			Childrens: []string{},
		},
		"/dev/mapper/ceph--metadata-part--1": {
			Kname: "/dev/dm-2", Type: "lvm", Rotational: true, MajMin: "253:2", Parent: []string{"/dev/vdd1"},
			Symlinks: []string{
				"/dev/disk/by-id/dm-name-ceph--metadata-part--1",
				"/dev/disk/by-id/dm-uuid-LVM-4NjWWqNazXbV26cmzMqOasZgbTwEPpZaUW1YZSAnvx7CLqXUAIZ5UKlcZx8w8lWo",
			},
			Childrens: []string{},
		},
		"/dev/mapper/ceph--metadata-part--2": {
			Kname: "/dev/dm-3", Type: "lvm", Rotational: true, MajMin: "253:3", Parent: []string{"/dev/vdd1"},
			Symlinks: []string{
				"/dev/disk/by-id/dm-name-ceph--metadata-part--2",
				"/dev/disk/by-id/dm-uuid-LVM-4NjWWqNazXbV26cmzMqOasZgbTwEPpZaH1waxWke4fbXEDXucEbZNeB4ZDfBeUrW",
			},
			Childrens: []string{},
		},
		"/dev/mapper/vg_root-lv_root": {
			Kname: "/dev/dm-0", Type: "lvm", Rotational: true, MajMin: "252:0", Parent: []string{"/dev/md127"},
			Symlinks: []string{
				"/dev/disk/by-id/dm-name-vg_root-lv_root",
				"/dev/disk/by-id/dm-uuid-LVM-hVhQGAaFSKQ12ENRZABVk0nXCAos3JGulscWc97Kr4AQJNmIG0CbWYNy7fSDiVCe",
				"/dev/disk/by-uuid/a3387596-cb0c-4b14-ae91-b6992b767f50",
			},
			Childrens: []string{},
		},
		"/dev/md127": {
			Kname: "/dev/md127", Type: "part", Rotational: true, MajMin: "9:127", Parent: []string{"/dev/vda15", "/dev/vdh1"},
			Symlinks: []string{
				"/dev/disk/by-id/lvm-pv-uuid-K7zgwt-1AY8-QxFu-ltxQ-lCnI-XMe0-91XwfL",
				"/dev/disk/by-id/md-name-any:md_root",
				"/dev/disk/by-id/md-uuid-2fd11014:2ffb43fd:06d3979e:b213232b",
				"/dev/md/md_root",
			},
			Childrens: []string{"/dev/mapper/vg_root-lv_root"},
		},
		"/dev/vda": {
			Kname: "/dev/vda", Type: "disk", Rotational: true, MajMin: "8:0", Parent: []string{""}, Serial: "8dad5ae9-ddf7-40bf-8",
			Symlinks: []string{
				"/dev/disk/by-id/virtio-8dad5ae9-ddf7-40bf-8",
				"/dev/disk/by-path/pci-0000:00:09.0",
				"/dev/disk/by-path/virtio-pci-0000:00:09.0",
			},
			Childrens: []string{"/dev/vda1", "/dev/vda14", "/dev/vda15"},
		},
		"/dev/vda1": {
			Kname: "/dev/vda1", Type: "part", Rotational: true, MajMin: "8:1", Parent: []string{"/dev/vda"},
			Symlinks: []string{
				"/dev/disk/by-label/cloudimg-rootfs",
				"/dev/disk/by-partuuid/8a13887f-ec53-4427-b9ec-8291e6213b29",
				"/dev/disk/by-path/pci-0000:00:09.0-part1",
				"/dev/disk/by-path/virtio-pci-0000:00:09.0-part1",
				"/dev/disk/by-uuid/f51a1ffe-1cc1-4274-a35e-7b6dd027ddc9",
			},
			Childrens: []string{},
		},
		"/dev/vda14": {
			Kname: "/dev/vda14", Type: "part", Rotational: true, MajMin: "8:14", Parent: []string{"/dev/vda"},
			Symlinks: []string{
				"/dev/disk/by-partuuid/40dba738-2c45-4236-a681-75198bc111ae",
				"/dev/disk/by-path/pci-0000:00:09.0-part14",
				"/dev/disk/by-path/virtio-pci-0000:00:09.0-part14",
			},
			Childrens: []string{},
		},
		"/dev/vda15": {
			Kname: "/dev/vda15", Type: "part", Rotational: true, MajMin: "8:15", Parent: []string{"/dev/vda"},
			Symlinks: []string{
				"/dev/disk/by-label/UEFI",
				"/dev/disk/by-partuuid/ef825b91-d4cc-47b3-bf54-99c78546a9c4",
				"/dev/disk/by-path/pci-0000:00:09.0-part15",
				"/dev/disk/by-path/virtio-pci-0000:00:09.0-part15",
				"/dev/disk/by-uuid/A82C-5E66",
			},
			Childrens: []string{"/dev/md127"},
		},
		"/dev/vdb": {
			Kname: "/dev/vdb", Type: "disk", Rotational: true, MajMin: "8:16", Parent: []string{""}, Serial: "996ea59f-7f47-4fac-b",
			Symlinks: []string{
				"/dev/disk/by-id/lvm-pv-uuid-yd92Oj-9hBf-2w2n-IEjf-nBJ1-2dMk-kBeMZI",
				"/dev/disk/by-id/virtio-996ea59f-7f47-4fac-b",
				"/dev/disk/by-path/pci-0000:00:0a.0",
				"/dev/disk/by-path/virtio-pci-0000:00:0a.0",
			},
			Childrens: []string{"/dev/mapper/ceph--992bbd78--3d8e--4cc3--93dc--eae387309364-osd--block--f4edb5cd--fb1e--4620--9419--3f9a4fcecba5"},
		},
		"/dev/vdc": {
			Kname: "/dev/vdc", Type: "disk", Rotational: true, MajMin: "8:32", Parent: []string{""},
			Symlinks: []string{
				"/dev/disk/by-label/config-2",
				"/dev/disk/by-path/pci-0000:00:0b.0",
				"/dev/disk/by-path/virtio-pci-0000:00:0b.0",
				"/dev/disk/by-uuid/BA42-906E",
			},
			Childrens: []string{},
		},
		"/dev/vdd": {
			Kname: "/dev/vdd", Type: "disk", Rotational: true, MajMin: "8:48", Parent: []string{""}, Serial: "e8d89e2f-ffc6-4988-9",
			Symlinks: []string{
				"/dev/disk/by-id/virtio-e8d89e2f-ffc6-4988-9",
				"/dev/disk/by-path/pci-0000:00:0e.0",
				"/dev/disk/by-path/virtio-pci-0000:00:0e.0",
			},
			Childrens: []string{"/dev/vdd1"},
		},
		"/dev/vdd1": {
			Kname: "/dev/vdd1", Type: "part", Rotational: true, MajMin: "8:49", Parent: []string{"/dev/vdd"}, Serial: "",
			Symlinks: []string{
				"/dev/disk/by-id/lvm-pv-uuid-7nUuVo-Zpzv-Tqze-5rtG-Y8f0-HdvQ-m6WXIU",
				"/dev/disk/by-id/virtio-e8d89e2f-ffc6-4988-9-part1",
				"/dev/disk/by-path/pci-0000:00:0e.0-part1",
				"/dev/disk/by-path/virtio-pci-0000:00:0e.0-part1",
			},
			Childrens: []string{"/dev/mapper/ceph--metadata-part--1", "/dev/mapper/ceph--metadata-part--2"},
		},
		"/dev/vde": {
			Kname: "/dev/vde", Type: "disk", Rotational: true, MajMin: "8:112", Parent: []string{""}, Serial: "2926ff77-7491-4447-a",
			Symlinks: []string{
				"/dev/disk/by-id/lvm-pv-uuid-nzJOk1-kLTM-ErxQ-0N4c-DpDU-0zhE-Q9hRJP",
				"/dev/disk/by-id/virtio-2926ff77-7491-4447-a",
				"/dev/disk/by-path/pci-0000:00:0f.0",
				"/dev/disk/by-path/virtio-pci-0000:00:0f.0",
			},
			Childrens: []string{
				"/dev/mapper/ceph--21312wds--sdfv--vs3f--scv3--sdfdsg23edaa-osd--block--vbsgs3a3--sdcv--casq--sd11--asd12dasczsf",
			},
		},
		"/dev/vdf": {
			Kname: "/dev/vdf", Type: "disk", Rotational: true, MajMin: "8:80", Parent: []string{""}, Serial: "b7ea1c8c-89b8-4354-8",
			Symlinks: []string{
				"/dev/disk/by-id/lvm-pv-uuid-fZ7Efo-X0nc-lAR3-lzik-MjMT-0rml-lZNf7b",
				"/dev/disk/by-id/virtio-b7ea1c8c-89b8-4354-8",
				"/dev/disk/by-path/pci-0000:00:10.0",
				"/dev/disk/by-path/virtio-pci-0000:00:10.0",
			},
			Childrens: []string{"/dev/mapper/ceph--2efce189--afb7--452f--bd32--c73b5017a0da-osd--block--d49fd9bf--d2dd--4c3d--824d--87f3f17ea44a"},
		},
		"/dev/vdh": {
			Kname: "/dev/vdh", Type: "disk", Rotational: true, MajMin: "8:96", Parent: []string{""}, Serial: "cf77cbec-ca01-45d9-a",
			Symlinks: []string{
				"/dev/disk/by-id/lvm-pv-uuid-gN4hiQ-gqT4-V19I-kvfA-fHWf-YIsh-gPFLTB",
				"/dev/disk/by-id/virtio-cf77cbec-ca01-45d9-a",
				"/dev/disk/by-path/pci-0000:00:11.0",
				"/dev/disk/by-path/virtio-pci-0000:00:11.0",
			},
			Childrens: []string{"/dev/vdh1"},
		},
		"/dev/vdh1": {
			Kname: "/dev/vdh1", Type: "part", Rotational: true, MajMin: "8:97", Parent: []string{"/dev/vdh"},
			Symlinks: []string{
				"/dev/disk/by-partuuid/cf77cbec-ec53-4427-45d9-8291e6213b29",
				"/dev/disk/by-path/pci-0000:00:11.0-part1",
				"/dev/disk/by-path/virtio-pci-0000:00:11.0-part1",
				"/dev/disk/by-uuid/8a13887f-1cc1-4427-ec53-7b6dd027ddc9",
			},
			Childrens: []string{"/dev/md127"},
		},
	},
	Aliases: map[string]string{
		// lvm partition pathes used in ceph-volumes
		"/dev/ceph-21312wds-sdfv-vs3f-scv3-sdfdsg23edaa/osd-block-vbsgs3a3-sdcv-casq-sd11-asd12dasczsf": "/dev/mapper/ceph--21312wds--sdfv--vs3f--scv3--sdfdsg23edaa-osd--block--vbsgs3a3--sdcv--casq--sd11--asd12dasczsf",
		"/dev/ceph-2efce189-afb7-452f-bd32-c73b5017a0da/osd-block-d49fd9bf-d2dd-4c3d-824d-87f3f17ea44a": "/dev/mapper/ceph--2efce189--afb7--452f--bd32--c73b5017a0da-osd--block--d49fd9bf--d2dd--4c3d--824d--87f3f17ea44a",
		"/dev/ceph-992bbd78-3d8e-4cc3-93dc-eae387309364/osd-block-f4edb5cd-fb1e-4620-9419-3f9a4fcecba5": "/dev/mapper/ceph--992bbd78--3d8e--4cc3--93dc--eae387309364-osd--block--f4edb5cd--fb1e--4620--9419--3f9a4fcecba5",
		"/dev/ceph-metadata/part-1": "/dev/mapper/ceph--metadata-part--1",
		"/dev/ceph-metadata/part-2": "/dev/mapper/ceph--metadata-part--2",
		"/dev/vg_root/lv_root":      "/dev/mapper/vg_root-lv_root",
		// disk path by-id
		"/dev/disk/by-id/dm-name-ceph--2efce189--afb7--452f--bd32--c73b5017a0da-osd--block--d49fd9bf--d2dd--4c3d--824d--87f3f17ea44a": "/dev/mapper/ceph--2efce189--afb7--452f--bd32--c73b5017a0da-osd--block--d49fd9bf--d2dd--4c3d--824d--87f3f17ea44a",
		"/dev/disk/by-id/dm-name-ceph--21312wds--sdfv--vs3f--scv3--sdfdsg23edaa-osd--block--vbsgs3a3--sdcv--casq--sd11--asd12dasczsf": "/dev/mapper/ceph--21312wds--sdfv--vs3f--scv3--sdfdsg23edaa-osd--block--vbsgs3a3--sdcv--casq--sd11--asd12dasczsf",
		"/dev/disk/by-id/dm-name-ceph--992bbd78--3d8e--4cc3--93dc--eae387309364-osd--block--f4edb5cd--fb1e--4620--9419--3f9a4fcecba5": "/dev/mapper/ceph--992bbd78--3d8e--4cc3--93dc--eae387309364-osd--block--f4edb5cd--fb1e--4620--9419--3f9a4fcecba5",
		"/dev/disk/by-id/dm-name-ceph--metadata-part--1":                                               "/dev/mapper/ceph--metadata-part--1",
		"/dev/disk/by-id/dm-name-ceph--metadata-part--2":                                               "/dev/mapper/ceph--metadata-part--2",
		"/dev/disk/by-id/dm-name-vg_root-lv_root":                                                      "/dev/mapper/vg_root-lv_root",
		"/dev/disk/by-id/dm-uuid-LVM-4NjWWqNazXbV26cmzMqOasZgbTwEPpZaH1waxWke4fbXEDXucEbZNeB4ZDfBeUrW": "/dev/mapper/ceph--metadata-part--2",
		"/dev/disk/by-id/dm-uuid-LVM-4NjWWqNazXbV26cmzMqOasZgbTwEPpZaUW1YZSAnvx7CLqXUAIZ5UKlcZx8w8lWo": "/dev/mapper/ceph--metadata-part--1",
		"/dev/disk/by-id/dm-uuid-LVM-LMoz5X0a3VV3TO2rMomDqfh24zt91NaCiZmlePb5dTd9cws2kHF6Q28W96aUWWgJ": "/dev/mapper/ceph--2efce189--afb7--452f--bd32--c73b5017a0da-osd--block--d49fd9bf--d2dd--4c3d--824d--87f3f17ea44a",
		"/dev/disk/by-id/dm-uuid-LVM-VjASpFzahZwHYS2XN4EblEfLAfVwAImtnWhRvxcC38bhRLCw9S8sCCR7JvTuSbco": "/dev/mapper/ceph--992bbd78--3d8e--4cc3--93dc--eae387309364-osd--block--f4edb5cd--fb1e--4620--9419--3f9a4fcecba5",
		"/dev/disk/by-id/dm-uuid-LVM-hVhQGAaFSKQ12ENRZABVk0nXCAos3JGulscWc97Kr4AQJNmIG0CbWYNy7fSDiVCe": "/dev/mapper/vg_root-lv_root",
		"/dev/disk/by-id/dm-uuid-LVM-oPXPcruZ1AK9dkZOsPR9ZW7PzVb9xtFrOnhN24VqDzKIOPBZLd60UQpS6PpCEzQs": "/dev/mapper/ceph--21312wds--sdfv--vs3f--scv3--sdfdsg23edaa-osd--block--vbsgs3a3--sdcv--casq--sd11--asd12dasczsf",
		"/dev/disk/by-id/lvm-pv-uuid-7nUuVo-Zpzv-Tqze-5rtG-Y8f0-HdvQ-m6WXIU":                           "/dev/vdd1",
		"/dev/disk/by-id/lvm-pv-uuid-K7zgwt-1AY8-QxFu-ltxQ-lCnI-XMe0-91XwfL":                           "/dev/md127",
		"/dev/disk/by-id/lvm-pv-uuid-fZ7Efo-X0nc-lAR3-lzik-MjMT-0rml-lZNf7b":                           "/dev/vdf",
		"/dev/disk/by-id/lvm-pv-uuid-gN4hiQ-gqT4-V19I-kvfA-fHWf-YIsh-gPFLTB":                           "/dev/vdh",
		"/dev/disk/by-id/lvm-pv-uuid-nzJOk1-kLTM-ErxQ-0N4c-DpDU-0zhE-Q9hRJP":                           "/dev/vde",
		"/dev/disk/by-id/lvm-pv-uuid-yd92Oj-9hBf-2w2n-IEjf-nBJ1-2dMk-kBeMZI":                           "/dev/vdb",
		"/dev/disk/by-id/md-name-any:md_root":                                                          "/dev/md127",
		"/dev/disk/by-id/md-uuid-2fd11014:2ffb43fd:06d3979e:b213232b":                                  "/dev/md127",
		"/dev/disk/by-id/virtio-2926ff77-7491-4447-a":                                                  "/dev/vde",
		"/dev/disk/by-id/virtio-8dad5ae9-ddf7-40bf-8":                                                  "/dev/vda",
		"/dev/disk/by-id/virtio-996ea59f-7f47-4fac-b":                                                  "/dev/vdb",
		"/dev/disk/by-id/virtio-b7ea1c8c-89b8-4354-8":                                                  "/dev/vdf",
		"/dev/disk/by-id/virtio-cf77cbec-ca01-45d9-a":                                                  "/dev/vdh",
		"/dev/disk/by-id/virtio-e8d89e2f-ffc6-4988-9":                                                  "/dev/vdd",
		"/dev/disk/by-id/virtio-e8d89e2f-ffc6-4988-9-part1":                                            "/dev/vdd1",
		// disk path by label
		"/dev/disk/by-label/UEFI":            "/dev/vda15",
		"/dev/disk/by-label/cloudimg-rootfs": "/dev/vda1",
		"/dev/disk/by-label/config-2":        "/dev/vdc",
		// disk path by part uuid
		"/dev/disk/by-partuuid/40dba738-2c45-4236-a681-75198bc111ae": "/dev/vda14",
		"/dev/disk/by-partuuid/8a13887f-ec53-4427-b9ec-8291e6213b29": "/dev/vda1",
		"/dev/disk/by-partuuid/cf77cbec-ec53-4427-45d9-8291e6213b29": "/dev/vdh1",
		"/dev/disk/by-partuuid/ef825b91-d4cc-47b3-bf54-99c78546a9c4": "/dev/vda15",
		// disk path by path
		"/dev/disk/by-path/pci-0000:00:09.0":               "/dev/vda",
		"/dev/disk/by-path/pci-0000:00:09.0-part1":         "/dev/vda1",
		"/dev/disk/by-path/pci-0000:00:09.0-part14":        "/dev/vda14",
		"/dev/disk/by-path/pci-0000:00:09.0-part15":        "/dev/vda15",
		"/dev/disk/by-path/pci-0000:00:0a.0":               "/dev/vdb",
		"/dev/disk/by-path/pci-0000:00:0b.0":               "/dev/vdc",
		"/dev/disk/by-path/pci-0000:00:0e.0":               "/dev/vdd",
		"/dev/disk/by-path/pci-0000:00:0e.0-part1":         "/dev/vdd1",
		"/dev/disk/by-path/pci-0000:00:0f.0":               "/dev/vde",
		"/dev/disk/by-path/pci-0000:00:10.0":               "/dev/vdf",
		"/dev/disk/by-path/pci-0000:00:11.0":               "/dev/vdh",
		"/dev/disk/by-path/pci-0000:00:11.0-part1":         "/dev/vdh1",
		"/dev/disk/by-path/virtio-pci-0000:00:09.0":        "/dev/vda",
		"/dev/disk/by-path/virtio-pci-0000:00:09.0-part1":  "/dev/vda1",
		"/dev/disk/by-path/virtio-pci-0000:00:09.0-part14": "/dev/vda14",
		"/dev/disk/by-path/virtio-pci-0000:00:09.0-part15": "/dev/vda15",
		"/dev/disk/by-path/virtio-pci-0000:00:0a.0":        "/dev/vdb",
		"/dev/disk/by-path/virtio-pci-0000:00:0b.0":        "/dev/vdc",
		"/dev/disk/by-path/virtio-pci-0000:00:0e.0":        "/dev/vdd",
		"/dev/disk/by-path/virtio-pci-0000:00:0e.0-part1":  "/dev/vdd1",
		"/dev/disk/by-path/virtio-pci-0000:00:0f.0":        "/dev/vde",
		"/dev/disk/by-path/virtio-pci-0000:00:10.0":        "/dev/vdf",
		"/dev/disk/by-path/virtio-pci-0000:00:11.0":        "/dev/vdh",
		"/dev/disk/by-path/virtio-pci-0000:00:11.0-part1":  "/dev/vdh1",
		// disk path by uuid
		"/dev/disk/by-uuid/8a13887f-1cc1-4427-ec53-7b6dd027ddc9": "/dev/vdh1",
		"/dev/disk/by-uuid/A82C-5E66":                            "/dev/vda15",
		"/dev/disk/by-uuid/BA42-906E":                            "/dev/vdc",
		"/dev/disk/by-uuid/a3387596-cb0c-4b14-ae91-b6992b767f50": "/dev/mapper/vg_root-lv_root",
		"/dev/disk/by-uuid/f51a1ffe-1cc1-4274-a35e-7b6dd027ddc9": "/dev/vda1",
		// device mapper aliases
		"/dev/dm-0": "/dev/mapper/vg_root-lv_root",
		"/dev/dm-1": "/dev/mapper/ceph--992bbd78--3d8e--4cc3--93dc--eae387309364-osd--block--f4edb5cd--fb1e--4620--9419--3f9a4fcecba5",
		"/dev/dm-2": "/dev/mapper/ceph--metadata-part--1",
		"/dev/dm-3": "/dev/mapper/ceph--metadata-part--2",
		"/dev/dm-4": "/dev/mapper/ceph--2efce189--afb7--452f--bd32--c73b5017a0da-osd--block--d49fd9bf--d2dd--4c3d--824d--87f3f17ea44a",
		"/dev/dm-5": "/dev/mapper/ceph--21312wds--sdfv--vs3f--scv3--sdfdsg23edaa-osd--block--vbsgs3a3--sdcv--casq--sd11--asd12dasczsf",
		"/dev/mapper/ceph--21312wds--sdfv--vs3f--scv3--sdfdsg23edaa-osd--block--vbsgs3a3--sdcv--casq--sd11--asd12dasczsf": "/dev/mapper/ceph--21312wds--sdfv--vs3f--scv3--sdfdsg23edaa-osd--block--vbsgs3a3--sdcv--casq--sd11--asd12dasczsf",
		"/dev/mapper/ceph--2efce189--afb7--452f--bd32--c73b5017a0da-osd--block--d49fd9bf--d2dd--4c3d--824d--87f3f17ea44a": "/dev/mapper/ceph--2efce189--afb7--452f--bd32--c73b5017a0da-osd--block--d49fd9bf--d2dd--4c3d--824d--87f3f17ea44a",
		"/dev/mapper/ceph--992bbd78--3d8e--4cc3--93dc--eae387309364-osd--block--f4edb5cd--fb1e--4620--9419--3f9a4fcecba5": "/dev/mapper/ceph--992bbd78--3d8e--4cc3--93dc--eae387309364-osd--block--f4edb5cd--fb1e--4620--9419--3f9a4fcecba5",
		"/dev/mapper/ceph--metadata-part--1": "/dev/mapper/ceph--metadata-part--1",
		"/dev/mapper/ceph--metadata-part--2": "/dev/mapper/ceph--metadata-part--2",
		"/dev/mapper/vg_root-lv_root":        "/dev/mapper/vg_root-lv_root",
		// raid
		"/dev/md/md_root": "/dev/md127",
		"/dev/md127":      "/dev/md127",
		// by dev name
		"/dev/vda":   "/dev/vda",
		"/dev/vda1":  "/dev/vda1",
		"/dev/vda14": "/dev/vda14",
		"/dev/vda15": "/dev/vda15",
		"/dev/vdb":   "/dev/vdb",
		"/dev/vdc":   "/dev/vdc",
		"/dev/vdd":   "/dev/vdd",
		"/dev/vdd1":  "/dev/vdd1",
		"/dev/vde":   "/dev/vde",
		"/dev/vdf":   "/dev/vdf",
		"/dev/vdh":   "/dev/vdh",
		"/dev/vdh1":  "/dev/vdh1",
	},
}

var DiskInfoReportCephVolumeFromNode1 = &lcmcommon.DiskDaemonDisksReport{
	BlockInfo: DiskInfoReportLsblkFromNode1.BlockInfo,
	Aliases:   DiskInfoReportLsblkFromNode1.Aliases,
	DiskToOsd: map[string][]string{
		"/dev/vda": {"30"},
		"/dev/vdb": {"30"},
		"/dev/vdd": {"20", "25"},
		"/dev/vde": {"20"},
		"/dev/vdf": {"25"},
	},
}

var DiskInfoReportCephVolumeSomeOsdLostFromNode1 = &lcmcommon.DiskDaemonDisksReport{
	BlockInfo: map[string]lcmcommon.BlockDeviceInfo{
		"/dev/mapper/ceph--2efce189--afb7--452f--bd32--c73b5017a0da-osd--block--d49fd9bf--d2dd--4c3d--824d--87f3f17ea44a": {
			Kname: "/dev/dm-4", Type: "lvm", Rotational: true, MajMin: "253:5", Parent: []string{"/dev/vdf"},
			Symlinks: []string{
				"/dev/disk/by-id/dm-name-ceph--2efce189--afb7--452f--bd32--c73b5017a0da-osd--block--d49fd9bf--d2dd--4c3d--824d--87f3f17ea44a",
				"/dev/disk/by-id/dm-uuid-LVM-LMoz5X0a3VV3TO2rMomDqfh24zt91NaCiZmlePb5dTd9cws2kHF6Q28W96aUWWgJ",
			},
			Childrens: []string{},
		},
		"/dev/mapper/ceph--992bbd78--3d8e--4cc3--93dc--eae387309364-osd--block--f4edb5cd--fb1e--4620--9419--3f9a4fcecba5": {
			Kname: "/dev/dm-1", Type: "lvm", Rotational: true, MajMin: "253:1", Parent: []string{"/dev/vdb"},
			Symlinks: []string{
				"/dev/disk/by-id/dm-name-ceph--992bbd78--3d8e--4cc3--93dc--eae387309364-osd--block--f4edb5cd--fb1e--4620--9419--3f9a4fcecba5",
				"/dev/disk/by-id/dm-uuid-LVM-VjASpFzahZwHYS2XN4EblEfLAfVwAImtnWhRvxcC38bhRLCw9S8sCCR7JvTuSbco",
			},
			Childrens: []string{},
		},
		"/dev/mapper/ceph--21312wds--sdfv--vs3f--scv3--sdfdsg23edaa-osd--block--vbsgs3a3--sdcv--casq--sd11--asd12dasczsf": {
			Kname: "/dev/dm-5", Type: "lvm", Rotational: true, MajMin: "253:6", Parent: []string{"/dev/vde"},
			Symlinks: []string{
				"/dev/disk/by-id/dm-name-ceph--21312wds--sdfv--vs3f--scv3--sdfdsg23edaa-osd--block--vbsgs3a3--sdcv--casq--sd11--asd12dasczsf",
				"/dev/disk/by-id/dm-uuid-LVM-oPXPcruZ1AK9dkZOsPR9ZW7PzVb9xtFrOnhN24VqDzKIOPBZLd60UQpS6PpCEzQs",
			},
			Childrens: []string{},
		},
		"/dev/mapper/vg_root-lv_root": {
			Kname: "/dev/dm-0", Type: "lvm", Rotational: true, MajMin: "252:0", Parent: []string{"/dev/md127"},
			Symlinks: []string{
				"/dev/disk/by-id/dm-name-vg_root-lv_root",
				"/dev/disk/by-id/dm-uuid-LVM-hVhQGAaFSKQ12ENRZABVk0nXCAos3JGulscWc97Kr4AQJNmIG0CbWYNy7fSDiVCe",
				"/dev/disk/by-uuid/a3387596-cb0c-4b14-ae91-b6992b767f50",
			},
			Childrens: []string{},
		},
		"/dev/md127": {
			Kname: "/dev/md127", Type: "part", Rotational: true, MajMin: "9:127", Parent: []string{"/dev/vda15", "/dev/vdh1"},
			Symlinks: []string{
				"/dev/disk/by-id/lvm-pv-uuid-K7zgwt-1AY8-QxFu-ltxQ-lCnI-XMe0-91XwfL",
				"/dev/disk/by-id/md-name-any:md_root",
				"/dev/disk/by-id/md-uuid-2fd11014:2ffb43fd:06d3979e:b213232b",
				"/dev/md/md_root",
			},
			Childrens: []string{"/dev/mapper/vg_root-lv_root"},
		},
		"/dev/vda": {
			Kname: "/dev/vda", Type: "disk", Rotational: true, MajMin: "8:0", Parent: []string{""}, Serial: "8dad5ae9-ddf7-40bf-8",
			Symlinks: []string{
				"/dev/disk/by-id/virtio-8dad5ae9-ddf7-40bf-8",
				"/dev/disk/by-path/pci-0000:00:09.0",
				"/dev/disk/by-path/virtio-pci-0000:00:09.0",
			},
			Childrens: []string{"/dev/vda1", "/dev/vda14", "/dev/vda15"},
		},
		"/dev/vda1": {
			Kname: "/dev/vda1", Type: "part", Rotational: true, MajMin: "8:1", Parent: []string{"/dev/vda"},
			Symlinks: []string{
				"/dev/disk/by-label/cloudimg-rootfs",
				"/dev/disk/by-partuuid/8a13887f-ec53-4427-b9ec-8291e6213b29",
				"/dev/disk/by-path/pci-0000:00:09.0-part1",
				"/dev/disk/by-path/virtio-pci-0000:00:09.0-part1",
				"/dev/disk/by-uuid/f51a1ffe-1cc1-4274-a35e-7b6dd027ddc9",
			},
			Childrens: []string{},
		},
		"/dev/vda14": {
			Kname: "/dev/vda14", Type: "part", Rotational: true, MajMin: "8:14", Parent: []string{"/dev/vda"},
			Symlinks: []string{
				"/dev/disk/by-partuuid/40dba738-2c45-4236-a681-75198bc111ae",
				"/dev/disk/by-path/pci-0000:00:09.0-part14",
				"/dev/disk/by-path/virtio-pci-0000:00:09.0-part14",
			},
			Childrens: []string{},
		},
		"/dev/vda15": {
			Kname: "/dev/vda15", Type: "part", Rotational: true, MajMin: "8:15", Parent: []string{"/dev/vda"},
			Symlinks: []string{
				"/dev/disk/by-label/UEFI",
				"/dev/disk/by-partuuid/ef825b91-d4cc-47b3-bf54-99c78546a9c4",
				"/dev/disk/by-path/pci-0000:00:09.0-part15",
				"/dev/disk/by-path/virtio-pci-0000:00:09.0-part15",
				"/dev/disk/by-uuid/A82C-5E66",
			},
			Childrens: []string{"/dev/md127"},
		},
		"/dev/vdb": {
			Kname: "/dev/vdb", Type: "disk", Rotational: true, MajMin: "8:16", Parent: []string{""}, Serial: "996ea59f-7f47-4fac-b",
			Symlinks: []string{
				"/dev/disk/by-id/lvm-pv-uuid-yd92Oj-9hBf-2w2n-IEjf-nBJ1-2dMk-kBeMZI",
				"/dev/disk/by-id/virtio-996ea59f-7f47-4fac-b",
				"/dev/disk/by-path/pci-0000:00:0a.0",
				"/dev/disk/by-path/virtio-pci-0000:00:0a.0",
			},
			Childrens: []string{"/dev/mapper/ceph--992bbd78--3d8e--4cc3--93dc--eae387309364-osd--block--f4edb5cd--fb1e--4620--9419--3f9a4fcecba5"},
		},
		"/dev/vdc": {
			Kname: "/dev/vdc", Type: "disk", Rotational: true, MajMin: "8:32", Parent: []string{""},
			Symlinks: []string{
				"/dev/disk/by-label/config-2",
				"/dev/disk/by-path/pci-0000:00:0b.0",
				"/dev/disk/by-path/virtio-pci-0000:00:0b.0",
				"/dev/disk/by-uuid/BA42-906E",
			},
			Childrens: []string{},
		},
		"/dev/vde": {
			Kname: "/dev/vde", Type: "disk", Rotational: true, MajMin: "8:112", Parent: []string{""}, Serial: "2926ff77-7491-4447-a",
			Symlinks: []string{
				"/dev/disk/by-id/lvm-pv-uuid-nzJOk1-kLTM-ErxQ-0N4c-DpDU-0zhE-Q9hRJP",
				"/dev/disk/by-id/virtio-2926ff77-7491-4447-a",
				"/dev/disk/by-path/pci-0000:00:0f.0",
				"/dev/disk/by-path/virtio-pci-0000:00:0f.0",
			},
			Childrens: []string{
				"/dev/mapper/ceph--21312wds--sdfv--vs3f--scv3--sdfdsg23edaa-osd--block--vbsgs3a3--sdcv--casq--sd11--asd12dasczsf",
			},
		},
		"/dev/vdf": {
			Kname: "/dev/vdf", Type: "disk", Rotational: true, MajMin: "8:80", Parent: []string{""}, Serial: "b7ea1c8c-89b8-4354-8",
			Symlinks: []string{
				"/dev/disk/by-id/lvm-pv-uuid-fZ7Efo-X0nc-lAR3-lzik-MjMT-0rml-lZNf7b",
				"/dev/disk/by-id/virtio-b7ea1c8c-89b8-4354-8",
				"/dev/disk/by-path/pci-0000:00:10.0",
				"/dev/disk/by-path/virtio-pci-0000:00:10.0",
			},
			Childrens: []string{"/dev/mapper/ceph--2efce189--afb7--452f--bd32--c73b5017a0da-osd--block--d49fd9bf--d2dd--4c3d--824d--87f3f17ea44a"},
		},
		"/dev/vdh": {
			Kname: "/dev/vdh", Type: "disk", Rotational: true, MajMin: "8:96", Parent: []string{""}, Serial: "cf77cbec-ca01-45d9-a",
			Symlinks: []string{
				"/dev/disk/by-id/lvm-pv-uuid-gN4hiQ-gqT4-V19I-kvfA-fHWf-YIsh-gPFLTB",
				"/dev/disk/by-id/virtio-cf77cbec-ca01-45d9-a",
				"/dev/disk/by-path/pci-0000:00:11.0",
				"/dev/disk/by-path/virtio-pci-0000:00:11.0",
			},
			Childrens: []string{"/dev/vdh1"},
		},
		"/dev/vdh1": {
			Kname: "/dev/vdh1", Type: "part", Rotational: true, MajMin: "8:97", Parent: []string{"/dev/vdh"},
			Symlinks: []string{
				"/dev/disk/by-partuuid/cf77cbec-ec53-4427-45d9-8291e6213b29",
				"/dev/disk/by-path/pci-0000:00:11.0-part1",
				"/dev/disk/by-path/virtio-pci-0000:00:11.0-part1",
				"/dev/disk/by-uuid/8a13887f-1cc1-4427-ec53-7b6dd027ddc9",
			},
			Childrens: []string{"/dev/md127"},
		},
	},
	Aliases: map[string]string{
		// lvm partition pathes used in ceph-volumes
		"/dev/ceph-21312wds-sdfv-vs3f-scv3-sdfdsg23edaa/osd-block-vbsgs3a3-sdcv-casq-sd11-asd12dasczsf": "/dev/mapper/ceph--21312wds--sdfv--vs3f--scv3--sdfdsg23edaa-osd--block--vbsgs3a3--sdcv--casq--sd11--asd12dasczsf",
		"/dev/ceph-2efce189-afb7-452f-bd32-c73b5017a0da/osd-block-d49fd9bf-d2dd-4c3d-824d-87f3f17ea44a": "/dev/mapper/ceph--2efce189--afb7--452f--bd32--c73b5017a0da-osd--block--d49fd9bf--d2dd--4c3d--824d--87f3f17ea44a",
		"/dev/ceph-992bbd78-3d8e-4cc3-93dc-eae387309364/osd-block-f4edb5cd-fb1e-4620-9419-3f9a4fcecba5": "/dev/mapper/ceph--992bbd78--3d8e--4cc3--93dc--eae387309364-osd--block--f4edb5cd--fb1e--4620--9419--3f9a4fcecba5",
		"/dev/vg_root/lv_root": "/dev/mapper/vg_root-lv_root",
		// disk path by-id
		"/dev/disk/by-id/dm-name-ceph--2efce189--afb7--452f--bd32--c73b5017a0da-osd--block--d49fd9bf--d2dd--4c3d--824d--87f3f17ea44a": "/dev/mapper/ceph--2efce189--afb7--452f--bd32--c73b5017a0da-osd--block--d49fd9bf--d2dd--4c3d--824d--87f3f17ea44a",
		"/dev/disk/by-id/dm-name-ceph--21312wds--sdfv--vs3f--scv3--sdfdsg23edaa-osd--block--vbsgs3a3--sdcv--casq--sd11--asd12dasczsf": "/dev/mapper/ceph--21312wds--sdfv--vs3f--scv3--sdfdsg23edaa-osd--block--vbsgs3a3--sdcv--casq--sd11--asd12dasczsf",
		"/dev/disk/by-id/dm-name-ceph--992bbd78--3d8e--4cc3--93dc--eae387309364-osd--block--f4edb5cd--fb1e--4620--9419--3f9a4fcecba5": "/dev/mapper/ceph--992bbd78--3d8e--4cc3--93dc--eae387309364-osd--block--f4edb5cd--fb1e--4620--9419--3f9a4fcecba5",
		"/dev/disk/by-id/dm-name-vg_root-lv_root":                                                      "/dev/mapper/vg_root-lv_root",
		"/dev/disk/by-id/dm-uuid-LVM-LMoz5X0a3VV3TO2rMomDqfh24zt91NaCiZmlePb5dTd9cws2kHF6Q28W96aUWWgJ": "/dev/mapper/ceph--2efce189--afb7--452f--bd32--c73b5017a0da-osd--block--d49fd9bf--d2dd--4c3d--824d--87f3f17ea44a",
		"/dev/disk/by-id/dm-uuid-LVM-VjASpFzahZwHYS2XN4EblEfLAfVwAImtnWhRvxcC38bhRLCw9S8sCCR7JvTuSbco": "/dev/mapper/ceph--992bbd78--3d8e--4cc3--93dc--eae387309364-osd--block--f4edb5cd--fb1e--4620--9419--3f9a4fcecba5",
		"/dev/disk/by-id/dm-uuid-LVM-hVhQGAaFSKQ12ENRZABVk0nXCAos3JGulscWc97Kr4AQJNmIG0CbWYNy7fSDiVCe": "/dev/mapper/vg_root-lv_root",
		"/dev/disk/by-id/dm-uuid-LVM-oPXPcruZ1AK9dkZOsPR9ZW7PzVb9xtFrOnhN24VqDzKIOPBZLd60UQpS6PpCEzQs": "/dev/mapper/ceph--21312wds--sdfv--vs3f--scv3--sdfdsg23edaa-osd--block--vbsgs3a3--sdcv--casq--sd11--asd12dasczsf",
		"/dev/disk/by-id/lvm-pv-uuid-K7zgwt-1AY8-QxFu-ltxQ-lCnI-XMe0-91XwfL":                           "/dev/md127",
		"/dev/disk/by-id/lvm-pv-uuid-fZ7Efo-X0nc-lAR3-lzik-MjMT-0rml-lZNf7b":                           "/dev/vdf",
		"/dev/disk/by-id/lvm-pv-uuid-gN4hiQ-gqT4-V19I-kvfA-fHWf-YIsh-gPFLTB":                           "/dev/vdh",
		"/dev/disk/by-id/lvm-pv-uuid-nzJOk1-kLTM-ErxQ-0N4c-DpDU-0zhE-Q9hRJP":                           "/dev/vde",
		"/dev/disk/by-id/lvm-pv-uuid-yd92Oj-9hBf-2w2n-IEjf-nBJ1-2dMk-kBeMZI":                           "/dev/vdb",
		"/dev/disk/by-id/md-name-any:md_root":                                                          "/dev/md127",
		"/dev/disk/by-id/md-uuid-2fd11014:2ffb43fd:06d3979e:b213232b":                                  "/dev/md127",
		"/dev/disk/by-id/virtio-2926ff77-7491-4447-a":                                                  "/dev/vde",
		"/dev/disk/by-id/virtio-8dad5ae9-ddf7-40bf-8":                                                  "/dev/vda",
		"/dev/disk/by-id/virtio-996ea59f-7f47-4fac-b":                                                  "/dev/vdb",
		"/dev/disk/by-id/virtio-b7ea1c8c-89b8-4354-8":                                                  "/dev/vdf",
		"/dev/disk/by-id/virtio-cf77cbec-ca01-45d9-a":                                                  "/dev/vdh",
		// disk path by label
		"/dev/disk/by-label/UEFI":            "/dev/vda15",
		"/dev/disk/by-label/cloudimg-rootfs": "/dev/vda1",
		"/dev/disk/by-label/config-2":        "/dev/vdc",
		// disk path by part uuid
		"/dev/disk/by-partuuid/40dba738-2c45-4236-a681-75198bc111ae": "/dev/vda14",
		"/dev/disk/by-partuuid/8a13887f-ec53-4427-b9ec-8291e6213b29": "/dev/vda1",
		"/dev/disk/by-partuuid/cf77cbec-ec53-4427-45d9-8291e6213b29": "/dev/vdh1",
		"/dev/disk/by-partuuid/ef825b91-d4cc-47b3-bf54-99c78546a9c4": "/dev/vda15",
		// disk path by path
		"/dev/disk/by-path/pci-0000:00:09.0":               "/dev/vda",
		"/dev/disk/by-path/pci-0000:00:09.0-part1":         "/dev/vda1",
		"/dev/disk/by-path/pci-0000:00:09.0-part14":        "/dev/vda14",
		"/dev/disk/by-path/pci-0000:00:09.0-part15":        "/dev/vda15",
		"/dev/disk/by-path/pci-0000:00:0a.0":               "/dev/vdb",
		"/dev/disk/by-path/pci-0000:00:0b.0":               "/dev/vdc",
		"/dev/disk/by-path/pci-0000:00:0f.0":               "/dev/vde",
		"/dev/disk/by-path/pci-0000:00:10.0":               "/dev/vdf",
		"/dev/disk/by-path/pci-0000:00:11.0":               "/dev/vdh",
		"/dev/disk/by-path/pci-0000:00:11.0-part1":         "/dev/vdh1",
		"/dev/disk/by-path/virtio-pci-0000:00:09.0":        "/dev/vda",
		"/dev/disk/by-path/virtio-pci-0000:00:09.0-part1":  "/dev/vda1",
		"/dev/disk/by-path/virtio-pci-0000:00:09.0-part14": "/dev/vda14",
		"/dev/disk/by-path/virtio-pci-0000:00:09.0-part15": "/dev/vda15",
		"/dev/disk/by-path/virtio-pci-0000:00:0a.0":        "/dev/vdb",
		"/dev/disk/by-path/virtio-pci-0000:00:0b.0":        "/dev/vdc",
		"/dev/disk/by-path/virtio-pci-0000:00:0f.0":        "/dev/vde",
		"/dev/disk/by-path/virtio-pci-0000:00:10.0":        "/dev/vdf",
		"/dev/disk/by-path/virtio-pci-0000:00:11.0":        "/dev/vdh",
		"/dev/disk/by-path/virtio-pci-0000:00:11.0-part1":  "/dev/vdh1",
		// disk path by uuid
		"/dev/disk/by-uuid/8a13887f-1cc1-4427-ec53-7b6dd027ddc9": "/dev/vdh1",
		"/dev/disk/by-uuid/A82C-5E66":                            "/dev/vda15",
		"/dev/disk/by-uuid/BA42-906E":                            "/dev/vdc",
		"/dev/disk/by-uuid/a3387596-cb0c-4b14-ae91-b6992b767f50": "/dev/mapper/vg_root-lv_root",
		"/dev/disk/by-uuid/f51a1ffe-1cc1-4274-a35e-7b6dd027ddc9": "/dev/vda1",
		// device mapper aliases
		"/dev/dm-0": "/dev/mapper/vg_root-lv_root",
		"/dev/dm-1": "/dev/mapper/ceph--992bbd78--3d8e--4cc3--93dc--eae387309364-osd--block--f4edb5cd--fb1e--4620--9419--3f9a4fcecba5",
		"/dev/dm-4": "/dev/mapper/ceph--2efce189--afb7--452f--bd32--c73b5017a0da-osd--block--d49fd9bf--d2dd--4c3d--824d--87f3f17ea44a",
		"/dev/dm-5": "/dev/mapper/ceph--21312wds--sdfv--vs3f--scv3--sdfdsg23edaa-osd--block--vbsgs3a3--sdcv--casq--sd11--asd12dasczsf",
		"/dev/mapper/ceph--21312wds--sdfv--vs3f--scv3--sdfdsg23edaa-osd--block--vbsgs3a3--sdcv--casq--sd11--asd12dasczsf": "/dev/mapper/ceph--21312wds--sdfv--vs3f--scv3--sdfdsg23edaa-osd--block--vbsgs3a3--sdcv--casq--sd11--asd12dasczsf",
		"/dev/mapper/ceph--2efce189--afb7--452f--bd32--c73b5017a0da-osd--block--d49fd9bf--d2dd--4c3d--824d--87f3f17ea44a": "/dev/mapper/ceph--2efce189--afb7--452f--bd32--c73b5017a0da-osd--block--d49fd9bf--d2dd--4c3d--824d--87f3f17ea44a",
		"/dev/mapper/ceph--992bbd78--3d8e--4cc3--93dc--eae387309364-osd--block--f4edb5cd--fb1e--4620--9419--3f9a4fcecba5": "/dev/mapper/ceph--992bbd78--3d8e--4cc3--93dc--eae387309364-osd--block--f4edb5cd--fb1e--4620--9419--3f9a4fcecba5",
		"/dev/mapper/vg_root-lv_root": "/dev/mapper/vg_root-lv_root",
		// raid
		"/dev/md/md_root": "/dev/md127",
		"/dev/md127":      "/dev/md127",
		// by dev name
		"/dev/vda":   "/dev/vda",
		"/dev/vda1":  "/dev/vda1",
		"/dev/vda14": "/dev/vda14",
		"/dev/vda15": "/dev/vda15",
		"/dev/vdb":   "/dev/vdb",
		"/dev/vdc":   "/dev/vdc",
		"/dev/vde":   "/dev/vde",
		"/dev/vdf":   "/dev/vdf",
		"/dev/vdh":   "/dev/vdh",
		"/dev/vdh1":  "/dev/vdh1",
	},
	DiskToOsd: map[string][]string{
		"/dev/vda": {"30"},
		"/dev/vdb": {"30"},
		"/dev/vde": {"20"},
		"/dev/vdf": {"25"},
	},
}

var CephVolumeLvmReportFromNode1 = `{
    "20": [
        {
            "devices": [
                "/dev/vde"
            ],
            "lv_name": "osd-block-vbsgs3a3-sdcv-casq-sd11-asd12dasczsf",
            "lv_path": "/dev/ceph-21312wds-sdfv-vs3f-scv3-sdfdsg23edaa/osd-block-vbsgs3a3-sdcv-casq-sd11-asd12dasczsf",
            "lv_size": "4000783007744",
            "lv_tags": "ceph.block_device=/dev/ceph-21312wds-sdfv-vs3f-scv3-sdfdsg23edaa/osd-block-vbsgs3a3-sdcv-casq-sd11-asd12dasczsf,ceph.block_uuid=eN3l3N-wfCB-2kOO-uEeU-REUQ-repf-Q01EhF,ceph.cephx_lockbox_secret=,ceph.cluster_fsid=8668f062-3faa-358a-85f3-f80fe6c1e306,ceph.cluster_name=ceph,ceph.crush_device_class=hdd,ceph.db_device=/dev/ceph-metadata/part-1,ceph.db_uuid=H1waxW-ke4f-bXED-XucE-bZNe-B4ZD-fBeUrW,ceph.encrypted=0,ceph.osd_fsid=vbsgs3a3-sdcv-casq-sd11-asd12dasczsf,ceph.osd_id=20,ceph.osdspec_affinity=,ceph.type=block,ceph.vdo=0",
            "lv_uuid": "eN3l3N-wfCB-2kOO-uEeU-REUQ-repf-Q01EhF",
            "name": "osd-block-vbsgs3a3-sdcv-casq-sd11-asd12dasczsf",
            "path": "/dev/ceph-21312wds-sdfv-vs3f-scv3-sdfdsg23edaa/osd-block-vbsgs3a3-sdcv-casq-sd11-asd12dasczsf",
            "tags": {
                "ceph.block_device": "/dev/ceph-21312wds-sdfv-vs3f-scv3-sdfdsg23edaa/osd-block-vbsgs3a3-sdcv-casq-sd11-asd12dasczsf",
                "ceph.block_uuid": "eN3l3N-wfCB-2kOO-uEeU-REUQ-repf-Q01EhF",
                "ceph.cephx_lockbox_secret": "",
                "ceph.cluster_fsid": "8668f062-3faa-358a-85f3-f80fe6c1e306",
                "ceph.cluster_name": "ceph",
                "ceph.crush_device_class": "hdd",
                "ceph.db_device": "/dev/ceph-metadata/part-1",
                "ceph.db_uuid": "H1waxW-ke4f-bXED-XucE-bZNe-B4ZD-fBeUrW",
                "ceph.encrypted": "0",
                "ceph.osd_fsid": "vbsgs3a3-sdcv-casq-sd11-asd12dasczsf",
                "ceph.osd_id": "20",
                "ceph.osdspec_affinity": "",
                "ceph.type": "block",
                "ceph.vdo": "0"
            },
            "type": "block",
            "vg_name": "ceph-21312wds-sdfv-vs3f-scv3-sdfdsg23edaa"
        },
        {
            "devices": [
                "/dev/vdd1"
            ],
            "lv_name": "part-1",
            "lv_path": "/dev/ceph-metadata/part-1",
            "lv_size": "184318689280",
            "lv_tags": "ceph.block_device=/dev/ceph-21312wds-sdfv-vs3f-scv3-sdfdsg23edaa/osd-block-vbsgs3a3-sdcv-casq-sd11-asd12dasczsf,ceph.block_uuid=eN3l3N-wfCB-2kOO-uEeU-REUQ-repf-Q01EhF,ceph.cephx_lockbox_secret=,ceph.cluster_fsid=8668f062-3faa-358a-85f3-f80fe6c1e306,ceph.cluster_name=ceph,ceph.crush_device_class=hdd,ceph.db_device=/dev/ceph-metadata/part-1,ceph.db_uuid=H1waxW-ke4f-bXED-XucE-bZNe-B4ZD-fBeUrW,ceph.encrypted=0,ceph.osd_fsid=vbsgs3a3-sdcv-casq-sd11-asd12dasczsf,ceph.osd_id=20,ceph.osdspec_affinity=,ceph.type=db,ceph.vdo=0",
            "lv_uuid": "H1waxW-ke4f-bXED-XucE-bZNe-B4ZD-fBeUrW",
            "name": "meta-dev",
            "path": "/dev/ceph-metadata/part-1",
            "tags": {
                "ceph.block_device": "/dev/ceph-21312wds-sdfv-vs3f-scv3-sdfdsg23edaa/osd-block-vbsgs3a3-sdcv-casq-sd11-asd12dasczsf",
                "ceph.block_uuid": "eN3l3N-wfCB-2kOO-uEeU-REUQ-repf-Q01EhF",
                "ceph.cephx_lockbox_secret": "",
                "ceph.cluster_fsid": "8668f062-3faa-358a-85f3-f80fe6c1e306",
                "ceph.cluster_name": "ceph",
                "ceph.crush_device_class": "hdd",
                "ceph.db_device": "/dev/ceph-metadata/part-1",
                "ceph.db_uuid": "H1waxW-ke4f-bXED-XucE-bZNe-B4ZD-fBeUrW",
                "ceph.encrypted": "0",
                "ceph.osd_fsid": "vbsgs3a3-sdcv-casq-sd11-asd12dasczsf",
                "ceph.osd_id": "20",
                "ceph.osdspec_affinity": "",
                "ceph.type": "db",
                "ceph.vdo": "0"
            },
            "type": "db",
            "vg_name": "ceph-metadata"
        }
    ],
    "25": [
        {
            "devices": [
                "/dev/vdf"
            ],
            "lv_name": "osd-block-d49fd9bf-d2dd-4c3d-824d-87f3f17ea44a",
            "lv_path": "/dev/ceph-2efce189-afb7-452f-bd32-c73b5017a0da/osd-block-d49fd9bf-d2dd-4c3d-824d-87f3f17ea44a",
            "lv_size": "4000783007744",
            "lv_tags": "ceph.block_device=/dev/ceph-2efce189-afb7-452f-bd32-c73b5017a0da/osd-block-d49fd9bf-d2dd-4c3d-824d-87f3f17ea44a,ceph.block_uuid=eN3l3N-wfCB-2kOO-uEeU-REUQ-repf-Q01EhF,ceph.cephx_lockbox_secret=,ceph.cluster_fsid=8668f062-3faa-358a-85f3-f80fe6c1e306,ceph.cluster_name=ceph,ceph.crush_device_class=hdd,ceph.db_device=/dev/db5/meta-dev,ceph.db_uuid=8yzVYt-4FmS-mRA1-WtsA-PWLg-nPzs-LFSMFy,ceph.encrypted=0,ceph.osd_fsid=7a2c94f4-e346-4576-a8f5-33453efcf6d9,ceph.osd_id=25,ceph.osdspec_affinity=,ceph.type=block,ceph.vdo=0",
            "lv_uuid": "eN3l3N-wfCB-2kOO-uEeU-REUQ-repf-Q01EhF",
            "name": "osd-block-d49fd9bf-d2dd-4c3d-824d-87f3f17ea44a",
            "path": "/dev/ceph-2efce189-afb7-452f-bd32-c73b5017a0da/osd-block-d49fd9bf-d2dd-4c3d-824d-87f3f17ea44a",
            "tags": {
                "ceph.block_device": "/dev/ceph-2efce189-afb7-452f-bd32-c73b5017a0da/osd-block-d49fd9bf-d2dd-4c3d-824d-87f3f17ea44a",
                "ceph.block_uuid": "eN3l3N-wfCB-2kOO-uEeU-REUQ-repf-Q01EhF",
                "ceph.cephx_lockbox_secret": "",
                "ceph.cluster_fsid": "8668f062-3faa-358a-85f3-f80fe6c1e306",
                "ceph.cluster_name": "ceph",
                "ceph.crush_device_class": "hdd",
                "ceph.db_device": "/dev/ceph-metadata/part-2",
                "ceph.db_uuid": "8yzVYt-4FmS-mRA1-WtsA-PWLg-nPzs-LFSMFy",
                "ceph.encrypted": "0",
                "ceph.osd_fsid": "d49fd9bf-d2dd-4c3d-824d-87f3f17ea44a",
                "ceph.osd_id": "25",
                "ceph.osdspec_affinity": "",
                "ceph.type": "block",
                "ceph.vdo": "0"
            },
            "type": "block",
            "vg_name": "ceph-2efce189-afb7-452f-bd32-c73b5017a0da"
        },
        {
            "devices": [
                "/dev/vdd1"
            ],
            "lv_name": "part-2",
            "lv_path": "/dev/ceph-metadata/part-2",
            "lv_size": "184318689280",
            "lv_tags": "ceph.block_device=/dev/ceph-2efce189-afb7-452f-bd32-c73b5017a0da/osd-block-d49fd9bf-d2dd-4c3d-824d-87f3f17ea44a,ceph.block_uuid=eN3l3N-wfCB-2kOO-uEeU-REUQ-repf-Q01EhF,ceph.cephx_lockbox_secret=,ceph.cluster_fsid=8668f062-3faa-358a-85f3-f80fe6c1e306,ceph.cluster_name=ceph,ceph.crush_device_class=hdd,ceph.db_device=/dev/ceph-metadata/part-2,ceph.db_uuid=8yzVYt-4FmS-mRA1-WtsA-PWLg-nPzs-LFSMFy,ceph.encrypted=0,ceph.osd_fsid=d49fd9bf-d2dd-4c3d-824d-87f3f17ea44a,ceph.osd_id=25,ceph.osdspec_affinity=,ceph.type=db,ceph.vdo=0",
            "lv_uuid": "8yzVYt-4FmS-mRA1-WtsA-PWLg-nPzs-LFSMFy",
            "name": "part-2",
            "path": "/dev/ceph-metadata/part-2",
            "tags": {
                "ceph.block_device": "/dev/ceph-2efce189-afb7-452f-bd32-c73b5017a0da/osd-block-d49fd9bf-d2dd-4c3d-824d-87f3f17ea44a",
                "ceph.block_uuid": "eN3l3N-wfCB-2kOO-uEeU-REUQ-repf-Q01EhF",
                "ceph.cephx_lockbox_secret": "",
                "ceph.cluster_fsid": "8668f062-3faa-358a-85f3-f80fe6c1e306",
                "ceph.cluster_name": "ceph",
                "ceph.crush_device_class": "hdd",
                "ceph.db_device": "/dev/ceph-metadata/part-2",
                "ceph.db_uuid": "8yzVYt-4FmS-mRA1-WtsA-PWLg-nPzs-LFSMFy",
                "ceph.encrypted": "0",
                "ceph.osd_fsid": "d49fd9bf-d2dd-4c3d-824d-87f3f17ea44a",
                "ceph.osd_id": "25",
                "ceph.osdspec_affinity": "",
                "ceph.type": "db",
                "ceph.vdo": "0"
            },
            "type": "db",
            "vg_name": "ceph-metadata"
        }
    ],
    "30": [
        {
            "devices": [
                "/dev/vdb"
            ],
            "lv_name": "osd-block-f4edb5cd-fb1e-4620-9419-3f9a4fcecba5",
            "lv_path": "/dev/ceph-992bbd78-3d8e-4cc3-93dc-eae387309364/osd-block-f4edb5cd-fb1e-4620-9419-3f9a4fcecba5",
            "lv_size": "4000762036224",
            "lv_tags": "ceph.block_device=/dev/ceph-992bbd78-3d8e-4cc3-93dc-eae387309364/osd-block-f4edb5cd-fb1e-4620-9419-3f9a4fcecba5,ceph.block_uuid=dV9QaF-fQAz-3XcA-41g0-UDlt-fuai-109bdy,ceph.cephx_lockbox_secret=,ceph.cluster_fsid=8668f062-3faa-358a-85f3-f80fe6c1e306,ceph.cluster_name=ceph,ceph.crush_device_class=None,ceph.db_device=/dev/vda14,ceph.db_uuid=40dba738-2c45-4236-a681-75198bc111ae,ceph.encrypted=0,ceph.osd_fsid=635a6fd8-dcad-4601-84a6-2150f2eef8c8,ceph.osd_id=30,ceph.type=block,ceph.vdo=0",
            "lv_uuid": "dV9QaF-fQAz-3XcA-41g0-UDlt-fuai-109bdy",
            "name": "osd-block-f4edb5cd-fb1e-4620-9419-3f9a4fcecba5",
            "path": "/dev/ceph-992bbd78-3d8e-4cc3-93dc-eae387309364/osd-block-f4edb5cd-fb1e-4620-9419-3f9a4fcecba5",
            "tags": {
                "ceph.block_device": "/dev/ceph-6e6365e0-b9ae-478b-a43c-644074506aae/osd-block-635a6fd8-dcad-4601-84a6-2150f2eef8c8",
                "ceph.block_uuid": "dV9QaF-fQAz-3XcA-41g0-UDlt-fuai-109bdy",
                "ceph.cephx_lockbox_secret": "",
                "ceph.cluster_fsid": "8668f062-3faa-358a-85f3-f80fe6c1e306",
                "ceph.cluster_name": "ceph",
                "ceph.crush_device_class": "None",
                "ceph.db_device": "/dev/vda14",
                "ceph.db_uuid": "40dba738-2c45-4236-a681-75198bc111ae",
                "ceph.encrypted": "0",
                "ceph.osd_fsid": "f4edb5cd-fb1e-4620-9419-3f9a4fcecba5",
                "ceph.osd_id": "30",
                "ceph.type": "block",
                "ceph.vdo": "0"
            },
            "type": "block",
            "vg_name": "ceph-992bbd78-3d8e-4cc3-93dc-eae387309364"
        },
        {
            "path": "/dev/vda14",
            "tags": {
                "PARTUUID": "40dba738-2c45-4236-a681-75198bc111ae"
            },
            "type": "db"
        }
    ]
}`

var LvmLvsReportFromNode1 = `{
      "report": [
          {
              "lv": [
                  {"lv_dm_path":"/dev/mapper/ceph--metadata-part--1"},
                  {"lv_dm_path":"/dev/mapper/ceph--metadata-part--2"},
                  {"lv_dm_path":"/dev/mapper/vg_root-lv_root"},
                  {"lv_dm_path":"/dev/mapper/ceph--21312wds--sdfv--vs3f--scv3--sdfdsg23edaa-osd--block--vbsgs3a3--sdcv--casq--sd11--asd12dasczsf"},
                  {"lv_dm_path":"/dev/mapper/ceph--992bbd78--3d8e--4cc3--93dc--eae387309364-osd--block--f4edb5cd--fb1e--4620--9419--3f9a4fcecba5"},
                  {"lv_dm_path":"/dev/mapper/ceph--2efce189--afb7--452f--bd32--c73b5017a0da-osd--block--d49fd9bf--d2dd--4c3d--824d--87f3f17ea44a"}
              ]
          }
      ]
  }
`

var FoundLvmsFromNode1 = map[string][]string{
	"/dev/mapper/vg_root-lv_root": {"/dev/md127"},
	"/dev/mapper/ceph--992bbd78--3d8e--4cc3--93dc--eae387309364-osd--block--f4edb5cd--fb1e--4620--9419--3f9a4fcecba5": {"/dev/vdb"},
	"/dev/mapper/ceph--metadata-part--1": {"/dev/vdd1"},
	"/dev/mapper/ceph--metadata-part--2": {"/dev/vdd1"},
	"/dev/mapper/ceph--21312wds--sdfv--vs3f--scv3--sdfdsg23edaa-osd--block--vbsgs3a3--sdcv--casq--sd11--asd12dasczsf": {"/dev/vde"},
	"/dev/mapper/ceph--2efce189--afb7--452f--bd32--c73b5017a0da-osd--block--d49fd9bf--d2dd--4c3d--824d--87f3f17ea44a": {"/dev/vdf"},
}
