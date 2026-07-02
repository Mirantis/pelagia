# Upgrade notes 1.x to 2.x

Pelagia 1.x must be upgraded to version 2.x.
For Pelagia release notes, refer to [Pelagia Releases](https://github.com/Mirantis/pelagia/releases/).

## Breaking changes

* The default Ceph release is set to [Ceph Tentacle](https://docs.ceph.com/en/latest/releases/tentacle/).

* Rook is upgraded to v1.19. For upgrade details, see [Rook upgrade](https://rook.io/docs/rook/v1.19/Upgrade/rook-upgrade/).

* Pelagia now has Rook as separate chart, which allows user disable Pelagia-based Rook setup and use
  another Rook chart.

* Ceph CSI operator is enabled by default.

    Previously, the CSI driver was automatically configured by Rook.
    Now, the CSI operator has been factored out of Rook to run independently to manage the Ceph-CSI driver.
    For details, see [Ceph CSI operator README](https://github.com/ceph/ceph-csi-operator/tree/v0.6.0#ceph-csi-operator).

* CephDeployment API is refactored and now uses pure Rook API for several fields.
  For details, see [CephDeployment resource](../custom-resources/cephdeployment.md).
  `CephDeployment` is automatically migrated to a new API during upgrade, no manual steps are required.

* After changing the `CephDeployment` API, Pelagia supports multiple ObjectStore (Ceph RGW) instances.
  Consider the following changes:
    * The RGW name of `StorageClass` created by default has changed to `<object-store-name>-bucket`.

        The previously created `StorageClass` for a single ObjectStore (RGW) instance named `rgw-storage-class` is still present in the environment for backward compatibility.
        All existing buckets are not affected, but for newly created ones a new `StorageClass` must be used.
        Removal of `StorageClass` with old names must be done manually.

    * The self-signed ObjectStore (RGW) SSL certificate is regenerated and renamed from `rgw-ssl-certificate` to
      `<object-store-name>-ssl-certificate`.

        If the current description of ObjectStore (RGW) provides certificates in the `SSLCert` or `SSLCertInRef` section (for details, see [CephDeployment CephRGW params](https://mirantis.github.io/pelagia/1.x/custom-resources/cephdeployment/#rados-gateway-parameters)), these certificates will continue to be used and will be moved to the corresponding new sections.
        The `SSLCert` and `SSLCertInRef` sections will be removed.
        Specifying plain SSL certificates directly in the `CephDeployment` spec is now prohibited, use a secret reference instead.

    * The multisite realm cannot provide plain realm keys in the spec anymore, for security reasons.
      You must create the corresponding secret as described in [Rook documentation: Getting Realm Access Key and Secret Key](https://rook.io/docs/rook/v1.19/Storage-Configuration/Object-Storage-RGW/ceph-object-multisite/#getting-realm-access-key-and-secret-key).
      For existing environments, no actions are required.

    * Serving RGW DNS names must be now set using the ObjectStore (RGW) spec definition.
      This enables a more flexible configuration.
      For details, see [Rook documentation: Hosting Settings](https://rook.io/docs/rook/v1.19/CRDs/Object-Storage/ceph-object-store-crd/#hosting-settings).
      Existing environments do not require pre-upgrade steps, but consider switching to the new flow after upgrade.

* Gateway API support.

    Ingress NGINX controller is [deprecated](https://kubernetes.io/blog/2025/11/11/ingress-nginx-retirement/), but
    continues operating after Pelagia upgrade.
    However, its support will be removed in following release. Therefore, consider switching to the Gateway API after upgrade.
    For details, see the post-upgrade steps below.

## Pre-upgrade steps

As of Rook setup is moved now to separate chart and used as dependency in Pelagia, if current setup
has custom settings based in `values.rookConfig` for helm chart, they should be copied under `values.rook.rookConfig`.
Old `values.rookConfig` should be kept until upgrade is done.

## Post-upgrade steps

Complete the following steps after upgrading Pelagia to 2.x. For description
of the changes that require these steps, see the *Breaking changes* section above.

1. Optional. Manually remove the existing `StorageClass` for the ObjectStore (RGW) instances named `rgw-storage-class`.
2. Switch to setting RGW DNS names using the ObjectStore (RGW) spec definition.
   This enables a more flexible configuration.
   For details, see [Rook documentation: Hosting Settings](https://rook.io/docs/rook/v1.19/CRDs/Object-Storage/ceph-object-store-crd/#hosting-settings).
3. Switch to the [Gateway API](https://gateway-api.sigs.k8s.io/guides/getting-started/introduction/) due to
   the deprecation of the Ingress NGINX controller. The `Gateway` object and Controller must be configured separately.

     To configure Pelagia after switching to the Gateway API, use the corresponding `Gateway` object.
     For details, see [Configuration Reference](../configuration/index.md) and [CephDeployment resource](../custom-resources/cephdeployment.md).
4. If current Helm values contain `rookConfig` section it can be safelly removed.
