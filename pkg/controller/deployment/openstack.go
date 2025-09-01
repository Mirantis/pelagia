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
	"regexp"
	"sort"
	"strings"

	"github.com/pkg/errors"
	rookUtils "github.com/rook/rook/pkg/operator/k8sutil"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	cephlcmv1alpha1 "github.com/Mirantis/pelagia/pkg/apis/ceph.pelagia.lcm/v1alpha1"
	lcmcommon "github.com/Mirantis/pelagia/pkg/common"
)

type openstackSecretData struct {
	clientKeys       map[string]string
	monMap           *v1.ConfigMap
	adminSecret      *v1.Secret
	rgwSecret        *v1.Secret
	rgwInternalCert  *v1.Secret
	rgwMetricsSecret *v1.Secret
}

func (c *cephDeploymentConfig) ensureOpenstackSecret() (bool, error) {
	if c.cdConfig.cephDpl.Spec.ExtraOpts != nil && c.cdConfig.cephDpl.Spec.ExtraOpts.DisableOsKeys {
		c.log.Info().Msg("openstack secret ensure disabled, skipping it")
		return false, nil
	}
	// Skip OpenStack secret ensure if there is no OpenStack pools
	if !lcmcommon.IsOpenStackPoolsPresent(c.cdConfig.cephDpl.Spec.Pools) {
		c.log.Info().Msg("required Openstack pools are not specified in spec, skipping Openstack secret ensure")
		return false, nil
	}
	c.log.Info().Msgf("ensure Openstack secret %s/%s", c.lcmConfig.DeployParams.OpenstackCephSharedNamespace, openstackSharedSecret)
	cephFSDeployed := c.cdConfig.cephDpl.Spec.SharedFilesystem != nil

	notReadyOpenStackPools := []string{}
	for _, pool := range c.cdConfig.cephDpl.Spec.Pools {
		switch pool.Role {
		case "images", "vms", "backup", "volumes":
			poolName := buildPoolName(pool)
			if !isCephPoolReady(c.context, *c.log, c.api.Rookclientset, c.lcmConfig.RookNamespace, poolName) {
				notReadyOpenStackPools = append(notReadyOpenStackPools, poolName)
			}
		}
	}
	if len(notReadyOpenStackPools) > 0 {
		return false, errors.Errorf("skip openstack secret ensure since the following required OpenStack pools are not ready yet: %v", notReadyOpenStackPools)
	}
	if !c.openstackClientsFound(cephFSDeployed) {
		return false, errors.New("skip openstack secret ensure: no required ceph clients")
	}
	clientKeys, err := c.getCephClientAuthKeys(cephFSDeployed)
	if err != nil {
		return false, errors.Wrap(err, "failed to get auth keys for ceph clients")
	}

	// Get required values for OpenStack secret
	monMap, err := c.getMonMapConfigmap()
	if err != nil {
		return false, errors.Wrap(err, "failed to get ceph monitor endpoints")
	}
	adminSecret, err := c.getAdminSecret()
	if err != nil {
		return false, errors.Wrap(err, "failed to get ceph admin secret")
	}
	openstackSecretData := openstackSecretData{
		clientKeys:  clientKeys,
		monMap:      monMap,
		adminSecret: adminSecret,
	}

	if c.cdConfig.cephDpl.Spec.ObjectStorage != nil {
		rgwSecret, err := c.api.Kubeclientset.CoreV1().Secrets(c.lcmConfig.DeployParams.OpenstackCephSharedNamespace).Get(c.context, openstackRgwCredsName, metav1.GetOptions{})
		if err != nil {
			c.log.Error().Err(err).Msgf("failed to get rgw secret for %s secret update", openstackSharedSecret)
		} else {
			openstackSecretData.rgwSecret = rgwSecret
		}
		rgwInternalCert, err := c.api.Kubeclientset.CoreV1().Secrets(c.lcmConfig.RookNamespace).Get(c.context, rgwSslCertSecretName, metav1.GetOptions{})
		if err != nil {
			c.log.Error().Err(err).Msgf("failed to get rgw local certs secret for %s secret update", openstackSharedSecret)
		} else {
			openstackSecretData.rgwInternalCert = rgwInternalCert
		}
		rgwMetricsUserSecret, err := c.getRgwMetricsUserSecrets()
		if err != nil {
			c.log.Error().Err(err).Msgf("failed to get rgw metrics user secrets for %s secret update", openstackSharedSecret)
		} else {
			openstackSecretData.rgwMetricsSecret = rgwMetricsUserSecret
		}
	}

	// Build OpenStack secret with parameters defined above
	osSecret := c.generateOpenstackSecret(openstackSecretData)

	// Create/Update/Skip OpenStack secret with new spec
	currentSecret, err := c.api.Kubeclientset.CoreV1().Secrets(c.lcmConfig.DeployParams.OpenstackCephSharedNamespace).Get(c.context, openstackSharedSecret, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			c.log.Info().Msgf("create %s/%s secret", c.lcmConfig.DeployParams.OpenstackCephSharedNamespace, openstackSharedSecret)
			_, err := c.api.Kubeclientset.CoreV1().Secrets(c.lcmConfig.DeployParams.OpenstackCephSharedNamespace).Create(c.context, osSecret, metav1.CreateOptions{})
			if err != nil {
				c.log.Error().Err(err).Msg("")
				return false, errors.Wrapf(err, "failed to create %s/%s secret", c.lcmConfig.DeployParams.OpenstackCephSharedNamespace, openstackSharedSecret)
			}
			return true, nil
		}
		c.log.Error().Err(err).Msg("")
		return false, errors.Wrapf(err, "failed to get %s/%s secret", c.lcmConfig.DeployParams.OpenstackCephSharedNamespace, openstackSharedSecret)
	}
	if !reflect.DeepEqual(currentSecret.Data, osSecret.Data) {
		c.log.Info().Msgf("update %s/%s secret", c.lcmConfig.DeployParams.OpenstackCephSharedNamespace, openstackSharedSecret)
		currentSecret.Data = osSecret.Data
		_, err := c.api.Kubeclientset.CoreV1().Secrets(c.lcmConfig.DeployParams.OpenstackCephSharedNamespace).Update(c.context, currentSecret, metav1.UpdateOptions{})
		if err != nil {
			c.log.Error().Err(err).Msg("")
			return false, errors.Wrapf(err, "failed to update %s/%s secret", c.lcmConfig.DeployParams.OpenstackCephSharedNamespace, openstackSharedSecret)
		}
		return true, nil
	}
	return false, nil
}

