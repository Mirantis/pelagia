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

## Enable management of Ceph tolerations and resources

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

3. Specify the parameters in the `hyperconverge` section as required. The
   `hyperconverge` section includes the following parameters:

     - `tolerations` - Specifies tolerations for taint nodes for the defined daemon type.
       Each daemon type key contains the following parameters:
       ```yaml
       spec:
         hyperconverge:
           tolerations:
             <daemonType>:
               rules:
               - key: ""
                 operator: ""
                 value: ""
                 effect: ""
                 tolerationSeconds: 0
       ```

         Possible values for `<daemonType>` are `osd`, `mon`, `mgr`, and `rgw`. The following values are also supported:

         - `all` - specifies general toleration rules for all daemons if no separate daemon rule is specified.
         - `mds` - specifies the CephFS Metadata Server daemons.

        ??? "Example configuration"

            ```yaml
            spec:
              hyperconverge:
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
                  rgw:
                    rules:
                    - effect: NoSchedule
                      key: node-role.kubernetes.io/controlplane
                      operator: Exists
            ```

     - `resources` - Specifies resource requests or limits. The parameter is a map with the
       daemon type as a key and the following structure as a value:
       ```yaml
       spec:
         hyperconverge:
           resources:
             <daemonType>:
               requests: <kubernetes valid spec of daemon resource requests>
               limits: <kubernetes valid spec of daemon resource limits>
       ```

         Possible values for `<daemonType>` are `mon`, `mgr`, `osd`, `osd-hdd`, `osd-ssd`, `osd-nvme`, `prepareosd`,
         `rgw`, and `mds`. The `osd-hdd`, `osd-ssd`, and `osd-nvme` resource requirements handle only the Ceph OSDs
         with a corresponding device class.

        ??? "Example configuration"

            ```yaml
            spec:
              hyperconverge:
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

4. For the Ceph node-specific resource settings, specify the `resources`
   section in the corresponding `nodes` spec of `CephDeployment` CR:
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

5. For the RADOS Gateway instances specific resource settings, specify the
   `resources` section in the `rgw` spec of `CephDeployment` CR:
   ```yaml
   spec:
     objectStorage:
       rgw:
         gateway:
           resources:
             requests: <kubernetes valid spec of daemon resource requests>
             limits: <kubernetes valid spec of daemon resource limits>
   ```

     For example:
     ```yaml
     spec:
       objectStorage:
         rgw:
           gateway:
             resources:
               requests:
                 memory: 1Gi
                 cpu: 2
               limits:
                 memory: 2Gi
                 cpu: 3
     ```

6. Save the reconfigured `CephDeployment` CR and wait for Pelagia Deployment Controller to apply the updated
   Ceph configuration to Rook. Rook will recreate Ceph Monitors, Ceph Managers, or Ceph OSDs according to the
   specified `hyperconverge` configuration.
7. Specify tolerations for different Rook resources using Pelagia Helm chart values. For details, see
   [Specify Rook daemons placement](../rook-daemon-place.md#rook-daemon-place-specify-rook-daemons-placement).
8. After a successful Ceph reconfiguration, unset the flags set in step 1
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
