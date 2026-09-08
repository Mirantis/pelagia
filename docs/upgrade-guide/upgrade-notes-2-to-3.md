# Upgrade notes 2.x to 3.x

Pelagia 2.x must be upgraded to version 3.x.
For Pelagia release notes, refer to [Pelagia Releases](https://github.com/Mirantis/pelagia/releases/).

## Breaking changes

* Rook is upgraded to v1.20. For upgrade details, see [Rook upgrade](https://rook.io/docs/rook/v1.20/Upgrade/rook-upgrade/).
  
    Pelagia now has an ability to manage CSI Drivers and OperatorConfig through `CephDeployment` spec with new `spec.csi` field. For details,
    see [CephDeployment resource](../custom-resources/cephdeployment.md).

* Ceph CSI operator is updated to v1.0.4 version.

    For details, see [Ceph CSI operator release notes](https://github.com/ceph/ceph-csi-operator/releases#release-v1.0.4).

## Pre-upgrade steps

In this release, the Rook has no more ability to configure CSI resources. Its configuration now is supported by Pelagia.
If your setup has next chart options:

.. yaml:

  rook:
    rookConfig:
      csiPlacement:
        nodeAffinity:
          csiprovisioner: "<csi-provisioner-affinity>"
          csiplugin: "<csi-plugin-affinity>"
        tolerations:
          csiplugin: "<csi-plugin-toleratios>"
          csiprovisioner: "<csi-provisioner-toleratios>"
      csiKubeletPath: "<kubelet-path>"
      csiCephFsEnabled: "<cephfs-enabled>"
      csiNfsEnabled: "<nfs-enabled>"
      csiAddonsEnabled: "<addons-enabled>"

They should be migrated to new place:

.. yaml:

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
        nodePlugin: "<csi-plugin-toleratios>"
        controllerPlugin: "<csi-provisioner-toleratios>"
    addons: "<addons-enabled>"

## Post-upgrade steps

Complete the following steps after upgrading Pelagia to 3.x.

1. Remove if present next options from chart:

.. yaml:

  rook:
    rookConfig:
      csiPlacement:
        nodeAffinity:
          csiprovisioner: "<csi-provisioner-affinity>"
          csiplugin: "<csi-plugin-affinity>"
        tolerations:
          csiplugin: "<csi-plugin-toleratios>"
          csiprovisioner: "<csi-provisioner-toleratios>"
      csiKubeletPath: "<kubelet-path>"
      csiCephFsEnabled: "<cephfs-enabled>"
      csiNfsEnabled: "<nfs-enabled>"
      csiAddonsEnabled: "<addons-enabled>"
