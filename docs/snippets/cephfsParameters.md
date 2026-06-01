- `name` - Mandatory. CephFS instance name.
- `spec` - Represents the Rook `CephFilesystem` specification. For details, refer to the following Rook documentation: [CephFileystem CRD](https://rook.io/docs/rook/v1.19/CRDs/Shared-Filesystem/ceph-filesystem-crd/) and [CephFilesystem API specification](https://rook.io/docs/rook/v1.19/CRDs/specification/#ceph.rook.io/v1.FilesystemSpec).

     !!! note

         The CephFS metadata pool must be only of the `replicated` type.

     !!! note

         The first pool in the list is treated as the default data pool for CephFS and must always be `replicated`.

 The number of data pools is unlimited, but the default pool must always be present. For example:

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

 where `replicated.size` is the number of full copies of data on multiple nodes.

   {% include "./replicatedSize.md" %}

     !!! warning

         Modifying pools on a deployed CephFS has no effect.
         You can manually adjust pool settings through the Ceph CLI.
         However, for any changes in pools, we recommend re-creating CephFS.
