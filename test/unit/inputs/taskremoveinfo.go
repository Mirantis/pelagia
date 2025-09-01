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
	lcmv1alpha1 "github.com/Mirantis/pelagia/pkg/apis/ceph.pelagia.lcm/v1alpha1"
)

var EmptyRemoveMap = &lcmv1alpha1.TaskRemoveInfo{
	CleanupMap: map[string]lcmv1alpha1.HostMapping{},
	Issues:     []string{},
	Warnings:   []string{},
}

var DevNotInSpecRemoveMap = &lcmv1alpha1.TaskRemoveInfo{
	CleanupMap: map[string]lcmv1alpha1.HostMapping{
		"node-1": {
			OsdMapping: map[string]lcmv1alpha1.OsdMapping{
				"20": AdaptOsdMapping("node-1", "20",
					map[string]bool{"inCrush": true}, map[string]map[string]bool{"/dev/vde": {"zap": true}, "/dev/vdd": {}}),
			},
		},
		"node-2": {
			OsdMapping: map[string]lcmv1alpha1.OsdMapping{
				"4": AdaptOsdMapping("node-2", "4",
					map[string]bool{"inCrush": true}, map[string]map[string]bool{"/dev/vdd": {"zap": true}}),
				"5": AdaptOsdMapping("node-2", "5",
					map[string]bool{"inCrush": true}, map[string]map[string]bool{"/dev/vdd": {"zap": true}}),
			},
		},
	},
	Issues: []string{},
	Warnings: []string{
		"[node 'node-1'] found osd db partition '/dev/ceph-metadata/part-1' for osd '20', which is created not by rook, skipping disk/partition zap",
	},
}

var FullNodesRemoveMap = &lcmv1alpha1.TaskRemoveInfo{
	CleanupMap: map[string]lcmv1alpha1.HostMapping{
		"node-1": {
			CompleteCleanup: true,
			OsdMapping: map[string]lcmv1alpha1.OsdMapping{
				"20": AdaptOsdMapping("node-1", "20",
					map[string]bool{"inCrush": true}, map[string]map[string]bool{"/dev/vde": {"zap": true}, "/dev/vdd": {}}),
				"25": AdaptOsdMapping("node-1", "25",
					map[string]bool{"inCrush": true}, map[string]map[string]bool{"/dev/vdf": {"zap": true}, "/dev/vdd": {}}),
				"30": AdaptOsdMapping("node-1", "30",
					map[string]bool{"inCrush": true}, map[string]map[string]bool{"/dev/vda": {}, "/dev/vdb": {"zap": true}}),
			},
		},
		"node-2": {
			CompleteCleanup: true,
			OsdMapping: map[string]lcmv1alpha1.OsdMapping{
				"0": AdaptOsdMapping("node-2", "0",
					map[string]bool{"inCrush": true}, map[string]map[string]bool{"/dev/vdb": {"zap": true}}),
				"4": AdaptOsdMapping("node-2", "4",
					map[string]bool{"inCrush": true}, map[string]map[string]bool{"/dev/vdd": {"zap": true}}),
				"5": AdaptOsdMapping("node-2", "5",
					map[string]bool{"inCrush": true}, map[string]map[string]bool{"/dev/vdd": {"zap": true}}),
			},
		},
	},
	Issues: []string{},
	Warnings: []string{
		"[node 'node-1'] found osd db partition '/dev/ceph-metadata/part-1' for osd '20', which is created not by rook, skipping disk/partition zap",
		"[node 'node-1'] found osd db partition '/dev/ceph-metadata/part-2' for osd '25', which is created not by rook, skipping disk/partition zap",
		"[node 'node-1'] found osd db partition '/dev/vda14' for osd '30', which is created not by rook, skipping disk/partition zap",
		"[node 'node-1'] found physical osd db partition '/dev/vda14' for osd '30'",
	},
}

