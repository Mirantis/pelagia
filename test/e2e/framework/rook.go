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

package framework

import (
	"fmt"
	"strconv"

	"github.com/pkg/errors"
	rookv1 "github.com/rook/rook/pkg/apis/ceph.rook.io/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (c *ManagedConfig) GetCephCluster(clusterName string) (*rookv1.CephCluster, error) {
	cephcluster, err := c.RookClientset.CephV1().CephClusters(c.LcmConfig.RookNamespace).Get(c.Context, clusterName, metav1.GetOptions{})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get CephCluster '%s/%s'", c.LcmConfig.RookNamespace, clusterName)
	}
	return cephcluster, nil
}

func (c *ManagedConfig) GetCephBlockPool(name string) (*rookv1.CephBlockPool, error) {
	cephcluster, err := c.RookClientset.CephV1().CephBlockPools(c.LcmConfig.RookNamespace).Get(c.Context, name, metav1.GetOptions{})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get CephBlockPool %s/%s", c.LcmConfig.RookNamespace, name)
	}
	return cephcluster, nil
}

func (c *ManagedConfig) GetCephClient(name string) (*rookv1.CephClient, error) {
	cephclient, err := c.RookClientset.CephV1().CephClients(c.LcmConfig.RookNamespace).Get(c.Context, name, metav1.GetOptions{})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get CephClient %s/%s", c.LcmConfig.RookNamespace, name)
	}
	return cephclient, nil
}

func (c *ManagedConfig) GetCephClientByRole(role string) (*rookv1.CephClient, error) {
	listOpts := metav1.ListOptions{
		LabelSelector: fmt.Sprintf("cephdeployment.lcm.mirantis.com/client-role=%s", role),
	}
	cephclients, err := c.RookClientset.CephV1().CephClients(c.LcmConfig.RookNamespace).List(c.Context, listOpts)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to list CephClients in '%s' namespace", c.LcmConfig.RookNamespace)
	}
	rot := -1
	tidx := -1
	for idx, cephClient := range cephclients.Items {
		if _, ok := cephClient.Labels["cephdeployment.lcm.mirantis.com/client-latest"]; ok {
			return &cephClient, nil
		}
		if v, ok := cephClient.Labels["cephdeployment.lcm.mirantis.com/client-rotation"]; ok {
			rotation, _ := strconv.Atoi(v)
			if rotation > rot {
				rot = rotation
				tidx = idx
			}
		}
	}
	if tidx == -1 {
		return nil, errors.Errorf("failed to find CephClient with role '%s'", role)
	}
	return &cephclients.Items[tidx], nil
}

func (c *ManagedConfig) ListCephObjectStore() ([]rookv1.CephObjectStore, error) {
	rgws, err := c.RookClientset.CephV1().CephObjectStores(c.LcmConfig.RookNamespace).List(c.Context, metav1.ListOptions{})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to list CephObjectStore in %s namespace", c.LcmConfig.RookNamespace)
	}
	return rgws.Items, nil
}
