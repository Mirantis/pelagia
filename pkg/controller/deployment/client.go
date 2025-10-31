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
	"fmt"
	"reflect"
	"strings"

	"github.com/pkg/errors"
	cephv1 "github.com/rook/rook/pkg/apis/ceph.rook.io/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	cephlcmv1alpha1 "github.com/Mirantis/pelagia/pkg/apis/ceph.pelagia.lcm/v1alpha1"
	lcmcommon "github.com/Mirantis/pelagia/pkg/common"
)

func (c *cephDeploymentConfig) ensureCephClients() (bool, error) {
	c.log.Debug().Msg("ensure ceph clients")
	cephClients, err := c.api.Rookclientset.CephV1().CephClients(c.lcmConfig.RookNamespace).List(c.context, metav1.ListOptions{})
	if err != nil {
		return false, errors.Wrapf(err, "failed to list CephClients in %s namespace", c.lcmConfig.RookNamespace)
	}

	presentClients := map[string]cephv1.CephClient{}
	for _, client := range cephClients.Items {
		presentClients[client.Name] = client
	}
	errMsg := make([]error, 0)

	// If there is any additional OpenStack clients required, add them to clients list
	cephDplClients := c.cdConfig.cephDpl.Spec.Clients
	if !c.cdConfig.cephDpl.Spec.External && lcmcommon.IsOpenStackPoolsPresent(c.cdConfig.cephDpl.Spec.Pools) {
		osClients, err := c.calculateOpenStackClients()
		if err != nil {
			return false, errors.Wrap(err, "failed to calculate OpenStack CephClients")
		}
		cephDplClients = append(cephDplClients, osClients...)
	}

	clientsChanged := false
	for _, cephDplClient := range cephDplClients {
		newClient := generateClient(c.lcmConfig.RookNamespace, cephDplClient.Name, cephDplClient.Caps)
		if presentClient, ok := presentClients[newClient.Name]; ok {
			if presentClient.Status == nil || !isTypeReadyToUpdate(presentClient.Status.Phase) {
				err := fmt.Sprintf("found not ready CephClient %s/%s, waiting for readiness", c.lcmConfig.RookNamespace, presentClient.Name)
				if presentClient.Status != nil {
					err = fmt.Sprintf("%s (current phase is %v)", err, presentClient.Status.Phase)
				}
				c.log.Error().Msg(err)
				errMsg = append(errMsg, errors.New(err))
			} else {
				if !reflect.DeepEqual(newClient.Spec, presentClient.Spec) {
					lcmcommon.ShowObjectDiff(*c.log, presentClient.Spec, newClient.Spec)
					presentClient.Spec = newClient.Spec
					if err := c.processCephClients(objectUpdate, presentClient); err != nil {
						errMsg = append(errMsg, err)
					}
					clientsChanged = true
				}
			}
			delete(presentClients, newClient.Name)
		} else {
			if err := c.processCephClients(objectCreate, newClient); err != nil {
				errMsg = append(errMsg, err)
			}
			clientsChanged = true
		}
	}

	for _, client := range presentClients {
		if err := c.processCephClients(objectDelete, client); err != nil {
			errMsg = append(errMsg, err)
		}
		clientsChanged = true
	}

	// Return error if exists
	if len(errMsg) == 1 {
		return false, errors.Wrap(errMsg[0], "failed to ensure CephClients")
	} else if len(errMsg) > 1 {
		return false, errors.New("failed to ensure CephClients, multiple errors during CephClients ensure")
	}
	return clientsChanged, nil
}

func (c *cephDeploymentConfig) deleteCephClients() (bool, error) {
	clientList, err := c.api.Rookclientset.CephV1().CephClients(c.lcmConfig.RookNamespace).List(c.context, metav1.ListOptions{})
	if err != nil {
		return false, errors.Wrap(err, "failed to list ceph clients")
	}
	if len(clientList.Items) == 0 {
		return true, nil
	}
	errMsg := 0
	for _, client := range clientList.Items {
		if err := c.processCephClients(objectDelete, client); err != nil {
			errMsg++
		}
	}
	if errMsg > 0 {
		return false, errors.New("some CephClients failed to delete")
	}
	return false, nil
}

func (c *cephDeploymentConfig) processCephClients(process objectProcess, client cephv1.CephClient) error {
	var err error
	switch process {
	case objectCreate:
		c.log.Info().Msgf("creating CephClient %s/%s", client.Namespace, client.Name)
		_, err = c.api.Rookclientset.CephV1().CephClients(client.Namespace).Create(c.context, &client, metav1.CreateOptions{})
	case objectUpdate:
		c.log.Info().Msgf("updating CephClient %s/%s", client.Namespace, client.Name)
		_, err = c.api.Rookclientset.CephV1().CephClients(client.Namespace).Update(c.context, &client, metav1.UpdateOptions{})
	case objectDelete:
		c.log.Info().Msgf("deleting CephClient %s/%s", client.Namespace, client.Name)
		err = c.api.Rookclientset.CephV1().CephClients(client.Namespace).Delete(c.context, client.Name, metav1.DeleteOptions{})
	}
	if err != nil {
		if process == objectDelete && apierrors.IsNotFound(err) {
			return nil
		}
		err = errors.Wrapf(err, "failed to %v CephClient %s/%s", process, client.Namespace, client.Name)
		c.log.Error().Err(err).Msg("")
		return err
	}
	return nil
}

