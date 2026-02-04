<a id="enable-cephfs-mira"></a>

# Configure Ceph Shared File System (CephFS)

The Ceph Shared File System, or CephFS, provides the ability to create
read/write shared file system Persistent Volumes (PVs). These PVs support the
`ReadWriteMany` access mode for the `FileSystem` volume mode.
CephFS deploys its own daemons called MetaData Servers or Ceph MDS. For
details, see [Ceph Documentation: Ceph File System](https://docs.ceph.com/en/latest/cephfs/index.html).

!!! note

    By design, CephFS data pool and metadata pool must be `replicated` only.

## CephFS specification parameters <a name="cephfs-parameters"></a>

The `CephDeployment` custom resource (CR) `spec` includes the `sharedFilesystem.cephFS` section
with the following CephFS parameters:

- `name` - CephFS instance name.
- `dataPools` - A list of CephFS data pool specifications. Each spec contains the `name`, `replicated` or
  `erasureCoded`, `deviceClass`, and `failureDomain` parameters. The first pool in the list is treated
  as the default data pool for CephFS and must always be `replicated`. The `failureDomain` parameter
  may be set to `host`, `rack` and so on, defining the failure domain across which the data will
  be spread. The number of data pools is unlimited, but the default pool must always be present. For example:

    ```yaml
    spec:
      sharedFilesystem:
        cephFS:
        - name: cephfs-store
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
    ```

    where `replicated.size` is the number of full copies of data on multiple nodes.

    !!! warning

        When using the non-recommended Ceph pools `replicated.size` of less than `3`, Ceph OSD removal cannot be
        performed. The minimal replica size equals a rounded up half of the specified `replicated.size`.

        For example, if `replicated.size` is `2`, the minimal replica size is `1`, and if `replicated.size` is `3`,
        then the minimal replica size is `2`. The replica size of `1` allows Ceph having PGs with only one
        Ceph OSD in the `acting` state, which may cause a `PG_TOO_DEGRADED` health warning that blocks
        Ceph OSD removal. We recommend setting `replicated.size` to `3` for each Ceph pool.

    !!! warning

        Modifying of `dataPools` on a deployed CephFS has no effect. You can manually adjust pool settings
        through the Ceph CLI. However, for any changes in `dataPools`, we recommend re-creating CephFS.

- `metadataPool` - CephFS metadata pool spec that should only contain `replicated`, `deviceClass`, and `failureDomain`
  parameters. The `failureDomain` parameter may be set to `host`, `rack` and so on,
  defining the failure domain across which the data will be spread. Can use only `replicated` settings. For example:

    ```yaml
    spec:
      sharedFilesystem:
        cephFS:
        - name: cephfs-store
          metadataPool:
            deviceClass: nvme
            replicated:
              size: 3
            failureDomain: host
    ```

    where `replicated.size` is the number of full copies of data on multiple nodes.

    !!! warning

        Modifying of `metadataPool` on a deployed CephFS has no effect. You can manually adjust pool settings
        through the Ceph CLI. However, for any changes in `metadataPool`, we recommend re-creating CephFS.

- `preserveFilesystemOnDelete` - Defines whether to delete the data and metadata pools if CephFS is
  deleted. Set to `true` to avoid occasional data loss in case of human error. However, for security reasons,
  we recommend setting `preserveFilesystemOnDelete` to `false`.

- `metadataServer` - Metadata Server settings correspond to the Ceph MDS daemon settings. Contains the following fields:

    - `activeCount` - the number of active Ceph MDS instances. As a load
      increases, CephFS will automatically partition the file system
      across the Ceph MDS instances. Rook will create double the number
      of Ceph MDS instances as requested by `activeCount`. The extra
      instances will be in the standby mode for failover. We recommend specifying this parameter to `1` and
      increasing the MDS daemons count only in case of a high load.
    - `activeStandby` - defines whether the extra Ceph MDS instances
      will be in active standby mode and will keep a warm cache of the
      file system metadata for faster failover. CephFS will assign the instances in failover pairs.
      If `false`, the extra Ceph MDS instances will all be in passive standby mode and will
      not maintain a warm cache of the metadata. The default value is `false`.
    - `resources` - represents Kubernetes resource requirements for Ceph MDS pods.

    For example:

    ```yaml
    spec:
      sharedFilesystem:
        cephFS:
        - name: cephfs-store
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

## Configure CephFS

1. Optional. Override the CSI CephFS gRPC and liveness metrics port. For example, if an application is already using
   the default CephFS ports `9092` and `9082`, which may cause conflicts on the node. Upgrade Pelagia Helm release
   values with desired port numbers:

     ```bash
     helm upgrade --install pelagia-ceph oci://registry.mirantis.com/pelagia/pelagia-ceph --version 1.0.0 -n pelagia \
          --set rookConfig.csiCephFsGPCMetricsPort=<desiredPort>,rookConfig.csiCephFsLivenessMetricsPort=<desiredPort>
     ```

     Rook will enable the CephFS CSI plugin and provisioner.

2. Open the `CephDeployment` CR for editing:
   ```bash
   kubectl -n pelagia edit cephdpl
   ```

3. In the `sharedFilesystem` section, specify parameters according to [CephFS specification](#cephfs-parameters). For example:
   ```yaml
   spec:
     sharedFilesystem:
       cephFS:
       - name: cephfs-store
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
       worker-1:
         roles:
         ...
         - mds
   ```

Once CephFS is specified in the `CephDeployment` CR, Pelagia Deployment Controller will validate it and
request Rook to create CephFS. Then Pelagia Deployment Controller will create a Kubernetes `StorageClass`,
required to start provisioning the storage, which will operate the CephFS CSI driver to create Kubernetes PVs.
