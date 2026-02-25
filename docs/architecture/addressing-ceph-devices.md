<a id="addressing-ceph-devices-addressing-ceph-storage-devices"></a>
# Addressing Ceph storage devices

Selecting the correct identifier for storage devices is critical for ensuring system stability, particularly when configuring mount points or managing hardware-dependent services.

The list of supported formats for device identification includes `by-id`, `name`, and `by-path`.
The default and recommended device identification methods is `/dev/disk/by-id`.
This format is reliable and unaffected by the disk controller actions, for example, device name shuffling on boot.

This section explains each method in detail.

## The `by-id` identifier

The storage device `/dev/disk/by-id` format mostly bases on a disk serial
number, which is unique for each disk. A `by-id` symlink is created by the
`udev` rules in the following format, where `<BusID>` is an ID of the bus
to which the disk is attached and `<DiskSerialNumber>` stands for a unique
disk serial number:

```ini
/dev/disk/by-id/<BusID>-<DiskSerialNumber>
```

Typical `by-id` symlinks for storage devices look as follows:

```ini
/dev/disk/by-id/nvme-SAMSUNG_MZ1LB3T8HMLA-00007_S46FNY0R394543
/dev/disk/by-id/scsi-SATA_HGST_HUS724040AL_PN1334PEHN18ZS
/dev/disk/by-id/ata-WDC_WD4003FZEX-00Z4SA0_WD-WMC5D0D9DMEH
```

In the example above, symlinks contain the following IDs:

* Bus IDs: `nvme`, `scsi-SATA` and `ata`
* Disk serial numbers: `SAMSUNG_MZ1LB3T8HMLA-00007_S46FNY0R394543`,
  `HGST_HUS724040AL_PN1334PEHN18ZS` and
  `WDC_WD4003FZEX-00Z4SA0_WD-WMC5D0D9DMEH`.

Below is an example `CephDeployment` custom resource using the `/dev/disk/by-id`
format for storage devices specification:

```yaml
apiVersion: lcm.mirantis.com/v1alpha1
kind: CephDeployment
metadata:
  name: pelagia-ceph
  namespace: pelagia
spec:
  nodes:
    # Add the exact node names.
    # Obtain the name from the "kubectl get node" list.
    - name: cluster-storage-worker-0
      roles:
      - mgr
      - mon
      devices:
      - config:
          deviceClass: ssd
        fullPath: /dev/disk/by-id/scsi-1ATA_WDC_WDS100T2B0A-00SM50_200231440912
    - name: cluster-storage-worker-1
      roles:
      - mgr
      - mon
      devices:
      - config:
          deviceClass: ssd
        fullPath: /dev/disk/by-id/nvme-SAMSUNG_MZ1LB3T8HMLA-00007_S46FNY0R394543
    - name: cluster-storage-worker-2
      roles:
      - mgr
      - mon
      devices:
      - config:
          deviceClass: ssd
        fullPath: /dev/disk/by-id/nvme-SAMSUNG_ML1EB3T8HMLA-00007_S46FNY1R130423
  pools:
  - default: true
    deviceClass: ssd
    name: kubernetes
    replicated:
      size: 3
```

## The `name` identifier

The storage device `name` format cannot be considered
persistent because the sequence in which block devices are added during boot
is semi-arbitrary. This means that block device names, for example, `nvme0n1`
and `sdc`, are assigned to physical disks during discovery, which may vary
inconsistently from the previous node state.

## The `by-path` identifier

The storage device `by-path` format is supported, but we recommend using `by-id` symlinks instead of `by-path`
symlinks due to `by-id` symlinks directly refer to the disk serial number.

Therefore, we are highly recommending using storage device `by-id` symlinks
that contain disk serial numbers. This approach enables you to use a persistent
device identifier addressed in the Ceph cluster specification.
