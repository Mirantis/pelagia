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
	"slices"
	"strings"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"

	cephlcmv1alpha1 "github.com/Mirantis/pelagia/v3/pkg/apis/ceph.pelagia.lcm/v1alpha1"
	lcmcommon "github.com/Mirantis/pelagia/v3/pkg/common"
	faketestclients "github.com/Mirantis/pelagia/v3/test/unit/clients"
	unitinputs "github.com/Mirantis/pelagia/v3/test/unit/inputs"
)

func TestUpdateRuntimeParameters(t *testing.T) {
	tests := []struct {
		name            string
		configDump      string
		runtimeConfig   []configOption
		expectedActions map[string]bool
		configError     bool
		expectedError   string
	}{
		{
			name:          "config dump error",
			configDump:    "{||}",
			expectedError: "failed to parse output for command 'ceph config dump --format json': invalid character '|' looking for beginning of object key string",
			runtimeConfig: []configOption{{section: "test", key: "test"}},
		},
		{
			name:       "config runtime updated",
			configDump: unitinputs.CephConfigDumpDefaults,
			runtimeConfig: []configOption{
				{section: "global", key: "osd_max_backfills", value: "32"},
				{section: "osd.10", key: "osd_recovery_max_active", value: "16"},
			},
			expectedActions: map[string]bool{
				"ceph config set global osd_max_backfills 32":       true,
				"ceph config set osd.10 osd_recovery_max_active 16": true,
			},
		},
		{
			name:       "config runtime updated only rgw params",
			configDump: unitinputs.CephConfigDumpOverride,
			runtimeConfig: []configOption{
				{section: "global", key: "osd_max_backfills", value: "32"},
				{section: "global", key: "osd_recovery_max_active", value: "16"},
				{section: "client.rgw.rgw.store.a", key: "rgw_keystone_admin_password", value: "AMTqaDveAp8sWlLtf0fcg6RVjFRXs7FR", secret: true},
				{section: "client.rgw.rgw.store.a", key: "rgw_keystone_barbican_password", value: "AMTqaDveAp8sWlLtf0fcg6RVjFRXs7FR", secret: true},
			},
			expectedActions: map[string]bool{
				"ceph config set client.rgw.rgw.store.a rgw_keystone_admin_password AMTqaDveAp8sWlLtf0fcg6RVjFRXs7FR":    true,
				"ceph config set client.rgw.rgw.store.a rgw_keystone_barbican_password AMTqaDveAp8sWlLtf0fcg6RVjFRXs7FR": true,
			},
		},
		{
			name:       "no config runtime update",
			configDump: unitinputs.CephConfigDumpOverrideWithRgw,
			runtimeConfig: []configOption{
				{section: "global", key: "osd_max_backfills", value: "64"},
				{section: "global", key: "osd_recovery_max_active", value: "16"},
				{section: "client.rgw.rgw.store.a", key: "rgw_keystone_admin_password", value: "AMTqaDveAp8sWlLtf0fcg6RVjFRXs7FR", secret: true},
				{section: "client.rgw.rgw.store.a", key: "rgw_keystone_barbican_password", value: "AMTqaDveAp8sWlLtf0fcg6RVjFRXs7FR", secret: true},
			},
			expectedActions: map[string]bool{},
		},
		{
			name:       "config runtime updated params",
			configDump: unitinputs.CephConfigDumpOverride,
			runtimeConfig: []configOption{
				{section: "global", key: "osd_max_backfills", value: ""},
				{section: "global", key: "osd_recovery_max_active", value: "17"},
			},
			expectedActions: map[string]bool{
				"ceph config set global osd_recovery_max_active 17": true,
				"ceph config rm global osd_max_backfills":           true,
			},
		},
		{
			name:       "config runtime update fail",
			configDump: unitinputs.CephConfigDumpOverride,
			runtimeConfig: []configOption{
				{section: "global", key: "osd_max_backfills", value: "25"},
			},
			configError:   true,
			expectedError: "failed to update '[global] osd_max_backfills' parameter: failed to run command 'ceph config set global osd_max_backfills 25': failed to process parameter",
		},
		{
			name: "config runtime updates when same parameter found but another section",
			configDump: `[{
    "section": "osd.1",
    "name": "osd_max_backfills",
    "value": "32",
    "level": "advanced",
    "can_update_at_runtime": true,
    "mask": ""
}]`,
			runtimeConfig: []configOption{
				{section: "global", key: "osd_max_backfills", value: "25"},
			},
			expectedActions: map[string]bool{
				"ceph config set global osd_max_backfills 25": true,
			},
		},
	}
	actions := map[string]bool{}
	oldTimeFunc := lcmcommon.GetCurrentTimeString
	oldCmdFunc := lcmcommon.RunPodCommandWithValidation
	for idx, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			c := fakeDeploymentConfig(nil, nil)
			actions = map[string]bool{}

			lcmcommon.GetCurrentTimeString = func() string {
				return fmt.Sprintf("time-%d", idx)
			}

			lcmcommon.RunPodCommandWithValidation = func(e lcmcommon.ExecConfig) (string, string, error) {
				if strings.Contains(e.Command, "config dump") {
					return testCase.configDump, "", nil
				}
				if strings.Contains(e.Command, "config set") || strings.Contains(e.Command, "config rm") {
					actions[e.Command] = true
					if testCase.configError {
						return "", "", errors.New("failed to process parameter")
					}
					return "", "", nil
				}
				return "", "", errors.New("cant run ceph cmd: unknown command")
			}

			runtimeUpdated, err := c.updateRuntimeParameters(testCase.runtimeConfig)
			if testCase.expectedError != "" {
				assert.Equal(t, testCase.expectedError, err.Error())
			} else {
				assert.Nil(t, err)
				assert.Equal(t, testCase.expectedActions, actions)
			}
			runtimeUpdateExpected := len(testCase.expectedActions) > 0
			assert.Equal(t, runtimeUpdateExpected, runtimeUpdated)
		})
	}
	// unset global var to avoid intersection
	unsetTimestampsVar()
	lcmcommon.GetCurrentTimeString = oldTimeFunc
	lcmcommon.RunPodCommandWithValidation = oldCmdFunc
}

var rookConfigRgwNoOpenstackNoOverride = `[global]
cluster_network = 127.0.0.0/16
public_network = 192.168.0.0/16

[osd]
osd_class_dir = /usr/lib64/rados-classes

[client.rgw.rgw.store.a]
rgw_dns_name = rook-ceph-rgw-rgw-store.rook-ceph.svc
`

var rookConfigRgwOpenstackNoBarbicanNoOverride = `[global]
cluster_network = 127.0.0.0/16
public_network = 192.168.0.0/16

[osd]
osd_class_dir = /usr/lib64/rados-classes

[client.rgw.rgw.store.a]
rgw_dns_name = rook-ceph-rgw-rgw-store.rook-ceph.svc,rgw-store.openstack.com
rgw_keystone_accepted_admin_roles = 'admin, service'
rgw_keystone_accepted_roles = '_member_, Member, member, swiftoperator'
rgw_keystone_admin_domain = os-domain
rgw_keystone_admin_project = os-project
rgw_keystone_admin_user = auth-user
rgw_keystone_api_version = 3
rgw_keystone_implicit_tenants = true
rgw_keystone_url = https://keystone.openstack.com
`

var rookConfigRgwOpenstackBarbicanNoOverride = `[global]
cluster_network = 127.0.0.0/16
public_network = 192.168.0.0/16

[osd]
osd_class_dir = /usr/lib64/rados-classes

[client.rgw.rgw.store.a]
rgw_barbican_url = https://barbican.openstack.com
rgw_crypt_s3_kms_backend = barbican
rgw_dns_name = rook-ceph-rgw-rgw-store.rook-ceph.svc,rgw-store.openstack.com
rgw_keystone_accepted_admin_roles = 'admin, service'
rgw_keystone_accepted_roles = '_member_, Member, member, swiftoperator'
rgw_keystone_admin_domain = os-domain
rgw_keystone_admin_project = os-project
rgw_keystone_admin_user = auth-user
rgw_keystone_api_version = 3
rgw_keystone_barbican_domain = os-domain
rgw_keystone_barbican_project = os-project
rgw_keystone_barbican_user = auth-user
rgw_keystone_implicit_tenants = true
rgw_keystone_url = https://keystone.openstack.com
`

var defaultRuntimeConfig = []configOption{
	{section: "global", key: "mon_max_pg_per_osd", value: "300"},
	{section: "global", key: "mon_target_pg_per_osd", value: "100"},
	{section: "mon", key: "mon_warn_on_insecure_global_id_reclaim", value: "false"},
	{section: "mon", key: "mon_warn_on_insecure_global_id_reclaim_allowed", value: "false"},
	{section: "osd", key: "bdev_async_discard_threads", value: "1"},
	{section: "osd", key: "bdev_enable_discard", value: "true"},
}

