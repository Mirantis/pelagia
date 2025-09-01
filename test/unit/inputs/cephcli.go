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
	"fmt"
	"strings"
)

var CephStatusBaseHealthy = BuildCliOutput(CephStatusTmpl, "status", nil)
var CephStatusBaseUnhealthy = BuildCliOutput(CephStatusTmpl, "status", map[string]string{"quorum_names": `["a", "b"]`, "osdmap": `{"num_osds": 3, "num_up_osds": 2, "num_in_osds": 2}`})
var CephStatusCephFsRgwHealthy = BuildCliOutput(CephStatusTmpl, "status", map[string]string{
	"fsmap":      `{"by_rank": [{"name": "cephfs-1-a", "status": "up:active"}],"up:standby": 1}`,
	"servicemap": `{"services": {"rgw": {"daemons": {"11556688": {"gid": 11556688},"12065099":{"gid": 12065099},"summary": ""}}}}`,
})
var CephStatusCephFsRgwUnhealthy = BuildCliOutput(CephStatusTmpl, "status", map[string]string{
	"fsmap": `{"by_rank": [{"name": "cephfs-1-a", "status": "down:inactive"}],"up:standby": 0}`,
})
var CephStatusCephFewFsRgwHealthy = BuildCliOutput(CephStatusTmpl, "status", map[string]string{
	"fsmap":      `{"by_rank": [{"name": "cephfs-1-a", "status": "up:active"}, {"name": "cephfs-2-a", "status": "up:active"}, {"name": "cephfs-2-b", "status": "up:standby-replay"}],"up:standby": 1}`,
	"servicemap": `{"services": {"rgw": {"daemons": {"10223488": {"gid": 10223488},"11556688": {"gid": 11556688},"12065099":{"gid": 12065099},"summary": ""}}}}`,
})
var CephStatusCephFewFsRgwUnhealthy = BuildCliOutput(CephStatusTmpl, "status", map[string]string{
	"fsmap":      `{"by_rank": [{"name": "cephfs-1-a", "status": "down:inactive"}, {"name": "cephfs-2-a", "status": "up:active"}, {"name": "cephfs-3-a", "status": "down:inactive"}],"up:standby": 0}`,
	"servicemap": `{"services": {"rgw": {"daemons": {"10223488": {"gid": 10223488},"11556688": {"gid": 11556688},"12065099":{"gid": 12065099}, "12065109":{"gid": 12065109},"summary": ""}}}}`,
})
var CephStatusWithEvents = BuildCliOutput(CephStatusTmpl, "status", map[string]string{"progress_events": `{
  "12b640c7-9734-429e-a67d-a00ab20a7635": {
    "message":"Rebalancing after osd.3 marked in (33s)\n      [==========================..] (remaining: 1s)",
    "progress":0.94805192947387695
  },
  "eb643ce4-af7d-4297-b136-0cbddb5cd14f":{
    "message":"PG autoscaler increasing pool 9 PGs from 32 to 128 (0s)\n      [............................] ",
    "progress":0.52945859385684473
  }
}`})

var CephMgrDumpBaseHealthy = BuildCliOutput(CephMgrDumpTmpl, "mgr dump", nil)
var CephMgrDumpBaseUnhealthy = BuildCliOutput(CephMgrDumpTmpl, "mgr dump", map[string]string{"available": "false"})
var CephMgrDumpHAHealthy = BuildCliOutput(CephMgrDumpTmpl, "mgr dump", map[string]string{"standbys": `[{"name": "b"}]`})
var CephMgrDumpHAUnealthy = BuildCliOutput(CephMgrDumpTmpl, "mgr dump", map[string]string{"activename": `"b"`})

var CephOsdCrushRuleDump = BuildCliOutput(CephCrushRuleDumpTmpl, "osd crush rule dump", nil)

var CephStatusTmpl = `{
  "quorum_names": {quorum_names},
  "monmap": {monmap},
  "osdmap": {osdmap},
  "fsmap": {fsmap},
  "servicemap": {servicemap},
  "progress_events": {progress_events}
}`

var CephMgrDumpTmpl = `{
  "active_name": {activename},
  "available": {available},
  "standbys": {standbys}
}`

