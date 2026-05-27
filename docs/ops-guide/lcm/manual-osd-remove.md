<a id="manual-osd-remove-remove-ceph-osd-manually"></a>

# Remove Ceph OSD manually

You may need to manually remove a Ceph OSD, for example, in the following
cases:

- If you accidentally have removed an OSD with ceph CLI or `rook-ceph-osd`
  deployment.
- If you do not want to rely on Pelagia LCM operations and want to manage the Ceph
  OSDs lifecycle manually.

To safely remove one or multiple Ceph OSDs from a Ceph cluster, perform the
following procedure for each Ceph OSD one by one.

!!! warning

    The procedure presupposes the Ceph OSD disk or logical volumes partition cleanup.

## Remove a Ceph OSD manually

1. Open the `CephDeployment` custom resource (CR) for editing:
   ```bash
   kubectl -n pelagia edit cephdpl
   ```

2. In the `spec.nodes` section, remove the required `devices` item of the corresponding node spec.
   When using `deviceFilter` or `devicePathFilter`, update regexp accordingly.

     If after removal `devices`, `deviceFilter`, or `devicePathFilter`
     become empty and the node spec has no roles specified, also remove the node
     spec.

3. Verify that all Ceph OSDs are `up` and `in`, the Ceph cluster is
   healthy, and no rebalance or recovery is in progress:
   ```bash
   kubectl -n rook-ceph exec -it deploy/rook-ceph-tools -- ceph -s
   ```

     Example of system response:
     ```bash
     cluster:
       id:     8cff5307-e15e-4f3d-96d5-39d3b90423e4
       health: HEALTH_OK
       ...
       osd: 4 osds: 4 up (since 10h), 4 in (since 10h)
     ```

4. Stop all deployments in Pelagia namespace to prevent autoscaling `rook-ceph-operator` deployment to 1 replica:
   ```bash
   kubectl -n pelagia scale deploy --all --replicas 0
   ```

5. Stop the Rook Ceph Operator deployment to avoid premature re-orchestration of the Ceph cluster:
   ```bash
   kubectl -n rook-ceph scale deploy rook-ceph-operator --replicas 0
   ```

6. Enter the `pelagia-ceph-toolbox` pod:
   ```bash
   kubectl -n rook-ceph exec -it deploy/pelagia-ceph-toolbox -- bash
   ```

7. Mark the required Ceph OSD as `out`:
   ```bash
   ceph osd out osd.<ID>
   ```

    !!! note

         In the command above and in the steps below, substitute `<ID>`
         with the number of the Ceph OSD to remove.

8. Wait until data backfilling to other OSDs is complete:
   ```bash
   ceph -s
   ```

     Once all the PGs are `active+clean`, backfilling is complete, and it is
     safe to remove the disk.

    !!! note

         For additional information on PGs backfilling, run `ceph pg dump_stuck`.

9. Exit from the `pelagia-ceph-toolbox` pod:
   ```bash
   exit
   ```

10. Scale the `rook-ceph/rook-ceph-osd-<ID>` deployment to `0` replicas:
    ```bash
    kubectl -n rook-ceph scale deploy rook-ceph-osd-<ID> --replicas 0
    ```

11. Enter the `pelagia-ceph-toolbox` pod:
    ```bash
    kubectl -n rook-ceph exec -it deploy/pelagia-ceph-toolbox -- bash
    ```

12. Verify that the number of Ceph OSDs that are `up` and `in` has decreased
    by one daemon:
    ```bash
    ceph -s
    ```

      Example of system response:
      ```bash
      osd: 4 osds: 3 up (since 1h), 3 in (since 5s)
      ```

13. Remove the Ceph OSD from the Ceph cluster:
    ```bash
    ceph osd purge <ID> --yes-i-really-mean-it
    ```

14. Delete the Ceph OSD `auth` entry, if present. Otherwise, skip this step.
    ```bash
    ceph auth del osd.<ID>
    ```

15. If you have removed the last Ceph OSD on the node and want to remove this
    node from the Ceph cluster, remove the CRUSH map entry:
    ```bash
    ceph osd crush remove <nodeName>
    ```

      Substitute `<nodeName>` with the name of the node where the removed Ceph
      OSD was placed.

16. Verify that the failure domain within Ceph OSDs has been removed from the
    CRUSH map:
    ```bash
    ceph osd tree
    ```

      If you have removed the node, it will be removed from the CRUSH map.

17. Exit from the `pelagia-ceph-toolbox` pod:
    ```bash
    exit
    ```

18. Clean up the disk used by the removed Ceph OSD. For details, see official
    [Rook: Zapping Devices](https://github.com/rook/rook/blob/master/Documentation/Storage-Configuration/ceph-teardown.md#zapping-devices).

    !!! warning

         If you are using multiple Ceph OSDs per device or metadata
         device, make sure that you can clean up the entire disk. Otherwise,
         instead clean up only the logical volume partitions for the volume group
         by running `lvremove <lvpartion_uuid>` any Ceph OSD pod that
         belongs to the same host as the removed Ceph OSD.

19. Delete the `rook-ceph/rook-ceph-osd-<ID>` deployment previously scaled to
    `0` replicas:
    ```bash
    kubectl -n rook-ceph delete deploy rook-ceph-osd-<ID>
    ```

      Substitute `<ID>` with the number of the removed Ceph OSD.

20. Scale the `rook-ceph/rook-ceph-operator` deployment to `1` replica and
    wait for the orchestration to complete:
    ```bash
    kubectl -n rook-ceph scale deploy rook-ceph-operator --replicas 1
    kubectl -n rook-ceph get pod -w
    ```

21. Scale all deployments in Pelagia namespace to continue spec reconcile and regular work:
    ```bash
    kubectl -n pelagia scale deploy --all --replicas 3
    ```

      Once done, Ceph OSD removal is complete.