var NodesRemoveMapEmptyRemoveStatus = GetInfoWithStatus(FullNodesRemoveMap, map[string]*lcmv1alpha1.RemoveResult{"*": nil})
var NodesRemoveMapOsdFinishedStatus = GetInfoWithStatus(FullNodesRemoveMap,
	map[string]*lcmv1alpha1.RemoveResult{
		"*": {
			OsdRemoveStatus:    &lcmv1alpha1.RemoveStatus{Status: lcmv1alpha1.RemoveFinished},
			DeviceCleanUpJob:   &lcmv1alpha1.RemoveStatus{Status: lcmv1alpha1.RemoveCompleted},
			DeployRemoveStatus: &lcmv1alpha1.RemoveStatus{Status: lcmv1alpha1.RemoveFinished},
		},
	},
)
var NodesRemoveFullFinishedStatus = func() *lcmv1alpha1.TaskRemoveInfo {
	info := NodesRemoveMapOsdFinishedStatus.DeepCopy()
	host1 := info.CleanupMap["node-1"]
	host1.HostRemoveStatus = &lcmv1alpha1.RemoveStatus{Status: lcmv1alpha1.RemoveFinished}
	host2 := info.CleanupMap["node-2"]
	host2.HostRemoveStatus = &lcmv1alpha1.RemoveStatus{Status: lcmv1alpha1.RemoveFinished}
	info.CleanupMap["node-1"] = host1
	info.CleanupMap["node-2"] = host2
	return info
}()

var StrayOnlyInCrushRemoveMap = &lcmv1alpha1.TaskRemoveInfo{
	CleanupMap: map[string]lcmv1alpha1.HostMapping{
		"__stray": {
			OsdMapping: map[string]lcmv1alpha1.OsdMapping{
				"2": {
					UUID:        "61869d90-2c45-4f02-b7c3-96955f41e2ca",
					ClusterFSID: "8668f062-3faa-358a-85f3-f80fe6c1e306",
					InCrushMap:  true,
				},
			},
		},
	},
	Issues: []string{},
	Warnings: []string{
		"[stray] detected stray osds, but impossible to determine related host/device (probably disk(s) removed or host(s) down), device cleanup jobs will be skipped",
	},
}

var StrayOnlyOnNodeRemoveMap = &lcmv1alpha1.TaskRemoveInfo{
	CleanupMap: map[string]lcmv1alpha1.HostMapping{
		"node-2": {
			OsdMapping: map[string]lcmv1alpha1.OsdMapping{
				"0.06bf4d7c-9603-41a4-b250-284ecf3ecb2f.__stray": {
					UUID:          "06bf4d7c-9603-41a4-b250-284ecf3ecb2f",
					ClusterFSID:   "8668f062-0lsk-358a-1gt4-f80fe6c1e306",
					HostDirectory: "/var/lib/rook/rook-ceph/8668f062-0lsk-358a-1gt4-f80fe6c1e306_06bf4d7c-9603-41a4-b250-284ecf3ecb2f",
					InCrushMap:    false,
					DeviceMapping: map[string]lcmv1alpha1.DeviceInfo{
						"/dev/vdc": {
							ID:         "ffe08946-7614-4f69-b",
							Rotational: true,
							Path:       "/dev/disk/by-path/pci-0000:00:0c.0",
							Partition:  "/dev/ceph-c5628abe-ae41-4c3d-bdc6-ef86c54bf78c/osd-block-69481cd1-38b1-42fd-ac07-06bf4d7c0e19",
							Type:       "block",
							Zap:        true,
							Alive:      true,
						},
					},
				},
			},
		},
	},
	Issues: []string{},
	Warnings: []string{
		"[node 'node-2'] found partition with stray osd uuid '06bf4d7c-9603-41a4-b250-284ecf3ecb2f', id '0', will be cleaned up",
	},
}

