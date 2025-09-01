<a id="replace-meta"></a>

# Replace a failed metadata device

This section describes the scenario when an underlying metadata device fails
with all related Ceph OSDs. In this case, the only solution is to remove all
Ceph OSDs related to the failed metadata device, then attach a device that
will be used as a new metadata device, and re-create all affected Ceph OSDs.

## Remove failed Ceph OSDs with affected metadata device

1. Save the `CephDeployment` specification of all Ceph OSDs affected by the
   failed metadata device to re-use this specification during re-creation of
   Ceph OSDs after disk replacement.
2. Identify Ceph OSD IDs related to the failed metadata device, for example,
   using Ceph CLI in the `pelagia-ceph-toolbox` Pod:
   ```bash
   ceph osd metadata
   ```

     Example of system response:
     ```json
     {
         "id": 11,
         ...
         "bluefs_db_devices": "vdc",
         ...
         "bluestore_bdev_devices": "vde",
         ...
         "devices": "vdc,vde",
         ...
         "hostname": "kaas-node-6c5e76f9-c2d2-4b1a-b047-3c299913a4bf",
         ...
     },
     {
         "id": 12,
         ...
         "bluefs_db_devices": "vdd",
         ...
         "bluestore_bdev_devices": "vde",
         ...
         "devices": "vdd,vde",
         ...
         "hostname": "kaas-node-6c5e76f9-c2d2-4b1a-b047-3c299913a4bf",
         ...
     },
     {
         "id": 13,
         ...
         "bluefs_db_devices": "vdf",
         ...
         "bluestore_bdev_devices": "vde",
         ...
         "devices": "vde,vdf",
         ...
         "hostname": "kaas-node-6c5e76f9-c2d2-4b1a-b047-3c299913a4bf",
         ...
     },
     ...
     ```

3. Open the `CephDeployment` custom resource (CR) for editing:
   ```bash
   kubectl -n pelagia edit miraceph
   ```

