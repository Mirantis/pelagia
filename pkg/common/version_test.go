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

func TestGetCephVersionByReleaseName(t *testing.T) {
	tests := []struct {
		name            string
		releaseName     string
		expectedVersion *CephVersion
		expectedError   string
	}{
		{
			name: "no release, used latest",
			expectedVersion: &CephVersion{
				Name:            "Tentacle",
				MajorVersion:    "v20.2",
				Order:           20,
				SupportedMinors: []string{"0", "1"},
			},
		},
		{
			name:        "tentacle release",
			releaseName: "Tentacle",
			expectedVersion: &CephVersion{
				Name:            "Tentacle",
				MajorVersion:    "v20.2",
				Order:           20,
				SupportedMinors: []string{"0", "1"},
			},
		},
		{
			name:        "squid release",
			releaseName: "squid",
			expectedVersion: &CephVersion{
				Name:            "Squid",
				MajorVersion:    "v19.2",
				Order:           19,
				SupportedMinors: []string{"3", "4"},
			},
		},
		{
			name:          "unknown release",
			releaseName:   "octopus",
			expectedError: "specified not supported Ceph release 'octopus'. Is version correct?",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			expectedVersion, err := GetCephVersionByReleaseName(test.releaseName)
			if test.expectedError != "" {
				assert.NotNil(t, err)
				assert.Equal(t, test.expectedError, err.Error())
			} else {
				assert.Nil(t, err)
				assert.Equal(t, test.expectedVersion, expectedVersion)
			}
		})
	}
}

func TestParseCephVersion(t *testing.T) {
	tests := []struct {
		name            string
		cephVersion     string
		expectedVersion *CephVersion
		expectedError   string
	}{
		{
			name:          "check ceph version - no current version provided",
			cephVersion:   "",
			expectedError: "failed to parse version '', expected format 'ceph version x.x.x'",
		},
		{
			name:          "check ceph version - incorrect version provided",
			cephVersion:   "some wrong version passed",
			expectedError: "failed to parse version 'some wrong version passed', expected format 'ceph version x.x.x'",
		},
		{
			name:          "check ceph version - invalid ceph version",
			cephVersion:   "ceph version 18.32.5 (safmsdgldfhglkfdhdlstet) custom",
			expectedError: "unsupported Ceph major version 'v18.32' provided. Supported are: [Tentacle (v20.2) Squid (v19.2)]",
		},
		{
			name:          "check ceph version - image is not in list supported minors",
			cephVersion:   "ceph version 19.2.20 (safmsdgldfhglkfdhdlstet) custom",
			expectedError: "specified Ceph version 'v19.2.20' is not supported. Please use one of: [v19.2.3 v19.2.4]",
		},
		{
			name:        "check ceph version - tentacle image version passed",
			cephVersion: "ceph version 20.2.1 (6a49aff47758778a5f5951e731d437c317f72fb2) tentacle",
			expectedVersion: &CephVersion{
				Name:         "Tentacle",
				MajorVersion: "v20.2",
				MinorVersion: "1",
				Order:        20,
			},
		},
		{
			name:        "check ceph version - squid image version passed",
			cephVersion: "ceph version 19.2.3 (6a49aff47758778a5f5951e731d437c317f72fb2) squid",
			expectedVersion: &CephVersion{
				Name:         "Squid",
				MajorVersion: "v19.2",
				MinorVersion: "3",
				Order:        19,
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actualVersion, err := ParseCephVersion(test.cephVersion)
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
