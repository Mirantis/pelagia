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
	"strconv"
	"strings"

	"github.com/pkg/errors"
	cephv1 "github.com/rook/rook/pkg/apis/ceph.rook.io/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	cephlcmv1alpha1 "github.com/Mirantis/pelagia/v3/pkg/apis/ceph.pelagia.lcm/v1alpha1"
	lcmcommon "github.com/Mirantis/pelagia/v3/pkg/common"
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
	expectedClients := make([]cephv1.CephClient, len(c.cdConfig.cephDpl.Spec.Clients))
	for idx, cephDplClient := range c.cdConfig.cephDpl.Spec.Clients {
		clientSpec, _ := cephDplClient.GetSpec()
		expectedClients[idx] = generateClient(c.lcmConfig.RookNamespace, clientSpec, cephDplClient.Role)
	}

	if !c.cdConfig.clusterSpec.External.Enable && c.cdConfig.openstackSetup {
		osClients, err := c.defaultOpenstackClients(c.cdConfig.cephDpl.Spec.Clients)
		if err != nil {
			return false, errors.Wrap(err, "failed to verify default OpenStack CephClients")
		}
		for role, clientSpec := range osClients {
			defaultOsClient := generateClient(c.lcmConfig.RookNamespace, clientSpec, role)
			expectedRotation := int16(0)
			minRotationToKeep := int16(0)
			if c.cdConfig.cephDpl.Spec.ExtraOpts != nil {
				expectedRotation = c.cdConfig.cephDpl.Spec.ExtraOpts.RotateOsClients.Rotation
				minRotationToKeep = c.cdConfig.cephDpl.Spec.ExtraOpts.RotateOsClients.Rotation - int16(c.cdConfig.cephDpl.Spec.ExtraOpts.RotateOsClients.KeepPrevious)
			}
			// add rotation label for clients created by pelagia
			newLabels := map[string]string{cephDeploymentClientRotationLabel: fmt.Sprintf("%d", expectedRotation)}
			defaultOsClient.Labels = lcmcommon.ExtendLabels(newLabels, defaultOsClient.Labels)
			for _, cephClient := range cephClients.Items {
				// since clients names are generated - find by label previously generated if exist
				// or find by raw service name, eg. nova - for clients created before 3.x release
				clientLabel := ""
				clientRotation := int16(-1)
				if cephClient.Labels != nil {
					clientLabel = cephClient.Labels[cephDeploymentClientRoleLabel]
					rotationStr, ok := cephClient.Labels[cephDeploymentClientRotationLabel]
					if ok {
						rotation, err := strconv.Atoi(rotationStr)
						if err != nil {
							return false, errors.Wrapf(err, "failed to check rotation '%s/%s' CephClient: label '%s' must be of int value",
								cephClient.Namespace, cephClient.Name, cephDeploymentClientRotationLabel)
						}
						clientRotation = int16(rotation)
					}
				}
				if clientLabel == role {
					if expectedRotation == clientRotation {
						defaultOsClient.Name = cephClient.Name
						defaultOsClient.Spec.Name = cephClient.Spec.Name
						break
					}
					if minRotationToKeep <= clientRotation {
						// keep client as is w/o any updates
						delete(presentClients, cephClient.Name)
					}
				} else {
					// fallback for previous client names
					// will be used only once after upgrade to set correct labels
					// TODO: remove in 4.x
					if cephClient.Spec.Name == role && clientLabel == "" {
						defaultOsClient.Name = cephClient.Name
						defaultOsClient.Spec.Name = cephClient.Spec.Name
						break
					}
				}
			}
			expectedClients = append(expectedClients, defaultOsClient)
		}
	}

	clientsChanged := false
	for _, cephClient := range expectedClients {
		if presentClient, ok := presentClients[cephClient.Name]; ok {
			if presentClient.Status == nil || !isTypeReadyToUpdate(presentClient.Status.Phase) {
				err := fmt.Sprintf("found not ready CephClient %s/%s, waiting for readiness", c.lcmConfig.RookNamespace, presentClient.Name)
				if presentClient.Status != nil {
					err = fmt.Sprintf("%s (current phase is %v)", err, presentClient.Status.Phase)
				}
				c.log.Error().Msg(err)
				errMsg = append(errMsg, errors.New(err))
			} else {
				labelsUpdated := lcmcommon.AlignBaseLabels(*c.log, "CephClient", &presentClient.ObjectMeta, cephClient.Labels)
				specUpdated := !reflect.DeepEqual(cephClient.Spec, presentClient.Spec)
				if specUpdated {
					lcmcommon.ShowObjectDiff(*c.log, presentClient.Spec, cephClient.Spec)
					presentClient.Spec = cephClient.Spec
				}
				if specUpdated || labelsUpdated {
					if err := c.processCephClients(objectUpdate, presentClient); err != nil {
						errMsg = append(errMsg, err)
					}
					clientsChanged = true
				}
			}
			delete(presentClients, cephClient.Name)
		} else {
			if err := c.processCephClients(objectCreate, cephClient); err != nil {
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

func (c *cephDeploymentConfig) defaultOpenstackClients(specClients []cephlcmv1alpha1.CephClient) (map[string]cephv1.ClientSpec, error) {
	defaultClients := map[string]cephv1.ClientSpec{"cinder": {}, "glance": {}, "nova": {}, "manila": {}}
	// do not generate manila client if there is no cephfs enabled
	if c.cdConfig.cephDpl.Spec.SharedFilesystem == nil || len(c.cdConfig.cephDpl.Spec.SharedFilesystem.Filesystems) == 0 {
		delete(defaultClients, "manila")
	}
	for _, client := range specClients {
		switch client.Role {
		case "cinder", "glance", "nova", "manila":
			delete(defaultClients, client.Role)
		}
	}
	if len(defaultClients) == 0 {
		return nil, nil
	}

	errs := 0
	for client := range defaultClients {
		osClientSpec, err := c.generateOpenStackClient(client)
		if err != nil {
			c.log.Error().Err(err).Msgf("failed to generate spec for Ceph Openstack client %s", client)
			errs++
		}
		defaultClients[client] = osClientSpec
	}
	if errs > 0 {
		return nil, errors.New("failed to generate default Openstack Ceph client(s)")
	}
	return defaultClients, nil
}

func (c *cephDeploymentConfig) generateOpenStackClient(name string) (cephv1.ClientSpec, error) {
	client := cephv1.ClientSpec{}
	pools := map[string][]string{"vms": nil, "volumes": nil, "images": nil, "backup": nil}
	for idx, pool := range c.cdConfig.cephDpl.Spec.BlockStorage.Pools {
		poolName := c.cdConfig.pools[idx]

		switch pool.Role {
		case "images":
			pools["images"] = []string{poolName}
		case "vms":
			pools["vms"] = []string{poolName}
		case "backup":
			pools["backup"] = []string{poolName}
		case "volumes":
			pools["volumes"] = append(pools["volumes"], poolName)
		case "volumes-backend":
			// set basic volumes role
			pool.Role = "volumes"
			pools["volumes"] = append(pools["volumes"], poolName)
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

	switch name {
	case "cinder":
		if err := checkPoolsFn(name, []string{"volumes", "images", "backup"}); err != nil {
			return client, err
		}
		client.Caps = map[string]string{
			"mon": "allow profile rbd",
			"osd": fmt.Sprintf("%s, profile rbd-read-only pool=%s, profile rbd pool=%s", volumeBackendsProfile, pools["images"][0], pools["backup"][0]),
		}
	case "glance":
		if err := checkPoolsFn(name, []string{"images"}); err != nil {
			return client, err
		}
		client.Caps = map[string]string{
			"mon": "allow profile rbd",
			"osd": `profile rbd pool=` + pools["images"][0],
		}
	case "nova":
		if err := checkPoolsFn(name, []string{"vms", "images", "volumes"}); err != nil {
			return client, err
		}
		client.Caps = map[string]string{
			"mon": "allow profile rbd",
			"osd": fmt.Sprintf("profile rbd pool=%s, profile rbd pool=%s, %s", pools["vms"][0], pools["images"][0], volumeBackendsProfile),
		}
	case "manila":
		client.Caps = map[string]string{
			"mds": "allow rw",
			"mgr": "allow rw",
			"osd": "allow rw tag cephfs *=*",
			"mon": `allow r, allow command "auth del", allow command "auth caps", allow command "auth get", allow command "auth get-or-create"`,
		}
	default:
		return client, errors.Errorf("failed to find pool type for '%s' client", name)
	}
	client.Name = lcmcommon.RandomizedName(name)
	return client, nil
}

func generateClient(namespace string, clientSpec cephv1.ClientSpec, role string) cephv1.CephClient {
	// remove dots and underscores
	clientName := strings.ReplaceAll(strings.ReplaceAll(clientSpec.Name, ".", "-"), "_", "-")
	client := cephv1.CephClient{
		ObjectMeta: metav1.ObjectMeta{
			Name:      strings.ToLower(clientName),
			Namespace: namespace,
			Labels:    baseResourceLabels,
		},
		Spec: clientSpec,
	}
	if role != "" {
		client.Labels = lcmcommon.ExtendLabels(map[string]string{cephDeploymentClientRoleLabel: role}, baseResourceLabels)
	}
	return client
}
