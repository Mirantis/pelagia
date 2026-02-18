<a id="ceph-exporter-crash-the-ceph-exporter-pods-are-present-in-the-ceph-crash-list"></a>

# The ceph-exporter pods are present in the Ceph crash list

After a managed cluster update, the `ceph-exporter` pods may be present in
the **ceph crash ls** or **ceph health detail** list while
`rook-ceph-exporter` attempts to obtain the port that is still in use.
For example:
```bash
$ ceph health detail

HEALTH_WARN 1 daemons have recently crashed
[WRN] RECENT_CRASH: 1 daemons have recently crashed
    client.ceph-exporter crashed on host kaas-node-b59f5e63-2bfd-43aa-bc80-42116d71188c at 2024-10-01T16:43:31.311563Z
```

The issue does not block the managed cluster update. Once the port becomes
available, `rook-ceph-exporter` obtains the port and the issue disappears.

**To apply the issue resolution**, run the following command to remove `ceph-exporter` pods from the Ceph crash list:
```bash
ceph crash archive-all
```
