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

package lcmcommon

type BlockDeviceInfo struct {
	// pkg kernel device name
	Kname string `json:"kname"`
	// device serial number
	Serial string `json:"serial,omitempty"`
	// device type: disk, part, lvm, raid
	Type string `json:"type"`
	// rotational device
	Rotational bool `json:"rota"`
	// major:minor device number
	MajMin string `json:"maj:min"`
	// aliases for block device by-{id,path,uuid}
	Symlinks []string `json:"symlinks"`
	// pkg parent kernel device name
	Parent []string `json:"parent,omitempty"`
	// childrens for device
	Childrens []string `json:"children,omitempty"`
}

type DiskDaemonState string

const (
	DiskDaemonStateOk         DiskDaemonState = "ok"
	DiskDaemonStateFailed     DiskDaemonState = "failed"
	DiskDaemonStateInProgress DiskDaemonState = "preparing"
	DiskDaemonStateSkipped    DiskDaemonState = "skipped"
)

type DiskDaemonReport struct {
	// current osd report state
	State DiskDaemonState `json:"state"`
	// current issues for node
	Issues []string `json:"issues,omitempty"`
	// current ready disk report
	DisksReport *DiskDaemonDisksReport `json:"disks_report,omitempty"`
	// current ready osd disk usage report
	OsdsReport *DiskDaemonOsdsReport `json:"osds_report,omitempty"`
}

type DiskDaemonDisksReport struct {
	// parsed info from lsblk and udevadm cmds
	BlockInfo map[string]BlockDeviceInfo `json:"block_info"`
	// aliases map to block device name
	Aliases map[string]string `json:"aliases"`
	// Map for quick search disk -> osd on it
	DiskToOsd map[string][]string `json:"disk_to_osd_map,omitempty"`
}

type DiskDaemonOsdsReport struct {
	// warnings faced during osd report prepare
	Warnings []string `json:"warnings,omitempty"`
	// regular osd devices info
	Osds map[string][]OsdDaemonInfo `json:"osds,omitempty"`
}

type OsdDaemonInfo struct {
	OsdUUID     string         `json:"osd_uuid,omitempty"`
	ClusterFSID string         `json:"osd_fsid,omitempty"`
	Devices     []OsdDevice    `json:"osd_device,omitempty"`
	Partitions  []OsdPartition `json:"osd_partitions,omitempty"`
}

type OsdDevice struct {
	Name             string   `json:"name,omitempty"`
	DeviceID         string   `json:"device_id,omitempty"`
	DeviceSymlinks   []string `json:"device_pathes,omitempty"`
	Rotational       bool     `json:"rotational,omitempty"`
	RelatedPartition string   `json:"partition,omitempty"`
	PartedBy         string   `json:"parted_by,omitempty"`
}

type OsdPartition struct {
	Partition         string   `json:"partition,omitempty"`
	PartitionSymlinks []string `json:"partition_symlinks,omitempty"`
	Type              string   `json:"type,omitempty"`
	Exists            bool     `json:"exists,omitempty"`
	Lvm               bool     `json:"lvm,omitempty"`
}
