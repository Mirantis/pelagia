<a id="ceph-client-manage-ceph-rbd-or-cephfs-clients"></a>

# Manage Ceph RBD or CephFS clients

The `CephDeployment` custom resource (CR) allows managing custom Ceph RADOS Block Device (RBD)
or Ceph File System (CephFS) clients. This section describes how to create,
access, and remove Ceph RBD or CephFS clients.

For all supported parameters of Ceph clients, refer to [Clients parameters](../../../architecture/custom-resources/cephdeployment.md#cephdeployment-clients-parameters).

## Create an RBD or CephFS client

1. Edit the `CephDeployment` CR by adding a new Ceph client to the `spec` section:
   ```bash
   kubectl -n pelagia edit cephdpl
   ```

     Example of adding an RBD client to the `kubernetes-ssd` pool:
     ```yaml
     spec:
       clients:
       - name: rbd-client
         caps:
           mon: allow r, allow command "osd blacklist"
           osd: profile rbd pool=kubernetes-ssd
     ```

     Example of adding a CephFS client to the `cephfs-1` Ceph File System:
     ```yaml
     spec:
       clients:
       - name: cephfs-1-client
         caps:
           mds: allow rw
           mon: allow r, allow command "osd blacklist"
           osd: allow rw tag cephfs data=cephfs-1 metadata=*
     ```

     For details about `caps`, refer to
     [Ceph documentation: Authorization (capabilities)](https://docs.ceph.com/en/latest/rados/operations/user-management/#authorization-capabilities).

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
       `spec.clients` the `CephDeployment` resource, for example, `rbd-client`.

     Example output:
     ```bash
     AQAGHDNjxWYXJhAAjafCn3EtC6KgzgI1x4XDlg==
     ```

3. Using the obtained credentials, create two configuration files on the
   required workloads to connect them with Ceph pools or file systems:

     * `/etc/ceph/ceph.conf`:
       ```bash
       [default]
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

         * `<clientName>` is a client name set in `spec.clients` of the
           `CephDeployment` resource. For example, `rbd-client`.
         * `<cephClientCredentials>` are the client credentials obtained in the
           previous steps. For example, `AQAGHDNjxWYXJhAAjafCn3EtC6KgzgI1x4XDlg==`.

4. If the client `caps` parameters contain `mon: allow r`, verify the
   client access using the following command:
   ```bash
   ceph -n client.<clientName> -s
   ```

## Remove an RBD or CephFS client

1. Edit the `CephDeployment` CR by removing the Ceph client from `spec.clients`:
   ```bash
   kubectl -n pelagia edit cephdpl
   ```

2. Wait for the client to be removed from the `CephDeployment`
   status in `status.healthReport.rookCephObjects.cephClients`:
   ```bash
   kubectl -n pelagia get cephdeploymenthealth -o yaml
   ```