var StrayOnNodeAndInCrushRemoveMap = &lcmv1alpha1.TaskRemoveInfo{
	CleanupMap: map[string]lcmv1alpha1.HostMapping{
		"node-2": {
			OsdMapping: map[string]lcmv1alpha1.OsdMapping{
				"0.06bf4d7c-9603-41a4-b250-284ecf3ecb2f.__stray": StrayOnlyOnNodeRemoveMap.CleanupMap["node-2"].OsdMapping["0.06bf4d7c-9603-41a4-b250-284ecf3ecb2f.__stray"],
				"2.61869d90-2c45-4f02-b7c3-96955f41e2ca.__stray": {
					UUID:          "61869d90-2c45-4f02-b7c3-96955f41e2ca",
					ClusterFSID:   "8668f062-3faa-358a-85f3-f80fe6c1e306",
					HostDirectory: "/var/lib/rook/rook-ceph/8668f062-3faa-358a-85f3-f80fe6c1e306_61869d90-2c45-4f02-b7c3-96955f41e2ca",
					InCrushMap:    true,
					DeviceMapping: map[string]lcmv1alpha1.DeviceInfo{
						"/dev/vde": {
							ID:         "8cbb9ce3-6fb4-4216-8",
							Rotational: true,
							Path:       "/dev/disk/by-path/pci-0000:00:0e.0",
							Partition:  "/dev/ceph-0e03d5c6-d0e9-4f04-b9af-38d15e14369f/osd-block-61869d90-2c45-4f02-b7c3-96955f41e2ca",
							Type:       "block",
							Zap:        true,
							Alive:      true,
						},
					},
				},
			},
		},
	},
	Issues: []string{},
	Warnings: []string{
		"[node 'node-2'] found partition with stray osd uuid '06bf4d7c-9603-41a4-b250-284ecf3ecb2f', id '0', will be cleaned up",
		"[node 'node-2'] found partition with stray osd uuid '61869d90-2c45-4f02-b7c3-96955f41e2ca', id '2', will be cleaned up",
	},
}

var NotLabeledNodesFullRemoveMap = &lcmv1alpha1.TaskRemoveInfo{
	CleanupMap: map[string]lcmv1alpha1.HostMapping{
		"node-1": {
			CompleteCleanup:   true,
			VolumesInfoMissed: true,
			OsdMapping: map[string]lcmv1alpha1.OsdMapping{
				"20": AdaptOsdMapping("node-1", "20", map[string]bool{"inCrush": true, "noDaemon": true}, nil),
				"25": AdaptOsdMapping("node-1", "25", map[string]bool{"inCrush": true, "noDaemon": true}, nil),
				"30": AdaptOsdMapping("node-1", "30", map[string]bool{"inCrush": true, "noDaemon": true}, nil),
			},
		},
		"node-2": {
			DropFromCrush:     true,
			VolumesInfoMissed: true,
			OsdMapping: map[string]lcmv1alpha1.OsdMapping{
				"0": AdaptOsdMapping("node-2", "0", map[string]bool{"inCrush": true, "noDaemon": true}, nil),
				"4": AdaptOsdMapping("node-2", "4", map[string]bool{"inCrush": true, "noDaemon": true}, nil),
				"5": AdaptOsdMapping("node-2", "5", map[string]bool{"inCrush": true, "noDaemon": true}, nil),
			},
		},
	},
	Issues: []string{},
	Warnings: []string{
		"[node 'node-1'] node is available, but has no disk daemon running, device cleanup jobs will be skipped",
		"[node 'node-2'] node is available, but has no disk daemon running, device cleanup jobs will be skipped",
	},
}

var NotAvailableNodesFullRemoveMap = func() *lcmv1alpha1.TaskRemoveInfo {
	newMap := NotLabeledNodesFullRemoveMap.DeepCopy()
	for node, mapping := range newMap.CleanupMap {
		mapping.VolumesInfoMissed = false
		mapping.NodeIsDown = true
		newMap.CleanupMap[node] = mapping
	}
	newMap.Warnings = []string{
		"[node 'node-1'] node is not available, device cleanup jobs will be skipped",
		"[node 'node-2'] node is not available, device cleanup jobs will be skipped",
	}
	return newMap
}()

