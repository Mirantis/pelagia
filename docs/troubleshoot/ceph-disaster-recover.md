<a id="ceph-disaster-recover-ceph-disaster-recovery"></a>

# Ceph disaster recovery

This section describes how to recover a failed or accidentally removed Ceph
cluster in the following cases:

* If Pelagia Controller underlying a running Rook Ceph cluster has failed, and you
  want to install a new Pelagia Helm release and recover the failed
  Ceph cluster onto the new Pelagia Controllers.
* To migrate the data of an existing Ceph cluster to a new deployment in case
  downtime can be tolerated.

Consider the common state of a failed or removed Ceph cluster:

* The Rook namespace does not contain pods, or they are in the `Terminating` state.
* The Rook or/and Pelagia namespaces are in the `Terminating` state.
* Pelagia Helm release is failed or absent.
* The Rook `CephCluster`, `CephBlockPool`, `CephObjectStore` CRs in the
  Rook namespace cannot be found or have the `deletionTimestamp` parameter in the `metadata` section.

!!! note

    Prior to recovering the Ceph cluster, verify that your deployment meets the following prerequisites:

    1. The Ceph cluster `fsid` exists.
    2. The Ceph cluster Monitor keyrings exist.
    3. The Ceph cluster devices exist and include the data previously handled by
       Ceph OSDs.

## Ceph cluster recovery workflow

1. Create a backup of the remaining data and resources.
2. Clean up the failed or removed Pelagia Helm release.
3. Deploy a new Pelagia Helm release with the previously used `CephDeployment` custom resource (CR)
   and one Ceph Monitor.
4. Replace the `ceph-mon` data with the old cluster data.
5. Replace `fsid` in `secrets/rook-ceph-mon` with the old one.
6. Fix the Monitor map in the `ceph-mon` database.
7. Fix the Ceph Monitor authentication key and disable authentication.
8. Start the restored cluster and inspect the recovery.
9. Fix the admin authentication key and enable authentication.
10. Restart the cluster.

## Recover a failed or removed Ceph cluster

1. Back up the remaining resources. Skip the commands for the resources that
   have already been removed:
   ```bash
   kubectl -n rook-ceph get cephcluster <clusterName> -o yaml > backup/cephcluster.yaml
   # perform this for each cephblockpool
   kubectl -n rook-ceph get cephblockpool <cephBlockPool-i> -o yaml > backup/<cephBlockPool-i>.yaml
   # perform this for each client
   kubectl -n rook-ceph get cephclient <cephclient-i> -o yaml > backup/<cephclient-i>.yaml
   kubectl -n rook-ceph get cephobjectstore <cephObjectStoreName> -o yaml > backup/<cephObjectStoreName>.yaml
   kubectl -n rook-ceph get cephfilesystem <cephfilesystemName> -o yaml > backup/<cephfilesystemName>.yaml
   # perform this for each secret
   kubectl -n rook-ceph get secret <secret-i> -o yaml > backup/<secret-i>.yaml
   # perform this for each configMap
   kubectl -n rook-ceph get cm <cm-i> -o yaml > backup/<cm-i>.yaml
   ```

2. SSH to each node where the Ceph Monitors or Ceph OSDs were placed before the
   failure and back up the valuable data:
   ```bash
   mv /var/lib/rook /var/lib/rook.backup
   mv /etc/ceph /etc/ceph.backup
   mv /etc/rook /etc/rook.backup
   ```

     Once done, close the SSH connection.

