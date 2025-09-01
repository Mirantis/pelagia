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
	"strings"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"

	cephlcmv1alpha1 "github.com/Mirantis/pelagia/pkg/apis/ceph.pelagia.lcm/v1alpha1"
	lcmcommon "github.com/Mirantis/pelagia/pkg/common"
	faketestclients "github.com/Mirantis/pelagia/test/unit/clients"
	unitinputs "github.com/Mirantis/pelagia/test/unit/inputs"
)

func TestUpdateRuntimeParameters(t *testing.T) {
	tests := []struct {
		name            string
		configDump      string
		runtimeConfig   map[string]string
		expectedActions map[string]bool
		configError     bool
		expectedOsdTime string
		expectedRgwTime string
		expectedError   string
	}{
		{
			name:          "config dump error",
			configDump:    "{||}",
			expectedError: "failed to parse output for command 'ceph config dump --format json': invalid character '|' looking for beginning of object key string",
			runtimeConfig: map[string]string{"test": "test"},
		},
		{
			name:       "config runtime updated",
			configDump: unitinputs.CephConfigDumpDefaults,
			runtimeConfig: map[string]string{
				"global|osd_max_backfills":       "32",
				"osd.10|osd_recovery_max_active": "16",
			},
			expectedOsdTime: "time-1",
			expectedActions: map[string]bool{
				"ceph config set global osd_max_backfills 32":       true,
				"ceph config set osd.10 osd_recovery_max_active 16": true,
			},
		},
		{
			name:       "config runtime updated only rgw params",
			configDump: unitinputs.CephConfigDumpOverride,
			runtimeConfig: map[string]string{
				"global|osd_max_backfills":                              "32",
				"global|osd_recovery_max_active":                        "16",
				"client.rgw.rgw.store.a|rgw_keystone_admin_password":    "AMTqaDveAp8sWlLtf0fcg6RVjFRXs7FR",
				"client.rgw.rgw.store.a|rgw_keystone_barbican_password": "AMTqaDveAp8sWlLtf0fcg6RVjFRXs7FR",
			},
			expectedOsdTime: "time-1",
			expectedRgwTime: "time-2",
			expectedActions: map[string]bool{
				"ceph config set client.rgw.rgw.store.a rgw_keystone_admin_password AMTqaDveAp8sWlLtf0fcg6RVjFRXs7FR":    true,
				"ceph config set client.rgw.rgw.store.a rgw_keystone_barbican_password AMTqaDveAp8sWlLtf0fcg6RVjFRXs7FR": true,
			},
		},
		{
			name:       "no config runtime update",
			configDump: unitinputs.CephConfigDumpOverrideWithRgw,
			runtimeConfig: map[string]string{
				"global|osd_max_backfills":                              "64",
				"global|osd_recovery_max_active":                        "16",
				"client.rgw.rgw.store.a|rgw_keystone_admin_password":    "AMTqaDveAp8sWlLtf0fcg6RVjFRXs7FR",
				"client.rgw.rgw.store.a|rgw_keystone_barbican_password": "AMTqaDveAp8sWlLtf0fcg6RVjFRXs7FR",
			},
			expectedOsdTime: "time-1",
			expectedRgwTime: "time-2",
			expectedActions: map[string]bool{},
		},
		{
			name:       "config runtime updated only osd params, no section for removing",
			configDump: unitinputs.CephConfigDumpOverride,
			runtimeConfig: map[string]string{
				"global|osd_max_backfills":       "",
				"global|osd_recovery_max_active": "17",
			},
			expectedOsdTime: "time-4",
			expectedRgwTime: "time-2",
			expectedActions: map[string]bool{
				"ceph config set global osd_recovery_max_active 17": true,
				"ceph config rm global osd_max_backfills":           true,
			},
		},
		{
			name:       "config runtime update fail",
			configDump: unitinputs.CephConfigDumpOverride,
			runtimeConfig: map[string]string{
				"global|osd_max_backfills": "25",
			},
			configError:     true,
			expectedError:   "failed to update 'osd_max_backfills' parameter: failed to run command 'ceph config set global osd_max_backfills 25': failed to process parameter",
			expectedOsdTime: "time-4",
			expectedRgwTime: "time-2",
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
			runtimeConfig: map[string]string{
				"global|osd_max_backfills": "25",
			},
			expectedOsdTime: "time-6",
			expectedRgwTime: "time-2",
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
			assert.Equal(t, testCase.expectedRgwTime, resourceUpdateTimestamps.rgwRuntimeParams)
			assert.Equal(t, testCase.expectedOsdTime, resourceUpdateTimestamps.osdRuntimeParams)
			runtimeUpdateExpected := len(testCase.expectedActions) > 0
			assert.Equal(t, runtimeUpdateExpected, runtimeUpdated)
		})
	}
	// unset global var to avoid intersection
	unsetTimestampsVar()
	lcmcommon.GetCurrentTimeString = oldTimeFunc
	lcmcommon.RunPodCommandWithValidation = oldCmdFunc
}

var rookConfigNoRgwNoOpenstackNoOverrideMultus = `[global]
mon_max_pg_per_osd = 300
mon_target_pg_per_osd = 100

[mon]
mon_warn_on_insecure_global_id_reclaim = false
mon_warn_on_insecure_global_id_reclaim_allowed = false

[osd]
osd_class_dir = /usr/lib64/rados-classes
`

var rookConfigNoRgwNoOpenstackOverrideWithSection = `[global]
cluster_network = 127.0.0.0/16
public_network = 192.168.0.0/16
mon_max_pg_per_osd = 300
mon_target_pg_per_osd = 100

[mon]
mon_health_to_clog = true
mon_warn_on_insecure_global_id_reclaim = false
mon_warn_on_insecure_global_id_reclaim_allowed = false

[mgr]
mgr/prometheus/scrape_interval = 100

[mon.b]
mon_max_pg_per_osd = 300

[osd]
osd_class_dir = /usr/lib64/rados-classes
`

var rookConfigNoRgwNoOpenstackOverride = `[global]
cluster_network = 127.0.0.0/16
public_network = 192.168.0.0/16
mon_max_pg_per_osd = 400
mon_target_pg_per_osd = 100

[mon]
mon_warn_on_insecure_global_id_reclaim = false
mon_warn_on_insecure_global_id_reclaim_allowed = false

[osd]
osd_class_dir = /usr/lib64/rados-classes
`

var rookConfigRgwNoOpenstackNoOverride = `[global]
cluster_network = 127.0.0.0/16
public_network = 192.168.0.0/16
mon_max_pg_per_osd = 300
mon_target_pg_per_osd = 100

[mon]
mon_warn_on_insecure_global_id_reclaim = false
mon_warn_on_insecure_global_id_reclaim_allowed = false

[client.rgw.rgw.store.a]
rgw_bucket_quota_ttl = 30
rgw_data_log_backing = omap
rgw_dns_name = rook-ceph-rgw-rgw-store.rook-ceph.svc
rgw_max_attr_name_len = 64
rgw_max_attr_size = 1024
rgw_max_attrs_num_in_req = 32
rgw_thread_pool_size = 256
rgw_trust_forwarded_https = true
rgw_user_quota_bucket_sync_interval = 30
rgw_user_quota_sync_interval = 30

[osd]
osd_class_dir = /usr/lib64/rados-classes
`

var rookConfigRgwSyncDaemonNoOpenstackNoOverride = `[global]
cluster_network = 127.0.0.0/16
public_network = 192.168.0.0/16
mon_max_pg_per_osd = 300
mon_target_pg_per_osd = 100

[mon]
mon_warn_on_insecure_global_id_reclaim = false
mon_warn_on_insecure_global_id_reclaim_allowed = false

[client.rgw.rgw.store.a]
rgw_bucket_quota_ttl = 30
rgw_data_log_backing = omap
rgw_dns_name = rook-ceph-rgw-rgw-store.rook-ceph.svc
rgw_max_attr_name_len = 64
rgw_max_attr_size = 1024
rgw_max_attrs_num_in_req = 32
rgw_thread_pool_size = 256
rgw_trust_forwarded_https = true
rgw_user_quota_bucket_sync_interval = 30
rgw_user_quota_sync_interval = 30

[client.rgw.rgw.store.sync.a]
rgw_bucket_quota_ttl = 30
rgw_data_log_backing = omap
rgw_max_attr_name_len = 64
rgw_max_attr_size = 1024
rgw_max_attrs_num_in_req = 32
rgw_thread_pool_size = 256
rgw_trust_forwarded_https = true
rgw_user_quota_bucket_sync_interval = 30
rgw_user_quota_sync_interval = 30

[osd]
osd_class_dir = /usr/lib64/rados-classes
`

