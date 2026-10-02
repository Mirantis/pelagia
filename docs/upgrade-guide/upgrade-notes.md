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

* New Ceph Tentacle v20.2.4 and Ceph Squid v19.2.6 versions, with new Ceph Authx AES256k encryption.

  Ceph Tentacle v20.2.4 introduced new Ceph authx method for client authorization - aes256k. For additional details, see
  [Ceph blog](https://ceph.io/en/news/blog/2026/v20-2-4-v19-2-6-combo-released/)

  During upgrade Pelagia automatically set backward compatibility with all existing keys based on aes encryption. All
  Ceph related keyrings for daemons (mon, mgr, osd, rgw, mds) will be rotated automatically. All other clients (including
  CSI clients) will remain on old aes encryption method and should be rotated manually.
  See [Rook key rotation](https://rook.io/docs/rook/v1.20/Storage-Configuration/Advanced/cephx-key-rotation/) for details.

  CSI plugins will require kernel version 7.0+ to support new aes256k. Pelagia will pin old aes for CSI during upgrade.
  Since old aes will remain active, as well as some keyrings, Pelagia will add mutes for the following list of Ceph health
  warnings: AUTH_INSECURE_ROTATING_SERVICE_KEY_TYPE, AUTH_INSECURE_CLIENT_KEY_TYPE, AUTH_INSECURE_KEYS_ALLOWED,
  AUTH_INSECURE_KEYS_CREATABLE, AUTH_EMERGENCY_CIPHERS_SET. Mutes can be removed once all keyrings switched to new aes256k format.

* Ceph CSI Operator is updated to v1.0.4.

    For details, see [Ceph CSI Operator release notes](https://github.com/ceph/ceph-csi-operator/releases#release-v1.0.4).

* Ingress support is disabled by default. The default value of `lcmConfig.useIngress` is changed from `true` to `false`
  due to the [NGINX Ingress retirement](https://kubernetes.io/blog/2025/11/11/ingress-nginx-retirement/).

    After the upgrade, Pelagia removes the existing Ingress for Ceph Object Storage (RGW) unless `lcmConfig.useIngress: true`
    is set explicitly in Helm values. If your setup still uses Ingress, switch to the Gateway API before the upgrade or set
    `lcmConfig.useIngress: true` to keep Ingress temporarily. For the Gateway API options, see [Helm chart values](../configuration/helm-values.md).

* Management of VolumeStorageClasses moved from Helm chart to Pelagia controller. New helm option `cephDeployment.manageVolumeSnapshotClasses`
  allows to switch management of VolumeStorageClasses, which is enabled by default and requires `volumegroupsnapshotclasses.groupsnapshot.storage.k8s.io` be present (which is present by default with enabled snapshot-controller). Option `rook.rookConfig.volumeSnapshotsEnabled` is deprecated and will be removed in future, but should be kept until upgrade is done and then removed.

## Pre-upgrade steps

* In this release, Rook no longer configures CSI resources. Pelagia now manages the CSI configuration.
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
  ...
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

* If setup enables option `rook.rookConfig.volumeSnapshotsEnabled`

```yaml
rook:
  rookConfig:
    ...
    volumeSnapshotsEnabled: true
```

all existing `VolumeSnapshotClasses` will be unchanged during upgrade.

In case if that option is not present in options or explicitly disabled, add the following
option:

```yaml
cephDeployment:
  ...
  manageVolumeSnapshotClasses: false
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
        volumeSnapshotsEnabled: <value>
    ```

2. Once all Ceph clients rotated with new aes256k method, remove next parameters from
   `CephDeployment` spec:

   ```yaml
   spec:
     cluster:
      ...
      healthCheck:
        muteHealthWarning:
          AUTH_EMERGENCY_CIPHERS_SET:
            policy: mute
          AUTH_INSECURE_CLIENT_KEY_TYPE:
            policy: mute
          AUTH_INSECURE_KEYS_ALLOWED:
            policy: mute
          AUTH_INSECURE_KEYS_CREATABLE:
            policy: mute
          AUTH_INSECURE_ROTATING_SERVICE_KEY_TYPE:
            policy: mute
      ...
      security:
        cephx:
          allowedCiphers:
          - aes
          - aes256k
          csi:
            keyType: aes
          daemon:
            keyType: aes256k
   ```

  To verify all clients are using aes256k method:

  ```bash
   kubectl -n rook-ceph exec -it \
       deploy/pelagia-ceph-toolbox \
       -- ceph auth dump-keys -f json-pretty
   ```

  No entities with aes encryprion should be present.
