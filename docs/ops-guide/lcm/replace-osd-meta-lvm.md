<a id="replace-osd-meta-lvm-replace-a-failed-ceph-osd-with-a-metadata-device-as-a-logical-volume-path"></a>

# Replace a failed Ceph OSD with a metadata device as a logical volume path

You can apply the below procedure in the following cases:

* A Ceph OSD failed without a data or metadata device outage. In this case,
  first remove a failed Ceph OSD and clean up all corresponding disks and
  partitions. Then add a new Ceph OSD to the same data and metadata paths.
* A Ceph OSD failed with data or metadata device outage. In this case, you
  also first remove a failed Ceph OSD and clean up all corresponding disks and
  partitions. Then add a new Ceph OSD to a newly replaced data device with the
  same metadata path.

!!! note

    The below procedure also applies to manually created metadata partitions.

<a name="replace-osd-meta-lvm-remove-a-failed-ceph-osd-by-id-with-a-defined-metadata-device"></a>
## Remove a failed Ceph OSD by ID with a defined metadata device

1. Identify the ID of Ceph OSD related to a failed device. For example, use
   the Ceph CLI in the `pelagia-ceph-toolbox` Pod:
   ```bash
   ceph osd metadata
   ```

     Example of system response:
     ```json
     {
         "id": 0,
         ...
         "bluestore_bdev_devices": "vdc",
         ...
         "devices": "vdc",
         ...
         "hostname": "kaas-node-6c5e76f9-c2d2-4b1a-b047-3c299913a4bf",
         ...
         "pod_name": "rook-ceph-osd-0-7b8d4d58db-f6czn",
         ...
     },
     {
         "id": 1,
         ...
         "bluefs_db_devices": "vdf",
         ...
         "bluestore_bdev_devices": "vde",
         ...
         "devices": "vde,vdf",
         ...
         "hostname": "kaas-node-6c5e76f9-c2d2-4b1a-b047-3c299913a4bf",
         ...
         "pod_name": "rook-ceph-osd-1-78fbc47dc5-px9n2",
         ...
     },
     ...
     ```

2. Open the `CephDeployment` custom resource (CR) for editing:
   ```bash
   kubectl -n pelagia edit cephdpl
   ```

3. In the `nodes` section:

     1. Find and capture the `metadataDevice` path to reuse it during re-creation of the Ceph OSD.
     2. Remove the required device. Example configuration snippet:
        ```yaml
        spec:
          nodes:
          - name: <nodeName>
            devices:
            - name: <deviceName>  # remove the entire item from the devices list
              # fullPath: <deviceByPath> if device is specified using by-path instead of name
              config:
                deviceClass: hdd
                metadataDevice: /dev/bluedb/meta_1
        ```

          In the example above, `<nodeName>` is the name of node on which
          the device `<deviceName>` or `<deviceByPath>` must be replaced.

4. Create a `CephOsdRemoveTask` CR template and save it as `replace-failed-osd-<nodeName>-<osdID>-task.yaml`:
   ```yaml
   apiVersion: lcm.mirantis.com/v1alpha1
   kind: CephOsdRemoveTask
   metadata:
     name: replace-failed-osd-<nodeName>-<deviceName>
     namespace: pelagia
   spec:
     nodes:
       <nodeName>:
         cleanupByOsdId:
         - id: <osdID>
   ```

     Substitute the following parameters:
     - `<nodeName>` and `<deviceName>` with the node and device names
       from the previous step;
     - `<osdID>` with the ID of the affected Ceph OSD.

5. Apply the template to the cluster:
   ```bash
   kubectl apply -f replace-failed-osd-<nodeName>-<osdID>-task.yaml
   ```

6. Verify that the corresponding task has been created:
   ```bash
   kubectl -n pelagia get cephosdremovetask
   ```

