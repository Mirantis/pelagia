<a id="stuck-maintenance-maintenance-stuck-on-a-compact-ceph-cluster"></a>

# Maintenance stuck on a compact Ceph cluster

{% include "../../snippets/replicatedSize.md" %}

When disabling or removing a Ceph node during upgrade or maintenance operations
such as rolling reboot, Ceph may not complete rebalancing if only two of three OSD
nodes remain active. The `CephDeployment` object can remain in `Maintenance`, causing
the rebalance process to wait indefinitely for Ceph to become ready.

The issue may only affect environments with a small number of Ceph OSD nodes (for example, three),
pool replica count set to one less than the number of storage nodes (`replicas=storage_nodes_count-1`),
and failure domain `host`.

**To apply the issue resolution**, run the following command for the affected Ceph OSD node:
```bash
ceph osd reweight <osdId> 0
```
