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
	"errors"
	"strings"
	"testing"
	"time"

	cephv1 "github.com/rook/rook/pkg/apis/ceph.rook.io/v1"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	cephlcmv1alpha1 "github.com/Mirantis/pelagia/v3/pkg/apis/ceph.pelagia.lcm/v1alpha1"
	lcmcommon "github.com/Mirantis/pelagia/v3/pkg/common"
	faketestclients "github.com/Mirantis/pelagia/v3/test/unit/clients"
	unitinputs "github.com/Mirantis/pelagia/v3/test/unit/inputs"
)

func TestDeleteOpenstackSecret(t *testing.T) {
	tests := []struct {
		name          string
		nonmosk       bool
		deleteError   error
		deleted       bool
		expectedError string
	}{
		{
			name: "delete openstack secrets - no error, in progress",
		},
		{
			name:    "delete openstack secrets - not found, success",
			deleted: true,
		},
		{
			name:          "delete openstack secrets - failed with error",
			deleteError:   errors.New("secrets delete failed"),
			expectedError: "failed to delete openstack secret openstack-ceph-shared/openstack-ceph-keys: secrets delete failed",
		},
		{
			name:    "delete openstack secrets - non-mosk, success",
			nonmosk: true,
			deleted: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := fakeDeploymentConfig(nil, nil)
			if test.nonmosk {
				c.lcmConfig.DeployParams.OpenstackCephSharedNamespace = ""
			}
			inputRes := map[string]runtime.Object{"secrets": &corev1.SecretList{}}
			if !test.deleted {
				inputRes["secrets"] = &corev1.SecretList{Items: []corev1.Secret{unitinputs.OpenstackSecretGenerated}}
			}
			faketestclients.FakeReaction(c.api.Kubeclientset.CoreV1(), "delete", []string{"secrets"}, inputRes, map[string]error{"delete-secrets": test.deleteError})
			deleted, err := c.deleteOpenstackSecret()
			if test.expectedError != "" {
				assert.NotNil(t, err)
				assert.Equal(t, test.expectedError, err.Error())
			} else {
				assert.Nil(t, err)
			}
			assert.Equal(t, test.deleted, deleted)
			faketestclients.CleanupFakeClientReactions(c.api.Kubeclientset.CoreV1())
		})
	}
}

