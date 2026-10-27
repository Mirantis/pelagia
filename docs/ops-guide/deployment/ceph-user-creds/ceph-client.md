---
description: How to manage Ceph RBD and CephFS clients using the CephDeployment CR,
  including creating, accessing, and removing clients.
keywords: pelagia, manage ceph clients, ceph clients, ceph rbd clients, cephdeployment,
  create rbd client, create cephfs client, access data, remove rbd client, remove cephfs client
---

<a id="ceph-client-manage-ceph-rbd-or-cephfs-clients"></a>

# Manage Ceph RBD or CephFS clients

The `CephDeployment` custom resource (CR) allows managing custom Ceph RADOS Block Device (RBD)
or Ceph File System (CephFS) clients. This section describes how to create,
access, and remove Ceph RBD or CephFS clients.

For all supported parameters of Ceph clients, refer to [CephxClients parameters](../../../custom-resources/cephdeployment.md#cephdeployment-cephxclients-parameters).

## Create an RBD or CephFS client

1. Edit the `CephDeployment` CR by adding a new Ceph client to the `spec` section:
   ```bash
   kubectl -n pelagia edit cephdpl
   ```

     Example of adding an RBD client to the `kubernetes-ssd` pool:
     ```yaml
     spec:
       cephxClients:
       - spec:
           name: rbd-client
           caps:
             mon: allow r, allow command "osd blacklist"
             osd: profile rbd pool=kubernetes-ssd
     ```

     Example of adding a CephFS client to the `cephfs-1` Ceph File System:
     ```yaml
     spec:
       cephxClients:
       - spec:
           name: cephfs-1-client
           caps:
             mds: allow rw
             mon: allow r, allow command "osd blacklist"
             osd: allow rw tag cephfs data=cephfs-1 metadata=*
     ```

     For details about `caps`, refer to
     [Ceph documentation: Authorization (capabilities)](https://docs.ceph.com/en/{{ ceph_doc_version }}/rados/operations/user-management/#authorization-capabilities).

2. Wait for created clients to become ready in the `CephDeploymentHealth` CR status:
   ```bash
   kubectl -n pelagia get cephdeploymenthealth -o yaml
   ```

     Example output:
     ```yaml
     status:
       healthReport:
         rookCephObjects:
           cephClients:
             rbd-client:
               info:
                 secretName: rook-ceph-client-rbd-client
               observedGeneration: 1
               phase: Ready
             cephfs-1-client:
               info:
                 secretName: rook-ceph-client-cephfs-1-client
               observedGeneration: 1
               phase: Ready
     ```

## Access data using an RBD or CephFS client

1. Using the `CephDeploymentSecret` status, obtain `secretInfo` with the Ceph
   client credentials:
   ```bash
   kubectl -n pelagia get cephdeploymentsecret -o yaml
   ```

     Example output:
     ```yaml
     status:
       secretInfo:
         clientSecrets:
         - name: client.rbd-client
           secretName: rook-ceph-client-rbd-client
           secretNamespace: rook-ceph
         - name: client.cephfs-1-client
           secretName: rook-ceph-client-cephfs-1-client
           secretNamespace: rook-ceph
     ```

2. Use `secretName` and `secretNamespace` to access the Ceph client credentials:
   ```bash
   kubectl -n <secretNamespace> get secret <secretName> -o jsonpath='{.data.<clientName>}' | base64 -d; echo
   ```

     Substitute the following parameters:

     * `<secretNamespace>` with `secretNamespace` from the previous step;
     * `<secretName>` with `secretName` from the previous step;
     * `<clientName>` with the Ceph RBD or CephFS client name set in
       `spec.cephxClients[].spec.name` of the `CephDeployment` resource, for example, `rbd-client`.

     Example output:
     ```bash
     AQAGHDNjxWYXJhAAjafCn3EtC6KgzgI1x4XDlg==
     ```

3. Using the obtained credentials, create two configuration files on the
   required workloads to connect them with Ceph pools or file systems:

     * `/etc/ceph/ceph.conf`:
       ```bash
       [global]
          mon_host = <mon1IP>:6789,<mon2IP>:6789,...,<monNIP>:6789
       ```

        where `mon_host` are the comma-separated IP addresses with `6789`
        ports of the current Ceph Monitors. For example,
        `10.10.0.145:6789,10.10.0.153:6789,10.10.0.235:6789`.

     * `/etc/ceph/ceph.client.<clientName>.keyring`:
       ```bash
       [client.<clientName>]
           key = <cephClientCredentials>
       ```

         * `<clientName>` is a client name set in `spec.cephxClients[].spec.name` of the
           `CephDeployment` resource. For example, `rbd-client`.
         * `<cephClientCredentials>` are the client credentials obtained in the
           previous steps. For example, `AQAGHDNjxWYXJhAAjafCn3EtC6KgzgI1x4XDlg==`.

4. If the client `caps` parameters contain `mon: allow r`, verify the
   client access using the following command:
   ```bash
   ceph -n client.<clientName> -s
   ```

## Remove an RBD or CephFS client

1. Edit the `CephDeployment` CR by removing the Ceph client from `spec.cephxClients`:
   ```bash
   kubectl -n pelagia edit cephdpl
   ```

2. Wait for the client to be removed from the `CephDeployment`
   status in `status.healthReport.rookCephObjects.cephClients`:
   ```bash
   kubectl -n pelagia get cephdeploymenthealth -o yaml
   ```

<a id="move-default-clients-to-spec"></a>

## Manage the default OpenStack clients using the CephDeployment spec

To take full control of the `CephClient` resources that Pelagia creates
automatically, define them in the `CephDeployment` spec. The following example
illustrates how to manage the default `CephClient` for the OpenStack Nova service.

1. Obtain the current `CephClient` name for the OpenStack Nova service:
   ```bash
   kubectl -n rook-ceph get cephclient -l cephdeployment.lcm.mirantis.com/client-role=nova
   ```

     Example output:
     ```bash
     NAME                PHASE   AGE
     nova-y5ucvajlxscd   Ready   15m
     ```

2. Obtain the spec of this client:
   ```bash
   kubectl get cephclients -n rook-ceph -o yaml nova-y5ucvajlxscd
   ```

     Example output:
     ```yaml
     apiVersion: ceph.rook.io/v1
     kind: CephClient
     metadata:
       creationTimestamp: "2026-10-19T13:58:30Z"
       finalizers:
       - cephclient.ceph.rook.io
       generation: 1
       labels:
         app.kubernetes.io/created-by: pelagia-deployment-controller
         app.kubernetes.io/managed-by: pelagia-deployment-controller
         app.kubernetes.io/part-of: ceph.pelagia.lcm
         cephdeployment.lcm.mirantis.com/client-role: nova
         cephdeployment.lcm.mirantis.com/client-rotation: "0"
       name: nova-y5ucvajlxscd
       namespace: rook-ceph
       resourceVersion: "24182"
       uid: c6e436c7-bd15-440d-b10c-bfc39b4d7f11
     spec:
       caps:
         mon: allow profile rbd
         osd: profile rbd pool=vms-hdd, profile rbd pool=images-hdd, profile rbd pool=volumes-hdd
       name: nova_Y5UCvaJlxScd
       security:
         cephx: {}
     status:
       cephx:
         keyCephVersion: 20.2.4-0
         keyGeneration: 1
       info:
         secretName: rook-ceph-client-nova-y5ucvajlxscd
       observedGeneration: 1
       phase: Ready
     ```

3. Copy the `spec` section as is to the `CephDeployment` spec and assign the related role:
   ```yaml
   spec:
     cephxClients:
     - role: nova
       spec:
         caps:
           mon: allow profile rbd
           osd: profile rbd pool=vms-hdd, profile rbd pool=images-hdd, profile rbd pool=volumes-hdd
         name: nova_Y5UCvaJlxScd
   ```

4. Save the changes and verify `CephClient`:
   ```bash
   kubectl -n rook-ceph get cephclient nova-y5ucvajlxscd -o yaml
   ```

     Example output:
     ```yaml
     apiVersion: ceph.rook.io/v1
     kind: CephClient
     metadata:
       creationTimestamp: "2026-10-19T13:58:30Z"
       finalizers:
       - cephclient.ceph.rook.io
       generation: 1
       labels:
         app.kubernetes.io/created-by: pelagia-deployment-controller
         app.kubernetes.io/managed-by: pelagia-deployment-controller
         app.kubernetes.io/part-of: ceph.pelagia.lcm
         cephdeployment.lcm.mirantis.com/client-latest: "true"
         cephdeployment.lcm.mirantis.com/client-role: nova
       name: nova-y5ucvajlxscd
       namespace: rook-ceph
       resourceVersion: "45362"
       uid: c6e436c7-bd15-440d-b10c-bfc39b4d7f11
     spec:
       caps:
         mon: allow profile rbd
         osd: profile rbd pool=vms-hdd, profile rbd pool=images-hdd, profile rbd pool=volumes-hdd
       name: nova_Y5UCvaJlxScd
       security:
         cephx: {}
     status:
       cephx:
         keyCephVersion: 20.2.4-0
         keyGeneration: 1
       info:
         secretName: rook-ceph-client-nova-y5ucvajlxscd
       observedGeneration: 1
       phase: Ready
     ```

If the `CephDeployment` spec defines no other `CephClient` with the `nova` role,
this `CephClient` now has the `cephdeployment.lcm.mirantis.com/client-latest` label
but not the `cephdeployment.lcm.mirantis.com/client-rotation` label.

!!! note

    `spec.extraOpts.rotateOsClients` applies only to the OpenStack clients that
    Pelagia creates by default. A client defined in the `CephDeployment` spec is
    no longer managed as a default client. To rotate it, add a new item with the
    same `role` to `spec.cephxClients`. For details, see
    [ExtraOpts parameters](../../../custom-resources/cephdeployment.md#cephdeployment-extraopts-parameters).
