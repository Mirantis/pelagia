<a id="s3-create-user"></a>

# Create Ceph Object Storage users

Ceph Object Storage users can create Amazon S3 buckets and bucket policies that
grant access to other users.

This section describes how to create two Ceph Object Storage users and
configure their S3 credentials.

## Create and configure Ceph Object Storage users

1. Open the `CephDeployment` custom resource for editing:
   ```bash
   kubectl -n pelagia edit cephdpl
   ```

2. In the `spec.objectStorage.rgw` section, add new Ceph Object Storage users.
   For example:
   ```yaml
   spec:
     objectStorage:
       rgw:
         objectUsers:
         - name: user-b
           displayName: user-a
           capabilities:
             bucket: "*"
             user: read
         - name: user-t
           displayName: user-t
           capabilities:
             bucket: "*"
             user: read
   ```

3. Verify that `rgwUserSecrets` are created for both users:
   ```bash
   kubectl -n pelagia get cephdeploymentsecret -o yaml
   ```

     Example of a positive system response:
     ```yaml
     status:
       secretInfo:
         rgwUserSecrets:
         - name: user-a
           secretName: <user-aCredSecretName>
           secretNamespace: <user-aCredSecretNamespace>
         - name: user-t
           secretName: <user-tCredSecretName>
           secretNamespace: <user-tCredSecretNamespace>
     ```

4. Obtain S3 user credentials from the cluster secrets. Specify an access key and a secret key for both users:
   ```bash
   kubectl -n <user-aCredSecretNamespace> get secret <user-aCredSecretName> -o jsonpath='{.data.AccessKey}' | base64 -d
   kubectl -n <user-aCredSecretNamespace> get secret <user-aCredSecretName> -o jsonpath='{.data.SecretKey}' | base64 -d
   kubectl -n <user-tCredSecretNamespace> get secret <user-tCredSecretName> -o jsonpath='{.data.AccessKey}' | base64 -d
   kubectl -n <user-tCredSecretNamespace> get secret <user-tCredSecretName> -o jsonpath='{.data.SecretKey}' | base64 -d
   ```

     Substitute the corresponding `secretNamespace` and `secretName` for both
     users.

5. Obtain Ceph Object Storage public endpoint from the  `CephDeploymentHealth` status:
   ```bash
   kubectl -n pelagia get cephdeploymenthealth -o yaml | grep publicEndpoint
   ```

     Example of a positive system response:
     ```bash
     publicEndpoint: https://object-storage.just.example.com
     ```

6. Obtain the CA certificate to use an HTTPS endpoint:
   ```bash
   kubectl -n rook-ceph get secret $(kubectl -n rook-ceph get ingress -o jsonpath='{.items[0].spec.tls[0].secretName}{"\n"}') -o jsonpath='{.data.ca\.crt}' | base64 -d; echo
   ```

     Save the output to `ca.crt`.
