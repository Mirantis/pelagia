<a id="add-rm-ceph-osd-with-meta"></a>

# Add, remove, or reconfigure Ceph OSDs with metadata devices

Pelagia Lifecycle Management (LCM) Controller simplifies Ceph cluster management
by automating LCM operations. This section describes how to add, remove, or reconfigure Ceph
OSDs with a separate metadata device.

## Add a Ceph OSD with a metadata device <a name="ceph-osd-meta-add"></a>

1. Configure one disk for data and one logical volume for metadata of a Ceph OSD to be added to the Ceph cluster.

    !!! note

         If you add a new disk after node provisioning, manually prepare the required node devices using
         Logical Volume Manager (LVM) 2 on the existing node.

2. Optional. If you want to add a Ceph OSD on top of a **raw** device that already exists
   on a node or is **hot-plugged**, add the required device using the following
   guidelines:

    - You can add a raw device to a node during node deployment.
    - If a node supports adding devices without a node reboot, you can hot plug
      a raw device to a node.
    - If a node does not support adding devices without a node reboot, you can
      hot plug a raw device during node shutdown.

3. Open the `CephDeployment` custom resource (CR) for editing:
   ```bash
   kubectl -n pelagia edit cephdpl
   ```

4. In the `nodes.<nodeName>.devices` section, specify the
   parameters for a Ceph OSD as required. For the parameters description, see
   [CephDeployment: Nodes parameters](https://mirantis.github.io/pelagia/architecture/custom-resources/cephdeployment#nodes).

     The example configuration of the `nodes` section with the new node:
     ```yaml
     nodes:
     - name: storage-worker-505
       roles:
       - mon
       - mgr
       devices:
       - config: # existing item
           deviceClass: hdd
         fullPath: /dev/disk/by-id/scsi-SATA_HGST_HUS724040AL_PN1334PEHN18ZS
       - config: # new item
           deviceClass: hdd
           metadataDevice: /dev/bluedb/meta_1
         fullPath: /dev/disk/by-id/scsi-0ATA_HGST_HUS724040AL_PN1334PEHN1VBC
     ```

    !!! warning

        We highly recommend using the non-wwn `by-id` symlinks to specify storage devices in the `devices` list.
        For details, see [Architecture: Addressing Ceph devices](https://mirantis.github.io/pelagia/architecture/addressing-ceph-devices).

5. Verify that the Ceph OSD is successfully deployed on the specified node. The `CephDeploymentHealth` CR
   `status.healthReport.cephDaemons.cephDaemons` section should not contain any issues:
   ```bash
   kubectl -n pelagia get cephdeploymenthealth -o yaml
   ```

     For example:
     ```yaml
     status:
       healthReport:
         cephDaemons:
           cephDaemons:
             osd:
               info:
               - 4 osds, 4 up, 4 in
               status: ok
     ```

6. Verify the Ceph OSD status:
   ```bash
   kubectl -n rook-ceph get pod -l app=rook-ceph-osd -o wide | grep <nodeName>
   ```

     Substitute `<nodeName>` with the corresponding node name.

     Example of system response:
     ```bash
     rook-ceph-osd-0-7b8d4d58db-f6czn   1/1     Running   0          42h   10.100.91.6   kaas-node-6c5e76f9-c2d2-4b1a-b047-3c299913a4bf   <none>           <none>
     rook-ceph-osd-1-78fbc47dc5-px9n2   1/1     Running   0          21h   10.100.91.6   kaas-node-6c5e76f9-c2d2-4b1a-b047-3c299913a4bf   <none>           <none>
     rook-ceph-osd-3-647f8d6c69-87gxt   1/1     Running   0          21h   10.100.91.6   kaas-node-6c5e76f9-c2d2-4b1a-b047-3c299913a4bf   <none>           <none>
     ```

## Remove a Ceph OSD with a metadata device <a name="ceph-osd-meta-remove"></a>

!!! note

    Ceph OSD removal presupposes usage of a `CephOsdRemoveTask` CR. For workflow overview, see
    [High-level workflow of Ceph OSD or node removal](https://mirantis.github.io/pelagia/ops-guide/lcm/create-task-workflow).

!!! warning

    When using the non-recommended Ceph pools `replicated.size` of
    less than `3`, Ceph OSD removal cannot be performed. The minimal replica
    size equals a rounded up half of the specified `replicated.size`.

    For example, if `replicated.size` is `2`, the minimal replica size is
    `1`, and if `replicated.size` is `3`, then the minimal replica size
    is `2`. The replica size of `1` allows Ceph having PGs with only one
    Ceph OSD in the `acting` state, which may cause a `PG_TOO_DEGRADED`
    health warning that blocks Ceph OSD removal. We recommend setting
    `replicated.size` to `3` for each Ceph pool.

1. Open the `CephDeployment` object for editing:
   ```bash
   kubectl -n pelagia edit cephdpl
   ```

2. Remove the required Ceph OSD specification from the `spec.nodes.<nodeName>.devices` list:

     The example configuration of the `nodes` section with the new node:
     ```yaml
     nodes:
     - name: storage-worker-505
       roles:
       - mon
       - mgr
       storageDevices:
       - config:
           deviceClass: hdd
         fullPath: /dev/disk/by-id/scsi-SATA_HGST_HUS724040AL_PN1334PEHN18ZS
       - config: # remove the entire item entry from devices list
           deviceClass: hdd
           metadataDevice: /dev/bluedb/meta_1
         fullPath: /dev/disk/by-id/scsi-0ATA_HGST_HUS724040AL_PN1334PEHN1VBC
     ```

3. Create a YAML template for the `CephOsdRemoveTask` CR. Select from the following options:

    - Remove Ceph OSD by device name, `by-path` symlink, or `by-id` symlink:
      ```yaml
      apiVersion: lcm.mirantis.com/v1alpha1
      kind: CephOsdRemoveTask
      metadata:
        name: remove-osd-<nodeName>-task
        namespace: pelagia
      spec:
        nodes:
          <nodeName>:
            cleanupByDevice:
            - device: sdb
            - device: sdc
      ```

        !!! warning

            We do not recommend setting device name or device `by-path` symlink in the `cleanupByDevice` field
            as these identifiers are not persistent and can change at node boot. Remove Ceph OSDs with `by-id`
            symlinks or use `cleanupByOsdId` instead. For details, see
            [Architecture: Addressing Ceph devices](https://mirantis.github.io/pelagia/architecture/addressing-ceph-devices).

        !!! note

            If a device was physically removed from a node, `cleanupByDevice` is not supported. Therefore, use
            `cleanupByOsdId` instead.

    - Remove Ceph OSD by OSD ID:
      ```yaml
      apiVersion: lcm.mirantis.com/v1alpha1
      kind: CephOsdRemoveTask
      metadata:
        name: remove-osd-<nodeName>-task
        namespace: pelagia
      spec:
        nodes:
          <nodeName>:
            cleanupByOsdId:
            - id: 5
            - id: 10
      ```

4. Apply the template:
   ```bash
   kubectl apply -f remove-osd-<nodeName>-task.yaml
   ```

5. Verify that the corresponding task has been created:
   ```bash
   kubectl -n pelagia get cephosdremovetask remove-osd-<nodeName>-task
   ```

6. Verify that the `removeInfo` section appeared in the `CephOsdRemoveTask` CR `status`:
   ```bash
   kubectl -n pelagia get cephosdremovetask remove-osd-<nodeName>-task -o yaml
   ```

     Example of system response:
     ```yaml
     status:
       removeInfo:
         cleanupMap:
           storage-worker-505:
             osdMapping:
               "10":
                 deviceMapping:
                   sdb:
                     path: "/dev/disk/by-path/pci-0000:00:1t.9"
                     partition: "/dev/ceph-b-vg_sdb/osd-block-b-lv_sdb"
                     type: "block"
                     class: "hdd"
                     zapDisk: true
               "5":
                 deviceMapping:
                   /dev/sdc:
                     deviceClass: hdd
                     devicePath: /dev/disk/by-path/pci-0000:00:0f.0
                     devicePurpose: block
                     usedPartition: /dev/ceph-2d11bf90-e5be-4655-820c-fb4bdf7dda63/osd-block-e41ce9a8-4925-4d52-aae4-e45167cfcf5c
                     zapDisk: true
                   /dev/sdf:
                     deviceClass: hdd
                     devicePath: /dev/disk/by-path/pci-0000:00:12.0
                     devicePurpose: db
                     usedPartition: /dev/bluedb/meta_1
     ```

7. Verify that the `cleanupMap` section matches the required removal and
   wait for the `ApproveWaiting` phase to appear in `status`:
   ```bash
   kubectl -n pelagia get cephosdremovetask remove-osd-<nodeName>-task -o yaml
   ```

     Example of system response:
     ```yaml
     status:
       phase: ApproveWaiting
     ```

8. In the `CephOsdRemoveTask` CR, set the `approve` flag to `true`:
   ```bash
   kubectl -n pelagia edit cephosdremovetask remove-osd-<nodeName>-task
   ```

     Configuration snippet:
     ```yaml
     spec:
       approve: true
     ```

9. Review the following `status` fields of the Ceph LCM CR processing:

     - `status.phase` - current state of task processing;
     - `status.messages` - description of the current phase;
     - `status.conditions` - full history of task processing before the
       current phase;
     - `status.removeInfo.issues` and `status.removeInfo.warnings` - error
       and warning messages occurred during task processing, if any.

10. Verify that the `CephOsdRemoveTask` has been completed.

      Example of the positive `status.phase` field:
      ```yaml
      status:
        phase: Completed # or CompletedWithWarnings if there are non-critical issues
      ```

11. Remove the device cleanup jobs:
    ```bash
    kubectl delete jobs -n pelagia -l app=pelagia-lcm-cleanup-disks
    ```

## Reconfigure a partition of a Ceph OSD metadata device <a name="ceph-osd-meta-reconfig"></a>

There is no hot reconfiguration procedure for existing Ceph OSDs. To
reconfigure an existing Ceph node, remove and re-add a Ceph OSD with a
metadata device. However, the automated LCM will clean up the logical volume without a removal,
and it can be reused. For this reason, to reconfigure a partition of a Ceph
OSD metadata device:

1. Remove a Ceph OSD from the Ceph cluster as described in
   [Remove a Ceph OSD with a metadata device](#ceph-osd-meta-remove).
2. Add the same Ceph OSD but with a modified configuration as described in
   [Add a Ceph OSD with a metadata device](#ceph-osd-meta-add).