var SkipCleanupJobRemoveMap = &lcmv1alpha1.TaskRemoveInfo{
	CleanupMap: map[string]lcmv1alpha1.HostMapping{
		"node-1": {
			OsdMapping: map[string]lcmv1alpha1.OsdMapping{
				"30": {
					UUID:                 "f4edb5cd-fb1e-4620-9419-3f9a4fcecba5",
					ClusterFSID:          "8668f062-3faa-358a-85f3-f80fe6c1e306",
					HostDirectory:        "/var/lib/rook/rook-ceph/8668f062-3faa-358a-85f3-f80fe6c1e306_f4edb5cd-fb1e-4620-9419-3f9a4fcecba5",
					SkipDeviceCleanupJob: true,
					InCrushMap:           true,
					DeviceMapping: map[string]lcmv1alpha1.DeviceInfo{
						"/dev/vda": {
							ID:         "8dad5ae9-ddf7-40bf-8",
							Rotational: true,
							Path:       "/dev/disk/by-path/pci-0000:00:09.0",
							Partition:  "/dev/vda14",
							Type:       "db",
							Alive:      true,
						},
						"/dev/vdb": {
							ID:         "996ea59f-7f47-4fac-b",
							Rotational: true,
							Path:       "/dev/disk/by-path/pci-0000:00:0a.0",
							Partition:  "/dev/ceph-992bbd78-3d8e-4cc3-93dc-eae387309364/osd-block-f4edb5cd-fb1e-4620-9419-3f9a4fcecba5",
							Type:       "block",
							Alive:      true,
						},
					},
				},
			},
		},
		"node-2": {
			OsdMapping: map[string]lcmv1alpha1.OsdMapping{
				"4": {
					UUID:          "ad76cf53-5cb5-48fe-a39a-343734f5ccde",
					ClusterFSID:   "8668f062-3faa-358a-85f3-f80fe6c1e306",
					InCrushMap:    true,
					HostDirectory: "/var/lib/rook/rook-ceph/8668f062-3faa-358a-85f3-f80fe6c1e306_ad76cf53-5cb5-48fe-a39a-343734f5ccde",
					DeviceMapping: map[string]lcmv1alpha1.DeviceInfo{
						"/dev/vdd": {
							ID:         "35a15532-8b56-4f83-9",
							Rotational: false,
							Path:       "/dev/disk/by-path/pci-0000:00:1e.0",
							Partition:  "/dev/ceph-dada9f25-41b4-4c26-9a20-448ac01e1d06/osd-block-ad76cf53-5cb5-48fe-a39a-343734f5ccde",
							Type:       "block",
							Alive:      true,
							Zap:        true,
						},
					},
				},
				"5": {
					UUID:                 "af39b794-e1c6-41c0-8997-d6b6c631b8f2",
					ClusterFSID:          "8668f062-3faa-358a-85f3-f80fe6c1e306",
					HostDirectory:        "/var/lib/rook/rook-ceph/8668f062-3faa-358a-85f3-f80fe6c1e306_af39b794-e1c6-41c0-8997-d6b6c631b8f2",
					SkipDeviceCleanupJob: true,
					InCrushMap:           true,
					DeviceMapping: map[string]lcmv1alpha1.DeviceInfo{
						"/dev/vdd": {
							ID:         "35a15532-8b56-4f83-9",
							Rotational: false,
							Path:       "/dev/disk/by-path/pci-0000:00:1e.0",
							Partition:  "/dev/ceph-dada9f25-41b4-4c26-9a20-448ac01e1d06/osd-block-7d09cceb-4de0-478e-9d8d-bd09cb0c904e",
							Type:       "block",
							Alive:      true,
						},
					},
				},
			},
		},
	},
	Issues: []string{},
	Warnings: []string{
		"[node 'node-1'] device 'vdb' has set 'skip device clean up' flag set in spec. Related osd deployment (osd id '30') should be removed manually as well",
		"[node 'node-1'] found physical osd db partition '/dev/vda14' for osd '30'",
		"[node 'node-2'] osd with id '5' has 'skip device clean up' flag set in spec. Related deployment should be removed manually as well",
	},
}

