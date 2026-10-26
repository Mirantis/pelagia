---
description: How to enable RADOS Gateway Object Storage, including configuration steps,
  RADOS Gateway deployment details, and key considerations for RGW enablement.
keywords: pelagia, ceph rgw, enable ceph rgw, rados gateway, ceph object storage,
  ceph rgw deployment, swift, rgw configuration
---

<a id="rgw-enable-ceph-rgw-object-storage"></a>

# Enable Ceph RADOS Gateway Object Storage

Pelagia enables you to deploy Ceph RADOS Gateway (RGW) Object Storage
instances and automatically manage its resources such as users.

Pelagia has an integration for Ceph Object Storage with OpenStack Object Storage (`Swift`) provided by Rockoon.

<a name="rgw-ceph-rgw-object-storage-parameters"></a>
## Ceph RGW Object Storage parameters

{% include "../../../snippets/rgwParameters.md" %}

## Enable Ceph RGW Object Storage

1. Open the `CephDeployment` resource for editing:
   ```bash
   kubectl -n pelagia edit cephdpl <name>
   ```
   Substitute `<name>` with the name of your `CephDeployment`.
2. Update the `objectStorage.objectStores` section specification using the configuration reference above. For example:
     ```yaml
     objectStores:
     - name: rgw-store
       spec:
         dataPool:
           deviceClass: hdd
           erasureCoded:
             codingChunks: 1
             dataChunks: 2
           failureDomain: host
         metadataPool:
           deviceClass: hdd
           failureDomain: host
           replicated:
             size: 3
         gateway:
           instances: 3
           port: 80
           securePort: 8443
         preservePoolsOnDelete: false
     ```
3. Save the changes and exit the editor.

To expose the Ceph Object Gateway on a public endpoint and configure TLS,
use the Gateway API. For details, see
[Configure Ceph Object Gateway TLS](./rgw-tls.md#rgw-tls-configure-ceph-object-gateway-tls).

<a name="rgw-ceph-with-multiple-objectstores"></a>

## Configure multiple Ceph RGW Object Storage instances

You can configure multiple Ceph RGW Object Storage instances within one cluster for a multi-realm
configuration. Each realm has a globally unique namespace and can contain either a single
or multiple zonegroups. Users and buckets accessible within one realm are not accessible in other realms.

In a multi-realm configuration, one realm must be always marked as default.

??? "Example multi-realm configuration for Ceph RGW Object Storage"

    ```yaml
    spec:
      ...
      objectStorage:
        objectStores:
        - name: rgw-a
          spec:
            dataPool: {}
            gateway:
              instances: 3
              port: 8082
              securePort: 8443
            metadataPool: {}
            preservePoolsOnDelete: false
            zone:
              name: zone-a
        - name: rgw-b
          spec:
            dataPool: {}
            gateway:
              instances: 3
              port: 18082
              securePort: 18443
            metadataPool: {}
            preservePoolsOnDelete: false
            zone:
              name: zone-b
        realms:
        - name: realm-a
          spec:
            defaultRealm: true
        - name: realm-b
          spec:
            defaultRealm: false
        zonegroups:
        - name: zonegroup-a
          spec:
            realm: realm-a
        - name: zonegroup-b
          spec:
            realm: realm-b
        zones:
        - name: zone-a
          spec:
            dataPool:
              deviceClass: hdd
              replicated:
                size: 3
              failureDomain: host
            metadataPool:
              deviceClass: hdd
              failureDomain: host
              replicated:
                size: 3
            zoneGroup: zonegroup-a
        - name: zone-b
          spec:
            dataPool:
              deviceClass: custom
              replicated:
                size: 3
              failureDomain: host
            metadataPool:
              deviceClass: custom
              failureDomain: host
              replicated:
                size: 3
            zoneGroup: zonegroup-b
    ```

When you configure multiple object stores, define the `.rgw.root` pool explicitly.
Otherwise, it inherits the parameters of the first RGW pool created, which can change
when object stores are added or removed.

{% include "../../../snippets/rgwPoolsNaming.md" %}

```yaml
spec:
  blockStorage:
    pools:
    - name: .rgw.root
      spec:
        deviceClass: hdd
        enableCrushUpdates: true
        replicated:
          size: 3
      storageClassOpts:
        skipClassCreate: true
      useAsFullName: true
```

<a name="rgw-ceph-with-shared-pools"></a>

## Configure Ceph RGW Object Storage with shared and placement pools

You can configure Ceph RGW Object Storage to use shared Ceph RGW metadata and data pools,
which allows reducing the overhead of additional Ceph pools for each additional object store.

To configure Ceph RGW with shared pools, specify all pools to be used as Ceph RGW pools in the `spec.blockStorage.pools` section of the `CephDeployment` spec.

{% include "../../../snippets/rgwPoolsNaming.md" %}

??? "Example configuration of placement pools for Ceph RGW"

    ```yaml
    spec:
      blockStorage:
        pools:
        - name: .rgw.root
          spec:
            deviceClass: hdd
            enableCrushUpdates: true
            replicated:
              size: 3
          storageClassOpts:
            skipClassCreate: true
          useAsFullName: true
        - name: rgw-data-reduced
          spec:
            deviceClass: hdd
            enableCrushUpdates: true
            replicated:
              size: 3
          storageClassOpts:
            skipClassCreate: true
          useAsFullName: true
        - name: rgw-data
          spec:
            deviceClass: hdd
            enableCrushUpdates: true
            replicated:
              size: 3
          storageClassOpts:
            skipClassCreate: true
          useAsFullName: true
        - name: rgw-meta
          spec:
            deviceClass: hdd
            enableCrushUpdates: true
            replicated:
              size: 3
          storageClassOpts:
            skipClassCreate: true
          useAsFullName: true
      ...
      objectStorage:
        objectStores:
        - name: rgw-store
          spec:
            gateway:
              instances: 2
              port: 80
              securePort: 18443
            preservePoolsOnDelete: false
            sharedPools:
              poolPlacements:
              - dataPoolName: rgw-data
                default: true
                metadataPoolName: rgw-meta
                name: us
                storageClasses:
                - dataPoolName: rgw-data-reduced
                  name: REDUCED_REDUNDANCY
      ...
    ```

!!! info "See also"

    Rook documentation:

    * [ObjectStorage with shared pools](https://rook.io/docs/rook/v1.20/Storage-Configuration/Object-Storage-RGW/object-storage/#create-local-object-stores-with-shared-pools)
    * [ObjectStorage with placement pools](https://rook.io/docs/rook/v1.20/Storage-Configuration/Object-Storage-RGW/object-storage/#create-local-object-stores-with-pool-placements)
