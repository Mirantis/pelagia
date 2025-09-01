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
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/pkg/errors"
)

func GetDaemonReport(daemonPort int, reportObj string) error {
	report := ""
	switch reportObj {
	case "full-report":
		report = "fullReport"
	case "osd-report":
		report = "osdReport"
	case "api-check":
		report = "apiCheck"
	default:
		// just stub in case if someone will forgot to update URLs handler
		return errors.Errorf("unknown report type requested: '%s'", reportObj)
	}
	// very simple client with hardcodes, but since we are using it only
	// inside container - does not matter
	requestURL := fmt.Sprintf("http://127.0.0.1:%d/%s", daemonPort, report)
	resp, err := http.Get(requestURL)
	if err != nil {
		return errors.Wrapf(err, "failed to get %s report info", report)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return errors.Wrap(err, "could not read response body")
	}
	responseStr := strings.Trim(string(body), "\n")
	if responseStr == "null" {
		fmt.Print("{}")
	} else {
		fmt.Print(responseStr)
	}
	return nil
}
