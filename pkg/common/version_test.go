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
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCephVersionGreaterOrEqual(t *testing.T) {
	tests := []struct {
		name            string
		cephImage       *CephVersion
		requiredVersion *CephVersion
		expected        bool
	}{
		{
			name:            "ceph version comparison - current Squid greater than required Reef",
			cephImage:       Squid,
			requiredVersion: Reef,
			expected:        true,
		},
		{
			name:            "ceph version comparison - current equal to required, Reef",
			cephImage:       Reef,
			requiredVersion: Reef,
			expected:        true,
		},
		{
			name:            "ceph version comparison - current Reef less than required Squid",
			cephImage:       Reef,
			requiredVersion: Squid,
			expected:        false,
		},
		{
			name:      "ceph version comparison - current version is empty",
			cephImage: &CephVersion{},
			expected:  false,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual := CephVersionGreaterOrEqual(test.cephImage, test.requiredVersion)
			assert.Equal(t, test.expected, actual)
		})
	}
}

func TestCheckExpectedCephVersion(t *testing.T) {
	tests := []struct {
		name            string
		cephImage       string
		cephRelease     string
		expectedVersion *CephVersion
		expectedError   string
	}{
		{
			name:          "check ceph version - no current version provided",
			expectedError: "expected ceph image is not specified",
		},
		{
			name:          "check ceph version - current image version different from expected release",
			cephImage:     "ceph/ceph:v18.2.7",
			expectedError: "expected Ceph release Squid 'v19.2' version, but specified Reef 'v18.2' version (image: ceph/ceph:v18.2.7)",
		},
		{
			name:          "check ceph version - invalid version release version is set",
			cephImage:     "ceph/ceph:v18.2.7",
			cephRelease:   "rif",
			expectedError: "failed to find appropriate Ceph version of 'rif' release. Is release name correct?",
		},
		{
			name:          "check ceph version - invalid image version is set",
			cephImage:     "ceph/ceph:v20.32.5",
			expectedError: "failed to identify Ceph version for image 'ceph/ceph:v20.32.5': failed to find supported Ceph version for specified 'v20.32.5' version. Is version correct?",
		},
		{
			name:          "check ceph version - image is not in list supported minors",
			cephImage:     "ceph/ceph:v18.2.10",
			expectedError: "failed to identify Ceph version for image 'ceph/ceph:v18.2.10': specified Ceph version 'v18.2.10' is not supported. Please use one of: [v18.2.3 v18.2.4 v18.2.7]",
		},
		{
			name:        "check ceph version - reef image version passed and reef release",
			cephImage:   "ceph/ceph:v18.2.3",
			cephRelease: "reef",
			expectedVersion: &CephVersion{
				Name:            "Reef",
				MajorVersion:    "v18.2",
				MinorVersion:    "3",
				Order:           18,
				SupportedMinors: []string{"3", "4", "7"},
			},
		},
		{
			name:        "check ceph version - squid image version passed and squid release",
			cephImage:   "ceph/ceph:v19.2.3",
			cephRelease: "squid",
			expectedVersion: &CephVersion{
				Name:            "Squid",
				MajorVersion:    "v19.2",
				MinorVersion:    "3",
				Order:           19,
				SupportedMinors: []string{"3"},
			},
		},
		{
			name:      "check ceph version - latest image version passed and no release",
			cephImage: "ceph/ceph:v19.2.3",
			expectedVersion: &CephVersion{
				Name:            "Squid",
				MajorVersion:    "v19.2",
				MinorVersion:    "3",
				Order:           19,
				SupportedMinors: []string{"3"},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actualVersion, err := CheckExpectedCephVersion(test.cephImage, test.cephRelease)
			if test.expectedError != "" {
				assert.NotNil(t, err)
				assert.Equal(t, test.expectedError, err.Error())
			} else {
				assert.Nil(t, err)
				assert.Equal(t, test.expectedVersion, actualVersion)
			}
		})
	}
}
