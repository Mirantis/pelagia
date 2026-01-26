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
		SupportedMinors: []string{"0"},
	}
	Squid = &CephVersion{
		Name:            "Squid",
		MajorVersion:    "v19.2",
		Order:           19,
		SupportedMinors: []string{"3"},
	}
	LatestRelease = Tentacle
)

func GetCephVersionFromImage(cephImage string) string {
	reg := regexp.MustCompile(`:v.*\..*\.*`)
	ver := reg.FindString(cephImage)
	ver = strings.TrimPrefix(ver, ":")
	// drop cve/release suffix and return pure version in format v1.1.1
	return strings.Split(ver, "-")[0]
}

func ParseCephVersion(version string) (*CephVersion, error) {
	var cephVersion *CephVersion
	for _, supported := range AvailableCephVersions {
		if strings.HasPrefix(version, supported.MajorVersion) {
			minorVersion := strings.TrimPrefix(version, supported.MajorVersion+".")
			if Contains(supported.SupportedMinors, minorVersion) {
				cephVersion = &CephVersion{
					Name:            supported.Name,
					MajorVersion:    supported.MajorVersion,
					MinorVersion:    minorVersion,
					Order:           supported.Order,
					SupportedMinors: supported.SupportedMinors,
				}
				break
			}
			supportedMinors := []string{}
			for _, minor := range supported.SupportedMinors {
				supportedMinors = append(supportedMinors, fmt.Sprintf("%s.%s", supported.MajorVersion, minor))
			}
			return nil, errors.Errorf("specified Ceph version '%s' is not supported. Please use one of: %v", version, supportedMinors)
		}
	}
	if cephVersion == nil {
		return nil, errors.Errorf("failed to find supported Ceph version for specified '%s' version. Is version correct?", version)
	}
	return cephVersion, nil
}

func CheckExpectedCephVersion(expectedCephImage, expectedCephRelease string) (*CephVersion, error) {
	if expectedCephImage == "" {
		return nil, errors.New("expected ceph image is not specified")
	}
	var expectedCephVersion *CephVersion
	if expectedCephRelease == "" {
		expectedCephVersion = LatestRelease
	} else {
		for _, version := range AvailableCephVersions {
			if strings.EqualFold(expectedCephRelease, version.Name) {
				expectedCephVersion = version
				break
			}
		}
	}
	if expectedCephVersion == nil {
		return nil, errors.Errorf("failed to find appropriate Ceph version of '%s' release. Is release name correct?", expectedCephRelease)
	}
	cephVersion, err := ParseCephVersion(GetCephVersionFromImage(expectedCephImage))
	if err != nil {
		return nil, errors.Wrapf(err, "failed to identify Ceph version for image '%s'", expectedCephImage)
	}
	if cephVersion.MajorVersion != expectedCephVersion.MajorVersion {
		return nil, errors.Errorf("expected Ceph release %s '%s' version, but specified %s '%s' version (image: %s)",
			expectedCephVersion.Name, expectedCephVersion.MajorVersion, cephVersion.Name, cephVersion.MajorVersion, expectedCephImage)
	}
	return cephVersion, nil
}

func CephVersionGreaterOrEqual(currentVersion, requiredVersion *CephVersion) bool {
	if currentVersion.Name == "" {
		return false
	}
	return currentVersion.Order >= requiredVersion.Order
}