4. In the `nodes` section, remove all `devices` items that relate
   to the failed metadata device. When using a metadata device with device filters, remove the whole node
   section, including the node `name: <nodeName>` field, and perform a complete node cleanup as described in
   [Remove a Ceph node](https://mirantis.github.io/pelagia/ops-guide/lcm/add-rm-ceph-node#ceph-node-remove).

     For example:
     ```yaml
     spec:
       nodes:
       - name: <nodeName>
         devices:
         - name: <deviceName1>  # remove the entire item from the devices list
           # fullPath: <deviceByPath> if device is specified using symlink instead of name
           config:
             deviceClass: hdd
             metadataDevice: <metadataDevice>
         - name: <deviceName2>  # remove the entire item from the devices list
           config:
             deviceClass: hdd
             metadataDevice: <metadataDevice>
         - name: <deviceName3>  # remove the entire item from the devices list
           config:
             deviceClass: hdd
             metadataDevice: <metadataDevice>
         ...
     ```

     In the example above, `<nodeName>` is the node name where the metadata device `<metadataDevice>` must be replaced.

5. Create a `CephOsdRemoveTask` CR template and save it as `replace-failed-meta-<nodeName>-<metadataDevice>-task.yaml`:
   ```yaml
   apiVersion: lcm.mirantis.com/v1alpha1
   kind: CephOsdRemoveTask
   metadata:
     name: replace-failed-meta-<nodeName>-<metadataDevice>
     namespace: pelagia
   spec:
     nodes:
       <nodeName>:
         cleanupByOsdId:
         - id: <osdID-1>
         - id: <osdID-2>
         ...
   ```

     Substitute the following parameters:

     * `<nodeName>` and `<metadataDevice>` with the node and device
       names from the previous step
     * `<osdID-*>` with IDs of the affected Ceph OSDs

6. Apply the template to the cluster:
   ```bash
   kubectl apply -f replace-failed-meta-<nodeName>-<metadataDevice>-task.yaml
   ```

7. Verify that the corresponding request has been created:
   ```bash
   kubectl -n pelagia get cephosdremovetask
   ```

8. Verify that the `removeInfo` section is present in the `CephOsdRemoveTask` CR `status` and that the `cleanupMap`
   section matches the required removal:
   ```bash
   kubectl -n pelagia get cephosdremovetask replace-failed-meta-<nodeName>-<metadataDevice> -o yaml
   ```

     Example of system response:
     ```yaml
     removeInfo:
       cleanupMap:
         <nodeName>:
           osdMapping:
             "<osdID-1>":
               deviceMapping:
                 <dataDevice-1>:
                   deviceClass: hdd
                   devicePath: <dataDeviceByPath-1>
                   devicePurpose: block
                   usedPartition: <dataLvPartition-1>
                   zapDisk: true
                 <metadataDevice>:
                   deviceClass: hdd
                   devicePath: <metadataDeviceByPath>
                   devicePurpose: db
                   usedPartition: /dev/ceph-b0c70c72-8570-4c9d-93e9-51c3ab4dd9f9/osd-db-ecf64b20-1e07-42ac-a8ee-32ba3c0b7e2f
               uuid: ef516477-d2da-492f-8169-a3ebfc3417e2
             "<osdID-2>":
               deviceMapping:
                 <dataDevice-2>:
                   deviceClass: hdd
                   devicePath: <dataDeviceByPath-2>
                   devicePurpose: block
                   usedPartition: <dataLvPartition-2>
                   zapDisk: true
                 <metadataDevice>:
                   deviceClass: hdd
                   devicePath: <metadataDeviceByPath>
                   devicePurpose: db
                   usedPartition: /dev/ceph-b0c70c72-8570-4c9d-93e9-51c3ab4dd9f9/osd-db-ecf64b20-1e07-42ac-a8ee-32ba3c0b7e2f
               uuid: ef516477-d2da-492f-8169-a3ebfc3417e2
             ...
     ```

     Definition of values in angle brackets:

     * `<nodeName>` - underlying node name of the machine, for example,
       `storage-worker-55`
     * `<osdId>` - Ceph OSD ID for the device being replaced, for example,
       `1`
     * `<dataDeviceByPath>` - `by-path` of the device placed on the node,
       for example, `/dev/disk/by-path/pci-0000:00:1t.9`
     * `<dataDevice>` - name of the device placed on the node, for example,
       `/dev/vdc`
     * `<metadataDevice>` - metadata name of the device placed on the node,
       for example, `/dev/vde`
     * `<metadataDeviceByPath>` - metadata `by-path` of the device placed
       on the node, for example, `/dev/disk/by-path/pci-0000:00:12.0`
     * `<dataLvPartition>` - logical volume partition of the data device

9. Wait for the `ApproveWaiting` phase to appear in `status`:
   ```bash
   kubectl -n pelagia get cephosdremovetask replace-failed-meta-<nodeName>-<metadataDevice> -o yaml
   ```

     Example of system response:
     ```yaml
     status:
       phase: ApproveWaiting
     ```

10. In the `CephOsdRemoveTask` CR, set the `approve` flag to `true`:
    ```bash
    kubectl -n pelagia edit cephosdremovetask replace-failed-meta-<nodeName>-<metadataDevice>
    ```

      Configuration snippet:
      ```yaml
      spec:
        approve: true
      ```

11. Review the following `status` fields of the Ceph LCM CR processing:

      - `status.phase` - current state of task processing;
      - `status.messages` - description of the current phase;
      - `status.conditions` - full history of task processing before the
        current phase;
      - `status.removeInfo.issues` and `status.removeInfo.warnings` - error
        and warning messages occurred during task processing, if any.

12. Verify that the `CephOsdRemoveTask` has been completed. For example:
    ```yaml
    status:
      phase: Completed # or CompletedWithWarnings if there are non-critical issues
    ```

## Prepare the replaced metadata device for Ceph OSD re-creation

!!! note

    This section describes how to create a metadata disk partition
    on N logical volumes. To create one partition on a metadata disk, refer to
    [Re-create the partition on the existing metadata disk](https://mirantis.github.io/pelagia/ops-guide/lcm/replace-osd-meta-device#recreate-meta-lvm).

1. Partition the replaced metadata device by N logical volumes (LVs), where N
   is the number of Ceph OSDs previously located on a failed metadata device.

     Calculate the new metadata LV percentage of used volume group capacity using the `100 / N` formula.

2. Log in to the node with the replaced metadata disk.
3. Create an LVM physical volume atop the replaced metadata device:
   ```bash
   pvcreate <metadataDisk>
   ```

     Substitute `<metadataDisk>` with the replaced metadata device.

4. Create an LVM volume group atop of the physical volume:
   ```bash
   vgcreate bluedb <metadataDisk>
   ```

     Substitute `<metadataDisk>` with the replaced metadata device.

5. Create N LVM logical volumes with the calculated capacity per each volume:
   ```bash
   lvcreate -l <X>%VG -n meta_<i> bluedb
   ```

     Substitute `<X>` with the result of the `100 / N` formula and `<i>`
     with the current number of metadata partitions.

As a result, the replaced metadata device will have N LVM paths, for example,
`/dev/bluedb/meta_1`.

## Re-create a Ceph OSD on the replaced metadata device

!!! note

    You can spawn Ceph OSD on a raw device, but it must be clean and
    without any data or partitions. If you want to add a device that was in use,
    also ensure it is raw and clean. To clean up all data and partitions from a
    device, refer to official [Rook documentation](https://github.com/rook/rook/blob/master/Documentation/Storage-Configuration/ceph-teardown.md#zapping-devices).

1. Open the `CephDeployment` CR for editing:
   ```bash
   kubectl -n pelagia edit cephdpl
   ```

2. In the `nodes` section, add the cleaned Ceph OSD device with the replaced
   LVM paths of the metadata device from previous steps. For example:
   ```yaml
   spec:
     nodes:
     - name: <nodeName>
       devices:
       - fullPath: <deviceByID-1> # Recommended. Add the new device by ID /dev/disk/by-id/...
         #name: <deviceByID-1> # Not recommended. Add a new device by ID, for example, /dev/disk/by-id/...
         #fullPath: <deviceByPath-1> # Not recommended. Add a new device by path /dev/disk/by-path/...
         config:
           deviceClass: hdd
           metadataDevice: /dev/<vgName>/<lvName-1>
       - fullPath: <deviceByID-2> # Recommended. Add the new device by ID /dev/disk/by-id/...
         #name: <deviceByID-2> # Not recommended. Add a new device by ID, for example, /dev/disk/by-id/...
         #fullPath: <deviceByPath-2> # Not recommended. Add a new device by path /dev/disk/by-path/...
         config:
           deviceClass: hdd
           metadataDevice: /dev/<vgName>/<lvName-2>
       - fullPath: <deviceByID-3> # Recommended. Add the new device by ID /dev/disk/by-id/...
         #name: <deviceByID-3> # Not recommended. Add a new device by ID, for example, /dev/disk/by-id/...
         #fullPath: <deviceByPath-3> # Not recommended. Add a new device by path /dev/disk/by-path/...
         config:
           deviceClass: hdd
           metadataDevice: /dev/<vgName>/<lvName-3>
   ```

     - Substitute `<nodeName>` with the node name where the
       metadata device has been replaced.
     - Add all data devices for re-created Ceph OSDs and specify
       `metadataDevice` that is the path to the previously created logical
       volume. Substitute `<vgName>` with a volume group name that contains N
       logical volumes `<lvName-i>`.

3. Wait for the re-created Ceph OSDs to apply to the Ceph cluster. You can monitor the application state using
   either the `status` section of the `CephDeploymentHealth` CR or in the `pelagia-ceph-toolbox` Pod:
   ```bash
   kubectl -n pelagia get cephdeploymenthealth -o yaml
   kubectl -n rook-ceph exec -it deploy/pelagia-ceph-toolbox -- ceph -s
   ```