var defaultRgwRuntimeConfig = slices.Insert(defaultRuntimeConfig, 0,
	[]configOption{
		{section: "client.rgw.rgw.store.a", key: "rgw_bucket_quota_ttl", value: "30"},
		{section: "client.rgw.rgw.store.a", key: "rgw_data_log_backing", value: "omap"},
		{section: "client.rgw.rgw.store.a", key: "rgw_max_attr_name_len", value: "64"},
		{section: "client.rgw.rgw.store.a", key: "rgw_max_attr_size", value: "1024"},
		{section: "client.rgw.rgw.store.a", key: "rgw_max_attrs_num_in_req", value: "32"},
		{section: "client.rgw.rgw.store.a", key: "rgw_thread_pool_size", value: "256"},
		{section: "client.rgw.rgw.store.a", key: "rgw_trust_forwarded_https", value: "true"},
		{section: "client.rgw.rgw.store.a", key: "rgw_user_quota_bucket_sync_interval", value: "30"},
		{section: "client.rgw.rgw.store.a", key: "rgw_user_quota_sync_interval", value: "30"},
	}...)

var defaultRgwOpenstackRuntimeConfigNoBarbican = func() []configOption {
	s := slices.Insert(defaultRgwRuntimeConfig, 2,
		[]configOption{
			{section: "client.rgw.rgw.store.a", key: "rgw_enable_usage_log", value: "true"},
			{section: "client.rgw.rgw.store.a", key: "rgw_enforce_swift_acls", value: "true"},
			{section: "client.rgw.rgw.store.a", key: "rgw_keystone_admin_password", value: "auth-password", secret: true},
		}...)
	s = slices.Insert(s, 8,
		[]configOption{
			{section: "client.rgw.rgw.store.a", key: "rgw_s3_auth_use_keystone", value: "true"},
			{section: "client.rgw.rgw.store.a", key: "rgw_swift_account_in_url", value: "true"},
			{section: "client.rgw.rgw.store.a", key: "rgw_swift_versioning_enabled", value: "true"},
		}...)
	s = slices.Insert(s, 13,
		[]configOption{
			{section: "client.rgw.rgw.store.a", key: "rgw_usage_log_flush_threshold", value: "1024"},
			{section: "client.rgw.rgw.store.a", key: "rgw_usage_log_tick_interval", value: "30"},
			{section: "client.rgw.rgw.store.a", key: "rgw_usage_max_shards", value: "32"},
			{section: "client.rgw.rgw.store.a", key: "rgw_usage_max_user_shards", value: "1"},
		}...)
	return s
}()

var defaultRgwOpenstackRuntimeConfigBarbican = func() []configOption {
	s := slices.Clone(defaultRgwOpenstackRuntimeConfigNoBarbican)
	s = slices.Insert(s, 5, configOption{section: "client.rgw.rgw.store.a", key: "rgw_keystone_barbican_password", value: "auth-password", secret: true})
	return s
}()

