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

package secret

import (
	"fmt"

	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	cephlcmv1alpha1 "github.com/Mirantis/pelagia/pkg/apis/ceph.pelagia.lcm/v1alpha1"
)

const (
	cephAdminKeyringSecret = "rook-ceph-admin-keyring"
)

func (c *cephDeploymentSecretConfig) getSecretsStatusInfo() (*cephlcmv1alpha1.CephDeploymentSecretsInfo, []string) {
	c.log.Debug().Msgf("verifying Ceph secrets updated for cluster %s/%s", c.secretsConfig.cephDpl.Namespace, c.secretsConfig.cephDpl.Name)
	secretsInfo := cephlcmv1alpha1.CephDeploymentSecretsInfo{}
	infoIssues := []string{}

	clientSecrets, issues := c.getClientSecrets(c.secretsConfig.cephDpl.Spec.External)
	if len(clientSecrets) > 0 {
		secretsInfo.ClientSecrets = clientSecrets
	}
	if len(issues) > 0 {
		infoIssues = append(infoIssues, issues...)
	}

	if c.secretsConfig.cephDpl.Spec.ObjectStorage != nil {
		rgwUsers, issues := c.getRgwUserSecrets()
		if len(issues) > 0 {
			infoIssues = append(infoIssues, issues...)
		}
		if len(rgwUsers) > 0 {
			secretsInfo.RgwUserSecrets = rgwUsers
		}
	}
	return &secretsInfo, infoIssues
}

func (c *cephDeploymentSecretConfig) getClientSecrets(isExternal bool) ([]cephlcmv1alpha1.CephDeploymentSecretInfo, []string) {
	clientSecrets := []cephlcmv1alpha1.CephDeploymentSecretInfo{}
	issues := []string{}
	if !isExternal {
		_, err := c.api.Kubeclientset.CoreV1().Secrets(c.lcmConfig.RookNamespace).Get(c.context, cephAdminKeyringSecret, metav1.GetOptions{})
		if err != nil {
			c.log.Err(err).Msg("")
			issues = append(issues, errors.Wrap(err, "admin keyring secret is not available").Error())
		} else {
			clientSecrets = append(clientSecrets, cephlcmv1alpha1.CephDeploymentSecretInfo{
				ObjectName:      "client.admin",
				SecretName:      cephAdminKeyringSecret,
				SecretNamespace: c.lcmConfig.RookNamespace,
			})
		}
	}

	cephClients, err := c.api.Rookclientset.CephV1().CephClients(c.lcmConfig.RookNamespace).List(c.context, metav1.ListOptions{})
	if err != nil {
		c.log.Err(err).Msg("")
		issues = append(issues, errors.Wrap(err, "failed to list ceph clients").Error())
	} else {
		for _, client := range cephClients.Items {
			if client.Status != nil && client.Status.Info["secretName"] != "" {
				clientSecrets = append(clientSecrets, cephlcmv1alpha1.CephDeploymentSecretInfo{
					ObjectName:      fmt.Sprintf("client.%s", client.Name),
					SecretName:      client.Status.Info["secretName"],
					SecretNamespace: c.lcmConfig.RookNamespace,
				})
			} else {
				issues = append(issues, fmt.Sprintf("client %s secret is not ready", client.Name))
			}
		}
	}

	return clientSecrets, issues
}

func (c *cephDeploymentSecretConfig) getRgwUserSecrets() ([]cephlcmv1alpha1.CephDeploymentSecretInfo, []string) {
	issues := []string{}
	rgwUserSecrets := []cephlcmv1alpha1.CephDeploymentSecretInfo{}
	rgwUsers, err := c.api.Rookclientset.CephV1().CephObjectStoreUsers(c.lcmConfig.RookNamespace).List(c.context, metav1.ListOptions{})
	if err != nil {
		c.log.Err(err).Msg("")
		issues = append(issues, errors.Wrap(err, "failed to list ceph clients").Error())
	} else {
		for _, rgwUser := range rgwUsers.Items {
			if rgwUser.Status != nil && rgwUser.Status.Info["secretName"] != "" {
				rgwUserSecrets = append(rgwUserSecrets, cephlcmv1alpha1.CephDeploymentSecretInfo{
					ObjectName:      rgwUser.Name,
					SecretName:      rgwUser.Status.Info["secretName"],
					SecretNamespace: c.lcmConfig.RookNamespace,
				})
			} else {
				issues = append(issues, fmt.Sprintf("rgw user %s secret is not ready", rgwUser.Name))
			}
		}
	}
	return rgwUserSecrets, issues
}