func BuildCliOutput(template string, cmd string, overrideForOutput map[string]string) string {
	replaceStatusParams := map[string]string{}
	switch cmd {
	case "status":
		replaceStatusParams = map[string]string{
			"quorum_names":    `["a", "b", "c"]`,
			"monmap":          `{"min_mon_release_name": "reef", "num_mons": 3}`,
			"osdmap":          `{"num_osds": 3, "num_up_osds": 3, "num_in_osds": 3}`,
			"fsmap":           `{"by_rank": [], "up:standby": 0}`,
			"servicemap":      `{"services": {}}`,
			"progress_events": "{}",
		}
	case "mgr dump":
		replaceStatusParams = map[string]string{
			"activename": `"a"`,
			"available":  "true",
			"standbys":   "[]",
		}
	case "osd crush rule dump":
		replaceStatusParams = map[string]string{
			"pool1_deviceclass":   "default~hdd",
			"pool1_failuredomain": "host",
			"pool2_deviceclass":   "default~hdd",
			"pool2_failuredomain": "host",
			"pool3_deviceclass":   "default~hdd",
			"pool3_failuredomain": "host",
		}
	}
	for k, v := range overrideForOutput {
		replaceStatusParams[k] = v
	}
	args := []string{}
	for k, v := range replaceStatusParams {
		if k == "" {
			continue
		}
		if v == "" {
			v = "{}"
		}
		args = append(args, fmt.Sprintf("{%s}", k), v)
	}
	return strings.NewReplacer(args...).Replace(template)
}

var CephDfBase = `{
    "stats": {
        "total_bytes": 509981204480,
        "total_avail_bytes": 428350242816,
        "total_used_bytes": 81630961664,
        "total_used_raw_bytes": 81630961664,
        "total_used_raw_ratio": 0.16006660461425781,
        "num_osds": 8,
        "num_per_pool_osds": 8,
        "num_per_pool_omap_osds": 8
    },
    "stats_by_class": {
        "hdd": {
            "total_bytes": 509981204480,
            "total_avail_bytes": 428350242816,
            "total_used_bytes": 81630961664,
            "total_used_raw_bytes": 81630961664,
            "total_used_raw_ratio": 0.16006660461425781
        }
    },
    "pools": [
        {
            "name": "pool-hdd",
            "id": 1,
            "stats": {
                "stored": 19,
                "objects": 1,
                "kb_used": 12,
                "bytes_used": 12288,
                "percent_used": 3.9081324842982212e-08,
                "max_avail": 104807096320
            }
        },
        {
            "name": ".mgr",
            "id": 2,
            "stats": {
                "stored": 459280,
                "objects": 2,
                "kb_used": 1356,
                "bytes_used": 1388544,
                "percent_used": 4.4161702135170344e-06,
                "max_avail": 104807096320
            }
        }
    ]
}`

