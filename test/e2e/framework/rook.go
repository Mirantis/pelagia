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

func (c *ManagedConfig) ListCephObjectStore() ([]rookv1.CephObjectStore, error) {
	rgws, err := c.RookClientset.CephV1().CephObjectStores(c.LcmConfig.RookNamespace).List(c.Context, metav1.ListOptions{})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to list CephObjectStore in %s namespace", c.LcmConfig.RookNamespace)
	}
	return rgws.Items, nil
}
