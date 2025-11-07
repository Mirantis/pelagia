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
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/runtime"

	cephlcmv1alpha1 "github.com/Mirantis/pelagia/pkg/apis/ceph.pelagia.lcm/v1alpha1"
	lcmcommon "github.com/Mirantis/pelagia/pkg/common"
	faketestclients "github.com/Mirantis/pelagia/test/unit/clients"
	unitinputs "github.com/Mirantis/pelagia/test/unit/inputs"
)

func TestDeleteIngressProxy(t *testing.T) {
	tests := []struct {
		name           string
		cephDpl        *cephlcmv1alpha1.CephDeployment
		inputResources map[string]runtime.Object
		apiErrors      map[string]error
		deleted        bool
		expectedError  error
	}{
		{
			name:    "delete ingress - no rgw spec, no ingress found, skipped",
			cephDpl: &unitinputs.BaseCephDeployment,
			inputResources: map[string]runtime.Object{
				"ingresses": &unitinputs.IngressesListEmpty,
				"secrets":   &unitinputs.SecretsListEmpty,
			},
			deleted: true,
		},
		{
			name:           "delete ingress - no rgw spec, list ingress failed, failed",
			cephDpl:        &unitinputs.BaseCephDeployment,
			inputResources: map[string]runtime.Object{},
			expectedError:  errors.New("failed to find ingress to remove"),
		},
		{
			name:    "delete ingress - no rgw spec, ingress found, no secrets, success",
			cephDpl: &unitinputs.BaseCephDeployment,
			inputResources: map[string]runtime.Object{
				"ingresses": unitinputs.IngressesList.DeepCopy(),
				"secrets":   &unitinputs.SecretsListEmpty,
			},
		},
		{
			name:    "delete ingress - no rgw spec, ingress found, no secrets, failed",
			cephDpl: &unitinputs.BaseCephDeployment,
			inputResources: map[string]runtime.Object{
				"ingresses": unitinputs.IngressesList.DeepCopy(),
				"secrets":   &unitinputs.SecretsListEmpty,
			},
			apiErrors: map[string]error{
				"delete-ingresses": errors.New("failed to delete ingress"),
			},
			expectedError: errors.New("failed to delete ingress rook-ceph-rgw-rgw-store-ingress"),
		},
		{
			name:    "delete ingress - no rgw spec, ingress found, secrets removed, success",
			cephDpl: &unitinputs.BaseCephDeployment,
			inputResources: map[string]runtime.Object{
				"ingresses": unitinputs.IngressesList.DeepCopy(),
				"secrets":   &v1.SecretList{Items: []v1.Secret{unitinputs.IngressRuleSecret}},
			},
		},
		{
			name:    "delete ingress - no rgw spec, ingress found, failed to list secret",
			cephDpl: &unitinputs.BaseCephDeployment,
			inputResources: map[string]runtime.Object{
				"ingresses": unitinputs.IngressesList.DeepCopy(),
			},
			expectedError: errors.New("failed to get list ingress secrets to remove: failed to list secrets"),
		},
		{
			name:    "delete ingress - no rgw spec, ingress found, failed to delete secret",
			cephDpl: &unitinputs.BaseCephDeployment,
			inputResources: map[string]runtime.Object{
				"ingresses": unitinputs.IngressesList.DeepCopy(),
				"secrets":   &v1.SecretList{Items: []v1.Secret{unitinputs.IngressRuleSecret}},
			},
			apiErrors: map[string]error{
				"delete-collection-secrets": errors.New("failed to delete secret"),
			},
			expectedError: errors.New("failed to delete tls secrets for ingress"),
		},
		{
			name:    "delete ingress - rgw spec, ingress removed, no secrets, success",
			cephDpl: &unitinputs.CephDeployNonMoskWithIngress,
			inputResources: map[string]runtime.Object{
				"ingresses": unitinputs.IngressesList.DeepCopy(),
				"secrets":   &unitinputs.SecretsListEmpty,
			},
		},
		{
			name:    "delete ingress - rgw spec, no secrets, ingress delete failed",
			cephDpl: &unitinputs.CephDeployNonMoskWithIngress,
			inputResources: map[string]runtime.Object{
				"ingresses": unitinputs.IngressesList.DeepCopy(),
				"secrets":   &unitinputs.SecretsListEmpty,
			},
			apiErrors: map[string]error{
				"delete-ingresses": errors.New("failed to delete ingress"),
			},
			expectedError: errors.New("failed to delete ingress rook-ceph-rgw-rgw-store-ingress"),
		},
		{
			name:    "delete ingress - rgw spec, no secrets, ingress delete not found, success",
			cephDpl: &unitinputs.CephDeployNonMoskWithIngress,
			inputResources: map[string]runtime.Object{
				"ingresses": &unitinputs.IngressesListEmpty,
				"secrets":   &unitinputs.SecretsListEmpty,
			},
			deleted: true,
		},
		{
			name: "delete ingress - no rgw spec, custom class name, ingress found, no secrets, success",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				CustomIngress := unitinputs.BaseCephDeployment.DeepCopy()
				CustomIngress.Spec.IngressConfig = &cephlcmv1alpha1.CephDeploymentIngressConfig{
					ControllerClassName: "fake.com",
				}
				return CustomIngress
			}(),
			inputResources: map[string]runtime.Object{
				"ingresses": &networkingv1.IngressList{
					Items: []networkingv1.Ingress{
						func() networkingv1.Ingress {
							ingress := unitinputs.RgwIngress.DeepCopy()
							ingress.Labels["ingress-type"] = "fake.com-rgw"
							return *ingress
						}(),
					},
				},
				"secrets": &unitinputs.SecretsListEmpty,
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := fakeDeploymentConfig(&deployConfig{cephDpl: test.cephDpl}, nil)
			faketestclients.FakeReaction(c.api.Kubeclientset.CoreV1(), "list", []string{"secrets"}, test.inputResources, nil)
			faketestclients.FakeReaction(c.api.Kubeclientset.CoreV1(), "delete-collection", []string{"secrets"}, test.inputResources, test.apiErrors)
			faketestclients.FakeReaction(c.api.Kubeclientset.NetworkingV1(), "list", []string{"ingresses"}, test.inputResources, nil)
			faketestclients.FakeReaction(c.api.Kubeclientset.NetworkingV1(), "delete", []string{"ingresses"}, test.inputResources, test.apiErrors)

			deleted, err := c.deleteIngressProxy()
			if test.expectedError != nil {
				assert.NotNil(t, err)
				assert.Contains(t, err.Error(), test.expectedError.Error())
			} else {
				assert.Nil(t, err)
				assert.Equal(t, test.deleted, deleted)
			}

			faketestclients.CleanupFakeClientReactions(c.api.Kubeclientset.CoreV1())
			faketestclients.CleanupFakeClientReactions(c.api.Kubeclientset.NetworkingV1())
		})
	}
}