func (c *cephDeploymentConfig) deleteOpenstackSecret() (bool, error) {
	err := c.api.Kubeclientset.CoreV1().Secrets(c.lcmConfig.DeployParams.OpenstackCephSharedNamespace).Delete(c.context, openstackSharedSecret, metav1.DeleteOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return true, nil
		}
		return false, errors.Wrapf(err, "failed to delete openstack secret %s/%s", c.lcmConfig.DeployParams.OpenstackCephSharedNamespace, openstackSharedSecret)
	}
	c.log.Info().Msgf("removed openstack secret %s/%s", c.lcmConfig.DeployParams.OpenstackCephSharedNamespace, openstackSharedSecret)
	return false, nil
}

func (c *cephDeploymentConfig) getCephClientAuthKeys(cephFSDeployed bool) (map[string]string, error) {
	authKeys := map[string]string{
		"nova":   "",
		"cinder": "",
		"glance": "",
	}
	if cephFSDeployed {
		authKeys["manila"] = ""
	}
	errs := []string{}
	for client := range authKeys {
		cmd := fmt.Sprintf("ceph auth get-key client.%s", client)
		key, err := lcmcommon.RunCephToolboxCLI(c.context, c.api.Kubeclientset, c.api.Config, c.lcmConfig.RookNamespace, cmd)
		if err != nil {
			c.log.Error().Err(err).Msgf("command '%s' failed", cmd)
			errs = append(errs, fmt.Sprintf("failed to run '%s' command", cmd))
		} else if key == "" {
			errs = append(errs, fmt.Sprintf("command '%s' output is empty", cmd))
		} else {
			authKeys[client] = key
		}
	}
	if len(errs) > 0 {
		sort.Strings(errs)
		return nil, errors.Errorf("some auth keys failed to get: %s", strings.Join(errs, ", "))
	}
	return authKeys, nil
}

func (c *cephDeploymentConfig) openstackClientsFound(cephFSDeployed bool) bool {
	result := true
	osClients := []string{"cinder", "glance", "nova"}
	if cephFSDeployed {
		osClients = append(osClients, "manila")
	}
	for _, name := range osClients {
		_, err := c.api.Rookclientset.CephV1().CephClients(c.lcmConfig.RookNamespace).Get(c.context, name, metav1.GetOptions{})
		if err != nil {
			result = false
			c.log.Error().Err(err).Msgf("can not check ceph client %q", name)
			continue
		}
	}
	return result
}

func (c *cephDeploymentConfig) getMonMapConfigmap() (*v1.ConfigMap, error) {
	monMap, err := c.api.Kubeclientset.CoreV1().ConfigMaps(c.lcmConfig.RookNamespace).Get(c.context, lcmcommon.MonMapConfigMapName, metav1.GetOptions{})
	if err != nil {
		c.log.Error().Err(err).Msg("")
		return nil, errors.Wrapf(err, "failed to get %s/%s configmap", c.lcmConfig.RookNamespace, lcmcommon.MonMapConfigMapName)
	}
	return monMap, nil
}

