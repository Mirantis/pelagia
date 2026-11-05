---
description: How to fix ceph-exporter pods appearing in Ceph crash list after cluster update.
keywords: pelagia, ceph, ceph exporter crash, ceph crash list, rook ceph exporter, ceph crash
  archive, cluster update
---

<a id="ceph-exporter-crash-the-ceph-exporter-pods-are-present-in-the-ceph-crash-list"></a>

# The ceph-exporter pods are present in the Ceph crash list

After a cluster update, the `ceph-exporter` pods may be present in
the **ceph crash ls** or **ceph health detail** list while
`rook-ceph-exporter` attempts to obtain the port that is still in use.
For example:
```bash
$ ceph health detail

HEALTH_WARN 1 daemons have recently crashed
[WRN] RECENT_CRASH: 1 daemons have recently crashed
    client.ceph-exporter crashed on host <nodeName> at 2024-10-01T16:43:31.311563Z
```

The issue does not block the cluster update. Once the port becomes
available, `rook-ceph-exporter` obtains the port and the issue disappears.

**To resolve the issue**, run the following command in the `pelagia-ceph-toolbox` pod to remove
`ceph-exporter` pods from the Ceph crash list:
```bash
kubectl -n rook-ceph exec -it deploy/pelagia-ceph-toolbox -- ceph crash archive-all
```
