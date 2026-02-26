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

package connector

import (
	"testing"

	"github.com/pkg/errors"
	cephv1 "github.com/rook/rook/pkg/apis/ceph.rook.io/v1"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"

	lcmcommon "github.com/Mirantis/pelagia/pkg/common"
	faketestclients "github.com/Mirantis/pelagia/test/unit/clients"
	unitinputs "github.com/Mirantis/pelagia/test/unit/inputs"
)

func TestGetClusterFSID(t *testing.T) {
	c := FakeConnector()
	tests := []struct {
		name           string
		inputResources map[string]runtime.Object
		fsid           string
		expectedError  string
	}{
		{
			name:           "failed to get ceph cluster",
			inputResources: map[string]runtime.Object{},
			expectedError:  "failed to check CephCluster object in namespace 'rook-ceph': failed to list cephclusters",
		},
		{
			name: "more than one cluster found",
			inputResources: map[string]runtime.Object{
				"cephclusters": &cephv1.CephClusterList{Items: []cephv1.CephCluster{unitinputs.ReefCephClusterReady, unitinputs.ReefCephClusterReady}},
			},
			expectedError: "multiple CephCluster objects found in namespace 'rook-ceph'",
		},
		{
			name: "no cluster found",
			inputResources: map[string]runtime.Object{
				"cephclusters": &unitinputs.CephClusterListEmpty,
			},
			expectedError: "no CephCluster objects found in namespace 'rook-ceph'",
		},
		{
			name: "cluster has no ceph status",
			inputResources: map[string]runtime.Object{
				"cephclusters": &cephv1.CephClusterList{Items: []cephv1.CephCluster{unitinputs.BuildBaseCephCluster(unitinputs.ReefCephClusterReady.Name, unitinputs.ReefCephClusterReady.Namespace)}},
			},
			expectedError: "status is not present for CephCluster 'rook-ceph/cephcluster'",
		},
		{
			name: "cluster has no fsid info in status",
			inputResources: map[string]runtime.Object{
				"cephclusters": &cephv1.CephClusterList{
					Items: []cephv1.CephCluster{
						func() cephv1.CephCluster {
							cephCluster := unitinputs.ReefCephClusterHasHealthIssues.DeepCopy()
							cephCluster.Status.CephStatus.FSID = ""
							return *cephCluster
						}(),
					},
				},
			},
			expectedError: "cluster fsid is empty in CephCluster 'rook-ceph/cephcluster' status",
		},
		{
			name: "get fsid ok",
			inputResources: map[string]runtime.Object{
				"cephclusters": &unitinputs.CephClusterListReady,
			},
			fsid: "8668f062-3faa-358a-85f3-f80fe6c1e306",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			faketestclients.FakeReaction(c.Rookclientset, "list", []string{"cephclusters"}, test.inputResources, nil)

			fsid, err := c.getClusterFSID("rook-ceph")
			if test.expectedError != "" {
				assert.NotNil(t, err)
				assert.Equal(t, test.expectedError, err.Error())
			} else {
				assert.Nil(t, err)
				assert.Equal(t, test.fsid, fsid)
			}

			faketestclients.CleanupFakeClientReactions(c.Rookclientset)
		})
	}
}

func TestGetClusterMonEndpoints(t *testing.T) {
	c := FakeConnector()
	tests := []struct {
		name           string
		inputResources map[string]runtime.Object
		monEndpoints   string
		expectedError  string
	}{
		{
			name:           "no configmap found",
			inputResources: map[string]runtime.Object{"configmaps": unitinputs.ConfigMapListEmpty},
			expectedError:  "failed to get ConfigMap 'rook-ceph/rook-ceph-mon-endpoints': configmaps \"rook-ceph-mon-endpoints\" not found",
		},
		{
			name: "no monEndpoints in config map",
			inputResources: map[string]runtime.Object{
				"configmaps": &v1.ConfigMapList{Items: []v1.ConfigMap{*unitinputs.GetConfigMap("rook-ceph-mon-endpoints", "rook-ceph", nil)}},
			},
			expectedError: "mon endpoints are empty in ConfigMap 'rook-ceph/rook-ceph-mon-endpoints'",
		},
		{
			name: "get fsid ok",
			inputResources: map[string]runtime.Object{
				"configmaps": &v1.ConfigMapList{Items: []v1.ConfigMap{unitinputs.RookCephMonEndpoints}},
			},
			monEndpoints: "a=127.0.0.1,b=127.0.0.2,c=127.0.0.3",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			faketestclients.FakeReaction(c.Kubeclientset.CoreV1(), "get", []string{"configmaps"}, test.inputResources, nil)

			monEndpoints, err := c.getClusterMonEndpoints("rook-ceph")
			if test.expectedError != "" {
				assert.NotNil(t, err)
				assert.Equal(t, test.expectedError, err.Error())
			} else {
				assert.Nil(t, err)
				assert.Equal(t, test.monEndpoints, monEndpoints)
			}
			faketestclients.CleanupFakeClientReactions(c.Kubeclientset.CoreV1())
		})
	}
}

