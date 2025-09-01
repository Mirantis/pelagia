/*
Copyright 2025 Mirantis IT.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless taskuired by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package lcmcommon

type OsdMetadataInfo struct {
	Devices             string `json:"devices"`
	DevicePathes        string `json:"device_paths"`
	DeviceIDs           string `json:"device_ids"`
	Hostname            string `json:"hostname"`
	BluestoreDevices    string `json:"bluestore_bdev_devices"`
	BluestoreDeviceType string `json:"bluestore_bdev_type"`
	BluestorePartition  string `json:"bluestore_bdev_partition_path"`
	MetadataDiskUsed    string `json:"bluefs_dedicated_db"`
	MetadataDevices     string `json:"bluefs_db_devices"`
	MetadataDeviceType  string `json:"bluefs_db_type"`
	MetadataPartition   string `json:"bluefs_db_partition_path"`
	OsdID               int    `json:"id"`
}

type OsdInfo struct {
	OsdID int    `json:"osd"`
	UUID  string `json:"uuid"`
	Up    int    `json:"up"`
	In    int    `json:"in"`
}

type OsdTree struct {
	Nodes []struct {
		ID          int     `json:"id"`
		Name        string  `json:"name"`
		Type        string  `json:"type"`
		Children    []int   `json:"children,omitempty"`
		DeviceClass string  `json:"device_class"`
		Status      string  `json:"status"`
		Weight      float64 `json:"crush_weight"`
		Reweight    int     `json:"reweight"`
	} `json:"nodes"`
}

type MgrModuleLs struct {
	AlwaysOn []string `json:"always_on_modules"`
	Enabled  []string `json:"enabled_modules"`
}

type ZoneGroupInfo struct {
	Hostnames []string `json:"hostnames"`
}

type CephDetails struct {
	StatsByClass map[string]ClassStats `json:"stats_by_class"`
	Pools        []PoolDetails         `json:"pools"`
}

type ClassStats struct {
	TotalBytes     uint64 `json:"total_bytes"`
	UsedBytes      uint64 `json:"total_used_bytes"`
	AvailableBytes uint64 `json:"total_avail_bytes"`
}

type PoolDetails struct {
	Name  string    `json:"name"`
	Stats PoolStats `json:"stats"`
}

type PoolStats struct {
	TotalBytes  uint64  `json:"max_avail"`
	UsedBytes   uint64  `json:"bytes_used"`
	PercentUsed float64 `json:"percent_used"`
}
