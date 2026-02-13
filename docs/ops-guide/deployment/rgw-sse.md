<a id="rgw-sse-mira"></a>

# Configure Ceph Object Storage server-side encryption

{% include "../../snippets/techpreview.md" %}

!!! note

    Pelagia supports Ceph Object Storage storage-side encryption for the clusters
    with Rockoon installed because Pelagia uses OpenStack Barbican KMS. Other KMS types
    are not supported yet.

When you use Ceph Object Storage server-side encryption (SSE),
unencrypted data sent over HTTPS is stored encrypted by the Ceph Object Gateway
in the Ceph cluster. The current implementation integrates OpenStack Barbican as a key
management service.

The Object Storage SSE feature is enabled by default in Pelagia with OpenStack Barbican.
To use object storage SSE, the AWS CLI S3 client is used.

**To use object storage server-side encryption:**

1. Create Amazon Elastic Compute Cloud (EC2) credentials:
   ```bash
   openstack ec2 credentials create
   ```

2. Configure AWS CLI with `access` and `secret` created in the previous
   step:
   ```bash
   aws configure
   ```

3. Create a secret key in OpenStack Barbican KMS:
   ```bash
   openstack secret order create --name <name> \
       --algorithm <algorithm> \
       --mode <mode> \
       --bit-length 256 \
       --payload-content-type=<payload-content-type> key
   ```

     Substitute the parameters enclosed in angle brackets:

     * `<name>` - human-friendly name.
     * `<algorithm>` - algorithm to use with the requested key. For example,
       `aes`.
     * `<mode>` - algorithm mode to use with the requested key. For example,
       `ctr`.
     * `<payload-content-type>` - type/format of the secret to generate. For
       example, `application/octet-stream`.

4. Verify that the key has been created:
   ```bash
   openstack secret order get <order-href>
   ```

     Substitute `<order-href>` with the corresponding value from the output of the previous command.

5. Specify the `ceph-rgw` user in the Barbican secret Access Control List (ACL):

     1. Obtain the list of `ceph-rgw` users:
        ```bash
        openstack user list --domain service  | grep ceph-rgw
        ```

          Example output:
          ```bash
          | c63b70134e0845a2b13c3f947880f66a | ceph-rgwZ6ycK3dY         |
          ```

          In the output, capture the first value as the `<user-id>`,
          which is `c63b70134e0845a2b13c3f947880f66a` in the above
          example.

     2. Specify the `ceph-rgw` user in the Barbican ACL:
        ```bash
        openstack acl user add --user <user-id> <secret-href>
        ```

          Substitute `<user-id>` with the corresponding value from the output of
          the previous command and `<secret-href>` with the corresponding value
          obtained in step 3.

6. Create an S3 bucket:
   ```bash
   aws --endpoint-url <rgw-endpoint-url> \
       --ca-bundle <ca-bundle> s3api create-bucket \
       --bucket <bucket-name>
   ```

     Substitute the parameters enclosed in angle brackets:

     * `<rgw-endpoint-url>` - Ceph Object Gateway endpoint DNS name
     * `<ca-bundle>` - CA Certificate Bundle
     * `<bucket-name>` - human-friendly bucket name

7. Upload a file using object storage SSE:
   ```bash
   aws --endpoint-url <rgw-endpoint-url> \
       --ca-bundle <ca-bundle> \
       s3 cp <path-to-file> "s3://<bucket-name>/<filename>" \
       --sse aws:kms \
       --sse-kms-key-id <key-id>
   ```

     Substitute the parameters enclosed in angle brackets:

     * `<path-to-file>` - path to the file that you want to upload
     * `<filename>` - name under which the uploaded file will be stored
       in the bucket
     * `<key-id>` - Barbican secret key ID

8. Select one of the following options to download the file:

     * Download the file using a key:
       ```bash
       aws --endpoint-url <rgw-endpoint-url> \
           --ca-bundle <ca-bundle> \
           s3 cp "s3://<bucket-name>/<filename>" <path-to-output-file> \
           --sse aws:kms \
           --sse-kms-key-id <key-id>
       ```

         Substitute `<path-to-output-file>` with the path to the file you want to download.

     * Download the file without a key:
       ```bash
       aws --endpoint-url <rgw-endpoint-url> \
           --ca-bundle <ca-bundle> \
           s3 cp "s3://<bucket-name>/<filename>" <output-filename>
       ```
