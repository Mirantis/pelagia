# Set a bucket policy for a Ceph Object Storage user

Amazon S3 is an object storage service with different access policies. A bucket
policy is a resource-based policy that grants permissions to a bucket and
objects in it. For more details, see
[Amazon S3 documentation: Using bucket policies](https://docs.aws.amazon.com/AmazonS3/latest/userguide/bucket-policies.html).

The following procedure illustrates the process of setting a bucket policy for
a bucket (`test01`) stored in a Ceph Object Storage. The bucket policy
requires at least two users: a bucket owner (`user-a`) and a bucket user
(`user-t`). The bucket owner creates the bucket and sets the policy that
regulates access for the bucket user.

The procedure uses `s3cmd` command-line tool. To configure `s3cmd` for
using it with Ceph Object Storage, please refer to [Configure s3cmd](#s3cmd).

## Configure `s3cmd` command-line tool <a name="s3cmd"></a>

!!! note

    The s3cmd is a free command-line tool and client for uploading,
    retrieving, and managing data in Amazon S3 and other cloud storage service
    providers that use the S3 protocol. You can download the s3cmd CLI tool from
    [Amazon S3 tools: Download s3cmd](https://s3tools.org/download).

Configure the s3cmd client with some s3 credentials, you need to run:
```bash
s3cmd --configure --ca-certs=ca.crt
```

where `ca.crt` is a CA certificate signs Ceph Object Storage public endpoint.

The command will ask to specify the following bucket access parameters:

- `Access Key` - Public part of access credentials. Specify a user access key.
- `Secret Key` - Secret part of access credentials. Specify a user secret key.
- `Default Region` - Region of AWS servers where requests are sent by default. Use the default value.
- `S3 Endpoint` - Connection point to the Ceph Object Storage. Specify the Ceph Object Storage public endpoint.
- `DNS-style bucket+hostname:port template for accessing a bucket` - Bucket location. Specify the Ceph Object Storage public endpoint.
- `Path to GPG program` - Path to the GNU Privacy Guard encryption suite. Use the default value.
- `Use HTTPS protocol` - HTTPS protocol switch. Specify `Yes`.
- `HTTP Proxy server name` - HTTP Proxy server name. Skip this parameter.

When configured correctly, the `s3cmd` tool connects to the Ceph Object Storage.
Save new settings when prompted by the system.

## Configure an Amazon S3 bucket policy


1. Configure the `s3cmd` client with the `user-a` credentials `AccessKey` and `SecretKey`.
2. As `user-a`, create a new bucket `test01`:
   ```bash
   s3cmd mb s3://test01
   ```

     Example of a positive system response:
     ```bash
     Bucket 's3://test01/' created
     ```

3. Upload an object to the bucket:
   ```bash
   touch test.txt
   s3cmd put test.txt s3://test01
   ```

     Example of a positive system response:
     ```bash
     upload: 'test.txt' -> 's3://test01/test.txt'  [1 of 1]
     0 of 0     0% in    0s     0.00 B/s  done
     ```

4. Verify that the object is in the `test01` bucket:
   ```bash
   s3cmd ls s3://test01
   ```

     Example of a positive system response:
     ```bash
     2022-09-02 13:06            0  s3://test01/test.txt
     ```

5. Create the bucket policy file and add bucket CRUD permissions
   for `user-t`:
   ```json
   {
     "Version": "2012-10-17",
     "Id": "S3Policy1",
     "Statement": [
       {
        "Sid": "BucketAllow",
        "Effect": "Allow",
        "Principal": {
          "AWS": ["arn:aws:iam:::user/user-t"]
        },
        "Action": [
          "s3:ListBucket",
          "s3:PutObject",
          "s3:GetObject"
        ],
        "Resource": [
          "arn:aws:s3:::test01",
          "arn:aws:s3:::test01/*"
        ]
       }
     ]
   }
   ```

6. Set the bucket policy for the `test01` bucket:
   ```bash
   s3cmd setpolicy policy.json s3://test01
   ```

     Example of a positive system response:
     ```bash
     s3://test01/: Policy updated
     ```

7. Verify that the `user-t` has access for the `test01` bucket by
   reconfiguring the `s3cmd` client with the `user-t` credentials `AccessKey` and `SecretKey`.
   Verify that the `user-t` can read the bucket `test01` content:
   ```bash
   s3cmd ls s3://test01
   ```

     Example of a positive system response:
     ```bash
     2022-09-02 13:06            0  s3://test01/test.txt
     ```

8. Download the object from the `test01` bucket:
   ```bash
   s3cmd get s3://test01/test.txt check.txt
   ```

     Example of a positive system response:
     ```bash
     download: 's3://test01/test.txt' -> 'check.txt'  [1 of 1]
      0 of 0     0% in    0s     0.00 B/s  done
     ```

9. Upload a new object to the `test01` bucket:
   ```bash
   s3cmd put check.txt s3://test01
   ```

     Example of a positive system response:
     ```bash
     upload: 'check.txt' -> 's3://test01/check.txt'  [1 of 1]
      0 of 0     0% in    0s     0.00 B/s  done
     ```

10. Verify that the object is in the `test01` bucket:
    ```bash
    s3cmd ls s3://test01
    ```

      Example of a positive system response:
      ```bash
      2022-09-02 14:41            0  s3://test01/check.txt
      2022-09-02 13:06            0  s3://test01/test.txt
      ```

11. Verify the new object by reconfiguring the `s3cmd` client with the
    `user-a` credentials:
    ```bash
    s3cmd --configure --ca-certs=ca.crt
    ```

12. List `test01` bucket objects:
    ```bash
    s3cmd ls s3://test01
    ```

      Example of a positive system response:
      ```bash
      2022-09-02 14:41            0  s3://test01/check.txt
      2022-09-02 13:06            0  s3://test01/test.txt
      ```
