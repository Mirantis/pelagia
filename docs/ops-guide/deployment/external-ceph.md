<a id="external-ceph-share-ceph-across-two-clusters"></a>
# Share Ceph across two clusters

{% include "../../snippets/techpreview.md" %}

This section describes how to share a Ceph cluster with another Kubernetes cluster and how to manage such Ceph cluster.

A shared Ceph cluster allows connecting of a consumer cluster to a producer cluster.
The consumer cluster uses the Ceph cluster deployed on the producer
to store the necessary data. In other words, the producer cluster contains the
Ceph cluster with ``mon``, ``mgr``, ``osd``, and ``mds`` daemons. And the
consumer cluster contains clients that require access to the Ceph storage.
For example, an NGINX application that runs in a cluster without storage
requires a persistent volume to store data. In this case, such a cluster can
connect to a Ceph cluster and use it as a block or file storage.

!!! warning

    Limitations:

      - The producer and consumer clusters must have the same Ceph external and cluster networks reachable.
      - Monitor endpoints' network of the producer cluster must be available in the consumer cluster.

## Plan a shared Ceph cluster

To plan a shared Ceph cluster, select resources to share on the producer Ceph cluster:

- Select the RADOS Block Device (RBD) pools to share from the Ceph cluster.
- Select the CephFS name to share from the Ceph cluster.

### Obtain resources to share on the producer Ceph cluster

1. Open the ``CephDeployment`` object.
2. In ``pools`` section, identify the Ceph cluster pools assigned to RBD pools. To obtain full names of RBD pools:

     ```bash
     kubectl -n rook-ceph exec -it deploy/rook-ceph-tools -- ceph osd lspools
     ```
      
     Example of system response:

       ```bash
       ...
       2 kubernetes-hdd
       3 anotherpool-hdd
       ...
       ```

      In the example above, ``kubernetes-hdd`` and ``anotherpool-hdd`` are RBD pools.

3. In ``sharedFilesystem``, identify the CephFS name. For example:
   
     ```yaml
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

     In the example above, the CephFS name is ``cephfs-store``.

## Create a Ceph non-admin client for a shared Ceph cluster

Ceph requires a non-admin client to share the producer cluster resources with
the consumer cluster. To connect the consumer cluster with the producer
cluster, the Ceph client requires the following ``caps`` (permissions):

- Read-write access to Ceph Managers.
- Read and role-definer access to Ceph Monitors.
- Read-write access to Ceph Metadata servers if CephFS pools must be shared.
- Profile access to shared RBD/CephFS pools' access for Ceph OSDs.

To create a Ceph non-admin client, add the following snippet to the ``clients`` section of the ``CephDeployment`` object:

```yaml
spec:
  clients:
  - name: <nonAdminClientName>
    caps:
      mgr: "allow rw"
      mon: "allow r, profile role-definer"
      mds: "allow rw" # if CephFS must be shared
      osd: <poolsProfileCaps>
```

Substitute ``<nonAdminClientName>`` with a Ceph non-admin client name and ``<poolsProfileCaps>`` with a comma-separated profile list of RBD and CephFS pools in the following format:

- ``profile rbd pool=<rbdPoolName>`` for each RBD pool;
- ``allow rw tag cephfs data=<cephFsName>`` for each CephFS pool.

For example:

```yaml
spec:
  clients:
  - name: non-admin-client
    caps:
      mgr: "allow rw"
      mon: "allow r, profile role-definer"
      mds: "allow rw"
      osd: "profile rbd pool=kubernetes-hdd,profile rbd pool=anotherpool-hdd,allow rw tag cephfs data=cephfs-store"
```

To verify the status of the created Ceph client, inspect the ``status`` section of the ``MiraCephHealth`` object. For example:

```yaml
status:
  fullClusterInfo:
    blockStorageStatus:
      clientsStatus:
        non-admin-client:
          present: true
          status: Ready