func TestGenerateOpenstackSecret(t *testing.T) {
	tests := []struct {
		name          string
		cephDpl       *cephlcmv1alpha1.CephDeployment
		lcmConfig     map[string]string
		secretError   error
		adminSecret   *corev1.Secret
		rgwSecret     *corev1.Secret
		rgwMetrics    *corev1.Secret
		ingressSecret *corev1.Secret
		expected      *corev1.Secret
		expectedError error
	}{
		{
			name:        "generate openstack shared secret no extra services - success",
			cephDpl:     &unitinputs.CephDeployMoskWithoutRgw,
			adminSecret: &unitinputs.RookCephMonSecret,
			expected:    &unitinputs.CephKeysOpenstackSecretBase,
		},
		{
			name:    "generate openstack shared secret full - success",
			cephDpl: &unitinputs.CephDeployMosk,
			lcmConfig: map[string]string{
				"KEEP_INGRESS": "true",
			},
			adminSecret: &unitinputs.RookCephMonSecret,
			rgwSecret:   &unitinputs.OpenstackRgwCredsSecret,
			expected:    &unitinputs.OpenstackSecretGenerated,
		},
		{
			name:    "generate openstack shared secret with cephfs - success",
			cephDpl: &unitinputs.CephDeployMoskWithCephFS,
			lcmConfig: map[string]string{
				"KEEP_INGRESS": "true",
			},
			adminSecret: &unitinputs.RookCephMonSecret,
			rgwSecret:   &unitinputs.OpenstackRgwCredsSecret,
			expected:    &unitinputs.OpenstackSecretGeneratedCephFS,
		},
		{
			name: "generate openstack shared secret with ingress and cert by ref - failed",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				mc := unitinputs.CephDeployMosk.DeepCopy()
				mc.Spec.IngressConfig.TLSConfig.TLSCerts = nil
				mc.Spec.IngressConfig.TLSConfig.TLSSecretRefName = "rgw-store-ingress-secret"
				return mc
			}(),
			lcmConfig: map[string]string{
				"KEEP_INGRESS": "true",
			},
			secretError: errors.New("failed to get secret"),
			adminSecret: &unitinputs.RookCephMonSecret,
			rgwSecret:   &unitinputs.OpenstackRgwCredsSecret,
			expected: func() *corev1.Secret {
				secret := unitinputs.OpenstackSecretGenerated.DeepCopy()
				delete(secret.Data, "rgw_external_custom_cacert")
				return secret
			}(),
		},
		{
			name: "generate openstack shared secret with ingress and cert by ref - success",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				mc := unitinputs.CephDeployMosk.DeepCopy()
				mc.Spec.IngressConfig.TLSConfig.TLSCerts = nil
				mc.Spec.IngressConfig.TLSConfig.TLSSecretRefName = "rgw-store-ingress-secret"
				return mc
			}(),
			lcmConfig: map[string]string{
				"KEEP_INGRESS": "true",
			},
			adminSecret:   &unitinputs.RookCephMonSecret,
			rgwSecret:     &unitinputs.OpenstackRgwCredsSecret,
			ingressSecret: unitinputs.IngressRuleSecret.DeepCopy(),
			expected: func() *corev1.Secret {
				secret := unitinputs.OpenstackSecretGenerated.DeepCopy()
				secret.Data["rgw_external_custom_cacert"] = []byte("ingress-cacert")
				return secret
			}(),
		},
		{
			name: "generate openstack shared secret with ingress with custom hostname - success",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				mc := unitinputs.CephDeployMosk.DeepCopy()
				mc.Spec.IngressConfig.TLSConfig.Hostname = "custom-hostname"
				return mc
			}(),
			lcmConfig: map[string]string{
				"KEEP_INGRESS": "true",
			},
			adminSecret:   &unitinputs.RookCephMonSecret,
			rgwSecret:     &unitinputs.OpenstackRgwCredsSecret,
			ingressSecret: unitinputs.IngressRuleSecret.DeepCopy(),
			expected: func() *corev1.Secret {
				secret := unitinputs.OpenstackSecretGenerated.DeepCopy()
				secret.Data["rgw_external"] = []byte("https://custom-hostname.test/")
				return secret
			}(),
		},
		{
			name: "generate openstack shared secret with multiple volumes pools - success",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				mc := unitinputs.CephDeployMosk.DeepCopy()
				mc.Spec.BlockStorage = &cephlcmv1alpha1.CephBlockStorage{
					Pools: []cephlcmv1alpha1.CephPool{
						unitinputs.GetCephDeployPool("vms", "vms"),
						unitinputs.GetCephDeployPool("images", "images"),
						unitinputs.GetCephDeployPool("volumes", "volumes"),
						unitinputs.GetCephDeployPool("volumes-2", "volumes"),
						unitinputs.GetCephDeployPool("volumes-backend-1", "volumes-backend"),
						unitinputs.GetCephDeployPool("backup", "backup"),
					},
				}
				return mc
			}(),
			lcmConfig: map[string]string{
				"KEEP_INGRESS": "true",
			},
			adminSecret: &unitinputs.RookCephMonSecret,
			rgwSecret:   &unitinputs.OpenstackRgwCredsSecret,
			expected: func() *corev1.Secret {
				secret := unitinputs.OpenstackSecretGenerated.DeepCopy()
				secret.Data["nova"] = []byte(`{"client_name":"client.novaN9y4kl7t4vTb","client_id":"novaN9y4kl7t4vTb","key":"nova-key","pools":["vms-hdd:vms:hdd","images-hdd:images:hdd","volumes-hdd:volumes:hdd","volumes-2-hdd:volumes:hdd","volumes-backend-1-hdd:volumes:hdd"]}`)
				secret.Data["cinder"] = []byte(`{"client_name":"client.cinderVMucF_ZXr9Yz","client_id":"cinderVMucF_ZXr9Yz","key":"cinder-key","pools":["images-hdd:images:hdd","volumes-hdd:volumes:hdd","volumes-2-hdd:volumes:hdd","volumes-backend-1-hdd:volumes:hdd","backup-hdd:backup:hdd"]}`)
				return secret
			}(),
		},
		{
			name: "generate openstack shared secret http rgw endpoints - success",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				object := unitinputs.CephDeployMosk.DeepCopy()
				object.Spec.IngressConfig = nil
				rgwCasted, _ := object.Spec.ObjectStorage.Rgws[0].GetSpec()
				rgwCasted.Gateway.SecurePort = int32(0)
				rgwCasted.Gateway.Port = int32(8989)
				object.Spec.ObjectStorage.Rgws[0].Spec.Raw = unitinputs.ConvertStructToRaw(rgwCasted)
				return object
			}(),
			adminSecret: &unitinputs.RookCephMonSecret,
			rgwSecret: func() *corev1.Secret {
				secret := unitinputs.OpenstackRgwCredsSecret.DeepCopy()
				delete(secret.Data, "tls_crt")
				return secret
			}(),
			expected: func() *corev1.Secret {
				expected := unitinputs.OpenstackSecretGenerated.DeepCopy()
				expected.Data["rgw_internal"] = []byte("http://rook-ceph-rgw-rgw-store.rook-ceph.svc:8989/")
				expected.Data["rgw_external"] = []byte("http://rgw-store.openstack.com/")
				delete(expected.Data, "rgw_external_custom_cacert")
				return expected
			}(),
		},
		{
			name:        "generate openstack shared secret no ingress - success",
			cephDpl:     &unitinputs.CephDeployMoskWithoutIngress,
			adminSecret: &unitinputs.RookCephMonSecret,
			rgwSecret:   &unitinputs.OpenstackRgwCredsSecret,
			expected: func() *corev1.Secret {
				secret := unitinputs.OpenstackSecretGenerated.DeepCopy()
				delete(secret.Data, "rgw_external_custom_cacert")
				secret.Data["rgw_external"] = []byte("https://rgw-store.openstack.com/")
				return secret
			}(),
		},
		{
			name:        "generate openstack shared secret with httproute - success",
			cephDpl:     &unitinputs.CephDeployMoskWithHTTPRoute,
			adminSecret: &unitinputs.RookCephMonSecret,
			rgwSecret:   &unitinputs.OpenstackRgwCredsSecret,
			expected: func() *corev1.Secret {
				secret := unitinputs.OpenstackSecretGenerated.DeepCopy()
				delete(secret.Data, "rgw_external_custom_cacert")
				secret.Data["rgw_external"] = []byte("https://rgw-store-custom.openstack.com/")
				return secret
			}(),
		},
		{
			name: "generate openstack shared secret - external ceph cluster, ip rgw plain endpoint",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				object := unitinputs.CephDeployMosk.DeepCopy()
				object.Spec.Cluster = unitinputs.CephDeployExternal.Spec.Cluster.DeepCopy()
				rgwCasted, _ := object.Spec.ObjectStorage.Rgws[0].GetSpec()
				rgwCasted.Gateway.SecurePort = 0
				rgwCasted.Gateway.ExternalRgwEndpoints = []cephv1.EndpointAddress{{IP: "172.168.0.15"}}
				object.Spec.ObjectStorage.Rgws[0].Spec.Raw = unitinputs.ConvertStructToRaw(rgwCasted)
				return object
			}(),
			adminSecret: &unitinputs.RookCephMonSecret,
			expected: func() *corev1.Secret {
				expected := unitinputs.OpenstackSecretGenerated.DeepCopy()
				expected.Data["rgw_external"] = []byte("http://172.168.0.15:80")
				expected.Data["rgw_external_custom_cacert"] = unitinputs.RgwSSLCertSecret.Data["cabundle"]
				delete(expected.Data, "rgw_internal")
				delete(expected.Data, "rgw_internal_cacert")
				return expected
			}(),
		},
		{
			name: "generate openstack shared secret - external ceph cluster, hostname secure rgw endpoint",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				object := unitinputs.CephDeployMosk.DeepCopy()
				object.Spec.Cluster = unitinputs.CephDeployExternal.Spec.Cluster.DeepCopy()
				rgwCasted, _ := object.Spec.ObjectStorage.Rgws[0].GetSpec()
				rgwCasted.Gateway.ExternalRgwEndpoints = []cephv1.EndpointAddress{{IP: "some-rgw-domain"}}
				object.Spec.ObjectStorage.Rgws[0].Spec.Raw = unitinputs.ConvertStructToRaw(rgwCasted)
				return object
			}(),
			adminSecret: &unitinputs.RookCephMonSecret,
			expected: func() *corev1.Secret {
				expected := unitinputs.OpenstackSecretGenerated.DeepCopy()
				delete(expected.Data, "rgw_internal")
				delete(expected.Data, "rgw_internal_cacert")
				expected.Data["rgw_external"] = []byte("https://some-rgw-domain:8443")
				expected.Data["rgw_external_custom_cacert"] = unitinputs.RgwSSLCertSecret.Data["cabundle"]
				return expected
			}(),
		},
		{
			name: "generate openstack shared secret - external ceph cluster, ip secure rgw endpoint",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				object := unitinputs.CephDeployMosk.DeepCopy()
				object.Spec.Cluster = unitinputs.CephDeployExternal.Spec.Cluster.DeepCopy()
				rgwCasted, _ := object.Spec.ObjectStorage.Rgws[0].GetSpec()
				rgwCasted.Gateway.ExternalRgwEndpoints = []cephv1.EndpointAddress{{IP: "10.0.0.1"}}
				object.Spec.ObjectStorage.Rgws[0].Spec.Raw = unitinputs.ConvertStructToRaw(rgwCasted)
				return object
			}(),
			adminSecret: &unitinputs.RookCephMonSecret,
			expected: func() *corev1.Secret {
				expected := unitinputs.OpenstackSecretGenerated.DeepCopy()
				delete(expected.Data, "rgw_internal")
				delete(expected.Data, "rgw_internal_cacert")
				expected.Data["rgw_external"] = []byte("https://10.0.0.1:8443")
				expected.Data["rgw_external_custom_cacert"] = unitinputs.RgwSSLCertSecret.Data["cabundle"]
				return expected
			}(),
		},
		{
			name: "generate openstack shared secret - external ceph cluster, no rgw endpoint provided",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				object := unitinputs.CephDeployMosk.DeepCopy()
				object.Spec.Cluster = unitinputs.CephDeployExternal.Spec.Cluster.DeepCopy()
				return object
			}(),
			adminSecret: &unitinputs.RookCephMonSecret,
			expected: func() *corev1.Secret {
				expected := unitinputs.OpenstackSecretGenerated.DeepCopy()
				delete(expected.Data, "rgw_internal")
				delete(expected.Data, "rgw_internal_cacert")
				delete(expected.Data, "rgw_external")
				delete(expected.Data, "rgw_external_custom_cacert")
				return expected
			}(),
		},
		{
			name: "generate openstack shared secret - external ceph cluster, no rgw",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				object := unitinputs.CephDeployMosk.DeepCopy()
				object.Spec.Cluster = unitinputs.CephDeployExternal.Spec.Cluster.DeepCopy()
				object.Spec.ObjectStorage = nil
				return object
			}(),
			adminSecret: &unitinputs.RookCephMonSecret,
			expected:    &unitinputs.CephKeysOpenstackSecretBase,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := fakeDeploymentConfig(&deployConfig{cephDpl: test.cephDpl}, test.lcmConfig)
			err := c.castExtensions()
			assert.Nil(t, err)

			secretData := openstackSecretData{
				clientsInfo: map[string]OpenstackClientInfo{
					"nova":   {Name: "client.novaN9y4kl7t4vTb", ID: "novaN9y4kl7t4vTb", Keyring: "nova-key"},
					"cinder": {Name: "client.cinderVMucF_ZXr9Yz", ID: "cinderVMucF_ZXr9Yz", Keyring: "cinder-key"},
					"glance": {Name: "client.glanceo1sdvEEqDxxo", ID: "glanceo1sdvEEqDxxo", Keyring: "glance-key"},
				},
				monMap:           &unitinputs.RookCephMonEndpoints,
				adminSecret:      test.adminSecret,
				rgwSecret:        test.rgwSecret,
				rgwInternalCert:  &unitinputs.RgwSSLCertSecret,
				rgwMetricsSecret: test.rgwMetrics,
			}
			if test.cephDpl.Spec.SharedFilesystem != nil {
				secretData.clientsInfo["manila"] = OpenstackClientInfo{
					Name: "client.manilaqnJWCfLAghgC", ID: "manilaqnJWCfLAghgC", Keyring: "manila-key",
				}
			}

			inputRes := map[string]runtime.Object{}
			if test.ingressSecret != nil {
				inputRes["secrets"] = &corev1.SecretList{Items: []corev1.Secret{*test.ingressSecret}}
			}
			faketestclients.FakeReaction(c.api.Kubeclientset.CoreV1(), "get", []string{"secrets"}, inputRes, map[string]error{"get-secrets": test.secretError})

			actual := c.generateOpenstackSecret(secretData)
			assert.Equal(t, test.expected, actual)
			faketestclients.CleanupFakeClientReactions(c.api.Kubeclientset.CoreV1())
		})
	}
}

