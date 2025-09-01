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
	"context"
	"testing"

	"github.com/pkg/errors"
	fakerook "github.com/rook/rook/pkg/client/clientset/versioned/fake"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	fakekube "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/rest"
	gotesting "k8s.io/client-go/testing"

	lcmcommon "github.com/Mirantis/pelagia/pkg/common"
	faketestclients "github.com/Mirantis/pelagia/test/unit/clients"
	unitinputs "github.com/Mirantis/pelagia/test/unit/inputs"
)

func FakeConnector() *CephConnector {
	ks := fakekube.NewSimpleClientset()
	ks.ReactionChain = make([]gotesting.Reactor, 0)
	rs := fakerook.NewSimpleClientset()
	rs.ReactionChain = make([]gotesting.Reactor, 0)
	return &CephConnector{
		Context:       context.TODO(),
		Config:        &rest.Config{},
		Kubeclientset: ks,
		Rookclientset: rs,
	}
}

func TestPrepareConnectionString(t *testing.T) {
	c := FakeConnector()
	tests := []struct {
		name            string
		inputResources  map[string]runtime.Object
		opts            Opts
		cliOutputs      map[string]string
		expectedInfoStr string
		expectedError   string
	}{
		{
			name: "failed to get connection string",
			opts: Opts{RookNamespace: "rook-ceph"},
			inputResources: map[string]runtime.Object{
				"cephclusters": &unitinputs.CephClusterListEmpty,
			},
			expectedError: "failed to prepare connection info: no CephCluster objects found in namespace 'rook-ceph'",
		},
		{
			name: "connection info for admin client prepared",
			opts: Opts{
				RookNamespace:    "rook-ceph",
				AuthClient:       "admin",
				ToolBoxLabel:     lcmcommon.PelagiaToolBox,
				ToolBoxNamespace: "lcm-namespace",
			},
			inputResources: map[string]runtime.Object{
				"cephclusters": &unitinputs.CephClusterListReady,
				"configmaps":   &v1.ConfigMapList{Items: []v1.ConfigMap{unitinputs.RookCephMonEndpointsExternal}},
			},
			cliOutputs:      map[string]string{"ceph auth get-key client.admin": "AQAcpuJiITYXMhAAXaOoAqOKJ4mhNOAqxFb1Hw=="},
			expectedInfoStr: string(unitinputs.ExternalConnectionSecretWithAdmin.Data["connection"]),
		},
		{
			name: "connection full info for non admin client is built",
			opts: Opts{
				RookNamespace:    "rook-ceph",
				AuthClient:       "test",
				UseRBD:           true,
				UseCephFS:        true,
				UseRgw:           true,
				RgwUserName:      "rgw-admin-ops-user",
				ToolBoxLabel:     lcmcommon.PelagiaToolBox,
				ToolBoxNamespace: "lcm-namespace",
			},
			inputResources: map[string]runtime.Object{
				"cephclusters": &unitinputs.CephClusterListReady,
				"configmaps":   &v1.ConfigMapList{Items: []v1.ConfigMap{unitinputs.RookCephMonEndpoints}},
			},
			cliOutputs: map[string]string{
				"ceph auth get-key client.test":                    "some-keyring",
				"ceph auth get-key client.csi-rbd-node":            "some-rbdnode-keyring",
				"ceph auth get-key client.csi-rbd-provisioner":     "some-rbdprovisioner-keyring",
				"ceph auth get-key client.csi-cephfs-node":         "some-cephfsnode-keyring",
				"ceph auth get-key client.csi-cephfs-provisioner":  "some-cephfsprovisioner-keyring",
				"radosgw-admin user info --uid rgw-admin-ops-user": `{"user_id": "rgw-admin-ops-user", "keys": [{"user": "rgw-admin-ops-user", "access_key": "5TABLO7H0I6BTW6N25X5","secret_key": "Wd8SDDrtyyAuiD1klOGn9vJqOJh5dOSVlJ6kir9Q"}]}`,
			},
			expectedInfoStr: `{"client_name":"test","client_keyring":"some-keyring","fsid":"8668f062-3faa-358a-85f3-f80fe6c1e306","mon_endpoints_map":"a=127.0.0.1,b=127.0.0.2,c=127.0.0.3","rbd_keyring_info":{"node_key":"some-rbdnode-keyring","provisioner_key":"some-rbdprovisioner-keyring"},"cephfs_keyring_info":{"node_key":"some-cephfsnode-keyring","provisioner_key":"some-cephfsprovisioner-keyring"},"rgw_admin_keys":{"accessKey":"5TABLO7H0I6BTW6N25X5","secretKey":"Wd8SDDrtyyAuiD1klOGn9vJqOJh5dOSVlJ6kir9Q"}}`,
		},
		{
			name: "connection info for admin client prepared and base64 output",
			opts: Opts{
				RookNamespace:    "rook-ceph",
				AuthClient:       "admin",
				ToolBoxLabel:     lcmcommon.PelagiaToolBox,
				ToolBoxNamespace: "lcm-namespace",
				EncodedBase64:    true,
			},
			inputResources: map[string]runtime.Object{
				"cephclusters": &unitinputs.CephClusterListReady,
				"configmaps":   &v1.ConfigMapList{Items: []v1.ConfigMap{unitinputs.RookCephMonEndpoints}},
			},
			cliOutputs:      map[string]string{"ceph auth get-key client.admin": "some-keyring"},
			expectedInfoStr: "eyJjbGllbnRfbmFtZSI6ImFkbWluIiwiY2xpZW50X2tleXJpbmciOiJzb21lLWtleXJpbmciLCJmc2lkIjoiODY2OGYwNjItM2ZhYS0zNThhLTg1ZjMtZjgwZmU2YzFlMzA2IiwibW9uX2VuZHBvaW50c19tYXAiOiJhPTEyNy4wLjAuMSxiPTEyNy4wLjAuMixjPTEyNy4wLjAuMyJ9",
		},
	}
	oldRunCmd := lcmcommon.RunPodCommand
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			faketestclients.FakeReaction(c.Rookclientset, "list", []string{"cephclusters"}, test.inputResources, nil)
			faketestclients.FakeReaction(c.Kubeclientset.CoreV1(), "get", []string{"configmaps"}, test.inputResources, nil)
			faketestclients.FakeReaction(c.Kubeclientset.CoreV1(), "list", []string{"pods"}, map[string]runtime.Object{"pods": unitinputs.ToolBoxPodList}, nil)

			lcmcommon.RunPodCommand = func(e lcmcommon.ExecConfig) (string, string, error) {
				if output, ok := test.cliOutputs[e.Command]; ok {
					return output, "", nil
				}
				return "", "", errors.New("cmd run failed")
			}

			infoStr, err := c.PrepareConnectionString(test.opts)
			if test.expectedError != "" {
				assert.NotNil(t, err)
				assert.Equal(t, test.expectedError, err.Error())
			} else {
				assert.Nil(t, err)
			}
			assert.Equal(t, test.expectedInfoStr, infoStr)
			faketestclients.CleanupFakeClientReactions(c.Kubeclientset.CoreV1())
			faketestclients.CleanupFakeClientReactions(c.Rookclientset)
		})
	}
	lcmcommon.RunPodCommand = oldRunCmd
}

