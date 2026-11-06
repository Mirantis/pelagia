!!! warning

    We do not recommend using replicated Ceph pools with `replicated.size` of less
    than `3` in production environments. For such pools, Ceph OSD removal may cause
    issues. If `replicated.size` is `1`, Ceph LCM operations cannot function correctly.
    Set `replicated.size` to `3` for each replicated Ceph pool. For erasure-coded pools,
    the minimal recommended configuration is `2+1`, that is,
    `erasureCoded.dataChunks: 2` and `erasureCoded.codingChunks: 1`.
