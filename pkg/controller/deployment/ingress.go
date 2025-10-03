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
	v1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	cephlcmv1alpha1 "github.com/Mirantis/pelagia/pkg/apis/ceph.pelagia.lcm/v1alpha1"
	lcmcommon "github.com/Mirantis/pelagia/pkg/common"
)

func buildHostName(rgwName, rgwOverrideName, domain string) string {
	if rgwOverrideName != "" {
		return fmt.Sprintf("%s.%s", rgwOverrideName, domain)
	}
	return fmt.Sprintf("%s.%s", rgwName, domain)
}

func (c *cephDeploymentConfig) ensureIngressProxy() (bool, error) {
	cleanupIngress := false
	if c.cdConfig.cephDpl.Spec.ObjectStorage == nil {
		c.log.Debug().Msg("skipping ingress ensure since rgw is not specified")
		cleanupIngress = true
	} else if !isSpecIngressProxyRequired(c.cdConfig.cephDpl.Spec) {
		c.log.Debug().Msg("skipping ingress ensure since no custom ingress provided and no required OpenStack configuration")
		cleanupIngress = true
	}

	if cleanupIngress {
		c.log.Debug().Msg("ensure ingress proxy stuff is cleaned up")
		removed, err := c.deleteIngressProxy()
		if err != nil {
			return false, errors.Wrap(err, "deletion not complete for ingress proxy")
		}
		return !removed, nil
	}
	c.log.Debug().Msg("ensure Ingress proxy")

	ingressName := buildRGWName(c.cdConfig.cephDpl.Spec.ObjectStorage.Rgw.Name, "ingress")
	ingress, err := c.api.Kubeclientset.NetworkingV1().Ingresses(c.lcmConfig.RookNamespace).Get(c.context, ingressName, metav1.GetOptions{})
	if err != nil {
		if !apierrors.IsNotFound(err) {
			return false, errors.Wrapf(err, "failed to get %s ingress", ingressName)
		}
		ingress = nil
	}
	defaultIngressClass := "openstack-ingress-nginx"
	ingressConfig := c.cdConfig.cephDpl.Spec.IngressConfig
	if ingressConfig == nil {
		ingressConfig = &cephlcmv1alpha1.CephDeploymentIngressConfig{}
	}
	if ingressConfig.ControllerClassName == "" {
		ingressConfig.ControllerClassName = defaultIngressClass
	}
	defaultAnnotations := map[string]string{}
	if ingressConfig.ControllerClassName == defaultIngressClass {
		defaultAnnotations["nginx.ingress.kubernetes.io/proxy-body-size"] = "0"
		defaultAnnotations["nginx.ingress.kubernetes.io/rewrite-target"] = "/"
	}
	if len(ingressConfig.Annotations) > 0 {
		for k, v := range defaultAnnotations {
			if _, ok := ingressConfig.Annotations[k]; ok {
				continue
			}
			ingressConfig.Annotations[k] = v
		}
	} else {
		ingressConfig.Annotations = defaultAnnotations
	}
	// handle case when no certs in ingress spec and no secret by ref, try to get default openstack certs
	if ingressConfig.TLSConfig == nil || (ingressConfig.TLSConfig.TLSCerts == nil && ingressConfig.TLSConfig.TLSSecretRefName == "") {
		if c.lcmConfig.DeployParams.OpenstackCephSharedNamespace == "" {
			c.log.Error().Msgf("ingress certs are not set, openstack-ceph shared namespace with openstack certs is not set, skipping ingress deploy ")
			return false, nil
		}
		osSecret, err := c.api.Kubeclientset.CoreV1().Secrets(c.lcmConfig.DeployParams.OpenstackCephSharedNamespace).Get(c.context, openstackRgwCredsName, metav1.GetOptions{})
		if err != nil {
			if !apierrors.IsNotFound(err) {
				return false, errors.Wrapf(err, "failed to get %s secret to ensure ingress", openstackRgwCredsName)
			}
			osSecret = nil
		}
		if osSecret == nil || osSecret.Data["ca_cert"] == nil || osSecret.Data["tls_crt"] == nil || osSecret.Data["tls_key"] == nil {
			c.log.Error().Msg("ingress certs are not set, openstack certs are not found, skipping ingress deploy")
			return false, nil
		}
		tlsCerts := &cephlcmv1alpha1.CephDeploymentCert{
			Cacert:  string(osSecret.Data["ca_cert"]),
			TLSCert: string(osSecret.Data["tls_crt"]),
			TLSKey:  string(osSecret.Data["tls_key"]),
		}
		// since it is possible to specify publicDomain/hostname without certs - handle such case
		// publicDomain is mandatory - so it can be passed within spec
		if ingressConfig.TLSConfig == nil {
			ingressConfig.TLSConfig = &cephlcmv1alpha1.CephDeploymentIngressTLSConfig{
				Domain:   string(osSecret.Data["public_domain"]),
				TLSCerts: tlsCerts,
			}
		} else {
			ingressConfig.TLSConfig.TLSCerts = tlsCerts
		}
	}
	// always set upstream vhost annotation to actual public domain for ingress and do not allow to override it
	if ingressConfig.ControllerClassName == defaultIngressClass {
		ingressConfig.Annotations["nginx.ingress.kubernetes.io/upstream-vhost"] = buildHostName(c.cdConfig.cephDpl.Spec.ObjectStorage.Rgw.Name, ingressConfig.TLSConfig.Hostname, ingressConfig.TLSConfig.Domain)
	}

	var tlsSecretName string
	var tlsSecretResource *v1.Secret
	if ingressConfig.TLSConfig.TLSSecretRefName != "" {
		tlsSecretName = ingressConfig.TLSConfig.TLSSecretRefName
		tlsSecret, err := c.api.Kubeclientset.CoreV1().Secrets(c.lcmConfig.RookNamespace).Get(c.context, tlsSecretName, metav1.GetOptions{})
		if err != nil {
			return false, errors.Wrapf(err, "failed to get specified ingress tls certs secret '%s/%s'", c.lcmConfig.RookNamespace, tlsSecretName)
		}
		tlsSecretResource = tlsSecret
	} else {
		if ingressConfig.TLSConfig.Hostname != "" {
			tlsSecretName = buildRGWName(ingressConfig.TLSConfig.Hostname, fmt.Sprintf("tls-public-%v", lcmcommon.GetCurrentUnixTimeString()))
		} else {
			tlsSecretName = buildRGWName(c.cdConfig.cephDpl.Spec.ObjectStorage.Rgw.Name, fmt.Sprintf("tls-public-%v", lcmcommon.GetCurrentUnixTimeString()))
		}
		tlsSecretResource = &v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      tlsSecretName,
				Namespace: c.lcmConfig.RookNamespace,
				Labels: map[string]string{
					"objectStore":    c.cdConfig.cephDpl.Spec.ObjectStorage.Rgw.Name,
					"ingress":        ingressName,
					cephIngressLabel: "ceph-object-store-ingress",
				},
			},
			Data: map[string][]byte{
				"ca.crt":  []byte(ingressConfig.TLSConfig.TLSCerts.Cacert),
				"tls.crt": []byte(ingressConfig.TLSConfig.TLSCerts.TLSCert),
				"tls.key": []byte(ingressConfig.TLSConfig.TLSCerts.TLSKey),
			},
		}
	}
	changedIngressConfig := false
	createNewSecret := true
	tlsSecretToDelete := ""
	if ingress != nil {
		hostName := buildHostName(c.cdConfig.cephDpl.Spec.ObjectStorage.Rgw.Name, ingressConfig.TLSConfig.Hostname, ingressConfig.TLSConfig.Domain)
		for _, tls := range ingress.Spec.TLS {
			if lcmcommon.Contains(tls.Hosts, hostName) {
				if ingressConfig.TLSConfig.TLSSecretRefName != "" {
					if tls.SecretName != ingressConfig.TLSConfig.TLSSecretRefName {
						tlsSecretToDelete = tls.SecretName
					}
					break
				}
				tlsSecret, err := c.api.Kubeclientset.CoreV1().Secrets(c.lcmConfig.RookNamespace).Get(c.context, tls.SecretName, metav1.GetOptions{})
				if err != nil {
					if apierrors.IsNotFound(err) {
						c.log.Error().Err(err).Msgf("not found ingress cert secret '%s/%s'", c.lcmConfig.RookNamespace, tls.SecretName)
						break
					}
					return false, errors.Wrapf(err, "failed to get ingress cert secret '%s/%s'", c.lcmConfig.RookNamespace, tls.SecretName)
				}
				if reflect.DeepEqual(tlsSecret.Data, tlsSecretResource.Data) {
					tlsSecretName = tlsSecret.Name
					createNewSecret = false
					if tlsSecret.Labels == nil {
						tlsSecret.Labels = make(map[string]string)
					}
					updateLabels := false
					for key, value := range tlsSecretResource.Labels {
						if tlsSecret.Labels[key] != value {
							tlsSecret.Labels[key] = value
							updateLabels = true
							c.log.Info().Msgf("setting label '%s=%s' for secret %s/%s", key, value, c.lcmConfig.RookNamespace, tlsSecret.Name)
						}
					}
					if updateLabels {
						_, err = c.api.Kubeclientset.CoreV1().Secrets(c.lcmConfig.RookNamespace).Update(c.context, tlsSecret, metav1.UpdateOptions{})
						if err != nil {
							return false, errors.Wrapf(err, "failed to update labels for secret %s/%s", c.lcmConfig.RookNamespace, tlsSecret.Name)
						}
						changedIngressConfig = true
					}
				} else {
					tlsSecretToDelete = tls.SecretName
				}
				break
			}
			// if no matched hostName in current tls.Hosts - fqdn changed, drop previous cert
			// and use secret name with hostname naming
			tlsSecretToDelete = tls.SecretName
		}
	}
	if createNewSecret && ingressConfig.TLSConfig.TLSSecretRefName == "" {
		c.log.Info().Msgf("creating %s secret for ingress %s/%s", tlsSecretName, c.lcmConfig.RookNamespace, ingressName)
		_, err = c.api.Kubeclientset.CoreV1().Secrets(c.lcmConfig.RookNamespace).Create(c.context, tlsSecretResource, metav1.CreateOptions{})
		if err != nil {
			return false, errors.Wrapf(err, "failed to create ingress cert secret %s", tlsSecretName)
		}
		changedIngressConfig = true
	}
	externalAccessLabel, err := metav1.ParseToLabelSelector(c.lcmConfig.DeployParams.RgwPublicAccessLabel)
	if err != nil {
		return false, errors.Wrapf(err, "failed to parse provided ingress public access label '%s'", c.lcmConfig.DeployParams.RgwPublicAccessLabel)
	}
	ingressResource := generateIngress(c.cdConfig.cephDpl.Spec.ObjectStorage.Rgw.Name, c.lcmConfig.RookNamespace, tlsSecretName, ingressConfig, externalAccessLabel)
	if ingress == nil {
		c.log.Info().Msgf("creating ingress %s/%s", c.lcmConfig.RookNamespace, ingressName)
		_, err = c.api.Kubeclientset.NetworkingV1().Ingresses(c.lcmConfig.RookNamespace).Create(c.context, ingressResource, metav1.CreateOptions{})
		if err != nil {
			return false, errors.Wrapf(err, "failed to create ingress %s/%s", c.lcmConfig.RookNamespace, ingressName)
		}
		return true, nil
	}
	update := false
	if ingress.Labels == nil {
		update = true
		ingress.Labels = ingressResource.Labels
	} else {
		for k, v := range ingressResource.Labels {
			if curValue, ok := ingress.Labels[k]; !ok || curValue != v {
				c.log.Info().Msgf("setting label '%s=%s' for ingress %s/%s", k, v, ingress.Namespace, ingress.Name)
				ingress.Labels[k] = v
				update = true
			}
		}
	}
	if !reflect.DeepEqual(ingress.Annotations, ingressResource.Annotations) {
		if ingress.Annotations == nil {
			for k, v := range ingressResource.Annotations {
				c.log.Info().Msgf("setting ingress annotation %s/%s: '%s=%v'", ingress.Namespace, ingress.Name, k, v)
			}
			ingress.Annotations = ingressResource.Annotations
		} else {
			for k, v := range ingressResource.Annotations {
				if curValue, ok := ingress.Annotations[k]; !ok || curValue != v {
					c.log.Info().Msgf("setting ingress annotation %s/%s: '%s=%v'", ingress.Namespace, ingress.Name, k, v)
					ingress.Annotations[k] = v
				}
			}
			for k, v := range ingress.Annotations {
				if _, ok := ingressResource.Annotations[k]; !ok {
					c.log.Info().Msgf("removing ingress annotation %s/%s: '%s=%v'", ingress.Namespace, ingress.Name, k, v)
					delete(ingress.Annotations, k)
				}
			}
		}
		update = true
	}
	if !reflect.DeepEqual(ingress.Spec, ingressResource.Spec) {
		c.log.Info().Msgf("updating ingress configuration %s/%s", ingress.Namespace, ingress.Name)
		lcmcommon.ShowObjectDiff(*c.log, ingress, ingressResource)
		ingress.Spec = ingressResource.Spec
		update = true
	}
	if update {
		changedIngressConfig = true
		_, err = c.api.Kubeclientset.NetworkingV1().Ingresses(c.lcmConfig.RookNamespace).Update(c.context, ingress, metav1.UpdateOptions{})
		if err != nil {
			return false, errors.Wrapf(err, "failed to update ingress %s/%s", ingress.Namespace, ingress.Name)
		}
		if tlsSecretToDelete != "" {
			c.log.Info().Msgf("removing previous ingress secret '%s/%s'", c.lcmConfig.RookNamespace, tlsSecretToDelete)
			err := c.api.Kubeclientset.CoreV1().Secrets(c.lcmConfig.RookNamespace).Delete(c.context, tlsSecretToDelete, metav1.DeleteOptions{})
			if err != nil {
				return false, errors.Wrapf(err, "failed to delete old ingress cert secret %s", tlsSecretToDelete)
			}
		}
	}
	return changedIngressConfig, nil
}

