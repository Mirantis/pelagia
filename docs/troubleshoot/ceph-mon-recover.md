<a id="ceph-mon-recover-ceph-monitors-recovery"></a>

# Ceph Monitors recovery

This section describes how to recover failed Ceph Monitors of an existing Ceph
cluster in the following state:

* The Ceph cluster contains failed Ceph Monitors that cannot start and hang
  in the `Error` or `CrashLoopBackOff` state.
* The logs of the failed Ceph Monitor pods contain the following lines:
  ```bash
  mon.g does not exist in monmap, will attempt to join an existing cluster
  ...
  mon.g@-1(???) e11 not in monmap and have been in a quorum before; must have been removed
  mon.g@-1(???) e11 commit suicide!
  ```

* Rook Ceph Operator failover procedure produces new Ceph Monitor Pods, which are failing to
  `CrashLoopBackOff` status.
* The Ceph cluster contains at least one `Running` Ceph Monitor and the
  `ceph -s` command outputs one healthy `mon` and one healthy
  `mgr` instance.

Perform the following steps for all failed Ceph Monitors at a time if not
stated otherwise.

**To recover failed Ceph Monitors:**

1. Scale the Rook Ceph Operator deployment down to `0` replicas:
   ```bash
   kubectl -n rook-ceph scale deploy rook-ceph-operator --replicas 0
   ```

2. Delete all failed Ceph Monitor deployments:

     1. Identify the Ceph Monitor pods in the `Error` or `CrashLookBackOff`
        state:
        ```bash
        kubectl -n rook-ceph get pod -l 'app in (rook-ceph-mon,rook-ceph-mon-canary)'
        ```

     2. Verify that the affected pods contain the failure logs described above:
        ```bash
        kubectl -n rook-ceph logs <failedMonPodName>
        ```

          Substitute `<failedMonPodName>` with the Ceph Monitor pod name. For
          example, `rook-ceph-mon-g-845d44b9c6-fjc5d`.

     3. Save the identifying letters of failed Ceph Monitors for further usage.
        For example, `f`, `e`, and so on.
     4. Delete all corresponding deployments of these pods:

          1. Identify the affected Ceph Monitor pod deployments:
             ```bash
             kubectl -n rook-ceph get deploy -l 'app in (rook-ceph-mon,rook-ceph-mon-canary)'
             ```

          2. Delete the affected Ceph Monitor pod deployments. For example, if the
             Ceph cluster has the `rook-ceph-mon-c-845d44b9c6-fjc5d` pod in the
             `CrashLoopBackOff` state, remove the corresponding `rook-ceph-mon-c`:
             ```bash
             kubectl -n rook-ceph delete deploy rook-ceph-mon-c
             ```

               Canary `mon` deployments have the suffix `-canary`.

3. Remove all corresponding entries of Ceph Monitors from the MON map:

     1. Enter the `pelagia-ceph-toolbox` pod:
        ```bash
        kubectl -n rook-ceph exec -it deploy/pelagia-ceph-toolbox -- bash
        ```

     2. Inspect the current MON map and save the IP addresses of the failed Ceph
        monitors for further usage:
        ```bash
        ceph mon dump
        ```

     3. Remove all entries of failed Ceph Monitors using the previously saved
        letters:
        ```bash
        ceph mon rm <monLetter>
        ```

          Substitute `<monLetter>` with the corresponding letter of a failed Ceph Monitor.

     4. Exit the `pelagia-ceph-toolbox` pod.