var CephDfExtraPools = `{
    "stats": {
        "total_bytes": 563664101376,
        "total_avail_bytes": 481489813504,
        "total_used_bytes": 82174287872,
        "total_used_raw_bytes": 82174287872,
        "total_used_raw_ratio": 0.14578591287136078,
        "num_osds": 9,
        "num_per_pool_osds": 9,
        "num_per_pool_omap_osds": 8
    },
    "stats_by_class": {
        "hdd": {
            "total_bytes": 509981204480,
            "total_avail_bytes": 427884044288,
            "total_used_bytes": 82097160192,
            "total_used_raw_bytes": 82097160192,
            "total_used_raw_ratio": 0.16098076105117798
        },
        "ssd": {
            "total_bytes": 53682896896,
            "total_avail_bytes": 53605769216,
            "total_used_bytes": 77127680,
            "total_used_raw_bytes": 77127680,
            "total_used_raw_ratio": 0.0014367272378876805
        }
    },
    "pools": [
        {
            "name": "pool-hdd",
            "id": 1,
            "stats": {
                "stored": 83894984,
                "objects": 32,
                "kb_used": 245820,
                "bytes_used": 251719680,
                "percent_used": 0.00080068089300766587,
                "max_avail": 104710103040
            }
        },
        {
            "name": ".mgr",
            "id": 2,
            "stats": {
                "stored": 918560,
                "objects": 2,
                "kb_used": 2712,
                "bytes_used": 2777088,
                "percent_used": 8.8404822236043401e-06,
                "max_avail": 104710103040
            }
        },
        {
            "name": "my-cephfs-metadata",
            "id": 29,
            "stats": {
                "stored": 26994,
                "objects": 22,
                "kb_used": 112,
                "bytes_used": 114688,
                "percent_used": 3.6509675283014076e-07,
                "max_avail": 157065150464
            }
        },
        {
            "name": "rgw-store.rgw.buckets.non-ec",
            "id": 30,
            "stats": {
                "stored": 0,
                "objects": 0,
                "kb_used": 0,
                "bytes_used": 0,
                "percent_used": 0,
                "max_avail": 104710103040
            }
        },
        {
            "name": "rgw-store.rgw.control",
            "id": 31,
            "stats": {
                "stored": 0,
                "objects": 8,
                "kb_used": 0,
                "bytes_used": 0,
                "percent_used": 0,
                "max_avail": 104710103040
            }
        },
        {
            "name": "rgw-store.rgw.buckets.index",
            "id": 32,
            "stats": {
                "stored": 0,
                "objects": 0,
                "kb_used": 0,
                "bytes_used": 0,
                "percent_used": 0,
                "max_avail": 104710103040
            }
        },
        {
            "name": "rgw-store.rgw.meta",
            "id": 33,
            "stats": {
                "stored": 867,
                "objects": 6,
                "kb_used": 48,
                "bytes_used": 49152,
                "percent_used": 1.5647006534891261e-07,
                "max_avail": 104710103040
            }
        },
        {
            "name": ".rgw.root",
            "id": 34,
            "stats": {
                "stored": 4874,
                "objects": 17,
                "kb_used": 192,
                "bytes_used": 196608,
                "percent_used": 6.2588003402197501e-07,
                "max_avail": 104710103040
            }
        },
        {
            "name": "my-cephfs-data-1",
            "id": 35,
            "stats": {
                "stored": 0,
                "objects": 0,
                "kb_used": 0,
                "bytes_used": 0,
                "percent_used": 0,
                "max_avail": 157065150464
            }
        },
        {
            "name": "rgw-store.rgw.otp",
            "id": 36,
            "stats": {
                "stored": 0,
                "objects": 0,
                "kb_used": 0,
                "bytes_used": 0,
                "percent_used": 0,
                "max_avail": 104710103040
            }
        },
        {
            "name": "rgw-store.rgw.log",
            "id": 37,
            "stats": {
                "stored": 23602,
                "objects": 307,
                "kb_used": 1944,
                "bytes_used": 1990656,
                "percent_used": 6.3369989220518619e-06,
                "max_avail": 104710103040
            }
        },
        {
            "name": "my-cephfs-data-2",
            "id": 38,
            "stats": {
                "stored": 0,
                "objects": 0,
                "kb_used": 0,
                "bytes_used": 0,
                "percent_used": 0,
                "max_avail": 209420206080
            }
        },
        {
            "name": "rgw-store.rgw.buckets.data",
            "id": 39,
            "stats": {
                "stored": 0,
                "objects": 0,
                "kb_used": 0,
                "bytes_used": 0,
                "percent_used": 0,
                "max_avail": 209420206080
            }
        }
    ]
}`

