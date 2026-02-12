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

package main

import (
	"flag"
	"fmt"
	"os"

	_ "k8s.io/client-go/plugin/pkg/client/auth"

	lcmversion "github.com/Mirantis/pelagia/codeversion"
	diskdaemon "github.com/Mirantis/pelagia/pkg/disk-daemon"
)

// dummy mutually exclusive flags checker
func mutuallyExclusivePassed(flags ...bool) bool {
	alreadySet := false
	for _, flag := range flags {
		if flag {
			if alreadySet {
				return true
			}
			alreadySet = true
		}
	}
	return false
}

func main() {
	var daemonMode, apiCheckMode, fullReportMode, osdReportMode, version bool
	var diskDaemonPort int
	flag.BoolVar(&daemonMode, "daemon", false, "daemon mode for collecting hardware disk/partitions/volumes info")
	flag.BoolVar(&apiCheckMode, "api-check", false, "check disk daemon api")
	flag.BoolVar(&fullReportMode, "full-report", false, "get full report from daemon (extended with disks info)")
	flag.BoolVar(&osdReportMode, "osd-report", false, "get osd report from daemon (osd inforation for lcm)")
	flag.BoolVar(&version, "version", false, "get version of binary")
	flag.IntVar(&diskDaemonPort, "port", 9999, "disk daemon API port, usually not required to be changed")
	flag.Parse()

	if mutuallyExclusivePassed(daemonMode, apiCheckMode, fullReportMode, osdReportMode, version) {
		panic("unknown mode: flags --daemon, --api-check, --full-report, --osd-report and --version are mutually exclusive")
	}

	if version {
		fmt.Println(lcmversion.GetCodeVersion("Disk daemon"))
		fmt.Println(lcmversion.GetGoRuntimeVersion())
		os.Exit(0)
	}

	if daemonMode {
		err := diskdaemon.Daemon(diskDaemonPort)
		if err != nil {
			panic(err.Error())
		}
		os.Exit(0)
	}

	if apiCheckMode || fullReportMode || osdReportMode {
		mode := "api-check"
		if fullReportMode {
			mode = "full-report"
		} else if osdReportMode {
			mode = "osd-report"
		}
		err := diskdaemon.GetDaemonReport(diskDaemonPort, mode)
		if err != nil {
			panic(err.Error())
		}
		os.Exit(0)
	}

	panic("unknown mode: use --daemon or --api-check or --full-report or --osd-report or --version or --help for details")
}
