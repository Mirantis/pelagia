# Replace a failed Ceph OSD disk with a metadata device as a device name

You can apply the below procedure if a Ceph OSD failed with data disk outage
and Ceph OSD metadata device specified as a disk. This scenario implies that the Ceph cluster
automatically creates a required metadata logical volume on a desired device.

## Remove a Ceph OSD with a metadata device as a disk name

To remove the affected Ceph OSD with a metadata device as a device name,
follow the
[Remove a failed Ceph OSD by ID with a defined metadata device](https://mirantis.github.io/pelagia/ops-guide/lcm/replace-osd-meta-lcm#replace-osd-meta-by-id)
procedure and capture the following details:

- While editing `CephDeployment` custom resource (CR) in the `nodes` section, capture the
  `metadataDevice` path to reuse it during re-creation of the Ceph OSD.

    Example of the `spec.nodes` section:
    ```yaml
    spec:
      nodes:
      - name: <nodeName>
        devices:
        - name: <deviceName>  # remove the entire item from the devices list
          # fullPath: <deviceByPath> if device is specified using by-path instead of name
          config:
            deviceClass: hdd
            metadataDevice: /dev/nvme0n1
    ```

    In the example above, save the `metadataDevice` device name `/dev/nvme0n1`.

- During `CephOsdRemoveTask` CR verification of `removeInfo`, capture the `usedPartition` value
  of the metadata device located in the `deviceMapping.<metadataDevice>` section.

      Example of the `removeInfo` section:
    ```yaml
    removeInfo:
      cleanUpMap:
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
                  usedPartition: /dev/ceph-b0c70c72-8570-4c9d-93e9-51c3ab4dd9f9/osd-db-ecf64b20-1e07-42ac-a8ee-32ba3c0b7e2f
              uuid: ef516477-d2da-492f-8169-a3ebfc3417e2
    ```

    In the example above, capture the following values from the `<metadataDevice>` section:

      - `ceph-b0c70c72-8570-4c9d-93e9-51c3ab4dd9f9` - name of the volume group
        that contains all metadata partitions on the `<metadataDevice>` disk;
      - `osd-db-ecf64b20-1e07-42ac-a8ee-32ba3c0b7e2f` - name of the logical
        volume that relates to a failed Ceph OSD.

## Re-create the partition on the existing metadata disk <a name="recreate-meta-lvm"></a>

After you remove the Ceph OSD disk, manually create a separate logical volume
for the metadata partition in an existing volume group on the metadata device:

```bash
lvcreate -l 100%FREE -n meta_1 <vgName>
```

Substitute `<vgName>` with the name of a volume group captured in the `usedPartiton` parameter.

!!! note

    If you removed more than one OSD, replace `100%FREE` with the corresponding partition size. For example:
    ```bash
    lvcreate -l <partitionSize> -n meta_1 <vgName>
    ```

    Substitute `<partitionSize>` with the corresponding value that matches the
    size of other partitions placed on the affected metadata drive. To obtain
    `<partitionSize>`, use the output of the **lvs** command. For example:
    `16G`.

During execution of the `lvcreate` command, the system asks you to wipe the found bluestore label on a metadata device.
For example:
```bash
WARNING: ceph_bluestore signature detected on /dev/ceph-b0c70c72-8570-4c9d-93e9-51c3ab4dd9f9/meta_1 at offset 0. Wipe it? [y/n]:
```

Using the interactive shell, answer `n` to keep all metadata partitions
alive. After answering `n`, the system outputs the following:

```bash
Aborted wiping of ceph_bluestore.
1 existing signature left on the device.
Logical volume "meta_1" created.
```

## Re-create the Ceph OSD with the re-created metadata partition

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

3. In the `nodes` section, add the replaced device with the same `metadataDevice` path as in the previous Ceph OSD:
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
           metadataDevice: /dev/<vgName>/meta_1
   ```

     Substitute `<nodeName>` with the node name where the new device `<deviceByID>` or `<deviceByPath>` must be added.
     Also specify `metadataDevice` with the path to the logical volume created earlier.

4. Wait for the replaced disk to apply to the Ceph cluster as a new Ceph OSD.
   You can monitor the application state using either the `status` section
   of the `CephDeploymentHealth` CR or in the `pelagia-ceph-toolbox` pod:
   ```bash
   kubectl -n pelagia get cephdeploymenthealth -o yaml
   kubectl -n rook-ceph exec -it deploy/pelagia-cephtoolbox -- ceph -s
   ```
