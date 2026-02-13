<a id="bucket-policy-openstack"></a>

# Set a bucket policy for OpenStack users

The following procedure illustrates the process of setting a bucket policy for
a bucket between two OpenStack users deployed by Rockoon.

Due to specifics of the Ceph integration with OpenStack, you should configure the bucket policy
for OpenStack users indirectly through setting permissions for corresponding OpenStack projects.

For illustration purposes, we use the following names in the procedure:

- `test01` for the bucket
- `user-a`, `user-t` for the OpenStack users
- `project-a`, `project-t` for the OpenStack projects

## Configure an Amazon S3 bucket policy for OpenStack users

1. Specify the `rookConfig` section of the `CephDeployment` custom resource:
   ```yaml
   spec:
     rookConfig:
       rgw keystone implicit tenants: "swift"
   ```

2. Prepare the Ceph Object Storage similarly to the procedure described in
   [Create Ceph Object Storage users](./s3-create-user.md#s3-create-user).

3. Create two OpenStack projects:
   ```bash
   openstack project create project-a
   openstack project create project-t
   ```

     Example of system response:
     ```bash
     +-------------+----------------------------------+
     | Field       | Value                            |
     +-------------+----------------------------------+
     | description |                                  |
     | domain_id   | default                          |
     | enabled     | True                             |
     | id          | faf957b776874a2e80384cb882ebf6ab |
     | is_domain   | False                            |
     | name        | project-a                         |
     | options     | {}                               |
     | parent_id   | default                          |
     | tags        | []                               |
     +-------------+----------------------------------+
     ```

     You can also use existing projects. Save the ID of each project for the bucket policy specification.

    !!! note

         For details how to access OpenStack CLI, refer [Rockoon documentation: Access OpenStack](https://mirantis.github.io/rockoon/ops/openstack/getting-access/).

4. Create an OpenStack user for each project:
   ```bash
   openstack user create user-a --project project-a
   openstack user create user-t --project project-t
   ```

     Example of system response:
     ```bash
     +---------------------+----------------------------------+
     | Field               | Value                            |
     +---------------------+----------------------------------+
     | default_project_id  | faf957b776874a2e80384cb882ebf6ab |
     | domain_id           | default                          |
     | enabled             | True                             |
     | id                  | cc2607dc383e4494948d68eeb556f03b |
     | name                | user-a                            |
     | options             | {}                               |
     | password_expires_at | None                             |
     +---------------------+----------------------------------+
     ```

     You can also use existing project users.

5. Assign the `member` role to the OpenStack users:
   ```bash
   openstack role add member --user user-a --project project-a
   openstack role add member --user user-t --project project-t
   ```

6. Verify that the OpenStack users have obtained the `member` roles paying attention to the role IDs:
   ```bash
   openstack role show member
   ```

     Example of system response:
     ```bash
     +-------------+----------------------------------+
     | Field       | Value                            |
     +-------------+----------------------------------+
     | description | None                             |
     | domain_id   | None                             |
     | id          | 8f0ce4f6cd61499c809d6169b2b5bd93 |
     | name        | member                           |
     | options     | {'immutable': True}              |
     +-------------+----------------------------------+
     ```

7. List the role assignments for the `user-a` and `user-t`:
   ```bash
   openstack role assignment list --user user-a --project project-a
   openstack role assignment list --user user-t --project project-t
   ```

     Example of system response:
     ```bash
     +----------------------------------+----------------------------------+-------+----------------------------------+--------+--------+-----------+
     | Role                             | User                             | Group | Project                          | Domain | System | Inherited |
     +----------------------------------+----------------------------------+-------+----------------------------------+--------+--------+-----------+
     | 8f0ce4f6cd61499c809d6169b2b5bd93 | cc2607dc383e4494948d68eeb556f03b |       | faf957b776874a2e80384cb882ebf6ab |        |        | False     |
     +----------------------------------+----------------------------------+-------+----------------------------------+--------+--------+-----------+
     ```

8. Create Amazon EC2 credentials for `user-a` and `user-t`:
   ```bash
   openstack ec2 credentials create --user user-a --project project-a
   openstack ec2 credentials create --user user-t --project project-t
   ```

     Example of system response:
     ```bash
     +------------+----------------------------------------------------------------------------------------------------------------------------------------------------------------+
     | Field      | Value                                                                                                                                                          |
     +------------+----------------------------------------------------------------------------------------------------------------------------------------------------------------+
     | access     | d03971aedc2442dd9a79b3b409c32046                                                                                                                               |
     | links      | {'self': 'http://keystone-api.openstack.svc.cluster.local:5000/v3/users/cc2607dc383e4494948d68eeb556f03b/credentials/OS-EC2/d03971aedc2442dd9a79b3b409c32046'} |
     | project_id | faf957b776874a2e80384cb882ebf6ab                                                                                                                               |
     | secret     | 0a9fd8d9e0d24aecacd6e75951154d0f                                                                                                                               |
     | trust_id   | None                                                                                                                                                           |
     | user_id    | cc2607dc383e4494948d68eeb556f03b                                                                                                                               |
     +------------+----------------------------------------------------------------------------------------------------------------------------------------------------------------+
     ```

     Obtain the values from the `access` and `secret` fields to connect with Ceph Object Storage
     through the `s3cmd` tool.

    !!! note

        The `s3cmd` is a free command-line tool for uploading, retrieving, and managing data in Amazon S3 and other
        cloud storage service providers that use the S3 protocol. You can download the `s3cmd` tool from
        [Amazon S3 tools: Download s3cmd](https://s3tools.org/download).

9. Create bucket users and configure a bucket policy for the `project-t`
   OpenStack project similarly to the procedure described in
   [Set a bucket policy for a Ceph Object Storage user](./bucket-policy-s3.md#set-bucket-policy-for-ceph-object-storage-user).
   Ceph integration does not allow providing permissions for OpenStack users
   directly. Therefore, you need to set permissions for the project that
   corresponds to the user:
   ```json
   {
     "Version": "2012-10-17",
     "Id": "S3Policy1",
     "Statement": [
       {
        "Sid": "BucketAllow",
        "Effect": "Allow",
        "Principal": {
          "AWS": ["arn:aws:iam::<PROJECT-T_ID>:root"]
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

## Ceph Object Storage bucket policy examples

You can configure different bucket policies for various situations. See
examples below.

### Provide access to a bucket from one OpenStack project to another

```json
{
  "Version": "2012-10-17",
  "Id": "S3Policy1",
  "Statement": [
    {
     "Sid": "BucketAllow",
     "Effect": "Allow",
     "Principal": {
       "AWS": ["arn:aws:iam::<osProjectId>:root"]
     },
     "Action": [
       "s3:ListBucket",
       "s3:PutObject",
       "s3:GetObject"
     ],
     "Resource": [
       "arn:aws:s3:::<bucketName>",
       "arn:aws:s3:::<bucketName>/*"
     ]
    }
  ]
}
```

Substitute the following parameters:

* `<osProjectId>` - the target OpenStack project ID
* `<bucketName>` - the target bucket name where the policy will be set

### Provide access to a bucket from a Ceph RGW user to an OpenStack project

```json
{
  "Version": "2012-10-17",
  "Id": "S3Policy1",
  "Statement": [
    {
     "Sid": "BucketAllow",
     "Effect": "Allow",
     "Principal": {
       "AWS": ["arn:aws:iam::<osProjectId>:root"]
     },
     "Action": [
       "s3:ListBucket",
       "s3:PutObject",
       "s3:GetObject"
     ],
     "Resource": [
       "arn:aws:s3:::<bucketName>",
       "arn:aws:s3:::<bucketName>/*"
     ]
    }
  ]
}
```

Substitute the following parameters:

* `<osProjectId>` - the target OpenStack project ID
* `<bucketName>` - the target bucket name where policy will be set

### Provide access to a bucket from an OpenStack user to a Ceph Object Storage user

```json
{
  "Version": "2012-10-17",
  "Id": "S3Policy1",
  "Statement": [
    {
     "Sid": "BucketAllow",
     "Effect": "Allow",
     "Principal": {
       "AWS": ["arn:aws:iam:::user/<userName>"]
     },
     "Action": [
       "s3:ListBucket",
       "s3:PutObject",
       "s3:GetObject"
     ],
     "Resource": [
       "arn:aws:s3:::<bucketName>",
       "arn:aws:s3:::<bucketName>/*"
     ]
    }
  ]
}
```

Substitute the following parameters:

* `<userName>` - the target Ceph Object Storage User name
* `<bucketName>` - the target bucket name where policy will be set

### Provide access to a bucket from one Ceph Object Storage user to another

```json
{
  "Version": "2012-10-17",
  "Id": "S3Policy1",
  "Statement": [
    {
     "Sid": "BucketAllow",
     "Effect": "Allow",
     "Principal": {
       "AWS": ["arn:aws:iam:::user/<userName>"]
     },
     "Action": [
       "s3:ListBucket",
       "s3:PutObject",
       "s3:GetObject"
     ],
     "Resource": [
       "arn:aws:s3:::<bucketName>",
       "arn:aws:s3:::<bucketName>/*"
     ]
    }
  ]
}
```

Substitute the following parameters:

* `<userName>` - the target Ceph Object Storage user name
* `<bucketName>` - the target bucket name where policy will be set

!!! info "See also"

    * [AWS S3: Bucket policy examples](https://docs.aws.amazon.com/AmazonS3/latest/userguide/example-bucket-policies.html)
    * [Ceph documentation: Bucket policies](https://docs.ceph.com/en/latest/radosgw/bucketpolicy/)