func TestGetOpenstackClientNames(t *testing.T) {
	tests := []struct {
		name           string
		inputResources map[string]runtime.Object
		cephFSDeployed bool
		expectedNames  map[string]string
		expectedError  string
	}{
		{
			name:           "openstack clients failed to list",
			inputResources: map[string]runtime.Object{},
			expectedError:  "failed to list CephClients in namespace 'rook-ceph': failed to list cephclients",
		},
		{
			name:           "openstack clients found, no cephfs deployed - success",
			inputResources: map[string]runtime.Object{"cephclients": &unitinputs.CephClientListOpenstack},
			expectedNames: map[string]string{
				"nova":   "novaN9y4kl7t4vTb",
				"glance": "glanceo1sdvEEqDxxo",
				"cinder": "cinderVMucF_ZXr9Yz",
			},
		},
		{
			name:           "openstack clients found, cephfs deployed - success",
			cephFSDeployed: true,
			inputResources: map[string]runtime.Object{"cephclients": &unitinputs.CephClientListOpenstackFull},
			expectedNames: map[string]string{
				"nova":   "novaN9y4kl7t4vTb",
				"glance": "glanceo1sdvEEqDxxo",
				"cinder": "cinderVMucF_ZXr9Yz",
				"manila": "manilaqnJWCfLAghgC",
			},
		},
		{
			name: "openstack clients found, multiple client, take last - success",
			inputResources: map[string]runtime.Object{
				"cephclients": &cephv1.CephClientList{
					Items: []cephv1.CephClient{
						unitinputs.CephClientCinder, unitinputs.CephClientGlance, unitinputs.CephClientNova,
						func() cephv1.CephClient {
							cl := unitinputs.CephClientCinder.DeepCopy()
							cl.CreationTimestamp = metav1.NewTime(time.Now())
							cl.Spec.Name = "cinder-last"
							return *cl
						}(),
					},
				},
			},
			expectedNames: map[string]string{
				"nova":   "novaN9y4kl7t4vTb",
				"glance": "glanceo1sdvEEqDxxo",
				"cinder": "cinder-last",
			},
		},
		{
			name:           "openstack clients failed to find some clients",
			inputResources: map[string]runtime.Object{"cephclients": &unitinputs.CephClientListOpenstack},
			cephFSDeployed: true,
			expectedError:  "failed to find some OpenStack CephClient(s)",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := fakeDeploymentConfig(nil, nil)
			faketestclients.FakeReaction(c.api.Rookclientset, "list", []string{"cephclients"}, test.inputResources, nil)
			expectedNames, err := c.getOpenstackClientNames(test.cephFSDeployed)
			if test.expectedError != "" {
				assert.NotNil(t, err)
				assert.Equal(t, test.expectedError, err.Error())
			} else {
				assert.Nil(t, err)
				assert.Equal(t, test.expectedNames, expectedNames)
			}
			faketestclients.CleanupFakeClientReactions(c.api.Rookclientset)
		})
	}
}