func TestGetCephKeyringFromSecret(t *testing.T) {
	c := FakeConnector()
	tests := []struct {
		name           string
		inputResources map[string]runtime.Object
		userID         string
		keyring        string
		expectedError  string
	}{
		{
			name:           "no secret found",
			inputResources: map[string]runtime.Object{"secrets": &unitinputs.SecretsListEmpty},
			expectedError:  "failed to get Secret 'rook-ceph/rook-csi-rbd-node': secrets \"rook-csi-rbd-node\" not found",
		},
		{
			name: "no keyring specified in secret",
			inputResources: map[string]runtime.Object{
				"secrets": &v1.SecretList{Items: []v1.Secret{{ObjectMeta: unitinputs.CSIRBDNodeSecret.ObjectMeta}}},
			},
			expectedError: "Secret 'rook-ceph/rook-csi-rbd-node' has empty userKey or userID",
		},
		{
			name: "no keyring specified in secret",
			inputResources: map[string]runtime.Object{
				"secrets": &v1.SecretList{Items: []v1.Secret{unitinputs.CSIRBDNodeSecret}},
			},
			userID:  "csi-rbd-node.1",
			keyring: "AQDd+HRjKiMBOhAATVfdzSNdlOAG3vaPSeTBzw==",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			faketestclients.FakeReaction(c.Kubeclientset.CoreV1(), "get", []string{"secrets"}, test.inputResources, nil)

			userID, keyring, err := c.getCephKeyringFromSecret("rook-ceph", "rook-csi-rbd-node")
			if test.expectedError != "" {
				assert.NotNil(t, err)
				assert.Equal(t, test.expectedError, err.Error())
			} else {
				assert.Nil(t, err)
			}
			assert.Equal(t, test.userID, userID)
			assert.Equal(t, test.keyring, keyring)
			faketestclients.CleanupFakeClientReactions(c.Kubeclientset.CoreV1())
		})
	}
}

func TestGetClusterClientKeyring(t *testing.T) {
	c := FakeConnector()
	tests := []struct {
		name          string
		cmdOut        string
		cmdFailed     bool
		clientKeyring string
		expectedError string
	}{
		{
			name:          "failed to execute cmd",
			cmdFailed:     true,
			expectedError: "failed to get keyring for client: failed to run command 'ceph auth get-key client.test': run failed",
		},
		{
			name:          "keyring is empty",
			expectedError: "client.test has empty keyring",
		},
		{
			name:          "get keyring ok",
			cmdOut:        "some-keyring",
			clientKeyring: "some-keyring",
		},
	}
	oldRunCmd := lcmcommon.RunPodCommand
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			faketestclients.FakeReaction(c.Kubeclientset.CoreV1(), "list", []string{"pods"}, map[string]runtime.Object{"pods": unitinputs.ToolBoxPodList}, nil)

			lcmcommon.RunPodCommand = func(_ lcmcommon.ExecConfig) (string, string, error) {
				if test.cmdFailed {
					return "", "", errors.New("run failed")
				}
				return test.cmdOut, "", nil
			}

			keyring, err := c.getClusterClientKeyring("rook-ceph", "test")
			if test.expectedError != "" {
				assert.NotNil(t, err)
				assert.Equal(t, test.expectedError, err.Error())
			} else {
				assert.Nil(t, err)
				assert.Equal(t, test.clientKeyring, keyring)
			}
			faketestclients.CleanupFakeClientReactions(c.Kubeclientset.CoreV1())
		})
	}
	lcmcommon.RunPodCommand = oldRunCmd
}

func TestGetRgwAdminOpsKeys(t *testing.T) {
	c := FakeConnector()
	tests := []struct {
		name          string
		cmdOutput     string
		expectedKeys  *lcmcommon.RgwUserKeys
		expectedError string
	}{
		{
			name:          "failed to get rgw ops admin keys",
			expectedError: "failed to get rgw user keys: failed to run command 'radosgw-admin user info --uid rgw-admin-ops-user': run failed",
		},
		{
			name:          "failed to parse rgw ops admin keys",
			cmdOutput:     "{||}",
			expectedError: "failed to parse 'radosgw-admin user info --uid rgw-admin-ops-user' output: invalid character '|' looking for beginning of object key string",
		},
		{
			name:          "admin ops keys not found",
			cmdOutput:     `{"keys": [{"user": "rgw-admin-subuser", "access_key": "5TABLO7H0I6BTW6N25X5", "secret_key": "Wd8SDDrtyyAuiD1klOGn9vJqOJh5dOSVlJ6kir9Q"}]}`,
			expectedError: "failed to find RGW 'rgw-admin-ops-user' user keys",
		},
		{
			name:      "admin ops keys found",
			cmdOutput: `{"user_id": "rgw-admin-ops-user", "keys": [{"user": "rgw-admin-ops-user", "access_key": "5TABLO7H0I6BTW6N25X5","secret_key": "Wd8SDDrtyyAuiD1klOGn9vJqOJh5dOSVlJ6kir9Q"}]}`,
			expectedKeys: &lcmcommon.RgwUserKeys{
				AccessKey: "5TABLO7H0I6BTW6N25X5",
				SecretKey: "Wd8SDDrtyyAuiD1klOGn9vJqOJh5dOSVlJ6kir9Q",
			},
		},
	}
	oldRunCmd := lcmcommon.RunPodCommand
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			faketestclients.FakeReaction(c.Kubeclientset.CoreV1(), "list", []string{"pods"}, map[string]runtime.Object{"pods": unitinputs.ToolBoxPodList}, nil)

			lcmcommon.RunPodCommand = func(_ lcmcommon.ExecConfig) (string, string, error) {
				if test.cmdOutput == "" {
					return "", "", errors.New("run failed")
				}
				return test.cmdOutput, "", nil
			}

			keys, err := c.getRgwKeys("rook-ceph", "rgw-admin-ops-user")
			if test.expectedError != "" {
				assert.NotNil(t, err)
				assert.Equal(t, test.expectedError, err.Error())
			} else {
				assert.Nil(t, err)
				assert.Equal(t, test.expectedKeys, keys)
			}
			faketestclients.CleanupFakeClientReactions(c.Kubeclientset.CoreV1())
		})
	}
	lcmcommon.RunPodCommand = oldRunCmd
}
