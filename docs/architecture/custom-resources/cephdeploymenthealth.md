# CephDeploymentHealth Custom Resource

Verifying Ceph cluster state is **an entry point for issues investigation**.
`CephDeploymentHealth` (`cephdeploymenthealths.lcm.mirantis.com`) custom resource (CR)
allows you to verify the current health of a Ceph cluster and identify potentially
problematic components. To obtain the detailed status for a particular Rook resources and Ceph cluster,
which Pelagia manages:

```bash
kubectl -n pelagia get cephdeploymenthealth -o yaml
```

Example output:

<details>
<summary>Example CephDeploymentHealth status</summary>
<div>
```yaml
apiVersion: v1
items:
- apiVersion: lcm.mirantis.com/v1alpha1
  kind: CephDeploymentHealth
  metadata:
    name: pelagia-ceph
    namespace: pelagia
  status:
    healthReport:
      cephDaemons:
        cephCSIPluginDaemons:
          csi-cephfsplugin:
            info:
            - 3/3 ready
            status: ok
          csi-rbdplugin:
            info:
            - 3/3 ready
            status: ok
        cephDaemons:
          mds:
            info:
            - 'mds active: 1/1 (cephfs ''cephfs-store'')'
            status: ok
          mgr:
            info:
            - 'a is active mgr, standbys: [b]'
            status: ok
          mon:
            info:
            - 3 mons, quorum [a b c]
            status: ok
          osd:
            info:
            - 3 osds, 3 up, 3 in
            status: ok
          rgw:
            info:
            - '2 rgws running, daemons: [21273 38213]'
            status: ok
      clusterDetails:
        cephEvents:
          PgAutoscalerDetails:
            state: Idle
          rebalanceDetails:
            state: Idle
        rgwInfo:
          publicEndpoint: https://192.10.1.101:443
        usageDetails:
          deviceClasses:
            hdd:
              availableBytes: "159676964864"
              totalBytes: "161048690688"
              usedBytes: "1371725824"
          pools:
            .mgr:
              availableBytes: "75660169216"
              totalBytes: "75661557760"
              usedBytes: "1388544"
              usedBytesPercentage: "0.001"
            .rgw.root:
              availableBytes: "75661426688"
              totalBytes: "75661557760"
              usedBytes: "131072"
              usedBytesPercentage: "0.000"
            cephfs-store-cephfs-pool-1:
              availableBytes: "75661557760"
              totalBytes: "75661557760"
              usedBytes: "0"
              usedBytesPercentage: "0.000"
            cephfs-store-metadata:
              availableBytes: "75660517376"
              totalBytes: "75661557760"
              usedBytes: "1040384"
              usedBytesPercentage: "0.001"
            kubernetes-hdd:
              availableBytes: "75661549568"
              totalBytes: "75661557760"
              usedBytes: "8192"
              usedBytesPercentage: "0.000"
            rgw-store.rgw.buckets.data:
              availableBytes: "75661557760"
              totalBytes: "75661557760"
              usedBytes: "0"
              usedBytesPercentage: "0.000"
            rgw-store.rgw.buckets.index:
              availableBytes: "75661557760"
              totalBytes: "75661557760"
              usedBytes: "0"
              usedBytesPercentage: "0.000"
            rgw-store.rgw.buckets.non-ec:
              availableBytes: "75661557760"
              totalBytes: "75661557760"
              usedBytes: "0"
              usedBytesPercentage: "0.000"
            rgw-store.rgw.control:
              availableBytes: "75661557760"
              totalBytes: "75661557760"
              usedBytes: "0"
              usedBytesPercentage: "0.000"
            rgw-store.rgw.log:
              availableBytes: "75660230656"
              totalBytes: "75661557760"
              usedBytes: "1327104"
              usedBytesPercentage: "0.001"
            rgw-store.rgw.meta:
              availableBytes: "75661557760"
              totalBytes: "75661557760"
              usedBytes: "0"
              usedBytesPercentage: "0.000"
            rgw-store.rgw.otp:
              availableBytes: "75661557760"
              totalBytes: "75661557760"
              usedBytes: "0"
              usedBytesPercentage: "0.000"
      osdAnalysis:
        cephClusterSpecGeneration: 1
        diskDaemon:
          info:
          - 3/3 ready
          status: ok
        specAnalysis:
          cluster-storage-worker-0:
            status: ok
          cluster-storage-worker-1:
            status: ok
          cluster-storage-worker-2:
            status: ok
      rookCephObjects:
        blockStorage:
          cephBlockPools:
            builtin-mgr:
              info:
                failureDomain: host
                type: Replicated
              observedGeneration: 1
              phase: Ready
              poolID: 11
            builtin-rgw-root:
              info:
                failureDomain: host
                type: Replicated
              observedGeneration: 1
              phase: Ready
              poolID: 1
            kubernetes-hdd:
              info:
                failureDomain: host
                type: Replicated
              observedGeneration: 1
              phase: Ready
              poolID: 10
        cephCluster:
          ceph:
            capacity:
              bytesAvailable: 159676964864
              bytesTotal: 161048690688
              bytesUsed: 1371725824
              lastUpdated: "2025-08-15T12:10:39Z"
            fsid: 92d56f80-b7a8-4a35-80ef-eb6a877c2a73
            health: HEALTH_OK
            lastChanged: "2025-08-14T14:07:43Z"
            lastChecked: "2025-08-15T12:10:39Z"
            previousHealth: HEALTH_WARN
            versions:
              mds:
                ceph version 19.2.3 (c92aebb279828e9c3c1f5d24613efca272649e62) squid (stable): 2
              mgr:
                ceph version 19.2.3 (c92aebb279828e9c3c1f5d24613efca272649e62) squid (stable): 2
              mon:
                ceph version 19.2.3 (c92aebb279828e9c3c1f5d24613efca272649e62) squid (stable): 3
              osd:
                ceph version 19.2.3 (c92aebb279828e9c3c1f5d24613efca272649e62) squid (stable): 3
              overall:
                ceph version 19.2.3 (c92aebb279828e9c3c1f5d24613efca272649e62) squid (stable): 12
              rgw:
                ceph version 19.2.3 (c92aebb279828e9c3c1f5d24613efca272649e62) squid (stable): 2
          conditions:
          - lastHeartbeatTime: "2025-08-15T12:10:40Z"
            lastTransitionTime: "2025-08-12T09:35:27Z"
            message: Cluster created successfully
            reason: ClusterCreated
            status: "True"
            type: Ready
          message: Cluster created successfully
          observedGeneration: 1
          phase: Ready
          state: Created
          storage:
            deviceClasses:
            - name: hdd
            osd:
              migrationStatus: {}
              storeType:
                bluestore: 3
          version:
            image: 127.0.0.1/ceph/ceph:v19.2.3
            version: 19.2.3-0
        objectStorage:
          cephObjectStore:
            rgw-store:
              endpoints:
                insecure:
                - http://rook-ceph-rgw-rgw-store.rook-ceph.svc:8081
                secure:
                - https://rook-ceph-rgw-rgw-store.rook-ceph.svc:8443
              info:
                endpoint: http://rook-ceph-rgw-rgw-store.rook-ceph.svc:8081
                secureEndpoint: https://rook-ceph-rgw-rgw-store.rook-ceph.svc:8443
              observedGeneration: 1
              phase: Ready
        sharedFilesystem:
          cephFilesystems:
            cephfs-store:
              observedGeneration: 1
              phase: Ready
      rookOperator:
        status: ok
    lastHealthCheck: "2025-08-15T12:11:00Z"
    lastHealthUpdate: "2025-08-15T12:11:00Z"
    state: Ok
kind: List
metadata:
  resourceVersion: ""
```
</div>
</details>