var CephOsdTreeForSizingCheck = `{
  "nodes":[
    {"id":-1,"name":"default","type":"root","type_id":11,"children":[-15,-17]},
    {"id":-17,"name":"rack-1","type":"rack","type_id":3,"pool_weights":{},"children":[-13,-11,-3]},
    {"id":-3,"name":"de-ps-rjshyprsmxpi-0-tc7ms3qx6x6c-server-ptlqq6wjm4oh","type":"host","type_id":1,"pool_weights":{},"children":[3]},
    {"id":3,"device_class":"ssd","name":"osd.3","type":"osd","type_id":0,"crush_weight":0.048797607421875,"depth":3,"pool_weights":{},"exists":1,"status":"up","reweight":1,"primary_affinity":1},
    {"id":-11,"name":"de-ps-rjshyprsmxpi-1-rwf5aumltlse-server-me4x3bovgfgv","type":"host","type_id":1,"pool_weights":{},"children":[5]},
    {"id":5,"device_class":"hdd","name":"osd.5","type":"osd","type_id":0,"crush_weight":0.048797607421875,"depth":3,"pool_weights":{},"exists":1,"status":"up","reweight":1,"primary_affinity":1},
    {"id":-13,"name":"de-ps-rjshyprsmxpi-2-4ymltwv3jhpz-server-qkvgrh3vxesh","type":"host","type_id":1,"pool_weights":{},"children":[4]},
    {"id":4,"device_class":"hdd","name":"osd.4","type":"osd","type_id":0,"crush_weight":0.048797607421875,"depth":3,"pool_weights":{},"exists":1,"status":"up","reweight":1,"primary_affinity":1},
    {"id":-15,"name":"rack-2","type":"rack","type_id":3,"pool_weights":{},"children":[-7,-25,-9]},
    {"id":-9,"name":"de-ds-r6l4djqhmmfn-0-mmk3bbmxtq53-server-xuz6ryuh7qbg","type":"host","type_id":1,"pool_weights":{},"children":[1,0]},
    {"id":0,"device_class":"hdd","name":"osd.0","type":"osd","type_id":0,"crush_weight":0.0731964111328125,"depth":3,"pool_weights":{},"exists":1,"status":"up","reweight":1,"primary_affinity":1},
    {"id":1,"device_class":"hdd","name":"osd.1","type":"osd","type_id":0,"crush_weight":0.048797607421875,"depth":3,"pool_weights":{},"exists":1,"status":"up","reweight":1,"primary_affinity":1},
    {"id":-25,"name":"de-ds-r6l4djqhmmfn-1-xastfhatmjqc-server-g7g7co5e467q","type":"host","type_id":1,"pool_weights":{},"children":[7]},
    {"id":7,"device_class":"ssd","name":"osd.7","type":"osd","type_id":0,"crush_weight":0.048797607421875,"depth":3,"pool_weights":{},"exists":1,"status":"up","reweight":1,"primary_affinity":1},
    {"id":-7,"name":"de-ds-r6l4djqhmmfn-2-xupcpjofrkgm-server-5baxrpw2ouy3","type":"host","type_id":1,"pool_weights":{},"children":[8,6,2]},
    {"id":2,"device_class":"hdd","name":"osd.2","type":"osd","type_id":0,"crush_weight":0.0731964111328125,"depth":3,"pool_weights":{},"exists":1,"status":"up","reweight":1,"primary_affinity":1},
    {"id":6,"device_class":"hdd","name":"osd.6","type":"osd","type_id":0,"crush_weight":0.0731964111328125,"depth":3,"pool_weights":{},"exists":1,"status":"up","reweight":1,"primary_affinity":1},
    {"id":8,"device_class":"ssd","name":"osd.8","type":"osd","type_id":0,"crush_weight":0.048797607421875,"depth":3,"pool_weights":{},"exists":1,"status":"up","reweight":1,"primary_affinity":1}
  ],
    "stray":[]
}
`

var CephOsdTreeOutputTmpl = `{
    "nodes": [
        {
            "id": -1,
            "name": "default",
            "type": "root",
            "type_id": 11,
            "children": [
                -5,
                -3
            ]
        },
        {
            "id": -5,
            "name": "node-2",
            "type": "host",
            "type_id": 1,
            "pool_weights": {},
            "children": [{childs_2}]
        },
        {
            "id": -3,
            "name": "node-1",
            "type": "host",
            "type_id": 1,
            "pool_weights": {},
            "children": [{childs_1}]
        }
    ]
}`

var CephOsdTreeOutput = BuildCliOutput(CephOsdTreeOutputTmpl, "", map[string]string{"childs_1": "20,25,30", "childs_2": "0,4,5"})
var CephOsdTreeOutputNoOsdsOnHost = BuildCliOutput(CephOsdTreeOutputTmpl, "", map[string]string{"childs_1": "\n", "childs_2": "\n"})

var CephPoolsDetails = `[
  {"pool_name": "pool-1", "size": 3, "crush_rule": 2},
  {"pool_name": "pool-2", "size": 3, "crush_rule": 3},
  {"pool_name": "pool-3", "size": 3, "crush_rule": 5}
]`

