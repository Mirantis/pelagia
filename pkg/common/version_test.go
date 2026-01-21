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
			name:            "ceph version comparison - current greater than required",
			cephImage:       Tentacle,
			requiredVersion: Squid,
			expected:        true,
		},
		{
			name:            "ceph version comparison - current equal to required",
			cephImage:       Squid,
			requiredVersion: Squid,
			expected:        true,
		},
		{
			name:            "ceph version comparison - current less than required",
			cephImage:       Squid,
			requiredVersion: Tentacle,
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
			cephImage:     "ceph/ceph:v19.2.3",
			expectedError: "expected Ceph release Tentacle 'v20.2' version, but specified Squid 'v19.2' version (image: ceph/ceph:v19.2.3)",
		},
		{
			name:          "check ceph version - invalid version release version is set",
			cephImage:     "ceph/ceph:v19.2.3",
			cephRelease:   "sqid",
			expectedError: "failed to find appropriate Ceph version of 'sqid' release. Is release name correct?",
		},
		{
			name:          "check ceph version - invalid image version is set",
			cephImage:     "ceph/ceph:v20.32.5",
			expectedError: "failed to identify Ceph version for image 'ceph/ceph:v20.32.5': failed to find supported Ceph version for specified 'v20.32.5' version. Is version correct?",
		},
		{
			name:          "check ceph version - image is not in list supported minors",
			cephImage:     "ceph/ceph:v19.2.20",
			expectedError: "failed to identify Ceph version for image 'ceph/ceph:v19.2.20': specified Ceph version 'v19.2.20' is not supported. Please use one of: [v19.2.3]",
		},
		{
			name:        "check ceph version - tentacle image version passed and tentacle release",
			cephImage:   "ceph/ceph:v20.2.0",
			cephRelease: "tentacle",
			expectedVersion: &CephVersion{
				Name:            "Tentacle",
				MajorVersion:    "v20.2",
				MinorVersion:    "0",
				Order:           20,
				SupportedMinors: []string{"0"},
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
			cephImage: "ceph/ceph:v20.2.0",
			expectedVersion: &CephVersion{
				Name:            "Tentacle",
				MajorVersion:    "v20.2",
				MinorVersion:    "0",
				Order:           20,
				SupportedMinors: []string{"0"},
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
