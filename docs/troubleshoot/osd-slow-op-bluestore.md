---
description: How to fix Ceph OSD experiencing slow operations in BlueStore
  during cluster deployment.
keywords: pelagia, ceph, ceph osd slow operation, ceph bluestore slow
  operation, ceph cluster deployment
---

<a id="osd-slow-op-bluestore-ceph-osd-experiencing-slow-operations-in-bluestore"></a>

# Ceph OSD experiencing slow operations in BlueStore

During cluster deployment, the following false-positive example alert for
Ceph may raise:
```bash
Failed to configure Ceph cluster: ceph cluster verification is failed:
[BLUESTORE_SLOW_OP_ALERT: 3 OSD(s) experiencing slow operations in BlueStore]
```

The issue occurs due to the following upstream Ceph issues:

- [Discussion 15403](https://github.com/rook/rook/discussions/15403)
- [Bug 68337](https://tracker.ceph.com/issues/68337)
- [Bug 70940](https://tracker.ceph.com/issues/70940)
- [Bug 71168](https://tracker.ceph.com/issues/71168)

**To verify whether the cluster is affected:**

1. Enter the `pelagia-ceph-tools` pod:
2. Verify the Ceph cluster status:

   - Verify Ceph health:
     ```bash
     ceph -s
     ```

     Example of a positive system response in the affected cluster:
     ```bash
     cluster:
       id:     6ae41eb3-262e-4da9-8847-25efed2fcaa2
       health: HEALTH_WARN
               2 OSD(s) experiencing slow operations in BlueStore

     services:
       mon: 3 daemons, quorum a,b,c (age 9h)
       mgr: a(active, since 9h), standbys: b
       osd: 4 osds: 4 up (since 9h), 4 in (since 9h)
       rgw: 2 daemons active (2 hosts, 1 zones)

     data:
       pools:   15 pools, 409 pgs
       objects: 1.67k objects, 4.6 GiB
       usage:   11 GiB used, 2.1 TiB / 2.1 TiB avail
       pgs:     409 active+clean

     io:
       client:   85 B/s rd, 500 KiB/s wr, 0 op/s rd, 27 op/s wr
     ```

   - Verify Ceph health details:
     ```bash
     ceph health detail
     ```

     Example of a positive system response in the affected cluster:
     ```bash
     HEALTH_WARN 2 OSD(s) experiencing slow operations in BlueStore
     [WRN] BLUESTORE_SLOW_OP_ALERT: 2 OSD(s) experiencing slow operations in BlueStore
          osd.2 observed slow operation indications in BlueStore
          osd.3 observed slow operation indications in BlueStore
     ```

3. Exit the `pelagia-ceph-tools` pod.

**To resolve the issue:**

Configure the `bluestore_slow_ops_warn` options as follows:
```bash
kubectl -n ceph-lcm-mirantis edit cephdeployment
```

```yaml
spec:
  cephClusterSpec:
    rookConfig:
      osd|bluestore_slow_ops_warn_lifetime: "600"
      osd|bluestore_slow_ops_warn_threshold: "10"
```

Wait for up to five minutes for the change to apply and the alert to disappear
during cluster deployment.

This configuration triggers the alert only if at least 10 BlueStore slow
operations occur during last 10 minutes. If triggered, it indicates a potential
hardware disk issue on the BlueStore host that must be verified and
reconfigured accordingly.