var CephCrushRuleDumpTmpl = `[
    {
        "rule_id": 0,
        "rule_name": "replicated_rule",
        "type": 1,
        "steps": [
            {
                "op": "take",
                "item": -1,
                "item_name": "default"
            },
            {
                "op": "chooseleaf_firstn",
                "num": 0,
                "type": "host"
            },
            {
                "op": "emit"
            }
        ]
    },
    {
        "rule_id": 2,
        "rule_name": "pool-1_rule",
        "type": 1,
        "steps": [
            {
                "op": "take",
                "item": -2,
                "item_name": "{pool1_deviceclass}"
            },
            {
                "op": "chooseleaf_firstn",
                "num": 0,
                "type": "{pool1_failuredomain}"
            },
            {
                "op": "emit"
            }
        ]
    },
    {
        "rule_id": 3,
        "rule_name": "pool-2_rule",
        "type": 1,
        "steps": [
            {
                "op": "take",
                "item": -1,
                "item_name": "{pool2_deviceclass}"
            },
            {
                "op": "chooseleaf_firstn",
                "num": 0,
                "type": "{pool2_failuredomain}"
            },
            {
                "op": "emit"
            }
        ]
    },
    {
        "rule_id": 5,
        "rule_name": "pool-3_rule",
        "type": 1,
        "steps": [
            {
                "op": "set_chooseleaf_tries",
                "num": 5
            },
            {
                "op": "set_choose_tries",
                "num": 100
            },
            {
                "op": "take",
                "item": -2,
                "item_name": "{pool3_deviceclass}"
            },
            {
                "op": "chooseleaf_indep",
                "num": 0,
                "type": "{pool3_failuredomain}"
            },
            {
                "op": "emit"
            }
        ]
    }
]`

var RadosgwAdminMasterSyncStatusOk = `
          realm a46a61a7-46c0-41dd-8f62-9f989b9de803 (rgw-store)
      zonegroup 5c6c92c1-632c-4db0-8aa9-8dcbea5d87ec (rgw-store)
           zone 362d9d90-1151-41a0-80aa-e8aa6d036730 (rgw-store)
      current time 2024-04-18T13:08:34Z
      zonegroup features enabled: resharding
                   disabled: compress-encrypted
      metadata sync no sync (zone is master)
      data sync source: 4abcf593-157b-46bb-8209-0f8f7f5a7e8e (rgw-store-backup)
                        syncing
                        full sync: 0/128 shards
                        incremental sync: 128/128 shards
                        data is caught up with source
`

var RadosgwAdminSecondarySyncStatusOk = `
          realm a46a61a7-46c0-41dd-8f62-9f989b9de803 (rgw-store)
      zonegroup 5c6c92c1-632c-4db0-8aa9-8dcbea5d87ec (rgw-store)
           zone 362d9d90-1151-41a0-80aa-e8aa6d036730 (rgw-store-backup)
   current time 2024-04-18T13:08:34Z
zonegroup features enabled: resharding
                   disabled: compress-encrypted
  metadata sync syncing
                full sync: 0/64 shards
                incremental sync: 64/64 shards
                metadata is caught up with master
      data sync source: 4abcf593-157b-46bb-8209-0f8f7f5a7e8e (rgw-store)
                        syncing
                        full sync: 0/128 shards
                        incremental sync: 128/128 shards
                        data is caught up with source
`

/*
Output contains next setup:
* node-1 - 3 osd on different disks, with metadata:
  - vdd is metadata, vde and vdf are block devices;
  - vda14 is physical metadata partition for legacy envs check, vdb is block device;
* node-2 - 3 osd w/o metadata:
  - vdd contains 2 osds;
  - vdb single osd;
* one stray osd - optionally
*/