func TestEnsureOpenstackSecret(t *testing.T) {
	tests := []struct {
		name              string
		cephDpl           *cephlcmv1alpha1.CephDeployment
		inputResources    map[string]runtime.Object
		lcmConfig         map[string]string
		expectedResources map[string]runtime.Object
		apiErrors         map[string]error
		getAuthKeyError   error
		changed           bool
		expectedError     string
	}{
		{
			name: "generate openstack secret - generate disabled",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				mc := unitinputs.BaseCephDeployment.DeepCopy()
				mc.Spec.ExtraOpts = &cephlcmv1alpha1.CephDeploymentExtraOpts{DisableOsKeys: true}
				return mc
			}(),
		},
		{
			name:    "generate openstack secret - no openstack pools, skipped",
			cephDpl: &unitinputs.BaseCephDeployment,
		},
		{
			name:    "generate openstack secret - pools are not ready",
			cephDpl: &unitinputs.CephDeployMosk,
			inputResources: map[string]runtime.Object{
				"cephblockpools": &unitinputs.OpenstackCephBlockPoolsList,
			},
			expectedError: "skip openstack secret ensure since the following required OpenStack pools are not ready yet: [vms-hdd volumes-hdd images-hdd backup-hdd]",
		},
		{
			name:    "generate openstack secret - openstack clients not found",
			cephDpl: &unitinputs.CephDeployMosk,
			inputResources: map[string]runtime.Object{
				"cephblockpools": &unitinputs.OpenstackCephBlockPoolsListReady,
				"cephclients":    &unitinputs.CephClientListEmpty,
			},
			expectedError: "failed to verify Openstack CephClient(s): failed to find some OpenStack CephClient(s)",
		},
		{
			name:    "generate openstack secret - failed to get auth keys",
			cephDpl: &unitinputs.CephDeployMosk,
			inputResources: map[string]runtime.Object{
				"cephblockpools": &unitinputs.OpenstackCephBlockPoolsListReady,
				"cephclients":    &unitinputs.CephClientListOpenstack,
			},
			getAuthKeyError: errors.New("cmd failed"),
			expectedError:   "failed to get auth keys for ceph clients: some auth keys failed to get: failed to run 'ceph auth get-key client.cinderVMucF_ZXr9Yz' command, failed to run 'ceph auth get-key client.glanceo1sdvEEqDxxo' command, failed to run 'ceph auth get-key client.novaN9y4kl7t4vTb' command",
		},
		{
			name:    "generate openstack secret - get monmap cm error",
			cephDpl: &unitinputs.CephDeployMosk,
			inputResources: map[string]runtime.Object{
				"cephblockpools": &unitinputs.OpenstackCephBlockPoolsListReady,
				"cephclients":    &unitinputs.CephClientListOpenstack,
				"configmaps":     unitinputs.ConfigMapListEmpty,
			},
			expectedError: "failed to get ceph monitor endpoints: failed to get rook-ceph/rook-ceph-mon-endpoints configmap: configmaps \"rook-ceph-mon-endpoints\" not found",
		},
		{
			name:    "generate openstack secret - get admin secret error",
			cephDpl: &unitinputs.CephDeployMosk,
			inputResources: map[string]runtime.Object{
				"cephblockpools": &unitinputs.OpenstackCephBlockPoolsListReady,
				"cephclients":    &unitinputs.CephClientListOpenstack,
				"configmaps": &corev1.ConfigMapList{
					Items: []corev1.ConfigMap{unitinputs.RookCephMonEndpoints},
				},
				"secrets": &unitinputs.SecretsListEmpty,
			},
			expectedError: "failed to get ceph admin secret: failed to get rook-ceph/rook-ceph-mon admin secret: secrets \"rook-ceph-mon\" not found",
		},
		{
			name:    "generate openstack secret - failed to get rgw info, secret created",
			cephDpl: &unitinputs.CephDeployMosk,
			inputResources: map[string]runtime.Object{
				"cephblockpools": &unitinputs.OpenstackCephBlockPoolsListReady,
				"cephclients":    &unitinputs.CephClientListOpenstack,
				"configmaps": &corev1.ConfigMapList{
					Items: []corev1.ConfigMap{unitinputs.RookCephMonEndpoints},
				},
				"secrets": &corev1.SecretList{
					Items: []corev1.Secret{unitinputs.RookCephMonSecret},
				},
				"cephobjectstoreusers": &unitinputs.CephObjectStoreUserListEmpty,
			},
			lcmConfig: map[string]string{
				"KEEP_INGRESS": "true",
			},
			changed: true,
			expectedResources: map[string]runtime.Object{
				"secrets": &corev1.SecretList{Items: []corev1.Secret{unitinputs.RookCephMonSecret, unitinputs.CephKeysOpenstackSecretRgwBase}},
			},
		},
		{
			name:    "generate openstack secret - failed to check shared secret presence",
			cephDpl: &unitinputs.CephDeployMosk,
			inputResources: map[string]runtime.Object{
				"cephblockpools": &unitinputs.OpenstackCephBlockPoolsListReady,
				"cephclients":    &unitinputs.CephClientListOpenstack,
				"configmaps": &corev1.ConfigMapList{
					Items: []corev1.ConfigMap{unitinputs.RookCephMonEndpoints},
				},
				"secrets": &corev1.SecretList{
					Items: []corev1.Secret{unitinputs.RookCephMonSecret},
				},
				"cephobjectstoreusers": &unitinputs.CephObjectStoreUserListEmpty,
			},
			apiErrors:     map[string]error{"get-secrets-openstack-ceph-keys": errors.New("failed to get")},
			expectedError: "failed to get openstack-ceph-shared/openstack-ceph-keys secret: failed to get",
		},
		{
			name:    "generate openstack secret - failed to create shared secret",
			cephDpl: &unitinputs.CephDeployMosk,
			inputResources: map[string]runtime.Object{
				"cephblockpools": &unitinputs.OpenstackCephBlockPoolsListReady,
				"cephclients":    &unitinputs.CephClientListOpenstack,
				"configmaps": &corev1.ConfigMapList{
					Items: []corev1.ConfigMap{unitinputs.RookCephMonEndpoints},
				},
				"secrets": &corev1.SecretList{
					Items: []corev1.Secret{unitinputs.RookCephMonSecret},
				},
				"cephobjectstoreusers": &unitinputs.CephObjectStoreUserListEmpty,
			},
			apiErrors:     map[string]error{"create-secrets-openstack-ceph-keys": errors.New("failed to create")},
			expectedError: "failed to create openstack-ceph-shared/openstack-ceph-keys secret: failed to create",
		},
		{
			name:    "generate openstack secret - full info, secret created",
			cephDpl: &unitinputs.CephDeployMosk,
			inputResources: map[string]runtime.Object{
				"cephblockpools": &unitinputs.OpenstackCephBlockPoolsListReady,
				"cephclients":    &unitinputs.CephClientListOpenstack,
				"configmaps": &corev1.ConfigMapList{
					Items: []corev1.ConfigMap{unitinputs.RookCephMonEndpoints},
				},
				"secrets": &corev1.SecretList{
					Items: []corev1.Secret{
						unitinputs.RookCephMonSecret, unitinputs.OpenstackRgwCredsSecret, unitinputs.RgwSSLCertSecret, unitinputs.RookCephRgwMetricsSecret,
					},
				},
				"cephobjectstoreusers": &unitinputs.CephObjectStoreUserListMetrics,
			},
			lcmConfig: map[string]string{
				"KEEP_INGRESS": "true",
			},
			changed: true,
			expectedResources: map[string]runtime.Object{
				"secrets": &corev1.SecretList{
					Items: []corev1.Secret{
						unitinputs.RookCephMonSecret, unitinputs.OpenstackRgwCredsSecret, unitinputs.RgwSSLCertSecret, unitinputs.RookCephRgwMetricsSecret, unitinputs.ReconcileOpenstackSecret,
					},
				},
			},
		},
		{
			name:    "generate openstack secret - update success",
			cephDpl: &unitinputs.CephDeployMosk,
			inputResources: map[string]runtime.Object{
				"cephblockpools": &unitinputs.OpenstackCephBlockPoolsListReady,
				"cephclients":    &unitinputs.CephClientListOpenstack,
				"configmaps": &corev1.ConfigMapList{
					Items: []corev1.ConfigMap{unitinputs.RookCephMonEndpoints},
				},
				"secrets": &corev1.SecretList{
					Items: []corev1.Secret{
						*unitinputs.CephKeysOpenstackSecretRgwBase.DeepCopy(),
						unitinputs.RookCephMonSecret, unitinputs.OpenstackRgwCredsSecret, unitinputs.RgwSSLCertSecret, unitinputs.RookCephRgwMetricsSecret,
					},
				},
				"cephobjectstoreusers": &unitinputs.CephObjectStoreUserListMetrics,
			},
			lcmConfig: map[string]string{
				"KEEP_INGRESS": "true",
			},
			changed: true,
			expectedResources: map[string]runtime.Object{
				"secrets": &corev1.SecretList{
					Items: []corev1.Secret{
						unitinputs.ReconcileOpenstackSecret, unitinputs.RookCephMonSecret, unitinputs.OpenstackRgwCredsSecret, unitinputs.RgwSSLCertSecret, unitinputs.RookCephRgwMetricsSecret,
					},
				},
			},
		},
		{
			name:    "generate openstack secret - update failed",
			cephDpl: &unitinputs.CephDeployMosk,
			inputResources: map[string]runtime.Object{
				"cephblockpools": &unitinputs.OpenstackCephBlockPoolsListReady,
				"cephclients":    &unitinputs.CephClientListOpenstack,
				"configmaps": &corev1.ConfigMapList{
					Items: []corev1.ConfigMap{unitinputs.RookCephMonEndpoints},
				},
				"secrets": &corev1.SecretList{
					Items: []corev1.Secret{
						*unitinputs.CephKeysOpenstackSecretRgwBase.DeepCopy(),
						unitinputs.RookCephMonSecret, unitinputs.OpenstackRgwCredsSecret, unitinputs.RgwSSLCertSecret, unitinputs.RookCephRgwMetricsSecret,
					},
				},
				"cephobjectstoreusers": &unitinputs.CephObjectStoreUserListMetrics,
			},
			apiErrors:     map[string]error{"update-secrets": errors.New("secret update failed")},
			expectedError: "failed to update openstack-ceph-shared/openstack-ceph-keys secret: secret update failed",
		},
		{
			name:    "generate openstack secret - only labels update",
			cephDpl: &unitinputs.CephDeployMosk,
			inputResources: map[string]runtime.Object{
				"cephblockpools": &unitinputs.OpenstackCephBlockPoolsListReady,
				"cephclients":    &unitinputs.CephClientListOpenstack,
				"configmaps": &corev1.ConfigMapList{
					Items: []corev1.ConfigMap{unitinputs.RookCephMonEndpoints},
				},
				"secrets": &corev1.SecretList{
					Items: []corev1.Secret{
						func() corev1.Secret {
							secret := unitinputs.ReconcileOpenstackSecret.DeepCopy()
							secret.Labels = nil
							return *secret
						}(),
						unitinputs.RookCephMonSecret, unitinputs.OpenstackRgwCredsSecret, unitinputs.RgwSSLCertSecret, unitinputs.RookCephRgwMetricsSecret,
					},
				},
				"cephobjectstoreusers": &unitinputs.CephObjectStoreUserListMetrics,
			},
			lcmConfig: map[string]string{
				"KEEP_INGRESS": "true",
			},
			changed: true,
			expectedResources: map[string]runtime.Object{
				"secrets": &corev1.SecretList{
					Items: []corev1.Secret{
						unitinputs.ReconcileOpenstackSecret, unitinputs.RookCephMonSecret, unitinputs.OpenstackRgwCredsSecret, unitinputs.RgwSSLCertSecret, unitinputs.RookCephRgwMetricsSecret,
					},
				},
			},
		},
		{
			name:    "generate openstack secret - nothing todo",
			cephDpl: &unitinputs.CephDeployMosk,
			inputResources: map[string]runtime.Object{
				"cephblockpools": &unitinputs.OpenstackCephBlockPoolsListReady,
				"cephclients":    &unitinputs.CephClientListOpenstack,
				"configmaps": &corev1.ConfigMapList{
					Items: []corev1.ConfigMap{unitinputs.RookCephMonEndpoints},
				},
				"secrets": &corev1.SecretList{
					Items: []corev1.Secret{
						*unitinputs.ReconcileOpenstackSecret.DeepCopy(),
						unitinputs.RookCephMonSecret, unitinputs.OpenstackRgwCredsSecret, unitinputs.RgwSSLCertSecret, unitinputs.RookCephRgwMetricsSecret,
					},
				},
				"cephobjectstoreusers": &unitinputs.CephObjectStoreUserListMetrics,
			},
			apiErrors: map[string]error{"update-secrets": errors.New("secret update failed")},
		},
	}
	oldCmdFunc := lcmcommon.RunPodCommandWithValidation
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := fakeDeploymentConfig(&deployConfig{cephDpl: test.cephDpl}, test.lcmConfig)
			err := c.castExtensions()
			assert.Nil(t, err)
			faketestclients.FakeReaction(c.api.Kubeclientset.CoreV1(), "get", []string{"configmaps", "secrets"}, test.inputResources, test.apiErrors)
			faketestclients.FakeReaction(c.api.Kubeclientset.CoreV1(), "create", []string{"secrets"}, test.inputResources, test.apiErrors)
			faketestclients.FakeReaction(c.api.Kubeclientset.CoreV1(), "update", []string{"secrets"}, test.inputResources, test.apiErrors)
			faketestclients.FakeReaction(c.api.Kubeclientset.CoreV1(), "get", []string{"secrets"}, test.inputResources, test.apiErrors)
			faketestclients.FakeReaction(c.api.Rookclientset, "get", []string{"cephblockpools", "cephobjectstoreusers"}, test.inputResources, test.apiErrors)
			faketestclients.FakeReaction(c.api.Rookclientset, "list", []string{"cephclients"}, test.inputResources, test.apiErrors)
			test.expectedResources = faketestclients.PrepareExpectedResources(test.inputResources, test.expectedResources)

			lcmcommon.RunPodCommandWithValidation = func(e lcmcommon.ExecConfig) (string, string, error) {
				if test.getAuthKeyError != nil {
					return "", "", test.getAuthKeyError
				}
				cmds := strings.Split(e.Command, ".")
				if strings.HasPrefix(cmds[1], "cinder") {
					return "cinder-key", "", nil
				}
				if strings.HasPrefix(cmds[1], "nova") {
					return "nova-key", "", nil
				}
				if strings.HasPrefix(cmds[1], "glance") {
					return "glance-key", "", nil
				}
				return "fake-keyring", "", nil
			}

			changed, err := c.ensureOpenstackSecret()
			if test.expectedError != "" {
				assert.NotNil(t, err)
				assert.Equal(t, test.expectedError, err.Error())
			}
			assert.Equal(t, test.expectedResources, test.inputResources)
			assert.Equal(t, test.changed, changed)
			faketestclients.CleanupFakeClientReactions(c.api.Kubeclientset.CoreV1())
			faketestclients.CleanupFakeClientReactions(c.api.Rookclientset)
		})
	}
	lcmcommon.RunPodCommandWithValidation = oldCmdFunc
}