func (c *cephDeploymentConfig) deleteIngressProxy() (bool, error) {
	ingressResourcesRemoved := true
	listOptions := metav1.ListOptions{
		LabelSelector: cephIngressLabel,
	}
	ingresses, err := c.api.Kubeclientset.NetworkingV1().Ingresses(c.lcmConfig.RookNamespace).List(c.context, listOptions)
	if err != nil {
		return false, errors.Wrap(err, "failed to find ingress to remove")
	}
	if len(ingresses.Items) != 0 {
		for _, ingress := range ingresses.Items {
			err = c.api.Kubeclientset.NetworkingV1().Ingresses(c.lcmConfig.RookNamespace).Delete(c.context, ingress.Name, metav1.DeleteOptions{})
			if err != nil && !apierrors.IsNotFound(err) {
				return false, errors.Wrapf(err, "failed to delete ingress %s", ingress.Name)
			}
			c.log.Info().Msgf("removed ingress '%s/%s'", ingress.Namespace, ingress.Name)
			ingressResourcesRemoved = false
		}
	}
	secrets, err := c.api.Kubeclientset.CoreV1().Secrets(c.lcmConfig.RookNamespace).List(c.context, listOptions)
	if err != nil {
		return false, errors.Wrap(err, "failed to get list ingress secrets to remove")
	}
	if len(secrets.Items) > 0 {
		err := c.api.Kubeclientset.CoreV1().Secrets(c.lcmConfig.RookNamespace).DeleteCollection(c.context, metav1.DeleteOptions{}, listOptions)
		if err != nil {
			if !apierrors.IsNotFound(err) {
				return false, errors.Wrap(err, "failed to delete tls secrets for ingress")
			}
		}
		c.log.Info().Msg("removed tls secrets for CephDeployment based ingress")
		ingressResourcesRemoved = false
	}
	return ingressResourcesRemoved, nil
}

