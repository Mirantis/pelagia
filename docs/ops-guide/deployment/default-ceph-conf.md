<a id="default-ceph-conf"></a>
# Ceph default configuration options

Pelagia Deployment Controller provides the capability to specify configuration options for
the Ceph cluster through the `rookConfig` key-value section of the
`CephDeployment` CR as if they were set in a usual `ceph.conf` file.

However, if `rookConfig` is empty, Pelagia Deployment Controller still specifies the
following default configuration options for each Ceph cluster:

* Required network parameters that you can change through the `network`
  section:
  ```ini
  [global]
  cluster network = <spec.network.clusterNet>
  public network = <spec.network.publicNet>
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
  ```
* If `rookConfig` is empty but the `objectStore.rgw` section is defined, Pelagia
  specifies the following Ceph RADOS Gateway default configuration options:
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

If Pelagia is integrated with [Rockoon](https://github.com/Mirantis/rockoon) and `objectStore.rgw` section
is defined in the `CephDeployment` custom resource, Pelagia Deployment Controller specifies the OpenStack-related
default configuration options for each Ceph cluster:

* Ceph Object Gateway options that you can override using the `rookConfig` parameter:
  ```ini
  [client.rgw.rgw.store.a]
  rgw swift account in url = true
  rgw keystone accepted roles = '_member_, Member, member, swiftoperator'
  rgw keystone accepted admin roles = admin
  rgw keystone implicit tenants = true
  rgw swift versioning enabled = true
  rgw enforce swift acls = true
  rgw_max_attr_name_len = 64
  rgw_max_attrs_num_in_req = 32
  rgw_max_attr_size = 1024
  rgw_bucket_quota_ttl = 0
  rgw_user_quota_bucket_sync_interval = 30
  rgw_user_quota_sync_interval = 30
  rgw s3 auth use keystone = true
  ```
* Additional parameters for the Keystone integration provided by Rockoon in shared secret:
  ```ini
  rgw keystone api version = 3
  rgw keystone url = <keystoneAuthURL>
  rgw keystone admin user = <keystoneUser>
  rgw keystone admin password = <keystonePassword>
  rgw keystone admin domain = <keystoneProjectDomain>
  rgw keystone admin project = <keystoneProjectName>
  ```

!!! info "See also"

    [CephDeployment Custom Resource](../../architecture/custom-resources/cephdeployment.md#cephdeployment-custom-resource)
