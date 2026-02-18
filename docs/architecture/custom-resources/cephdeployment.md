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
      pools:
      - default: true
        deviceClass: ssd
        name: kubernetes
        replicated:
          size: 3
      objectStorage:
        rgw:
          name: rgw-store
          dataPool:
            deviceClass: ssd
            replicated:
              size: 3
          metadataPool:
            deviceClass: ssd
            replicated:
              size: 3
          gateway:
            allNodes: false
            instances: 3
            port: 8081
            securePort: 8443
          preservePoolsOnDelete: false
      sharedFilesystem:
        cephFS:
        - name: cephfs-store
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
            activeStandby: false
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

4. Verify `CephDeployment` reconcile status with the Status fields.

## CephDeployment configuration options

The following subsections contain a description of `CephDeployment` parameters for an
advanced configuration.

<a name="cephdeployment-general-parameters"></a>
### General parameters

- `network` - Specifies access and public networks for the Ceph cluster. For details, see [Network parameters](#cephdeployment-network-parameters).
- `nodes` - Specifies the list of Ceph nodes. For details, see [Node parameters](#cephdeployment-nodes-parameters). The `nodes` parameter is a list with Ceph node specifications. List item could define Ceph node specification for a single node or a group of nodes listed or defined by label. It could be also combined.
- `pools` - Specifies the list of Ceph pools. For details, see [Pool parameters](#cephdeployment-pools-parameters).
- `clients` - List of Ceph clients. For details, see [Clients parameters](#cephdeployment-clients-parameters).
- `objectStorage` - Specifies the parameters for Object Storage, such as RADOS Gateway, the Ceph Object Storage. Also specifies the RADOS Gateway Multisite configuration. For details, see [RADOS Gateway parameters](#cephdeployment-rados-gateway-parameters) and [Multisite parameters](#cephdeployment-rados-gateway-multisite-parameters).
- `ingressConfig` - Enables a custom ingress rule for public access on Ceph services, for example, Ceph RADOS Gateway. For details, see [Configure Ceph Object Gateway TLS](https://mirantis.github.io/pelagia/ops-guide/deployment/rgw-tls).
- `sharedFilesystem` - Enables Ceph Filesystem. For details, see [CephFS parameters](#cephdeployment-cephfs-parameters).
- `rookConfig` - String key-value parameter that allows overriding Ceph configuration options. For details, see [RookConfig parameters](#cephdeployment-rookconfig-parameters).
- `healthCheck` - Configures health checks and liveness probe settings for Ceph daemons. For details, see [Health check parameters](#cephdeployment-healthcheck-parameters).
- `extraOpts` - Enables specification of extra options for a setup, includes the `deviceLabels` parameter. Refer to [Extra options](#cephdeployment-extraopts-parameters) for details.
- `mgr` - Specifies a list of Ceph Manager modules to be enabled or disabled. For details, see [Manager modules parameters](#cephdeployment-manager-modules-parameters). Modules `balancer` and `pg_autoscaler` are enabled by default.
- `dashboard` - Enables Ceph dashboard. Currently, Pelagia has no support of Ceph Dashboard. Defaults to `false`.
- `rbdMirror` - Specifies the parameters for RBD Mirroring. For details, see [RBD Mirroring parameters](#cephdeployment-rbd-mirroring-parameters).
- `external` - Enables external Ceph cluster mode. If enabled, Pelagia will read a special `Secret` with external Ceph cluster credentials data connect to.

<a name="cephdeployment-network-parameters"></a>
### Network parameters

- `clusterNet` - specifies a Classless Inter-Domain Routing (CIDR)
  for the Ceph OSD replication network.

    !!! warning

        To avoid ambiguous behavior of Ceph daemons, do not specify
        ``0.0.0.0/0`` in ``clusterNet``. Otherwise, Ceph daemons can select
        an incorrect public interface that can cause the Ceph cluster to
        become unavailable.

    !!! note

        The `clusterNet` and `publicNet` parameters support
        multiple IP networks. For details, see [Enable multinetworking](../../ops-guide/deployment/multinetworking.md#multinetworking-enable-ceph-multinetwork).

- `publicNet` - specifies a CIDR for communication between
  the service and operator.

    !!! warning

        To avoid ambiguous behavior of Ceph daemons, do not specify
        ``0.0.0.0/0`` in ``publicNet``. Otherwise, Ceph daemons can select
        an incorrect public interface that can cause the Ceph cluster to
        become unavailable.

    !!! note

        The ``clusterNet`` and ``publicNet`` parameters support
        multiple IP networks. For details, see [Enable multinetworking](../../ops-guide/deployment/multinetworking.md#multinetworking-enable-ceph-multinetwork).

Example configuration:
```yaml
spec:
  network:
    clusterNet: 10.10.0.0/24
    publicNet:  192.100.0.0/24
```
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

- ``monitorIP`` - Optional. If defined, specifies a custom IP address for Ceph Monitor which
  should be placed on the node. If not set, Ceph Monitor on the node will use
  default `hostNetwork` address of a node. General recommendation is to use IP
  address from Ceph public network address range.

    !!! note

        To update ``monitorIP`` the corresponding Ceph Monitor daemon should be re-created.

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
        [Addressing Ceph storage devices](../addressing-ceph-devices.md#addressing-ceph-devices-addressing-ceph-storage-devices).

    - ``fullPath`` - a storage device symlink. Accepts the following values:

        - The device ``by-id`` symlink that contains the serial number of the
          physical device and does not contain ``wwn``. For example,
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
          physical device and does not contain ``wwn``. For example,
          ``/dev/disk/by-id/nvme-SAMSUNG_MZ1LB3T8HMLA-00007_S46FNY0R394543``.
        - The device label from ``extraOpts.deviceLabels`` section which is
          generally used for templating Ceph node specification for node groups.
          For details, see [Extra options](#cephdeployment-extraopts-parameters).

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

- `deviceClass` - Mandatory. Specifies the device class for the defined pool. Common possible
  values are `hdd`, `ssd` and `nvme`. Also allows customized device classes, refers to [Extra options](#cephdeployment-extraopts-parameters).
- `replicated` - The `replicated` parameter is mutually exclusive with `erasureCoded`
  and includes the following parameters:

    - `size` - the number of pool replicas.
    - `targetSizeRatio` - A float percentage from `0.0` to `1.0`, which
      specifies the expected consumption of the total Ceph cluster capacity.
      The default values are as follows:

        - The default ratio of the Ceph Object Storage `dataPool` 10.0%.
        - Target ratios for the pools required for Rockoon, described in
          [Integrate Pelagia with Rockoon](../../ops-guide/rockoon/rockoon-integration.md#rockoon-integration-integrate-pelagia-with-rockoon).

        !!! note

            Mirantis recommends defining target ratio with the `parameters.target_size_ratio` **string** field instead.

- `erasureCoded` - Enables the erasure-coded pool. For details, see [Rook documentation: Erasure Coded RBD Pool](https://rook.io/docs/rook/latest-release/CRDs/Block-Storage/ceph-block-pool-crd/#erasure-coded-rbd-pool).
  and [Ceph documentation: Erasure coded pool](https://docs.ceph.com/en/latest/dev/erasure-coded-pool/). The
  `erasureCoded` parameter is mutually exclusive with `replicated`.
- `failureDomain` - Optional. The failure domain across which the replicas or chunks
  of data will be spread. Set to `host` by default. The list of
  possible recommended values includes: `host`, `rack`, `room`,
  and `datacenter`.

    !!! caution

        We do not recommend using the following intermediate topology keys: `pdu`, `row`, `chassis`. Consider
        the `rack` topology instead. The `osd` failure domain is prohibited.

- `mirroring` - Optional. Enables the mirroring feature for the defined pool.
  Includes the `mode` parameter that can be set to `pool` or `image`. For details, see [Enable Ceph RBD Mirroring](../../ops-guide/deployment/rbd-mirror.md#rbd-mirror-enable-ceph-rbd-mirroring).
- `parameters` - Optional. Specifies the key-value map for the parameters of the Ceph pool.
  For details, see [Ceph documentation: Set Pool values](https://docs.ceph.com/en/latest/rados/operations/pools/#set-pool-values).
- `enableCrushUpdates` - Optional. Enables automatic updates of the CRUSH map
  when the pool is created or updated. Defaulted to `false`.

??? "Example configuration of Pools specification"

    ```yaml
    spec:
      pools:
      - name: kubernetes
        deviceClass: hdd
        replicated:
          size: 3
        parameters:
          target_size_ratio: "10.0"
        storageClassOpts:
          default: true
        preserveOnDelete: true
      - name: kubernetes
        deviceClass: nvme
        erasureCoded:
          codingChunks: 1
          dataChunks: 2
        failureDomain: host
      - name: archive
        useAsFullName: true
        deviceClass: hdd
        failureDomain: rack
        replicated:
          size: 3
    ```

As a result, the following Ceph pools will be created: `kubernetes-hdd`, `kubernetes-nvme`, and `archive`.

To configure additional required pools for Rockoon, see
[Integrate Pelagia with Rockoon](../../ops-guide/rockoon/rockoon-integration.md#rockoon-integration-integrate-pelagia-with-rockoon).

!!! caution

    Since Ceph Pacific, Ceph CSI driver does not propagate the `777`
    permission on the mount point of persistent volumes based on any
    `StorageClass` of the Ceph pool.

<a name="cephdeployment-clients-parameters"></a>
### Clients parameters

- `name` - Mandatory. Ceph client name.
- `caps` - Mandatory. Key-value parameter with Ceph client capabilities. For details about
  `caps`, refer to [Ceph documentation: Authorization (capabilities)](https://docs.ceph.com/en/latest/rados/operations/user-management/#authorization-capabilities).

??? "Example configuration of Clients specification"

    ```yaml
    spec:
      clients:
      - name: test-client
        caps:
          mon: allow r, allow command "osd blacklist"
          osd: profile rbd pool=kubernetes-nvme
    ```
<a name="cephdeployment-rados-gateway-parameters"></a>
### RADOS Gateway parameters 

{% include "../../snippets/rgwParameters.md" %}

??? "Example configuration of RADOS gateway specification"

    ```yaml
    spec:
      objectStorage:
        rgw:
          name: rgw-store
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
            allNodes: false
            instances: 3
            port: 80
            securePort: 8443
          preservePoolsOnDelete: false
    ```

<a name="cephdeployment-rados-gateway-multisite-parameters"></a>
### RADOS Gateway Multisite parameters

!!! warning

    This feature is in Technical Preview, use it on own risk.

{% include "../../snippets/multisiteParameters.md" %}

For configuration example, see [Enable Multisite for Ceph Object Storage](../../ops-guide/deployment/rgw-multisite.md#rgw-multisite-enable-multisite-for-ceph-object-storage).

<a name="cephdeployment-cephfs-parameters"></a>
### CephFS parameters

`sharedFilesystem` contains a list of Ceph Filesystems `cephFS`. Each `cephFS` item
contains the following parameters:

{% include "../../snippets/cephfsParameters.md" %}

??? "Example configuration of shared Filesystem specification"

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

<a name="cephdeployment-rookconfig-parameters"></a>
### RookConfig parameters

String key-value parameter that allows overriding Ceph configuration options.

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

<a name="cephdeployment-healthcheck-parameters"></a>
### HealthCheck parameters

- `daemonHealth` - Optional. Specifies health check settings for Ceph daemons. Contains
  the following parameters:

    - `status` - configures health check settings for Ceph health
    - `mon` - configures health check settings for Ceph Monitors
    - `osd` - configures health check settings for Ceph OSDs

    Each parameter allows defining the following settings:

      - `disabled` - a flag that disables the health check.
      - `interval` - an interval in seconds or minutes for the health
        check to run. For example, `60s` for 60 seconds.
      - `timeout` - a timeout for the health check in seconds or minutes.
        For example, `60s` for 60 seconds.

- `livenessProbe` - Optional. Key-value parameter with liveness probe settings for
  the defined daemon types. Can be one of the following: `mgr`,
  `mon`, `osd`, or `mds`. Includes the `disabled` flag and
  the `probe` parameter. The `probe` parameter accepts
  the following options:

    - `initialDelaySeconds` - the number of seconds after the container
      has started before the liveness probes are initiated. Integer.
    - `timeoutSeconds` - the number of seconds after which the probe
      times out. Integer.
    - `periodSeconds` - the frequency (in seconds) to perform the
      probe. Integer.
    - `successThreshold` - the minimum consecutive successful probes
      for the probe to be considered successful after a failure. Integer.
    - `failureThreshold` - the minimum consecutive failures for the
      probe to be considered failed after having succeeded. Integer.

    !!! note

        Pelagia Deployment Controller specifies the following `livenessProbe` defaults
        for `mon`, `mgr`, `osd`, and `mds` (if CephFS is enabled):

          - `5` for `timeoutSeconds`
          - `5` for `failureThreshold`

- `startupProbe` - Optional. Key-value parameter with startup probe settings for
  the defined daemon types. Can be one of the following: `mgr`,
  `mon`, `osd`, or `mds`. Includes the `disabled` flag and
  the `probe` parameter. The `probe` parameter accepts
  the following options:

    - `timeoutSeconds` - the number of seconds after which the probe
      times out. Integer.
    - `periodSeconds` - the frequency (in seconds) to perform the
      probe. Integer.
    - `successThreshold` - the minimum consecutive successful probes
      for the probe to be considered successful after a failure. Integer.
    - `failureThreshold` - the minimum consecutive failures for the
      probe to be considered failed after having succeeded. Integer.

??? "Example configuration of health check specification"

    ```yaml
    spec:
      healthCheck:
        daemonHealth:
          mon:
            disabled: false
            interval: 45s
            timeout: 600s
          osd:
            disabled: false
            interval: 60s
          status:
            disabled: true
        livenessProbe:
          mon:
            disabled: false
            probe:
              timeoutSeconds: 10
              periodSeconds: 3
              successThreshold: 3
          mgr:
            disabled: false
            probe:
              timeoutSeconds: 5
              failureThreshold: 5
          osd:
            probe:
              initialDelaySeconds: 5
              timeoutSeconds: 10
              failureThreshold: 7
        startupProbe:
          mon:
            disabled: true
          mgr:
            probe:
              successThreshold: 3
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

- `customDeviceClasses` - Optional. TechPreview. A list of custom device class names to use in the
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
      - default: false
        deviceClass: <custom_class_name>
        erasureCoded:
        codingChunks: 1
        dataChunks: 2
        failureDomain: host
    ```

<a name="cephdeployment-manager-modules-parameters"></a>
### Manager modules parameters

`CephDeployment` specification `mgr` section contains `mgrModules` parameter. It includes the following
parameters:

- `name` - Ceph Manager module name.
- `enabled` - Flag that defines whether the Ceph Manager module
  is enabled.

     For example:

     ```yaml
     spec:
       mgr:
         mgrModules:
         - name: balancer
           enabled: true
         - name: pg_autoscaler
           enabled: true
     ```

     The `balancer` and `pg_autoscaler` Ceph Manager modules are
     enabled by default and cannot be disabled.

!!! note

    Most Ceph Manager modules require additional configuration that you can perform through the `pelagia-lcm-tooblox`
    pod.

<a name="cephdeployment-rbd-mirroring-parameters"></a>
### RBD Mirroring parameters

- `daemonsCount` - Count of `rbd-mirror` daemons to spawn. We recommend using one instance of the `rbd-mirror` daemon.                                                                                                                                                                                                                                                                                                                                                                                                                  |
- `peers` - Optional. List of mirroring peers of an external cluster to connect to. Only a single peer is supported.
   The `peer` section includes the following parameters:

     - `site` - the label of a remote Ceph cluster associated with the token.
     - `token` - the token that will be used by one site (Ceph cluster) to pull images from the other site.
       To obtain the token, use the **rbd mirror pool peer bootstrap create** command.
     - `pools` - optional, a list of pool names to mirror.

<a name="cephdeployment-status-fields"></a>
## Status fields

- `phase` - Current handling phase of the applied Ceph cluster spec. Can equal to `Creating`, `Deploying`, `Validation`, `Ready`, `Deleting`, `OnHold` or `Failed`.
- `message` - Detailed description of the current phase or an error message if the phase is `Failed`.
- `lastRun` - `DateTime` when previous spec reconcile occurred.
- `clusterVersion` - Current Ceph cluster version, for example, `v19.2.3`.
- `validation` - Validation result (`Succeed` or `Failed`) of the spec with a list of messages, if any. The `validation` section includes the following fields:

   - `result` - Succeed or Failed
   - `messages` - the list of error messages
   - `lastValidatedGeneration` - the last validated `metadata.generation` of `CephDeployment`

- `objRefs` - Pelagia API object refereneces such as `CephDeploymentHealth` and `CephDeploymentSecret`.
