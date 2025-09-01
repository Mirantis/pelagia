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

import (
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
)

func IsNodeWithDiskDaemon(node corev1.Node, lcmDiskDaemonLabel string) bool {
	if len(node.Labels) > 0 {
		// no need to check an error, since it is checked in config controller
		selector, _ := labels.Parse(lcmDiskDaemonLabel)
		return selector.Matches(labels.Set(node.Labels))
	}
	return false
}

func FindDiskName(name string, diskReport *DiskDaemonDisksReport) (string, error) {
	// case for just device name like 'sda'
	name = PathDevPrepended(name)
	if curBlockDev, present := diskReport.Aliases[name]; present {
		if diskReport.BlockInfo[curBlockDev].Type == "disk" {
			return curBlockDev, nil
		}
		parents := diskReport.BlockInfo[curBlockDev].Parent
		// we are not supporting raids/lvm spreaded accross multiple disks
		if len(parents) == 1 {
			return FindDiskName(parents[0], diskReport)
		} else if len(parents) > 1 {
			return "", fmt.Errorf("detected multidisk setup, which is not supported: %s", strings.Join(parents, ","))
		}
	}
	return "", fmt.Errorf("device '%s' is not found on a node", name)
}

func PathDevPrepended(name string) string {
	if !strings.HasPrefix(name, "/dev/") {
		name = fmt.Sprintf("/dev/%s", name)
	}
	return name
}