func AdaptOsdMapping(node, osdID string, osdConfig map[string]bool, devConfig map[string]map[string]bool) lcmv1alpha1.OsdMapping {
	var osdMapping lcmv1alpha1.OsdMapping
	if osdConfig["noDaemon"] {
		osdMapping = FullNodesInfoFromOsdMeta[node].OsdMapping[osdID]
	} else {
		osdMapping = FullNodesInfoFromDaemon[node].OsdMapping[osdID]
	}
	osdMapping.InCrushMap = osdConfig["inCrush"]
	if len(devConfig) > 0 {
		osdMapping.DeviceMapping = AdaptDeviceMapping(node, osdID, devConfig)
	}
	return osdMapping
}

func AdaptDeviceMapping(node, osdID string, devConfig map[string]map[string]bool) map[string]lcmv1alpha1.DeviceInfo {
	getDeviceInfo := func(devName string, lost bool) lcmv1alpha1.DeviceInfo {
		if lost {
			return FullNodesInfoFromOsdMeta[node].OsdMapping[osdID].DeviceMapping[devName]
		}
		return FullNodesInfoFromDaemon[node].OsdMapping[osdID].DeviceMapping[devName]
	}
	devMapping := map[string]lcmv1alpha1.DeviceInfo{}
	for dev, conf := range devConfig {
		devInfo := getDeviceInfo(dev, conf["lost"])
		if conf["lost"] {
			devInfo.Alive = false
			devInfo.Zap = false
		} else if conf["zap"] {
			devInfo.Zap = true
		}
		devMapping[dev] = devInfo
	}
	return devMapping
}

func GetInfoWithStatus(sourceInfo *lcmv1alpha1.TaskRemoveInfo, statusMap map[string]*lcmv1alpha1.RemoveResult) *lcmv1alpha1.TaskRemoveInfo {
	newInfo := sourceInfo.DeepCopy()
	statusForAll, forAll := statusMap["*"]
	for host, hostMapping := range newInfo.CleanupMap {
		for osd, osdMapping := range hostMapping.OsdMapping {
			statusForOsd, forOsd := statusMap[osd]
			if forOsd || forAll {
				if forOsd && statusForOsd != nil {
					osdMapping.RemoveStatus = statusForOsd
				} else if forAll && statusForAll != nil {
					osdMapping.RemoveStatus = statusForAll
				} else {
					osdMapping.RemoveStatus = &lcmv1alpha1.RemoveResult{}
				}
				newInfo.CleanupMap[host].OsdMapping[osd] = osdMapping
			}
		}
	}
	return newInfo
}

/* general info for spec validation from disk-daemon or osd metadata */

