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

import (
	"os"
	"testing"

	"github.com/pkg/errors"
	cephv1 "github.com/rook/rook/pkg/apis/ceph.rook.io/v1"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/runtime"

	cephlcmv1alpha1 "github.com/Mirantis/pelagia/v3/pkg/apis/ceph.pelagia.lcm/v1alpha1"
	faketestclients "github.com/Mirantis/pelagia/v3/test/unit/clients"
	unitinputs "github.com/Mirantis/pelagia/v3/test/unit/inputs"
)

func TestCephUpgradeAllowed(t *testing.T) {
	tests := []struct {
		name           string
		osdplst        runtime.Object
		varError       bool
		upgradeAllowed bool
		expectedError  string
	}{
		{
			name:          "get envvar failed - fail",
			varError:      true,
			expectedError: "required env variable 'CEPH_CONTROLLER_CLUSTER_RELEASE' is not set",
		},
		{
			name:          "failed to get osdpl list",
			osdplst:       unitinputs.GetOpenstackDeploymentStatusList("", "", false),
			expectedError: "failed to get openstackdeploymentstatus state and release: OpenstackDeploymentStatus required values in status.osdpl not found",
		},
		{
			name:           "no osdpl present - allow upgrade",
			upgradeAllowed: true,
		},
		{
			name:    "osdpl has different release - disallow upgrade",
			osdplst: unitinputs.GetOpenstackDeploymentStatusList("new", "APPLIED", true),
		},
		{
			name:    "osdpl has current release, but not ready - disallow upgrade",
			osdplst: unitinputs.GetOpenstackDeploymentStatusList("new", "ERROR", true),
		},
		{
			name:           "osdpl has current release, ready - allow upgrade",
			osdplst:        unitinputs.GetOpenstackDeploymentStatusList("cur", "APPLIED", true),
			upgradeAllowed: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := fakeDeploymentConfig(nil, nil)
			if test.osdplst != nil {
				c.api.Client = faketestclients.GetClient(faketestclients.GetClientBuilder().WithRuntimeObjects(test.osdplst))
			} else {
				c.api.Client = faketestclients.GetClient(nil)
			}

			if test.varError {
				os.Unsetenv("CEPH_CONTROLLER_CLUSTER_RELEASE")
			} else {
				t.Setenv("CEPH_CONTROLLER_CLUSTER_RELEASE", "cur")
			}

			allowed, err := c.cephUpgradeAllowed()
			if test.expectedError != "" {
				assert.NotNil(t, err)
				assert.Equal(t, test.expectedError, err.Error())
			} else {
				assert.Nil(t, err)
			}
			assert.Equal(t, test.upgradeAllowed, allowed)
		})
	}
}

