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
| `cephDeployment.drainRequestLabelKey` | Label key marking node as drained. | `""` |
| `cephDeployment.drainReadyLabelKey` | Label key marking node as ready to be drained. | `""` |
| `lcmConfig.rookNamespace` | Rook namespace name used across the Pelagia deployment. | `"rook-ceph"` |
| `lcmConfig.rgwPublicAccessServiceSelector` | Label of the service or proxy exposing RGW to public access. | `"external_access=rgw"` |
| `lcmConfig.diskDaemonPortParameter` | Port for the disk daemon API. | `9999` |
| `lcmConfig.diskDaemonNodeSelector` | Label for disk daemon placement. | `"ceph_role_osd=true"` |
| `lcmConfig.cephDaemonsetLabelExclude` | Label for nodes where no Ceph daemons must be scheduled. | `""` |
| `lcmConfig.gatewayAPIEnabled` | Enable usage of the Gateway API. | `true` |
| `lcmConfig.gatewayName` | Name of the `Gateway` object used by default. | `""` |
| `lcmConfig.gatewayNamespace` | Namespace of the `Gateway` object used by default. | `""` |
| `lcmConfig.useIngress` | Deprecated. Enable support for Ingress usage. Will be removed in the following release due to [Ingress deprecation](https://kubernetes.io/blog/2025/11/11/ingress-nginx-retirement/). | `true` |
| `controllers.cephdeployment.replicas` | Replica count for Pelagia deployment controllers. | `3` |
| `controllers.lcm.replicas` | Replica count for Pelagia LCM controllers. | `3` |
| `cephRelease` | Pin the Ceph release for the current setup. If empty, uses the latest available release for the current version. | `""` |
| `rook.enabled` | Enable the `rook` deployment using the Pelagia Helm chart. For available `rook` options, see [values.yaml](https://github.com/Mirantis/pelagia/blob/main/charts/rook/values.yaml). | `true` |
| `rook.rookConfig.rookNamespace` | Rook namespace. By default, inherited from the `lcmConfig.rookNamespace` value defined in the main Pelagia chart. | `"rook-ceph"` |
| `rook.rookConfig.rookOperatorReplicas` | Replica count for Rook Operator. | `1` |
| `rook.rookConfig.rookOperatorPlacement.affinity` | Affinity settings for the Rook Operator placement. | `{"nodeAffinity": {"preferredDuringSchedulingIgnoredDuringExecution": [{"weight": 100, "preference": {"matchExpressions": [{"key": "ceph_role_mon","operator": "In","values": ["true"]}]}}]}}` |
| `rook.rookConfig.rookOperatorPlacement.nodeSelector` | Node selector for the Rook Operator placement. | `{}` |
| `rook.rookConfig.rookOperatorPlacement.tolerations` | Toleration settings for the Rook Operator placement. | `[]` |
| `rook.rookConfig.csiPlacement.nodeAffinity.csiprovisioner` | Node affinity settings for CSI provisioner. | `""` |
| `rook.rookConfig.csiPlacement.nodeAffinity.csiplugin` | Node affinity settings for CSI plugins. | `"ceph-daemonset-available-node=true"` |
| `rook.rookConfig.csiPlacement.tolerations.csiprovisioner` | Toleration settings for CSI provisioner. | `""` |
| `rook.rookConfig.csiPlacement.tolerations.csiplugin` | Toleration settings for CSI plugins. | `""` |
| `rook.rookConfig.rookDiscoverPlacement.nodeAffinity` | Node affinity settings for the `rook-discover` daemon. | `"ceph-daemonset-available-node=true;ceph_role_osd=true"` |
| `rook.rookConfig.rookDiscoverPlacement.tolerations` | Toleration settings for the `rook-discover` daemon. | `""` |
| `rook.rookConfig.csiKubeletPath` | Path to kubelet on a host. | `""` |
| `rook.rookConfig.csiCephFsEnabled` | Enable CephFS support in Rook. | `true` |
| `rook.rookConfig.csiNfsEnabled` | Enable NFS support in Rook. | `false` |
| `rook.rookConfig.csiAddonsEnabled` | Enable CSI add-ons support in Rook. | `false` |
| `rook.rookConfig.volumeSnapshotsEnabled` | Enable volume snapshot classes support in Rook. | `false` |
| `ceph-csi-operator.enabled` | Enable the `ceph-csi-operator` deployment. For available `ceph-csi-operator` options, see [values.yaml](https://github.com/Mirantis/pelagia/blob/main/charts/ceph-csi-operator/values.yaml). | `true` |
| `ceph-csi-operator.csiOperatorConfig.rookNamespace` | Rook namespace. By default, inherited from the `lcmConfig.rookNamespace` value defined in the main Pelagia chart. | `"rook-ceph"` |
| `ceph-csi-operator.csiOperatorConfig.placement.affinity` | Affinity settings for the `ceph-csi-operator` deployment placement. | `{}` |
| `ceph-csi-operator.csiOperatorConfig.placement.tolerations` | Tolerations for the `ceph-csi-operator` deployment to enable running on nodes with particular taints. | `[]` |
| `snapshot-controller.enabled` | Enable the `snapshot-controller` deployment. For available `snapshot-controller` options, see [values.yaml](https://github.com/Mirantis/pelagia/blob/main/charts/snapshot-controller/values.yaml). | `true` |
| `snapshot-controller.snapshotControllerConfig.affinity` | Affinity settings for the `snapshot-controller` placement. | `{}` |
| `snapshot-controller.snapshotControllerConfig.nodeSelector` | Node selector for the `snapshot-controller` deployment to run on nodes with specific labels. | `{}` |
| `snapshot-controller.snapshotControllerConfig.tolerations` | Tolerations for the `snapshot-controller` deployment to enable running on nodes with particular taints. | `[]` |

## Custom images and settings

You can specify custom images for testing environments.
Ceph images are derived from the `cephRelease` value.
The following example illustrates available Ceph images based on the Ceph release:

```yaml
images:
  ceph:
    repository: pelagia/ceph
    tag:
      latest: &latestImageCeph v20.2.1-3.release
      squid: v19.2.3-25.cve
      tentacle: *latestImageCeph
```

Also, you can specify custom settings for dependency charts, such as Rook, snapshot controller, and Ceph CSI operator
if they are installed by the Pelagia chart. For example:

```yaml
ceph-csi-operator:
  csiOperatorConfig:
    placement:
      tolerations:
      - effect: NoSchedule
        key: node-role.kubernetes.io/controlplane
        operator: Exists
rook:
  rookConfig:
    csiCephFsEnabled: false
    csiPlacement:
      tolerations:
        csiplugin: |
        - effect: NoSchedule
          key: node-role.kubernetes.io/controlplane
          operator: Exists
snapshot-controller:
  snapshotControllerConfig:
    nodeSelector:
      disktype: ssd
```