var rookConfigRgwIngressNoOpenstackNoOverride = `[global]
cluster_network = 127.0.0.0/16
public_network = 192.168.0.0/16
mon_max_pg_per_osd = 300
mon_target_pg_per_osd = 100

[mon]
mon_warn_on_insecure_global_id_reclaim = false
mon_warn_on_insecure_global_id_reclaim_allowed = false

[client.rgw.rgw.store.a]
rgw_bucket_quota_ttl = 30
rgw_data_log_backing = omap
rgw_dns_name = rgw-store.example.com
rgw_max_attr_name_len = 64
rgw_max_attr_size = 1024
rgw_max_attrs_num_in_req = 32
rgw_thread_pool_size = 256
rgw_trust_forwarded_https = true
rgw_user_quota_bucket_sync_interval = 30
rgw_user_quota_sync_interval = 30

[osd]
osd_class_dir = /usr/lib64/rados-classes
`

var rookConfigRgwIngressNoOpenstackOverride = `[global]
cluster_network = 127.0.0.0/16
public_network = 192.168.0.0/16
mon_max_pg_per_osd = 300
mon_target_pg_per_osd = 100

[mon]
mon_warn_on_insecure_global_id_reclaim = false
mon_warn_on_insecure_global_id_reclaim_allowed = false

[client.rgw.rgw.store.a]
rgw_bucket_quota_ttl = 30
rgw_data_log_backing = omap
rgw_dns_name = rgw-store.fromspec.com
rgw_max_attr_name_len = 64
rgw_max_attr_size = 1024
rgw_max_attrs_num_in_req = 32
rgw_thread_pool_size = 256
rgw_trust_forwarded_https = true
rgw_user_quota_bucket_sync_interval = 30
rgw_user_quota_sync_interval = 30

[osd]
osd_class_dir = /usr/lib64/rados-classes
`

var rookConfigRgwOpenstackNoBarbicanNoOverride = `[global]
cluster_network = 127.0.0.0/16
public_network = 192.168.0.0/16
mon_max_pg_per_osd = 300
mon_target_pg_per_osd = 100

[mon]
mon_warn_on_insecure_global_id_reclaim = false
mon_warn_on_insecure_global_id_reclaim_allowed = false

[client.rgw.rgw.store.a]
rgw_bucket_quota_ttl = 30
rgw_data_log_backing = omap
rgw_dns_name = rgw-store.openstack.com
rgw_enable_usage_log = true
rgw_enforce_swift_acls = true
rgw_keystone_accepted_admin_roles = 'admin, service'
rgw_keystone_accepted_roles = '_member_, Member, member, swiftoperator'
rgw_keystone_admin_domain = os-domain
rgw_keystone_admin_project = os-project
rgw_keystone_admin_user = auth-user
rgw_keystone_api_version = 3
rgw_keystone_implicit_tenants = true
rgw_keystone_url = https://keystone.openstack.com
rgw_max_attr_name_len = 64
rgw_max_attr_size = 1024
rgw_max_attrs_num_in_req = 32
rgw_s3_auth_use_keystone = true
rgw_swift_account_in_url = true
rgw_swift_versioning_enabled = true
rgw_thread_pool_size = 256
rgw_trust_forwarded_https = true
rgw_usage_log_flush_threshold = 1024
rgw_usage_log_tick_interval = 30
rgw_usage_max_shards = 32
rgw_usage_max_user_shards = 1
rgw_user_quota_bucket_sync_interval = 30
rgw_user_quota_sync_interval = 30

[osd]
osd_class_dir = /usr/lib64/rados-classes
`

var rookConfigRgwIngressOpenstackNoBarbicanNoOverride = `[global]
cluster_network = 127.0.0.0/16
public_network = 192.168.0.0/16
mon_max_pg_per_osd = 300
mon_target_pg_per_osd = 100

[mon]
mon_warn_on_insecure_global_id_reclaim = false
mon_warn_on_insecure_global_id_reclaim_allowed = false

[client.rgw.rgw.store.a]
rgw_bucket_quota_ttl = 30
rgw_data_log_backing = omap
rgw_dns_name = rgw-store.test
rgw_enable_usage_log = true
rgw_enforce_swift_acls = true
rgw_keystone_accepted_admin_roles = 'admin, service'
rgw_keystone_accepted_roles = '_member_, Member, member, swiftoperator'
rgw_keystone_admin_domain = os-domain
rgw_keystone_admin_project = os-project
rgw_keystone_admin_user = auth-user
rgw_keystone_api_version = 3
rgw_keystone_implicit_tenants = true
rgw_keystone_url = https://keystone.openstack.com
rgw_max_attr_name_len = 64
rgw_max_attr_size = 1024
rgw_max_attrs_num_in_req = 32
rgw_s3_auth_use_keystone = true
rgw_swift_account_in_url = true
rgw_swift_versioning_enabled = true
rgw_thread_pool_size = 256
rgw_trust_forwarded_https = true
rgw_usage_log_flush_threshold = 1024
rgw_usage_log_tick_interval = 30
rgw_usage_max_shards = 32
rgw_usage_max_user_shards = 1
rgw_user_quota_bucket_sync_interval = 30
rgw_user_quota_sync_interval = 30

[osd]
osd_class_dir = /usr/lib64/rados-classes
`

var rookConfigRgwOpenstackBarbicanNoOverride = `[global]
cluster_network = 127.0.0.0/16
public_network = 192.168.0.0/16
mon_max_pg_per_osd = 300
mon_target_pg_per_osd = 100

[mon]
mon_warn_on_insecure_global_id_reclaim = false
mon_warn_on_insecure_global_id_reclaim_allowed = false

[client.rgw.rgw.store.a]
rgw_barbican_url = https://barbican.openstack.com
rgw_bucket_quota_ttl = 30
rgw_crypt_s3_kms_backend = barbican
rgw_data_log_backing = omap
rgw_dns_name = rgw-store.openstack.com
rgw_enable_usage_log = true
rgw_enforce_swift_acls = true
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
rgw_max_attr_name_len = 64
rgw_max_attr_size = 1024
rgw_max_attrs_num_in_req = 32
rgw_s3_auth_use_keystone = true
rgw_swift_account_in_url = true
rgw_swift_versioning_enabled = true
rgw_thread_pool_size = 256
rgw_trust_forwarded_https = true
rgw_usage_log_flush_threshold = 1024
rgw_usage_log_tick_interval = 30
rgw_usage_max_shards = 32
rgw_usage_max_user_shards = 1
rgw_user_quota_bucket_sync_interval = 30
rgw_user_quota_sync_interval = 30

[osd]
osd_class_dir = /usr/lib64/rados-classes
`

var rookConfigRgwOpenstackBarbicanOverride = `[global]
cluster_network = 127.0.0.0/16
public_network = 192.168.0.0/16
mon_max_pg_per_osd = 400
mon_target_pg_per_osd = 100

[mon]
mon_warn_on_insecure_global_id_reclaim = false
mon_warn_on_insecure_global_id_reclaim_allowed = false

[client.rgw.rgw.store.a]
rgw_barbican_url = https://barbican.openstack.com
rgw_bucket_quota_ttl = 30
rgw_crypt_s3_kms_backend = barbican
rgw_data_log_backing = omap
rgw_dns_name = rgw-store.ms2.wxlsd.com
rgw_enable_usage_log = true
rgw_enforce_swift_acls = false
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
rgw_max_attr_name_len = 64
rgw_max_attr_size = 1024
rgw_max_attrs_num_in_req = 32
rgw_s3_auth_use_keystone = true
rgw_swift_account_in_url = true
rgw_swift_versioning_enabled = true
rgw_thread_pool_size = 256
rgw_trust_forwarded_https = true
rgw_usage_log_flush_threshold = 1024
rgw_usage_log_tick_interval = 30
rgw_usage_max_shards = 32
rgw_usage_max_user_shards = 1
rgw_user_quota_bucket_sync_interval = 10
rgw_user_quota_sync_interval = 30

[osd]
osd_class_dir = /usr/lib64/rados-classes
`

