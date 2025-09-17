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
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	lcmcommon "github.com/Mirantis/pelagia/pkg/common"
)

func (c *cephDeploymentConfig) addExternalResources(ownerRefs []metav1.OwnerReference) (bool, error) {
	c.log.Debug().Msg("configuring requiring resources for external ceph configuration")
	stringSecret, stringErr := c.api.Kubeclientset.CoreV1().Secrets(c.cdConfig.cephDpl.Namespace).Get(c.context, externalStringSecretName, metav1.GetOptions{})
	if stringErr != nil {
		return false, errors.Wrapf(stringErr, "failed to get secret '%s/%s' with external connection info", c.cdConfig.cephDpl.Namespace, externalStringSecretName)
	}

	var cephCon lcmcommon.CephConnection
	if v, ok := stringSecret.Data["connection"]; ok && len(v) > 0 {
		err := json.Unmarshal(v, &cephCon)
		if err != nil {
			return false, errors.Wrapf(err, "failed to parse external connection string from secret '%s/%s'", c.cdConfig.cephDpl.Namespace, externalStringSecretName)
		}
	} else {
		return false, errors.Errorf("required for connection to external cluster parameters ('connection' field) is not specified in secret '%s/%s'", c.cdConfig.cephDpl.Namespace, externalStringSecretName)
	}

	monEndpointsConfigMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:            rookCephMonEndpointsMapName,
			Namespace:       c.lcmConfig.RookNamespace,
			OwnerReferences: ownerRefs,
		},
		Data: map[string]string{
			"data":     cephCon.MonEndpoints,
			"mapping":  "{}",
			"maxMonId": strconv.Itoa(len(strings.Split(cephCon.MonEndpoints, ","))),
		},
	}
	configMapUpdated, err := c.manageConfigMap(monEndpointsConfigMap)
	if err != nil {
		return false, errors.Wrapf(err, "failed to manage config map for external cluster")
	}

	createCephExternalSecret := func(secretName string, data map[string][]byte) *corev1.Secret {
		return &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:            secretName,
				Namespace:       c.lcmConfig.RookNamespace,
				OwnerReferences: ownerRefs,
			},
			Data: data,
		}
	}

	secretsToManage := []*corev1.Secret{}
	if cephCon.ClientName == "admin" {
		secretsToManage = append(secretsToManage, createCephExternalSecret(lcmcommon.RookCephMonSecretName,
			map[string][]byte{
				"cluster-name":  []byte(c.lcmConfig.RookNamespace),
				"fsid":          []byte(cephCon.FSID),
				"admin-secret":  []byte(cephCon.ClientKeyring),
				"ceph-username": []byte("client.admin"),
				"ceph-secret":   []byte(cephCon.ClientKeyring),
				"mon-secret":    []byte("mon-secret"),
				"ceph-args":     []byte(""),
			}),
		)
	} else {
		secretsToManage = append(secretsToManage, createCephExternalSecret(lcmcommon.RookCephMonSecretName,
			map[string][]byte{
				"cluster-name":  []byte(c.lcmConfig.RookNamespace),
				"fsid":          []byte(cephCon.FSID),
				"admin-secret":  []byte("admin-secret"),
				"ceph-username": []byte(fmt.Sprintf("client.%s", cephCon.ClientName)),
				"ceph-secret":   []byte(cephCon.ClientKeyring),
				"mon-secret":    []byte("mon-secret"),
				"ceph-args":     []byte(fmt.Sprintf("-n client.%s", cephCon.ClientName)),
			}),
		)

		// create csi secrets only if connection string contains non-admin client
		if cephCon.ClientName != "client.admin" {
			secretsToManage = append(secretsToManage,
				createCephExternalSecret(fmt.Sprintf("rook-%s", lcmcommon.CephCSIRBDNodeClientName),
					map[string][]byte{
						"userID":  []byte(lcmcommon.CephCSIRBDNodeClientName),
						"userKey": []byte(cephCon.RBDKeyring.NodeKey),
					}),
				createCephExternalSecret(fmt.Sprintf("rook-%s", lcmcommon.CephCSIRBDProvisionerClientName),
					map[string][]byte{
						"userID":  []byte(lcmcommon.CephCSIRBDProvisionerClientName),
						"userKey": []byte(cephCon.RBDKeyring.ProvisionerKey),
					}),
			)

			if cephCon.CephFSKeyring.NodeKey != "" && cephCon.CephFSKeyring.ProvisionerKey != "" && c.cdConfig.cephDpl.Spec.SharedFilesystem != nil {
				secretsToManage = append(secretsToManage,
					createCephExternalSecret(fmt.Sprintf("rook-%s", lcmcommon.CephCSICephFSNodeClientName),
						map[string][]byte{
							"adminID":  []byte(lcmcommon.CephCSICephFSNodeClientName),
							"adminKey": []byte(cephCon.CephFSKeyring.NodeKey),
						}),
					createCephExternalSecret(fmt.Sprintf("rook-%s", lcmcommon.CephCSICephFSProvisionerClientName),
						map[string][]byte{
							"adminID":  []byte(lcmcommon.CephCSICephFSProvisionerClientName),
							"adminKey": []byte(cephCon.CephFSKeyring.ProvisionerKey),
						}),
				)
			}
		}
	}
	if cephCon.RgwAdminUserKeys != nil && c.cdConfig.cephDpl.Spec.ObjectStorage != nil {
		secretsToManage = append(secretsToManage, createCephExternalSecret(rgwAdminUserSecretName,
			map[string][]byte{
				"accessKey": []byte(cephCon.RgwAdminUserKeys.AccessKey),
				"secretKey": []byte(cephCon.RgwAdminUserKeys.SecretKey),
			}),
		)
	}
	secretsUpdated, err := c.manageSecrets(secretsToManage)
	if err != nil {
		return false, err
	}
	return configMapUpdated || secretsUpdated, nil
}

