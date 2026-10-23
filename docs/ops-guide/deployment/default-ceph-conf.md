---
description: Overview of default Ceph configuration options applied when the rookConfig section is empty.
keywords: pelagia, ceph default config, ceph configuration options, ceph defaults
---

<a id="default-ceph-conf-ceph-default-configuration"></a>
# Ceph default configuration

Pelagia Deployment Controller provides the capability to specify configuration options for
the Ceph cluster through the `rookConfig` key-value section of the
`CephDeployment` CR as if they were set in a usual `ceph.conf` file.

## Default configuration options

However, if `rookConfig` is empty, Pelagia Deployment Controller still specifies the
following default configuration options for each Ceph cluster:

* Required network parameters that you can change through the `cluster.network`
  section:
  ```ini
  [global]
  cluster network = <spec.cluster.network.addressRanges.cluster>
  public network = <spec.cluster.network.addressRanges.public>
  ```
* General default configuration options that you can override using the
  `rookConfig` parameter:
  ```ini
  [global]
  mon_max_pg_per_osd = 300
  mon_target_pg_per_osd = 100

  [mon]
  mon_warn_on_insecure_global_id_reclaim = false
  mon_warn_on_insecure_global_id_reclaim_allowed = false

  [osd]
  osd_class_dir = /usr/lib64/rados-classes
  bdev_enable_discard = true
  bdev_async_discard_threads = 1
  ```
* If the `spec.objectStorage.objectStores` section is defined, Pelagia
  specifies the following Ceph RADOS Gateway default configuration options
  for each object store, for example, `rgw-store`:
  ```ini
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
  ```

## Rockoon-related default configuration options

If Pelagia is integrated with Rockoon, Pelagia Deployment Controller specifies the
OpenStack-related default configuration options for a Ceph Object Storage instance
only if both of the following conditions are met:

* The object store has `usedForOpenstack: true` in `spec.objectStorage.objectStores`
* The `openstack-rgw-creds` secret provided by Rockoon exists in the OpenStack-Ceph
  shared namespace (`openstack-ceph-shared` by default)

These options are added on top of the general Ceph RADOS Gateway defaults above:

* Ceph Object Gateway options that you can override using the `rookConfig` parameter:
  ```ini
  [client.rgw.rgw.store.a]
  rgw_swift_account_in_url = true
  rgw_keystone_accepted_roles = '_member_, Member, member, swiftoperator'
  rgw_keystone_accepted_admin_roles = 'admin, service'
  rgw_keystone_implicit_tenants = true
  rgw_swift_versioning_enabled = true
  rgw_enforce_swift_acls = true
  rgw_s3_auth_use_keystone = true
  rgw_enable_usage_log = true
  rgw_usage_log_tick_interval = 30
  rgw_usage_log_flush_threshold = 1024
  rgw_usage_max_shards = 32
  rgw_usage_max_user_shards = 1
  ```
* Parameters for the Keystone integration taken from the `openstack-rgw-creds` secret:
  ```ini
  rgw_keystone_api_version = 3
  rgw_keystone_url = <keystoneAuthURL>
  rgw_keystone_admin_user = <keystoneUser>
  rgw_keystone_admin_password = <keystonePassword>
  rgw_keystone_admin_domain = <keystoneProjectDomain>
  rgw_keystone_admin_project = <keystoneProjectName>
  ```

    You cannot override `rgw_keystone_api_version`.

* If the secret contains `barbican_url`, parameters for the Barbican integration:
  ```ini
  rgw_crypt_s3_kms_backend = barbican
  rgw_barbican_url = <barbicanURL>
  rgw_keystone_barbican_user = <keystoneUser>
  rgw_keystone_barbican_password = <keystonePassword>
  rgw_keystone_barbican_domain = <keystoneProjectDomain>
  rgw_keystone_barbican_project = <keystoneProjectName>
  ```

    You cannot override `rgw_crypt_s3_kms_backend`.

!!! info "See also"

    [CephDeployment custom resource](../../custom-resources/cephdeployment.md#cephdeployment-cephdeployment-custom-resource)
