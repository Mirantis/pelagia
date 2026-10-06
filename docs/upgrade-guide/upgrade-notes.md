# Upgrade notes 2.x to 3.x

!!! warning

    This page describes the upgrade to Pelagia 3.x, which is under development
    and not released yet. The instructions below apply only after Pelagia 3.0
    is published. Until then, do not use them on production environments.

This page describes upgrading Pelagia 2.x to 3.x.
For Pelagia release notes, refer to [Pelagia Releases](https://github.com/Mirantis/pelagia/releases/).

## Breaking changes

* Rook is upgraded to v1.20. For upgrade details, see [Rook upgrade](https://rook.io/docs/rook/v1.20/Upgrade/rook-upgrade/).

    Pelagia now manages Ceph CSI drivers and OperatorConfig through the new `spec.csi` field of `CephDeployment`. For details,
    see [CephDeployment resource](../custom-resources/cephdeployment.md).

* Ceph Tentacle is upgraded to v20.2.4 and Ceph Squid - to v19.2.6.

    This version of Ceph Tentacle introduces a new client authentication method with a more secure AES256K cipher type.
    For details, see [Ceph blog: Tentacle 20.2.4 and Squid 19.2.6 hotfixes for CVEs](https://ceph.io/en/news/blog/2026/v20-2-4-v19-2-6-combo-released/).

    During upgrade, Pelagia automatically ensures backward compatibility with all existing keys based on AES encryption.
    All Ceph-related keyrings for daemons (`mon`, `mgr`, `osd`, `rgw`, `mds`) are rotated automatically. All other clients (including
    Ceph CSI) remain on the legacy AES encryption method and must be rotated manually after upgrade.
    For details, see [Rook documentation: CephX Keys and Rotation](https://rook.io/docs/rook/v1.20/Storage-Configuration/Advanced/cephx-key-rotation/).

    Ceph CSI plugins require kernel version 7.0+ to support AES256K. Pelagia pins the legacy AES for Ceph CSI during upgrade.
    Because the legacy cipher and some legacy keyrings remain active, Pelagia mutes the following Ceph health warnings during upgrade:

    - AUTH_INSECURE_ROTATING_SERVICE_KEY_TYPE
    - AUTH_INSECURE_CLIENT_KEY_TYPE
    - AUTH_INSECURE_KEYS_ALLOWED
    - AUTH_INSECURE_KEYS_CREATABLE
    - AUTH_EMERGENCY_CIPHERS_SET

    You can remove these mutes once all keyrings are switched to the new AES256K format.

* Ceph CSI Operator is updated to v1.0.4.

    For details, see [Ceph CSI Operator GitHub project: Release notes](https://github.com/ceph/ceph-csi-operator/releases#release-v1.0.4).

* Ingress support is disabled by default. The default value of `lcmConfig.useIngress` is changed from `true` to `false`
  due to the [NGINX Ingress retirement](https://kubernetes.io/blog/2025/11/11/ingress-nginx-retirement/).

    After the upgrade, Pelagia removes the existing Ingress for Ceph Object Storage (RGW) unless `lcmConfig.useIngress: true`
    is set explicitly in Helm values. If your setup still uses Ingress, switch to the Gateway API before the upgrade or set
    `lcmConfig.useIngress: true` to keep Ingress temporarily. For the Gateway API options, see [Helm chart values](../configuration/helm-values.md).

* Management of VolumeSnapshotClasses is moved from the Helm chart to the Pelagia Controller.

    The new `cephDeployment.manageVolumeSnapshotClasses` Helm option controls this behavior. It is enabled by default and
    requires the `volumesnapshotclasses.snapshot.storage.k8s.io` CRD, which is installed by default with
    `snapshot-controller`.

    Existing classes with the default names are kept. To manage the classes on your own, set the option to `false`. Pelagia then neither creates nor removes them.

    The `rook.rookConfig.volumeSnapshotsEnabled` option is deprecated and will be removed in a future release. Keep it
    until the upgrade is complete, and then remove it from your Helm values.

## Pre-upgrade steps

Complete the following steps before upgrading Pelagia to 3.x:

1. Because Rook no longer configures CSI resources, add the following options to the new location in the Pelagia Helm chart values. Do not move or remove the existing options during the upgrade: both locations must remain in place until the upgrade completes. You will remove old options during post-upgrade.

    Existing location of the options (keep as is):

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

    New location of the options (add to the `cephDeployment` section):

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

2. Define how Pelagia handles `VolumeSnapshotClasses` after the upgrade. Pelagia Controller now manages default
   `VolumeSnapshotClasses` for the Ceph CSI RBD and CephFS drivers, and the `cephDeployment.manageVolumeSnapshotClasses`
   option, which is enabled by default, controls this behavior. Depending on your current configuration, do the following:

    * If `rook.rookConfig.volumeSnapshotsEnabled` is not set or is explicitly set to `false` in the Helm chart values, disable the management of
      `VolumeSnapshotClasses` by Pelagia Controller to avoid creation of new default classes during the upgrade:

        ```yaml
        cephDeployment:
          ...
          manageVolumeSnapshotClasses: false
        ```

    * If `rook.rookConfig.volumeSnapshotsEnabled` is set to `true`, no action is required.
      All existing `VolumeSnapshotClasses` remain unchanged during the upgrade.

        ```yaml
        rook:
          rookConfig:
            ...
            volumeSnapshotsEnabled: true
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

2. Once all Ceph clients are rotated using the aes256k method, remove the following parameters from the `CephDeployment` spec:

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

     To verify that all clients are using the aes256k method:

     ```bash
     kubectl -n rook-ceph exec -it \
         deploy/pelagia-ceph-toolbox \
         -- ceph auth dump-keys -f json-pretty
     ```

     No entities with the aes encryption must be present in the output.