var CephOsdMetadataOutputTmpl = `[
  {
    "devices": "vdb",
    "bluestore_bdev_devices": "vdb",
    "bluestore_bdev_type": "hdd",
    "bluestore_bdev_partition_path": "/dev/dm-0",
    "bluefs_dedicated_db": "0",
    "id": 0,
    "hostname": "node-2",
    "device_ids": "vdb=b4eaf39c-b561-4269-1",
    "device_paths": "vdb=/dev/disk/by-path/pci-0000:00:0a.0"
  },
  {stray}
  {
    "devices": "vdd",
    "bluestore_bdev_devices": "vdd",
    "bluestore_bdev_type": "ssd",
    "bluestore_bdev_partition_path": "/dev/dm-2",
    "bluefs_dedicated_db": "0",
    "id": 4,
    "hostname": "node-2",
    "device_ids": "vdd=35a15532-8b56-4f83-9",
    "device_paths": "vdd=/dev/disk/by-path/pci-0000:00:1e.0"
  },
  {
    "devices": "vdd",
    "bluestore_bdev_devices": "vdd",
    "bluestore_bdev_type": "ssd",
    "bluestore_bdev_partition_path": "/dev/dm-3",
    "bluefs_dedicated_db": "0",
    "id": 5,
    "hostname": "node-2",
    "device_ids": "vdd=35a15532-8b56-4f83-9",
    "device_paths": "vdd=/dev/disk/by-path/pci-0000:00:1e.0"
  },
  {
    "devices": "vde,vdd",
    "bluestore_bdev_devices": "vde",
    "bluestore_bdev_type": "hdd",
    "bluestore_bdev_partition_path": "/dev/dm-0",
    "bluefs_dedicated_db": "1",
    "bluefs_db_devices": "vdd",
    "bluefs_db_type": "hdd",
    "bluefs_db_partition_path": "/dev/dm-1",
    "id": 20,
    "hostname": "node-1",
    "device_ids": "vde=2926ff77-7491-4447-a,vdd=e8d89e2f-ffc6-4988-9",
    "device_paths": "vde=/dev/disk/by-path/pci-0000:00:0f.0,vdd=/dev/disk/by-path/pci-0000:00:0e.0"
  },
  {
    "devices": "vdf,vdd",
    "bluestore_bdev_devices": "vdf",
    "bluestore_bdev_type": "hdd",
    "bluestore_bdev_partition_path": "/dev/dm-2",
    "bluefs_dedicated_db": "1",
    "bluefs_db_devices": "vdd",
    "bluefs_db_type": "hdd",
    "bluefs_db_partition_path": "/dev/dm-3",
    "id": 25,
    "hostname": "node-1",
    "device_ids": "vdf=b7ea1c8c-89b8-4354-8,vdd=e8d89e2f-ffc6-4988-9",
    "device_paths": "vdf=/dev/disk/by-path/pci-0000:00:10.0,vdd=/dev/disk/by-path/pci-0000:00:0e.0"
  },
  {
    "devices": "vdb,vda",
    "bluestore_bdev_devices": "vdb",
    "bluestore_bdev_type": "hdd",
    "bluestore_bdev_partition_path": "/dev/dm-4",
    "bluefs_dedicated_db": "1",
    "bluefs_db_devices": "vda",
    "bluefs_db_type": "hdd",
    "bluefs_db_partition_path": "/dev/vda14",
    "id": 30,
    "hostname": "node-1",
    "device_ids": "vdb=996ea59f-7f47-4fac-b,vda=8dad5ae9-ddf7-40bf-8",
    "device_paths": "vdb=/dev/disk/by-path/pci-0000:00:0a.0,vda=/dev/disk/by-path/pci-0000:00:09.0"
  }
]`

var CephOsdMetadataOutput = BuildCliOutput(CephOsdMetadataOutputTmpl, "", map[string]string{"stray": `{"id": 2},`})
var CephOsdMetadataOutputNoStray = BuildCliOutput(CephOsdMetadataOutputTmpl, "", map[string]string{"stray": "\n"})

