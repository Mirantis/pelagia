# Upgrade notes 2.x to 3.x

!!! warning

    This page describes the upgrade to Pelagia 3.x, which is under development
    and not released yet. The instructions below apply only after Pelagia 3.0
    is published. Until then, do not use them on production environments.

This page describes upgrading Pelagia 2.x to 3.x.
For Pelagia release notes, refer to [Pelagia Releases](https://github.com/Mirantis/pelagia/releases/).

## Breaking changes

* Rook is upgraded to v1.20. For upgrade details, see [Rook upgrade](https://rook.io/docs/rook/v1.20/Upgrade/rook-upgrade/).

    Pelagia now manages CSI drivers and OperatorConfig through the new `spec.csi` field of `CephDeployment`. For details,
    see [CephDeployment resource](../custom-resources/cephdeployment.md).

* Ceph CSI Operator is updated to v1.0.4.

    For details, see [Ceph CSI Operator release notes](https://github.com/ceph/ceph-csi-operator/releases#release-v1.0.4).

* Ingress support is disabled by default. The default value of `lcmConfig.useIngress` is changed from `true` to `false`
  due to the [NGINX Ingress retirement](https://kubernetes.io/blog/2025/11/11/ingress-nginx-retirement/).

    After the upgrade, Pelagia removes the existing Ingress for Ceph Object Storage (RGW) unless `lcmConfig.useIngress: true`
    is set explicitly in Helm values. If your setup still uses Ingress, switch to the Gateway API before the upgrade or set
    `lcmConfig.useIngress: true` to keep Ingress temporarily. For the Gateway API options, see [Helm chart values](../configuration/helm-values.md).

## Pre-upgrade steps

In this release, Rook no longer configures CSI resources. Pelagia now manages the CSI configuration.
If your setup has the following chart options:

```yaml
rook:
  rookConfig:
    csiPlacement:
      nodeAffinity:
        csiprovisioner: "<csi-provisioner-affinity>"
        csiplugin: "<csi-plugin-affinity>"
      tolerations:
        csiplugin: "<csi-plugin-tolerations>"
        csiprovisioner: "<csi-provisioner-tolerations>"
    csiKubeletPath: "<kubelet-path>"
    csiCephFsEnabled: "<cephfs-enabled>"
    csiNfsEnabled: "<nfs-enabled>"
    csiAddonsEnabled: "<addons-enabled>"
```

Move them to the following options before the upgrade:

```yaml
cephDeployment:
  csi:
    kubeletPath: "<kubelet-path>"
    defaultDriversCreate:
      cephfs: "<cephfs-enabled>"
      nfs: "<nfs-enabled>"
    placement:
      nodeAffinity:
        controllerPlugin: "<csi-provisioner-affinity>"
        nodePlugin: "<csi-plugin-affinity>"
      tolerations:
        nodePlugin: "<csi-plugin-tolerations>"
        controllerPlugin: "<csi-provisioner-tolerations>"
    addons: "<addons-enabled>"
```

<a id="upgrade-notes-post-upgrade-steps"></a>

## Post-upgrade steps

Complete the following steps after upgrading Pelagia to 3.x:

1. Remove the following options from the Helm chart if present:

    ```yaml
    rook:
      rookConfig:
        csiPlacement:
          nodeAffinity:
            csiprovisioner: "<csi-provisioner-affinity>"
            csiplugin: "<csi-plugin-affinity>"
          tolerations:
            csiplugin: "<csi-plugin-tolerations>"
            csiprovisioner: "<csi-provisioner-tolerations>"
        csiKubeletPath: "<kubelet-path>"
        csiCephFsEnabled: "<cephfs-enabled>"
        csiNfsEnabled: "<nfs-enabled>"
        csiAddonsEnabled: "<addons-enabled>"
    ```