func (c *cephDeploymentConfig) getAdminSecret() (*v1.Secret, error) {
	adminSecret, err := c.api.Kubeclientset.CoreV1().Secrets(c.lcmConfig.RookNamespace).Get(c.context, adminSecretName, metav1.GetOptions{})
	if err != nil {
		c.log.Error().Err(err).Msg("")
		return nil, errors.Wrapf(err, "failed to get %s/%s admin secret", c.lcmConfig.RookNamespace, adminSecretName)
	}
	return adminSecret, nil
}

func (c *cephDeploymentConfig) getRgwExternalEndpoint(cephDplRGW cephlcmv1alpha1.CephRGW) string {
	if cephDplRGW.Gateway.ExternalRgwEndpoint != nil {
		endpoint := "%s://%s:%d"
		address := ""
		if cephDplRGW.Gateway.ExternalRgwEndpoint.Hostname != "" {
			address = cephDplRGW.Gateway.ExternalRgwEndpoint.Hostname
		} else {
			address = cephDplRGW.Gateway.ExternalRgwEndpoint.IP
		}
		if cephDplRGW.Gateway.SecurePort != 0 {
			endpoint = fmt.Sprintf(endpoint, "https", address, cephDplRGW.Gateway.SecurePort)
		} else {
			endpoint = fmt.Sprintf(endpoint, "http", address, cephDplRGW.Gateway.Port)
		}
		return endpoint
	}
	c.log.Error().Msg("Unable to find external RadosGW endpoint for Keystone public interface")
	return ""
}

func (c *cephDeploymentConfig) getRgwMetricsUserSecrets() (*v1.Secret, error) {
	rgwUser, err := c.api.Rookclientset.CephV1().CephObjectStoreUsers(c.lcmConfig.RookNamespace).Get(c.context, rgwMetricsUser, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	if rgwUser.Status == nil || rgwUser.Status.Phase != rookUtils.ReadyStatus {
		return nil, fmt.Errorf("rgw metrics user %s is not ready", rgwMetricsUser)
	}
	secretName, present := rgwUser.Status.Info["secretName"]
	if !present || secretName == "" {
		return nil, fmt.Errorf("rgw metrics user %s secret is not ready yet", rgwMetricsUser)
	}
	rgwSecret, err := c.api.Kubeclientset.CoreV1().Secrets(c.lcmConfig.RookNamespace).Get(c.context, secretName, metav1.GetOptions{})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get secret '%s/%s' for user %s", c.lcmConfig.RookNamespace, secretName, rgwMetricsUser)
	}
	return rgwSecret, nil
}

