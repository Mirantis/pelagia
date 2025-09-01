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
	"sync"
	"time"

	lcmcommon "github.com/Mirantis/pelagia/pkg/common"
)

var (
	log           = lcmcommon.InitLogger(false)
	checkInterval = 30 * time.Second
)

type diskDaemon struct {
	// critical error lead to panic error
	criticalError error
	// main data used by daemon
	data daemonData
	// chan holder for quit
	quit chan struct{}
}

type daemonData struct {
	// current runtime data
	runtime runtimeData
	// api reports data with mutex
	report reportData
}

type runtimeData struct {
	// in-memory last prepared disk report
	disksReport *lcmcommon.DiskDaemonDisksReport
	// in-memory last prepared volumes report
	volumesReport map[string][]OsdVolumeInfo
	// in-memory last prepared osds report
	osdsReport *lcmcommon.DiskDaemonOsdsReport
	// in-memory known lvm partitions
	knownLvms map[string][]string
}

type reportData struct {
	mu   sync.RWMutex
	node lcmcommon.DiskDaemonReport
}

func initDaemonData() daemonData {
	return daemonData{
		runtime: runtimeData{
			volumesReport: map[string][]OsdVolumeInfo{},
			knownLvms:     map[string][]string{},
		},
		report: reportData{
			node: lcmcommon.DiskDaemonReport{
				State: lcmcommon.DiskDaemonStateInProgress,
			},
		},
	}
}

func Daemon(daemonPort int) error {
	log.Info().Msg("initializing disk-daemon")
	diskDaemon := &diskDaemon{
		data: initDaemonData(),
		quit: make(chan struct{}),
	}
	log.Info().Msg("initializing local API server")
	// run very simple api server to handle requests only inside container itself
	// to always get reports from in-memory
	go diskDaemon.serveAPIServer(int32(daemonPort))
	log.Info().Msg("running disk-daemon")
	ticker := time.NewTicker(checkInterval)
	for {
		select {
		case <-ticker.C:
			diskDaemon.prepareReport()
		case <-diskDaemon.quit:
			ticker.Stop()
			return diskDaemon.criticalError
		}
	}
}

func (d *diskDaemon) prepareReport() {
	d.updateNodeReportState(false, nil, nil, nil)
	stateChanged, err := d.checkDisks()
	if err != nil {
		// reset runtime vars
		d.updateNodeReportState(false, nil, nil, []string{err.Error()})
		return
	}
	var osdIssues []string
	if stateChanged {
		osdIssues = d.checkOsds()
	}
	d.updateNodeReportState(true, d.data.runtime.disksReport, d.data.runtime.osdsReport, osdIssues)
}

func (d *diskDaemon) updateNodeReportState(ready bool, disksReport *lcmcommon.DiskDaemonDisksReport, osdsReport *lcmcommon.DiskDaemonOsdsReport, issues []string) {
	d.data.report.mu.Lock()
	defer d.data.report.mu.Unlock()

	d.data.report.node.DisksReport = disksReport
	d.data.report.node.OsdsReport = osdsReport
	if len(issues) > 0 {
		d.data.report.node.Issues = issues
		d.data.report.node.State = lcmcommon.DiskDaemonStateFailed
	} else {
		d.data.report.node.Issues = nil
		if ready {
			d.data.report.node.State = lcmcommon.DiskDaemonStateOk
		} else {
			d.data.report.node.State = lcmcommon.DiskDaemonStateInProgress
		}
	}
}
