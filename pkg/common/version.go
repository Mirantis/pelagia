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

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/pkg/errors"
)

type CephVersion struct {
	// ceph release name
	Name string
	// ceph major version
	MajorVersion string
	// ceph minor version
	MinorVersion string
	// major version for simple compare with other versions
	Order int
	// minor versions supported and available to use
	SupportedMinors []string
}

var AvailableCephVersions = []*CephVersion{Tentacle, Squid}

var (
	Tentacle = &CephVersion{
		Name:            "Tentacle",
		MajorVersion:    "v20.2",
		Order:           20,
		SupportedMinors: []string{"0", "1", "2"},
	}
	Squid = &CephVersion{
		Name:            "Squid",
		MajorVersion:    "v19.2",
		Order:           19,
		SupportedMinors: []string{"3", "4"},
	}
	LatestRelease = Tentacle
)

func GetCephVersionByReleaseName(releaseName string) (*CephVersion, error) {
	if releaseName == "" {
		return LatestRelease, nil
	}
	for _, version := range AvailableCephVersions {
		if strings.EqualFold(version.Name, releaseName) {
			return version, nil
		}
	}
	return nil, errors.Errorf("specified not supported Ceph release '%s'. Is version correct?", releaseName)
}

func ParseCephVersion(cephVersion string) (*CephVersion, error) {
	versionPattern := regexp.MustCompile(`ceph version (\d+)\.(\d+)\.(\d+)`)
	versionMatch := versionPattern.FindStringSubmatch(cephVersion)
	if len(versionMatch) < 4 {
		return nil, errors.Errorf("failed to parse version '%s', expected format 'ceph version x.x.x'", cephVersion)
	}
	cephVersionCLI := &CephVersion{
		MajorVersion: fmt.Sprintf("v%s.%s", versionMatch[1], versionMatch[2]),
		MinorVersion: versionMatch[3],
	}
	supportedMajors := []string{}
	for _, supported := range AvailableCephVersions {
		supportedMajors = append(supportedMajors, fmt.Sprintf("%s (%s)", supported.Name, supported.MajorVersion))
		if supported.MajorVersion == cephVersionCLI.MajorVersion {
			if Contains(supported.SupportedMinors, cephVersionCLI.MinorVersion) {
				cephVersionCLI.Name = supported.Name
				cephVersionCLI.Order = supported.Order
				return cephVersionCLI, nil
			}
			// TODO: get rid of supported minors and keep list supported for major only?
			supportedMinors := []string{}
			for _, minor := range supported.SupportedMinors {
				supportedMinors = append(supportedMinors, fmt.Sprintf("%s.%s", supported.MajorVersion, minor))
			}
			return nil, errors.Errorf("specified Ceph version '%s.%s' is not supported. Please use one of: %v", cephVersionCLI.MajorVersion, cephVersionCLI.MinorVersion, supportedMinors)
		}
	}
	return nil, errors.Errorf("unsupported Ceph major version '%s' provided. Supported are: %v", cephVersionCLI.MajorVersion, supportedMajors)
}

func CephVersionGreaterOrEqual(currentVersion, requiredVersion *CephVersion) bool {
	if currentVersion.Name == "" {
		return false
	}
	return currentVersion.Order >= requiredVersion.Order
}