func TestAlignSpecForAES256k(t *testing.T) {
	tests := []struct {
		name            string
		cephDpl         *cephlcmv1alpha1.CephDeployment
		cephxConfig     *cephv1.ClusterCephxConfig
		expectedCephDpl *cephlcmv1alpha1.CephDeployment
		apiError        bool
		expectedError   string
	}{
		{
			name: "incorrect cephdpl without cluster spec as raw",
			cephDpl: &cephlcmv1alpha1.CephDeployment{
				Spec: cephlcmv1alpha1.CephDeploymentSpec{
					Cluster: &cephlcmv1alpha1.CephCluster{RawExtension: runtime.RawExtension{Raw: nil}},
				},
			},
			expectedError: "spec.cluster does not contain raw CephCluster spec",
		},
		{
			name:          "failed to update spec",
			cephDpl:       unitinputs.BaseCephDeployment.DeepCopy(),
			apiError:      true,
			expectedError: "failed to update CephDeployment spec: failed to update",
		},
		{
			name: "add new cephx config and health check to spec",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				cdpl := unitinputs.BaseCephDeployment.DeepCopy()
				cdpl.Spec.Cluster.Raw = []byte(`{"network":{"addressRanges":{"cluster":["127.0.0.0/16"],"public":["192.168.0.0/16"]}}}`)
				return cdpl
			}(),
			cephxConfig: &cephv1.ClusterCephxConfig{
				AllowedCiphers: []cephv1.CephxKeyType{"aes", "aes256k"},
				Daemon: cephv1.CephxConfig{
					KeyGeneration: 2,
					KeyType:       cephv1.CephxKeyTypeAes256k,
				},
			},
			expectedCephDpl: func() *cephlcmv1alpha1.CephDeployment {
				cdpl := unitinputs.BaseCephDeployment.DeepCopy()
				cdpl.Labels = map[string]string{"cephdeployment.lcm.mirantis.com/aes256kApplied": "true"}
				cdpl.Spec.Cluster.Raw = []byte(`{"healthCheck":{"muteHealthWarning":{"AUTH_EMERGENCY_CIPHERS_SET":{"policy":"mute"},"AUTH_INSECURE_CLIENT_KEY_TYPE":{"policy":"mute"},"AUTH_INSECURE_KEYS_ALLOWED":{"policy":"mute"},"AUTH_INSECURE_KEYS_CREATABLE":{"policy":"mute"},"AUTH_INSECURE_ROTATING_SERVICE_KEY_TYPE":{"policy":"mute"}}},"network":{"addressRanges":{"cluster":["127.0.0.0/16"],"public":["192.168.0.0/16"]}},"security":{"cephx":{"allowedCiphers":["aes","aes256k"],"daemon":{"keyGeneration":2,"keyType":"aes256k"},"rbdMirrorPeer":{},"csi":{}}}}`)
				return cdpl
			}(),
		},
		{
			name: "update security with cephx config and health checks",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				cdpl := unitinputs.BaseCephDeployment.DeepCopy()
				cdpl.Spec.Cluster.Raw = []byte(`{"healthCheck":{"muteHealthWarning":{"CUSTOM_WARNING":{"policy":"mute"}}},"network":{"addressRanges":{"cluster":["127.0.0.0/16"],"public":["192.168.0.0/16"]}},"security":{"keyRotation":{"enabled":false},"cephx":{"allowedCiphers":["aes","aes256k"],"daemon":{"keyGeneration":2,"keyType":"aes256k"},"rbdMirrorPeer":{},"csi":{}}}}`)
				return cdpl
			}(),
			cephxConfig: &cephv1.ClusterCephxConfig{
				AllowedCiphers: []cephv1.CephxKeyType{"aes", "aes256k"},
				Daemon: cephv1.CephxConfig{
					KeyGeneration: 2,
					KeyType:       cephv1.CephxKeyTypeAes256k,
				},
			},
			expectedCephDpl: func() *cephlcmv1alpha1.CephDeployment {
				cdpl := unitinputs.BaseCephDeployment.DeepCopy()
				cdpl.Labels = map[string]string{"cephdeployment.lcm.mirantis.com/aes256kApplied": "true"}
				cdpl.Spec.Cluster.Raw = []byte(`{"healthCheck":{"muteHealthWarning":{"AUTH_EMERGENCY_CIPHERS_SET":{"policy":"mute"},"AUTH_INSECURE_CLIENT_KEY_TYPE":{"policy":"mute"},"AUTH_INSECURE_KEYS_ALLOWED":{"policy":"mute"},"AUTH_INSECURE_KEYS_CREATABLE":{"policy":"mute"},"AUTH_INSECURE_ROTATING_SERVICE_KEY_TYPE":{"policy":"mute"},"CUSTOM_WARNING":{"policy":"mute"}}},"network":{"addressRanges":{"cluster":["127.0.0.0/16"],"public":["192.168.0.0/16"]}},"security":{"cephx":{"allowedCiphers":["aes","aes256k"],"daemon":{"keyGeneration":2,"keyType":"aes256k"},"rbdMirrorPeer":{},"csi":{}},"keyRotation":{"enabled":false}}}`)
				return cdpl
			}(),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := fakeDeploymentConfig(&deployConfig{cephDpl: test.cephDpl}, nil)
			c.cdConfig.cephxConfigWithAes256k = test.cephxConfig
			inputResources := map[string]runtime.Object{
				"cephdeployments": &cephlcmv1alpha1.CephDeploymentList{Items: []cephlcmv1alpha1.CephDeployment{*test.cephDpl}},
			}
			apiErrors := map[string]error{}
			if test.apiError {
				apiErrors["update-cephdeployments"] = errors.New("failed to update")
			}
			faketestclients.FakeReaction(c.api.CephLcmclientset, "get", []string{"cephdeployments"}, inputResources, apiErrors)
			faketestclients.FakeReaction(c.api.CephLcmclientset, "update", []string{"cephdeployments"}, inputResources, apiErrors)

			err := c.alignSpecForAES256k()
			if test.expectedError != "" {
				assert.NotNil(t, err)
				assert.Equal(t, test.expectedError, err.Error())
			} else {
				assert.Nil(t, err)
				cephDpl := inputResources["cephdeployments"].(*cephlcmv1alpha1.CephDeploymentList).Items[0]
				assert.Equal(t, *test.expectedCephDpl, cephDpl)
			}
		})
	}
}