func (c *cephDeploymentConfig) generateOpenstackSecret(secretData openstackSecretData) *v1.Secret {
	monmapData := secretData.monMap.Data
	reg, _ := regexp.Compile("([a-z]=)")
	monmapString := reg.ReplaceAllString(monmapData["data"], "")
	monIPs := strings.Split(monmapString, ",")
	sort.Strings(monIPs)
	monmapString = strings.Join(monIPs, ",")

	buildPoolDescription := func(pool cephlcmv1alpha1.CephPool) string {
		return fmt.Sprintf("%s:%s:%s", buildPoolName(pool), pool.Role, pool.DeviceClass)
	}

	glance := "client.glance;" + secretData.clientKeys["glance"] + "\n"
	nova := "client.nova;" + secretData.clientKeys["nova"] + "\n"
	cinder := "client.cinder;" + secretData.clientKeys["cinder"] + "\n"
	for _, pool := range c.cdConfig.cephDpl.Spec.Pools {
		switch role := pool.Role; role {
		case "volumes", "volumes-backend":
			// set basic volumes role
			if pool.Role == "volumes-backend" {
				pool.Role = "volumes"
			}
			nova = nova + ";" + buildPoolDescription(pool)
			cinder = cinder + ";" + buildPoolDescription(pool)
		case "vms":
			nova = nova + ";" + buildPoolDescription(pool)
		case "images":
			nova = nova + ";" + buildPoolDescription(pool)
			glance = glance + ";" + buildPoolDescription(pool)
			cinder = cinder + ";" + buildPoolDescription(pool)
		case "backup":
			cinder = cinder + ";" + buildPoolDescription(pool)
		}
	}

	var clientAdminSecret []byte
	if c.cdConfig.cephDpl.Spec.External {
		// external cluster is connected with admin key
		clientAdminSecret = secretData.adminSecret.Data["admin-secret"]
	} else {
		clientAdminSecret = secretData.adminSecret.Data["ceph-secret"]
	}
	secret := v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      openstackSharedSecret,
			Namespace: c.lcmConfig.DeployParams.OpenstackCephSharedNamespace,
		},
		Data: map[string][]byte{
			"client.admin":  clientAdminSecret,
			"glance":        []byte(glance),
			"nova":          []byte(nova),
			"cinder":        []byte(cinder),
			"mon_endpoints": []byte(monmapString),
		},
	}

	// if manila key is here, add it to openstack-ceph-keys
	if _, ok := secretData.clientKeys["manila"]; ok {
		secret.Data["manila"] = []byte("client.manila;" + secretData.clientKeys["manila"] + "\n")
	}

	if c.cdConfig.cephDpl.Spec.ObjectStorage != nil {
		if !c.cdConfig.cephDpl.Spec.External {
			fqdn := fmt.Sprintf("%s.%s.svc", buildRGWName(c.cdConfig.cephDpl.Spec.ObjectStorage.Rgw.Name, ""), c.lcmConfig.RookNamespace)
			if c.cdConfig.cephDpl.Spec.ObjectStorage.Rgw.Gateway.SecurePort != int32(0) {
				secret.Data["rgw_internal"] = []byte(fmt.Sprintf("https://%s:%d/", fqdn, c.cdConfig.cephDpl.Spec.ObjectStorage.Rgw.Gateway.SecurePort))
			} else {
				secret.Data["rgw_internal"] = []byte(fmt.Sprintf("http://%s:%d/", fqdn, c.cdConfig.cephDpl.Spec.ObjectStorage.Rgw.Gateway.Port))
			}
			if secretData.rgwInternalCert != nil {
				secret.Data["rgw_internal_cacert"] = secretData.rgwInternalCert.Data["cacert"]
			}
		}

		ingressTLS := getIngressTLS(c.cdConfig.cephDpl)
		if c.cdConfig.cephDpl.Spec.External {
			rgwExternal := c.getRgwExternalEndpoint(c.cdConfig.cephDpl.Spec.ObjectStorage.Rgw)
			if rgwExternal != "" {
				secret.Data["rgw_external"] = []byte(rgwExternal)
				// provide bundle set in rgw-ssl-certificate if present as public cacert
				if secretData.rgwInternalCert != nil {
					if bundle, ok := secretData.rgwInternalCert.Data["cabundle"]; ok {
						secret.Data["rgw_external_custom_cacert"] = bundle
					}
				}
			}
		} else if ingressTLS != nil {
			domain := ingressTLS.Domain
			protocol := "https"
			if ingressTLS.Hostname != "" {
				secret.Data["rgw_external"] = []byte(fmt.Sprintf("%s://%s.%s/", protocol, ingressTLS.Hostname, domain))
			} else {
				secret.Data["rgw_external"] = []byte(fmt.Sprintf("%s://%s.%s/", protocol, c.cdConfig.cephDpl.Spec.ObjectStorage.Rgw.Name, domain))
			}
			if ingressTLS.TLSCerts != nil {
				secret.Data["rgw_external_custom_cacert"] = []byte(ingressTLS.TLSCerts.Cacert)
			} else if ingressTLS.TLSSecretRefName != "" {
				tlsSecret, err := c.api.Kubeclientset.CoreV1().Secrets(c.lcmConfig.RookNamespace).Get(c.context, ingressTLS.TLSSecretRefName, metav1.GetOptions{})
				if err != nil {
					c.log.Error().Err(err).Msgf("failed to get specified for ingress tls certs secret %q", ingressTLS.TLSSecretRefName)
				} else if cacert, present := tlsSecret.Data["ca.crt"]; present {
					secret.Data["rgw_external_custom_cacert"] = cacert
				} else {
					c.log.Error().Msgf("specified for ingress tls certs secret %q doesnt contain ca cert", ingressTLS.TLSSecretRefName)
				}
			}
		} else if secretData.rgwSecret != nil {
			domain := secretData.rgwSecret.Data["public_domain"]
			protocol := "http"
			if secretData.rgwSecret.Data["tls_crt"] != nil {
				protocol = "https"
			}
			if string(domain) != "" {
				secret.Data["rgw_external"] = []byte(fmt.Sprintf("%s://%s.%s/", protocol, c.cdConfig.cephDpl.Spec.ObjectStorage.Rgw.Name, string(domain)))
			}
		}

		if secretData.rgwMetricsSecret != nil {
			secret.Data["rgw_metrics_user_access_key"] = secretData.rgwMetricsSecret.Data["AccessKey"]
			secret.Data["rgw_metrics_user_secret_key"] = secretData.rgwMetricsSecret.Data["SecretKey"]
		}
	}

	return &secret
}
