!!! warning

    If you define RGW pools yourself in `spec.blockStorage.pools`, set
    `useAsFullName: true` for each of them. Without it, Pelagia adds the
    `deviceClass` suffix to the pool name, so the pool does not match the name
    that Ceph RGW expects, and shared pools fail validation with
    `pool … is not found in spec`. Also, set `storageClassOpts.skipClassCreate: true`
    so that no `StorageClass` is created for these pools.