3. Clean up the previous installation of Pelagia Helm release. For details of Rook cleanup, see
   [Rook documentation: Cleaning up a cluster](https://rook.github.io/docs/rook/latest/Getting-Started/ceph-teardown/).

     1. Delete all deployments in the Pelagia namespace:
        ```bash
        kubectl -n pelagia delete deployment --all
        ```

     2. Delete all deployments, DaemonSets, and jobs from the Rook namespace, if any:
        ```bash
        kubectl -n rook-ceph delete deployment --all
        kubectl -n rook-ceph delete daemonset --all
        kubectl -n rook-ceph delete job --all
        ```

     3. Edit the `CephDeployment`, `CephDeploymentHealth`, and `CephDeploymentSecret` CRs of the
        Pelagia namespace and remove the `finalizer` parameter from the `metadata` section:
        ```bash
        kubectl -n pelagia edit cephdpl
        kubectl -n pelagia edit cephdeploymenthealth
        ```

     4. Edit the `CephCluster`, `CephBlockPool`, `CephClient`, and
        `CephObjectStore` CRs of the Rook namespace and remove the
        `finalizer` parameter from the `metadata` section:
        ```bash
        kubectl -n rook-ceph edit cephclusters
        kubectl -n rook-ceph edit cephblockpools
        kubectl -n rook-ceph edit cephclients
        kubectl -n rook-ceph edit cephobjectstores
        kubectl -n rook-ceph edit cephobjectusers
        ```

     5. Remove Pelagia Helm release:
        ```bash
        helm uninstall <releaseName>
        ```

          where `<releaseName>` is Pelagia Helm release, for example, `pelagia-ceph`.

4. Create the `CephDeployment` CR template and edit the roles of nodes. The entire
   `nodes` spec must contain only one `mon` and one `mgr` role. Save the `CephDeployment`
   template after editing:
   ```yaml
   apiVersion: lcm.mirantis.com/v1alpha1
   kind: CephDeployment
   metadata:
     name: pelagia-ceph
     namespace: pelagia
   spec:
     nodes:
     - name: <nodeName>
       roles:
       - mon
       - mgr
   ```

     Substitute `<nodeName>` with node name of the node where monitor is placed.

5. Install Pelagia Helm release:
   ```bash
   helm upgrade --install pelagia-ceph oci://registry.mirantis.com/pelagia/pelagia-ceph --version <version> -n pelagia --create-namespace
   ```

     where `<version>` is Pelagia Helm chart version, for example, `1.0.0`.

6. Verify that the Pelagia Helm release is deployed:

     1. Inspect the Rook Ceph Operator logs and wait until the orchestration has
        settled:
        ```bash
        kubectl -n rook-ceph logs -l app=rook-ceph-operator
        ```

     2. Verify that the pods in the Rook namespace have `rook-ceph-mon-a`, `rook-ceph-mgr-a`,
        and all the auxiliary pods are up and running, and no `rook-ceph-osd-ID-xxxxxx` are running:
        ```bash
        kubectl -n rook-ceph get pod
        ```

     3. Verify the Ceph state. The output must indicate that one `mon` and one
        `mgr` are running, all Ceph OSDs are down, and all PGs are in the
        `Unknown` state.
        ```bash
        kubectl -n rook-ceph exec -it deploy/rook-ceph-tools -- ceph -s
        ```

        !!! note

            Rook should not start any Ceph OSD daemon because all devices
            belong to the old cluster that has a different `fsid`. To verify the
            Ceph OSD daemons, inspect the `osd-prepare` pods logs:
            ```bash
            kubectl -n rook-ceph logs -l app=rook-ceph-osd-prepare
            ```

7. Connect to the terminal of the `rook-ceph-mon-a` pod:
   ```bash
   kubectl -n rook-ceph exec -it deploy/rook-ceph-mon-a -- bash
   ```

8. Output the `keyring` file and save it for further usage:
   ```bash
   cat /etc/ceph/keyring-store/keyring
   exit
   ```

9. Obtain and save the `nodeName` of `mon-a` for further usage:
    ```bash
    kubectl -n rook-ceph get pod $(kubectl -n rook-ceph get pod \
    -l app=rook-ceph-mon -o jsonpath='{.items[0].metadata.name}') -o jsonpath='{.spec.nodeName}'
    ```

10. Obtain and save the `DEPLOYMENT_CEPH_IMAGE` used in the Ceph cluster for further
    usage:
    ```bash
    kubectl -n pelagia get cm pelagia-lcmconfig -o jsonpath='{.data.DEPLOYMENT_CEPH_IMAGE}'
    ```

11. Stop all deployments in the Pelagia namespace:
    ```bash
    kubectl -n pelagia scale deploy --all --replicas 0
    ```

12. Stop Rook Ceph Operator and scale the deployment replicas to `0`:
    ```bash
    kubectl -n rook-ceph scale deploy rook-ceph-operator --replicas 0
    ```

13. Remove the Rook deployments generated with Rook Operator:
    ```bash
    kubectl -n rook-ceph delete deploy -l app=rook-ceph-mon
    kubectl -n rook-ceph delete deploy -l app=rook-ceph-mgr
    kubectl -n rook-ceph delete deploy -l app=rook-ceph-osd
    kubectl -n rook-ceph delete deploy -l app=rook-ceph-crashcollector
    ```

14. Using the saved `nodeName`, SSH to the host where `rook-ceph-mon-a` in
    the new Kubernetes cluster is placed and perform the following steps:

      1. Remove `/var/lib/rook/mon-a` or copy it to another folder:
         ```bash
         mv /var/lib/rook/mon-a /var/lib/rook/mon-a.new
         ```

      2. Pick a healthy `rook-ceph-mon-ID` directory (`/var/lib/rook.backup/mon-ID`) in the previous backup, copy to
         `/var/lib/rook/mon-a`:
         ```bash
         cp -rp /var/lib/rook.backup/mon-<ID> /var/lib/rook/mon-a
         ```

           Substitute `ID` with any healthy `mon` node ID of the old cluster.

      3. Replace `/var/lib/rook/mon-a/keyring` with the previously saved
         keyring, preserving only the `[mon.]` section. Remove the
         `[client.admin]` section.

      4. Run the `DEPLOYMENT_CEPH_IMAGE` Docker container using the previously saved
         `DEPLOYMENT_CEPH_IMAGE` image:
         ```bash
         docker run -it --rm -v /var/lib/rook:/var/lib/rook <DEPLOYMENT_CEPH_IMAGE> bash
         ```

      5. Inside the container, create `/etc/ceph/ceph.conf` for a stable operation of `ceph-mon`:
         ```bash
         touch /etc/ceph/ceph.conf
         ```

      6. Change the directory to `/var/lib/rook` and edit `monmap` by
         replacing the existing `mon` hosts with the new `mon-a` endpoints:
         ```bash
         cd /var/lib/rook
         rm /var/lib/rook/mon-a/data/store.db/LOCK # Make sure the quorum lock file does not exist
         ceph-mon --extract-monmap monmap --mon-data ./mon-a/data  # Extract monmap from old ceph-mon db and save as monmap
         monmaptool --print monmap  # Print the monmap content, which reflects the old cluster ceph-mon configuration.
         monmaptool --rm a monmap  # Delete `a` from monmap.
         monmaptool --rm b monmap  # Repeat and delete `b` from monmap.
         monmaptool --rm c monmap  # Repeat this pattern until all the old ceph-mons are removed and monmap is empty
         monmaptool --addv a [v2:<nodeIP>:3300,v1:<nodeIP>:6789] monmap   # Replace it with the rook-ceph-mon-a address you obtained from the previous command.
         ceph-mon --inject-monmap monmap --mon-data ./mon-a/data  # Replace monmap in ceph-mon db with our modified version.
         rm monmap
         exit
         ```

           Substitute `<nodeIP>` with the IP address of the current `<nodeName>` node.

      7. Close the SSH connection.

15. Change `fsid` to the original one to run Rook as an old cluster:
    ```bash
    kubectl -n rook-ceph edit secret/rook-ceph-mon
    ```

    !!! note

        The `fsid` is `base64` encoded and must not contain a trailing carriage return. For example:
        ```bash
        echo -n a811f99a-d865-46b7-8f2c-f94c064e4356 | base64  # Replace with the fsid from the old cluster.
        ```

16. Disable authentication:

      1. Open the `rook-config-override` ConfigMap for editing:
         ```bash
         kubectl -n rook-ceph edit cm/rook-config-override
         ```

      2. Add the following content:
         ```yaml
         data:
           config: |
             [global]
             ...
             auth cluster required = none
             auth service required = none
             auth client required = none
             auth supported = none
         ```

17. Start Rook Operator by scaling its deployment replicas to `1`:
    ```bash
    kubectl -n rook-ceph scale deploy rook-ceph-operator --replicas 1
    ```

18. Inspect the Rook Operator logs and wait until the orchestration has settled:
    ```bash
    kubectl -n rook-ceph logs -l app=rook-ceph-operator
    ```

19. Verify that the pods in the `rook-ceph` namespace have the
    `rook-ceph-mon-a`, `rook-ceph-mgr-a`, and all the auxiliary pods are up
    and running, and all `rook-ceph-osd-ID-xxxxxx` greater than zero are
    running:
    ```bash
    kubectl -n rook-ceph get pod
    ```

20. Verify the Ceph state. The output must indicate that one `mon`, one
    `mgr`, and all Ceph OSDs must be up and running and all PGs are either in
    the `Active` or `Degraded` state:
    ```bash
    kubectl -n rook-ceph exec -it deploy/pelagia-ceph-toolbox -- ceph -s
    ```

21. Enter the `pelagia-ceph-toolbox` pod and import the authentication key:
    ```bash
    kubectl -n rook-ceph exec -it deploy/pelagia-ceph-toolbox -- bash
    vi key
    [paste keyring content saved before, preserving only `[client admin]` section]
    ceph auth import -i key
    rm key
    exit
    ```

22. Stop Rook Ceph Operator by scaling the deployment to `0` replicas:
    ```bash
    kubectl -n rook-ceph scale deploy rook-ceph-operator --replicas 0
    ```

23. Re-enable authentication:

      1. Open the `rook-config-override` ConfigMap for editing:
         ```bash
         kubectl -n rook-ceph edit cm/rook-config-override
         ```

      2. Remove the following content:
         ```yaml
         data:
           config: |
             [global]
             ...
             auth cluster required = none
             auth service required = none
             auth client required = none
             auth supported = none
         ```

24. Remove all Rook deployments generated with Rook Ceph Operator:
    ```bash
    kubectl -n rook-ceph delete deploy -l app=rook-ceph-mon
    kubectl -n rook-ceph delete deploy -l app=rook-ceph-mgr
    kubectl -n rook-ceph delete deploy -l app=rook-ceph-osd
    kubectl -n rook-ceph delete deploy -l app=rook-ceph-crashcollector
    ```

25. Start Pelagia Controllers by scaling its deployment replicas to `1`:
    ```bash
    kubectl -n pelagia scale deployment --all --replicas 1
    ```

26. Start Rook Ceph Operator by scaling its deployment replicas to `1`:
    ```bash
    kubectl -n rook-ceph scale deploy rook-ceph-operator --replicas 1
    ```

27. Inspect the Rook Ceph Operator logs and wait until the orchestration has settled:
    ```bash
    kubectl -n rook-ceph logs -l app=rook-ceph-operator
    ```

28. Verify that the pods in the Rook namespace have the
    `rook-ceph-mon-a`, `rook-ceph-mgr-a`, and all the auxiliary pods are up
    and running, and all `rook-ceph-osd-ID-xxxxxx` greater than zero are
    running:
    ```bash
    kubectl -n rook-ceph get pod
    ```

29. Verify the Ceph state. The output must indicate that one `mon`, one
    `mgr`, and all Ceph OSDs must be up and running and the overall stored
    data size equals to the old cluster data size.
    ```bash
    kubectl -n rook-ceph exec -it deploy/pelagia-ceph-toolbox -- ceph -s
    ```

30. Edit the `CephDeployment` CR and add two more `mon` and `mgr` roles to the
    corresponding nodes:
    ```bash
    kubectl -n pelagia edit cephdpl
    ```

31. Inspect the Rook namespace and wait until all Ceph Monitors are in the
    `Running` state:
    ```bash
    kubectl -n rook-ceph get pod -l app=rook-ceph-mon
    ```

32. Verify the Ceph state. The output must indicate that three `mon` (three in
    quorum), one `mgr`, and all Ceph OSDs must be up and running and the
    overall stored data size equals to the old cluster data size.
    ```bash
    kubectl -n rook-ceph exec -it deploy/pelagia-ceph-toolbox -- ceph -s
    ```

Once done, the data from the failed or removed Ceph cluster is restored and
ready to use.
