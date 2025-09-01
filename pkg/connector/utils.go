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
	"encoding/json"
	"fmt"

	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	lcmcommon "github.com/Mirantis/pelagia/pkg/common"
)

func (c *CephConnector) getClusterFSID(rookNamespace string) (string, error) {
	cephClusters, err := c.Rookclientset.CephV1().CephClusters(rookNamespace).List(c.Context, metav1.ListOptions{})
	if err != nil {
		return "", errors.Wrapf(err, "failed to check CephCluster object in namespace '%s'", rookNamespace)
	}
	if len(cephClusters.Items) > 1 {
		return "", errors.Errorf("multiple CephCluster objects found in namespace '%s'", rookNamespace)
	}
	if len(cephClusters.Items) == 0 {
		return "", errors.Errorf("no CephCluster objects found in namespace '%s'", rookNamespace)
	}
	cephCluster := cephClusters.Items[0]
	if cephCluster.Status.CephStatus == nil {
		return "", errors.Errorf("status is not present for CephCluster '%s/%s'", cephCluster.Namespace, cephCluster.Name)
	}
	if cephCluster.Status.CephStatus.FSID == "" {
		return "", errors.Errorf("cluster fsid is empty in CephCluster '%s/%s' status", cephCluster.Namespace, cephCluster.Name)
	}
	return cephCluster.Status.CephStatus.FSID, nil
}

func (c *CephConnector) getClusterMonEndpoints(rookNamespace string) (string, error) {
	monEndpointsMap, err := c.Kubeclientset.CoreV1().ConfigMaps(rookNamespace).Get(c.Context, lcmcommon.MonMapConfigMapName, metav1.GetOptions{})
	if err != nil {
		return "", errors.Wrapf(err, "failed to get ConfigMap '%s/%s'", rookNamespace, lcmcommon.MonMapConfigMapName)
	}
	endpoints := monEndpointsMap.Data["data"]
	if endpoints == "" {
		return "", errors.Errorf("mon endpoints are empty in ConfigMap '%s/%s'", monEndpointsMap.Namespace, monEndpointsMap.Name)
	}
	return endpoints, nil
}

func (c *CephConnector) getClusterClientKeyring(toolBoxNamespace, toolboxLabel, clientName string) (string, error) {
	command := fmt.Sprintf("ceph auth get-key client.%s", clientName)
	e := lcmcommon.ExecConfig{
		Context:    c.Context,
		Kubeclient: c.Kubeclientset,
		Config:     c.Config,
		Namespace:  toolBoxNamespace,
		Command:    command,
		Labels:     []string{fmt.Sprintf("app=%s", toolboxLabel)},
	}
	keyring, _, err := lcmcommon.RunPodCmdAndCheckError(e)
	if err != nil {
		return "", errors.Wrap(err, "failed to get keyring for client")
	}
	if keyring == "" {
		return "", errors.Errorf("client.%s has empty keyring", clientName)
	}
	return keyring, nil
}

func (c *CephConnector) getRgwKeys(toolBoxNamespace, toolboxLabel, username string) (*lcmcommon.RgwUserKeys, error) {
	cmd := fmt.Sprintf("radosgw-admin user info --uid %s", username)
	e := lcmcommon.ExecConfig{
		Context:    c.Context,
		Kubeclient: c.Kubeclientset,
		Config:     c.Config,
		Namespace:  toolBoxNamespace,
		Command:    cmd,
		Labels:     []string{fmt.Sprintf("app=%s", toolboxLabel)},
	}
	rgwAdminOpsUserInfo, _, err := lcmcommon.RunPodCmdAndCheckError(e)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get rgw user keys")
	}
	var userInfo struct {
		UserID string `json:"user_id"`
		Keys   []struct {
			User      string `json:"user"`
			AccessKey string `json:"access_key"`
			SecretKey string `json:"secret_key"`
		} `json:"keys"`
	}
	err = json.Unmarshal([]byte(rgwAdminOpsUserInfo), &userInfo)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse '%s' output", cmd)
	}
	for _, key := range userInfo.Keys {
		if key.User == userInfo.UserID {
			return &lcmcommon.RgwUserKeys{
				AccessKey: key.AccessKey,
				SecretKey: key.SecretKey,
			}, nil
		}
	}
	return nil, errors.Errorf("failed to find RGW '%s' user keys", username)
}
