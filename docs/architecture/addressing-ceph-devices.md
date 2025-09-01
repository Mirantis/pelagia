# Addressing Ceph storage devices

There are several formats to use when specifying and addressing storage devices
of a Ceph cluster. The default and recommended one is the `/dev/disk/by-id`
format. This format is reliable and unaffected by the disk controller actions,
for example, device name shuffling on boot.

## Difference between `by-id`, `name`, and `by-path` formats

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

An exception to this rule is the `wwn` `by-id` symlinks, which are
programmatically generated at boot. They are not solely based on disk
serial numbers but also include other node information. This can lead
to the `wwn` being recalculated when the node reboots. As a result,
this symlink type cannot guarantee a persistent disk identifier and should
not be used as a stable storage device symlink in a Ceph cluster.

The storage device `name` format cannot be considered
persistent because the sequence in which block devices are added during boot
is semi-arbitrary. This means that block device names, for example, `nvme0n1`
and `sdc`, are assigned to physical disks during discovery, which may vary
inconsistently from the previous node state.

The storage device `by-path` format is supported, but we recommend using `by-id` symlinks instead of `by-path`
symlinks due to `by-id` symlinks directly refer to the disk serial number.

Therefore, we are highly recommending using storage device `by-id` symlinks
that contain disk serial numbers. This approach enables you to use a persistent
device identifier addressed in the Ceph cluster specification.

## Example `CephDeployment` with device `by-id` identifiers

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
