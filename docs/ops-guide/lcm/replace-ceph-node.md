<a id="replace-ceph-node"></a>

# Replace a failed Ceph node

After a physical node replacement, you can use the Pelagia LCM API to redeploy
failed Ceph nodes. The common flow of replacing a failed Ceph node is as
follows:

1. Remove the obsolete Ceph node from the Ceph cluster.
2. Add a new Ceph node with the same configuration to the Ceph cluster.

Ceph OSD removal presupposes usage of a `CephOsdRemoveTask` CR. For workflow overview, see [Creating a Ceph OSD remove task](../../ops-guide/lcm/create-task-workflow.md#create-osd-rm-request).

<a name="replace-ceph-node-remove"></a>
## Remove a failed Ceph node

1. Open the `CephDeployment` CR for editing:
   ```bash
   kubectl -n pelagia edit cephdpl
   ```

2. In the `nodes` section, remove the required device. When using device
   filters, update regexp accordingly.

     For example:
     ```yaml
     spec:
       nodes:
       - name: <nodeName> # remove the entire entry for the node to replace
         devices: {...}
         role: [...]
     ```

     Substitute `<nodeName>` with the node name to replace.

3. Save `CephDeployment` changes and close the editor.
4. Create a `CephOsdRemoveTask` CR template and save it as `replace-failed-<nodeName>-task.yaml`:
   ```yaml
   apiVersion: lcm.mirantis.com/v1alpha1
   kind: CephOsdRemoveTask
   metadata:
     name: replace-failed-<nodeName>-task
     namespace: pelagia
   spec:
     nodes:
       <nodeName>:
         completeCleanUp: true
   ```

5. Apply the template to the cluster:
   ```bash
   kubectl apply -f replace-failed-<nodeName>-task.yaml
   ```

6. Verify that the corresponding task has been created:
   ```bash
   kubectl -n pelagia get cephosdremovetask
   ```

7. Verify that the `removeInfo` section appeared in the `CephOsdRemoveTask` CR `status`:
   ```bash
   kubectl -n pelagia get cephosdremovetask replace-failed-<nodeName>-task -o yaml
   ```

     Example of system response:
     ```yaml
     removeInfo:
       cleanupMap:
         <nodeName>:
           osdMapping:
             ...
             <osdId>:
               deviceMapping:
                 ...
                 <deviceName>:
                   path: <deviceByPath>
                   partition: "/dev/ceph-b-vg_sdb/osd-block-b-lv_sdb"
                   type: "block"
                   class: "hdd"
                   zapDisk: true
     ```

     Definition of values in angle brackets:

     * `<nodeName>` - underlying machine node name, for example,
       `storage-worker-5`.
     * `<osdId>` - actual Ceph OSD ID for the device being replaced, for
       example, `1`.
     * `<deviceName>` - actual device name placed on the node, for
       example, `sdb`.
     * `<deviceByPath>` - actual device `by-path` placed on the node, for
       example, `/dev/disk/by-path/pci-0000:00:1t.9`.

8. Verify that the `cleanupMap` section matches the required removal and wait
   for the `ApproveWaiting` phase to appear in `status`:
   ```bash
   kubectl -n pelagia get cephosdremovetask replace-failed-<nodeName>-task -o yaml
   ```

     Example of system response:
     ```yaml
     status:
       phase: ApproveWaiting
     ```

9. Edit the `CephOsdRemoveTask` CR and set the `approve` flag to `true`:
   ```bash
   kubectl -n pelagia edit cephosdremovetask replace-failed-<nodeName>-task
   ```

     For example:
     ```yaml
     spec:
       approve: true
     ```

10. Review the following `status` fields of the Ceph LCM CR processing:

      - `status.phase` - current state of task processing;
      - `status.messages` - description of the current phase;
      - `status.conditions` - full history of task processing before the
        current phase;
      - `status.removeInfo.issues` and `status.removeInfo.warnings` - error
        and warning messages occurred during task processing, if any.

11. Verify that the `CephOsdRemoveTask` has been completed.
    For example:
    ```yaml
    status:
      phase: Completed # or CompletedWithWarnings if there are non-critical issues
    ```

12. Remove the device cleanup jobs:
    ```bash
    kubectl delete jobs -n pelagia -l app=pelagia-lcm-cleanup-disks
    ```

<a name="replace-ceph-node-add"></a>
## Deploy a new Ceph node after removal of a failed one

!!! note

    You can spawn Ceph OSD on a raw device, but it must be clean and
    without any data or partitions. If you want to add a device that was in use,
    also ensure it is raw and clean. To clean up all data and partitions from a
    device, refer to official [Rook documentation](https://github.com/rook/rook/blob/master/Documentation/Storage-Configuration/ceph-teardown.md#zapping-devices).

1. Open the `CephDeployment` CR for editing:
   ```bash
   kubectl -n pelagia edit cephdpl
   ```

2. In the `nodes` section, add a new device:
   ```yaml
   spec:
     nodes:
     - name: <nodeName> # add new configuration for replaced Ceph node
       devices:
       - fullPath: <deviceByID> # Recommended. Non-wwn by-id symlink.
         # name: <deviceByID> # Not recommended. If a device is supposed to be added with by-id.
         # fullPath: <deviceByPath> # if device is supposed to be added with by-path.
         config:
           deviceClass: hdd
         ...
   ```

     Substitute `<nodeName>` with the replaced node name and configure it as required.

    !!! warning

        We highly recommend using the non-wwn `by-id` symlinks to specify storage devices in the `devices` list.
        For details, see [Addressing Ceph storage devices](../../architecture/addressing-ceph-devices.md#addressing-ceph-storage-devices).

3. Verify that all Ceph daemons from the replaced node have appeared on the
   Ceph cluster and are `in` and `up`. The `healthReport` section of `CephDeploymentHealth` CR
   should not contain any issues.
   ```bash
   kubectl -n pelagia get cephdeploymenthealth -o yaml
   ```

     Example of system response:
     ```yaml
     status:
       healthReport:
         rookCephObjects:
           cephCluster:
             ceph:
               health: HEALTH_OK
               ...
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

4. Verify the Ceph node in the Rook namespace:
   ```bash
   kubectl -n rook-ceph get pod -o wide | grep <nodeName>
   ```
