<a id="cephdeployment-cephdeployment-custom-resource"></a>
# CephDeployment custom resource

This section describes how to configure a Ceph cluster using the `CephDeployment`
(`cephdeployments.lcm.mirantis.com`) custom resource (CR).

The `CephDeployment` CR spec specifies the nodes to deploy as Ceph components.
Based on the roles definitions in the `CephDeployment` CR, Pelagia Deployment Controller
automatically labels nodes for Ceph Monitors and Managers. Ceph OSDs are
deployed based on the `devices` parameter defined for each Ceph node.

For the default `CephDeployment` CR, see the following example:

??? "Example configuration of Ceph specification"

    ```yaml
    apiVersion: lcm.mirantis.com/v1alpha1
    kind: CephDeployment
    metadata:
      name: pelagia-ceph
      namespace: pelagia
    spec:
      cluster:
        network:
          addressRanges:
            cluster:
            - 10.12.0.0/24
            public:
            - 10.12.1.0/24
      nodes:
      - name: cluster-storage-controlplane-0
        roles:
        - mgr
        - mon
        - mds
      - name: cluster-storage-controlplane-1
        roles:
        - mgr
        - mon
        - mds
      - name: cluster-storage-controlplane-2
        roles:
        - mgr
        - mon
        - mds
      - name: cluster-storage-worker-0
        roles: []
        devices:
        - config:
            deviceClass: ssd
          fullPath: /dev/disk/by-id/scsi-1ATA_WDC_WDS100T2B0A-00SM50_200231434939
      - name: cluster-storage-worker-1
        roles: []
        devices:
        - config:
            deviceClass: ssd
          fullPath: /dev/disk/by-id/scsi-1ATA_WDC_WDS100T2B0A-00SM50_200231440912
      - name: cluster-storage-worker-2
        roles: []
        devices:
        - config:
            deviceClass: ssd
          fullPath: /dev/disk/by-id/scsi-1ATA_WDC_WDS100T2B0A-00SM50_200231443409
      blockStorage:
        pools:
        - name: kubernetes
          storageClassOpts:
            default: true
          spec:
            deviceClass: ssd
            replicated:
              size: 3
      objectStorage:
        objectStores:
          - name: rgw-store
            spec:
              dataPool:
                deviceClass: ssd
                replicated:
                  size: 3
              metadataPool:
                deviceClass: ssd
                replicated:
                  size: 3
              gateway:
                instances: 3
                port: 8081
                securePort: 8443
              preservePoolsOnDelete: false
      sharedFilesystem:
        cephFilesystems:
        - name: cephfs-store
          spec:
            dataPools:
            - name: cephfs-pool-1
              deviceClass: ssd
              replicated:
                size: 3
            metadataPool:
              deviceClass: ssd
              replicated:
                size: 3
            metadataServer:
              activeCount: 1
    ```

## Configure a Ceph cluster with CephDeployment

1. Select from the following options:

     - If you do not have a Ceph cluster yet, create `cephdeployment.yaml` for editing.
     - If the Ceph cluster is already deployed, open the `CephDeployment` CR for editing:

       ```
       kubectl -n pelagia edit cephdpl
       ```

2. Set up the Ceph cluster using the configuration reference below.

3. Select from the following options:

    - If you are creating Ceph cluster, save the updated
      `CephDeployment` template to the corresponding file and apply the file to a cluster:
      ```
      kubectl apply -f cephdeployment.yaml
      ```

    - If you are editing `CephDeployment` , save the changes and exit the text editor to apply it.

4. Verify the `CephDeployment` reconcile status. For description of the ``status`` fields, refer to the *Status fields* subsection below.

## CephDeployment configuration options

The following subsections contain a description of `CephDeployment` parameters for an
advanced configuration.

!!! warning

    `CephDeployment` has changed since version 1.x. Deprecated parameters are automatically migrated
    to the new API fields during the Pelagia upgrade. No manual steps are required.

<a name="cephdeployment-general-parameters"></a>
### General parameters

