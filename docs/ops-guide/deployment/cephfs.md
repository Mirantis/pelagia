---
description: How to configure Ceph Shared File System (CephFS) to enable shared read/write file system volumes.
keywords: pelagia, enable cephfs, configure cephfs, cephfs spec, pvs, ceph file system, shared file system,
  persistent volumes, cephfs, sharedfilesystem
---

<a id="cephfs-configure-ceph-shared-file-system-cephfs"></a>

# Configure Ceph Shared File System (CephFS)

The Ceph Shared File System, or CephFS, provides the ability to create
read/write shared file system Persistent Volumes (PVs). These PVs support the
`ReadWriteMany` access mode for the `FileSystem` volume mode.
CephFS deploys its own daemons called MetaData Servers or Ceph MDS. For
details, see [Ceph Documentation: Ceph File System](https://docs.ceph.com/en/{{ ceph_doc_version }}/cephfs/index.html).

!!! note

    By design, the default CephFS data and metadata pools must be `replicated` only.

<a name="cephfs-cephfs-specification-parameters"></a>
## CephFS specification parameters

The `CephDeployment` custom resource (CR) `spec` includes the `sharedFilesystem.cephFilesystems` section
with the following CephFS parameters:

- `name` - CephFS instance name.
- `spec` - CephFS spec definition based on the [Rook API CephFS](https://rook.io/docs/rook/{{ rook_doc_version }}/CRDs/Shared-Filesystem/ceph-filesystem-crd/) specification.

For configuration details, refer to the [CephDeployment API](../../custom-resources/cephdeployment.md#cephdeployment-cephfs-parameters).

Example of the CephFS specification:
```yaml
spec:
  sharedFilesystem:
    cephFilesystems:
    - name: cephfs-store
      spec:
        dataPools:
        - name: default-pool
          deviceClass: ssd
          replicated:
            size: 3
          failureDomain: host
        - name: second-pool
          deviceClass: hdd
          failureDomain: rack
          erasureCoded:
            dataChunks: 2
            codingChunks: 1
        metadataPool:
          deviceClass: nvme
          replicated:
            size: 3
          failureDomain: host
        metadataServer:
          activeCount: 1
          activeStandby: false
          resources: # example, non-prod values
            requests:
              memory: 1Gi
              cpu: 1
            limits:
              memory: 2Gi
              cpu: 2
```

{% include "../../snippets/replicatedSize.md" %}

!!! warning

    Modifying `dataPools` or `metadataPool` on a deployed CephFS has no effect. You can manually adjust pool settings
    through the Ceph CLI. However, for any changes in `dataPools` or `metadataPool`, we recommend re-creating CephFS.

## Configure CephFS

1. Optional. Verify that the default CephFS CSI driver is enabled. Pelagia creates it when the
   `cephDeployment.csi.defaultDriversCreate.cephfs` Helm chart value is `true`, which is the default. If you have
   disabled it, re-enable it by upgrading the Pelagia Helm release:

     ```bash
     helm upgrade --install pelagia-ceph oci://registry.mirantis.com/pelagia/pelagia-ceph --version ${PELAGIA_VERSION} -n pelagia \
          --set cephDeployment.csi.defaultDriversCreate.cephfs=true
     ```

     Pelagia will create the CephFS CSI driver, which deploys the CephFS CSI node and controller plugins.

2. Open the `CephDeployment` CR for editing:
   ```bash
   kubectl -n pelagia edit cephdpl
   ```

3. Update the `sharedFilesystem` section specification as required using the configuration reference above. For example:

     ```yaml
     spec:
       sharedFilesystem:
         cephFilesystems:
         - name: cephfs-store
           spec:
             dataPools:
             - name: cephfs-pool-1
               deviceClass: hdd
               replicated:
                 size: 3
               failureDomain: host
             metadataPool:
               deviceClass: nvme
               replicated:
                 size: 3
               failureDomain: host
             metadataServer:
               activeCount: 1
               activeStandby: false
     ```

4. Define the `mds` role for the corresponding nodes where Ceph MDS daemons should be deployed. We recommend
   labeling only one node with the `mds` role. For example:
   ```yaml
   spec:
     nodes:
     ...
     - name: worker-1
       roles:
       ...
       - mds
   ```

Once CephFS is specified in the `CephDeployment` CR, Pelagia Deployment Controller will validate it and
request Rook to create CephFS. Then Pelagia Deployment Controller will create a Kubernetes `StorageClass`,
required to start provisioning the storage, which will operate the CephFS CSI driver to create Kubernetes PVs.
