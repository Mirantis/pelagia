<a id="ceph-rgw-user"></a>

# Manage Ceph Object Storage users

The `CephDeployment` custom resource (CR) allows managing custom Ceph Object Storage
users. This section describes how to create, access, and remove Ceph Object
Storage users.

For all supported parameters of Ceph Object Storage users, refer to
[CephDeployment: Ceph Object Storage parameters](https://mirantis.github.io/pelagia/architecture/custom-resources/cephdeployment#rgw).

## Create a Ceph Object Storage user

1. Edit the `CephDeployment` CR by adding a new Ceph Object Storage user to
   the `spec` section:
   ```bash
   kubectl -n pelagia edit cephdpl
   ```

     Example of adding the Ceph Object Storage user `user-a`:
     ```yaml
     spec:
       objectStorage:
         rgw:
           name: rgw-store
           objectUsers:
           - capabilities:
               bucket: '*'
               metadata: read
               user: read
             displayName: user-a
             name: userA
             quotas:
               maxBuckets: 10
               maxSize: 10G
     ```

2. Wait for the created user to become ready in the `CephDeploymentHealth` status:
   ```bash
   kubectl -n pelagia get cephdeploymenthealth -o yaml
   ```

     Example output:
     ```yaml
     status:
       healthReport:
         rookCephObjects:
           objectStorage:
             cephObjectStoreUsers:
               user-a:
                 info:
                   secretName: rook-ceph-object-user-rgw-store-user-a
                 observedGeneration: 1
                 phase: Ready
     ```

## Access data using a Ceph Object Storage user

1. Using the `CephDeploymentSecret` status, obtain `secretInfo` with the Ceph
   user credentials:
   ```bash
   kubectl -n pelagia get cephdeploymentsecret -o yaml
   ```

     Example output:
     ```yaml
     status:
       secretInfo:
         rgwUserSecrets:
         - name: user-a
           secretName: rook-ceph-object-user-<objstoreName>-<username>
           secretNamespace: rook-ceph
     ```

     Substitute `<objstoreName>` with a Ceph Object Storage name and `<username>` with a Ceph Object Storage user name.

2. Use `secretName` and `secretNamespace` to access the Ceph Object
   Storage user credentials. The secret contains Amazon S3 access and secret
   keys.

     * To obtain the user S3 access key:
       ```bash
       kubectl -n <secretNamespace> get secret <secretName> -o jsonpath='{.data.AccessKey}' | base64 -d; echo
       ```

         Substitute the following parameters in the commands above and below:

         * `<secretNamespace>` with `secretNamespace` from the previous step
         * `<secretName>` with `secretName` from the previous step

         Example output:
         ```bash
         D49G060HQ86U5COBTJ13
         ```

     * To obtain the user S3 secret key:
       ```bash
       kubectl -n <secretNamespace> get secret <secretName> -o jsonpath='{.data.SecretKey}' | base64 -d; echo
       ```

         Example output:
         ```bash
         bpuYqIieKvzxl6nzN0sd7L06H40kZGXNStD4UNda
         ```

3. Configure the S3 client with the access and secret keys of the created user.
   You can access the S3 client using various tools such as **s3cmd** or **awscli**.

## Remove a Ceph Object Storage user

1. Edit the `CephDeployment` CR by removing the required Ceph
   Object Storage user from `spec.objectStorage.rgw.objectUsers`:
   ```bash
   kubectl -n pelagia edit cephdpl
   ```

2. Wait for the removed user to be removed from the `CephDeploymentHealth`
   status in `status.healthReport.rookCephObjects.objectStorageStatus.cephObjectStoreUsers`:
   ```bash
   kubectl -n pelagia get cephdeploymenthealth -o yaml
   ```
