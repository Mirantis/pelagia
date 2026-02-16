<a id="replace-osd"></a>

# Replace a failed Ceph OSD

After a physical disk replacement, you can use Pelagia Lifecycle Management (LCM) API to redeploy
a failed Ceph OSD. The common flow of replacing a failed Ceph OSD is as
follows:

1. Remove the obsolete Ceph OSD from the Ceph cluster by device symlink, by device name, or by Ceph
   OSD ID.
2. Add a new Ceph OSD on the new disk to the Ceph cluster.

Ceph OSD removal presupposes usage of a `CephOsdRemoveTask` CR. For workflow overview, see [Creating a Ceph OSD remove task](../../ops-guide/lcm/create-task-workflow.md#create-osd-rm-request).

## Remove a failed Ceph OSD by device name, path, or ID <a name="replace-by-device"></a>

!!! warning

    The procedure below presupposes that the cloud operator knows the exact
    device name, `by-path`, or `by-id` of the replaced device, as well as on
    which node the replacement occurred.

!!! warning

    A Ceph OSD removal using `by-path`, `by-id`, or device name is
    not supported if a device was physically removed from a node. Therefore, use
    `cleanupByOsdId` instead.

!!! warning

     We do not recommend setting device name or device `by-path` symlink in the `cleanupByDevice` field
     as these identifiers are not persistent and can change at node boot. Remove Ceph OSDs with `by-id`
     symlinks or use `cleanupByOsdId` instead. For details, see
     [Addressing Ceph storage devices](../../architecture/addressing-ceph-devices.md#addressing-ceph-storage-devices).

1. Open the `CephDeployment` CR for editing:
   ```bash
   kubectl -n pelagia edit cephdpl
   ```

2. In the `nodes` section, remove the required device from `devices`. When
   using device filters, update the `deviceFilter` or `devicePathFilter`
   regexp accordingly.

     For example:
     ```yaml
     spec:
       nodes:
       - name: <nodeName>
         devices:
         - name: <deviceName>  # remove the entire item from devices list
           # fullPath: <deviceByPath> if device is specified with symlink instead of name
           config:
             deviceClass: hdd
     ```

     Substitute `<nodeName>` with the node name where the device
     `<deviceName>` or `<deviceByPath>` is going to be replaced.

3. Save `CephDeployment` changes and close the editor.

4. Create a `CephOsdRemoveTask` CR template and save it as `replace-failed-osd-<nodeName>-<deviceName>-task.yaml`:
   ```yaml
   apiVersion: lcm.mirantis.com/v1alpha1
   kind: CephOsdRemoveTask
   metadata:
     name: replace-failed-osd-<nodeName>-<deviceName>
     namespace: pelagia
   spec:
     nodes:
       <nodeName>:
         cleanupByDevice:
         - name: <deviceName>
           # If a device is specified with by-path or by-id instead of
           # name, path: <deviceByPath> or <deviceById>.
   ```

5. Apply the template to the cluster:
   ```bash
   kubectl apply -f replace-failed-osd-<nodeName>-<deviceName>-task.yaml
   ```

6. Verify that the corresponding request has been created:
   ```bash
   kubectl -n pelagia get cephosdremovetask
   ```

7. Verify that the `removeInfo` section appeared in the `CephOsdRemoveTask` CR `status`:
   ```bash
   kubectl -n pelagia get cephosdremovetask replace-failed-osd-<nodeName>-<deviceName> -o yaml
   ```

     Example of system response:
     ```yaml
     status:
       osdRemoveStatus:
         removeInfo:
           cleanupMap:
             <nodeName>:
               osdMapping:
                 <osdId>:
                   deviceMapping:
                     <dataDevice>:
                       deviceClass: hdd
                       devicePath: <dataDeviceByPath>
                       devicePurpose: block
                       usedPartition: /dev/ceph-d2d3a759-2c22-4304-b890-a2d87e056bd4/osd-block-ef516477-d2da-492f-8169-a3ebfc3417e2
                       zapDisk: true
     ```

     Definition of values in angle brackets:

     - `<nodeName>` - underlying node name of the machine, for example,
       `storage-worker-52`;
     - `<osdId>` - Ceph OSD ID for the device being replaced, for example,
       `1`;
     - `<dataDeviceByPath>` - `by-path` of the device placed on the node,
       for example, `/dev/disk/by-path/pci-0000:00:1t.9`;
     - `<dataDevice>` - name of the device placed on the node, for example,
       `/dev/sdb`.

8. Verify that the `cleanupMap` section matches the required removal and wait
   for the `ApproveWaiting` phase to appear in `status`:
   ```bash
   kubectl -n pelagia get cephosdremovetask replace-failed-osd-<nodeName>-<deviceName> -o yaml
   ```

     Example of system response:
     ```yaml
     status:
       phase: ApproveWaiting
     ```

9. Edit the `CephOsdRemoveTask` CR and set the `approve` flag to `true`:
   ```bash
   kubectl -n pelagia edit cephosdremovetask replace-failed-osd-<nodeName>-<deviceName>
   ```

     For example:
     ```yaml
     spec:
       approve: true
     ```

10. Review the following `status` fields of the Ceph LCM CR processing:

    - `status.phase` - current state of request processing;
    - `status.messages` - description of the current phase;
    - `status.conditions` - full history of request processing before the
      current phase;
    - `status.removeInfo.issues` and `status.removeInfo.warnings` - error
      and warning messages occurred during request processing, if any.

11. Verify that the `CephOsdRemoveTask` has been completed. For example:
    ```yaml
    status:
      phase: Completed # or CompletedWithWarnings if there are non-critical issues
    ```

12. Remove the device cleanup jobs:
    ```bash
    kubectl delete jobs -n pelagia -l app=pelagia-lcm-cleanup-disks
    ```

## Remove a failed Ceph OSD by Ceph OSD ID <a name="replace-by-osd-id"></a>

1. Identify the node and device names used by the affected Ceph OSD. Using the
   Ceph CLI in the `pelagia-ceph-toolbox` Pod, run:
   ```bash
   kubectl -n rook-ceph exec -it deploy/pelagia-ceph-toolbox -- ceph osd metadata <osdId>
   ```

     Substitute `<osdId>` with the affected OSD ID.

     Example output:
     ```json
     {
       "id": 1,
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
     ```

     In the example above, `hostname` is the node name and `devices` are
     all devices used by the affected Ceph OSD.

2. Open the `CephDeployment` CR for editing:
   ```bash
   kubectl -n pelagia edit cephdpl
   ```

3. In the `nodes` section, remove the required device:
   ```yaml
   spec:
     nodes:
     - name: <nodeName>
       devices:
       - name: <deviceName>  # remove the entire item from devices list
         config:
           deviceClass: hdd
   ```

     Substitute `<nodeName>` with the node name where the device `<deviceName>` is going to be replaced.

4. Save `CephDeployment` changes and close the editor.

5. Create a `CephOsdRemoveTask` CR template and save it as `replace-failed-<nodeName>-osd-<osdId>-task.yaml`:
   ```yaml
   apiVersion: lcm.mirantis.com/v1alpha1
   kind: CephOsdRemoveTask
   metadata:
     name: replace-failed-<nodeName>-osd-<osdId>
     namespace: pelagia
   spec:
     nodes:
       <nodeName>:
         cleanupByOsdId:
         - id: <osdId>
   ```

6. Apply the template to the cluster:
   ```bash
   kubectl apply -f replace-failed-<nodeName>-osd-<osdId>-task.yaml
   ```

7. Verify that the corresponding request has been created:
   ```bash
   kubectl -n pelagia get cephosdremovetask
   ```

8. Verify that the `removeInfo` section appeared in the `CephOsdRemoveTask` CR `status`:
   ```bash
   kubectl -n pelagia get cephosdremovetask replace-failed-<nodeName>-osd-<osdId>-task -o yaml
   ```

     Example of system response
     ```yaml
     status:
       osdRemoveStatus:
         removeInfo:
           cleanupMap:
             <nodeName>:
               osdMapping:
                 <osdId>:
                   deviceMapping:
                     <dataDevice>:
                       deviceClass: hdd
                       devicePath: <dataDeviceByPath>
                       devicePurpose: block
                       usedPartition: /dev/ceph-d2d3a759-2c22-4304-b890-a2d87e056bd4/osd-block-ef516477-d2da-492f-8169-a3ebfc3417e2
                       zapDisk: true
     ```

     Definition of values in angle brackets:

     - `<nodeName>` - underlying node name of the machine, for example,
       `storage-worker-52`;
     - `<osdId>` - Ceph OSD ID for the device being replaced, for example,
       `1`;
     - `<dataDeviceByPath>` - `by-path` of the device placed on the node,
       for example, `/dev/disk/by-path/pci-0000:00:1t.9`;
     - `<dataDevice>` - name of the device placed on the node, for example,
       `/dev/sdb`.

9. Verify that the `cleanupMap` section matches the required removal and wait
   for the `ApproveWaiting` phase to appear in `status`:
   ```bash
   kubectl -n pelagia get cephosdremovetask replace-failed-<nodeName>-osd-<osdId>-task -o yaml
   ```

     Example of system response:
     ```yaml
     status:
       phase: ApproveWaiting
     ```

10. Edit the `CephOsdRemoveTask` CR and set the `approve` flag to `true`:
    ```bash
    kubectl -n pelagia edit cephosdremovetask replace-failed-<nodeName>-osd-<osdId>-request
    ```

      For example:
      ```yaml
      spec:
        approve: true
      ```

11. Review the following `status` fields of the Ceph LCM CR processing:

    - `status.phase` - current state of request processing;
    - `status.messages` - description of the current phase;
    - `status.conditions` - full history of request processing before the
      current phase;
    - `status.removeInfo.issues` and `status.removeInfo.warnings` - error
      and warning messages occurred during request processing, if any.

12. Verify that the `CephOsdRemoveTask` has been completed. For example:
    ```yaml
    status:
      phase: Completed # or CompletedWithWarnings if there are non-critical issues
    ```

13. Remove the device cleanup jobs:
    ```bash
    kubectl delete jobs -n pelagia -l app=pelagia-lcm-cleanup-disks
    ```

## Deploy a new device after removal of a failed one <a name="add-new"></a>

!!! note

    You can spawn Ceph OSD on a raw device, but it must be clean and
    without any data or partitions. If you want to add a device that was in use,
    also ensure it is raw and clean. To clean up all data and partitions from a
    device, refer to official
    [Rook documentation](https://github.com/rook/rook/blob/master/Documentation/Storage-Configuration/ceph-teardown.md#zapping-devices).

1. Manually prepare the replacement device on the existing node.

2. Optional. If you want to add a Ceph OSD on top of a **raw** device that already exists
   on a node or is **hot-plugged**, add the required device using the following
   guidelines:

    - You can add a raw device to a node during node deployment.
    - If a node supports adding devices without a node reboot, you can hot plug
      a raw device to a node.
    - If a node does not support adding devices without a node reboot, you can
      hot plug a raw device during node shutdown.

3. Open the `CephDeployment` CR for editing:
   ```bash
   kubectl -n pelagia edit cephdpl
   ```

4. In the `nodes` section, add a new device:
   ```yaml
   spec:
     nodes:
     - name: <nodeName>
       devices:
       - fullPath: <deviceByID> # Recommended. Non-wwn by-id symlink.
         # name: <deviceByID> # Not recommended. If a device is supposed to be added with by-id.
         # fullPath: <deviceByPath> # Not recommended. If a device is supposed to be added with by-path.
         config:
           deviceClass: hdd
   ```

     Substitute `<nodeName>` with the node name where device `<deviceName>`
     or `<deviceByPath>` is going to be added as a Ceph OSD.

5. Verify that the Ceph OSD on the specified node is successfully deployed. The
   `CephDeploymentHealth` CR `status.healthReport.cephDaemons.cephDaemons` section should not contain any issues.
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
               - 3 osds, 3 up, 3 in
               status: ok
     ```

6. Verify the desired Ceph OSD pod is `Running`:
   ```bash
   kubectl -n rook-ceph get pod -l app=rook-ceph-osd -o wide | grep <nodeName>
   ```