var FullNodesInfoFromDaemon = map[string]lcmv1alpha1.HostMapping{
	"node-1": {
		OsdMapping: map[string]lcmv1alpha1.OsdMapping{
			"20": {
				UUID:          "vbsgs3a3-sdcv-casq-sd11-asd12dasczsf",
				ClusterFSID:   "8668f062-3faa-358a-85f3-f80fe6c1e306",
				HostDirectory: "/var/lib/rook/rook-ceph/8668f062-3faa-358a-85f3-f80fe6c1e306_vbsgs3a3-sdcv-casq-sd11-asd12dasczsf",
				DeviceMapping: map[string]lcmv1alpha1.DeviceInfo{
					"/dev/vde": {
						ID:         "2926ff77-7491-4447-a",
						Rotational: true,
						Path:       "/dev/disk/by-path/pci-0000:00:0f.0",
						Partition:  "/dev/ceph-21312wds-sdfv-vs3f-scv3-sdfdsg23edaa/osd-block-vbsgs3a3-sdcv-casq-sd11-asd12dasczsf",
						Type:       "block",
						Alive:      true,
					},
					"/dev/vdd": {
						ID:         "e8d89e2f-ffc6-4988-9",
						Rotational: true,
						Path:       "/dev/disk/by-path/pci-0000:00:0e.0",
						Partition:  "/dev/ceph-metadata/part-1",
						Type:       "db",
						Alive:      true,
					},
				},
			},
			"25": {
				UUID:          "d49fd9bf-d2dd-4c3d-824d-87f3f17ea44a",
				ClusterFSID:   "8668f062-3faa-358a-85f3-f80fe6c1e306",
				HostDirectory: "/var/lib/rook/rook-ceph/8668f062-3faa-358a-85f3-f80fe6c1e306_d49fd9bf-d2dd-4c3d-824d-87f3f17ea44a",
				DeviceMapping: map[string]lcmv1alpha1.DeviceInfo{
					"/dev/vdf": {
						ID:         "b7ea1c8c-89b8-4354-8",
						Rotational: true,
						Path:       "/dev/disk/by-path/pci-0000:00:10.0",
						Partition:  "/dev/ceph-2efce189-afb7-452f-bd32-c73b5017a0da/osd-block-d49fd9bf-d2dd-4c3d-824d-87f3f17ea44a",
						Type:       "block",
						Alive:      true,
					},
					"/dev/vdd": {
						ID:         "e8d89e2f-ffc6-4988-9",
						Rotational: true,
						Path:       "/dev/disk/by-path/pci-0000:00:0e.0",
						Partition:  "/dev/ceph-metadata/part-2",
						Type:       "db",
						Alive:      true,
					},
				},
			},
			"30": {
				UUID:          "f4edb5cd-fb1e-4620-9419-3f9a4fcecba5",
				ClusterFSID:   "8668f062-3faa-358a-85f3-f80fe6c1e306",
				HostDirectory: "/var/lib/rook/rook-ceph/8668f062-3faa-358a-85f3-f80fe6c1e306_f4edb5cd-fb1e-4620-9419-3f9a4fcecba5",
				DeviceMapping: map[string]lcmv1alpha1.DeviceInfo{
					"/dev/vda": {
						ID:         "8dad5ae9-ddf7-40bf-8",
						Rotational: true,
						Path:       "/dev/disk/by-path/pci-0000:00:09.0",
						Partition:  "/dev/vda14",
						Type:       "db",
						Alive:      true,
					},
					"/dev/vdb": {
						ID:         "996ea59f-7f47-4fac-b",
						Rotational: true,
						Path:       "/dev/disk/by-path/pci-0000:00:0a.0",
						Partition:  "/dev/ceph-992bbd78-3d8e-4cc3-93dc-eae387309364/osd-block-f4edb5cd-fb1e-4620-9419-3f9a4fcecba5",
						Type:       "block",
						Alive:      true,
					},
				},
			},
		},
	},
	"node-2": {
		OsdMapping: map[string]lcmv1alpha1.OsdMapping{
			"0": {
				UUID:          "69481cd1-38b1-42fd-ac07-06bf4d7c0e19",
				ClusterFSID:   "8668f062-3faa-358a-85f3-f80fe6c1e306",
				HostDirectory: "/var/lib/rook/rook-ceph/8668f062-3faa-358a-85f3-f80fe6c1e306_69481cd1-38b1-42fd-ac07-06bf4d7c0e19",
				DeviceMapping: map[string]lcmv1alpha1.DeviceInfo{
					"/dev/vdb": {
						ID:         "b4eaf39c-b561-4269-1",
						Rotational: true,
						Path:       "/dev/disk/by-path/pci-0000:00:0a.0",
						Partition:  "/dev/ceph-cf7c8b53-27c7-4cfc-94de-6ad4c7d9f92d/osd-block-af39b794-e1c6-41c0-8997-d6b6c631b8f2",
						Type:       "block",
						Alive:      true,
					},
				},
			},
			"4": {
				UUID:          "ad76cf53-5cb5-48fe-a39a-343734f5ccde",
				ClusterFSID:   "8668f062-3faa-358a-85f3-f80fe6c1e306",
				HostDirectory: "/var/lib/rook/rook-ceph/8668f062-3faa-358a-85f3-f80fe6c1e306_ad76cf53-5cb5-48fe-a39a-343734f5ccde",
				DeviceMapping: map[string]lcmv1alpha1.DeviceInfo{
					"/dev/vdd": {
						ID:         "35a15532-8b56-4f83-9",
						Rotational: false,
						Path:       "/dev/disk/by-path/pci-0000:00:1e.0",
						Partition:  "/dev/ceph-dada9f25-41b4-4c26-9a20-448ac01e1d06/osd-block-ad76cf53-5cb5-48fe-a39a-343734f5ccde",
						Type:       "block",
						Alive:      true,
					},
				},
			},
			"5": {
				UUID:          "af39b794-e1c6-41c0-8997-d6b6c631b8f2",
				ClusterFSID:   "8668f062-3faa-358a-85f3-f80fe6c1e306",
				HostDirectory: "/var/lib/rook/rook-ceph/8668f062-3faa-358a-85f3-f80fe6c1e306_af39b794-e1c6-41c0-8997-d6b6c631b8f2",
				DeviceMapping: map[string]lcmv1alpha1.DeviceInfo{
					"/dev/vdd": {
						ID:         "35a15532-8b56-4f83-9",
						Rotational: false,
						Path:       "/dev/disk/by-path/pci-0000:00:1e.0",
						Partition:  "/dev/ceph-dada9f25-41b4-4c26-9a20-448ac01e1d06/osd-block-7d09cceb-4de0-478e-9d8d-bd09cb0c904e",
						Type:       "block",
						Alive:      true,
					},
				},
			},
		},
	},
}

