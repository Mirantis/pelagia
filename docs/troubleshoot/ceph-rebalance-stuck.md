---
description: How to fix Ceph rebalancing that gets stuck during disabled node
  removal by reweighting the OSD.
keywords: pelagia, ceph osd rebalance stuck, reweight osd, node removal,
  cephdeployment, ceph maintenance
---

<a id="ceph-rebalance-stuck-ceph-rebalancing-gets-stuck-during-disabled-node-removal"></a>

# Ceph rebalancing gets stuck during disabled node removal

When disabling or removing a Ceph node during operations such as a rolling
reboot, Ceph may not finish rebalancing if only two of three OSD nodes remain
active. The `CephDeployment` object can remain in `Maintenance`, causing
the rebalance process to wait indefinitely for Ceph to become ready. The issue
may only affect environments with a small number of Ceph OSD nodes, pool
replica count set to one less than the number of storage nodes
(`replicas=storage_nodes_count-1`), and failure domain `host`.

To resolve the issue, run the following command on the affected Ceph OSD node:

```bash
ceph osd reweight <osdId> 0
```
