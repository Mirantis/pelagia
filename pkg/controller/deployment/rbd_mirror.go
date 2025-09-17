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

	"github.com/pkg/errors"
	cephv1 "github.com/rook/rook/pkg/apis/ceph.rook.io/v1"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	cephlcmv1alpha1 "github.com/Mirantis/pelagia/pkg/apis/ceph.pelagia.lcm/v1alpha1"
	lcmcommon "github.com/Mirantis/pelagia/pkg/common"
)

func (c *cephDeploymentConfig) ensureRBDMirroring() (bool, error) {
	// remove CephRBDMirror while it isn't defined in spec
	if c.cdConfig.cephDpl.Spec.RBDMirror == nil {
		c.log.Debug().Msg("no Ceph RBD Mirroring section specified, ensure resource is not exist")
		removed, err := c.deleteRBDMirroring()
		if err != nil {
			return false, errors.Wrap(err, "failed to cleanup CephRBDMirrors")
		}
		return !removed, nil
	}
	c.log.Debug().Msg("ensure Ceph RBD Mirroring")

	// check if no additional CephRBDMirror exist
	mirrors, err := c.api.Rookclientset.CephV1().CephRBDMirrors(c.lcmConfig.RookNamespace).List(c.context, metav1.ListOptions{})
	if err != nil {
		return false, errors.Wrapf(err, "failed to get list of CephRBDMirrors")
	}
	var rbdMirror *cephv1.CephRBDMirror
	rbdConfgChanged := false
	for _, mirror := range mirrors.Items {
		if mirror.Name == c.cdConfig.cephDpl.Name {
			rbdMirror = mirror.DeepCopy()
			continue
		}
		c.log.Warn().Msgf("found unknown CephRBDMirror %v, removing...", mirror.Name)
		err = c.api.Rookclientset.CephV1().CephRBDMirrors(c.lcmConfig.RookNamespace).Delete(c.context, mirror.Name, metav1.DeleteOptions{})
		if err != nil && !apierrors.IsNotFound(err) {
			return false, errors.Wrapf(err, "failed to remove unspecified %v CephRBDMirror", mirror.Name)
		}
		rbdConfgChanged = true
	}

	// ensure secretes before cephrbdmirror will be created
	changed, err := c.ensureRBDSecrets()
	if err != nil {
		return false, err
	}
	rbdConfgChanged = rbdConfgChanged || changed

	// get current CephRBDMirror and create new one if doesn't exist
	newRbdMirror := generateRBDMirroring(c.cdConfig.cephDpl, c.lcmConfig.RookNamespace)
	if rbdMirror == nil {
		c.log.Info().Msg("creating CephRBDMirror")
		_, err := c.api.Rookclientset.CephV1().CephRBDMirrors(c.lcmConfig.RookNamespace).Create(c.context, newRbdMirror, metav1.CreateOptions{})
		if err != nil {
			return false, errors.Wrapf(err, "failed to create %s CephRBDMirror", c.cdConfig.cephDpl.Name)
		}
		return true, nil
	}

	// check for status
	if rbdMirror.Status == nil || rbdMirror.Status.Phase == "" || !isTypeReadyToUpdate(cephv1.ConditionType(rbdMirror.Status.Phase)) {
		status := "not available"
		if rbdMirror.Status != nil {
			status = rbdMirror.Status.Phase
		}
		return false, errors.Errorf("resource RBDMirror is not ready, status is %s, waiting", status)
	}

	// update existing object
	if !reflect.DeepEqual(rbdMirror.Spec, newRbdMirror.Spec) {
		c.log.Info().Msgf("updating CephRBDMirror %q", newRbdMirror.Name)
		lcmcommon.ShowObjectDiff(*c.log, rbdMirror.Spec, newRbdMirror.Spec)
		rbdMirror.Spec = newRbdMirror.Spec
		_, err := c.api.Rookclientset.CephV1().CephRBDMirrors(c.lcmConfig.RookNamespace).Update(c.context, rbdMirror, metav1.UpdateOptions{})
		if err != nil {
			return false, errors.Wrap(err, "failed to update CephRBDMirror")
		}
		rbdConfgChanged = true
	}
	return rbdConfgChanged, nil
}

func (c *cephDeploymentConfig) deleteRBDMirroring() (bool, error) {
	rbdRemoved := true
	err := c.api.Rookclientset.CephV1().CephRBDMirrors(c.lcmConfig.RookNamespace).Delete(c.context, c.cdConfig.cephDpl.Name, metav1.DeleteOptions{})
	if err != nil {
		if !apierrors.IsNotFound(err) {
			return false, errors.Wrapf(err, "failed to remove CephRBDMirroring")
		}
	} else {
		c.log.Info().Msgf("CephRBDMirror %q removed", c.cdConfig.cephDpl.Name)
		rbdRemoved = false
	}
	removedSecrets, err := c.deleteRBDSecrets()
	return rbdRemoved && removedSecrets, err
}