var FullNodesInfoFromOsdMeta = map[string]lcmv1alpha1.HostMapping{
	"node-1": {
		OsdMapping: map[string]lcmv1alpha1.OsdMapping{
			"20": {
				UUID:          "vbsgs3a3-sdcv-casq-sd11-asd12dasczsf",
				ClusterFSID:   "8668f062-3faa-358a-85f3-f80fe6c1e306",
				HostDirectory: "/var/lib/rook/rook-ceph/8668f062-3faa-358a-85f3-f80fe6c1e306_vbsgs3a3-sdcv-casq-sd11-asd12dasczsf",
				DeviceMapping: map[string]lcmv1alpha1.DeviceInfo{
					"/dev/vde": {
						Rotational: true,
						Path:       "/dev/disk/by-path/pci-0000:00:0f.0",
						Partition:  "/dev/dm-0",
						ID:         "2926ff77-7491-4447-a",
						Type:       "block",
					},
					"/dev/vdd": {
						Rotational: true,
						Path:       "/dev/disk/by-path/pci-0000:00:0e.0",
						Partition:  "/dev/dm-1",
						ID:         "e8d89e2f-ffc6-4988-9",
						Type:       "db",
					},
				},
			},
			"25": {
				UUID:          "d49fd9bf-d2dd-4c3d-824d-87f3f17ea44a",
				ClusterFSID:   "8668f062-3faa-358a-85f3-f80fe6c1e306",
				HostDirectory: "/var/lib/rook/rook-ceph/8668f062-3faa-358a-85f3-f80fe6c1e306_d49fd9bf-d2dd-4c3d-824d-87f3f17ea44a",
				DeviceMapping: map[string]lcmv1alpha1.DeviceInfo{
					"/dev/vdf": {
						Rotational: true,
						Path:       "/dev/disk/by-path/pci-0000:00:10.0",
						Partition:  "/dev/dm-2",
						ID:         "b7ea1c8c-89b8-4354-8",
						Type:       "block",
					},
					"/dev/vdd": {
						Rotational: true,
						Path:       "/dev/disk/by-path/pci-0000:00:0e.0",
						Partition:  "/dev/dm-3",
						ID:         "e8d89e2f-ffc6-4988-9",
						Type:       "db",
					},
				},
			},
			"30": {
				UUID:          "f4edb5cd-fb1e-4620-9419-3f9a4fcecba5",
				ClusterFSID:   "8668f062-3faa-358a-85f3-f80fe6c1e306",
				HostDirectory: "/var/lib/rook/rook-ceph/8668f062-3faa-358a-85f3-f80fe6c1e306_f4edb5cd-fb1e-4620-9419-3f9a4fcecba5",
				DeviceMapping: map[string]lcmv1alpha1.DeviceInfo{
					"/dev/vda": {
						Rotational: true,
						Path:       "/dev/disk/by-path/pci-0000:00:09.0",
						Partition:  "/dev/vda14",
						ID:         "8dad5ae9-ddf7-40bf-8",
						Type:       "db",
					},
					"/dev/vdb": {
						Rotational: true,
						Path:       "/dev/disk/by-path/pci-0000:00:0a.0",
						Partition:  "/dev/dm-4",
						ID:         "996ea59f-7f47-4fac-b",
						Type:       "block",
					},
				},
			},
		},
	},
	"node-2": {
		OsdMapping: map[string]lcmv1alpha1.OsdMapping{
			"0": {
				UUID:          "69481cd1-38b1-42fd-ac07-06bf4d7c0e19",
				ClusterFSID:   "8668f062-3faa-358a-85f3-f80fe6c1e306",
				HostDirectory: "/var/lib/rook/rook-ceph/8668f062-3faa-358a-85f3-f80fe6c1e306_69481cd1-38b1-42fd-ac07-06bf4d7c0e19",
				DeviceMapping: map[string]lcmv1alpha1.DeviceInfo{
					"/dev/vdb": {
						Rotational: true,
						Path:       "/dev/disk/by-path/pci-0000:00:0a.0",
						ID:         "b4eaf39c-b561-4269-1",
						Partition:  "/dev/dm-0",
						Type:       "block",
					},
				},
			},
			"4": {
				UUID:          "ad76cf53-5cb5-48fe-a39a-343734f5ccde",
				ClusterFSID:   "8668f062-3faa-358a-85f3-f80fe6c1e306",
				HostDirectory: "/var/lib/rook/rook-ceph/8668f062-3faa-358a-85f3-f80fe6c1e306_ad76cf53-5cb5-48fe-a39a-343734f5ccde",
				DeviceMapping: map[string]lcmv1alpha1.DeviceInfo{
					"/dev/vdd": {
						Rotational: false,
						Path:       "/dev/disk/by-path/pci-0000:00:1e.0",
						Partition:  "/dev/dm-2",
						ID:         "35a15532-8b56-4f83-9",
						Type:       "block",
					},
				},
			},
			"5": {
				UUID:          "af39b794-e1c6-41c0-8997-d6b6c631b8f2",
				ClusterFSID:   "8668f062-3faa-358a-85f3-f80fe6c1e306",
				HostDirectory: "/var/lib/rook/rook-ceph/8668f062-3faa-358a-85f3-f80fe6c1e306_af39b794-e1c6-41c0-8997-d6b6c631b8f2",
				DeviceMapping: map[string]lcmv1alpha1.DeviceInfo{
					"/dev/vdd": {
						Rotational: false,
						Path:       "/dev/disk/by-path/pci-0000:00:1e.0",
						Partition:  "/dev/dm-3",
						ID:         "35a15532-8b56-4f83-9",
						Type:       "block",
					},
				},
			},
		},
	},
}
