!!! warning

    We do not recommend setting device name or device `by-path` symlink in the `cleanupByDevice` field
    as these identifiers are not persistent and can change at node boot. Remove Ceph OSDs with `by-id`
    symlinks or use `cleanupByOsdId` instead. For details, see
    [Addressing Ceph storage devices](../../architecture/addressing-ceph-devices.md#addressing-ceph-devices-addressing-ceph-storage-devices).