func TestBuildRookConfig(t *testing.T) {
	tests := []struct {
		name                  string
		cephDpl               *cephlcmv1alpha1.CephDeployment
		lcmConfig             map[string]string
		openstackSecret       *v1.Secret
		currentCephVersion    *lcmcommon.CephVersion
		expectedRookConfig    string
		expectedRuntimeConfig []configOption
		expectedHashConfig    map[string]string
	}{
		{
			name:                  "base rook config - no extra opts",
			cephDpl:               &unitinputs.BaseCephDeployment,
			expectedRookConfig:    unitinputs.BaseRookConfigOverride.Data["config"],
			expectedRuntimeConfig: defaultRuntimeConfig,
			expectedHashConfig: map[string]string{
				"global": unitinputs.BaseRookConfigOverride.Annotations["cephdeployment.lcm.mirantis.com/config-global-hash"],
				"osd":    unitinputs.BaseRookConfigOverride.Annotations["cephdeployment.lcm.mirantis.com/config-osd-hash"],
			},
		},
		{
			name: "base rook config - not allow override network configs",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				cd := unitinputs.BaseCephDeployment.DeepCopy()
				cd.Spec.ExtraOpts = &cephlcmv1alpha1.CephDeploymentExtraOpts{
					StaticCephConfig: map[string]map[string]string{
						"global": {
							"public_network":  "0.0.0.0/0",
							"cluster_network": "0.0.0.0/0",
						},
					},
				}
				return cd
			}(),
			expectedRookConfig:    unitinputs.BaseRookConfigOverride.Data["config"],
			expectedRuntimeConfig: defaultRuntimeConfig,
			expectedHashConfig: map[string]string{
				"global": unitinputs.BaseRookConfigOverride.Annotations["cephdeployment.lcm.mirantis.com/config-global-hash"],
				"osd":    unitinputs.BaseRookConfigOverride.Annotations["cephdeployment.lcm.mirantis.com/config-osd-hash"],
			},
		},
		{
			name:    "base rook config - overriden static from spec, no runtime",
			cephDpl: &unitinputs.CephDeployRookConfigNoRuntime,
			expectedRookConfig: `[global]
cluster_network = 127.0.0.0/16
public_network = 192.168.0.0/16

[mon]
mon_max_pg_per_osd = 400

[osd]
osd_class_dir = /usr/lib64/rados-classes
`,
			expectedRuntimeConfig: defaultRuntimeConfig,
			expectedHashConfig: map[string]string{
				"global": unitinputs.BaseRookConfigOverride.Annotations["cephdeployment.lcm.mirantis.com/config-global-hash"],
				"mon":    "e71a6754b81203c17d7e687f623fa2836883415c31e809dcd6a142d4b3ac6d95",
				"osd":    unitinputs.BaseRookConfigOverride.Annotations["cephdeployment.lcm.mirantis.com/config-osd-hash"],
			},
		},
		{
			name:               "base rook config - overriden runtime from spec, no static",
			cephDpl:            &unitinputs.CephDeployRookConfigRuntimeNoStatic,
			expectedRookConfig: unitinputs.BaseRookConfigOverride.Data["config"],
			expectedRuntimeConfig: []configOption{
				{section: "global", key: "mon_max_pg_per_osd", value: "300"},
				{section: "global", key: "mon_target_pg_per_osd", value: "100"},
				{section: "mon", key: "mon_max_pg_per_osd", value: "400"},
				{section: "mon", key: "mon_warn_on_insecure_global_id_reclaim", value: "false"},
				{section: "mon", key: "mon_warn_on_insecure_global_id_reclaim_allowed", value: "false"},
				{section: "osd", key: "bdev_async_discard_threads", value: "1"},
				{section: "osd", key: "bdev_enable_discard", value: "true"},
				{section: "osd", key: "osd_max_backfills", value: "64"},
				{section: "osd", key: "osd_recovery_max_active", value: "16"},
				{section: "osd", key: "osd_recovery_op_priority", value: "3"},
				{section: "osd", key: "osd_recovery_sleep_hdd", value: "0.000000"},
			},
			expectedHashConfig: map[string]string{
				"global": unitinputs.BaseRookConfigOverride.Annotations["cephdeployment.lcm.mirantis.com/config-global-hash"],
				"osd":    unitinputs.BaseRookConfigOverride.Annotations["cephdeployment.lcm.mirantis.com/config-osd-hash"],
			},
		},
		{
			name:                  "base rgw rook config - no override from spec",
			cephDpl:               &unitinputs.CephDeployObjectStorageCeph,
			expectedRookConfig:    rookConfigRgwNoOpenstackNoOverride,
			expectedRuntimeConfig: defaultRgwRuntimeConfig,
			expectedHashConfig: map[string]string{
				"global":                 unitinputs.BaseRookConfigOverride.Annotations["cephdeployment.lcm.mirantis.com/config-global-hash"],
				"client.rgw.rgw.store.a": "4c1707fbe7a550c18c78ed381bd33c6ebb46113869ffc5caacee61095640965b",
				"osd":                    unitinputs.BaseRookConfigOverride.Annotations["cephdeployment.lcm.mirantis.com/config-osd-hash"],
			},
		},
		{
			name: "base rgw rook config ingress no openstack - override from spec",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				cd := unitinputs.CephDeployNonMoskWithIngress.DeepCopy()
				cd.Spec.ExtraOpts = &cephlcmv1alpha1.CephDeploymentExtraOpts{
					RuntimeCephConfig: map[string]map[string]string{
						"global": {"osd_max_backfills": "24"},
					},
				}
				return cd
			}(),
			lcmConfig: map[string]string{
				"KEEP_INGRESS": "true",
			},
			expectedRookConfig: `[global]
cluster_network = 127.0.0.0/16
public_network = 192.168.0.0/16

[osd]
osd_class_dir = /usr/lib64/rados-classes

[client.rgw.rgw.store.a]
rgw_dns_name = rook-ceph-rgw-rgw-store.rook-ceph.svc,rgw-store.example.com
`,
			expectedRuntimeConfig: slices.Insert(defaultRgwRuntimeConfig, 11, configOption{section: "global", key: "osd_max_backfills", value: "24"}),
			expectedHashConfig: map[string]string{
				"global":                 unitinputs.BaseRookConfigOverride.Annotations["cephdeployment.lcm.mirantis.com/config-global-hash"],
				"osd":                    unitinputs.BaseRookConfigOverride.Annotations["cephdeployment.lcm.mirantis.com/config-osd-hash"],
				"client.rgw.rgw.store.a": "db378cf4854afd80693651393d8ae26487492b991f55e62bfd0564e83f802063",
			},
		},
		{
			name:                  "rook rgw openstack no barbican config - no override from spec",
			cephDpl:               &unitinputs.CephDeployMoskWithoutIngress,
			openstackSecret:       &unitinputs.OpenstackRgwCredsSecretNoBarbican,
			expectedRookConfig:    rookConfigRgwOpenstackNoBarbicanNoOverride,
			expectedRuntimeConfig: defaultRgwOpenstackRuntimeConfigNoBarbican,
			expectedHashConfig: map[string]string{
				"global":                 unitinputs.BaseRookConfigOverride.Annotations["cephdeployment.lcm.mirantis.com/config-global-hash"],
				"osd":                    unitinputs.BaseRookConfigOverride.Annotations["cephdeployment.lcm.mirantis.com/config-osd-hash"],
				"client.rgw.rgw.store.a": "b8b1dae3a4e91d51b4301de7ee816552a4c6d26f4449ba5b88bcea13ed767f9e",
			},
		},
		{
			name:            "rook rgw openstack no barbican config - overriden from spec",
			cephDpl:         &unitinputs.CephDeployMoskWithoutIngressConfigOverride,
			openstackSecret: &unitinputs.OpenstackRgwCredsSecretNoBarbican,
			expectedRookConfig: `[global]
cluster_network = 127.0.0.0/16
public_network = 192.168.0.0/16

[osd]
osd_class_dir = /usr/lib64/rados-classes

[client.rgw.rgw.store.a]
rgw_dns_name = rook-ceph-rgw-rgw-store.rook-ceph.svc,rgw-store.openstack.com
rgw_keystone_accepted_admin_roles = 'admin, service'
rgw_keystone_accepted_roles = '_member_, Member, member, swiftoperator'
rgw_keystone_admin_domain = os-domain
rgw_keystone_admin_project = os-project
rgw_keystone_admin_user = override-user
rgw_keystone_api_version = 3
rgw_keystone_implicit_tenants = true
rgw_keystone_url = https://keystone.openstack.com
`,
			expectedRuntimeConfig: func() []configOption {
				s := slices.Clone(defaultRgwOpenstackRuntimeConfigNoBarbican)
				s[12].value = "false"
				return s
			}(),
			expectedHashConfig: map[string]string{
				"global":                 unitinputs.BaseRookConfigOverride.Annotations["cephdeployment.lcm.mirantis.com/config-global-hash"],
				"osd":                    unitinputs.BaseRookConfigOverride.Annotations["cephdeployment.lcm.mirantis.com/config-osd-hash"],
				"client.rgw.rgw.store.a": "7bdde449aa78ac165e83d44039b56cdebcf175b6248147fd0354f23f6d29ed7d",
			},
		},
		{
			name:                  "rook rgw openstack barbican config - no override from spec",
			cephDpl:               &unitinputs.CephDeployMoskWithoutIngress,
			openstackSecret:       &unitinputs.OpenstackRgwCredsSecret,
			expectedRookConfig:    rookConfigRgwOpenstackBarbicanNoOverride,
			expectedRuntimeConfig: defaultRgwOpenstackRuntimeConfigBarbican,
			expectedHashConfig: map[string]string{
				"global":                 unitinputs.BaseRookConfigOverride.Annotations["cephdeployment.lcm.mirantis.com/config-global-hash"],
				"osd":                    unitinputs.BaseRookConfigOverride.Annotations["cephdeployment.lcm.mirantis.com/config-osd-hash"],
				"client.rgw.rgw.store.a": "c07a39372af1b1f603ee1d62df3ecde0db79014b9f43b6e926fd6ec8ba86737f",
			},
		},
		{
			name: "rook rgw ingress openstack barbican config - overriden from spec",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				cephDpl := unitinputs.CephDeployMoskWithoutIngressRookConfigOverrideBarbican.DeepCopy()
				cephDpl.Spec.IngressConfig = unitinputs.CephIngressConfig.DeepCopy()
				return cephDpl
			}(),
			lcmConfig: map[string]string{
				"KEEP_INGRESS": "true",
			},
			openstackSecret: &unitinputs.OpenstackRgwCredsSecret,
			expectedRookConfig: `[global]
cluster_network = 127.0.0.0/16
public_network = 192.168.0.0/16

[osd]
osd_class_dir = /usr/lib64/rados-classes

[client.rgw.rgw.store.a]
rgw_barbican_url = https://barbican.openstack.com
rgw_crypt_s3_kms_backend = barbican
rgw_dns_name = rgw-store.ms2.wxlsd.com
rgw_keystone_accepted_admin_roles = 'admin, service'
rgw_keystone_accepted_roles = '_member_, Member, member, swiftoperator'
rgw_keystone_admin_domain = os-domain
rgw_keystone_admin_project = os-project
rgw_keystone_admin_user = auth-user
rgw_keystone_api_version = 3
rgw_keystone_barbican_domain = os-domain
rgw_keystone_barbican_project = os-project
rgw_keystone_barbican_user = override-user
rgw_keystone_implicit_tenants = true
rgw_keystone_url = https://keystone.openstack.com
`,
			expectedRuntimeConfig: func() []configOption {
				s := slices.Clone(defaultRgwOpenstackRuntimeConfigBarbican)
				s[3].value = "false"
				s[18].value = "10"
				s[20].value = "400"
				return s
			}(),
			expectedHashConfig: map[string]string{
				"global":                 unitinputs.BaseRookConfigOverride.Annotations["cephdeployment.lcm.mirantis.com/config-global-hash"],
				"osd":                    unitinputs.BaseRookConfigOverride.Annotations["cephdeployment.lcm.mirantis.com/config-osd-hash"],
				"client.rgw.rgw.store.a": "d48b4edd655c83ad25fcff565895a69240ec064e166e6d7e0d1dd32a9e953ce6",
			},
		},
		{
			name:    "rook rgw openstack ingress - no override from spec",
			cephDpl: &unitinputs.CephDeployMosk,
			lcmConfig: map[string]string{
				"KEEP_INGRESS": "true",
			},
			openstackSecret: &unitinputs.OpenstackRgwCredsSecretNoBarbican,
			expectedRookConfig: `[global]
cluster_network = 127.0.0.0/16
public_network = 192.168.0.0/16

[osd]
osd_class_dir = /usr/lib64/rados-classes

[client.rgw.rgw.store.a]
rgw_dns_name = rook-ceph-rgw-rgw-store.rook-ceph.svc,rgw-store.test
rgw_keystone_accepted_admin_roles = 'admin, service'
rgw_keystone_accepted_roles = '_member_, Member, member, swiftoperator'
rgw_keystone_admin_domain = os-domain
rgw_keystone_admin_project = os-project
rgw_keystone_admin_user = auth-user
rgw_keystone_api_version = 3
rgw_keystone_implicit_tenants = true
rgw_keystone_url = https://keystone.openstack.com
`,
			expectedRuntimeConfig: defaultRgwOpenstackRuntimeConfigNoBarbican,
			expectedHashConfig: map[string]string{
				"global":                 unitinputs.BaseRookConfigOverride.Annotations["cephdeployment.lcm.mirantis.com/config-global-hash"],
				"osd":                    unitinputs.BaseRookConfigOverride.Annotations["cephdeployment.lcm.mirantis.com/config-osd-hash"],
				"client.rgw.rgw.store.a": "b2e21dc135b2a1986c992afaf6089ec46a44122b7da1cb3b2ae66774d6c72b16",
			},
		},
		{
			name: "base rgw ingress config - rgw dns overriden from spec",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				cephDpl := unitinputs.CephDeployObjectStorageCeph.DeepCopy()
				cephDpl.Spec.IngressConfig = unitinputs.CephIngressConfig.DeepCopy()
				cephDpl.Spec.ExtraOpts = &cephlcmv1alpha1.CephDeploymentExtraOpts{
					StaticCephConfig: map[string]map[string]string{
						"client.rgw.rgw.store.a": {"rgw_dns_name": "rgw-store.fromspec.com"},
					},
				}
				return cephDpl
			}(),
			lcmConfig: map[string]string{
				"KEEP_INGRESS": "true",
			},
			expectedRookConfig: `[global]
cluster_network = 127.0.0.0/16
public_network = 192.168.0.0/16

[osd]
osd_class_dir = /usr/lib64/rados-classes

[client.rgw.rgw.store.a]
rgw_dns_name = rgw-store.fromspec.com
`,
			expectedRuntimeConfig: defaultRgwRuntimeConfig,
			expectedHashConfig: map[string]string{
				"global":                 unitinputs.BaseRookConfigOverride.Annotations["cephdeployment.lcm.mirantis.com/config-global-hash"],
				"osd":                    unitinputs.BaseRookConfigOverride.Annotations["cephdeployment.lcm.mirantis.com/config-osd-hash"],
				"client.rgw.rgw.store.a": "20f01b5d3c4ffd12d84f473da96d0b892af92b947728b80193f80a2f11f9ed68",
			},
		},
		{
			name: "rook config with extra opts and section override",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				cephDpl := unitinputs.BaseCephDeployment.DeepCopy()
				cephDpl.Spec.ExtraOpts = &cephlcmv1alpha1.CephDeploymentExtraOpts{
					StaticCephConfig: map[string]map[string]string{
						"mon.b": {"mon_max_pg_per_osd": "300"},
						"mgr":   {"mgr/prometheus/scrape_interval": "100"},
						"osd":   {"custom-key": "custom-value"},
					},
					RuntimeCephConfig: map[string]map[string]string{
						"osd.14": {"osd_journal_size": "6250"},
						"osd.12": {"osd_journal_size": "5120"},
					},
				}
				return cephDpl
			}(),
			expectedRookConfig: `[global]
cluster_network = 127.0.0.0/16
public_network = 192.168.0.0/16

[mon.b]
mon_max_pg_per_osd = 300

[mgr]
mgr/prometheus/scrape_interval = 100

[osd]
custom_key = custom-value
osd_class_dir = /usr/lib64/rados-classes
`,
			expectedRuntimeConfig: slices.Insert(defaultRuntimeConfig, 4,
				[]configOption{
					{section: "osd.12", key: "osd_journal_size", value: "5120"},
					{section: "osd.14", key: "osd_journal_size", value: "6250"},
				}...),
			expectedHashConfig: map[string]string{
				"global": unitinputs.BaseRookConfigOverride.Annotations["cephdeployment.lcm.mirantis.com/config-global-hash"],
				"mgr":    "5da5b6cafcade8cd83b5dd9708c6669e65b516637abaf45e47212cdc320ca412",
				"mon.b":  "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
				"osd":    "55e7a75341863f4e108e5c5859f5a8d219b4ba59ad03e858b4b37cad35d6f8a1",
			},
		},
		{
			name:               "rook config with rgw sync daemon",
			cephDpl:            &unitinputs.MultisiteRgwWithSyncDaemon,
			expectedRookConfig: rookConfigRgwNoOpenstackNoOverride,
			expectedRuntimeConfig: slices.Insert(defaultRgwRuntimeConfig, 9,
				[]configOption{
					{section: "client.rgw.rgw.store.sync.a", key: "rgw_bucket_quota_ttl", value: "30"},
					{section: "client.rgw.rgw.store.sync.a", key: "rgw_data_log_backing", value: "omap"},
					{section: "client.rgw.rgw.store.sync.a", key: "rgw_max_attr_name_len", value: "64"},
					{section: "client.rgw.rgw.store.sync.a", key: "rgw_max_attr_size", value: "1024"},
					{section: "client.rgw.rgw.store.sync.a", key: "rgw_max_attrs_num_in_req", value: "32"},
					{section: "client.rgw.rgw.store.sync.a", key: "rgw_thread_pool_size", value: "256"},
					{section: "client.rgw.rgw.store.sync.a", key: "rgw_trust_forwarded_https", value: "true"},
					{section: "client.rgw.rgw.store.sync.a", key: "rgw_user_quota_bucket_sync_interval", value: "30"},
					{section: "client.rgw.rgw.store.sync.a", key: "rgw_user_quota_sync_interval", value: "30"},
				}...),
			expectedHashConfig: map[string]string{
				"global":                 unitinputs.BaseRookConfigOverride.Annotations["cephdeployment.lcm.mirantis.com/config-global-hash"],
				"osd":                    unitinputs.BaseRookConfigOverride.Annotations["cephdeployment.lcm.mirantis.com/config-osd-hash"],
				"client.rgw.rgw.store.a": "4c1707fbe7a550c18c78ed381bd33c6ebb46113869ffc5caacee61095640965b",
			},
		},
		{
			name: "rook rgw openstack - custom ingress hostname",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				cephDpl := unitinputs.CephDeployObjectStorageCeph.DeepCopy()
				cephDpl.Spec.ObjectStorage.Rgws[0].ServedByIngress = true
				cephDpl.Spec.IngressConfig = unitinputs.CephIngressConfig.DeepCopy()
				cephDpl.Spec.IngressConfig.TLSConfig.Domain = "fromspec.com"
				cephDpl.Spec.IngressConfig.TLSConfig.Hostname = "rgw-store"
				cephDpl.Spec.IngressConfig.Annotations = nil
				cephDpl.Spec.IngressConfig.ControllerClassName = ""
				return cephDpl
			}(),
			lcmConfig: map[string]string{
				"KEEP_INGRESS": "true",
			},
			expectedRookConfig: `[global]
cluster_network = 127.0.0.0/16
public_network = 192.168.0.0/16

[osd]
osd_class_dir = /usr/lib64/rados-classes

[client.rgw.rgw.store.a]
rgw_dns_name = rook-ceph-rgw-rgw-store.rook-ceph.svc,rgw-store.fromspec.com
`,
			expectedRuntimeConfig: defaultRgwRuntimeConfig,
			expectedHashConfig: map[string]string{
				"global":                 unitinputs.BaseRookConfigOverride.Annotations["cephdeployment.lcm.mirantis.com/config-global-hash"],
				"osd":                    unitinputs.BaseRookConfigOverride.Annotations["cephdeployment.lcm.mirantis.com/config-osd-hash"],
				"client.rgw.rgw.store.a": "cc4eba6e1a3aa029ae53c6affc84a8a571697209a954d2e7056dae415a1fff80",
			},
		},
		{
			name:                  "base rook config - rgw hosting field set",
			cephDpl:               &unitinputs.CephDeployMoskWithRgwHosting,
			openstackSecret:       &unitinputs.OpenstackRgwCredsSecret,
			expectedRookConfig:    strings.Replace(rookConfigRgwOpenstackBarbicanNoOverride, "rgw_dns_name = rook-ceph-rgw-rgw-store.rook-ceph.svc,rgw-store.openstack.com\n", "", 1),
			expectedRuntimeConfig: defaultRgwOpenstackRuntimeConfigBarbican,
			expectedHashConfig: map[string]string{
				"global":                 unitinputs.BaseRookConfigOverride.Annotations["cephdeployment.lcm.mirantis.com/config-global-hash"],
				"osd":                    unitinputs.BaseRookConfigOverride.Annotations["cephdeployment.lcm.mirantis.com/config-osd-hash"],
				"client.rgw.rgw.store.a": "2b3a28d581d90f4e40dcc10bd5aea0f74d5b50485af51590849abf9b471e05aa",
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := fakeDeploymentConfig(&deployConfig{cephDpl: test.cephDpl}, test.lcmConfig)
			if test.currentCephVersion == nil {
				test.currentCephVersion = lcmcommon.LatestRelease
			}
			c.cdConfig.currentCephVersion = test.currentCephVersion
			err := c.castExtensions()
			assert.Nil(t, err)

			inputResources := map[string]runtime.Object{"secrets": &unitinputs.SecretsListEmpty}
			if test.openstackSecret != nil {
				inputResources["secrets"] = &v1.SecretList{Items: []v1.Secret{*test.openstackSecret}}
			}
			faketestclients.FakeReaction(c.api.Kubeclientset.CoreV1(), "get", []string{"secrets"}, inputResources, nil)

			config, runtime, hashes, err := c.buildCephConfig()
			assert.Equal(t, test.expectedRookConfig, config)
			assert.Equal(t, test.expectedRuntimeConfig, runtime)
			assert.Equal(t, test.expectedHashConfig, hashes)
			assert.Nil(t, err)
			faketestclients.CleanupFakeClientReactions(c.api.Kubeclientset.CoreV1())
		})
	}
	// unset global var to avoid intersection
	unsetTimestampsVar()
}