func (c *cephDeploymentConfig) ensureRBDSecrets() (bool, error) {
	c.log.Debug().Msg("ensuring RBD secrets")

	if len(c.cdConfig.cephDpl.Spec.RBDMirror.Peers) == 0 {
		removed, err := c.deleteRBDSecrets()
		if err != nil {
			return false, err
		}
		return !removed, nil
	}

	changedSecrets := false
	for _, peer := range c.cdConfig.cephDpl.Spec.RBDMirror.Peers {
		for _, pool := range peer.Pools {
			// check if mirroring is enabled on pool
			for _, poolDef := range c.cdConfig.cephDpl.Spec.Pools {
				if pool == buildPoolName(poolDef) && poolDef.Mirroring == nil {
					c.log.Warn().Msgf("Adds a secret for %v pool in which mirroring is disabled", pool)
				}
			}

			name := secretName(peer.Site, pool)
			genSecret := generateRBDSecret(peer.Site, peer.Token, pool, c.lcmConfig.RookNamespace)
			kubeSecret, err := c.api.Kubeclientset.CoreV1().Secrets(c.lcmConfig.RookNamespace).Get(c.context, name, metav1.GetOptions{})
			if err != nil {
				if !apierrors.IsNotFound(err) {
					return false, errors.Wrapf(err, "failed to get %v secret", name)
				}
				c.log.Info().Msgf("creating %s secret for RBD Mirror %s", name, c.cdConfig.cephDpl.Name)
				_, err = c.api.Kubeclientset.CoreV1().Secrets(c.lcmConfig.RookNamespace).Create(c.context, genSecret, metav1.CreateOptions{})
				if err != nil {
					return false, errors.Wrapf(err, "failed to create %v secret", name)
				}
				changedSecrets = true
				continue
			}

			if !reflect.DeepEqual(genSecret.Data, kubeSecret.Data) {
				c.log.Info().Msgf("updating %s secret for RBD Mirror %s", kubeSecret.Name, c.cdConfig.cephDpl.Name)
				kubeSecret.Data = genSecret.Data
				_, err := c.api.Kubeclientset.CoreV1().Secrets(c.lcmConfig.RookNamespace).Update(c.context, kubeSecret, metav1.UpdateOptions{})
				if err != nil {
					return false, errors.Wrapf(err, "failed to update %v secret", name)
				}
				changedSecrets = true
			}
		}
	}
	return changedSecrets, nil
}

func (c *cephDeploymentConfig) deleteRBDSecrets() (bool, error) {
	listOptions := metav1.ListOptions{
		FieldSelector: "type=RBDPeer",
	}
	secrets, err := c.api.Kubeclientset.CoreV1().Secrets(c.lcmConfig.RookNamespace).List(c.context, listOptions)
	if err != nil {
		return false, errors.Wrapf(err, "failed to get list rbd secrets to remove")
	}
	if len(secrets.Items) == 0 {
		return true, nil
	}
	c.log.Info().Msg("removing secrets used for RBD mirroring")
	err = c.api.Kubeclientset.CoreV1().Secrets(c.lcmConfig.RookNamespace).DeleteCollection(c.context, metav1.DeleteOptions{}, listOptions)
	if err != nil && !apierrors.IsNotFound(err) {
		c.log.Error().Err(err).Msg("failed to remove secrets used for RBDMirror")
		return false, errors.Errorf("failed to remove rbd secrets: %v", err)
	}
	return false, nil
}

func buildRBDMirrorSecretName(peers []cephlcmv1alpha1.CephRBDMirrorSecret) []string {
	var listOfSecrets []string
	for _, peer := range peers {
		for _, pool := range peer.Pools {
			listOfSecrets = append(listOfSecrets, secretName(peer.Site, pool))
		}
	}
	return listOfSecrets
}

func secretName(site string, pool string) string {
	return fmt.Sprintf("rbd-mirror-token-%s-%s", site, pool)
}

func generateRBDMirroring(cephDpl *cephlcmv1alpha1.CephDeployment, rookNamespace string) *cephv1.CephRBDMirror {
	rbdmirror := &cephv1.CephRBDMirror{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:      cephDpl.Name,
			Namespace: rookNamespace,
		},
		Spec: cephv1.RBDMirroringSpec{
			Count: int(cephDpl.Spec.RBDMirror.Count),
			Peers: cephv1.MirroringPeerSpec{SecretNames: buildRBDMirrorSecretName(cephDpl.Spec.RBDMirror.Peers)},
		},
	}
	return rbdmirror
}

func generateRBDSecret(site string, token string, pool string, namespace string) *v1.Secret {
	secret := v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName(site, pool),
			Namespace: namespace,
		},
		Data: map[string][]byte{
			"pool":  []byte(pool),
			"token": []byte(token),
		},
		Type: "RBDPeer",
	}
	return &secret
}
