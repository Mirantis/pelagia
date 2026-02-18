<a id="ceph-mon-store-size-grow-ceph-monitors-storedb-size-rapidly-growing"></a>

# Ceph Monitors store.db size rapidly growing

The `MON_DISK_LOW` Ceph Cluster health message indicates that the
`store.db` size of the Ceph Monitor is rapidly growing and the `compaction`
procedure is not working. In most cases, `store.db` starts storing a number
of `logm` keys that are buffered due to Ceph OSD shadow errors.

**To verify whether the store.db size is rapidly growing:**

1. Identify the Ceph Monitors `store.db` size:
   ```bash
   for pod in $(kubectl get pods -n rook-ceph | grep mon | awk '{print $1}'); \
   do printf "$pod:\n"; kubectl exec -n rook-ceph "$pod" -it -c mon -- \
   du -cms /var/lib/ceph/mon/ ; done
   ```

2. Repeat the previous step two or three times within the interval of 5–15
   seconds.

If between the command runs the total size increases by more than 10 MB,
perform the steps described below to resolve the issue.

**To apply the issue resolution:**

1. Verify the original state of placement groups (PGs):
   ```bash
   kubectl -n rook-ceph exec -it \
       deploy/pelagia-ceph-toolbox \
       -- ceph -s
   ```

2. Apply `clog_to_monitors` with the `false` value for all Ceph OSDs at
   runtime:
   ```bash
   kubectl -n rook-ceph exec -it \
       deploy/pelagia-ceph-toolbox \
       -- bash
   ceph tell osd.* config set clog_to_monitors false
   ```

3. Restart Ceph OSDs one by one:

     1. Restart one of the Ceph OSDs:
        ```bash
        for pod in $(kubectl get pods -n rook-ceph -l app=rook-ceph-osd | \
        awk 'FNR>1{print $1}'); do printf "$pod:\n"; kubectl -n rook-ceph \
        delete pod "$pod"; echo "Continue?"; read; done
        ```

     2. Once prompted `Continue?`, first verify that rebalancing has finished
        for the Ceph cluster, the Ceph OSD is `up` and `in`, and all PGs have
        returned to their original state:
        ```bash
        kubectl -n rook-ceph exec -it deploy/pelagia-ceph-toolbox -- ceph -s
        kubectl -n rook-ceph exec -it deploy/pelagia-ceph-toolbox -- ceph osd tree
        ```

        Once you are confident that the Ceph OSD restart and recovery is over,
        press `ENTER`.

     3. Restart the remaining Ceph OSDs.

        !!! note

              Periodically verify the Ceph Monitors `store.db` size:
              ```bash
              for pod in $(kubectl get pods -n rook-ceph | grep mon | awk \
              '{print $1}'); do printf "$pod:\n"; kubectl exec -n rook-ceph \
              "$pod" -it -c mon -- du -cms /var/lib/ceph/mon/ ; done
              ```

After some of the affected Ceph OSDs restart, Ceph Monitors will start
decreasing the `store.db` size to the original 100–300 MB. However,
complete the restart of all Ceph OSDs.
