<a id="lcmconfig-configmap-pelagia-lcmconfig"></a>

# ConfigMap pelagia-lcmconfig

Pelagia controllers use a common configuration place that is the
[`pelagia-lcmconfig` Configmap](https://github.com/Mirantis/pelagia/blob/main/charts/pelagia-ceph/templates/common.yaml)
object located in the same namespace as the Pelagia controllers. This Configmap is created by the Pelagia Helm chart.

Some parameters are applied from Helm chart values when you upgrade the chart,
others are not updated by the chart and must be edited directly in `pelagia-lcmconfig`.

## Configuration options for Pelagia controllers

If a parameter is not specified in `pelagia-lcmconfig`, it uses the code-based default value.

### Options configurable through chart values

The following Pelagia controller options are set in [Pelagia Helm chart values](./helm-values.md) during chart update.

| Parameter | Description | Default | Chart value |
|-----------|-------------|---------|----------------------------|
| ROOK_NAMESPACE | Rook namespace. | `"rook-ceph"` | `rookConfig.rookNamespace` |
| DISK_DAEMON_API_PORT | Port for the disk daemon API. | `9999` | `lcmConfig.diskDaemonPortParameter` |
| DISK_DAEMON_PLACEMENT_NODES_SELECTOR | Label for disk daemon placement. | `"pelagia-disk-daemon=true"` | `lcmConfig.diskDaemonNodeSelector` |
| RGW_PUBLIC_ACCESS_SERVICE_SELECTOR | Label of the service or proxy exposing RGW to public access. | `"external_access=rgw"` | `lcmConfig.rgwPublicAccessServiceSelector` |
| DEPLOYMENT_CEPH_RELEASE | Pin the Ceph release for the current setup. If empty, uses the latest available release for the current version. | `""` | `cephRelease` |
| DEPLOYMENT_NETPOL_ENABLED | Enable creation of network policy. | `"true"` | `cephDeployment.netpolEnabled` |
| DEPLOYMENT_OPENSTACK_CEPH_SHARED_NAMESPACE | Namespace for the Openstack-Ceph communication and secrets sharing. | `"openstack-ceph-shared"` | `cephDeployment.openstackSharedNamespace` |
| DEPLOYMENT_LABEL_TO_EXCLUDE_CEPH_DAEMONSETS | Label for nodes where no Ceph daemons must be scheduled. | `""` | `lcmConfig.cephDaemonsetLabelExclude` |

The `DEPLOYMENT_CEPH_IMAGE` and `DEPLOYMENT_ROOK_IMAGE` options are derived from the values of the `images` section.
For details, see [Configuration example for Ceph and Rook images](./helm-values.md) during chart update.

### Options configurable manually

The Pelagia Helm chart update does not affect the following Pelagia controller options.
Therefore, you can update them only manually using the `kubectl -n pelagia edit cm pelagia-lcmconfig` command.

| Parameter | Description | Default |
|-----------|-------------|---------|
| DEPLOYMENT_LOG_LEVEL | Log level of the Pelagia deployment controller. Possible values: `info`, `debug`, `error`, `warn`. | `"info"` |
| HEALTH_CHECKS_CEPH_ISSUES_TO_IGNORE | Ceph cluster health issues to ignore in the `health` state. | `["OSDMAP_FLAGS", "TOO_FEW_PGS", "SLOW_OPS", "OLD_CRUSH_TUNABLES", "OLD_CRUSH_STRAW_CALC_VERSION", "POOL_APP_NOT_ENABLED", "MON_DISK_LOW", "RECENT_CRASH",]` |
| HEALTH_CHECKS_SKIP | Checks to skip during Ceph cluster verification. Possible values: `ceph_daemons`, `ceph_csi_daemons`, `usage_details`, `ceph_events`, `pools_replicas`, `rgw_info`, `spec_analysis`. | `[]` |
| HEALTH_CHECKS_USAGE_CLASS_FILTER | Regexp-based filter to prepare usage details only for the specified device class. | `""` |
| HEALTH_CHECKS_USAGE_POOLS_FILTER | Regexp-based filter to prepare usage details only for the specified pools. | `""` |
| HEALTH_LOG_LEVEL | Log level of the Pelagia LCM health controller. Possible values: `info`, `debug`, `error`, `warn`. | `"info"` |
| TASK_LOG_LEVEL | Log level of the Pelagia LCM `osdremote-task` controller. Possible values: `info`, `debug`, `error`, `warn`. | `"info"` |
| TASK_OSD_PG_REBALANCE_TIMEOUT_MIN | Timeout in minutes to wait for an OSD to finish rebalancing to 0 before considering the rebalance failed. For the procedure, refer to [CephOsdRemoveTask failure with a timeout during rebalance](../troubleshoot/cephosdremovetask-timeout.md) | `"30"` |
| TASK_ALLOW_REMOVE_MANUALLY_CREATED_LVMS | Remove LVM partitions during OSD partition cleanup, even if they were created manually. | `"false"` |
