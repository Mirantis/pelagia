<a id="ceph-pg-damaged-ceph-health-reports-pg_damaged_after-a-failed-disk-or-node-replacement"></a>

# Ceph health reports *PG_DAMAGED* after a failed disk or node replacement

After adding a new OSD node on a compact cluster, Ceph health may report
`HEALTH_ERR` with the `ceph health detail` command output containing
`PG_DAMAGED` and `OSD_SCRUB_ERRORS` messages. For example:

```bash
$ ceph -s

cluster:
 id:     8bca9dfb-df99-4920-bba0-e5bca59876b4
 health: HEALTH_ERR
         1 scrub errors
         Possible data damage: 1 pg inconsistent

services:
 mon: 3 daemons, quorum a,b,c (age 3h)
 mgr: a(active, since 3h), standbys: b
 osd: 4 osds: 4 up (since 109m), 4 in (since 110m)
 rgw: 2 daemons active (2 hosts, 1 zones)
```

```bash
$ ceph health detail

HEALTH_ERR 1 scrub errors; Possible data damage: 1 pg inconsistent
[ERR] OSD_SCRUB_ERRORS: 1 scrub errors
[ERR] PG_DAMAGED: Possible data damage: 1 pg inconsistent
    pg 11.2a is active+clean+inconsistent, acting [3,1]
```

**To fix the PG_DAMAGED health error:**

1. Obtain the damaged placement group (PG) ID:
   ```bash
   ceph health detail
   ```

     Example of system response:
     ```bash
     HEALTH_ERR 1 scrub errors; Possible data damage: 1 pg inconsistent
     [ERR] OSD_SCRUB_ERRORS: 1 scrub errors
     [ERR] PG_DAMAGED: Possible data damage: 1 pg inconsistent
         pg 11.2a is active+clean+inconsistent, acting [3,1]
     ```

     In the example above, `11.2a` is the required PG ID.

2. Repair the damaged PG:
   ```bash
   ceph pg repair <pgid>
   ```

     Substitute `<pgid>` with a damaged PG ID. For example:
     ```bash
     ceph pg repair 11.2a
     ```