func generateIngress(rgwName, namespace, tlsSecretName string, ingressConfig *cephlcmv1alpha1.CephDeploymentIngressConfig, externalAccessLabel *metav1.LabelSelector) *networkingv1.Ingress {
	pathType := networkingv1.PathTypeImplementationSpecific
	ingressTypeLabel := fmt.Sprintf("%s-rgw", ingressConfig.ControllerClassName)
	annotations := ingressConfig.Annotations
	annotations["kubernetes.io/ingress.class"] = ingressConfig.ControllerClassName
	hostName := buildHostName(rgwName, ingressConfig.TLSConfig.Hostname, ingressConfig.TLSConfig.Domain)
	ingressLabels := map[string]string{
		"ingress-type":      ingressTypeLabel,
		cephIngressLabel:    "ceph-object-store-ingress",
		"app":               "rook-ceph-rgw",
		"rook_object_store": rgwName,
	}
	for key, val := range externalAccessLabel.MatchLabels {
		ingressLabels[key] = val
	}
	return &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:        buildRGWName(rgwName, "ingress"),
			Namespace:   namespace,
			Annotations: annotations,
			Labels:      ingressLabels,
		},
		Spec: networkingv1.IngressSpec{
			Rules: []networkingv1.IngressRule{
				{
					Host: hostName,
					IngressRuleValue: networkingv1.IngressRuleValue{
						HTTP: &networkingv1.HTTPIngressRuleValue{
							Paths: []networkingv1.HTTPIngressPath{
								{
									Backend: networkingv1.IngressBackend{
										Service: &networkingv1.IngressServiceBackend{
											Name: buildRGWName(rgwName, ""),
											Port: networkingv1.ServiceBackendPort{
												Name: "http",
											},
										},
									},
									Path:     "/",
									PathType: &pathType,
								},
							},
						},
					},
				},
			},
			TLS: []networkingv1.IngressTLS{
				{
					Hosts:      []string{hostName},
					SecretName: tlsSecretName,
				},
			},
		},
	}
}

func getIngressTLS(cephDpl *cephlcmv1alpha1.CephDeployment) *cephlcmv1alpha1.CephDeploymentIngressTLSConfig {
	ingressConfig := cephDpl.Spec.IngressConfig
	if ingressConfig != nil {
		if ingressConfig.TLSConfig != nil {
			return ingressConfig.TLSConfig
		}
	}
	return nil
}
