# Install Pelagia

This section provides instructions on how to install Pelagia that will deploy a Ceph cluster managed by Rook on a Kubernetes cluster, for example, deployed with [k0s](https://docs.k0sproject.io/stable/).

## Requirements and prerequisites

* A Kubernetes cluster, for example, deployed with k0s.
* Enough resources for a Rook installation as described in [Rook documentation: Prerequisites](https://rook.github.io/docs/rook/latest-release/Getting-Started/Prerequisites/prerequisites/).
* Enough nodes in the cluster to run Ceph control and data daemons. Nodes must have at least 4 GB of RAM for each Ceph daemon (monitors, managers, metadata servers, osd, rgw). The minimum recommended requirements are 6 nodes: 3 nodes for control daemons (Ceph Managers and Ceph Monitors) and 3 nodes for data daemons (Ceph OSDs).
For details about hardware recommendations, see [Ceph documentation: Hardware recommendations](https://docs.ceph.com/en/latest/start/hardware-recommendations/).

* At least one disk per data node to be used as Ceph OSD. Miminal disk size is `5G` but Mirantis recommends using disks with at least `100G` size.

## Installation

To install Pelagia, use the Helm chart provided in the repository:

```bash
export PELAGIA_VERSION="<version>"
helm upgrade --install pelagia-ceph oci://registry.mirantis.com/pelagia/pelagia-ceph --version ${PELAGIA_VERSION} -n pelagia --create-namespace
```

Substitute `<version>` with the latest stable version of Pelagia Helm chart.

This command installs Pelagia controllers in the `pelagia` namespace.
If the namespace does not exist, Helm creates it. As a result, the following controllers appear
in `pelagia` namespace:

```bash
$ kubectl -n pelagia get pods

NAME                                             READY   STATUS    RESTARTS   AGE
pelagia-deployment-controller-5bcb5c6464-7d8qf   2/2     Running   0          3m52s
pelagia-deployment-controller-5bcb5c6464-n2vzt   2/2     Running   0          3m52s
pelagia-deployment-controller-5bcb5c6464-vvzv2   2/2     Running   0          3m52s
pelagia-lcm-controller-6f858b6c5-4bqqr           3/3     Running   0          3m52s
pelagia-lcm-controller-6f858b6c5-6thhn           3/3     Running   0          3m52s
pelagia-lcm-controller-6f858b6c5-whq7x           3/3     Running   0          3m52s
```

Also, Pelagia Helm chart deploys the following Rook components in the `rook-ceph` namespace:
```bash
$ kubectl -n rook-ceph get pods

NAME                                  READY   STATUS    RESTARTS   AGE
rook-ceph-operator-8495877b67-m7lf5   1/1     Running   0          4m13s
```

Currently, Pelagia Deployment Controller does not support integration with the existing Rook Ceph cluster, only Pelagia Lifecycle Management Controller does. To install Pelagia in the LCM-only mode, please refer to [LCM-only installation guide](./lcm-installation.md#lcmonly-installation-guide).

## Post-installation

After the installation, you can deploy a Ceph cluster using the Pelagia `CephDeployment` custom resource.
It will create Rook resources for Rook Operator to deploy a Ceph cluster. After Rook Operator deploys
Ceph cluster, it could be managed by `CephDeployment` resource.

??? "Simple example of a `CephDeployment` resource"

    ```yaml
    apiVersion: lcm.mirantis.com/v1alpha1
    kind: CephDeployment
    metadata:
      name: pelagia-ceph
      namespace: pelagia
    spec:
      dashboard: false
      network:
        publicNet: ${CEPH_PUBLIC_NET}
        clusterNet: ${CEPH_CLUSTER_NET}
      nodes:
      - name: ${CEPH_NODE_CP_0}
        roles: [ "mon", "mgr", "mds" ]
      - name: ${CEPH_NODE_CP_1}
        roles: [ "mon", "mgr", "mds" ]
      - name: ${CEPH_NODE_CP_2}
        roles: [ "mon", "mgr", "mds" ]
      - name: ${CEPH_NODE_WORKER_0}
        devices:
        - config:
            deviceClass: hdd
          fullPath: /dev/disk/by-id/${CEPH_OSD_DEVICE_0}
      - name: ${CEPH_NODE_WORKER_1}
        devices:
        - config:
            deviceClass: hdd
          fullPath: /dev/disk/by-id/${CEPH_OSD_DEVICE_1}
      - name: ${CEPH_NODE_WORKER_2}
        devices:
        - config:
            deviceClass: hdd
          fullPath: /dev/disk/by-id/${CEPH_OSD_DEVICE_2}
      pools:
      - name: kubernetes
        deviceClass: hdd
        default: true
        replicated:
          size: 3
      objectStorage:
        rgw:
          name: rgw-store
          dataPool:
            deviceClass: hdd
            replicated:
              size: 3
          metadataPool:
            deviceClass: hdd
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
            deviceClass: hdd
            replicated:
              size: 3
          metadataPool:
            deviceClass: hdd
            replicated:
              size: 3
          metadataServer:
            activeCount: 1
            activeStandby: false
    ```

The example above contains `3` control plane nodes and `3` worker nodes in a Ceph cluster.
You can change the number of nodes and their roles according to your needs. As a result, you will have
a Ceph cluster with RBD pool, CephFS filesystem, and RGW object storage deployed in your Kubernetes cluster.
```bash
$ kubectl -n rook-ceph get pods

NAME                                                              READY   STATUS      RESTARTS   AGE
csi-cephfsplugin-lgm9v                                            2/2     Running     0          36m
csi-cephfsplugin-lr26d                                            2/2     Running     0          36m
csi-cephfsplugin-provisioner-766fddb5c8-rm4r7                     5/5     Running     0          37m
csi-cephfsplugin-provisioner-766fddb5c8-wxtpl                     5/5     Running     0          37m
csi-cephfsplugin-s67pp                                            2/2     Running     0          36m
csi-rbdplugin-provisioner-649fb94cf4-fr8nk                        5/5     Running     0          37m
csi-rbdplugin-provisioner-649fb94cf4-vcj45                        5/5     Running     0          37m
csi-rbdplugin-rjnx4                                               2/2     Running     0          36m
csi-rbdplugin-xb8qg                                               2/2     Running     0          36m
csi-rbdplugin-z48m9                                               2/2     Running     0          36m
pelagia-ceph-toolbox-8688f94564-xmtjn                             1/1     Running     0          61m
rook-ceph-crashcollector-009b4c37cd6752487fafef7c44bc6a4d-mxd5c   1/1     Running     0          56m
rook-ceph-crashcollector-2df4fa7c50e45baccd8ffa967ce36ec6-flxjn   1/1     Running     0          107s
rook-ceph-crashcollector-7a0999af02371dae358b7fcd28b3ae35-lp9jx   1/1     Running     0          111s
rook-ceph-crashcollector-8490eae0de0b41f6fcd6a61239515f12-47h5d   1/1     Running     0          56m
rook-ceph-crashcollector-cac4771a1b293acb738842052eb60bc2-qxbvf   1/1     Running     0          42s
rook-ceph-crashcollector-d7732aeb2025beef8dda88e8195f704b-fgtsh   1/1     Running     0          104s
rook-ceph-exporter-009b4c37cd6752487fafef7c44bc6a4d-995c84hb26h   1/1     Running     0          56m
rook-ceph-exporter-2df4fa7c50e45baccd8ffa967ce36ec6-7cf86d8zb24   1/1     Running     0          107s
rook-ceph-exporter-7a0999af02371dae358b7fcd28b3ae35-fb667bd7kc2   1/1     Running     0          111s
rook-ceph-exporter-8490eae0de0b41f6fcd6a61239515f12-676975jxmvc   1/1     Running     0          56m
rook-ceph-exporter-cac4771a1b293acb738842052eb60bc2-8c6f56988l8   1/1     Running     0          40s
rook-ceph-exporter-d7732aeb2025beef8dda88e8195f704b-5ff7ccbmpzd   1/1     Running     0          104s
rook-ceph-mds-cephfs-store-a-69b45f8c48-4s29t                     1/1     Running     0          56m
rook-ceph-mds-cephfs-store-b-5cd87b8765-8mqdb                     1/1     Running     0          56m
rook-ceph-mgr-a-6bf4b5b794-fhcmt                                  2/2     Running     0          57m
rook-ceph-mgr-b-56c698c596-gvtnd                                  2/2     Running     0          57m
rook-ceph-mon-a-85d5fbd55-6pchz                                   1/1     Running     0          58m
rook-ceph-mon-b-5b4668f949-s898d                                  1/1     Running     0          57m
rook-ceph-mon-c-64b6fb474b-d5zwd                                  1/1     Running     0          57m
rook-ceph-operator-8495877b67-48ztd                               1/1     Running     0          2m55s
rook-ceph-osd-0-5884bc8bcd-b4pgw                                  1/1     Running     0          111s
rook-ceph-osd-1-59844c4c76-t8kxp                                  1/1     Running     0          107s
rook-ceph-osd-2-54956b47f4-bdjgs                                  1/1     Running     0          104s
rook-ceph-osd-prepare-2df4fa7c50e45baccd8ffa967ce36ec6-njt49      0/1     Completed   0          2m4s
rook-ceph-osd-prepare-7a0999af02371dae358b7fcd28b3ae35-kllhb      0/1     Completed   0          2m8s
rook-ceph-osd-prepare-d7732aeb2025beef8dda88e8195f704b-7548x      0/1     Completed   0          2m
rook-ceph-rgw-rgw-store-a-5cbb8785ff-5bcvw                        1/1     Running     0          42s
rook-ceph-rgw-rgw-store-a-5cbb8785ff-9dzz4                        1/1     Running     0          42s
rook-discover-27b74                                               1/1     Running     0          66m
rook-discover-2t882                                               1/1     Running     0          66m
rook-discover-nqrk4                                               1/1     Running     0          66m
```

Pelagia deploys `StorageClass` resources for each Ceph pool to be used in workloads:
```bash
$ kubectl get sc

NAME                         PROVISIONER                     RECLAIMPOLICY   VOLUMEBINDINGMODE   ALLOWVOLUMEEXPANSION   AGE
cephfs-store-cephfs-pool-1   rook-ceph.cephfs.csi.ceph.com   Delete          Immediate           true                   62m
kubernetes-hdd (default)     rook-ceph.rbd.csi.ceph.com      Delete          Immediate           false                  62m
rgw-storage-class            rook-ceph.ceph.rook.io/bucket   Delete          Immediate           false                  65m
```
