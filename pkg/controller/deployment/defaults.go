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

package deployment

// Module contains functions which defines default values for different
// resources

func poolsDefaultTargetSizeRatioByRole(role string) float64 {
	switch role {
	case "images", "backup", "rgw data":
		return 0.1
	case "volumes":
		return 0.4
	case "vms":
		return 0.2
	}
	return 0
}

type updateTimestamps struct {
	cephConfigMap    map[string]string
	rgwSSLCert       string
	rgwRuntimeParams string
	osdRuntimeParams string
}

var resourceUpdateTimestamps = updateTimestamps{
	cephConfigMap: map[string]string{},
}

func unsetTimestampsVar() {
	resourceUpdateTimestamps = updateTimestamps{cephConfigMap: map[string]string{}}
}
