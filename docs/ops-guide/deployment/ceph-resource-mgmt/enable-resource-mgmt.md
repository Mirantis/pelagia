---
description: How to enable and configure Ceph tolerations and resource management.
keywords: pelagia, enable ceph resources, ceph resources, ceph tolerations, ceph resource management
---

<a id="enable-resource-mgmt-enable-management-of-ceph-tolerations-and-resources"></a>
# Enable management of Ceph tolerations and resources

!!! warning

    This document does not provide any specific recommendations on
    requests and limits for Ceph resources. The document stands for a native
    Ceph resources configuration.

You can configure Pelagia to manage Ceph resources by specifying their
requirements and constraints. To configure the resource consumption for Ceph
nodes, consider the following options that are based on different Helm release
configuration values:

* Configuring tolerations for taint nodes for the Ceph Monitor, Ceph Manager,
  and Ceph OSD daemons. For details, see
  [Kubernetes documentation: Taints and Tolerations](https://kubernetes.io/docs/concepts/scheduling-eviction/taint-and-toleration/).
* Configuring node resource requests or limits for the Ceph daemons and for
  each Ceph OSD device class such as HDD, SSD, or NVMe. For details, see
  [Kubernetes documentation: Managing Resources for Containers](https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/).

    !!! warning

        Affinity placement settings specified in the `CephDeployment` spec for Ceph daemons will be ignored in favor of using node roles.

**To enable management of Ceph tolerations and resources:**

1. To avoid Ceph cluster health issues during daemon configuration changes,
   set Ceph `noout`, `nobackfill`, `norebalance`, and `norecover`
   flags through the `pelagia-ceph-toolbox` pod before editing Ceph tolerations
   and resources:
   ```bash
   kubectl -n rook-ceph exec deploy/pelagia-ceph-toolbox -- bash
   ceph osd set noout
   ceph osd set nobackfill
   ceph osd set norebalance
   ceph osd set norecover
   exit
   ```

    !!! note

         Skip this step if you are only configuring the PG rebalance timeout and replicas count parameters.

2. Open the `CephDeployment` custom resource (CR) for editing:
   ```bash
   kubectl -n pelagia edit cephdpl
   ```

3. In the `cluster.placement` section, specify the placement parameters.
   For reference, see [Rook API documentation: Placement Configuration Settings](https://rook.io/docs/rook/v1.19/CRDs/Cluster/ceph-cluster-crd/#placement-configuration-settings).

    ??? "Example configuration"

        ```yaml
        spec:
          cluster:
            placement:
              tolerations:
                mon:
                  rules:
                  - effect: NoSchedule
                    key: node-role.kubernetes.io/controlplane
                    operator: Exists
                mgr:
                  rules:
                  - effect: NoSchedule
                    key: node-role.kubernetes.io/controlplane
                    operator: Exists
                osd:
                  rules:
                  - effect: NoSchedule
                    key: node-role.kubernetes.io/controlplane
                    operator: Exists
        ```

4. In the `cluster.resources` section, specify the resource requirement parameters.
   For reference, see [Rook API documentation: Resource Requirements/Limits](https://rook.io/docs/rook/v1.19/CRDs/Cluster/ceph-cluster-crd/#resource-requirementslimits).

    ??? "Example configuration"

        ```yaml
        spec:
          cluster:
            resources:
              mon:
                requests:
                  memory: 1Gi
                  cpu: 2
                limits:
                  memory: 2Gi
                  cpu: 3
              mgr:
                requests:
                  memory: 1Gi
                  cpu: 2
                limits:
                  memory: 2Gi
                  cpu: 3
              osd:
                requests:
                  memory: 1Gi
                  cpu: 2
                limits:
                  memory: 2Gi
                  cpu: 3
              osd-hdd:
                requests:
                  memory: 1Gi
                  cpu: 2
                limits:
                  memory: 2Gi
                  cpu: 3
              osd-ssd:
                requests:
                  memory: 1Gi
                  cpu: 2
                limits:
                  memory: 2Gi
                  cpu: 3
              osd-nvme:
                requests:
                  memory: 1Gi
                  cpu: 2
                limits:
                  memory: 2Gi
                  cpu: 3
        ```

5. In the `spec.nodes` section, specify the `resources` parameters for the Ceph node-specific resources:
   ```yaml
   spec:
     nodes:
     - name: <nodeName>
       resources:
         requests: <kubernetes valid spec of daemon resource requests>
         limits: <kubernetes valid spec of daemon resource limits>
   ```

     Substitute `<nodeName>` with the node requested for specific resources.
     For example:
     ```yaml
     spec:
       nodes:
       - name: kaas-node-worker-1
         resources:
           requests:
             memory: 1Gi
             cpu: 2
           limits:
             memory: 2Gi
             cpu: 3
     ```

6. In the `spec.objectStorage` section, specify the `resources` and `placement` parameters for the RADOS Gateway instances:
   ```yaml
   spec:
     objectStorage:
       objectStore:
       - name: rgw-store
         spec:
           ...
           gateway:
             resources:
               requests: <kubernetes valid spec of daemon resource requests>
               limits: <kubernetes valid spec of daemon resource limits>
             placement:
               tolerations:
               - effect: NoSchedule
                 key: node-role.kubernetes.io/control-plane
                 operator: Exists
               - key: node-role.kubernetes.io/master
                 effect: NoSchedule
                 operator: Exists
   ```

     For reference, see [Rook API documentation: Gateway Settings](https://rook.io/docs/rook/v1.19/CRDs/Object-Storage/ceph-object-store-crd/#gateway-settings).

7. In the `spec.sharedFilesystems` section, specify the `resources` and `placement` parameters for the CephFS daemons:
    ```yaml
    spec:
      sharedFilesystem:
        cephFilesystems:
        - name: cephfs-store
          spec:
            ...
            metadataServer:
              placement:
                tolerations:
                - effect: NoSchedule
                  key: node-role.kubernetes.io/control-plane
                  operator: Exists
                - key: node-role.kubernetes.io/master
                  effect: NoSchedule
                  operator: Exists
              resources: # example, non-prod values
                requests:
                  memory: 1Gi
                  cpu: 1
                limits:
                  memory: 2Gi
                  cpu: 2
    ```

    For reference, see [Rook API documentation: CephFilesystem](https://rook.io/docs/rook/v1.19/CRDs/Shared-Filesystem/ceph-filesystem-crd/#metadata-server-settings).

8. Save the reconfigured `CephDeployment` CR and wait for Pelagia Deployment Controller to apply the updated
   Ceph configuration to Rook.
   Rook will recreate Ceph Monitors, Ceph Managers, and Ceph OSDs according to the
   specified configuration.
9. Specify tolerations for different Rook resources using Pelagia Helm chart values. For details, see
   [Specify Rook daemons placement](../rook-daemon-place.md#rook-daemon-place-specify-rook-daemons-placement).
10. After a successful Ceph reconfiguration, unset the flags set in step 1
    through the `pelagia-ceph-toolbox` pod:
    ```bash
    kubectl -n rook-ceph exec -it deploy/pelagia-ceph-toolbox -- bash
    ceph osd unset
    ceph osd unset noout
    ceph osd unset nobackfill
    ceph osd unset norebalance
    ceph osd unset norecover
    exit
    ```

    !!! note

        Skip this step if you have only configured the PG rebalance timeout and replicas count parameters.

Once done, proceed to [Verify Ceph tolerations and resources](./verify-resource-mgmt.md#verify-resource-mgmt-verify-ceph-tolerations-and-resources).
