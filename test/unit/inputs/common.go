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

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	lcmcommon "github.com/Mirantis/pelagia/pkg/common"
)

var LatestCephVersionImage = fmt.Sprintf("%s.%s", lcmcommon.LatestRelease.MajorVersion, lcmcommon.LatestRelease.SupportedMinors[len(lcmcommon.LatestRelease.SupportedMinors)-1])
var PreviousCephVersionImage = fmt.Sprintf("%s.%s", previousRelease.MajorVersion, previousRelease.SupportedMinors[len(previousRelease.SupportedMinors)-1])
var LatestCephVersion = strings.ToLower(lcmcommon.LatestRelease.Name)
var PreviousCephVersion = strings.ToLower(previousRelease.Name)

var previousRelease = func() *lcmcommon.CephVersion {
	if len(lcmcommon.LatestRelease.SupportedMinors) > 1 {
		return lcmcommon.LatestRelease
	}
	releaseIdx := 0
	for idx, release := range lcmcommon.AvailableCephVersions {
		if release.Order+1 == lcmcommon.LatestRelease.Order {
			releaseIdx = idx
			break
		}
	}
	previousCephRelease := lcmcommon.AvailableCephVersions[releaseIdx]
	return previousCephRelease
}()

var LcmObjectMeta = metav1.ObjectMeta{
	Name:      "cephcluster",
	Namespace: "lcm-namespace",
}

var RookNamespace = "rook-ceph"

var ResourceListLimitsDefault = corev1.ResourceList{
	corev1.ResourceCPU:    resource.MustParse("200m"),
	corev1.ResourceMemory: resource.MustParse("256Mi"),
}

var ResourceListRequestsDefault = corev1.ResourceList{
	corev1.ResourceMemory: resource.MustParse("128Mi"),
	corev1.ResourceCPU:    resource.MustParse("100m"),
}
