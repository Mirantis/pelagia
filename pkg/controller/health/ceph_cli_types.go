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

package health

type cephStatus struct {
	QuorumNames []string `json:"quorum_names"`
	OsdMap      struct {
		NumOsd   int `json:"num_osds"`
		NumUpOsd int `json:"num_up_osds"`
		NumInOsd int `json:"num_in_osds"`
	} `json:"osdmap"`
	MonMap struct {
		NumMons int `json:"num_mons"`
	} `json:"monmap"`
	MgrMap struct {
		Available bool `json:"available"`
		Standbys  int  `json:"num_standbys"`
	} `json:"mgrmap"`
	ServiceMap struct {
		Services struct {
			Rgw struct {
				Daemons map[string]interface{} `json:"daemons"`
			} `json:"rgw"`
		} `json:"services"`
	} `json:"servicemap"`
	FsMap struct {
		Up      int `json:"up"`
		In      int `json:"in"`
		Max     int `json:"max"`
		Standby int `json:"up:standby"`
		ByRank  []struct {
			Rank   int    `json:"rank"`
			Name   string `json:"name"`
			FsID   int    `json:"filesystem_id"`
			Status string `json:"status"`
		} `json:"by_rank"`
	} `json:"fsmap"`
	ProgressEvents map[string]progressEvents `json:"progress_events,omitempty"`
}

type progressEvents struct {
	Message  string  `json:"message,omitempty"`
	Progress float64 `json:"progress,omitempty"`
}

type mgrDump struct {
	Available  bool         `json:"available"`
	ActiveName string       `json:"active_name"`
	Standbys   []mgrStandby `json:"standbys"`
}

type mgrStandby struct {
	Name string `json:"name"`
}
