!!! note

    You can spawn Ceph OSD on a raw device, but it must be clean and
    without any data or partitions. If you want to add a device that was in use,
    also ensure it is raw and clean. To clean up all data and partitions from a
    device, refer to [Rook documentation: Zapping Devices](https://rook.io/docs/rook/{{ rook_doc_version }}/Storage-Configuration/ceph-teardown/#zapping-devices).
