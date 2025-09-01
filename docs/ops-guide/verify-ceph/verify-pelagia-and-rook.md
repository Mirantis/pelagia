<a id="verify-ceph-controller-rook"></a>

# Verify Pelagia Controllers and Rook Ceph Operator

The starting point for Pelagia, Rook and Ceph troubleshooting is the Pelagia Controllers and
Rook Ceph Operator logs. Once you locate the component that causes issues,
verify the logs of the related pod. This section describes how to verify
Pelagia Controllers and Rook objects of a Ceph cluster.

## Verify Pelagia and Rook

1. Verify that the status of each pod in the Pelagia and Rook namespaces is `Running`:

     * For `pelagia`:
       ```bash
       kubectl -n pelagia get pod
       ```
     * For `rook-ceph`:
       ```bash
       kubectl -n rook-ceph get pod
       ```

2. Verify Pelagia Deployment Controller that prepares the configuration for Rook to deploy
   the Ceph cluster, which is managed using the `CephDeployment` custom resource (CR).

     1. List the pods:
        ```bash
        kubectl -n pelagia get pods
        ```

     2. Verify the logs of the required pod:
        ```bash
        kubectl -n pelagia logs <pelagia-deployment-controller-pod-name>
        ```

     3. Verify the configuration:
        ```bash
        kubectl -n pelagia get cephdpl -o yaml
        ```

     If Rook cannot finish the deployment, verify the Rook Operator logs as
     described in the following step.

3. Verify the Rook Ceph Operator logs. Rook deploys a Ceph cluster based on custom
   resources created by the Pelagia Deployment Controller, such as `cephblockpools`, `cephclients`,
   `cephcluster`, and so on. Rook Ceph Operator logs contain details about component
   orchestration.

     1. Verify the Rook Ceph Operator logs:
        ```bash
        kubectl -n rook-ceph logs -l app=rook-ceph-operator
        ```

     2. Verify the `CephCluster` configuration:

        !!! note

            In Pelagia, `CephDeployment` manages the `CephCluster` CR. Use the `CephCluster` CR only for verification
            and do not modify it manually.

          ```bash
          kubectl get cephcluster -n rook-ceph -o yaml
          ```

     For details about the Ceph cluster status and to get access to CLI tools,
     connect to the `pelagia-ceph-toolbox` pod as described in the following step.

4. Verify the `pelagia-ceph-toolbox` pod:

     1. Execute the `pelagia-ceph-toolbox` pod:
        ```bash
        kubectl -n rook-ceph exec -it deploy/pelagia-ceph-toolbox -- bash
        ```
     2. Verify that CLI commands can run on the `pelagia-ceph-toolbox` pod:
        ```bash
        ceph -s
        ```

5. Verify hardware:

     1. Through the `pelagia-ceph-toolbox` pod, obtain the required device in your
        cluster:
        ```bash
        ceph osd tree
        ```
     2. Enter all Ceph OSD pods in the `rook-ceph` namespace one by one:
        ```bash
        kubectl exec -it -n rook-ceph <osd-pod-name> bash
        ```
     3. Verify that the `ceph-volume` tool is available on all pods running on
        the target node:
        ```bash
        ceph-volume lvm list
        ```

6. Verify data access. Ceph volumes can be consumed directly by Kubernetes
   workloads and internally, for example, by OpenStack services. To verify the
   Kubernetes storage:

     1. Verify the available storage classes. The storage classes that are
        automatically managed by Ceph Controller use the
        `rook-ceph.rbd.csi.ceph.com` provisioner.
        ```bash
        kubectl get storageclass
        ```

          Example of system response:
          ```bash
          NAME                            PROVISIONER                    RECLAIMPOLICY   VOLUMEBINDINGMODE      ALLOWVOLUMEEXPANSION   AGE
          kubernetes-ssd (default)        rook-ceph.rbd.csi.ceph.com     Delete          Immediate              false                  55m
          ```

     2. Verify that volumes are properly connected to the Pod:

          1. Obtain the list of volumes in all namespaces or use a particular one:
             ```bash
             kubectl get persistentvolumeclaims -A
             ```

               Example of system response:
               ```bash
               NAMESPACE   NAME       STATUS   VOLUME    CAPACITY   ACCESS MODES   STORAGECLASS     AGE
               rook-ceph   app-test   Bound    pv-test   1Gi        RWO            kubernetes-ssd   11m
               ```

          2. For each volume, verify the connection. For example:
             ```bash
             kubectl describe pvc app-test -n rook-ceph
             ```

               Example of a positive system response:
               ```bash
               Name:          app-test
               Namespace:     kaas
               StorageClass:  rook-ceph
               Status:        Bound
               Volume:        pv-test
               Labels:        <none>
               Annotations:   pv.kubernetes.io/bind-completed: yes
                              pv.kubernetes.io/bound-by-controller: yes
                              volume.beta.kubernetes.io/storage-provisioner: rook-ceph.rbd.csi.ceph.com
               Finalizers:    [kubernetes.io/pvc-protection]
               Capacity:      1Gi
               Access Modes:  RWO
               VolumeMode:    Filesystem
               Events:        <none>
               ```

               In case of connection issues, inspect the Pod description for the
               volume information:
               ```bash
               kubectl describe pod <crashloopbackoff-pod-name>
               ```

               Example of system response:
               ```bash
               ...
               Events:
                 FirstSeen LastSeen Count From    SubObjectPath Type     Reason           Message
                 --------- -------- ----- ----    ------------- -------- ------           -------
                 1h        1h       3     default-scheduler     Warning  FailedScheduling PersistentVolumeClaim is not bound: "app-test" (repeated 2 times)
                 1h        35s      36    kubelet, 172.17.8.101 Warning  FailedMount      Unable to mount volumes for pod "wordpress-mysql-918363043-50pjr_default(08d14e75-bd99-11e7-bc4c-001c428b9fc8)": timeout expired waiting for volumes to attach/mount for pod "default"/"wordpress-mysql-918363043-50pjr". list of unattached/unmounted volumes=[mysql-persistent-storage]
                 1h        35s      36    kubelet, 172.17.8.101 Warning  FailedSync       Error syncing pod
               ```

     3. Verify that the CSI provisioner plugins started properly and are in
        the `Running` status:
        1. Obtain the list of CSI provisioner plugins:
           ```bash
           kubectl -n rook-ceph get pod -l app=csi-rbdplugin-provisioner
           ```
        2. Verify the logs of the required CSI provisioner:
           ```bash
           kubectl logs -n rook-ceph <csi-provisioner-plugin-name> csi-provisioner
           ```
