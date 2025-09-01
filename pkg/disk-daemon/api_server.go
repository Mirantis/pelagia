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
	"encoding/json"
	"fmt"
	"net/http"
)

func (d *diskDaemon) getFullReport(w http.ResponseWriter, _ *http.Request) {
	d.data.report.mu.RLock()
	defer d.data.report.mu.RUnlock()

	_ = json.NewEncoder(w).Encode(d.data.report.node)
}

func (d *diskDaemon) getOsdReport(w http.ResponseWriter, _ *http.Request) {
	d.data.report.mu.RLock()
	defer d.data.report.mu.RUnlock()

	osdReport := d.data.report.node
	osdReport.DisksReport = nil
	_ = json.NewEncoder(w).Encode(osdReport)
}

func (d *diskDaemon) checkAPI(w http.ResponseWriter, _ *http.Request) {
	_ = json.NewEncoder(w).Encode("ok")
}

func (d *diskDaemon) serveAPIServer(port int32) {
	http.HandleFunc("/apiCheck", d.checkAPI)
	http.HandleFunc("/fullReport", d.getFullReport)
	http.HandleFunc("/osdReport", d.getOsdReport)
	d.criticalError = http.ListenAndServe(fmt.Sprintf("127.0.0.1:%d", port), nil)
	close(d.quit)
}