var rookConfigRgwOpenstackNoBarbicanOverride = `[global]
cluster_network = 127.0.0.0/16
public_network = 192.168.0.0/16
mon_max_pg_per_osd = 300
mon_target_pg_per_osd = 100

[mon]
mon_warn_on_insecure_global_id_reclaim = false
mon_warn_on_insecure_global_id_reclaim_allowed = false

[client.rgw.rgw.store.a]
rgw_bucket_quota_ttl = 30
rgw_data_log_backing = omap
rgw_dns_name = rgw-store.openstack.com
rgw_enable_usage_log = true
rgw_enforce_swift_acls = true
rgw_keystone_accepted_admin_roles = 'admin, service'
rgw_keystone_accepted_roles = '_member_, Member, member, swiftoperator'
rgw_keystone_admin_domain = os-domain
rgw_keystone_admin_project = os-project
rgw_keystone_admin_user = override-user
rgw_keystone_api_version = 3
rgw_keystone_implicit_tenants = true
rgw_keystone_url = https://keystone.openstack.com
rgw_max_attr_name_len = 64
rgw_max_attr_size = 1024
rgw_max_attrs_num_in_req = 32
rgw_s3_auth_use_keystone = true
rgw_swift_account_in_url = true
rgw_swift_versioning_enabled = true
rgw_thread_pool_size = 256
rgw_trust_forwarded_https = false
rgw_usage_log_flush_threshold = 1024
rgw_usage_log_tick_interval = 30
rgw_usage_max_shards = 32
rgw_usage_max_user_shards = 1
rgw_user_quota_bucket_sync_interval = 30
rgw_user_quota_sync_interval = 30

[osd]
osd_class_dir = /usr/lib64/rados-classes
`

var rookConfigRgwOpenstackRgwPoolThreadSizeOverride = `[global]
cluster_network = 127.0.0.0/16
public_network = 192.168.0.0/16
mon_max_pg_per_osd = 300
mon_target_pg_per_osd = 100

[mon]
mon_warn_on_insecure_global_id_reclaim = false
mon_warn_on_insecure_global_id_reclaim_allowed = false

[client.rgw.rgw.store.a]
rgw_bucket_quota_ttl = 30
rgw_data_log_backing = omap
rgw_dns_name = rgw-store.openstack.com
rgw_enable_usage_log = true
rgw_enforce_swift_acls = true
rgw_keystone_accepted_admin_roles = 'admin, service'
rgw_keystone_accepted_roles = '_member_, Member, member, swiftoperator'
rgw_keystone_admin_domain = os-domain
rgw_keystone_admin_project = os-project
rgw_keystone_admin_user = override-user
rgw_keystone_api_version = 3
rgw_keystone_implicit_tenants = true
rgw_keystone_url = https://keystone.openstack.com
rgw_max_attr_name_len = 64
rgw_max_attr_size = 1024
rgw_max_attrs_num_in_req = 32
rgw_s3_auth_use_keystone = true
rgw_swift_account_in_url = true
rgw_swift_versioning_enabled = true
rgw_thread_pool_size = 100
rgw_trust_forwarded_https = false
rgw_usage_log_flush_threshold = 1024
rgw_usage_log_tick_interval = 30
rgw_usage_max_shards = 32
rgw_usage_max_user_shards = 1
rgw_user_quota_bucket_sync_interval = 30
rgw_user_quota_sync_interval = 30

[osd]
osd_class_dir = /usr/lib64/rados-classes
`

var rookConfigNoRgwNoOpenstackNoOverrideWithMDS = `[global]
cluster_network = 127.0.0.0/16
public_network = 192.168.0.0/16
mon_max_pg_per_osd = 300
mon_target_pg_per_osd = 100

[mon]
mon_warn_on_insecure_global_id_reclaim = false
mon_warn_on_insecure_global_id_reclaim_allowed = false

[mds]
mds_cache_memory_limit = 10G

[mds.testfs]
mds_cache_memory_limit = 5G

[mds.testfs2]
mds_cache_memory_limit = 6G

[osd]
osd_class_dir = /usr/lib64/rados-classes
`

var defaultRuntimeConfig = map[string]string{
	"osd|bdev_async_discard_threads": "1",
	"osd|bdev_enable_discard":        "true",
}