var CephOsdInfoOutputTmpl = `[
    {
        "osd":  0,
        "uuid": "69481cd1-38b1-42fd-ac07-06bf4d7c0e19",
        "up":   1,
        "in":   1
    },
    {stray}
    {
        "osd":  4,
        "uuid": "ad76cf53-5cb5-48fe-a39a-343734f5ccde",
        "up":   1,
        "in":   1
    },
    {
        "osd":  5,
        "uuid": "af39b794-e1c6-41c0-8997-d6b6c631b8f2",
        "up":   1,
        "in":   1
    },
    {
        "osd":  20,
        "uuid": "vbsgs3a3-sdcv-casq-sd11-asd12dasczsf",
        "up":   1,
        "in":   1
    },
    {
        "osd":  25,
        "uuid": "d49fd9bf-d2dd-4c3d-824d-87f3f17ea44a",
        "up":   1,
        "in":   1
    },
    {
        "osd":  30,
        "uuid": "f4edb5cd-fb1e-4620-9419-3f9a4fcecba5",
        "up":   1,
        "in":   1
    }
]`

var CephOsdInfoOutput = BuildCliOutput(CephOsdInfoOutputTmpl, "", map[string]string{"stray": `{"osd":  2,"uuid": "61869d90-2c45-4f02-b7c3-96955f41e2ca"},`})
var CephOsdInfoOutputNoStray = BuildCliOutput(CephOsdInfoOutputTmpl, "", map[string]string{"stray": "\n"})

var CephOsdLspools = `["kubernetes-hdd", ".rgw.root", "openstack-store.rgw.buckets.non-ec", "openstack-store.rgw.buckets.index", "openstack-store.rgw.meta", "openstack-store.rgw.log", "openstack-store.rgw.control", "openstack-store.rgw.buckets.data", ".mgr"]`
var CephOsdLspoolsWithRgwDefault = `["kubernetes-hdd", "default.rgw.log", "default.rgw.control", "default.rgw.meta", ".rgw.root", "openstack-store.rgw.buckets.non-ec", "openstack-store.rgw.buckets.index", "openstack-store.rgw.meta", "openstack-store.rgw.log", "openstack-store.rgw.control", "openstack-store.rgw.buckets.data", ".mgr"]`

var MgrModuleLsTmpl = `
{
   "always_on_modules":[
      "balancer",
      "crash",
      "devicehealth",
      "orchestrator",
      "pg_autoscaler",
      "progress",
      "rbd_support",
      "status",
      "telemetry",
      "volumes"
   ],
   "enabled_modules":[{modules}],
   "disabled_modules":[]
}`

var MgrModuleLsNoPrometheus = BuildCliOutput(MgrModuleLsTmpl, "", map[string]string{"modules": `"iostat","nfs","restful"`})
var MgrModuleLsWithPrometheus = BuildCliOutput(MgrModuleLsTmpl, "", map[string]string{"modules": `"iostat","nfs","prometheus","restful"`})

var CephZoneGroupInfoHostnamesTmpl = `{
    "id": "c2680072-93a5-4a11-8ee5-7139fcfff96a",
    "name": "rgw-store",
    "api_name": "rgw-store",
    "hostnames": {hostnames}
}`

var CephZoneGroupInfoEmptyHostnames = BuildCliOutput(CephZoneGroupInfoHostnamesTmpl, "", map[string]string{"hostnames": "[]"})
var CephZoneGroupInfoHostnamesFromOpenstack = BuildCliOutput(CephZoneGroupInfoHostnamesTmpl, "", map[string]string{"hostnames": `["rook-ceph-rgw-rgw-store.rook-ceph.svc","rgw-store.openstack.com"]`})
var CephZoneGroupInfoHostnamesFromIngress = BuildCliOutput(CephZoneGroupInfoHostnamesTmpl, "", map[string]string{"hostnames": `["rook-ceph-rgw-rgw-store.rook-ceph.svc","rgw-store.test"]`})
var CephZoneGroupInfoHostnamesFromConfig = BuildCliOutput(CephZoneGroupInfoHostnamesTmpl, "", map[string]string{"hostnames": `["rook-ceph-rgw-rgw-store.rook-ceph.svc","rgw-store.ms2.wxlsd.com"]`})