4. Remove all failed Ceph Monitors entries from the Rook `mon` endpoints
   ConfigMap:

     1. Open the `rook-ceph/rook-ceph-mon-endpoints` ConfigMap for editing:
        ```bash
        kubectl -n rook-ceph edit cm rook-ceph-mon-endpoints
        ```

     2. Remove all entries of failed Ceph Monitors from the ConfigMap data and
        update the `maxMonId` value with the current number of `Running`
        Ceph Monitors. For example, `rook-ceph-mon-endpoints` has the
        following `data`:
        ```yaml
        data:
          csi-cluster-config-json: '[{"clusterID":"rook-ceph","monitors":["172.0.0.222:6789","172.0.0.223:6789","172.0.0.224:6789","172.16.52.217:6789","172.16.52.216:6789"]}]'
          data: a=172.0.0.222:6789,b=172.0.0.223:6789,c=172.0.0.224:6789,f=172.0.0.217:6789,e=172.0.0.216:6789
          mapping: '{"node":{
              "a":{"Name":"node-21465871-42d0-4d56-911f-7b5b95cb4d34","Hostname":"node-21465871-42d0-4d56-911f-7b5b95cb4d34","Address":"172.16.52.222"},
              "b":{"Name":"node-43991b09-6dad-40cd-93e7-1f02ed821b9f","Hostname":"node-43991b09-6dad-40cd-93e7-1f02ed821b9f","Address":"172.16.52.223"},
              "c":{"Name":"node-15225c81-3f7a-4eba-b3e4-a23fd86331bd","Hostname":"node-15225c81-3f7a-4eba-b3e4-a23fd86331bd","Address":"172.16.52.224"},
              "e":{"Name":"node-ba3bfa17-77d2-467c-91eb-6291fb219a80","Hostname":"node-ba3bfa17-77d2-467c-91eb-6291fb219a80","Address":"172.16.52.216"},
              "f":{"Name":"node-6f669490-f0c7-4d19-bf73-e51fbd6c7672","Hostname":"node-6f669490-f0c7-4d19-bf73-e51fbd6c7672","Address":"172.16.52.217"}}
          }'
          maxMonId: "5"
        ```

          If `e` and `f` are the letters of failed Ceph Monitors, the resulting
          ConfigMap data must be as follows:
          ```yaml
          data:
            csi-cluster-config-json: '[{"clusterID":"rook-ceph","monitors":["172.0.0.222:6789","172.0.0.223:6789","172.0.0.224:6789"]}]'
            data: a=172.0.0.222:6789,b=172.0.0.223:6789,c=172.0.0.224:6789
            mapping: '{"node":{
                "a":{"Name":"node-21465871-42d0-4d56-911f-7b5b95cb4d34","Hostname":"node-21465871-42d0-4d56-911f-7b5b95cb4d34","Address":"172.16.52.222"},
                "b":{"Name":"node-43991b09-6dad-40cd-93e7-1f02ed821b9f","Hostname":"node-43991b09-6dad-40cd-93e7-1f02ed821b9f","Address":"172.16.52.223"},
                "c":{"Name":"node-15225c81-3f7a-4eba-b3e4-a23fd86331bd","Hostname":"node-15225c81-3f7a-4eba-b3e4-a23fd86331bd","Address":"172.16.52.224"}}
            }'
            maxMonId: "3"
          ```

5. Back up the data of the failed Ceph Monitors one by one:

     1. SSH to the node of a failed Ceph Monitor using the previously saved IP
        address.
     2. Move the Ceph Monitor data directory to another place:
        ```bash
        mv /var/lib/rook/mon-<letter> /var/lib/rook/mon-<letter>.backup
        ```

     3. Close the SSH connection.

6. Scale the Rook Ceph Operator deployment up to `1` replica:
   ```bash
   kubectl -n rook-ceph scale deploy rook-ceph-operator --replicas 1
   ```

7. Wait until all Ceph Monitors are in the `Running` state:
   ```bash
   kubectl -n rook-ceph get pod -l app=rook-ceph-mon -w
   ```

8. Restore the data from the backup for each recovered Ceph Monitor one by one:

     1. Enter a recovered Ceph Monitor pod:
        ```bash
        kubectl -n rook-ceph exec -it <monPodName> bash
        ```

          Substitute `<monPodName>` with the recovered Ceph Monitor pod name. For
          example, `rook-ceph-mon-g-845d44b9c6-fjc5d`.

     2. Recover the `mon` data backup for the current Ceph Monitor:
        ```bash
        ceph-monstore-tool /var/lib/rook/mon-<letter>.backup/data store-copy /var/lib/rook/mon-<letter>/data/
        ```

          Substitute `<letter>` with the current Ceph Monitor pod letter, for example, `e`.

9. Verify the Ceph state. The output must indicate the desired number of Ceph
   Monitors, and all of them must be in quorum.
   ```bash
   kubectl -n rook-ceph exec -it deploy/pelagia-ceph-toolbox -- ceph -s
   ```