```

## Connect the producer to the consumer

1. Install Pelagia Helm release in the consumer cluster.

2. On the producer cluster, obtain the previously created Ceph non-admin client as described above to use it as ``<clientName>`` in the following step.

3. On the producer cluster, generate connection string:

     ```bash
     kubectl -n <pelagiaNamespace> exec -it deploy/pelagia-deployment-controller -- sh
     /usr/local/bin/pelagia-connector --rook-namespace <rookNamespace> --client-name <clientName> --use-rbd --use-cephfs --toolbox-label pelagia-ceph-toolbox --toolbox-ns <rookNamespace> --use-rgw --rgw-username rgw-admin-ops-user --base64
     ```

     Substitute the following:

       - `pelagiaNamespace` with Pelagia namespace;
       - `rookNamespace` with Rook namespace;
       - `clientName` with created Ceph non-admin client.

     Example of output:
     ```bash
     eyJjbGllbnRfbmFtZSI6ImFkbWluIiwiY2xpZW50X2tleXJpbmciOiJBUUJabW5kcE9HNkhMUkFBUFBQT0tQWkcxTEI0N2EwWmpLQ0t6dz09IiwiZnNpZCI6IjYzNDg1MzQwLWJkNzMtNDNiOS1iNWQyLTY5ZjQ0OTM4N2I3OSIsIm1vbl9lbmRwb2ludHNfbWFwIjoiYT0xMC4xMC4wLjE3OjY3ODksYj0xMC4xMC4wLjE3MDo2Nzg5LGM9MTAuMTAuMC4xMzY6Njc4OSIsInJnd19hZG1pbl9rZXlzIjp7ImFjY2Vzc0tleSI6IjJOVlVTM0xKMDMwTEMzUTE4TTY0Iiwic2VjcmV0S2V5Ijoia0FPU2l3Q2xEeWVpdHpiT3prbGlySEVqWlNBN0JVUnZ6dzh1RExIcCJ9fQ==
     ```

4. On the consumer cluster, create the consumer ``CephDeployment`` object file with the following content:

     ```yaml
     apiVersion: lcm.mirantis.com/v1alpha1
     kind: CephDeployment
     metadata:
       name: consumer-ceph
       namespace: <consumerPelagiaNamespace>
     spec:
       external:
         enable: true
         connectionString: <generatedConnectionString>
       network:
          clusterNet: <clusterNetCIDR>
          publicNet: <publicNetCIDR>
       nodes: {}
     ```

     Specify the following values:

       - ``<generatedConnectionString>`` is the connection string generated in the previous step.
       - ``<clusterNetCIDR>`` and ``<publicNetCIDR>`` are values that must match the same values in the producer cluster.

5. Apply the file on the consumer cluster:
   ```bash
   kubectl apply -f consumer-cephdpl.yaml
   ```

Once the Ceph cluster is specified in the ``CephDeployment`` CR of the consumer cluster, Pelagia validates it and requests Rook to connect the consumer and producer.

## Consume pools from the Ceph cluster

In the ``spec.pools`` of the consumer ``CephDeployment``, specify pools from the producer cluster to be used by the consumer cluster. For example:

```yaml
pools:
- deviceClass: ssd
  useAsFullName: true
  name: kubernetes-ssd
  storageClassOpts:
    default: true
- deviceClass: hdd
  useAsFullName: true
  name: volumes-hdd
```

!!! danger

    Each ``name`` in the ``pools`` section must match the corresponding full pool ``name`` of the producer cluster.
    You can find full pools ``name`` in the ``CephDeploymentHealth`` CR.

After specifying pools in the consumer ``CephDeployment`` CR, Pelagia creates a corresponding ``StorageClass`` for each specified pool, which can be used for creating ``ReadWriteOnce`` persistent volumes (PVs) in the consumer cluster.

## Enable CephFS on a consumer Ceph cluster

In the ``sharedFilesystem`` section of the consumer ``CephDeployment``, specify the ``dataPools`` to share.

!!! note

    Sharing ``CephFS`` also requires specifying the ``metadataPool`` and ``metadataServer`` sections similarly to the corresponding sections of the producer cluster.

For example:
```yaml
sharedFilesystem:
  cephFS:
  - name: cephfs-store
    dataPools:
    - name: cephfs-pool-1
      replicated:
        size: 3
      failureDomain: host
    metadataPool:
      replicated:
        size: 3
      failureDomain: host
    metadataServer:
      activeCount: 1
      activeStandby: false
```

After specifying CephFS in the ``CephDeployment`` CR of the consumer cluster, Pelagia creates a corresponding ``StorageClass`` that allows creating ``ReadWriteMany`` (RWX) PVs in the consumer cluster.
