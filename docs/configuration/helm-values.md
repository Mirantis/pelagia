---
description: Reference for Pelagia Helm chart configuration parameters and their default values.
keywords: pelagia, helm chart, helm values, pelagia configuration, rook configuration, csi placement, ceph images
---

<a id="helm-values-helm-chart-configuration"></a>

# Helm chart configuration

Pelagia Helm chart contains multiple options to configure the Pelagia setup and prepare an environment for deployment.

For the complete list of available options, refer to [values.yaml](https://github.com/Mirantis/pelagia/tree/main/charts/pelagia-ceph/values.yaml).

## Configuration options

The following table lists the most commonly configured Pelagia chart parameters and their default values. For an example configuration procedure, refer to [Specify Rook daemons placement](../ops-guide/deployment/rook-daemon-place.md).

| Parameter | Description | Default |
|-----------|-------------|---------|
| `global.dockerBaseUrl` | Global address of Docker registry. | `"registry.mirantis.com"` |
| `global.clusterRelease` | Release version of the Kubernetes cluster. | `""` |
| `global.namespace` | Override for the chart release namespace. | `""` |
| `productSetup` | Specifies the Kubernetes cluster setup. By default, designed for [MKE](https://docs.mirantis.com/mke/3.7/overview.html). Change it for other setups. | `"mke"` |
| `cephDeployment.enabled` | Enable the Pelagia deployment controller. | `true` |
| `cephDeployment.netpolEnabled` | Enable creation of network policy. | `true` |
| `cephDeployment.installSharedNamespace` | Install a namespace for the Openstack-Ceph communication. | `true` |
| `cephDeployment.openstackSharedNamespace` | Namespace for the Openstack-Ceph communication and secrets sharing. | `"openstack-ceph-shared"` |
| `lcmConfig.rgwPublicAccessServiceSelector` | Label of the service or proxy exposing RGW to public access. | `"external_access=rgw"` |
| `lcmConfig.diskDaemonPortParameter` | Port for the disk daemon API. | `9999` |
| `lcmConfig.diskDaemonNodeSelector` | Label for disk daemon placement. | `"ceph_role_osd=true"` |
| `lcmConfig.cephDaemonsetLabelExclude` | Label for nodes where no Ceph daemons must be scheduled. | `""` |
| `controllers.cephdeployment.replicas` | Replica count for Pelagia deployment controllers. | `3` |
| `controllers.lcm.replicas` | Replica count for Pelagia LCM controllers. | `3` |
| `cephRelease` | Pin the Ceph release for the current setup. If empty, uses the latest available release for the current version. | `""` |
| `rookConfig.configureRook` | Configure Rook using the Pelagia Helm chart. | `true` |
| `rookConfig.rookNamespace` | Rook namespace. | `"rook-ceph"` |
| `rookConfig.rookOperatorReplicas` | Replica count for Rook Operator. | `1` |
| `rookConfig.rookOperatorPlacement.affinity` | Affinity settings for the Rook Operator placement. | `{"nodeAffinity": {"preferredDuringSchedulingIgnoredDuringExecution": [{"weight": 100, "preference": {"matchExpressions": [{"key": "ceph_role_mon","operator": "In","values": ["true"]}]}}]}}` |
| `rookConfig.rookOperatorPlacement.nodeSelector` | Node selector for the Rook Operator placement. | `{}` |
| `rookConfig.rookOperatorPlacement.tolerations` | Toleration settings for the Rook Operator placement. | `[]` |
| `rookConfig.csiPlacement.nodeAffinity.csiprovisioner` | Node affinity settings for CSI provisioner. | `""` |
| `rookConfig.csiPlacement.nodeAffinity.csiplugin` | Node affinity settings for CSI plugins. | `"ceph-daemonset-available-node=true"` |
| `rookConfig.csiPlacement.tolerations.csiprovisioner` | Toleration settings for CSI provisioner. | `""` |
| `rookConfig.csiPlacement.tolerations.csiplugin` | Toleration settings for CSI plugins. | `""` |
| `rookConfig.rookDiscoverPlacement.nodeAffinity` | Node affinity settings for Rook discover daemon. | `"ceph-daemonset-available-node=true;ceph_role_osd=true"` |
| `rookConfig.rookDiscoverPlacement.tolerations` | Toleration settings for Rook discover daemon. | `""` |
| `rookConfig.csiKubeletPath` | Path to kubelet on a host. | `""` |
| `rookConfig.csiCephFsEnabled` | Enable CephFS support in Rook. | `true` |
| `rookConfig.csiNfsEnabled` | Enable NFS support in Rook. | `false` |
| `rookConfig.csiAddonsEnabled` | Enable CSI add-ons support in Rook. | `false` |
| `rookConfig.volumeSnapshotsEnabled` | Enable volume snapshots classes support in Rook. | `false` |
| `snapshot-controller.enabled` | Enable the `snapshot-controller` deployment. For available `snapshot-controller` options, see [values.yaml](https://github.com/Mirantis/pelagia/blob/main/charts/snapshot-controller/values.yaml). | `true` |

You can also specify custom images for deployment of test environments.
Ceph and Rook images are derived from the `cephRelease` value.

??? "Configuration example for Ceph and Rook images"

    ```yaml
    images:
      pelagia:
        repository: pelagia/pelagia
        tag: 1.6.0
        fullName: ""
        pullPolicy: Always
      rook:
        operator:
          repository: pelagia/rook
          tag:
            latest: &latestImageRook v1.18.8-6
            squid: *latestImageRook
            reef: *latestImageRook
      ceph:
        repository: pelagia/ceph
        tag:
          latest: &latestImageCeph v19.2.3-18.cve
          squid: *latestImageCeph
          reef: v18.2.7-6.release
      csi:
        ceph:
          repository: pelagia/cephcsi
          tag:
            latest: &latestImageCephCSI v3.15.0-4.cve
            squid: *latestImageCephCSI
            reef: *latestImageCephCSI
        registrar:
          repository: pelagia/cephcsi-registrar
          tag:
            latest: &latestImageRegistar v2.13.0-11.cve
            squid: *latestImageRegistar
            reef: *latestImageRegistar
        provisioner:
          repository: pelagia/cephcsi-provisioner
          tag:
            latest: &latestImageProvisioner v5.2.0-11.cve
            squid: *latestImageProvisioner
            reef: *latestImageProvisioner
        snapshotter:
          repository: pelagia/cephcsi-snapshotter
          tag:
            latest: &latestImageSnapshotter v8.2.1-2.cve
            squid: *latestImageSnapshotter
            reef: *latestImageSnapshotter
        attacher:
          repository: pelagia/cephcsi-attacher
          tag:
            latest: &latestImageAttacher v4.8.1-2.cve
            squid: *latestImageAttacher
            reef: *latestImageAttacher
        resizer:
          repository: pelagia/cephcsi-resizer
          tag:
            latest: &latestImageResizer v1.13.2-11.cve
            squid: *latestImageResizer
            reef: *latestImageResizer
    ```