func TestGetConnectionInfo(t *testing.T) {
	c := FakeConnector()
	nonAdminOpts := Opts{
		RookNamespace:    "rook-ceph",
		AuthClient:       "test",
		UseRBD:           true,
		UseCephFS:        true,
		UseRgw:           true,
		RgwUserName:      "rgw-admin-ops-user",
		ToolBoxLabel:     lcmcommon.PelagiaToolBox,
		ToolBoxNamespace: "lcm-namespace",
	}
	adminOptsNoRgw := Opts{
		RookNamespace:    "rook-ceph",
		AuthClient:       "admin",
		ToolBoxLabel:     lcmcommon.PelagiaToolBox,
		ToolBoxNamespace: "lcm-namespace",
	}
	tests := []struct {
		name           string
		inputResources map[string]runtime.Object
		opts           Opts
		cliOutputs     map[string]string
		expectedInfo   *lcmcommon.CephConnection
		expectedError  string
	}{
		{
			name: "failed to get ceph cluster fsid",
			opts: adminOptsNoRgw,
			inputResources: map[string]runtime.Object{
				"cephclusters": &unitinputs.CephClusterListEmpty,
			},
			expectedError: "no CephCluster objects found in namespace 'rook-ceph'",
		},
		{
			name: "failed to get mon endpoints",
			opts: adminOptsNoRgw,
			inputResources: map[string]runtime.Object{
				"cephclusters": &unitinputs.CephClusterListReady,
				"configmaps":   unitinputs.ConfigMapListEmpty,
			},
			expectedError: "failed to get ConfigMap 'rook-ceph/rook-ceph-mon-endpoints': configmaps \"rook-ceph-mon-endpoints\" not found",
		},
		{
			name: "failed to get client keyring",
			opts: adminOptsNoRgw,
			inputResources: map[string]runtime.Object{
				"cephclusters": &unitinputs.CephClusterListReady,
				"configmaps":   &v1.ConfigMapList{Items: []v1.ConfigMap{unitinputs.RookCephMonEndpoints}},
			},
			expectedError: "failed to get keyring for client: failed to run command 'ceph auth get-key client.admin': cmd run failed",
		},
		{
			name: "connection info for admin client prepared",
			opts: adminOptsNoRgw,
			inputResources: map[string]runtime.Object{
				"cephclusters": &unitinputs.CephClusterListReady,
				"configmaps":   &v1.ConfigMapList{Items: []v1.ConfigMap{unitinputs.RookCephMonEndpoints}},
			},
			cliOutputs: map[string]string{"ceph auth get-key client.admin": "some-keyring"},
			expectedInfo: &lcmcommon.CephConnection{
				ClientName:    "admin",
				ClientKeyring: "some-keyring",
				FSID:          "8668f062-3faa-358a-85f3-f80fe6c1e306",
				MonEndpoints:  "a=127.0.0.1,b=127.0.0.2,c=127.0.0.3",
			},
		},
		{
			name: "failed to get client rbd node keyring",
			opts: nonAdminOpts,
			inputResources: map[string]runtime.Object{
				"cephclusters": &unitinputs.CephClusterListReady,
				"configmaps":   &v1.ConfigMapList{Items: []v1.ConfigMap{unitinputs.RookCephMonEndpoints}},
			},
			cliOutputs:    map[string]string{"ceph auth get-key client.test": "some-keyring"},
			expectedError: "failed to get keyring for client: failed to run command 'ceph auth get-key client.csi-rbd-node': cmd run failed",
		},
		{
			name: "failed to get client rbd provisioner keyring",
			opts: nonAdminOpts,
			inputResources: map[string]runtime.Object{
				"cephclusters": &unitinputs.CephClusterListReady,
				"configmaps":   &v1.ConfigMapList{Items: []v1.ConfigMap{unitinputs.RookCephMonEndpoints}},
			},
			cliOutputs: map[string]string{
				"ceph auth get-key client.test":         "some-keyring",
				"ceph auth get-key client.csi-rbd-node": "some-rbdnode-keyring",
			},
			expectedError: "failed to get keyring for client: failed to run command 'ceph auth get-key client.csi-rbd-provisioner': cmd run failed",
		},
		{
			name: "failed to get client cephfs node keyring",
			opts: nonAdminOpts,
			inputResources: map[string]runtime.Object{
				"cephclusters": &unitinputs.CephClusterListReady,
				"configmaps":   &v1.ConfigMapList{Items: []v1.ConfigMap{unitinputs.RookCephMonEndpoints}},
			},
			cliOutputs: map[string]string{
				"ceph auth get-key client.test":                "some-keyring",
				"ceph auth get-key client.csi-rbd-node":        "some-rbdnode-keyring",
				"ceph auth get-key client.csi-rbd-provisioner": "some-rbdprovisioner-keyring",
			},
			expectedError: "failed to get keyring for client: failed to run command 'ceph auth get-key client.csi-cephfs-node': cmd run failed",
		},
		{
			name: "failed to get client cephfs provisioner keyring",
			opts: nonAdminOpts,
			inputResources: map[string]runtime.Object{
				"cephclusters": &unitinputs.CephClusterListReady,
				"configmaps":   &v1.ConfigMapList{Items: []v1.ConfigMap{unitinputs.RookCephMonEndpoints}},
			},
			cliOutputs: map[string]string{
				"ceph auth get-key client.test":                "some-keyring",
				"ceph auth get-key client.csi-rbd-node":        "some-rbdnode-keyring",
				"ceph auth get-key client.csi-rbd-provisioner": "some-rbdprovisioner-keyring",
				"ceph auth get-key client.csi-cephfs-node":     "some-cephfsnode-keyring",
			},
			expectedError: "failed to get keyring for client: failed to run command 'ceph auth get-key client.csi-cephfs-provisioner': cmd run failed",
		},
		{
			name: "failed to get rgw user keys",
			opts: nonAdminOpts,
			inputResources: map[string]runtime.Object{
				"cephclusters": &unitinputs.CephClusterListReady,
				"configmaps":   &v1.ConfigMapList{Items: []v1.ConfigMap{unitinputs.RookCephMonEndpoints}},
			},
			cliOutputs: map[string]string{
				"ceph auth get-key client.test":                   "some-keyring",
				"ceph auth get-key client.csi-rbd-node":           "some-rbdnode-keyring",
				"ceph auth get-key client.csi-rbd-provisioner":    "some-rbdprovisioner-keyring",
				"ceph auth get-key client.csi-cephfs-node":        "some-cephfsnode-keyring",
				"ceph auth get-key client.csi-cephfs-provisioner": "some-cephfsprovisioner-keyring",
			},
			expectedError: "failed to get rgw user keys: failed to run command 'radosgw-admin user info --uid rgw-admin-ops-user': cmd run failed",
		},
		{
			name: "connection full info for non admin client is built",
			opts: nonAdminOpts,
			inputResources: map[string]runtime.Object{
				"cephclusters": &unitinputs.CephClusterListReady,
				"configmaps":   &v1.ConfigMapList{Items: []v1.ConfigMap{unitinputs.RookCephMonEndpoints}},
			},
			cliOutputs: map[string]string{
				"ceph auth get-key client.test":                    "some-keyring",
				"ceph auth get-key client.csi-rbd-node":            "some-rbdnode-keyring",
				"ceph auth get-key client.csi-rbd-provisioner":     "some-rbdprovisioner-keyring",
				"ceph auth get-key client.csi-cephfs-node":         "some-cephfsnode-keyring",
				"ceph auth get-key client.csi-cephfs-provisioner":  "some-cephfsprovisioner-keyring",
				"radosgw-admin user info --uid rgw-admin-ops-user": `{"user_id": "rgw-admin-ops-user", "keys": [{"user": "rgw-admin-ops-user", "access_key": "5TABLO7H0I6BTW6N25X5","secret_key": "Wd8SDDrtyyAuiD1klOGn9vJqOJh5dOSVlJ6kir9Q"}]}`,
			},
			expectedInfo: &lcmcommon.CephConnection{
				ClientName:    "test",
				ClientKeyring: "some-keyring",
				FSID:          "8668f062-3faa-358a-85f3-f80fe6c1e306",
				MonEndpoints:  "a=127.0.0.1,b=127.0.0.2,c=127.0.0.3",
				RBDKeyring: &lcmcommon.CSIKeyring{
					NodeKey:        "some-rbdnode-keyring",
					ProvisionerKey: "some-rbdprovisioner-keyring",
				},
				CephFSKeyring: &lcmcommon.CSIKeyring{
					NodeKey:        "some-cephfsnode-keyring",
					ProvisionerKey: "some-cephfsprovisioner-keyring",
				},
				RgwAdminUserKeys: &lcmcommon.RgwUserKeys{
					AccessKey: "5TABLO7H0I6BTW6N25X5",
					SecretKey: "Wd8SDDrtyyAuiD1klOGn9vJqOJh5dOSVlJ6kir9Q",
				},
			},
		},
		{
			name: "connection partial info for non admin client is built",
			opts: Opts{
				RookNamespace:    "rook-ceph",
				AuthClient:       "test",
				UseCephFS:        true,
				ToolBoxLabel:     lcmcommon.PelagiaToolBox,
				ToolBoxNamespace: "lcm-namespace",
			},
			inputResources: map[string]runtime.Object{
				"cephclusters": &unitinputs.CephClusterListReady,
				"configmaps":   &v1.ConfigMapList{Items: []v1.ConfigMap{unitinputs.RookCephMonEndpoints}},
			},
			cliOutputs: map[string]string{
				"ceph auth get-key client.test":                   "some-keyring",
				"ceph auth get-key client.csi-cephfs-node":        "some-cephfsnode-keyring",
				"ceph auth get-key client.csi-cephfs-provisioner": "some-cephfsprovisioner-keyring",
			},
			expectedInfo: &lcmcommon.CephConnection{
				ClientName:    "test",
				ClientKeyring: "some-keyring",
				FSID:          "8668f062-3faa-358a-85f3-f80fe6c1e306",
				MonEndpoints:  "a=127.0.0.1,b=127.0.0.2,c=127.0.0.3",
				CephFSKeyring: &lcmcommon.CSIKeyring{
					NodeKey:        "some-cephfsnode-keyring",
					ProvisionerKey: "some-cephfsprovisioner-keyring",
				},
			},
		},
		{
			name: "connection partial #2 info for non admin client is built",
			opts: Opts{
				RookNamespace:    "rook-ceph",
				AuthClient:       "test",
				UseRBD:           true,
				ToolBoxLabel:     lcmcommon.PelagiaToolBox,
				ToolBoxNamespace: "lcm-namespace",
			},
			inputResources: map[string]runtime.Object{
				"cephclusters": &unitinputs.CephClusterListReady,
				"configmaps":   &v1.ConfigMapList{Items: []v1.ConfigMap{unitinputs.RookCephMonEndpoints}},
			},
			cliOutputs: map[string]string{
				"ceph auth get-key client.test":                "some-keyring",
				"ceph auth get-key client.csi-rbd-node":        "some-rbdnode-keyring",
				"ceph auth get-key client.csi-rbd-provisioner": "some-rbdprovisioner-keyring",
			},
			expectedInfo: &lcmcommon.CephConnection{
				ClientName:    "test",
				ClientKeyring: "some-keyring",
				FSID:          "8668f062-3faa-358a-85f3-f80fe6c1e306",
				MonEndpoints:  "a=127.0.0.1,b=127.0.0.2,c=127.0.0.3",
				RBDKeyring: &lcmcommon.CSIKeyring{
					NodeKey:        "some-rbdnode-keyring",
					ProvisionerKey: "some-rbdprovisioner-keyring",
				},
			},
		},
	}
	oldRunCmd := lcmcommon.RunPodCommand
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			faketestclients.FakeReaction(c.Rookclientset, "list", []string{"cephclusters"}, test.inputResources, nil)
			faketestclients.FakeReaction(c.Kubeclientset.CoreV1(), "get", []string{"configmaps"}, test.inputResources, nil)
			faketestclients.FakeReaction(c.Kubeclientset.CoreV1(), "list", []string{"pods"}, map[string]runtime.Object{"pods": unitinputs.ToolBoxPodList}, nil)

			lcmcommon.RunPodCommand = func(e lcmcommon.ExecConfig) (string, string, error) {
				if output, ok := test.cliOutputs[e.Command]; ok {
					return output, "", nil
				}
				return "", "", errors.New("cmd run failed")
			}

			info, err := c.getConnectionInfo(test.opts)
			if test.expectedError != "" {
				assert.NotNil(t, err)
				assert.Equal(t, test.expectedError, err.Error())
				assert.Nil(t, info)
			} else {
				assert.Nil(t, err)
				assert.Equal(t, test.expectedInfo, info)
			}
			faketestclients.CleanupFakeClientReactions(c.Kubeclientset.CoreV1())
			faketestclients.CleanupFakeClientReactions(c.Rookclientset)
		})
	}
	lcmcommon.RunPodCommand = oldRunCmd
}
