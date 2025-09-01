<a id="verify-ceph-core"></a>

# Verify the Ceph core services

To confirm that all Ceph components including `mon`, `mgr`, `osd`, and
`rgw` have joined your cluster properly, analyze the logs for each pod and
verify the Ceph status:

```bash
kubectl -n rook-ceph exec -it deploy/pelagia-ceph-toolbox -- ceph -s
```

Example of a positive system response:

```bash
cluster:
    id:     4336ab3b-2025-4c7b-b9a9-3999944853c8
    health: HEALTH_OK

services:
    mon: 3 daemons, quorum a,b,c (age 20m)
    mgr: a(active, since 19m)
    osd: 6 osds: 6 up (since 16m), 6 in (since 16m)
    rgw: 1 daemon active (miraobjstore.a)

data:
    pools:   12 pools, 216 pgs
    objects: 201 objects, 3.9 KiB
    usage:   6.1 GiB used, 174 GiB / 180 GiB avail
    pgs:     216 active+clean
```

To verify Ceph cluster health, run:
```bash
kubectl -n rook-ceph exec -it deploy/pelagia-ceph-toolbox -- ceph health detail
```

A healthy cluster returns the following output:
```bash
HEALTH_OK
```

If the output contains `HEALTH_WARN` or `HEALTH_ERR`, please see
[Ceph Health Checks](https://docs.ceph.com/en/latest/rados/operations/health-checks/).
