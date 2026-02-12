<a id="add-rm-ceph-node"></a>

# Add, remove, or reconfigure Ceph nodes

Pelagia Lifecycle Management (LCM) Controller simplifies Ceph cluster management
by automating LCM operations. This section describes how to add, remove, or reconfigure Ceph
nodes.

!!! note

    When adding a Ceph node with the Ceph Monitor role, if any issues occur with
    the Ceph Monitor, `rook-ceph` removes it and adds a new Ceph Monitor instead,
    named using the next alphabetic character in order. Therefore, the Ceph Monitor
    names may not follow the alphabetical order. For example, `a`, `b`, `d`,
    instead of `a`, `b`, `c`.

## Add a Ceph node <a name="ceph-node-add"></a>

1. Prepare a new node for the cluster.
2. Open the `CephDeployment` custom resource (CR) for editing:
   ```bash
   kubectl -n pelagia edit cephdpl
   ```

3. In the `nodes` section, specify the parameters for a Ceph node as
   required. For the parameter description, see
   [CephDeployment: Nodes parameters](https://mirantis.github.io/pelagia/architecture/custom-resources/cephdeployment#nodes).

     The example configuration of the `nodes` section with the new node:
     ```yaml
     nodes:
     - name: storage-worker-414
       roles:
       - mon
       - mgr
       devices:
       - config:
           deviceClass: hdd
         fullPath: /dev/disk/by-id/scsi-SATA_HGST_HUS724040AL_PN1334PEHN18ZS
     ```

     You can also add a new node with device filters. For example:
     ```yaml
     nodes:
     - name: storage-worker-414
       roles:
       - mon
       - mgr
       config:
        deviceClass: hdd
       devicePathFilter: "^/dev/disk/by-id/scsi-SATA_HGST+*"
     ```

    !!! warning

        We highly recommend using the non-wwn `by-id` symlinks to specify storage devices in the `devices` list.
        For details, see [Architecture: Addressing Ceph devices](https://mirantis.github.io/pelagia/architecture/addressing-ceph-devices).

    !!! note

        - To use a new Ceph node for a Ceph Monitor or Ceph Manager deployment,
          also specify the `roles` parameter.
        - Reducing the number of Ceph Monitors is not supported and causes the
          Ceph Monitor daemons removal from random nodes.
        - Removal of the `mgr` role in the `nodes` section of the
          `CephDeployment` CR does not remove Ceph Managers. To remove a Ceph
          Manager from a node, remove it from the `nodes` spec and manually
          delete the `mgr` pod in the Rook namespace.

4. Verify that all new Ceph daemons for the specified node have been
   successfully deployed in the Ceph cluster. The `CephDeploymentHealth` CR
   `status.healthReport.cephDaemons.cephDaemons` should not contain any issues.
   ```bash
   kubectl -n pelagia get cephdeploymenthealth -o yaml
   ```

     Example of system response:
     ```yaml
     status:
       healthReport:
         cephDaemons:
           cephDaemons:
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
     ```

## Remove a Ceph node <a name="ceph-node-remove"></a>

!!! note

    Ceph node removal presupposes usage of a `CephOsdRemoveTask` CR. For workflow overview, see
    [High-level workflow of Ceph OSD or node removal](https://mirantis.github.io/pelagia/ops-guide/lcm/create-task-workflow).

!!! note

    To remove a Ceph node with a `mon` role, first move the Ceph
    Monitor to another node and remove the `mon` role from the Ceph node as
    described in
    [Move a Ceph Monitor daemon to another node](https://mirantis.github.io/pelagia/ops-guide/deployment/move-mon-daemon).

1. Open the `CephDeployment` CR for editing:
   ```bash
   kubectl -n pelagia edit cephdpl
   ```

2. In the `nodes` section, remove the required Ceph node specification.

     For example:
     ```yaml
     spec:
         nodes:
         - name: storage-worker-5 # remove the entire entry for the required node
           devices: {...}
           roles: [...]
     ```

3. Create a YAML template for the `CephOsdRemoveTask` CR. For example:
   ```yaml
   apiVersion: lcm.mirantis.com/v1alpha1
   kind: CephOsdRemoveTask
   metadata:
     name: remove-osd-worker-5
     namespace: pelagia
   spec:
     nodes:
       storage-worker-5:
         completeCleanUp: true
   ```

4. Apply the template on the Rockoon cluster:
   ```bash
   kubectl apply -f remove-osd-worker-5.yaml
   ```

5. Verify that the corresponding request has been created:
   ```bash
   kubectl -n pelagia get cephosdremovetask remove-osd-worker-5
   ```

6. Verify that the `removeInfo` section appeared in the `CephOsdRemoveTask` CR `status`:
   ```bash
   kubectl -n pelagia get cephosdremovetask remove-osd-worker-5 -o yaml
   ```

     Example of system response:

     ```yaml
     status:
       removeInfo:
         cleanupMap:
           storage-worker-5:
             osdMapping:
               "10":
                 deviceMapping:
                   sdb:
                     path: "/dev/disk/by-path/pci-0000:00:1t.9"
                     partition: "/dev/ceph-b-vg_sdb/osd-block-b-lv_sdb"
                     type: "block"
                     class: "hdd"
                     zapDisk: true
               "16":
                 deviceMapping:
                   sdc:
                     path: "/dev/disk/by-path/pci-0000:00:1t.10"
                     partition: "/dev/ceph-b-vg_sdb/osd-block-b-lv_sdc"
                     type: "block"
                     class: "hdd"
                     zapDisk: true
     ```

7. Verify that the `cleanupMap` section matches the required removal and wait
   for the `ApproveWaiting` phase to appear in `status`:
   ```bash
   kubectl -n pelagia get cephosdremovetask remove-osd-worker-5 -o yaml
   ```

     Example of system response:
     ```yaml
     status:
       phase: ApproveWaiting
     ```

8. Edit the `CephOsdRemoveTask` CR and set the `approve` flag to `true`:
   ```bash
   kubectl -n pelagia edit cephosdremovetask remove-osd-worker-5
   ```

     For example:
     ```yaml
     spec:
       approve: true
     ```

9. Review the status of the `CephOsdRemoveTask` resource
   processing. The valuable parameters are as follows:

     - `status.phase` - the current state of task processing
     - `status.messages` - the description of the current phase
     - `status.conditions` - full history of task processing before the
       current phase
     - `status.removeInfo.issues` and `status.removeInfo.warnings` - contain
       error and warning messages occurred during task processing

10. Verify that the `CephOsdRemoveTask` has been completed. For example:
    ```yaml
    status:
      phase: Completed # or CompletedWithWarnings if there are non-critical issues
    ```

11. Remove the device cleanup jobs:
    ```bash
    kubectl delete jobs -n pelagia -l app=pelagia-lcm-cleanup-disks
    ```

## Reconfigure a Ceph node

There is no hot reconfiguration procedure for existing Ceph OSDs and Ceph Monitors. To reconfigure an existing Ceph node:

1. Remove the Ceph node from the Ceph cluster.
2. Add the same Ceph node but with a modified configuration.