func TestBuildRookConfig(t *testing.T) {
	tests := []struct {
		name                  string
		cephDpl               *cephlcmv1alpha1.CephDeployment
		openstackSecret       *v1.Secret
		currentCephVersion    *lcmcommon.CephVersion
		expectedRookConfig    string
		expectedRuntimeConfig map[string]string
		expectedHashConfig    map[string]string
	}{
		{
			name:                  "base rook config no extra opts",
			cephDpl:               &unitinputs.BaseCephDeployment,
			expectedRookConfig:    unitinputs.BaseRookConfigOverride.Data["config"],
			expectedRuntimeConfig: defaultRuntimeConfig,
			expectedHashConfig: map[string]string{
				"global": "95b401f9fc7db148cf2cc3bbcbbe09f7722b2060acf714c142fdf07ee249f0bb",
				"mon":    "52235ccf3c9f953de0fc2b8e2928f8119e1be19c14a4cf300c55e8498ec81fa2",
			},
		},
		{
			name: "base rook config no extra opts - no override network configs",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				cd := unitinputs.BaseCephDeployment.DeepCopy()
				cd.Spec.RookConfig = map[string]string{
					"public_network":  "0.0.0.0/0",
					"cluster_network": "0.0.0.0/0",
				}
				return cd
			}(),
			expectedRookConfig:    unitinputs.BaseRookConfigOverride.Data["config"],
			expectedRuntimeConfig: defaultRuntimeConfig,
			expectedHashConfig: map[string]string{
				"global": "95b401f9fc7db148cf2cc3bbcbbe09f7722b2060acf714c142fdf07ee249f0bb",
				"mon":    "52235ccf3c9f953de0fc2b8e2928f8119e1be19c14a4cf300c55e8498ec81fa2",
			},
		},
		{
			name:                  "multus base rook config no extra opts",
			cephDpl:               unitinputs.BaseCephDeploymentMultus.DeepCopy(),
			expectedRookConfig:    rookConfigNoRgwNoOpenstackNoOverrideMultus,
			expectedRuntimeConfig: defaultRuntimeConfig,
			expectedHashConfig: map[string]string{
				"global": "b34b8d687a4d80ecfa466764a0a99d3a914fe3a7e066122b72c7734581c17a39",
				"mon":    "52235ccf3c9f953de0fc2b8e2928f8119e1be19c14a4cf300c55e8498ec81fa2",
			},
		},
		{
			name:                  "base rook config - overriden from spec no runtime",
			cephDpl:               &unitinputs.CephDeployRookConfigNoRuntimeNoOsd,
			expectedRookConfig:    rookConfigNoRgwNoOpenstackOverride,
			expectedRuntimeConfig: defaultRuntimeConfig,
			expectedHashConfig: map[string]string{
				"global": "5f9266382ef04130157dff017d4877df9beeaeee17feb89747cb569dfcbc7653",
				"mon":    "52235ccf3c9f953de0fc2b8e2928f8119e1be19c14a4cf300c55e8498ec81fa2",
			},
		},
		{
			name:               "base rook config - overriden from spec",
			cephDpl:            &unitinputs.CephDeployRookConfigNoRuntimeOsdParams,
			expectedRookConfig: rookConfigNoRgwNoOpenstackOverride,
			expectedRuntimeConfig: map[string]string{
				"global|osd_max_backfills":        "64",
				"global|osd_recovery_max_active":  "16",
				"global|osd_recovery_op_priority": "3",
				"global|osd_recovery_sleep_hdd":   "0.000000",
				"osd|bdev_async_discard_threads":  "1",
				"osd|bdev_enable_discard":         "true",
			},
			expectedHashConfig: map[string]string{
				"global": "5f9266382ef04130157dff017d4877df9beeaeee17feb89747cb569dfcbc7653",
				"mon":    "52235ccf3c9f953de0fc2b8e2928f8119e1be19c14a4cf300c55e8498ec81fa2",
			},
		},
		{
			name:                  "base rgw rook config - no override from spec",
			cephDpl:               &unitinputs.CephDeployObjectStorageCeph,
			expectedRookConfig:    rookConfigRgwNoOpenstackNoOverride,
			expectedRuntimeConfig: defaultRuntimeConfig,
			expectedHashConfig: map[string]string{
				"global":                 "95b401f9fc7db148cf2cc3bbcbbe09f7722b2060acf714c142fdf07ee249f0bb",
				"mon":                    "52235ccf3c9f953de0fc2b8e2928f8119e1be19c14a4cf300c55e8498ec81fa2",
				"client.rgw.rgw.store.a": "0344afbbde075b971d43bd0a284b310645019648a4fccab8853c4eb91d0e3562",
			},
		},
		{
			name: "base rgw rook config ingress no openstack - override from spec",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				mc := unitinputs.CephDeployNonMoskWithIngress.DeepCopy()
				mc.Spec.RookConfig = map[string]string{
					"osd_max_backfills": "64",
				}
				return mc
			}(),
			expectedRookConfig: rookConfigRgwIngressNoOpenstackNoOverride,
			expectedRuntimeConfig: map[string]string{
				"global|osd_max_backfills":       "64",
				"osd|bdev_async_discard_threads": "1",
				"osd|bdev_enable_discard":        "true",
			},
			expectedHashConfig: map[string]string{
				"global":                 "95b401f9fc7db148cf2cc3bbcbbe09f7722b2060acf714c142fdf07ee249f0bb",
				"mon":                    "52235ccf3c9f953de0fc2b8e2928f8119e1be19c14a4cf300c55e8498ec81fa2",
				"client.rgw.rgw.store.a": "a679a435d1e3735cd6342e565bc8ff631c451bc8935ecea8f4aac8f3b8f26609",
			},
		},
		{
			name:               "rook rgw openstack no barbican config - no override from spec",
			cephDpl:            &unitinputs.CephDeployObjectStorageCeph,
			openstackSecret:    &unitinputs.OpenstackRgwCredsSecretNoBarbican,
			expectedRookConfig: rookConfigRgwOpenstackNoBarbicanNoOverride,
			expectedRuntimeConfig: map[string]string{
				"client.rgw.rgw.store.a|rgw_keystone_admin_password": "auth-password",
				"osd|bdev_async_discard_threads":                     "1",
				"osd|bdev_enable_discard":                            "true",
			},
			expectedHashConfig: map[string]string{
				"global":                 "95b401f9fc7db148cf2cc3bbcbbe09f7722b2060acf714c142fdf07ee249f0bb",
				"mon":                    "52235ccf3c9f953de0fc2b8e2928f8119e1be19c14a4cf300c55e8498ec81fa2",
				"client.rgw.rgw.store.a": "276b0a7ed8707af0401d7d82ea286890151c50cb0d018ca45cf1a8115248c672",
			},
		},
		{
			name:               "rook rgw openstack no barbican config - overriden from spec",
			cephDpl:            &unitinputs.CephDeployObjectStorageRookConfigNoBarbicanCeph,
			openstackSecret:    &unitinputs.OpenstackRgwCredsSecretNoBarbican,
			expectedRookConfig: rookConfigRgwOpenstackNoBarbicanOverride,
			expectedRuntimeConfig: map[string]string{
				"client.rgw.rgw.store.a|rgw_keystone_admin_password": "auth-password",
				"osd|bdev_async_discard_threads":                     "1",
				"osd|bdev_enable_discard":                            "true",
			},
			expectedHashConfig: map[string]string{
				"global":                 "95b401f9fc7db148cf2cc3bbcbbe09f7722b2060acf714c142fdf07ee249f0bb",
				"mon":                    "52235ccf3c9f953de0fc2b8e2928f8119e1be19c14a4cf300c55e8498ec81fa2",
				"client.rgw.rgw.store.a": "2406df5a40ecde057ab292f40370a4e17d78cd1670ae22d624586dcc558194b6",
			},
		},
		{
			name: "rook rgw openstack no barbican config - overriden rgw_pool_thread_size < 256",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				mc := unitinputs.CephDeployObjectStorageRookConfigNoBarbicanCeph.DeepCopy()
				mc.Spec.RookConfig["rgw_thread_pool_size"] = "100"
				return mc
			}(),
			openstackSecret:    &unitinputs.OpenstackRgwCredsSecretNoBarbican,
			expectedRookConfig: rookConfigRgwOpenstackRgwPoolThreadSizeOverride,
			expectedRuntimeConfig: map[string]string{
				"client.rgw.rgw.store.a|rgw_keystone_admin_password": "auth-password",
				"osd|bdev_async_discard_threads":                     "1",
				"osd|bdev_enable_discard":                            "true",
			},
			expectedHashConfig: map[string]string{
				"global":                 "95b401f9fc7db148cf2cc3bbcbbe09f7722b2060acf714c142fdf07ee249f0bb",
				"mon":                    "52235ccf3c9f953de0fc2b8e2928f8119e1be19c14a4cf300c55e8498ec81fa2",
				"client.rgw.rgw.store.a": "f655900eaf80fc46710d6c02f93c2dd7f9cc7d6f605316015f7a3f269a694d3f",
			},
		},
		{
			name: "rook rgw openstack no barbican config - overriden rgw_pool_thread_size > 256",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				mc := unitinputs.CephDeployObjectStorageRookConfigNoBarbicanCeph.DeepCopy()
				mc.Spec.RookConfig["rgw_thread_pool_size"] = "512"
				return mc
			}(),
			openstackSecret:    &unitinputs.OpenstackRgwCredsSecretNoBarbican,
			expectedRookConfig: rookConfigRgwOpenstackNoBarbicanOverride,
			expectedRuntimeConfig: map[string]string{
				"client.rgw.rgw.store.a|rgw_keystone_admin_password": "auth-password",
				"osd|bdev_async_discard_threads":                     "1",
				"osd|bdev_enable_discard":                            "true",
			},
			expectedHashConfig: map[string]string{
				"global":                 "95b401f9fc7db148cf2cc3bbcbbe09f7722b2060acf714c142fdf07ee249f0bb",
				"mon":                    "52235ccf3c9f953de0fc2b8e2928f8119e1be19c14a4cf300c55e8498ec81fa2",
				"client.rgw.rgw.store.a": "2406df5a40ecde057ab292f40370a4e17d78cd1670ae22d624586dcc558194b6",
			},
		},
		{
			name: "rook rgw openstack no barbican config - overriden rgw_pool_thread_size invalid format",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				mc := unitinputs.CephDeployObjectStorageRookConfigNoBarbicanCeph.DeepCopy()
				mc.Spec.RookConfig["rgw_thread_pool_size"] = "stub"
				return mc
			}(),
			openstackSecret:    &unitinputs.OpenstackRgwCredsSecretNoBarbican,
			expectedRookConfig: rookConfigRgwOpenstackNoBarbicanOverride,
			expectedRuntimeConfig: map[string]string{
				"client.rgw.rgw.store.a|rgw_keystone_admin_password": "auth-password",
				"osd|bdev_async_discard_threads":                     "1",
				"osd|bdev_enable_discard":                            "true",
			},
			expectedHashConfig: map[string]string{
				"global":                 "95b401f9fc7db148cf2cc3bbcbbe09f7722b2060acf714c142fdf07ee249f0bb",
				"mon":                    "52235ccf3c9f953de0fc2b8e2928f8119e1be19c14a4cf300c55e8498ec81fa2",
				"client.rgw.rgw.store.a": "2406df5a40ecde057ab292f40370a4e17d78cd1670ae22d624586dcc558194b6",
			},
		},
		{
			name:    "rook rgw openstack barbican config - no override from spec",
			cephDpl: &unitinputs.CephDeployObjectStorageCeph,
			openstackSecret: func() *v1.Secret {
				newSc := unitinputs.OpenstackRgwCredsSecret.DeepCopy()
				newSc.Data["password"] = []byte("old-auth-password")
				return newSc
			}(),
			expectedRookConfig: rookConfigRgwOpenstackBarbicanNoOverride,
			expectedRuntimeConfig: map[string]string{
				"client.rgw.rgw.store.a|rgw_keystone_admin_password":    "old-auth-password",
				"client.rgw.rgw.store.a|rgw_keystone_barbican_password": "old-auth-password",
				"osd|bdev_async_discard_threads":                        "1",
				"osd|bdev_enable_discard":                               "true",
			},
			expectedHashConfig: map[string]string{
				"global":                 "95b401f9fc7db148cf2cc3bbcbbe09f7722b2060acf714c142fdf07ee249f0bb",
				"mon":                    "52235ccf3c9f953de0fc2b8e2928f8119e1be19c14a4cf300c55e8498ec81fa2",
				"client.rgw.rgw.store.a": "00fb4fce0c80136200d5ed9f9ec82384b71ef2e6cfd8d8b584f89c497aecb864",
			},
		},
		{
			name: "rook rgw ingress openstack barbican config - overriden from spec",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				cephDpl := unitinputs.CephDeployObjectStorageRookConfigCeph.DeepCopy()
				cephDpl.Spec.IngressConfig = unitinputs.CephIngressConfig.DeepCopy()
				cephDpl.Spec.RookConfig["osd_max_backfills"] = "64"
				return cephDpl
			}(),
			openstackSecret:    &unitinputs.OpenstackRgwCredsSecret,
			expectedRookConfig: rookConfigRgwOpenstackBarbicanOverride,
			expectedRuntimeConfig: map[string]string{
				"client.rgw.rgw.store.a|rgw_keystone_admin_password":    "auth-password",
				"client.rgw.rgw.store.a|rgw_keystone_barbican_password": "auth-password",
				"global|osd_max_backfills":                              "64",
				"osd|bdev_async_discard_threads":                        "1",
				"osd|bdev_enable_discard":                               "true",
			},
			expectedHashConfig: map[string]string{
				"global":                 "5f9266382ef04130157dff017d4877df9beeaeee17feb89747cb569dfcbc7653",
				"mon":                    "52235ccf3c9f953de0fc2b8e2928f8119e1be19c14a4cf300c55e8498ec81fa2",
				"client.rgw.rgw.store.a": "4cb9a0831c708b296ce8ff85e50877c4fd47896de620e927faa0b0ebdecbb5da",
			},
		},
		{
			name:               "rook rgw openstack ingress - no override from spec",
			cephDpl:            &unitinputs.CephDeployMosk,
			openstackSecret:    &unitinputs.OpenstackRgwCredsSecretNoBarbican,
			expectedRookConfig: rookConfigRgwIngressOpenstackNoBarbicanNoOverride,
			expectedRuntimeConfig: map[string]string{
				"client.rgw.rgw.store.a|rgw_keystone_admin_password": "auth-password",
				"osd|bdev_async_discard_threads":                     "1",
				"osd|bdev_enable_discard":                            "true",
			},
			expectedHashConfig: map[string]string{
				"global":                 "95b401f9fc7db148cf2cc3bbcbbe09f7722b2060acf714c142fdf07ee249f0bb",
				"mon":                    "52235ccf3c9f953de0fc2b8e2928f8119e1be19c14a4cf300c55e8498ec81fa2",
				"client.rgw.rgw.store.a": "50d901b60f46f8a30260335d3e41b8982706ae18c324608167290a51e392a36a",
			},
		},
		{
			name: "base rgw ingress config - overriden from spec",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				cephDpl := unitinputs.CephDeployObjectStorageCeph.DeepCopy()
				cephDpl.Spec.IngressConfig = unitinputs.CephIngressConfig.DeepCopy()
				cephDpl.Spec.RookConfig = map[string]string{"rgw_dns_name": "rgw-store.fromspec.com"}
				return cephDpl
			}(),
			expectedRookConfig:    rookConfigRgwIngressNoOpenstackOverride,
			expectedRuntimeConfig: defaultRuntimeConfig,
			expectedHashConfig: map[string]string{
				"global":                 "95b401f9fc7db148cf2cc3bbcbbe09f7722b2060acf714c142fdf07ee249f0bb",
				"mon":                    "52235ccf3c9f953de0fc2b8e2928f8119e1be19c14a4cf300c55e8498ec81fa2",
				"client.rgw.rgw.store.a": "3e52118e763b4846e65e4e28132acd95a91875e94c5f708118e653ba7d455e10",
			},
		},
		{
			name: "base rgw ingress new config - overriden from spec",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				cephDpl := unitinputs.CephDeployObjectStorageCeph.DeepCopy()
				cephDpl.Spec.IngressConfig = unitinputs.CephIngressConfig.DeepCopy()
				cephDpl.Spec.RookConfig = map[string]string{"rgw_dns_name": "rgw-store.fromspec.com"}
				return cephDpl
			}(),
			expectedRookConfig:    rookConfigRgwIngressNoOpenstackOverride,
			expectedRuntimeConfig: defaultRuntimeConfig,
			expectedHashConfig: map[string]string{
				"global":                 "95b401f9fc7db148cf2cc3bbcbbe09f7722b2060acf714c142fdf07ee249f0bb",
				"mon":                    "52235ccf3c9f953de0fc2b8e2928f8119e1be19c14a4cf300c55e8498ec81fa2",
				"client.rgw.rgw.store.a": "3e52118e763b4846e65e4e28132acd95a91875e94c5f708118e653ba7d455e10",
			},
		},
		{
			name: "rook config with extra opts and section override",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				mc := unitinputs.BaseCephDeployment.DeepCopy()
				mc.Spec.RookConfig = map[string]string{
					"mon_target_pg_per_osd":              "100",
					"mon.b|mon_max_pg_per_osd":           "300",
					"mon|mon_health_to_clog":             "true",
					"osd.14|osd_journal_size":            "6250",
					"osd.12|osd_journal_size":            "5120",
					"mgr|mgr/prometheus/scrape_interval": "100",
				}
				return mc
			}(),
			expectedRookConfig: rookConfigNoRgwNoOpenstackOverrideWithSection,
			expectedRuntimeConfig: map[string]string{
				"osd.12|osd_journal_size":        "5120",
				"osd.14|osd_journal_size":        "6250",
				"osd|bdev_async_discard_threads": "1",
				"osd|bdev_enable_discard":        "true",
			},
			expectedHashConfig: map[string]string{
				"global": "95b401f9fc7db148cf2cc3bbcbbe09f7722b2060acf714c142fdf07ee249f0bb",
				"mon":    "4e6b581e31eb701d1fcdc1128bd4dba0643d3ff4bc5478262afe83a1fb38ba49",
				"mgr":    "5da5b6cafcade8cd83b5dd9708c6669e65b516637abaf45e47212cdc320ca412",
			},
		},
		{
			name:                  "rook config with rgw sync daemon",
			cephDpl:               &unitinputs.MultisiteRgwWithSyncDaemon,
			expectedRookConfig:    rookConfigRgwSyncDaemonNoOpenstackNoOverride,
			expectedRuntimeConfig: defaultRuntimeConfig,
			expectedHashConfig: map[string]string{
				"global":                      "95b401f9fc7db148cf2cc3bbcbbe09f7722b2060acf714c142fdf07ee249f0bb",
				"mon":                         "52235ccf3c9f953de0fc2b8e2928f8119e1be19c14a4cf300c55e8498ec81fa2",
				"client.rgw.rgw.store.a":      "0344afbbde075b971d43bd0a284b310645019648a4fccab8853c4eb91d0e3562",
				"client.rgw.rgw.store.sync.a": "a9235fc0e5c1a7a80f166b05b1edf0667f8529751b9f802ab4109e28fcf4ea7a",
			},
		},
		{
			name: "rook rgw openstack - custom ingress hostname",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				cephDpl := unitinputs.CephDeployObjectStorageCeph.DeepCopy()
				cephDpl.Spec.IngressConfig = unitinputs.CephIngressConfig.DeepCopy()
				cephDpl.Spec.IngressConfig.TLSConfig.Domain = "fromspec.com"
				cephDpl.Spec.IngressConfig.TLSConfig.Hostname = "rgw-store"
				cephDpl.Spec.IngressConfig.Annotations = nil
				cephDpl.Spec.IngressConfig.ControllerClassName = ""
				return cephDpl
			}(),
			expectedRookConfig:    rookConfigRgwIngressNoOpenstackOverride,
			expectedRuntimeConfig: defaultRuntimeConfig,
			expectedHashConfig: map[string]string{
				"global":                 "95b401f9fc7db148cf2cc3bbcbbe09f7722b2060acf714c142fdf07ee249f0bb",
				"mon":                    "52235ccf3c9f953de0fc2b8e2928f8119e1be19c14a4cf300c55e8498ec81fa2",
				"client.rgw.rgw.store.a": "3e52118e763b4846e65e4e28132acd95a91875e94c5f708118e653ba7d455e10",
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := fakeDeploymentConfig(&deployConfig{cephDpl: test.cephDpl}, nil)
			if test.currentCephVersion == nil {
				test.currentCephVersion = lcmcommon.LatestRelease
			}
			c.cdConfig.currentCephVersion = test.currentCephVersion

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
	monUpdatedAnnotation := fmt.Sprintf(cephConfigParametersUpdateTimestampLabel, "mon")
	globalUpdatedAnnotation := fmt.Sprintf(cephConfigParametersUpdateTimestampLabel, "global")
	tests := []struct {
		name               string
		cephDpl            *cephlcmv1alpha1.CephDeployment
		openstackSecret    *v1.Secret
		rookCm             *v1.ConfigMap
		apiErrors          map[string]error
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
			cephDpl:       &unitinputs.CephDeployObjectStorageCeph,
			apiErrors:     map[string]error{"get-secrets-openstack-rgw-creds": errors.New("failed to get openstack rgw secret")},
			expectedError: "failed to prepare ceph config map: failed to get openstack rgw secret",
			expectedTimestamps: updateTimestamps{
				cephConfigMap: map[string]string{},
			},
		},
		{
			name:          "failed to get rook ceph override config map",
			cephDpl:       &unitinputs.BaseCephDeployment,
			apiErrors:     map[string]error{"get-configmaps-rook-config-override": errors.New("failed to get configmap rook-config-override")},
			expectedError: "failed to get configmap rook-config-override",
			expectedTimestamps: updateTimestamps{
				cephConfigMap: map[string]string{},
			},
		},
		{
			name:          "failed to create config map",
			cephDpl:       &unitinputs.BaseCephDeployment,
			apiErrors:     map[string]error{"create-configmaps-rook-config-override": errors.New("failed to create configmap rook-config-override")},
			expectedError: "failed to create configmap rook-config-override",
			expectedTimestamps: updateTimestamps{
				cephConfigMap: map[string]string{},
			},
		},
		{
			name:    "config map is created",
			cephDpl: &unitinputs.BaseCephDeployment,
			expectedTimestamps: updateTimestamps{
				cephConfigMap: map[string]string{
					"global": "time-3",
					"mon":    "time-3",
				},
			},
			expectedResources: map[string]runtime.Object{
				"configmaps": &v1.ConfigMapList{
					Items: []v1.ConfigMap{
						func() v1.ConfigMap {
							newCM := unitinputs.BaseRookConfigOverride.DeepCopy()
							newCM.Annotations[cephConfigMapUpdateTimestampLabel] = "time-3"
							newCM.Annotations[monUpdatedAnnotation] = "time-3"
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
				newCM.Annotations[monUpdatedAnnotation] = "time-3"
				newCM.Annotations[globalUpdatedAnnotation] = "time-3"
				return newCM
			}(),
			configDump:         unitinputs.CephConfigDumpDefaults,
			cephClusterPresent: true,
			expectedTimestamps: updateTimestamps{
				cephConfigMap: map[string]string{
					"global": "time-3",
					"mon":    "time-3",
				},
			},
		},
		{
			name:    "config map is present, no changes but global vars not set before",
			cephDpl: &unitinputs.BaseCephDeployment,
			rookCm: func() *v1.ConfigMap {
				newCM := unitinputs.BaseRookConfigOverride.DeepCopy()
				newCM.Annotations[cephConfigMapUpdateTimestampLabel] = "some-time"
				newCM.Annotations[monUpdatedAnnotation] = "some-time"
				newCM.Annotations[globalUpdatedAnnotation] = "some-time"
				newCM.Annotations[cephRuntimeRgwParametersUpdateTimestampLabel] = "some-rgw-time"
				newCM.Annotations[cephRuntimeOsdParametersUpdateTimestampLabel] = "some-osd-time"
				return newCM
			}(),
			configDump:         unitinputs.CephConfigDumpDefaults,
			cephClusterPresent: true,
			unsetGlobalVar:     true,
			expectedTimestamps: updateTimestamps{
				cephConfigMap: map[string]string{
					"global": "some-time",
					"mon":    "some-time",
				},
				rgwRuntimeParams: "some-rgw-time",
				osdRuntimeParams: "some-osd-time",
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
					"mon":    "some-time",
				},
				rgwRuntimeParams: "some-rgw-time",
				osdRuntimeParams: "some-osd-time",
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
					"mon":    "time-7",
				},
				rgwRuntimeParams: "some-rgw-time",
				osdRuntimeParams: "some-osd-time",
			},
			expectedResources: map[string]runtime.Object{
				"configmaps": &v1.ConfigMapList{Items: []v1.ConfigMap{
					func() v1.ConfigMap {
						newCM := unitinputs.BaseRookConfigOverride.DeepCopy()
						newCM.Annotations[cephConfigMapUpdateTimestampLabel] = "time-7"
						newCM.Annotations[monUpdatedAnnotation] = "time-7"
						newCM.Annotations[globalUpdatedAnnotation] = "time-7"
						newCM.Annotations[cephRuntimeRgwParametersUpdateTimestampLabel] = "some-rgw-time"
						newCM.Annotations[cephRuntimeOsdParametersUpdateTimestampLabel] = "some-osd-time"
						return *newCM
					}(),
				}},
			},
		},
		{
			name:    "config map and osd params are updated",
			cephDpl: &unitinputs.CephDeployRookConfigNoRuntimeOsdParams,
			rookCm: func() *v1.ConfigMap {
				newCM := unitinputs.BaseRookConfigOverride.DeepCopy()
				newCM.Annotations[cephConfigMapUpdateTimestampLabel] = "time-7"
				newCM.Annotations[monUpdatedAnnotation] = "time-7"
				newCM.Annotations[globalUpdatedAnnotation] = "time-7"
				newCM.Annotations[cephRuntimeRgwParametersUpdateTimestampLabel] = "some-rgw-time"
				newCM.Annotations[cephRuntimeOsdParametersUpdateTimestampLabel] = "some-osd-time"
				return newCM
			}(),
			configDump:         unitinputs.CephConfigDumpDefaults,
			cephClusterPresent: true,
			configRuntimeCmds: map[string]bool{
				"ceph config set global osd_max_backfills 64":            false,
				"ceph config set global osd_recovery_max_active 16":      false,
				"ceph config set global osd_recovery_op_priority 3":      false,
				"ceph config set global osd_recovery_sleep_hdd 0.000000": false,
			},
			configUpdated: true,
			expectedTimestamps: updateTimestamps{
				cephConfigMap: map[string]string{
					"global": "time-8",
					"mon":    "time-7",
				},
				rgwRuntimeParams: "some-rgw-time",
				osdRuntimeParams: "time-8",
			},
			expectedResources: map[string]runtime.Object{
				"configmaps": &v1.ConfigMapList{Items: []v1.ConfigMap{
					func() v1.ConfigMap {
						newCM := unitinputs.BaseRookConfigOverride.DeepCopy()
						newCM.Annotations[cephConfigMapUpdateTimestampLabel] = "time-8"
						newCM.Annotations[monUpdatedAnnotation] = "time-7"
						newCM.Annotations[globalUpdatedAnnotation] = "time-8"
						newCM.Annotations[cephRuntimeRgwParametersUpdateTimestampLabel] = "some-rgw-time"
						newCM.Annotations[cephRuntimeOsdParametersUpdateTimestampLabel] = "time-8"
						newCM.Annotations["cephdeployment.lcm.mirantis.com/config-global-hash"] = "5f9266382ef04130157dff017d4877df9beeaeee17feb89747cb569dfcbc7653"
						newCM.Data["config"] = rookConfigNoRgwNoOpenstackOverride
						newCM.Data["runtime"] = "global|osd_max_backfills = 64\nglobal|osd_recovery_max_active = 16\nglobal|osd_recovery_op_priority = 3\nglobal|osd_recovery_sleep_hdd = 0.000000\nosd|bdev_async_discard_threads = 1\nosd|bdev_enable_discard = true\n"
						return *newCM
					}(),
				}},
			},
		},
		{
			name:    "config map and runtime updated - rgw params has been added",
			cephDpl: &unitinputs.CephDeployObjectStorageCeph,
			rookCm: func() *v1.ConfigMap {
				newCM := unitinputs.BaseRookConfigOverride.DeepCopy()
				newCM.Annotations[cephConfigMapUpdateTimestampLabel] = "time-8"
				newCM.Annotations[monUpdatedAnnotation] = "time-7"
				newCM.Annotations[globalUpdatedAnnotation] = "time-8"
				newCM.Annotations[cephRuntimeRgwParametersUpdateTimestampLabel] = "some-rgw-time"
				newCM.Annotations[cephRuntimeOsdParametersUpdateTimestampLabel] = "time-8"
				return newCM
			}(),
			configDump:      unitinputs.CephConfigDumpOverride,
			openstackSecret: &unitinputs.OpenstackRgwCredsSecretNoBarbican,
			configRuntimeCmds: map[string]bool{
				"ceph config set client.rgw.rgw.store.a rgw_keystone_admin_password auth-password": false,
			},
			cephClusterPresent: true,
			expectedTimestamps: updateTimestamps{
				cephConfigMap: map[string]string{
					"global":                 "time-8",
					"mon":                    "time-7",
					"client.rgw.rgw.store.a": "time-9",
				},
				rgwRuntimeParams: "time-9",
				osdRuntimeParams: "time-8",
			},
			configUpdated: true,
			expectedResources: map[string]runtime.Object{
				"configmaps": &v1.ConfigMapList{Items: []v1.ConfigMap{
					func() v1.ConfigMap {
						newCM := unitinputs.BaseRookConfigOverride.DeepCopy()
						newCM.Data["config"] = rookConfigRgwOpenstackNoBarbicanNoOverride
						newCM.Data["runtime"] = "client.rgw.rgw.store.a|rgw_keystone_admin_password = *\nosd|bdev_async_discard_threads = 1\nosd|bdev_enable_discard = true\n"
						newCM.Annotations[cephConfigMapUpdateTimestampLabel] = "time-9"
						newCM.Annotations[monUpdatedAnnotation] = "time-7"
						newCM.Annotations[globalUpdatedAnnotation] = "time-8"
						newCM.Annotations[cephRuntimeRgwParametersUpdateTimestampLabel] = "time-9"
						newCM.Annotations[cephRuntimeOsdParametersUpdateTimestampLabel] = "time-8"
						newCM.Annotations["cephdeployment.lcm.mirantis.com/config-client.rgw.rgw.store.a-updated"] = "time-9"
						newCM.Annotations["cephdeployment.lcm.mirantis.com/config-client.rgw.rgw.store.a-hash"] = "276b0a7ed8707af0401d7d82ea286890151c50cb0d018ca45cf1a8115248c672"
						return *newCM
					}(),
				}},
			},
		},
		{
			name:    "no timestamps var set and only passwords is updated",
			cephDpl: &unitinputs.CephDeployObjectStorageCeph,
			rookCm: func() *v1.ConfigMap {
				newCM := unitinputs.BaseRookConfigOverride.DeepCopy()
				newCM.Data["config"] = rookConfigRgwOpenstackNoBarbicanNoOverride
				newCM.Data["runtime"] = "client.rgw.rgw.store.a|rgw_keystone_admin_password = *\nosd|bdev_async_discard_threads = 1\nosd|bdev_enable_discard = true\n"
				newCM.Annotations[cephConfigMapUpdateTimestampLabel] = "time-9"
				newCM.Annotations[monUpdatedAnnotation] = "time-7"
				newCM.Annotations[globalUpdatedAnnotation] = "time-8"
				newCM.Annotations[cephRuntimeRgwParametersUpdateTimestampLabel] = "time-9"
				newCM.Annotations[cephRuntimeOsdParametersUpdateTimestampLabel] = "time-8"
				newCM.Annotations["cephdeployment.lcm.mirantis.com/config-client.rgw.rgw.store.a-updated"] = "time-9"
				newCM.Annotations["cephdeployment.lcm.mirantis.com/config-client.rgw.rgw.store.a-hash"] = "276b0a7ed8707af0401d7d82ea286890151c50cb0d018ca45cf1a8115248c672"
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
					"mon":                    "time-7",
					"client.rgw.rgw.store.a": "time-9",
				},
				rgwRuntimeParams: "time-10",
				osdRuntimeParams: "time-8",
			},
			configUpdated: true,
			expectedResources: map[string]runtime.Object{
				"configmaps": &v1.ConfigMapList{Items: []v1.ConfigMap{
					func() v1.ConfigMap {
						newCM := unitinputs.BaseRookConfigOverride.DeepCopy()
						newCM.Data["config"] = rookConfigRgwOpenstackNoBarbicanNoOverride
						newCM.Data["runtime"] = "client.rgw.rgw.store.a|rgw_keystone_admin_password = *\nosd|bdev_async_discard_threads = 1\nosd|bdev_enable_discard = true\n"
						newCM.Annotations[cephConfigMapUpdateTimestampLabel] = "time-9"
						newCM.Annotations[monUpdatedAnnotation] = "time-7"
						newCM.Annotations[globalUpdatedAnnotation] = "time-8"
						newCM.Annotations[cephRuntimeRgwParametersUpdateTimestampLabel] = "time-10"
						newCM.Annotations[cephRuntimeOsdParametersUpdateTimestampLabel] = "time-8"
						newCM.Annotations["cephdeployment.lcm.mirantis.com/config-client.rgw.rgw.store.a-updated"] = "time-9"
						newCM.Annotations["cephdeployment.lcm.mirantis.com/config-client.rgw.rgw.store.a-hash"] = "276b0a7ed8707af0401d7d82ea286890151c50cb0d018ca45cf1a8115248c672"
						return *newCM
					}(),
				}},
			},
		},
		{
			name:    "config map and all runtime params are updated - rgw params removed",
			cephDpl: &unitinputs.CephDeployRookConfigNoRuntimeOsdParams,
			rookCm: func() *v1.ConfigMap {
				newCM := unitinputs.BaseRookConfigOverride.DeepCopy()
				newCM.Data["config"] = rookConfigRgwOpenstackNoBarbicanNoOverride
				newCM.Data["runtime"] = "client.rgw.rgw.store.a|rgw_keystone_admin_password = *\nosd|bdev_async_discard_threads = 1\nosd|bdev_enable_discard = true\n"
				newCM.Annotations[cephConfigMapUpdateTimestampLabel] = "time-9"
				newCM.Annotations[monUpdatedAnnotation] = "time-7"
				newCM.Annotations[globalUpdatedAnnotation] = "time-8"
				newCM.Annotations[cephRuntimeRgwParametersUpdateTimestampLabel] = "time-10"
				newCM.Annotations[cephRuntimeOsdParametersUpdateTimestampLabel] = "time-8"
				newCM.Annotations["cephdeployment.lcm.mirantis.com/config-client.rgw.rgw.store.a-updated"] = "time-9"
				newCM.Annotations["cephdeployment.lcm.mirantis.com/config-client.rgw.rgw.store.a-hash"] = "1d1baad1989f7e361c072d19b4988e32ff28b1670480b20510468725322b3a08"
				return newCM
			}(),
			configDump: unitinputs.CephConfigDumpOverrideWithRgw,
			configRuntimeCmds: map[string]bool{
				"ceph config set global osd_recovery_op_priority 3":                 false,
				"ceph config set global osd_recovery_sleep_hdd 0.000000":            false,
				"ceph config rm client.rgw.rgw.store.a rgw_keystone_admin_password": false,
			},
			cephClusterPresent: true,
			expectedTimestamps: updateTimestamps{
				cephConfigMap: map[string]string{
					"global": "time-11",
					"mon":    "time-7",
				},
				rgwRuntimeParams: "time-11",
				osdRuntimeParams: "time-11",
			},
			configUpdated: true,
			expectedResources: map[string]runtime.Object{
				"configmaps": &v1.ConfigMapList{Items: []v1.ConfigMap{
					func() v1.ConfigMap {
						newCM := unitinputs.BaseRookConfigOverride.DeepCopy()
						newCM.Data["config"] = rookConfigNoRgwNoOpenstackOverride
						newCM.Data["runtime"] = "global|osd_max_backfills = 64\nglobal|osd_recovery_max_active = 16\nglobal|osd_recovery_op_priority = 3\nglobal|osd_recovery_sleep_hdd = 0.000000\nosd|bdev_async_discard_threads = 1\nosd|bdev_enable_discard = true\n"
						newCM.Annotations[cephConfigMapUpdateTimestampLabel] = "time-11"
						newCM.Annotations[monUpdatedAnnotation] = "time-7"
						newCM.Annotations[globalUpdatedAnnotation] = "time-11"
						newCM.Annotations["cephdeployment.lcm.mirantis.com/config-global-hash"] = "5f9266382ef04130157dff017d4877df9beeaeee17feb89747cb569dfcbc7653"
						newCM.Annotations[cephRuntimeRgwParametersUpdateTimestampLabel] = "time-11"
						newCM.Annotations[cephRuntimeOsdParametersUpdateTimestampLabel] = "time-11"
						return *newCM
					}(),
				}},
			},
		},
		{
			name: "config map updated - mds params added",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				mc := unitinputs.BaseCephDeployment.DeepCopy()
				mc.Spec.RookConfig = map[string]string{
					"mds|mds_cache_memory_limit":         "10G",
					"mds.testfs|mds_cache_memory_limit":  "5G",
					"mds.testfs2|mds_cache_memory_limit": "6G",
				}
				return mc
			}(),
			rookCm: func() *v1.ConfigMap {
				newCM := unitinputs.BaseRookConfigOverride.DeepCopy()
				newCM.Data["config"] = unitinputs.BaseRookConfigOverride.Data["config"]
				newCM.Data["runtime"] = "osd|bdev_async_discard_threads = 1\nosd|bdev_enable_discard = true\n"
				newCM.Annotations[cephConfigMapUpdateTimestampLabel] = "time-9"
				newCM.Annotations[monUpdatedAnnotation] = "time-7"
				newCM.Annotations[globalUpdatedAnnotation] = "time-11"
				newCM.Annotations["cephdeployment.lcm.mirantis.com/config-global-hash"] = "95b401f9fc7db148cf2cc3bbcbbe09f7722b2060acf714c142fdf07ee249f0bb"
				newCM.Annotations[cephRuntimeRgwParametersUpdateTimestampLabel] = "time-11"
				newCM.Annotations[cephRuntimeOsdParametersUpdateTimestampLabel] = "time-11"
				return newCM
			}(),
			configDump:         unitinputs.CephConfigDumpDefaults,
			cephClusterPresent: true,
			expectedTimestamps: updateTimestamps{
				cephConfigMap: map[string]string{
					"global":      "time-11",
					"mon":         "time-7",
					"mds":         "time-12",
					"mds.testfs":  "time-12",
					"mds.testfs2": "time-12",
				},
				rgwRuntimeParams: "time-11",
				osdRuntimeParams: "time-11",
			},
			configUpdated: true,
			expectedResources: map[string]runtime.Object{
				"configmaps": &v1.ConfigMapList{Items: []v1.ConfigMap{
					func() v1.ConfigMap {
						newCM := unitinputs.BaseRookConfigOverride.DeepCopy()
						newCM.Data["config"] = rookConfigNoRgwNoOpenstackNoOverrideWithMDS
						newCM.Data["runtime"] = "osd|bdev_async_discard_threads = 1\nosd|bdev_enable_discard = true\n"
						newCM.Annotations[cephConfigMapUpdateTimestampLabel] = "time-12"
						newCM.Annotations[monUpdatedAnnotation] = "time-7"
						newCM.Annotations[globalUpdatedAnnotation] = "time-11"
						newCM.Annotations["cephdeployment.lcm.mirantis.com/config-global-hash"] = "95b401f9fc7db148cf2cc3bbcbbe09f7722b2060acf714c142fdf07ee249f0bb"
						newCM.Annotations["cephdeployment.lcm.mirantis.com/config-mds-updated"] = "time-12"
						newCM.Annotations["cephdeployment.lcm.mirantis.com/config-mds-hash"] = "159347878ba9b5eff291a3a76176581f76756de08c2835077546e9c63c469208"
						newCM.Annotations["cephdeployment.lcm.mirantis.com/config-mds.testfs-updated"] = "time-12"
						newCM.Annotations["cephdeployment.lcm.mirantis.com/config-mds.testfs-hash"] = "08ec6e7070ba2ec165c159b6226456159bf87f9ad5507e53d1ce82c69c3305c3"
						newCM.Annotations["cephdeployment.lcm.mirantis.com/config-mds.testfs2-updated"] = "time-12"
						newCM.Annotations["cephdeployment.lcm.mirantis.com/config-mds.testfs2-hash"] = "1b222702f00db584e622e4d13b9cefe38934f46b7ad1b03451c065f9675b2bf4"
						newCM.Annotations[cephRuntimeRgwParametersUpdateTimestampLabel] = "time-11"
						newCM.Annotations[cephRuntimeOsdParametersUpdateTimestampLabel] = "time-11"
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
				newCM.Data["config"] = rookConfigNoRgwNoOpenstackNoOverrideWithMDS
				newCM.Data["runtime"] = "osd|bdev_async_discard_threads = 1\nosd|bdev_enable_discard = true\n"
				newCM.Annotations[cephConfigMapUpdateTimestampLabel] = "time-12"
				newCM.Annotations[monUpdatedAnnotation] = "time-7"
				newCM.Annotations[globalUpdatedAnnotation] = "time-11"
				newCM.Annotations["cephdeployment.lcm.mirantis.com/config-global-hash"] = "95b401f9fc7db148cf2cc3bbcbbe09f7722b2060acf714c142fdf07ee249f0bb"
				newCM.Annotations["cephdeployment.lcm.mirantis.com/config-mds-updated"] = "time-12"
				newCM.Annotations["cephdeployment.lcm.mirantis.com/config-mds-hash"] = "159347878ba9b5eff291a3a76176581f76756de08c2835077546e9c63c469208"
				newCM.Annotations["cephdeployment.lcm.mirantis.com/config-mds.testfs-updated"] = "time-12"
				newCM.Annotations["cephdeployment.lcm.mirantis.com/config-mds.testfs-hash"] = "08ec6e7070ba2ec165c159b6226456159bf87f9ad5507e53d1ce82c69c3305c3"
				newCM.Annotations["cephdeployment.lcm.mirantis.com/config-mds.testfs2-updated"] = "time-12"
				newCM.Annotations["cephdeployment.lcm.mirantis.com/config-mds.testfs2-hash"] = "1b222702f00db584e622e4d13b9cefe38934f46b7ad1b03451c065f9675b2bf4"
				newCM.Annotations[cephRuntimeRgwParametersUpdateTimestampLabel] = "time-11"
				newCM.Annotations[cephRuntimeOsdParametersUpdateTimestampLabel] = "time-11"
				return newCM
			}(),
			configDump:         unitinputs.CephConfigDumpDefaults,
			cephClusterPresent: true,
			expectedTimestamps: updateTimestamps{
				cephConfigMap: map[string]string{
					"global": "time-11",
					"mon":    "time-7",
				},
				rgwRuntimeParams: "time-11",
				osdRuntimeParams: "time-11",
			},
			configUpdated: true,
			expectedResources: map[string]runtime.Object{
				"configmaps": &v1.ConfigMapList{Items: []v1.ConfigMap{
					func() v1.ConfigMap {
						newCM := unitinputs.BaseRookConfigOverride.DeepCopy()
						newCM.Data["config"] = unitinputs.BaseRookConfigOverride.Data["config"]
						newCM.Data["runtime"] = "osd|bdev_async_discard_threads = 1\nosd|bdev_enable_discard = true\n"
						newCM.Annotations[cephConfigMapUpdateTimestampLabel] = "time-13"
						newCM.Annotations[monUpdatedAnnotation] = "time-7"
						newCM.Annotations[globalUpdatedAnnotation] = "time-11"
						newCM.Annotations["cephdeployment.lcm.mirantis.com/config-global-hash"] = "95b401f9fc7db148cf2cc3bbcbbe09f7722b2060acf714c142fdf07ee249f0bb"
						newCM.Annotations[cephRuntimeRgwParametersUpdateTimestampLabel] = "time-11"
						newCM.Annotations[cephRuntimeOsdParametersUpdateTimestampLabel] = "time-11"
						return *newCM
					}(),
				}},
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

			c := fakeDeploymentConfig(&deployConfig{cephDpl: test.cephDpl}, nil)
			c.cdConfig.currentCephVersion = lcmcommon.LatestRelease
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