func (c *cephDeploymentConfig) calculateOpenStackClients() ([]cephlcmv1alpha1.CephClient, error) {
	notUsed := map[string]bool{"cinder": true, "glance": true, "nova": true, "manila": true}
	for _, cephDplClient := range c.cdConfig.cephDpl.Spec.Clients {
		switch cephDplClient.Name {
		case "cinder":
			notUsed["cinder"] = false
		case "glance":
			notUsed["glance"] = false
		case "nova":
			notUsed["nova"] = false
		case "manila":
			notUsed["manila"] = false
		}
	}

	clients := make([]cephlcmv1alpha1.CephClient, 0)
	for client, notUsedStatus := range notUsed {
		if notUsedStatus {
			// do not generate manila client if there is no cephfs enabled
			if client == "manila" && c.cdConfig.cephDpl.Spec.SharedFilesystem == nil {
				continue
			}
			osClient, err := c.generateOpenStackClient(client)
			if err != nil {
				c.log.Error().Err(err).Msgf("failed to generate spec for Ceph openstack client %s", client)
				return nil, errors.Wrapf(err, "failed to generate spec for Ceph openstack client %s", client)
			}
			clients = append(clients, osClient)
		}
	}
	return clients, nil
}

func (c *cephDeploymentConfig) generateOpenStackClient(name string) (cephlcmv1alpha1.CephClient, error) {
	pools := map[string][]string{"vms": nil, "volumes": nil, "images": nil, "backup": nil}
	for _, pool := range c.cdConfig.cephDpl.Spec.Pools {
		switch pool.Role {
		case "images":
			pools["images"] = []string{buildPoolName(pool)}
		case "vms":
			pools["vms"] = []string{buildPoolName(pool)}
		case "backup":
			pools["backup"] = []string{buildPoolName(pool)}
		case "volumes":
			pools["volumes"] = append(pools["volumes"], buildPoolName(pool))
		case "volumes-backend":
			// set basic volumes role
			pool.Role = "volumes"
			pools["volumes"] = append(pools["volumes"], buildPoolName(pool))
		}
	}

	checkPoolsFn := func(name string, poolTypes []string) error {
		for _, poolType := range poolTypes {
			if len(pools[poolType]) == 0 {
				return errors.Errorf("ceph block pool with role %s not found in pools", poolType)
			}
			for _, pool := range pools[poolType] {
				_, err := c.api.Rookclientset.CephV1().CephBlockPools(c.lcmConfig.RookNamespace).Get(c.context, pool, metav1.GetOptions{})
				if err != nil {
					return errors.Wrapf(err, "failed to get one of the required cephblockpools for %v client", name)
				}
			}
		}
		return nil
	}

	volumeBackendsProfiles := []string{}
	for _, v := range pools["volumes"] {
		volumeBackendsProfiles = append(volumeBackendsProfiles, fmt.Sprintf("profile rbd pool=%s", v))
	}
	volumeBackendsProfile := strings.Join(volumeBackendsProfiles, ", ")

	client := cephlcmv1alpha1.CephClient{}
	switch name {
	case "cinder":
		if err := checkPoolsFn(name, []string{"volumes", "images", "backup"}); err != nil {
			return client, err
		}
		client.Name = name
		client.Caps = map[string]string{
			"mon": "allow profile rbd",
			"osd": fmt.Sprintf("%s, profile rbd-read-only pool=%s, profile rbd pool=%s", volumeBackendsProfile, pools["images"][0], pools["backup"][0]),
		}
		return client, nil
	case "glance":
		if err := checkPoolsFn(name, []string{"images"}); err != nil {
			return client, err
		}
		client.Name = name
		client.Caps = map[string]string{
			"mon": "allow profile rbd",
			"osd": `profile rbd pool=` + pools["images"][0],
		}
		return client, nil
	case "nova":
		if err := checkPoolsFn(name, []string{"vms", "images", "volumes"}); err != nil {
			return client, err
		}
		client.Name = name
		client.Caps = map[string]string{
			"mon": "allow profile rbd",
			"osd": fmt.Sprintf("profile rbd pool=%s, profile rbd pool=%s, %s", pools["vms"][0], pools["images"][0], volumeBackendsProfile),
		}
		return client, nil
	case "manila":
		client.Name = name
		client.Caps = map[string]string{
			"mds": "allow rw",
			"mgr": "allow rw",
			"osd": "allow rw tag cephfs *=*",
			"mon": `allow r, allow command "auth del", allow command "auth caps", allow command "auth get", allow command "auth get-or-create"`,
		}
		return client, nil
	}
	return client, errors.Errorf("failed to find pool type for '%s' client", name)
}

func generateClient(namespace string, name string, caps map[string]string) cephv1.CephClient {
	return cephv1.CephClient{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: cephv1.ClientSpec{
			Caps: caps,
			Name: name,
		},
	}
}