To understand the status of a `CephDeploymentHealth`, learn the following:

- [High-level status fields](#general)
- [Health report status fields](#full)

## High-level status fields <a name="general"></a>

- `healthReport` - Complete information about Ceph cluster including cluster, Ceph resources, and daemon health. It helps reveal potentially problematic components.
- `lastHealthCheck` - `DateTime` when previous cluster state check occurred.
- `lastHealthUpdate` - `DateTime` when previous cluster state update occurred.
- `issues` - List of strings of all issues found during cluster state check.
- `state` - Cluster state that can be `Ok` or `Failed` depending on the Ceph cluster state check.


## Health report status fields <a name="full"></a>

- `rookOperator` - State of the Rook Ceph Operator pod which contains the following fields:

    - `status` - contains short state of current Rook Ceph Operator pod;
    - `issues` - represents found Rook Operator issues, otherwise it is empty;

- `rookCephObjects` - General information from Rook about the Ceph cluster health and current
  state. Contains the following fields:

    - `cephCluster` - Contains Ceph cluster status information.
    - `blockStorage` - Contains status of block-storage related objects status information.
    - `cephClients` - Represents a key-value mapping of Ceph client's name and its status.
    - `objectStorage` - Contains status of object-storage related objects status information.
    - `sharedFilesystems` - Contains status of shared filesystems related objects status information.

    <details>
    <summary>Example *rookCephObjects* status</summary>
    <div>
    ```yaml
    status:
      healthReport:
        rookCephObjects:
          cephCluster:
            state: <rook ceph cluster common status>
            phase: <rook ceph cluster spec reconcile phase>
            message: <rook ceph cluster phase details>
            conditions: <history of rook ceph cluster reconcile steps>
            ceph: <ceph cluster health>
            storage:
              deviceClasses: <list of used device classes in ceph cluster>
            version:
              image: <ceph image used in ceph cluster>
              version: <ceph version of ceph cluster>
        blockStorage:
          cephBlockPools:
            <cephBlockPoolName>:
              ...
              phase: <rook ceph block pool resource phase>
        cephClients:
          <cephClientName>:
            ...
            phase: <rook ceph client resource phase>
        objectStorage:
          cephObjectStore:
            <cephObjectStoreName>:
              ...
              phase: <rook ceph object store resource phase>
          cephObjectStoreUsers:
            <rgwUserName>:
              ...
              phase: <rook ceph object store user resource phase>
          objectBucketClaims:
            <bucketName>:
              ...
              phase: <rook ceph object bucket claims resource phase>
          cephObjectRealms:
            <realmName>:
              ...
              phase: <rook ceph object store realm resource phase>
          cephObjectZoneGroups:
            <zonegroupName>:
              ...
              phase: <rook ceph object store zonegroup resource phase>
          cephObjectZones:
            <zoneName>:
              ...
              phase: <rook ceph object store zone resource phase>
        sharedFilesystems:
          cephFilesystems:
            <cephFSName>:
              ...
              phase: <rook ceph filesystem resource phase>
    ```
    </div>
    </details>

- `cephDaemons` - Contains information about the state of the Ceph and Ceph CSI daemons in the cluster.
  Includes the following fields:

    - `cephDaemons` - Map of statuses for each Ceph cluster daemon type. Indicates the
      expected and actual number of Ceph daemons on the cluster. Available
      daemon types are: ``mgr``, ``mon``, ``osd``, and ``rgw``.
    - `cephCSIPluginDaemons` - Contains information, similar to the ``daemonsStatus`` format, for each
      Ceph CSI plugin deployed in the Ceph cluster: ``rbd`` and ``cephfs``.

    <details>
    <summary>Example *cephDaemons* status</summary>
    <div>
    ```yaml
    status:
      healthReport:
        cephDaemons:
          cephCSIPluginDaemons:
            csi-cephfsplugin:
              info:
              - 3/3 ready
              status: ok
            csi-rbdplugin:
              info:
              - 3/3 ready
              status: ok
          cephDaemons:
            mds:
              info:
              - 'mds active: 1/1 (cephfs ''cephfs-store'')'
              status: ok
            mgr:
              info:
              - 'a is active mgr, standbys: [b]'
              status: ok
            mon:
              info:
              - 3 mons, quorum [a b c]
              status: ok
            osd:
              info:
              - 3 osds, 3 up, 3 in
              status: ok
            rgw:
              info:
              - '2 rgws running, daemons: [21273 38213]'
              status: ok
    ```
    </div>
    </details>

- `clusterDetails` - Verbose details of the Ceph cluster state. Contains the following fields:

    - `usageDetails` - Describes the used, available, and total storage size for each
      `deviceClass` and `pool`.
    - `cephEvents` - Contains info about current ceph events happen in Ceph cluster
      if progress events module is enabled.
    - `rgwInfo` - represents additional Ceph Object Storage Multisite information like public endpoint
      to connect external zone and sync statuses.

    <details>
    <summary>Example *clusterDetails* status</summary>
    <div>
    ```yaml
    status:
      healthReport:
        clusterDetails:
          cephEvents:
            PgAutoscalerDetails:
              state: Idle
            rebalanceDetails:
              state: Idle
          rgwInfo:
            publicEndpoint: https://192.10.1.101:443
          usageDetails:
            deviceClasses:
              hdd:
                availableBytes: "159681224704"
                totalBytes: "161048690688"
                usedBytes: "1367465984"
            pools:
              .mgr:
                availableBytes: "75660169216"
                totalBytes: "75661557760"
                usedBytes: "1388544"
                usedBytesPercentage: "0.001"
              .rgw.root:
                availableBytes: "75661426688"
                totalBytes: "75661557760"
                usedBytes: "131072"
                usedBytesPercentage: "0.000"
              cephfs-store-cephfs-pool-1:
                availableBytes: "75661557760"
                totalBytes: "75661557760"
                usedBytes: "0"
                usedBytesPercentage: "0.000"
              cephfs-store-metadata:
                availableBytes: "75660517376"
                totalBytes: "75661557760"
                usedBytes: "1040384"
                usedBytesPercentage: "0.001"
              kubernetes-hdd:
                availableBytes: "75661549568"
                totalBytes: "75661557760"
                usedBytes: "8192"
                usedBytesPercentage: "0.000"
              rgw-store.rgw.buckets.data:
                availableBytes: "75661557760"
                totalBytes: "75661557760"
                usedBytes: "0"
                usedBytesPercentage: "0.000"
              ...
              rgw-store.rgw.otp:
                availableBytes: "75661557760"
                totalBytes: "75661557760"
                usedBytes: "0"
                usedBytesPercentage: "0.000"
    ```
    </div>
    </details>

- `osdAnalysis` - Ceph OSD analysis results based on Rook `CephCluster` specification and `disk-daemon` reports.
  Contains the following fields:

    - `diskDaemon` - Disk daemon status. Disk daemon is Pelagia LCM component that provides information about
      nodes' devices and their usage by Ceph OSDs.
    - `cephClusterSpecGeneration` - Last validated Rook `CephCluster` specification generation.
    - `specAnalysis` - Map of per-node analysis results based on the Rook `CephCluster` specification.

    <details>
    <summary>Example *osdAnalysis* status</summary>
    <div>
    ```yaml
    status:
      healthReport:
        osdAnalysis:
          cephClusterSpecGeneration: 1
          diskDaemon:
            info:
            - 3/3 ready
            status: ok
          specAnalysis:
            cluster-storage-worker-0:
              status: ok
            cluster-storage-worker-1:
              status: ok
            cluster-storage-worker-2:
              status: ok
    ```
    </div>
    </details>