func TestEnsureCephConfig(t *testing.T) {
	osdUpdatedAnnotation := fmt.Sprintf(cephConfigParametersUpdateTimestampLabel, "osd")
	globalUpdatedAnnotation := fmt.Sprintf(cephConfigParametersUpdateTimestampLabel, "global")
	tests := []struct {
		name               string
		cephDpl            *cephlcmv1alpha1.CephDeployment
		openstackSecret    *v1.Secret
		rookCm             *v1.ConfigMap
		apiErrors          map[string]error
		extraLcmConfig     map[string]string
		cephClusterPresent bool
		configDump         string
		configRuntimeCmds  map[string]bool
		unsetGlobalVar     bool
		configUpdated      bool
		expectedTimestamps updateTimestamps
		expectedResources  map[string]runtime.Object
		expectedError      string
	}{
		{
			name:          "failed to build ceph and runtime config",
			cephDpl:       &unitinputs.CephDeployMosk,
			apiErrors:     map[string]error{"get-secrets-openstack-rgw-creds": errors.New("failed to get openstack rgw secret")},
			expectedError: "failed to prepare ceph config map: failed to get openstack rgw secret",
			expectedTimestamps: updateTimestamps{
				cephConfigMap: map[string]string{},
				rgwSSLCert:    map[string]string{},
			},
		},
		{
			name:          "failed to get rook ceph override config map",
			cephDpl:       &unitinputs.BaseCephDeployment,
			apiErrors:     map[string]error{"get-configmaps-rook-config-override": errors.New("failed to get configmap rook-config-override")},
			expectedError: "failed to get configmap rook-config-override",
			expectedTimestamps: updateTimestamps{
				cephConfigMap: map[string]string{},
				rgwSSLCert:    map[string]string{},
			},
		},
		{
			name:          "failed to create config map",
			cephDpl:       &unitinputs.BaseCephDeployment,
			apiErrors:     map[string]error{"create-configmaps-rook-config-override": errors.New("failed to create configmap rook-config-override")},
			expectedError: "failed to create configmap rook-config-override",
			expectedTimestamps: updateTimestamps{
				cephConfigMap: map[string]string{},
				rgwSSLCert:    map[string]string{},
			},
		},
		{
			name:    "config map is created",
			cephDpl: &unitinputs.BaseCephDeployment,
			expectedTimestamps: updateTimestamps{
				cephConfigMap: map[string]string{
					"global": "time-3",
					"osd":    "time-3",
				},
				rgwSSLCert: map[string]string{},
			},
			expectedResources: map[string]runtime.Object{
				"configmaps": &v1.ConfigMapList{
					Items: []v1.ConfigMap{
						func() v1.ConfigMap {
							newCM := unitinputs.BaseRookConfigOverride.DeepCopy()
							newCM.Annotations[cephConfigMapUpdateTimestampLabel] = "time-3"
							newCM.Annotations[osdUpdatedAnnotation] = "time-3"
							newCM.Annotations[globalUpdatedAnnotation] = "time-3"
							return *newCM
						}(),
					},
				},
			},
			configUpdated: true,
		},
		{
			name:    "config map is present and no changes",
			cephDpl: &unitinputs.BaseCephDeployment,
			rookCm: func() *v1.ConfigMap {
				newCM := unitinputs.BaseRookConfigOverride.DeepCopy()
				newCM.Annotations[cephConfigMapUpdateTimestampLabel] = "time-3"
				newCM.Annotations[osdUpdatedAnnotation] = "time-3"
				newCM.Annotations[globalUpdatedAnnotation] = "time-3"
				return newCM
			}(),
			configDump:         unitinputs.CephConfigDumpDefaults,
			cephClusterPresent: true,
			expectedTimestamps: updateTimestamps{
				cephConfigMap: map[string]string{
					"global": "time-3",
					"osd":    "time-3",
				},
				rgwSSLCert: map[string]string{},
			},
		},
		{
			name:    "config map is present, no changes but global vars not set before",
			cephDpl: &unitinputs.BaseCephDeployment,
			rookCm: func() *v1.ConfigMap {
				newCM := unitinputs.BaseRookConfigOverride.DeepCopy()
				newCM.Annotations[cephConfigMapUpdateTimestampLabel] = "some-time"
				newCM.Annotations[osdUpdatedAnnotation] = "some-time"
				newCM.Annotations[globalUpdatedAnnotation] = "some-time"
				return newCM
			}(),
			configDump:         unitinputs.CephConfigDumpDefaults,
			cephClusterPresent: true,
			unsetGlobalVar:     true,
			expectedTimestamps: updateTimestamps{
				cephConfigMap: map[string]string{
					"global": "some-time",
					"osd":    "some-time",
				},
				rgwSSLCert: map[string]string{},
			},
		},
		{
			name:               "failed to update config map",
			cephDpl:            &unitinputs.BaseCephDeployment,
			rookCm:             unitinputs.EmptyRookConfigOverride.DeepCopy(),
			configDump:         unitinputs.CephConfigDumpDefaults,
			cephClusterPresent: true,
			expectedTimestamps: updateTimestamps{
				cephConfigMap: map[string]string{
					"global": "some-time",
					"osd":    "some-time",
				},
				rgwSSLCert: map[string]string{},
			},
			apiErrors:     map[string]error{"update-configmaps-rook-config-override": errors.New("failed to update configmap rook-config-override")},
			expectedError: "failed to update configmap rook-config-override",
		},
		{
			name:               "config map only is updated",
			cephDpl:            &unitinputs.BaseCephDeployment,
			rookCm:             unitinputs.EmptyRookConfigOverride.DeepCopy(),
			configDump:         unitinputs.CephConfigDumpDefaults,
			cephClusterPresent: true,
			configUpdated:      true,
			expectedTimestamps: updateTimestamps{
				cephConfigMap: map[string]string{
					"global": "time-7",
					"osd":    "time-7",
				},
				rgwSSLCert: map[string]string{},
			},
			expectedResources: map[string]runtime.Object{
				"configmaps": &v1.ConfigMapList{Items: []v1.ConfigMap{
					func() v1.ConfigMap {
						newCM := unitinputs.BaseRookConfigOverride.DeepCopy()
						newCM.Annotations[cephConfigMapUpdateTimestampLabel] = "time-7"
						newCM.Annotations[osdUpdatedAnnotation] = "time-7"
						newCM.Annotations[globalUpdatedAnnotation] = "time-7"
						return *newCM
					}(),
				}},
			},
		},
		{
			name:    "config map runtime mon and osd params are updated",
			cephDpl: &unitinputs.CephDeployRookConfigRuntimeNoStatic,
			rookCm: func() *v1.ConfigMap {
				newCM := unitinputs.BaseRookConfigOverride.DeepCopy()
				newCM.Annotations[cephConfigMapUpdateTimestampLabel] = "time-7"
				newCM.Annotations[osdUpdatedAnnotation] = "time-7"
				newCM.Annotations[globalUpdatedAnnotation] = "time-7"
				return newCM
			}(),
			configDump:         unitinputs.CephConfigDumpDefaults,
			cephClusterPresent: true,
			configRuntimeCmds: map[string]bool{
				"ceph config set mon mon_max_pg_per_osd 400":          false,
				"ceph config set osd osd_max_backfills 64":            false,
				"ceph config set osd osd_recovery_max_active 16":      false,
				"ceph config set osd osd_recovery_op_priority 3":      false,
				"ceph config set osd osd_recovery_sleep_hdd 0.000000": false,
			},
			configUpdated: true,
			expectedTimestamps: updateTimestamps{
				cephConfigMap: map[string]string{
					"global": "time-7",
					"osd":    "time-7",
				},
				rgwSSLCert: map[string]string{},
			},
			expectedResources: map[string]runtime.Object{
				"configmaps": &v1.ConfigMapList{Items: []v1.ConfigMap{
					func() v1.ConfigMap {
						newCM := unitinputs.BaseRookConfigOverride.DeepCopy()
						newCM.Annotations[cephConfigMapUpdateTimestampLabel] = "time-7"
						newCM.Annotations[osdUpdatedAnnotation] = "time-7"
						newCM.Annotations[globalUpdatedAnnotation] = "time-7"
						newCM.Data["config"] = unitinputs.BaseRookConfigOverride.Data["config"]
						newCM.Data["runtime"] = "global|mon_max_pg_per_osd = 300\nglobal|mon_target_pg_per_osd = 100\nmon|mon_max_pg_per_osd = 400\nmon|mon_warn_on_insecure_global_id_reclaim = false\nmon|mon_warn_on_insecure_global_id_reclaim_allowed = false\nosd|bdev_async_discard_threads = 1\nosd|bdev_enable_discard = true\nosd|osd_max_backfills = 64\nosd|osd_recovery_max_active = 16\nosd|osd_recovery_op_priority = 3\nosd|osd_recovery_sleep_hdd = 0.000000\n"
						return *newCM
					}(),
				}},
			},
		},
		{
			name:    "config map and runtime updated - rgw params has been added",
			cephDpl: &unitinputs.CephDeployMoskWithoutIngress,
			rookCm: func() *v1.ConfigMap {
				newCM := unitinputs.BaseRookConfigOverride.DeepCopy()
				newCM.Annotations[cephConfigMapUpdateTimestampLabel] = "time-8"
				newCM.Annotations[osdUpdatedAnnotation] = "time-7"
				newCM.Annotations[globalUpdatedAnnotation] = "time-8"
				return newCM
			}(),
			configDump:      unitinputs.CephConfigDumpOverride,
			openstackSecret: &unitinputs.OpenstackRgwCredsSecretNoBarbican,
			configRuntimeCmds: map[string]bool{
				"ceph config set client.rgw.rgw.store.a rgw_keystone_admin_password auth-password": false,
				"ceph config set client.rgw.rgw.store.a rgw_bucket_quota_ttl 30":                   false,
				"ceph config set client.rgw.rgw.store.a rgw_data_log_backing omap":                 false,
				"ceph config set client.rgw.rgw.store.a rgw_max_attr_name_len 64":                  false,
				"ceph config set client.rgw.rgw.store.a rgw_max_attr_size 1024":                    false,
				"ceph config set client.rgw.rgw.store.a rgw_max_attrs_num_in_req 32":               false,
				"ceph config set client.rgw.rgw.store.a rgw_thread_pool_size 256":                  false,
				"ceph config set client.rgw.rgw.store.a rgw_trust_forwarded_https true":            false,
				"ceph config set client.rgw.rgw.store.a rgw_user_quota_bucket_sync_interval 30":    false,
				"ceph config set client.rgw.rgw.store.a rgw_user_quota_sync_interval 30":           false,
				"ceph config set client.rgw.rgw.store.a rgw_enable_usage_log true":                 false,
				"ceph config set client.rgw.rgw.store.a rgw_enforce_swift_acls true":               false,
				"ceph config set client.rgw.rgw.store.a rgw_s3_auth_use_keystone true":             false,
				"ceph config set client.rgw.rgw.store.a rgw_swift_account_in_url true":             false,
				"ceph config set client.rgw.rgw.store.a rgw_swift_versioning_enabled true":         false,
				"ceph config set client.rgw.rgw.store.a rgw_usage_log_flush_threshold 1024":        false,
				"ceph config set client.rgw.rgw.store.a rgw_usage_log_tick_interval 30":            false,
				"ceph config set client.rgw.rgw.store.a rgw_usage_max_shards 32":                   false,
				"ceph config set client.rgw.rgw.store.a rgw_usage_max_user_shards 1":               false,
			},
			cephClusterPresent: true,
			expectedTimestamps: updateTimestamps{
				cephConfigMap: map[string]string{
					"global":                 "time-8",
					"osd":                    "time-7",
					"client.rgw.rgw.store.a": "time-9",
				},
				rgwSSLCert: map[string]string{},
			},
			configUpdated: true,
			expectedResources: map[string]runtime.Object{
				"configmaps": &v1.ConfigMapList{Items: []v1.ConfigMap{
					func() v1.ConfigMap {
						newCM := unitinputs.BaseRookConfigOverride.DeepCopy()
						newCM.Data["config"] = rookConfigRgwOpenstackNoBarbicanNoOverride
						newCM.Data["runtime"] = "client.rgw.rgw.store.a|rgw_bucket_quota_ttl = 30\nclient.rgw.rgw.store.a|rgw_data_log_backing = omap\nclient.rgw.rgw.store.a|rgw_enable_usage_log = true\nclient.rgw.rgw.store.a|rgw_enforce_swift_acls = true\nclient.rgw.rgw.store.a|rgw_keystone_admin_password = *\nclient.rgw.rgw.store.a|rgw_max_attr_name_len = 64\nclient.rgw.rgw.store.a|rgw_max_attr_size = 1024\nclient.rgw.rgw.store.a|rgw_max_attrs_num_in_req = 32\nclient.rgw.rgw.store.a|rgw_s3_auth_use_keystone = true\nclient.rgw.rgw.store.a|rgw_swift_account_in_url = true\nclient.rgw.rgw.store.a|rgw_swift_versioning_enabled = true\nclient.rgw.rgw.store.a|rgw_thread_pool_size = 256\nclient.rgw.rgw.store.a|rgw_trust_forwarded_https = true\nclient.rgw.rgw.store.a|rgw_usage_log_flush_threshold = 1024\nclient.rgw.rgw.store.a|rgw_usage_log_tick_interval = 30\nclient.rgw.rgw.store.a|rgw_usage_max_shards = 32\nclient.rgw.rgw.store.a|rgw_usage_max_user_shards = 1\nclient.rgw.rgw.store.a|rgw_user_quota_bucket_sync_interval = 30\nclient.rgw.rgw.store.a|rgw_user_quota_sync_interval = 30\nglobal|mon_max_pg_per_osd = 300\nglobal|mon_target_pg_per_osd = 100\nmon|mon_warn_on_insecure_global_id_reclaim = false\nmon|mon_warn_on_insecure_global_id_reclaim_allowed = false\nosd|bdev_async_discard_threads = 1\nosd|bdev_enable_discard = true\n"
						newCM.Annotations[cephConfigMapUpdateTimestampLabel] = "time-9"
						newCM.Annotations[osdUpdatedAnnotation] = "time-7"
						newCM.Annotations[globalUpdatedAnnotation] = "time-8"
						newCM.Annotations["cephdeployment.lcm.mirantis.com/config-client.rgw.rgw.store.a-updated"] = "time-9"
						newCM.Annotations["cephdeployment.lcm.mirantis.com/config-client.rgw.rgw.store.a-hash"] = "b8b1dae3a4e91d51b4301de7ee816552a4c6d26f4449ba5b88bcea13ed767f9e"
						return *newCM
					}(),
				}},
			},
		},
		{
			name:    "no timestamps var set and only passwords is updated",
			cephDpl: &unitinputs.CephDeployMoskWithoutIngress,
			rookCm: func() *v1.ConfigMap {
				newCM := unitinputs.BaseRookConfigOverride.DeepCopy()
				newCM.Data["config"] = rookConfigRgwOpenstackNoBarbicanNoOverride
				newCM.Data["runtime"] = "client.rgw.rgw.store.a|rgw_keystone_admin_password = *\nosd|bdev_async_discard_threads = 1\nosd|bdev_enable_discard = true\n"
				newCM.Annotations[cephConfigMapUpdateTimestampLabel] = "time-9"
				newCM.Annotations[osdUpdatedAnnotation] = "time-7"
				newCM.Annotations[globalUpdatedAnnotation] = "time-8"
				newCM.Annotations["cephdeployment.lcm.mirantis.com/config-client.rgw.rgw.store.a-updated"] = "time-9"
				newCM.Annotations["cephdeployment.lcm.mirantis.com/config-client.rgw.rgw.store.a-hash"] = "c8761bc0ae63593c4bdeb9309fea60abf6dea624f92b18d2dd398c742bdd1750"
				return newCM
			}(),
			configDump:     unitinputs.CephConfigDumpOverrideWithRgw,
			unsetGlobalVar: true,
			openstackSecret: func() *v1.Secret {
				sc := unitinputs.OpenstackRgwCredsSecretNoBarbican.DeepCopy()
				sc.Data["password"] = []byte("new-auth-password")
				return sc
			}(),
			configRuntimeCmds: map[string]bool{
				"ceph config set client.rgw.rgw.store.a rgw_keystone_admin_password new-auth-password": false,
			},
			cephClusterPresent: true,
			expectedTimestamps: updateTimestamps{
				cephConfigMap: map[string]string{
					"global":                 "time-8",
					"osd":                    "time-7",
					"client.rgw.rgw.store.a": "time-10",
				},
				rgwSSLCert: map[string]string{},
			},
			configUpdated: true,
			expectedResources: map[string]runtime.Object{
				"configmaps": &v1.ConfigMapList{Items: []v1.ConfigMap{
					func() v1.ConfigMap {
						newCM := unitinputs.BaseRookConfigOverride.DeepCopy()
						newCM.Data["config"] = rookConfigRgwOpenstackNoBarbicanNoOverride
						newCM.Data["runtime"] = "client.rgw.rgw.store.a|rgw_bucket_quota_ttl = 30\nclient.rgw.rgw.store.a|rgw_data_log_backing = omap\nclient.rgw.rgw.store.a|rgw_enable_usage_log = true\nclient.rgw.rgw.store.a|rgw_enforce_swift_acls = true\nclient.rgw.rgw.store.a|rgw_keystone_admin_password = *\nclient.rgw.rgw.store.a|rgw_max_attr_name_len = 64\nclient.rgw.rgw.store.a|rgw_max_attr_size = 1024\nclient.rgw.rgw.store.a|rgw_max_attrs_num_in_req = 32\nclient.rgw.rgw.store.a|rgw_s3_auth_use_keystone = true\nclient.rgw.rgw.store.a|rgw_swift_account_in_url = true\nclient.rgw.rgw.store.a|rgw_swift_versioning_enabled = true\nclient.rgw.rgw.store.a|rgw_thread_pool_size = 256\nclient.rgw.rgw.store.a|rgw_trust_forwarded_https = true\nclient.rgw.rgw.store.a|rgw_usage_log_flush_threshold = 1024\nclient.rgw.rgw.store.a|rgw_usage_log_tick_interval = 30\nclient.rgw.rgw.store.a|rgw_usage_max_shards = 32\nclient.rgw.rgw.store.a|rgw_usage_max_user_shards = 1\nclient.rgw.rgw.store.a|rgw_user_quota_bucket_sync_interval = 30\nclient.rgw.rgw.store.a|rgw_user_quota_sync_interval = 30\nglobal|mon_max_pg_per_osd = 300\nglobal|mon_target_pg_per_osd = 100\nmon|mon_warn_on_insecure_global_id_reclaim = false\nmon|mon_warn_on_insecure_global_id_reclaim_allowed = false\nosd|bdev_async_discard_threads = 1\nosd|bdev_enable_discard = true\n"
						newCM.Annotations[cephConfigMapUpdateTimestampLabel] = "time-10"
						newCM.Annotations[osdUpdatedAnnotation] = "time-7"
						newCM.Annotations[globalUpdatedAnnotation] = "time-8"
						newCM.Annotations["cephdeployment.lcm.mirantis.com/config-client.rgw.rgw.store.a-updated"] = "time-10"
						newCM.Annotations["cephdeployment.lcm.mirantis.com/config-client.rgw.rgw.store.a-hash"] = "ddc515ea217b41b7cda84c1121d8501d9a4139771ac8684d312ce1b31f2a6379"
						return *newCM
					}(),
				}},
			},
		},
		{
			name:    "config map and all runtime params are updated - rgw params removed",
			cephDpl: &unitinputs.BaseCephDeployment,
			rookCm: func() *v1.ConfigMap {
				newCM := unitinputs.BaseRookConfigOverride.DeepCopy()
				newCM.Data["config"] = rookConfigRgwOpenstackNoBarbicanNoOverride
				newCM.Data["runtime"] = "client.rgw.rgw.store.a|rgw_bucket_quota_ttl = 30\nclient.rgw.rgw.store.a|rgw_data_log_backing = omap\nclient.rgw.rgw.store.a|rgw_enable_usage_log = true\nclient.rgw.rgw.store.a|rgw_enforce_swift_acls = true\nclient.rgw.rgw.store.a|rgw_keystone_admin_password = *\nclient.rgw.rgw.store.a|rgw_max_attr_name_len = 64\nclient.rgw.rgw.store.a|rgw_max_attr_size = 1024\nclient.rgw.rgw.store.a|rgw_max_attrs_num_in_req = 32\nclient.rgw.rgw.store.a|rgw_s3_auth_use_keystone = true\nclient.rgw.rgw.store.a|rgw_swift_account_in_url = true\nclient.rgw.rgw.store.a|rgw_swift_versioning_enabled = true\nclient.rgw.rgw.store.a|rgw_thread_pool_size = 256\nclient.rgw.rgw.store.a|rgw_trust_forwarded_https = true\nclient.rgw.rgw.store.a|rgw_usage_log_flush_threshold = 1024\nclient.rgw.rgw.store.a|rgw_usage_log_tick_interval = 30\nclient.rgw.rgw.store.a|rgw_usage_max_shards = 32\nclient.rgw.rgw.store.a|rgw_usage_max_user_shards = 1\nclient.rgw.rgw.store.a|rgw_user_quota_bucket_sync_interval = 30\nclient.rgw.rgw.store.a|rgw_user_quota_sync_interval = 30\nglobal|mon_max_pg_per_osd = 300\nglobal|mon_target_pg_per_osd = 100\nmon|mon_warn_on_insecure_global_id_reclaim = false\nmon|mon_warn_on_insecure_global_id_reclaim_allowed = false\nosd|bdev_async_discard_threads = 1\nosd|bdev_enable_discard = true\n"
				newCM.Annotations[cephConfigMapUpdateTimestampLabel] = "time-10"
				newCM.Annotations[osdUpdatedAnnotation] = "time-7"
				newCM.Annotations[globalUpdatedAnnotation] = "time-8"
				newCM.Annotations["cephdeployment.lcm.mirantis.com/config-client.rgw.rgw.store.a-updated"] = "time-10"
				newCM.Annotations["cephdeployment.lcm.mirantis.com/config-client.rgw.rgw.store.a-hash"] = "ddc515ea217b41b7cda84c1121d8501d9a4139771ac8684d312ce1b31f2a6379"
				return newCM
			}(),
			configDump: unitinputs.CephConfigDumpOverrideWithRgw,
			configRuntimeCmds: map[string]bool{
				"ceph config rm client.rgw.rgw.store.a rgw_keystone_admin_password":         false,
				"ceph config rm client.rgw.rgw.store.a rgw_bucket_quota_ttl":                false,
				"ceph config rm client.rgw.rgw.store.a rgw_data_log_backing":                false,
				"ceph config rm client.rgw.rgw.store.a rgw_max_attr_name_len":               false,
				"ceph config rm client.rgw.rgw.store.a rgw_max_attr_size":                   false,
				"ceph config rm client.rgw.rgw.store.a rgw_max_attrs_num_in_req":            false,
				"ceph config rm client.rgw.rgw.store.a rgw_thread_pool_size":                false,
				"ceph config rm client.rgw.rgw.store.a rgw_trust_forwarded_https":           false,
				"ceph config rm client.rgw.rgw.store.a rgw_user_quota_bucket_sync_interval": false,
				"ceph config rm client.rgw.rgw.store.a rgw_user_quota_sync_interval":        false,
				"ceph config rm client.rgw.rgw.store.a rgw_enable_usage_log":                false,
				"ceph config rm client.rgw.rgw.store.a rgw_enforce_swift_acls":              false,
				"ceph config rm client.rgw.rgw.store.a rgw_s3_auth_use_keystone":            false,
				"ceph config rm client.rgw.rgw.store.a rgw_swift_account_in_url":            false,
				"ceph config rm client.rgw.rgw.store.a rgw_swift_versioning_enabled":        false,
				"ceph config rm client.rgw.rgw.store.a rgw_usage_log_flush_threshold":       false,
				"ceph config rm client.rgw.rgw.store.a rgw_usage_log_tick_interval":         false,
				"ceph config rm client.rgw.rgw.store.a rgw_usage_max_shards":                false,
				"ceph config rm client.rgw.rgw.store.a rgw_usage_max_user_shards":           false,
			},
			cephClusterPresent: true,
			expectedTimestamps: updateTimestamps{
				cephConfigMap: map[string]string{
					"global": "time-8",
					"osd":    "time-7",
				},
				rgwSSLCert: map[string]string{},
			},
			configUpdated: true,
			expectedResources: map[string]runtime.Object{
				"configmaps": &v1.ConfigMapList{Items: []v1.ConfigMap{
					func() v1.ConfigMap {
						newCM := unitinputs.BaseRookConfigOverride.DeepCopy()
						newCM.Annotations[cephConfigMapUpdateTimestampLabel] = "time-11"
						newCM.Annotations[osdUpdatedAnnotation] = "time-7"
						newCM.Annotations[globalUpdatedAnnotation] = "time-8"
						return *newCM
					}(),
				}},
			},
		},
		{
			name: "config map updated - mds params added",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				cd := unitinputs.BaseCephDeployment.DeepCopy()
				cd.Spec.RookConfig = map[string]string{
					"mds|mds_cache_memory_limit":         "10G",
					"mds.testfs|mds_cache_memory_limit":  "5G",
					"mds.testfs2|mds_cache_memory_limit": "6G",
				}
				cd.Spec.ExtraOpts = &cephlcmv1alpha1.CephDeploymentExtraOpts{
					RuntimeCephConfig: map[string]map[string]string{
						"mds.testfs":  {"mds_cache_memory_limit": "5G"},
						"mds.testfs2": {"mds_cache_memory_limit": "6G"},
					},
					StaticCephConfig: map[string]map[string]string{
						"mds": {"mds_cache_memory_limit": "10G"},
					},
				}
				return cd
			}(),
			rookCm: func() *v1.ConfigMap {
				newCM := unitinputs.BaseRookConfigOverride.DeepCopy()
				newCM.Annotations[cephConfigMapUpdateTimestampLabel] = "time-9"
				newCM.Annotations[osdUpdatedAnnotation] = "time-7"
				newCM.Annotations[globalUpdatedAnnotation] = "time-11"
				return newCM
			}(),
			configDump: unitinputs.CephConfigDumpDefaults,
			configRuntimeCmds: map[string]bool{
				"ceph config set mds.testfs mds_cache_memory_limit 5G":  false,
				"ceph config set mds.testfs2 mds_cache_memory_limit 6G": false,
			},
			cephClusterPresent: true,
			expectedTimestamps: updateTimestamps{
				cephConfigMap: map[string]string{
					"global": "time-11",
					"osd":    "time-7",
					"mds":    "time-12",
				},
				rgwSSLCert: map[string]string{},
			},
			configUpdated: true,
			expectedResources: map[string]runtime.Object{
				"configmaps": &v1.ConfigMapList{Items: []v1.ConfigMap{
					func() v1.ConfigMap {
						newCM := unitinputs.BaseRookConfigOverride.DeepCopy()
						newCM.Data["config"] = "[global]\ncluster_network = 127.0.0.0/16\npublic_network = 192.168.0.0/16\n\n[osd]\nosd_class_dir = /usr/lib64/rados-classes\n\n[mds]\nmds_cache_memory_limit = 10G\n"
						newCM.Data["runtime"] = "global|mon_max_pg_per_osd = 300\nglobal|mon_target_pg_per_osd = 100\nmds.testfs2|mds_cache_memory_limit = 6G\nmds.testfs|mds_cache_memory_limit = 5G\nmon|mon_warn_on_insecure_global_id_reclaim = false\nmon|mon_warn_on_insecure_global_id_reclaim_allowed = false\nosd|bdev_async_discard_threads = 1\nosd|bdev_enable_discard = true\n"
						newCM.Annotations[cephConfigMapUpdateTimestampLabel] = "time-12"
						newCM.Annotations[osdUpdatedAnnotation] = "time-7"
						newCM.Annotations[globalUpdatedAnnotation] = "time-11"
						newCM.Annotations["cephdeployment.lcm.mirantis.com/config-global-hash"] = "b6e2eff5c6df81944e8a54f26a29d01f1764a78786893c22f03acf87d5cf3256"
						newCM.Annotations["cephdeployment.lcm.mirantis.com/config-mds-updated"] = "time-12"
						newCM.Annotations["cephdeployment.lcm.mirantis.com/config-mds-hash"] = "159347878ba9b5eff291a3a76176581f76756de08c2835077546e9c63c469208"
						return *newCM
					}(),
				}},
			},
		},
		{
			name:    "config map updated - mds params removed",
			cephDpl: unitinputs.BaseCephDeployment.DeepCopy(),
			rookCm: func() *v1.ConfigMap {
				newCM := unitinputs.BaseRookConfigOverride.DeepCopy()
				newCM.Data["config"] = "[global]\ncluster_network = 127.0.0.0/16\npublic_network = 192.168.0.0/16\n\n[osd]\nosd_class_dir = /usr/lib64/rados-classes\n\n[mds]\nmds_cache_memory_limit = 10G\n"
				newCM.Data["runtime"] = "global|mon_max_pg_per_osd = 300\nglobal|mon_target_pg_per_osd = 100\nmds.testfs2|mds_cache_memory_limit = 6G\nmds.testfs|mds_cache_memory_limit = 5G\nmon|mon_warn_on_insecure_global_id_reclaim = false\nmon|mon_warn_on_insecure_global_id_reclaim_allowed = false\nosd|bdev_async_discard_threads = 1\nosd|bdev_enable_discard = true\n"
				newCM.Annotations[cephConfigMapUpdateTimestampLabel] = "time-12"
				newCM.Annotations[osdUpdatedAnnotation] = "time-7"
				newCM.Annotations[globalUpdatedAnnotation] = "time-11"
				newCM.Annotations["cephdeployment.lcm.mirantis.com/config-global-hash"] = "b6e2eff5c6df81944e8a54f26a29d01f1764a78786893c22f03acf87d5cf3256"
				newCM.Annotations["cephdeployment.lcm.mirantis.com/config-mds-updated"] = "time-12"
				newCM.Annotations["cephdeployment.lcm.mirantis.com/config-mds-hash"] = "159347878ba9b5eff291a3a76176581f76756de08c2835077546e9c63c469208"
				return newCM
			}(),
			configDump: unitinputs.CephConfigDumpDefaults,
			configRuntimeCmds: map[string]bool{
				"ceph config rm mds.testfs mds_cache_memory_limit":  false,
				"ceph config rm mds.testfs2 mds_cache_memory_limit": false,
			},
			cephClusterPresent: true,
			expectedTimestamps: updateTimestamps{
				cephConfigMap: map[string]string{
					"global": "time-11",
					"osd":    "time-7",
				},
				rgwSSLCert: map[string]string{},
			},
			configUpdated: true,
			expectedResources: map[string]runtime.Object{
				"configmaps": &v1.ConfigMapList{Items: []v1.ConfigMap{
					func() v1.ConfigMap {
						newCM := unitinputs.BaseRookConfigOverride.DeepCopy()
						newCM.Annotations[cephConfigMapUpdateTimestampLabel] = "time-13"
						newCM.Annotations[osdUpdatedAnnotation] = "time-7"
						newCM.Annotations[globalUpdatedAnnotation] = "time-11"
						return *newCM
					}(),
				}},
			},
		},
		{
			name:    "config map - failed to check shared namespace",
			cephDpl: &unitinputs.CephDeployMoskWithoutIngress,
			extraLcmConfig: map[string]string{
				"DEPLOYMENT_OPENSTACK_CEPH_SHARED_NAMESPACE": "",
			},
			expectedError: "failed to prepare ceph config map: CephRGW object storage 'rgw-store' has specified for Openstack usage, but Pelagia lcmconfig has no var 'DEPLOYMENT_OPENSTACK_CEPH_SHARED_NAMESPACE' set",
			expectedTimestamps: updateTimestamps{
				cephConfigMap: map[string]string{
					"global": "time-11",
					"osd":    "time-7",
				},
				rgwSSLCert: map[string]string{},
			},
		},
	}

	oldTimeFunc := lcmcommon.GetCurrentTimeString
	oldCmdFunc := lcmcommon.RunPodCommandWithValidation
	for idx, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if test.unsetGlobalVar {
				unsetTimestampsVar()
			}

			lcmcommon.GetCurrentTimeString = func() string {
				return fmt.Sprintf("time-%d", idx)
			}

			c := fakeDeploymentConfig(&deployConfig{cephDpl: test.cephDpl}, test.extraLcmConfig)
			c.cdConfig.currentCephVersion = lcmcommon.LatestRelease
			err := c.castExtensions()
			assert.Nil(t, err)
			inputResources := map[string]runtime.Object{
				"deployments": &appsv1.DeploymentList{Items: []appsv1.Deployment{*unitinputs.ToolBoxDeploymentReady}},
				"secrets":     unitinputs.SecretsListEmpty.DeepCopy(),
				"configmaps":  unitinputs.ConfigMapListEmpty.DeepCopy(),
			}
			if test.openstackSecret != nil {
				inputResources["secrets"] = &v1.SecretList{Items: []v1.Secret{*test.openstackSecret}}
			}
			if test.rookCm != nil {
				inputResources["configmaps"] = &v1.ConfigMapList{Items: []v1.ConfigMap{*test.rookCm}}
			}
			faketestclients.FakeReaction(c.api.Kubeclientset.CoreV1(), "get", []string{"configmaps", "secrets"}, inputResources, test.apiErrors)
			faketestclients.FakeReaction(c.api.Kubeclientset.AppsV1(), "get", []string{"deployments"}, inputResources, test.apiErrors)
			faketestclients.FakeReaction(c.api.Kubeclientset.CoreV1(), "create", []string{"configmaps"}, inputResources, test.apiErrors)
			faketestclients.FakeReaction(c.api.Kubeclientset.CoreV1(), "update", []string{"configmaps"}, inputResources, test.apiErrors)
			test.expectedResources = faketestclients.PrepareExpectedResources(inputResources, test.expectedResources)

			lcmcommon.RunPodCommandWithValidation = func(e lcmcommon.ExecConfig) (string, string, error) {
				if test.cephClusterPresent {
					if strings.Contains(e.Command, "config dump") {
						return test.configDump, "", nil
					}
					if _, ok := test.configRuntimeCmds[e.Command]; ok {
						test.configRuntimeCmds[e.Command] = true
						return "", "", nil
					}
				}
				return "", "", errors.New("cant run ceph cmd: unknown command: " + e.Command)
			}

			configUpdated, err := c.ensureCephConfig(test.cephClusterPresent)
			if test.expectedError != "" {
				assert.NotNil(t, err)
				assert.Equal(t, test.expectedError, err.Error())
			} else {
				assert.Nil(t, err)
			}
			assert.Equal(t, test.expectedTimestamps, resourceUpdateTimestamps)
			assert.Equal(t, test.configUpdated, configUpdated)
			assert.Equal(t, test.expectedResources, inputResources)
			for k, v := range test.configRuntimeCmds {
				if !v {
					assert.Fail(t, fmt.Sprintf("command '%s' is not executed", k))
				}
			}
			faketestclients.CleanupFakeClientReactions(c.api.Kubeclientset.CoreV1())
			faketestclients.CleanupFakeClientReactions(c.api.Kubeclientset.AppsV1())
		})

	}
	// unset global var to avoid intersection
	unsetTimestampsVar()
	lcmcommon.GetCurrentTimeString = oldTimeFunc
	lcmcommon.RunPodCommandWithValidation = oldCmdFunc
}