!!! warning

    To avoid ambiguous behavior of Ceph daemons, do not specify
    ``0.0.0.0/0`` as the Ceph network. Otherwise, Ceph daemons can select
    an incorrect interface that can cause the Ceph cluster to
    become unavailable.

!!! note

    A Ceph cluster supports multiple IP networks.
    For details, see [Enable multinetworking](../ops-guide/deployment/multinetworking.md#multinetworking-enable-ceph-multinetwork).

!!! note

    The Ceph cluster placement affinity and anti-affinity will be ignored in favor of the provided `nodes` roles.

- `cluster` - Specifies the complete Ceph cluster configuration and represents the Rook `CephCluster` object specification. For available configuration options, see Rook documentation: [CephCluster CRD](https://rook.io/docs/rook/v1.19/CRDs/Cluster/ceph-cluster-crd/) and [CephCluster API](https://rook.io/docs/rook/v1.19/CRDs/specification/#ceph.rook.io/v1.ClusterSpec).

    Example configuration:
    ```yaml
    spec:
      cluster:
        network:
          addressRanges:
            cluster:
            - 10.10.0.0/24
            public:
            - 192.100.0.0/24
        placement:
          mgr:
            tolerations:
            - effect: NoSchedule
              key: node-role.kubernetes.io/control-plane
              operator: Exists
            - key: node-role.kubernetes.io/master
              effect: NoSchedule
              operator: Exists
          mon:
            tolerations:
            - effect: NoSchedule
              key: node-role.kubernetes.io/control-plane
              operator: Exists
            - key: node-role.kubernetes.io/master
              effect: NoSchedule
              operator: Exists
    ```

- `blockStorages` - Specifies the Ceph block storage configuration. Contains the `pools` parameter that specifies the list of Ceph pools. For details, see the **Pools parameters** section below.
- `clients` - Specifies the list of Ceph clients. For details, see the **Clients parameters** section below.
- `extraOpts` - Enables specification of extra options for a Ceph cluster setup, includes the `deviceLabels` parameter. For details, see the **ExtraOpts parameters** section below.
- `nodes` - Specifies the list of Ceph nodes with node specifications. Each list item can define a Ceph node specification for a single node or a group of nodes specified by an explicit list, a label, or a combination of both. For details, see the **Nodes parameters** section below. 
- `objectStorage` - Specifies the parameters for Object Storage, such as RADOS Gateway, the Ceph Object Storage, the RADOS Gateway Multisite configuration, and the Gateway API HTTPRoutes for public access to Object Storage. For details, see the **Object storage parameters** section below.
- `rbdMirror` - Specifies the parameters for RBD mirroring. For details, see the **RBD Mirroring parameters** section below.
- `rookConfig` - Specifies the string key-value that allows overriding Ceph configuration options. For details, see the **RookConfig parameters** section below.
- `sharedFilesystem` - Enables Ceph Filesystem. For details, the **CephFS parameters** section below.
- `network` - Specifies access and public networks for the Ceph cluster. Deprecated, automatically migrated to the `cluster.network` field.
- `pools` - Specifies the list of Ceph pools. Deprecated, automatically migrated to the `blockStorage.pools` field.
- `ingressConfig` - Enables a custom ingress rule for public access on Ceph services, for example, Ceph RADOS Gateway. For details, see [Configure Ceph Object Gateway TLS](../ops-guide/deployment/object-storage/rgw-tls.md#rgw-tls-configure-ceph-object-gateway-tls). Deprecated in favor of the Gateway API due to [Ingress deprecation](https://kubernetes.io/blog/2025/11/11/ingress-nginx-retirement/).
- `healthCheck` - Configures health checks and liveness probe settings for Ceph daemons. Deprecated, automatically migrated to the `cluster.healthCheck` field.
- `mgr` - Specifies the list of Ceph Manager modules to be enabled or disabled. Deprecated, automatically migrated to the `cluster.mgr` field.
- `dashboard` - Enables Ceph dashboard. Deprecated, automatically migrated to the `cluster.dashboard` field. Currently, Pelagia does not support Ceph Dashboard.
- `external` - Enables external Ceph cluster mode. If enabled, Pelagia reads a dedicated `Secret` containing the connection credentials for the external Ceph cluster. Deprecated, automatically migrated to the `cluster.external` field.

<a name="cephdeployment-nodes-parameters"></a>
### Nodes parameters

- `name` - Mandatory. Specifies the following:

    - If node spec implies to be deployed on a single node, `name`
      stands for node name where Ceph node should be deployed. For example,
      it could be ``cluster-storage-worker-0``.
    - If node spec implies to be deployed on a group of nodes, `name`
      stands for group name, for example `group-rack-1`. In that case,
      Ceph node specification must contain either `nodeGroup` or
      `nodesByLabel` fields defined.

- ``nodeGroup`` - Optional. Specifies the list of nodes and used for specifying Ceph
  node specification for a group of nodes from the list. For example:

    ```yaml
    spec:
      nodes:
      - name: group-1
        nodeGroup: [node-X, node-Y]
    ```

- ``nodesByLabel`` - Optional. Specifies label expression and used for specifying Ceph
  node specification for a group of nodes found by label. For example:

    ```yaml
    spec:
      nodes:
      - name: group-1
        nodesByLabel: "ceph-storage-node=true,!ceph-control-node"
    ```

- ``roles`` - Optional. Specifies the ``mon``, ``mgr``, ``rgw`` or ``mds`` daemon
  to be installed on a Ceph node. You can place the daemons on any nodes
  upon your decision. Consider the following recommendations:

    - The recommended number of Ceph Monitors in a Ceph cluster is 3.
      Therefore, at least three Ceph nodes must contain the ``mon`` item in
      the ``roles`` parameter.
    - The number of Ceph Monitors must be odd.
    - Do not add more than three Ceph Monitors at a time and wait until the
      Ceph cluster is ``Ready`` before adding more daemons.
    - For better HA and fault tolerance, the number of ``mgr`` roles
      must equal the number of ``mon`` roles. Therefore, we recommend
      labeling at least three Ceph nodes with the ``mgr`` role.
    - If ``rgw`` roles are not specified, all ``rgw`` daemons will spawn
      on the same nodes with ``mon`` daemons.

    If a Ceph node contains a ``mon`` role, the Ceph Monitor Pod
    deploys on this node.

    If a Ceph node contains a ``mgr`` role, it informs the Ceph
    Controller that a Ceph Manager can be deployed on the node.
    Rook Operator selects the first available node to deploy the
    Ceph Manager on it. Pelagia supports deploying two Ceph Managers in total:
    one active and one stand-by.

    If you assign the ``mgr`` role to three recommended Ceph nodes,
    one back-up Ceph node is available to redeploy a failed Ceph Manager
    in a case of a node outage.

- ``monitorIP`` - Highly recommended for production and optional for staging
  deployments. If defined, specifies a custom IP address for Ceph Monitor which
  should be placed on the node. If not defined, Ceph Monitor on the node will
  use the default `hostNetwork` address of a node. We recommend using an IP
  address from the Ceph public network address range, which is defined in the
  `publicNet` parameter.

    !!! note

        To update ``monitorIP``, the corresponding Ceph Monitor daemon must be re-created.

- ``config`` - Mandatory. Specifies a map of device configurations that must contain a
  mandatory ``deviceClass`` parameter set to ``hdd``, ``ssd``, or ``nvme``.
  Applied for all OSDs on the current node. Can be overridden by the
  ``config`` parameter of each device defined in the ``devices`` parameter.

    For details, see [Rook documentation: OSD config settings](https://rook.io/docs/rook/latest/CRDs/Cluster/ceph-cluster-crd/#osd-configuration-settings).

- ``devices`` - Optional. Specifies the list of devices to use for Ceph OSD deployment.
  Includes the following parameters:

    !!! note

        Recommending to use the ``fullPath`` field for defining
        ``by-id`` symlinks as persistent device identifiers. For details, see
        [Addressing Ceph storage devices](../architecture/addressing-ceph-devices.md#addressing-ceph-devices-addressing-ceph-storage-devices).

    - ``fullPath`` - a storage device symlink. Accepts the following values:

        - The device ``by-id`` symlink that contains the serial number of the
          physical device or contains ``wwn``. For example,
          ``/dev/disk/by-id/nvme-SAMSUNG_MZ1LB3T8HMLA-00007_S46FNY0R394543``.

        - The device ``by-path`` symlink. For example,
          ``/dev/disk/by-path/pci-0000:00:11.4-ata-3``. We do not recommend
          specifying storage devices with device ``by-path`` symlinks
          because such identifiers are not persistent and can change at node boot.

        This parameter is mutually exclusive with ``name``.

    - ``name`` - a storage device name. Accepts the following values:

        - The device name, for example, ``sdc``. We do not recommend
          specifying storage devices with device names because such identifiers
          are not persistent and can change at node boot.
        - The device ``by-id`` symlink that contains the serial number of the
          physical device or contains ``wwn``. For example,
          ``/dev/disk/by-id/nvme-SAMSUNG_MZ1LB3T8HMLA-00007_S46FNY0R394543``.
        - The device label from ``extraOpts.deviceLabels`` section which is
          generally used for templating Ceph node specification for node groups.
          For details, see the **ExtraOpts parameters** section below.

        This parameter is mutually exclusive with ``fullPath``.

    - ``config`` - a map of device configurations that must contain a
      mandatory ``deviceClass`` parameter set to ``hdd``, ``ssd``, or
      ``nvme``. The device class must be defined in a pool and can
      optionally contain a metadata device, for example:

        ```yaml
        spec:
          nodes:
          - name: <node-a>
            devices:
            - fullPath: /dev/disk/by-id/scsi-SATA_HGST_HUS724040AL_PN1334PEHN18ZS
              config:
                deviceClass: hdd
                metadataDevice: /dev/meta-1/nvme-dev-1
                osdsPerDevice: "2"
        ```

        The underlying storage format to use for Ceph OSDs is BlueStore.

        The ``metadataDevice`` parameter accepts a device name or logical
        volume path for the BlueStore device. We recommend using
        logical volume paths created on ``nvme`` devices.

        The ``osdsPerDevice`` parameter accepts the string-type natural
        numbers and allows splitting one device on several Ceph OSD
        daemons. We recommend using this parameter only for ``ssd``
        or ``nvme`` disks.

- ``deviceFilter`` - Optional. Specifies regexp by names of devices to use for Ceph OSD
  deployment. Mutually exclusive with ``devices`` and ``devicePathFilter``.
  Requires the ``config`` parameter with ``deviceClass`` specified. For example:

    ```yaml
    spec:
      nodes:
      - name: <node-a>
        deviceFilter: "^sd[def]$"
        config:
          deviceClass: hdd
    ```

    For more details, see [Rook documentation: Storage selection settings](https://rook.io/docs/rook/latest/CRDs/Cluster/ceph-cluster-crd/#storage-selection-settings).

- ``devicePathFilter`` - Optional. Specifies regexp by paths of devices to use for Ceph OSD
  deployment. Mutually exclusive with ``devices`` and ``deviceFilter``.
  Requires the ``config`` parameter with ``deviceClass`` specified. For example:

    ```yaml
    spec:
      nodes:
      - name: <node-a>
        devicePathFilter: "^/dev/disk/by-id/scsi-SATA.+$"
        config:
          deviceClass: hdd
    ```

    For more details, see [Rook documentation: Storage selection settings](https://rook.io/docs/rook/latest/CRDs/Cluster/ceph-cluster-crd/#storage-selection-settings).

- ``crush`` - Optional. Specifies the explicit key-value CRUSH topology for a node.
  For details, see [Ceph documentation: CRUSH maps](https://docs.ceph.com/en/latest/rados/operations/crush-map/).
  Includes the following parameters:

    - ``datacenter`` - a physical data center that consists of rooms and
      handles data.
    - ``room`` - a room that accommodates one or more racks with hosts.
    - ``pdu`` - a power distribution unit (PDU) device that has multiple
      outputs and distributes electric power to racks located within a
      data center.
    - ``row`` - a row of computing racks inside ``room``.
    - ``rack`` - a computing rack that accommodates one or more hosts.
    - ``chassis`` - a bare metal structure that houses or physically
      assembles hosts.
    - ``region`` - the geographic location of one or more Ceph Object
      instances within one or more zones.
    - ``zone`` - a logical group that consists of one or more Ceph Object
      instances.

    Example configuration:

    ```yaml
    spec:
      nodes:
      - name: <node-a>
        crush:
          datacenter: dc1
          room: room1
          pdu: pdu1
          row: row1
          rack: rack1
          chassis: ch1
          region: region1
          zone: zone1
    ```

<a name="cephdeployment-pools-parameters"></a>
### Pools parameters

The `pools` parameters contain the Ceph block pool specification that represents the Rook `CephBlockPool` specification.

- `name` - Mandatory. Specifies the pool name as a prefix for each Ceph block pool.
  The resulting Ceph block pool name will be `<name>-<deviceClass>`.
- `useAsFullName` - Optional. Enables Ceph block pool to use only the `name` value as a name.
  The resulting Ceph block pool name will be `<name>` without the `deviceClass` suffix.
- `role` - Optional. Specifies the pool role for Rockoon integration.
- `preserveOnDelete` - Optional. Enables skipping Ceph pool delete on `pools` section item removal.
  If `pools` section item removed with this flag enabled, related `CephBlockPool` object would be
  kept untouched and will require manual deletion on demand. Defaulted to `false`.
- `storageClassOpts` - Optional. Allows to configure parameters for storage class, created for RBD pool.
  Includes the following parameters:

    - `default` - Optional. Defines whether the pool and dependent StorageClass must be set
      as default. Must be enabled only for one pool. Defaults to `false`.
    - `mapOptions` - Optional. Not updatable as it applies only once. Specifies custom
      `rbd device map` options to use with `StorageClass` of a corresponding pool. Allows customizing
      the Kubernetes CSI driver interaction with Ceph RBD for the defined `StorageClass`. For
      available options, see [Ceph documentation: Kernel RBD (KRBD) options](https://docs.ceph.com/en/latest/man/8/rbd/#kernel-rbd-krbd-options).
    - `unmapOptions` - Optional. Not updatable as it applies only once. Specifies custom
      `rbd device unmap` options to use with `StorageClass` of a corresponding pool. Allows customizing
      the Kubernetes CSI driver interaction with Ceph RBD for the defined `StorageClass`. For
      available options, see [Ceph documentation: Kernel RBD (KRBD) options](https://docs.ceph.com/en/latest/man/8/rbd/#kernel-rbd-krbd-options).
    - `imageFeatures` - Optional. Not updatable as it applies only once. Specifies is a comma-separated
      list of RBD image features, see [Ceph documentation: Manage Rados block device (RBD) images](https://docs.ceph.com/en/latest/man/8/rbd/#cmdoption-rbd-image-feature).
    - `reclaimPolicy` - Optional. Specifies reclaim policy for the underlying `StorageClass` of
      the pool. Accepts `Retain` and `Delete` values. Default is `Delete` if not set.
    - `allowVolumeExpansion` - Optional. Not updatable as it applies only once. Enables expansion of
      persistent volumes based on `StorageClass` of a corresponding pool. For details, see
      [Kubernetes documentation: Resizing persistent volumes using Kubernetes](https://kubernetes.io/blog/2018/07/12/resizing-persistent-volumes-using-kubernetes).

        !!! note

            A Kubernetes cluster only supports increase of storage size.

- `spec` - Represents the Rook `CephBlockPool` specification. For details see, Rook specifications: [CephBlockPool CRD](https://rook.io/docs/rook/v1.19/CRDs/Block-Storage/ceph-block-pool-crd/#spec) and [CephBlockPool API](https://rook.io/docs/rook/v1.19/CRDs/specification/#ceph.rook.io/v1.PoolSpec).

    !!! note

        If you are using the `replicated` type of pool and need to use the `targetSizeRation` parameter,
        we recommend defining the target ratio using the `parameters.target_size_ratio` **string** field instead.
        For details, see [Ceph documentation: Set Pool values](https://docs.ceph.com/en/latest/rados/operations/pools/#set-pool-values).

    !!! note

        Target ratios for the pools required for Rockoon are described in
        [Integrate Pelagia with Rockoon](../ops-guide/rockoon/rockoon-integration.md#rockoon-integration-integrate-pelagia-with-rockoon).

    !!! danger

        We do not recommend using the following intermediate topology keys as a failure domain: `pdu`, `row`, and `chassis`. Consider
        the `rack` topology instead. The `osd` failure domain is allowed only for single-node deployments.

??? "Example configuration of *pools* specification"

    ```yaml
    spec:
      blockStorage:
        pools:
        - name: kubernetes
          storageClassOpts:
            default: true
          spec:
            deviceClass: hdd
            replicated:
              size: 3
            parameters:
              target_size_ratio: "10.0"
          preserveOnDelete: true
        - name: kubernetes
          spec:
            deviceClass: nvme
            erasureCoded:
              codingChunks: 1
              dataChunks: 2
            failureDomain: host
        - name: archive
          useAsFullName: true
          spec:
            deviceClass: hdd
            failureDomain: rack
            replicated:
              size: 3
    ```

As a result of `pools` configuration, the following Ceph pools will be created: `kubernetes-hdd`, `kubernetes-nvme`, and `archive`.

To configure additional required pools for Rockoon, see
[Integrate Pelagia with Rockoon](../ops-guide/rockoon/rockoon-integration.md#rockoon-integration-integrate-pelagia-with-rockoon).

!!! danger

    Since Ceph Pacific, Ceph CSI driver does not propagate the `777`
    permission on the mount point of persistent volumes based on any
    `StorageClass` of the Ceph pool.

<a name="cephdeployment-clients-parameters"></a>
### Clients parameters

The `clients` parameters contain the Ceph client specification that represents the Rook `CephClient` specification.

- `name` - Mandatory. Ceph client name.
- `spec` - Represents the Rook `CephClient` specification. For details see the following Rook documentation: [CephClient CRD](https://rook.io/docs/rook/v1.19/CRDs/ceph-client-crd/) and [CephClient API specification](https://rook.io/docs/rook/v1.19/CRDs/specification/#ceph.rook.io/v1.ClientSpec).

For details about Ceph client capabilities (`caps`), refer to [Ceph documentation: Authorization (capabilities)](https://docs.ceph.com/en/latest/rados/operations/user-management/#authorization-capabilities).

??? "Example configuration of *clients* specification"

    ```yaml
    spec:
      clients:
      - name: test-client
        caps:
          mon: allow r, allow command "osd blacklist"
          osd: profile rbd pool=kubernetes-nvme
    ```

<a name="cephdeployment-object-storage-parameters"></a>
### ObjectStorage parameters

- `objectStores` - List of Ceph `ObjectStorage` (RGW) objects. Each `objectStore` item represents the Rook `CephObjectStore` specification. For details, see the **RADOS Gateway parameters** section below.
- `users` - List of Ceph `ObjectStorage` users. Each `user` item represents the Rook `CephObjectStoreUser` specification. For details, see the **RGW users parameters** section below.
- `realms` - List of Ceph `ObjectStorage` realms. Each `realm` item represents the Rook `CephObjectRealms` specification. Currently, only one realm is supported, so the list must contain exactly one item. For details, see the **RADOS Gateway Multisite parameters** section below.
- `zonegroups` - List of Ceph `ObjectStorage` zone groups. Each `zonegroup` item represents the Rook `CephObjectZoneGroup` specification. Currently, only one zone group is supported, so the list must contain exactly one item. For details, see the **RADOS Gateway Multisite parameters** section below.
- `zones` - List of Ceph `ObjectStorage` zones. Each `zone` item represents the Rook `CephObjectZone` specification. Currently, only one zone is supported, so the list must contain exactly one item. For details, see the **RADOS Gateway Multisite parameters** section below.
- `gatewayHTTPRoute` - List of Gateway API HTTP routes. Each item represents the `HTTPRoute` specification. For details, see the **Gateway HTTPRoute parameters** section below.
- `rgw` - Single definition of the Ceph `ObjectStorage` (RGW) object. Deprecated, automatically migrated to `objectStores` and `users` fields.
- `multiSite` - Definition of Ceph `ObjectStorage` (RGW) Multisite configuration. Deprecated, automatically migrated to the following fields:

    * `multisite.realms` -> `objectStorage.realms`
    * `multisite.zoneGroups` -> `objectStorage.zonegroups`
    * `multisite.zones` -> `objectStorage.zones`

    !!! caution

        The realm access keys defined in the deprecated `multiSite.realms` field are removed from the spec during migration.

        For the `objectStorage.realms` field on new deployments, the operator must manually create a new secret with realm keys.
        For the procedure, see [Rook documentation: Getting Realm Access Key and Secret Key](https://rook.io/docs/rook/v1.19/Storage-Configuration/Object-Storage-RGW/ceph-object-multisite/#getting-realm-access-key-and-secret-key).

<a name="cephdeployment-rados-gateway-parameters"></a>
#### RADOS Gateway parameters

{% include "../snippets/rgwParameters.md" %}

??? "Example configuration of RADOS gateway specification"

    ```yaml
    spec:
      objectStorage:
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

<a name="cephdeployment-rgw-users-parameters"></a>
#### RGW users parameters

The RGW `users` parameters represent the Rook `CephObjectStoreUser` specification.
For details, see [Rook documenation: CephObjectStoreUser CRD](https://rook.io/docs/rook/v1.19/CRDs/Object-Storage/ceph-object-store-user-crd/#spec).

??? "Example configuration of RGW *users* specification"

    ```yaml
    spec:
      objectStorage:
        objectStores:
        - name: rgw-store
          ...
        users:
        - name: user-a
          spec:
            store: rgw-store
            capabilities:
              bucket: '*'
              metadata: read
              user: read
            displayName: user-a
            quotas:
              maxBuckets: 10
              maxSize: 10G
    ```

<a name="cephdeployment-rados-gateway-multisite-parameters"></a>
#### RADOS Gateway Multisite parameters

{% include "../snippets/multisiteParameters.md" %}

For configuration example, see [Enable Multisite for Ceph Object Storage](../ops-guide/deployment/object-storage/rgw-multisite.md#rgw-multisite-enable-multisite-for-ceph-object-storage).

<a name="cephdeployment-httproute-parameters"></a>
#### Gateway HTTPRoute parameters

The `gatewayHTTPRoutes` parameters represent the Gateway API `HTTPRoute` specification.

- `name` - Name of `HTTPRoute`. Mandatory.
- `objectStoreName` - Name of the related `CephObjectStore` object. Mandatory.
- `spec` - Represents the Gateway API `HTTPRoute` specification. For details, see [Gateway API documentation: HTTPRoute](https://gateway-api.sigs.k8s.io/reference/api-types/httproute/).

??? "Example configuration of HTTPRoute specification"

    ```yaml
    spec:
      objectStorage:
        gatewayHTTPRoutes:
        - name: route-1
          objectStoreName: rgw-store
          spec:
            hostnames:
            - rgw-store.custom.dns.name
        objectStores:
        - name: rgw-store
          spec:
            ...
            hosting:
              dnsNames:
              - rgw-store.custom.dns.name
            ...
    ```

<a name="cephdeployment-cephfs-parameters"></a>
### CephFilesystems parameters

- `cephFS` - Contains a list of Ceph file systems. Deprecated, automatically migrated to the `cephFilesystems` field.
- `cephFilesystems` - Contains a list of Ceph file systems. Each `cephFilesystem` item represents the Rook `CephFilesystem` specification:

    {% include "../snippets/cephfsParameters.md" %}

??? "Example configuration of sharedFilesystem specification"

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
    ```

<a name="cephdeployment-rookconfig-parameters"></a>
### RookConfig parameters

`RookConfig` is a string key-value parameter that allows overriding Ceph configuration options.

Use the `|` delimiter to specify the section where a parameter
must be placed. For example, `mon` or `osd`. And, if required,
use the `.` delimiter to specify the exact number of the Ceph OSD or
Ceph Monitor to apply an option to a specific `mon` or `osd` and
override the configuration of the corresponding section.

The use of this option enables restart of only specific daemons related
to the corresponding section. If you do not specify the section,
a parameter is set in the `global` section, which includes restart
of all Ceph daemons except Ceph OSD.

```yaml
spec:
  rookConfig:
    "osd_max_backfills": "64"
    "mon|mon_health_to_clog":  "true"
    "osd|osd_journal_size": "8192"
    "osd.14|osd_journal_size": "6250"
```

<a name="cephdeployment-extraopts-parameters"></a>
### ExtraOpts parameters

- `deviceLabels` - Optional. A key-value mapping which is used to assign a specification label to any
  available device on a specific node. These labels can then be used for the
  `nodes` section items with `nodeGroup` or `nodesByLabel` defined to
  eliminate the need to specify different devices for each node individually.
  Additionally, it helps in avoiding the use of device names, facilitating
  the grouping of nodes with similar labels.

    Example usage:

    ```yaml
    spec:
      extraOpts:
        deviceLabels:
          <node-name>:
            <dev-label>: /dev/disk/by-id/<unique_ID>
            ...
            <dev-label-n>: /dev/disk/by-id/<unique_ID>
          ...
          <node-name-n>:
            <dev-label>: /dev/disk/by-id/<unique_ID>
            ...
            <dev-label-n>: /dev/disk/by-id/<unique_ID>
      nodes:
      - name: <group-name>
        devices:
        - name: <dev_label>
        - name: <dev_label_n>
        nodes:
        - <node_name>
        - <node_name_n>
    ```

- `customDeviceClasses` - Optional. A list of custom device class names to use in the
  specification. Enables you to specify the custom names different from
  the default ones, which include `ssd`, `hdd`, and `nvme`, and use
  them in nodes and pools definitions.

    Example usage:

    ```yaml
    spec:
      extraOpts:
        customDeviceClasses:
        - <custom_class_name>
      nodes:
      - name: kaas-node-5bgk6
        devices:
        - config: # existing item
          deviceClass: <custom_class_name>
          fullPath: /dev/disk/by-id/<unique_ID>
      pools:
      - name: pool1
        storageClassOpts:
          default: false
        spec:
          deviceClass: <custom_class_name>
          erasureCoded:
            codingChunks: 1
            dataChunks: 2
          failureDomain: host
    ```

<a name="cephdeployment-rbd-mirroring-parameters"></a>
### RBD mirroring parameters

- `daemonsCount` - Count of `rbd-mirror` daemons to spawn. We recommend using one instance of the `rbd-mirror` daemon.
- `peers` - Optional. List of mirroring peers of an external cluster to connect to. Only a single peer is supported.
   The `peer` section includes the following parameters:

     - `site` - Label of a remote Ceph cluster associated with the token.
     - `token` - Token to be used by one site (Ceph cluster) to pull images from another site.
       To obtain the token, use the `rbd mirror pool peer bootstrap create` command.
     - `pools` - Optional. A list of pool names to mirror.

<a name="cephdeployment-status-fields"></a>
## Status fields

- `phase` - Current handling phase of the applied Ceph cluster spec. Can equal to `Creating`, `Deploying`, `Validation`, `Ready`, `Deleting`, `OnHold` or `Failed`.
- `message` - Detailed description of the current phase or an error message if the phase is `Failed`.
- `lastRun` - `DateTime` when previous spec reconcile occurred.
- `clusterVersion` - Current Ceph cluster version, for example, `v19.2.3`.
- `validation` - Validation result (`Succeed` or `Failed`) of the spec with a list of messages, if any. The `validation` section includes the following fields:

   - `result` - `Succeed` or `Failed`
   - `messages` - List of error messages
   - `lastValidatedGeneration` - Last validated `metadata.generation` of `CephDeployment`

- `objRefs` - Pelagia API object refereneces such as `CephDeploymentHealth` and `CephDeploymentSecret`.