7. Verify that the `status` section of `CephOsdRemoveTask` contains
   the `removeInfo` section:
   ```bash
   kubectl -n pelagia get cephosdremovetask replace-failed-osd-<nodeName>-<osdID> -o yaml
   ```

     Example of system response:
     ```yaml
     removeInfo:
       cleanupMap:
         <nodeName>:
           osdMapping:
             "<osdID>":
               deviceMapping:
                 <dataDevice>:
                   deviceClass: hdd
                   devicePath: <dataDeviceByPath>
                   devicePurpose: block
                   usedPartition: /dev/ceph-d2d3a759-2c22-4304-b890-a2d87e056bd4/osd-block-ef516477-d2da-492f-8169-a3ebfc3417e2
                   zapDisk: true
                 <metadataDevice>:
                   deviceClass: hdd
                   devicePath: <metadataDeviceByPath>
                   devicePurpose: db
                   usedPartition: /dev/bluedb/meta_1
               uuid: ef516477-d2da-492f-8169-a3ebfc3417e2
     ```

     Definition of values in angle brackets:

     - `<nodeName>` - underlying node name of the machine, for example,
       `storage-worker-3`
     - `<osdId>` - Ceph OSD ID for the device being replaced, for example,
       `1`
     - `<dataDeviceByPath>` - `by-path` of the device placed on the node,
       for example, `/dev/disk/by-path/pci-0000:00:1t.9`
     - `<dataDevice>` - name of the device placed on the node, for example,
       `/dev/vde`
     - `<metadataDevice>` - metadata name of the device placed on the node,
       for example, `/dev/vdf`
     - `<metadataDeviceByPath>` - metadata `by-path` of the device placed
       on the node, for example, `/dev/disk/by-path/pci-0000:00:12.0`

8. Verify that the `cleanupMap` section matches the required removal and
   wait for the `ApproveWaiting` phase to appear in `status`:
   ```bash
   kubectl -n pelagia get cephosdremovetask replace-failed-osd-<nodeName>-<osdID> -o yaml
   ```

     Example of system response:
     ```yaml
     status:
       phase: ApproveWaiting
     ```

9. In the `CephOsdRemoveTask` CR, set the `approve` flag to `true`:
   ```bash
   kubectl -n pelagia edit cephosdremovetask replace-failed-osd-<nodeName>-<osdID>
   ```

     Configuration snippet:
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

11. Verify that the `CephOsdRemoveTask` has been completed. For example:
    ```yaml
    status:
      phase: Completed # or CompletedWithWarnings if there are non-critical issues
    ```

<a id="replace-osd-meta-lvm-re-create-a-ceph-osd-with-the-same-metadata-partition"></a>

## Re-create a Ceph OSD with the same metadata partition

!!! note

    You can spawn Ceph OSD on a raw device, but it must be clean and
    without any data or partitions. If you want to add a device that was in use,
    also ensure it is raw and clean. To clean up all data and partitions from a
    device, refer to official [Rook documentation](https://github.com/rook/rook/blob/master/Documentation/Storage-Configuration/ceph-teardown.md#zapping-devices).

1. Optional. If you want to add a Ceph OSD on top of a **raw** device that already exists
   on a node or is **hot-plugged**, add the required device using the following
   guidelines:

    - You can add a raw device to a node during node deployment.
    - If a node supports adding devices without a node reboot, you can hot plug
      a raw device to a node.
    - If a node does not support adding devices without a node reboot, you can
      hot plug a raw device during node shutdown.

2. Open the `CephDeployment` CR for editing:
   ```bash
   kubectl -n pelagia edit cephdpl
   ```

3. In the `nodes` section, add the replaced device with the same
   `metadataDevice` path as on the removed Ceph OSD. For example:
   ```yaml
   spec:
     nodes:
     - name: <nodeName>
       devices:
       - fullPath: <deviceByID> # Recommended. Add a new device by-id symlink, for example, /dev/disk/by-id/...
         #name: <deviceByID> # Not recommended. Add a new device by ID, for example, /dev/disk/by-id/...
         #fullPath: <deviceByPath> # Not recommended. Add a new device by path, for example, /dev/disk/by-path/...
         config:
           deviceClass: hdd
           metadataDevice: /dev/bluedb/meta_1 # Must match the value of the previously removed OSD
   ```

     Substitute `<nodeName>` with the node name where the new device `<deviceByID>` or `<deviceByPath>` must be added.

4. Wait for the replaced disk to apply to the Ceph cluster as a new Ceph OSD. You can monitor the application
   state using either the `status` section of the `CephDeploymentHealth` CR or in the `pelagia-ceph-toolbox` Pod:
   ```bash
   kubectl -n pelagia get cephdeploymenthealth -o yaml
   kubectl -n rook-ceph exec -it deploy/pelagia-ceph-toolbox -- ceph -s
   ```
