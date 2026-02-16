<a id="ceph-kcor-timeout"></a>

# CephOsdRemoveTask failure with a timeout during rebalance

Ceph OSD removal procedure includes the Ceph OSD `out` action that starts
the Ceph PGs rebalancing process. The total time for rebalancing depends on a
cluster hardware configuration: network bandwidth, Ceph PGs placement, number
of Ceph OSDs, and so on. The default rebalance timeout is limited by `30`
minutes, which applies to standard cluster configurations.

If the rebalance takes more than 30 minutes, the `CephOsdRemoveTask`
resources created for removing Ceph OSDs or nodes fail with the following
example message:

```yaml
status:
  messages:
  - Timeout (30m0s) reached for waiting pg rebalance for osd 2
```

**To apply the issue resolution**, increase the timeout for all future
`CephOsdRemoveTask` resources:

Update `pelagia-lcmconfig` ConfigMap in Pelagia namespace with the key `TASK_OSD_PG_REBALANCE_TIMEOUT_MIN` and desired
timeout in minutes in string format:

```bash
kubectl -n pelagia edit cm pelagia-lcmconfig
```

Example configuration:
```yaml
data:
  TASK_OSD_PG_REBALANCE_TIMEOUT_MIN: "180"
```

where `"180"` means 180 minutes before timeout.

If you have an existing `CephOsdRemoveTask` resource with issues in
`messages` to process:

1. In the failed `CephOsdRemoveTask` resource, copy the `spec` section.
2. Create a new `CephOsdRemoveTask` with a different name. For details,
   see [Creating a Ceph OSD remove task](../ops-guide/lcm/create-task-workflow.md#create-osd-rm-request).
3. Paste the previously copied `spec` section of the failed
   `CephOsdRemoveTask` resource to the new one.
4. Remove the failed `CephOsdRemoveTask` resource.