func (c *cephDeploymentConfig) manageSecrets(secrets []*corev1.Secret) (bool, error) {
	errs := []string{}
	updated := false
	for _, secret := range secrets {
		rookSecret, err := c.api.Kubeclientset.CoreV1().Secrets(secret.Namespace).Get(c.context, secret.Name, metav1.GetOptions{})
		if err != nil {
			if apierrors.IsNotFound(err) {
				c.log.Info().Msgf("create %s/%s secret", secret.Namespace, secret.Name)
				_, err = c.api.Kubeclientset.CoreV1().Secrets(secret.Namespace).Create(c.context, secret, metav1.CreateOptions{})
				if err != nil {
					c.log.Error().Err(err).Msgf("failed to create %s/%s secret", secret.Namespace, secret.Name)
					errs = append(errs, err.Error())
				} else {
					updated = true
				}
				continue
			}
			c.log.Error().Err(err).Msgf("failed to get %s/%s secret", secret.Namespace, secret.Name)
			errs = append(errs, err.Error())
			continue
		}
		secretUpdated := false
		if !reflect.DeepEqual(rookSecret.OwnerReferences, secret.OwnerReferences) {
			lcmcommon.ShowObjectDiff(*c.log, rookSecret.OwnerReferences, secret.OwnerReferences)
			rookSecret.OwnerReferences = secret.OwnerReferences
			secretUpdated = true
		}
		if !reflect.DeepEqual(rookSecret.Data, secret.Data) {
			rookSecret.Data = secret.Data
			secretUpdated = true
		}
		if secretUpdated {
			c.log.Info().Msgf("update %s/%s secret", rookSecret.Namespace, rookSecret.Name)
			_, err := c.api.Kubeclientset.CoreV1().Secrets(rookSecret.Namespace).Update(c.context, rookSecret, metav1.UpdateOptions{})
			if err != nil {
				c.log.Error().Err(err).Msgf("failed to update %s/%s secret", rookSecret.Namespace, rookSecret.Name)
				errs = append(errs, err.Error())
			} else {
				updated = true
			}
		}
	}
	if len(errs) > 0 {
		return false, errors.Errorf("failed to manage secrets for external cluster: %s", strings.Join(errs, ", "))
	}
	return updated, nil
}

func (c *cephDeploymentConfig) manageConfigMap(configMap *corev1.ConfigMap) (bool, error) {
	rookConfigMap, err := c.api.Kubeclientset.CoreV1().ConfigMaps(configMap.Namespace).Get(c.context, configMap.Name, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			c.log.Info().Msgf("create %s/%s config map", configMap.Namespace, configMap.Name)
			_, err = c.api.Kubeclientset.CoreV1().ConfigMaps(configMap.Namespace).Create(c.context, configMap, metav1.CreateOptions{})
			if err != nil {
				return false, errors.Wrapf(err, "failed to create %s/%s config map", configMap.Namespace, configMap.Name)
			}
			return true, nil
		}
		return false, errors.Wrapf(err, "failed to get %s/%s config map", configMap.Namespace, configMap.Name)
	}
	updated := false
	if !reflect.DeepEqual(rookConfigMap.OwnerReferences, configMap.OwnerReferences) {
		lcmcommon.ShowObjectDiff(*c.log, rookConfigMap.OwnerReferences, configMap.OwnerReferences)
		rookConfigMap.OwnerReferences = configMap.OwnerReferences
		updated = true
	}
	if !reflect.DeepEqual(rookConfigMap.Data, configMap.Data) {
		lcmcommon.ShowObjectDiff(*c.log, rookConfigMap.Data, configMap.Data)
		rookConfigMap.Data = configMap.Data
		updated = true
	}
	if updated {
		c.log.Info().Msgf("update %s/%s config map", rookConfigMap.Namespace, rookConfigMap.Name)
		_, err := c.api.Kubeclientset.CoreV1().ConfigMaps(rookConfigMap.Namespace).Update(c.context, rookConfigMap, metav1.UpdateOptions{})
		if err != nil {
			return false, errors.Wrapf(err, "failed to update %s/%s config map", rookConfigMap.Namespace, rookConfigMap.Name)
		}
	}
	return updated, nil
}

func (c *cephDeploymentConfig) deleteExternalConnectionSecret() (bool, error) {
	err := c.api.Kubeclientset.CoreV1().Secrets(c.cdConfig.cephDpl.Namespace).Delete(c.context, externalStringSecretName, metav1.DeleteOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return true, nil
		}
		return false, errors.Wrapf(err, "failed to delete external connection secret %s/%s", c.lcmConfig.RookNamespace, externalStringSecretName)
	}
	c.log.Info().Msgf("removed external connection secret %s/%s", c.lcmConfig.RookNamespace, externalStringSecretName)
	return false, nil
}