func TestGetCephClientAuthKeys(t *testing.T) {
	tests := []struct {
		name               string
		cliCommand         map[string]string
		authNames          map[string]string
		expectedClientKeys map[string]OpenstackClientInfo
		expectedError      string
	}{
		{
			name: "get ceph client auth keys - failed to get auth keys",
			authNames: map[string]string{
				"nova":   "novaN9y4kl7t4vTb",
				"glance": "glanceo1sdvEEqDxxo",
				"cinder": "cinderVMucF_ZXr9Yz",
			},
			expectedError: "some auth keys failed to get: failed to run 'ceph auth get-key client.cinderVMucF_ZXr9Yz' command, failed to run 'ceph auth get-key client.glanceo1sdvEEqDxxo' command, failed to run 'ceph auth get-key client.novaN9y4kl7t4vTb' command",
		},
		{
			name: "openstack clients found no manila - success",
			cliCommand: map[string]string{
				"ceph auth get-key client.novaN9y4kl7t4vTb":   "nova-key",
				"ceph auth get-key client.cinderVMucF_ZXr9Yz": "cinder-key",
				"ceph auth get-key client.glanceo1sdvEEqDxxo": "glance-key",
			},
			authNames: map[string]string{
				"nova":   "novaN9y4kl7t4vTb",
				"glance": "glanceo1sdvEEqDxxo",
				"cinder": "cinderVMucF_ZXr9Yz",
			},
			expectedClientKeys: map[string]OpenstackClientInfo{
				"nova":   {Name: "client.novaN9y4kl7t4vTb", ID: "novaN9y4kl7t4vTb", Keyring: "nova-key"},
				"cinder": {Name: "client.cinderVMucF_ZXr9Yz", ID: "cinderVMucF_ZXr9Yz", Keyring: "cinder-key"},
				"glance": {Name: "client.glanceo1sdvEEqDxxo", ID: "glanceo1sdvEEqDxxo", Keyring: "glance-key"},
			},
		},
		{
			name: "openstack clients found - empty",
			authNames: map[string]string{
				"nova":   "novaN9y4kl7t4vTb",
				"glance": "glanceo1sdvEEqDxxo",
				"cinder": "cinderVMucF_ZXr9Yz",
			},
			cliCommand: map[string]string{
				"ceph auth get-key client.novaN9y4kl7t4vTb":   "",
				"ceph auth get-key client.cinderVMucF_ZXr9Yz": "",
				"ceph auth get-key client.glanceo1sdvEEqDxxo": "",
				"ceph auth get-key client.manilaqnJWCfLAghgC": "",
			},
			expectedError: "some auth keys failed to get: command 'ceph auth get-key client.cinderVMucF_ZXr9Yz' output is empty, command 'ceph auth get-key client.glanceo1sdvEEqDxxo' output is empty, command 'ceph auth get-key client.novaN9y4kl7t4vTb' output is empty",
		},
		{
			name: "openstack clients found with manila - success",
			cliCommand: map[string]string{
				"ceph auth get-key client.novaN9y4kl7t4vTb":   "nova-key",
				"ceph auth get-key client.cinderVMucF_ZXr9Yz": "cinder-key",
				"ceph auth get-key client.glanceo1sdvEEqDxxo": "glance-key",
				"ceph auth get-key client.manilaqnJWCfLAghgC": "manila-key",
			},
			authNames: map[string]string{
				"nova":   "novaN9y4kl7t4vTb",
				"glance": "glanceo1sdvEEqDxxo",
				"cinder": "cinderVMucF_ZXr9Yz",
				"manila": "manilaqnJWCfLAghgC",
			},
			expectedClientKeys: map[string]OpenstackClientInfo{
				"nova":   {Name: "client.novaN9y4kl7t4vTb", ID: "novaN9y4kl7t4vTb", Keyring: "nova-key"},
				"cinder": {Name: "client.cinderVMucF_ZXr9Yz", ID: "cinderVMucF_ZXr9Yz", Keyring: "cinder-key"},
				"glance": {Name: "client.glanceo1sdvEEqDxxo", ID: "glanceo1sdvEEqDxxo", Keyring: "glance-key"},
				"manila": {Name: "client.manilaqnJWCfLAghgC", ID: "manilaqnJWCfLAghgC", Keyring: "manila-key"},
			},
		},
	}
	oldFunc := lcmcommon.RunPodCommandWithValidation
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := fakeDeploymentConfig(nil, nil)
			lcmcommon.RunPodCommandWithValidation = func(e lcmcommon.ExecConfig) (string, string, error) {
				if v, ok := test.cliCommand[e.Command]; ok {
					return v, "", nil
				}
				return "", "", errors.New("command failed")
			}

			actual, err := c.getCephClientAuthKeys(test.authNames)
			if test.expectedError != "" {
				assert.NotNil(t, err)
				assert.Equal(t, test.expectedError, err.Error())
			} else {
				assert.Equal(t, test.expectedClientKeys, actual)
			}
		})
	}
	lcmcommon.RunPodCommandWithValidation = oldFunc
}