var CephConfigSectionTmpl = `{"section": "{section}", "name": "{name}", "value": "{value}", "level": "advanced", "can_update_at_runtime": "{runtime_update}", "mask": ""}`
var cephConfigDumpDefaultsStr = fmt.Sprintf("%s, %s, %s, %s",
	BuildCliOutput(CephConfigSectionTmpl, "", map[string]string{"section": "global", "name": "osd_pool_default_pg_autoscale_mode", "value": "on", "runtime_update": "true"}),
	BuildCliOutput(CephConfigSectionTmpl, "", map[string]string{"section": "osd", "name": "bdev_async_discard_threads", "value": "1", "runtime_update": "true"}),
	BuildCliOutput(CephConfigSectionTmpl, "", map[string]string{"section": "osd", "name": "bdev_enable_discard", "value": "true", "runtime_update": "true"}),
	BuildCliOutput(CephConfigSectionTmpl, "", map[string]string{"section": "global", "name": "osd_scrub_auto_repair", "value": "true", "runtime_update": "true"}))

var CephConfigDumpDefaults = fmt.Sprintf("[%s]", cephConfigDumpDefaultsStr)
var CephConfigDumpOverride = fmt.Sprintf("[%s, %s, %s]",
	cephConfigDumpDefaultsStr,
	BuildCliOutput(CephConfigSectionTmpl, "", map[string]string{"section": "global", "name": "osd_max_backfills", "value": "32", "runtime_update": "true"}),
	BuildCliOutput(CephConfigSectionTmpl, "", map[string]string{"section": "global", "name": "osd_recovery_max_active", "value": "16", "runtime_update": "true"}))
var CephConfigDumpOverrideWithRgw = fmt.Sprintf("[%s, %s, %s, %s, %s]",
	cephConfigDumpDefaultsStr,
	BuildCliOutput(CephConfigSectionTmpl, "", map[string]string{"section": "global", "name": "osd_max_backfills", "value": "64", "runtime_update": "true"}),
	BuildCliOutput(CephConfigSectionTmpl, "", map[string]string{"section": "global", "name": "osd_recovery_max_active", "value": "16", "runtime_update": "true"}),
	BuildCliOutput(CephConfigSectionTmpl, "", map[string]string{"section": "client.rgw.rgw.store.a", "name": "rgw_keystone_admin_password", "value": "AMTqaDveAp8sWlLtf0fcg6RVjFRXs7FR", "runtime_update": "false"}),
	BuildCliOutput(CephConfigSectionTmpl, "", map[string]string{"section": "client.rgw.rgw.store.a", "name": "rgw_keystone_barbican_password", "value": "AMTqaDveAp8sWlLtf0fcg6RVjFRXs7FR", "runtime_update": "false"}))

var CephVersionsTemplate = `
{
    "mon": {
        "ceph version %[1]s (stable)": 3
    },
    "mgr": {
        "ceph version %[1]s (stable)": 1
    },
    "osd": {
        "ceph version %[1]s (stable)": 3
    },
    "overall": {
        "ceph version %[1]s (stable)": 7
    }
}
`

var CephVersionsTemplateWithExtraDaemons = `
{
    "mon": {
        "ceph version %[1]s (stable)": 3
    },
    "mgr": {
        "ceph version %[1]s (stable)": 2
    },
    "osd": {
        "ceph version %[1]s (stable)": 3
    },
    "rgw": {
        "ceph version %[1]s (stable)": 2
    },
    "mds": {
        "ceph version %[1]s (stable)": 2
    },
    "overall": {
        "ceph version %[1]s (stable)": 12
    }
}
`

var cephVersionsOutputTmplLatest = fmt.Sprintf("%s (76424b2fe1bb19c32c52140f39764599abf5e035) %s", strings.TrimPrefix(LatestCephVersionImage, "v"), LatestCephVersion)
var CephVersionsLatest = fmt.Sprintf(CephVersionsTemplate, cephVersionsOutputTmplLatest)
var CephVersionsLatestWithExtraDaemons = fmt.Sprintf(CephVersionsTemplateWithExtraDaemons, cephVersionsOutputTmplLatest)
var cephVersionsOutputTmplPrevious = fmt.Sprintf("%s (c44bc49e7a57a87d84dfff2a077a2058aa2172e2) %s", strings.TrimPrefix(PreviousCephVersionImage, "v"), PreviousCephVersion)
var CephVersionsPrevious = fmt.Sprintf(CephVersionsTemplate, cephVersionsOutputTmplPrevious)
var CephVersionsPreviousWithExtraDaemons = fmt.Sprintf(CephVersionsTemplateWithExtraDaemons, cephVersionsOutputTmplPrevious)