func TestEnsureIngressProxy(t *testing.T) {
	tests := []struct {
		name              string
		cephDpl           *cephlcmv1alpha1.CephDeployment
		lcmConfigParams   map[string]string
		inputResources    map[string]runtime.Object
		apiErrors         map[string]error
		stateChanged      bool
		expectedResources map[string]runtime.Object
		expectedError     error
	}{
		{
			name:    "ensure ingress - no object storage, nothing to delete",
			cephDpl: &unitinputs.BaseCephDeployment,
			inputResources: map[string]runtime.Object{
				"ingresses": &networkingv1.IngressList{},
				"secrets":   &v1.SecretList{},
			},
		},
		{
			name:    "ensure ingress - no ingress required, delete in progress",
			cephDpl: &unitinputs.CephDeployNonMoskWithIngress,
			inputResources: map[string]runtime.Object{
				"ingresses": &networkingv1.IngressList{Items: []networkingv1.Ingress{*unitinputs.RgwIngress.DeepCopy()}},
				"secrets":   &v1.SecretList{},
			},
			expectedResources: map[string]runtime.Object{
				"ingresses": &networkingv1.IngressList{Items: []networkingv1.Ingress{}},
			},
			stateChanged: true,
		},
		{
			name:          "ensure ingress - no object storage, delete failed",
			cephDpl:       &unitinputs.BaseCephDeployment,
			expectedError: errors.New("deletion not complete for ingress proxy"),
		},
		{
			name:    "ensure ingress - no ingress required, default settings not found, skipped",
			cephDpl: &unitinputs.CephDeployMoskWithoutIngress,
			inputResources: map[string]runtime.Object{
				"ingresses": &networkingv1.IngressList{},
				"secrets":   &v1.SecretList{},
			},
		},
		{
			name:    "ensure ingress - no ingress required, default settings get failed, skipped",
			cephDpl: &unitinputs.CephDeployMoskWithoutIngress,
			inputResources: map[string]runtime.Object{
				"ingresses": &networkingv1.IngressList{},
				"secrets":   &v1.SecretList{},
			},
			apiErrors: map[string]error{
				"get-secrets": errors.New("get secret failed"),
			},
			expectedError: errors.New("failed to get openstack-rgw-creds secret to ensure ingress"),
		},
		{
			name:    "ensure ingress - get ingress failed",
			cephDpl: &unitinputs.CephDeployMosk,
			inputResources: map[string]runtime.Object{
				"ingresses": &networkingv1.IngressList{},
				"secrets":   &v1.SecretList{},
			},
			apiErrors: map[string]error{
				"get-ingresses": errors.New("get ingress failed"),
			},
			expectedError: errors.New("failed to get rook-ceph-rgw-rgw-store-ingress ingress"),
		},
		{
			name:    "ensure ingress - no ingress found, default settings, create success",
			cephDpl: &unitinputs.CephDeployMoskWithoutIngress,
			inputResources: map[string]runtime.Object{
				"ingresses": &networkingv1.IngressList{},
				"secrets": &v1.SecretList{
					Items: []v1.Secret{unitinputs.OpenstackRgwCredsSecret},
				},
			},
			stateChanged: true,
			expectedResources: map[string]runtime.Object{
				"ingresses": &networkingv1.IngressList{
					Items: []networkingv1.Ingress{
						func() networkingv1.Ingress {
							ingress := unitinputs.RgwOpenstackIngress("rgw-store.openstack.com")
							ingress.Spec.TLS[0].SecretName = "rook-ceph-rgw-rgw-store-tls-public-6"
							return *ingress
						}(),
					},
				},
				"secrets": &v1.SecretList{
					Items: []v1.Secret{
						unitinputs.OpenstackRgwCredsSecret,
						func() v1.Secret {
							secret := unitinputs.IngressRuleSecretOpenstack.DeepCopy()
							secret.Name = "rook-ceph-rgw-rgw-store-tls-public-6"
							return *secret
						}(),
					},
				},
			},
		},
		{
			name:    "ensure ingress - no ingress found, default settings, create secret failed",
			cephDpl: &unitinputs.CephDeployMoskWithoutIngress,
			inputResources: map[string]runtime.Object{
				"ingresses": &networkingv1.IngressList{},
				"secrets": &v1.SecretList{
					Items: []v1.Secret{unitinputs.OpenstackRgwCredsSecret},
				},
			},
			apiErrors: map[string]error{
				"create-secrets": errors.New("create secret failed"),
			},
			expectedError: errors.New("failed to create ingress cert secret"),
		},
		{
			name:    "ensure ingress - no ingress found, default settings, create ingress failed",
			cephDpl: &unitinputs.CephDeployMoskWithoutIngress,
			inputResources: map[string]runtime.Object{
				"ingresses": &networkingv1.IngressList{},
				"secrets": &v1.SecretList{
					Items: []v1.Secret{unitinputs.OpenstackRgwCredsSecret},
				},
			},
			expectedResources: map[string]runtime.Object{
				"secrets": &v1.SecretList{
					Items: []v1.Secret{
						unitinputs.OpenstackRgwCredsSecret,
						func() v1.Secret {
							secret := unitinputs.IngressRuleSecretOpenstack.DeepCopy()
							secret.Name = "rook-ceph-rgw-rgw-store-tls-public-8"
							return *secret
						}(),
					},
				},
			},
			apiErrors: map[string]error{
				"create-ingresses": errors.New("create ingress failed"),
			},
			expectedError: errors.New("failed to create ingress rook-ceph/rook-ceph-rgw-rgw-store-ingress"),
		},
		{
			name:    "ensure ingress - no ingress found, cephdeployment settings, create success",
			cephDpl: &unitinputs.CephDeployMosk,
			inputResources: map[string]runtime.Object{
				"ingresses": &networkingv1.IngressList{},
				"secrets": &v1.SecretList{
					Items: []v1.Secret{unitinputs.OpenstackRgwCredsSecret},
				},
			},
			stateChanged: true,
			expectedResources: map[string]runtime.Object{
				"ingresses": &networkingv1.IngressList{
					Items: []networkingv1.Ingress{
						func() networkingv1.Ingress {
							ingress := unitinputs.RgwIngress.DeepCopy()
							ingress.Labels["ingress-type"] = "fake-class-name-rgw"
							ingress.Annotations = map[string]string{
								"kubernetes.io/ingress.class": "fake-class-name",
								"fake":                        "fake",
							}
							ingress.Spec.Rules[0].Host = "rgw-store.test"
							ingress.Spec.TLS[0].Hosts = []string{"rgw-store.test"}
							ingress.Spec.TLS[0].SecretName = "rook-ceph-rgw-rgw-store-tls-public-9"
							return *ingress
						}(),
					},
				},
				"secrets": &v1.SecretList{
					Items: []v1.Secret{
						unitinputs.OpenstackRgwCredsSecret,
						func() v1.Secret {
							secret := unitinputs.IngressRuleSecretCustom.DeepCopy()
							secret.Name = "rook-ceph-rgw-rgw-store-tls-public-9"
							return *secret
						}(),
					},
				},
			},
		},
		{
			name: "ensure ingress - ingress found, cephdeployment settings, update ingress success",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				mc := unitinputs.CephDeployMosk.DeepCopy()
				mc.Spec.IngressConfig.TLSConfig.Domain = "example.com"
				return mc
			}(),
			inputResources: map[string]runtime.Object{
				"ingresses": &networkingv1.IngressList{
					Items: []networkingv1.Ingress{
						func() networkingv1.Ingress {
							ingress := unitinputs.RgwIngress.DeepCopy()
							ingress.Annotations["test"] = "test"
							return *ingress
						}(),
					},
				},
				"secrets": &v1.SecretList{
					Items: []v1.Secret{*unitinputs.IngressRuleSecret.DeepCopy()},
				},
			},
			stateChanged: true,
			expectedResources: map[string]runtime.Object{
				"ingresses": &networkingv1.IngressList{
					Items: []networkingv1.Ingress{
						func() networkingv1.Ingress {
							ingress := unitinputs.RgwIngress.DeepCopy()
							ingress.Labels["ingress-type"] = "fake-class-name-rgw"
							ingress.Annotations = map[string]string{
								"kubernetes.io/ingress.class": "fake-class-name",
								"fake":                        "fake",
							}
							ingress.Spec.Rules[0].Host = "rgw-store.example.com"
							ingress.Spec.TLS[0].Hosts = []string{"rgw-store.example.com"}
							ingress.Spec.TLS[0].SecretName = "rook-ceph-rgw-rgw-store-tls-public-10"
							return *ingress
						}(),
					},
				},
				"secrets": &v1.SecretList{
					Items: []v1.Secret{
						func() v1.Secret {
							secret := unitinputs.IngressRuleSecretCustom.DeepCopy()
							secret.Name = "rook-ceph-rgw-rgw-store-tls-public-10"
							return *secret
						}(),
					},
				},
			},
		},
		{
			name: "ensure ingress - ingress found, cephdeployment settings, default class name, update ingress success",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				mc := unitinputs.CephDeployMosk.DeepCopy()
				mc.Spec.IngressConfig.TLSConfig.Domain = "example.com"
				mc.Spec.IngressConfig.ControllerClassName = ""
				return mc
			}(),
			inputResources: map[string]runtime.Object{
				"ingresses": &networkingv1.IngressList{
					Items: []networkingv1.Ingress{
						func() networkingv1.Ingress {
							ingress := unitinputs.RgwIngress.DeepCopy()
							ingress.Annotations["test"] = "test"
							return *ingress
						}(),
					},
				},
				"secrets": &v1.SecretList{
					Items: []v1.Secret{*unitinputs.IngressRuleSecret.DeepCopy()},
				},
			},
			stateChanged: true,
			expectedResources: map[string]runtime.Object{
				"ingresses": &networkingv1.IngressList{
					Items: []networkingv1.Ingress{
						func() networkingv1.Ingress {
							ingress := unitinputs.RgwIngress.DeepCopy()
							ingress.Annotations["fake"] = "fake"
							ingress.Spec.Rules[0].Host = "rgw-store.example.com"
							ingress.Spec.TLS[0].Hosts = []string{"rgw-store.example.com"}
							ingress.Spec.TLS[0].SecretName = "rook-ceph-rgw-rgw-store-tls-public-11"
							return *ingress
						}(),
					},
				},
				"secrets": &v1.SecretList{
					Items: []v1.Secret{
						func() v1.Secret {
							secret := unitinputs.IngressRuleSecretCustom.DeepCopy()
							secret.Name = "rook-ceph-rgw-rgw-store-tls-public-11"
							return *secret
						}(),
					},
				},
			},
		},
		{
			name: "ensure ingress - ingress found, cephdeployment settings, no annotations, update ingress success",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				mc := unitinputs.CephDeployMosk.DeepCopy()
				mc.Spec.IngressConfig.TLSConfig.Domain = "example.com"
				mc.Spec.IngressConfig.ControllerClassName = ""
				return mc
			}(),
			inputResources: map[string]runtime.Object{
				"ingresses": &networkingv1.IngressList{
					Items: []networkingv1.Ingress{
						func() networkingv1.Ingress {
							ingress := unitinputs.RgwIngress.DeepCopy()
							ingress.Annotations = nil
							return *ingress
						}(),
					},
				},
				"secrets": &v1.SecretList{
					Items: []v1.Secret{*unitinputs.IngressRuleSecret.DeepCopy()},
				},
			},
			stateChanged: true,
			expectedResources: map[string]runtime.Object{
				"ingresses": &networkingv1.IngressList{
					Items: []networkingv1.Ingress{
						func() networkingv1.Ingress {
							ingress := unitinputs.RgwIngress.DeepCopy()
							ingress.Annotations["fake"] = "fake"
							ingress.Spec.Rules[0].Host = "rgw-store.example.com"
							ingress.Spec.TLS[0].Hosts = []string{"rgw-store.example.com"}
							ingress.Spec.TLS[0].SecretName = "rook-ceph-rgw-rgw-store-tls-public-12"
							return *ingress
						}(),
					},
				},
				"secrets": &v1.SecretList{
					Items: []v1.Secret{
						func() v1.Secret {
							secret := unitinputs.IngressRuleSecretCustom.DeepCopy()
							secret.Name = "rook-ceph-rgw-rgw-store-tls-public-12"
							return *secret
						}(),
					},
				},
			},
		},
		{
			name: "ensure ingress - ingress found, cephdeployment settings, update ingress failed",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				mc := unitinputs.CephDeployMosk.DeepCopy()
				mc.Spec.IngressConfig.TLSConfig.Domain = "example.com"
				return mc
			}(),
			inputResources: map[string]runtime.Object{
				"ingresses": &networkingv1.IngressList{
					Items: []networkingv1.Ingress{*unitinputs.RgwIngress.DeepCopy()},
				},
				"secrets": &v1.SecretList{
					Items: []v1.Secret{*unitinputs.IngressRuleSecret.DeepCopy()},
				},
			},
			apiErrors: map[string]error{
				"update-ingresses": errors.New("failed to update ingress"),
			},
			expectedResources: map[string]runtime.Object{
				"secrets": &v1.SecretList{
					Items: []v1.Secret{
						*unitinputs.IngressRuleSecret.DeepCopy(),
						func() v1.Secret {
							secret := unitinputs.IngressRuleSecretCustom.DeepCopy()
							secret.Name = "rook-ceph-rgw-rgw-store-tls-public-13"
							return *secret
						}(),
					},
				},
			},
			expectedError: errors.New("failed to update ingress rook-ceph/rook-ceph-rgw-rgw-store-ingress"),
		},
		{
			name: "ensure ingress - ingress found, cephdeployment settings, get ingress secret failed",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				mc := unitinputs.CephDeployMosk.DeepCopy()
				mc.Spec.IngressConfig.TLSConfig.Domain = "example.com"
				return mc
			}(),
			inputResources: map[string]runtime.Object{
				"ingresses": &networkingv1.IngressList{
					Items: []networkingv1.Ingress{*unitinputs.RgwIngress.DeepCopy()},
				},
				"secrets": &v1.SecretList{
					Items: []v1.Secret{*unitinputs.IngressRuleSecret.DeepCopy()},
				},
			},
			apiErrors: map[string]error{
				"get-secrets": errors.New("failed to get secret"),
			},
			expectedError: errors.New("failed to get ingress cert secret"),
		},
		{
			name: "ensure ingress - ingress found, cephdeployment settings, ingress secret not found, recreated",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				mc := unitinputs.CephDeployMosk.DeepCopy()
				mc.Spec.IngressConfig.TLSConfig.Domain = "example.com"
				return mc
			}(),
			inputResources: map[string]runtime.Object{
				"ingresses": &networkingv1.IngressList{
					Items: []networkingv1.Ingress{*unitinputs.RgwIngress.DeepCopy()},
				},
				"secrets": &v1.SecretList{
					Items: []v1.Secret{},
				},
			},
			expectedResources: map[string]runtime.Object{
				"ingresses": &networkingv1.IngressList{
					Items: []networkingv1.Ingress{
						func() networkingv1.Ingress {
							ingress := unitinputs.RgwIngress.DeepCopy()
							ingress.Labels["ingress-type"] = "fake-class-name-rgw"
							ingress.Annotations = map[string]string{
								"fake":                        "fake",
								"kubernetes.io/ingress.class": "fake-class-name",
							}
							ingress.Spec.TLS[0].SecretName = "rook-ceph-rgw-rgw-store-tls-public-15"
							return *ingress
						}(),
					},
				},
				"secrets": &v1.SecretList{
					Items: []v1.Secret{
						func() v1.Secret {
							secret := unitinputs.IngressRuleSecretCustom.DeepCopy()
							secret.Name = "rook-ceph-rgw-rgw-store-tls-public-15"
							return *secret
						}(),
					},
				},
			},
			stateChanged: true,
		},
		{
			name: "ensure ingress - ingress found, cephdeployment settings, update secret failed",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				mc := unitinputs.CephDeployMosk.DeepCopy()
				mc.Spec.IngressConfig.TLSConfig.Domain = "example.com"
				return mc
			}(),
			inputResources: map[string]runtime.Object{
				"ingresses": &networkingv1.IngressList{
					Items: []networkingv1.Ingress{*unitinputs.RgwIngress.DeepCopy()},
				},
				"secrets": &v1.SecretList{
					Items: []v1.Secret{
						func() v1.Secret {
							secret := unitinputs.IngressRuleSecretCustom.DeepCopy()
							secret.Labels = nil
							return *secret
						}(),
					},
				},
			},
			apiErrors: map[string]error{
				"update-secrets": errors.New("failed to update secret"),
			},
			expectedError: errors.New("failed to update labels for secret"),
		},
		{
			name: "ensure ingress - ingress found, cephdeployment settings, update secret success",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				mc := unitinputs.CephDeployMosk.DeepCopy()
				mc.Spec.IngressConfig.TLSConfig.Domain = "example.com"
				return mc
			}(),
			inputResources: map[string]runtime.Object{
				"ingresses": &networkingv1.IngressList{
					Items: []networkingv1.Ingress{*unitinputs.RgwIngress.DeepCopy()},
				},
				"secrets": &v1.SecretList{
					Items: []v1.Secret{
						func() v1.Secret {
							secret := unitinputs.IngressRuleSecretCustom.DeepCopy()
							secret.Labels = nil
							return *secret
						}(),
					},
				},
			},
			expectedResources: map[string]runtime.Object{
				"ingresses": &networkingv1.IngressList{
					Items: []networkingv1.Ingress{
						func() networkingv1.Ingress {
							ingress := unitinputs.RgwIngress.DeepCopy()
							ingress.Labels["ingress-type"] = "fake-class-name-rgw"
							ingress.Annotations = map[string]string{
								"fake":                        "fake",
								"kubernetes.io/ingress.class": "fake-class-name",
							}
							return *ingress
						}(),
					},
				},
				"secrets": &v1.SecretList{
					Items: []v1.Secret{*unitinputs.IngressRuleSecretCustom.DeepCopy()},
				},
			},
			stateChanged: true,
		},
		{
			name: "ensure ingress - ingress found, cephdeployment settings, delete secret failed",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				mc := unitinputs.CephDeployMosk.DeepCopy()
				mc.Spec.IngressConfig.TLSConfig.Domain = "example.com"
				return mc
			}(),
			inputResources: map[string]runtime.Object{
				"ingresses": &networkingv1.IngressList{
					Items: []networkingv1.Ingress{*unitinputs.RgwIngress.DeepCopy()},
				},
				"secrets": &v1.SecretList{
					Items: []v1.Secret{*unitinputs.IngressRuleSecret.DeepCopy()},
				},
			},
			expectedResources: map[string]runtime.Object{
				"ingresses": &networkingv1.IngressList{
					Items: []networkingv1.Ingress{
						func() networkingv1.Ingress {
							ingress := unitinputs.RgwIngress.DeepCopy()
							ingress.Labels["ingress-type"] = "fake-class-name-rgw"
							ingress.Annotations = map[string]string{
								"kubernetes.io/ingress.class": "fake-class-name",
								"fake":                        "fake",
							}
							ingress.Spec.Rules[0].Host = "rgw-store.example.com"
							ingress.Spec.TLS[0].Hosts = []string{"rgw-store.example.com"}
							ingress.Spec.TLS[0].SecretName = "rook-ceph-rgw-rgw-store-tls-public-18"
							return *ingress
						}(),
					},
				},
				"secrets": &v1.SecretList{
					Items: []v1.Secret{
						*unitinputs.IngressRuleSecret.DeepCopy(),
						func() v1.Secret {
							secret := unitinputs.IngressRuleSecretCustom.DeepCopy()
							secret.Name = "rook-ceph-rgw-rgw-store-tls-public-18"
							return *secret
						}(),
					},
				},
			},
			apiErrors: map[string]error{
				"delete-secrets": errors.New("failed to delete secret"),
			},
			expectedError: errors.New("failed to delete old ingress cert secret"),
		},
		{
			name: "ensure ingress - ingress found, cephdeployment settings ignore wrong upstream-vhost, no update",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				mc := unitinputs.CephDeployMosk.DeepCopy()
				mc.Spec.IngressConfig.TLSConfig.Domain = "example.com"
				mc.Spec.IngressConfig.ControllerClassName = ""
				mc.Spec.IngressConfig.Annotations = map[string]string{"nginx.ingress.kubernetes.io/upstream-vhost": "some-shit"}
				return mc
			}(),
			inputResources: map[string]runtime.Object{
				"ingresses": &networkingv1.IngressList{
					Items: []networkingv1.Ingress{
						func() networkingv1.Ingress {
							ingress := unitinputs.RgwIngress.DeepCopy()
							ingress.Spec.Rules[0].Host = "rgw-store.example.com"
							ingress.Spec.TLS[0].Hosts = []string{"rgw-store.example.com"}
							return *ingress
						}(),
					},
				},
				"secrets": &v1.SecretList{
					Items: []v1.Secret{*unitinputs.IngressRuleSecretCustom.DeepCopy()},
				},
			},
			apiErrors: map[string]error{
				"update-secrets":   errors.New("unexpected secret update"),
				"update-ingresses": errors.New("unexpected ingress update"),
			},
		},
		{
			name:    "ensure ingress - no ingress required, delete in progress",
			cephDpl: &unitinputs.CephDeployNonMoskWithIngress,
			inputResources: map[string]runtime.Object{
				"ingresses": &networkingv1.IngressList{Items: []networkingv1.Ingress{*unitinputs.RgwIngress.DeepCopy()}},
				"secrets":   &v1.SecretList{Items: []v1.Secret{*unitinputs.IngressRuleSecretCustom.DeepCopy()}},
			},
			expectedResources: map[string]runtime.Object{
				"ingresses": &networkingv1.IngressList{Items: []networkingv1.Ingress{}},
			},
			stateChanged: true,
		},
		{
			name: "ensure ingress - ingress is created, secret by ref",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				mc := unitinputs.CephDeployMosk.DeepCopy()
				mc.Spec.IngressConfig.TLSConfig.TLSCerts = nil
				mc.Spec.IngressConfig.TLSConfig.TLSSecretRefName = "rgw-store-ingress-secret"
				return mc
			}(),
			inputResources: map[string]runtime.Object{
				"ingresses": &networkingv1.IngressList{},
				"secrets": &v1.SecretList{
					Items: []v1.Secret{unitinputs.OpenstackRgwCredsSecret, *unitinputs.IngressRuleSecretCustom.DeepCopy()},
				},
			},
			stateChanged: true,
			expectedResources: map[string]runtime.Object{
				"ingresses": &networkingv1.IngressList{
					Items: []networkingv1.Ingress{
						func() networkingv1.Ingress {
							ingress := unitinputs.RgwIngress.DeepCopy()
							ingress.Labels["ingress-type"] = "fake-class-name-rgw"
							ingress.Annotations = map[string]string{
								"kubernetes.io/ingress.class": "fake-class-name",
								"fake":                        "fake",
							}
							ingress.Spec.TLS[0].SecretName = "rgw-store-ingress-secret"
							ingress.Spec.Rules[0].Host = "rgw-store.test"
							ingress.Spec.TLS[0].Hosts = []string{"rgw-store.test"}
							return *ingress
						}(),
					},
				},
				"secrets": &v1.SecretList{
					Items: []v1.Secret{unitinputs.OpenstackRgwCredsSecret, *unitinputs.IngressRuleSecretCustom.DeepCopy()},
				},
			},
		},
		{
			name: "ensure ingress - tls secret by ref, no changes",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				mc := unitinputs.CephDeployMosk.DeepCopy()
				mc.Spec.IngressConfig.TLSConfig.TLSCerts = nil
				mc.Spec.IngressConfig.TLSConfig.TLSSecretRefName = "rgw-store-ingress-secret"
				return mc
			}(),
			inputResources: map[string]runtime.Object{
				"ingresses": &networkingv1.IngressList{Items: []networkingv1.Ingress{
					func() networkingv1.Ingress {
						ingress := unitinputs.RgwIngress.DeepCopy()
						ingress.Labels["ingress-type"] = "fake-class-name-rgw"
						ingress.Annotations = map[string]string{
							"kubernetes.io/ingress.class": "fake-class-name",
							"fake":                        "fake",
						}
						ingress.Spec.Rules[0].Host = "rgw-store.test"
						ingress.Spec.TLS[0].Hosts = []string{"rgw-store.test"}
						ingress.Spec.TLS[0].SecretName = "rgw-store-ingress-secret"
						return *ingress
					}(),
				}},
				"secrets": &v1.SecretList{
					Items: []v1.Secret{unitinputs.OpenstackRgwCredsSecret, *unitinputs.IngressRuleSecretCustom.DeepCopy()},
				},
			},
		},
		{
			name: "ensure ingress - tls secret by ref, failed to get secret",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				mc := unitinputs.CephDeployMosk.DeepCopy()
				mc.Spec.IngressConfig.TLSConfig.TLSCerts = nil
				mc.Spec.IngressConfig.TLSConfig.TLSSecretRefName = "rgw-store-ingress-secret"
				return mc
			}(),
			inputResources: map[string]runtime.Object{
				"ingresses": &networkingv1.IngressList{Items: []networkingv1.Ingress{
					func() networkingv1.Ingress {
						ingress := unitinputs.RgwIngress.DeepCopy()
						ingress.Labels["ingress-type"] = "fake-class-name-rgw"
						ingress.Annotations = map[string]string{
							"kubernetes.io/ingress.class": "fake-class-name",
							"fake":                        "fake",
						}
						ingress.Spec.Rules[0].Host = "rgw-store.test"
						ingress.Spec.TLS[0].Hosts = []string{"rgw-store.test"}
						ingress.Spec.TLS[0].SecretName = "rgw-store-ingress-secret"
						return *ingress
					}(),
				}},
				"secrets": &v1.SecretList{
					Items: []v1.Secret{unitinputs.OpenstackRgwCredsSecret, *unitinputs.IngressRuleSecretCustom.DeepCopy()},
				},
			},
			apiErrors: map[string]error{
				"get-secrets-rgw-store-ingress-secret": errors.New("failed to get secret"),
			},
			expectedError: errors.New("failed to get specified ingress tls certs secret 'rook-ceph/rgw-store-ingress-secret'"),
		},
		{
			name: "ensure ingress - no tls changes, switched on tls by ref",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				mc := unitinputs.CephDeployMosk.DeepCopy()
				mc.Spec.IngressConfig.TLSConfig.TLSCerts = nil
				mc.Spec.IngressConfig.TLSConfig.TLSSecretRefName = "rgw-store-ingress-secret-by-ref"
				return mc
			}(),
			inputResources: map[string]runtime.Object{
				"ingresses": &networkingv1.IngressList{Items: []networkingv1.Ingress{
					func() networkingv1.Ingress {
						ingress := unitinputs.RgwIngress.DeepCopy()
						ingress.Labels["ingress-type"] = "fake-class-name-rgw"
						ingress.Annotations = map[string]string{
							"kubernetes.io/ingress.class": "fake-class-name",
							"fake":                        "fake",
						}
						ingress.Spec.Rules[0].Host = "rgw-store.test"
						ingress.Spec.TLS[0].Hosts = []string{"rgw-store.test"}
						ingress.Spec.TLS[0].SecretName = "rgw-store-ingress-secret"
						return *ingress
					}(),
				}},
				"secrets": &v1.SecretList{
					Items: []v1.Secret{
						unitinputs.OpenstackRgwCredsSecret,
						*unitinputs.IngressRuleSecretCustom.DeepCopy(),
						func() v1.Secret {
							secret := unitinputs.IngressRuleSecretCustom.DeepCopy()
							secret.Name = "rgw-store-ingress-secret-by-ref"
							return *secret
						}(),
					},
				},
			},
			stateChanged: true,
			expectedResources: map[string]runtime.Object{
				"ingresses": &networkingv1.IngressList{Items: []networkingv1.Ingress{
					func() networkingv1.Ingress {
						ingress := unitinputs.RgwIngress.DeepCopy()
						ingress.Labels["ingress-type"] = "fake-class-name-rgw"
						ingress.Annotations = map[string]string{
							"kubernetes.io/ingress.class": "fake-class-name",
							"fake":                        "fake",
						}
						ingress.Spec.Rules[0].Host = "rgw-store.test"
						ingress.Spec.TLS[0].Hosts = []string{"rgw-store.test"}
						ingress.Spec.TLS[0].SecretName = "rgw-store-ingress-secret-by-ref"
						return *ingress
					}(),
				}},
				"secrets": &v1.SecretList{
					Items: []v1.Secret{
						unitinputs.OpenstackRgwCredsSecret,
						func() v1.Secret {
							secret := unitinputs.IngressRuleSecretCustom.DeepCopy()
							secret.Name = "rgw-store-ingress-secret-by-ref"
							return *secret
						}(),
					},
				},
			},
		},
		{
			name: "ensure ingress - create ingress with custom hostname",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				mc := unitinputs.CephDeployMosk.DeepCopy()
				mc.Spec.IngressConfig.TLSConfig.Domain = "example.com"
				mc.Spec.IngressConfig.TLSConfig.Hostname = "test"
				mc.Spec.IngressConfig.TLSConfig.TLSCerts = nil
				mc.Spec.IngressConfig.Annotations = nil
				mc.Spec.IngressConfig.ControllerClassName = ""
				return mc
			}(),
			inputResources: map[string]runtime.Object{
				"ingresses": &networkingv1.IngressList{},
				"secrets":   &v1.SecretList{Items: []v1.Secret{unitinputs.OpenstackRgwCredsSecret}},
			},
			stateChanged: true,
			expectedResources: map[string]runtime.Object{
				"ingresses": &networkingv1.IngressList{
					Items: []networkingv1.Ingress{
						func() networkingv1.Ingress {
							ingress := unitinputs.RgwIngress.DeepCopy()
							ingress.Annotations["nginx.ingress.kubernetes.io/upstream-vhost"] = "test.example.com"
							ingress.Spec.Rules[0].Host = "test.example.com"
							ingress.Spec.TLS[0].Hosts = []string{"test.example.com"}
							ingress.Spec.TLS[0].SecretName = "rook-ceph-rgw-test-tls-public-25"
							return *ingress
						}(),
					},
				},
				"secrets": &v1.SecretList{
					Items: []v1.Secret{
						unitinputs.OpenstackRgwCredsSecret,
						func() v1.Secret {
							secret := unitinputs.IngressRuleSecretOpenstack.DeepCopy()
							secret.Name = "rook-ceph-rgw-test-tls-public-25"
							return *secret
						}(),
					},
				},
			},
		},
		{
			name: "ensure ingress - no changes ingress with custom hostname",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				mc := unitinputs.CephDeployMosk.DeepCopy()
				mc.Spec.IngressConfig.TLSConfig.Domain = "example.com"
				mc.Spec.IngressConfig.TLSConfig.Hostname = "test"
				mc.Spec.IngressConfig.TLSConfig.TLSCerts = nil
				mc.Spec.IngressConfig.Annotations = nil
				mc.Spec.IngressConfig.ControllerClassName = ""
				return mc
			}(),
			inputResources: map[string]runtime.Object{
				"ingresses": &networkingv1.IngressList{
					Items: []networkingv1.Ingress{
						func() networkingv1.Ingress {
							ingress := unitinputs.RgwIngress.DeepCopy()
							ingress.Annotations["nginx.ingress.kubernetes.io/upstream-vhost"] = "test.example.com"
							ingress.Spec.Rules[0].Host = "test.example.com"
							ingress.Spec.TLS[0].Hosts = []string{"test.example.com"}
							ingress.Spec.TLS[0].SecretName = "rook-ceph-rgw-test-tls-public-25"
							return *ingress
						}(),
					},
				},
				"secrets": &v1.SecretList{
					Items: []v1.Secret{
						unitinputs.OpenstackRgwCredsSecret,
						func() v1.Secret {
							secret := unitinputs.IngressRuleSecretOpenstack.DeepCopy()
							secret.Name = "rook-ceph-rgw-test-tls-public-25"
							return *secret
						}(),
					},
				},
			},
		},
		{
			name: "ensure ingress - ingress has override for rgw and updated",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				mc := unitinputs.CephDeployMosk.DeepCopy()
				mc.Spec.IngressConfig.TLSConfig.Domain = "example.com"
				mc.Spec.IngressConfig.TLSConfig.TLSCerts = nil
				mc.Spec.IngressConfig.Annotations = nil
				mc.Spec.IngressConfig.ControllerClassName = ""
				return mc
			}(),
			inputResources: map[string]runtime.Object{
				"ingresses": &networkingv1.IngressList{
					Items: []networkingv1.Ingress{
						func() networkingv1.Ingress {
							ingress := unitinputs.RgwIngress.DeepCopy()
							ingress.Annotations["nginx.ingress.kubernetes.io/upstream-vhost"] = "test.example.com"
							ingress.Spec.Rules[0].Host = "test.example.com"
							ingress.Spec.TLS[0].Hosts = []string{"test.example.com"}
							ingress.Spec.TLS[0].SecretName = "rook-ceph-rgw-test-tls-public-25"
							return *ingress
						}(),
					},
				},
				"secrets": &v1.SecretList{
					Items: []v1.Secret{
						unitinputs.OpenstackRgwCredsSecret,
						func() v1.Secret {
							secret := unitinputs.IngressRuleSecret.DeepCopy()
							secret.Name = "rook-ceph-rgw-test-tls-public-25"
							return *secret
						}(),
					},
				},
			},
			lcmConfigParams: map[string]string{
				"RGW_PUBLIC_ACCESS_SERVICE_SELECTOR": "custom-label=custom-value",
			},
			expectedResources: map[string]runtime.Object{
				"ingresses": &networkingv1.IngressList{
					Items: []networkingv1.Ingress{
						func() networkingv1.Ingress {
							ingress := unitinputs.RgwIngress.DeepCopy()
							ingress.Labels["custom-label"] = "custom-value"
							ingress.Spec.TLS[0].SecretName = "rook-ceph-rgw-rgw-store-tls-public-27"
							return *ingress
						}(),
					},
				},
				"secrets": &v1.SecretList{
					Items: []v1.Secret{
						unitinputs.OpenstackRgwCredsSecret,
						func() v1.Secret {
							secret := unitinputs.IngressRuleSecretOpenstack.DeepCopy()
							secret.Name = "rook-ceph-rgw-rgw-store-tls-public-27"
							return *secret
						}(),
					},
				},
			},
			stateChanged: true,
		},
		{
			name:    "ensure ingress - no ingress required, default settings not set, skipped",
			cephDpl: &unitinputs.CephDeployMoskWithoutIngress,
			inputResources: map[string]runtime.Object{
				"ingresses": &networkingv1.IngressList{},
				"secrets":   &v1.SecretList{},
			},
			lcmConfigParams: map[string]string{"DEPLOYMENT_OPENSTACK_CEPH_SHARED_NAMESPACE": ""},
		},
		{
			name: "ensure ingress - no ingress required, custom ingress class and no tls config, skipped",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				cephDpl := unitinputs.CephDeployMosk.DeepCopy()
				cephDpl.Spec.IngressConfig.TLSConfig.TLSCerts = nil
				return cephDpl
			}(),
			inputResources: map[string]runtime.Object{
				"ingresses": &networkingv1.IngressList{},
				"secrets":   &v1.SecretList{},
			},
		},
	}
	oldFunc := lcmcommon.GetCurrentUnixTimeString
	for idx, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := fakeDeploymentConfig(&deployConfig{cephDpl: test.cephDpl}, test.lcmConfigParams)

			lcmcommon.GetCurrentUnixTimeString = func() string {
				return fmt.Sprintf("%d", idx)
			}

			faketestclients.FakeReaction(c.api.Kubeclientset.CoreV1(), "list", []string{"secrets"}, test.inputResources, nil)
			faketestclients.FakeReaction(c.api.Kubeclientset.NetworkingV1(), "list", []string{"ingresses"}, test.inputResources, nil)
			faketestclients.FakeReaction(c.api.Kubeclientset.CoreV1(), "get", []string{"secrets"}, test.inputResources, test.apiErrors)
			faketestclients.FakeReaction(c.api.Kubeclientset.NetworkingV1(), "get", []string{"ingresses"}, test.inputResources, test.apiErrors)
			faketestclients.FakeReaction(c.api.Kubeclientset.CoreV1(), "create", []string{"secrets"}, test.inputResources, test.apiErrors)
			faketestclients.FakeReaction(c.api.Kubeclientset.NetworkingV1(), "create", []string{"ingresses"}, test.inputResources, test.apiErrors)
			faketestclients.FakeReaction(c.api.Kubeclientset.CoreV1(), "update", []string{"secrets"}, test.inputResources, test.apiErrors)
			faketestclients.FakeReaction(c.api.Kubeclientset.NetworkingV1(), "update", []string{"ingresses"}, test.inputResources, test.apiErrors)
			faketestclients.FakeReaction(c.api.Kubeclientset.CoreV1(), "delete", []string{"secrets"}, test.inputResources, test.apiErrors)
			faketestclients.FakeReaction(c.api.Kubeclientset.NetworkingV1(), "delete", []string{"ingresses"}, test.inputResources, test.apiErrors)
			test.expectedResources = faketestclients.PrepareExpectedResources(test.inputResources, test.expectedResources)

			changed, err := c.ensureIngressProxy()
			if test.expectedError != nil {
				assert.NotNil(t, err)
				assert.Contains(t, err.Error(), test.expectedError.Error())
			} else {
				assert.Nil(t, err)
			}
			assert.Equal(t, test.stateChanged, changed)
			assert.Equal(t, test.expectedResources, test.inputResources)
			faketestclients.CleanupFakeClientReactions(c.api.Kubeclientset.CoreV1())
			faketestclients.CleanupFakeClientReactions(c.api.Kubeclientset.NetworkingV1())
		})
	}
	lcmcommon.GetCurrentUnixTimeString = oldFunc
}
