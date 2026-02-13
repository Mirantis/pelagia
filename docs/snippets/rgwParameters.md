- `name` - Required. Ceph Object Storage instance name.
- `dataPool` - Required if `zone.name` is not specified. Mutually exclusive with
  `zone`. Must be used together with `metadataPool`.

     Object storage data pool spec that must only contain `replicated` or
     `erasureCoded`, `deviceClass` and `failureDomain` parameters. The `failureDomain`
     parameter may be set to `host`, `rack`, `room`, or `datacenter`,
     defining the failure domain across which the data will be spread. The
     `deviceClass` must be explicitly defined. For `dataPool`, We recommend
     using an `erasureCoded` pool.

     ```yaml
     spec:
        objectStorage:
          rgw:
            dataPool:
              deviceClass: hdd
              failureDomain: host
              erasureCoded:
                codingChunks: 1
                dataChunks: 2
     ```

- `metadataPool` - Required if `zone.name` is not specified. Mutually exclusive with `zone`. Must be used together with `dataPool`.

    Object storage metadata pool spec that must only contain `replicated`, `deviceClass` and
    `failureDomain` parameters. The `failureDomain` parameter may be set to
    `host`, `rack`, `room`, or `datacenter`, defining the failure domain
    across which the data will be spread. The `deviceClass` must be explicitly
    defined. Can use only `replicated` settings. For example:

    ```yaml
    spec:
       objectStorage:
         rgw:
           metadataPool:
             deviceClass: hdd
             failureDomain: host
             replicated:
               size: 3
    ```

    where `replicated.size` is the number of full copies of data on
    multiple nodes.

    !!! warning

        When using the non-recommended Ceph pools `replicated.size` of
        less than `3`, Ceph OSD removal cannot be performed. The minimal replica
        size equals a rounded up half of the specified `replicated.size`.

        For example, if `replicated.size` is `2`, the minimal replica size is
        `1`, and if `replicated.size` is `3`, then the minimal replica size
        is `2`. The replica size of `1` allows Ceph having PGs with only one
        Ceph OSD in the `acting` state, which may cause a `PG_TOO_DEGRADED`
        health warning that blocks Ceph OSD removal. We recommend setting
        `replicated.size` to `3` for each Ceph pool.

- `gateway` - Required. The gateway settings corresponding to the `rgw` daemon
  settings. Includes the following parameters:

    - `port` - the port on which the Ceph RGW service will be listening on
      HTTP.
    - `securePort` - the port on which the Ceph RGW service will be
      listening on HTTPS.
    - `instances` - the number of pods in the Ceph RGW ReplicaSet. If
      `allNodes` is set to `true`, a DaemonSet is created instead.

        !!! note

            We recommend using 3 instances for Ceph Object Storage.

    - `allNodes` - defines whether to start the Ceph RGW pods as a
      DaemonSet on all nodes. The `instances` parameter is ignored if
      `allNodes` is set to `true`.
    - `splitDaemonForMultisiteTrafficSync` - Optional. For multisite setup defines
      whether to split RGW daemon on daemon responsible for sync between zones and daemon
      for serving clients request.
    - `rgwSyncPort` - Optional. Port the rgw multisite traffic service will be listening on (http).
      Has effect only for multisite configuration.
    - `resources` - Optional. Represents Kubernetes resource requirements for Ceph RGW pods. For details
      see: [Kubernetes docs: Resource Management for Pods and Containers](https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/).
    - `externalRgwEndpoint` - Required for external Ceph cluster Setup. Represents external RGW Endpoint to use,
      only when external Ceph cluster is used. Contains the following parameters:

        - `ip` - represents the IP address of RGW endpoint.
        - `hostname` - represents the DNS-addressable hostname of RGW endpoint.
          This field will be preferred over IP if both are given.

            ```yaml
            spec:
              objectStorage:
                rgw:
                  gateway:
                    allNodes: false
                    instances: 3
                    port: 80
                    securePort: 8443
            ```

- `preservePoolsOnDelete` - Optional. Defines whether to delete the data and metadata pools in
  the `rgw` section if the Object Storage is deleted. Set this parameter
  to `true` if you need to store data even if the object storage is
  deleted. However, we recommend setting this parameter to `false`.

- `objectUsers` and `buckets` - Optional. To create new Ceph RGW resources, such as buckets or users,
  specify the following keys. Ceph Controller will automatically create
  the specified object storage users and buckets in the Ceph cluster.

    - `objectUsers` - a list of user specifications to create for object
      storage. Contains the following fields:

        - `name` - a user name to create.
        - `displayName` - the Ceph user name to display.
        - `capabilities` - user capabilities:

            - `user` - admin capabilities to read/write Ceph Object Store
              users.
            - `bucket` - admin capabilities to read/write Ceph Object Store
              buckets.
            - `metadata` - admin capabilities to read/write Ceph Object Store
              metadata.
            - `usage` - admin capabilities to read/write Ceph Object Store
              usage.
            - `zone` - admin capabilities to read/write Ceph Object Store
              zones.

            The available options are `*`, `read`, `write`, `read, write`. For details, see
            [Ceph documentation: Add/remove admin capabilities](https://docs.ceph.com/en/latest/radosgw/admin/?#add-remove-admin-capabilities).

        - `quotas` - user quotas:

            - `maxBuckets` - the maximum bucket limit for the Ceph user.
              Integer, for example, `10`.
            - `maxSize` - the maximum size limit of all objects across all the
              buckets of a user. String size, for example, `10G`.
            - `maxObjects` - the maximum number of objects across all buckets
              of a user. Integer, for example, `10`.

        ```yaml
        spec:
          objectStorage:
            rgw:
              objectUsers:
              - name: test-user
                 displayName: test-user
                 capabilities:
                   bucket: '*'
                   metadata: read
                   user: read
                 quotas:
                   maxBuckets: 10
                   maxSize: 10G
        ```

    - `buckets` - a list of strings that contain bucket names to create
      for object storage.

- `zone` - Required if `dataPool` and `metadataPool` are not specified. Mutually exclusive with these parameters.

    Defines the Ceph Multisite zone where the object storage must be placed.
    Includes the `name` parameter that must be set to one of the `zones`
    items:

      ```yaml
      spec:
        objectStorage:
          multisite:
            zones:
            - name: master-zone
              ...
          rgw:
            zone:
              name: master-zone
      ```

- `SSLCert` - Optional. Custom TLS certificate parameters used to access the Ceph RGW endpoint. If not specified, a self-signed certificate will be generated.

    ```yaml
    spec:
      objectStorage:
        rgw:
          SSLCert:
            cacert: |
              -----BEGIN CERTIFICATE-----
              ca-certificate here
              -----END CERTIFICATE-----
            tlsCert: |
              -----BEGIN CERTIFICATE-----
              private TLS certificate here
              -----END CERTIFICATE-----
            tlsKey: |
              -----BEGIN RSA PRIVATE KEY-----
              private TLS key here
              -----END RSA PRIVATE KEY-----
    ```

- `SSLCertInRef` - Optional. Flag to determine that a TLS
    certificate for accessing the Ceph RGW endpoint is used but not exposed
    in `spec`. For example:

      ```yaml
      spec:
        objectStorage:
          rgw:
            SSLCertInRef: true
      ```

      The operator must manually provide TLS configuration using the
      `rgw-ssl-certificate` secret in the `rook-ceph` namespace of the
      managed cluster. The secret object must have the following structure:

      ```yaml
      data:
        cacert: <base64encodedCaCertificate>
        cert: <base64encodedCertificate>
      ```

      When removing an already existing `SSLCert` block, no additional actions
      are required, because this block uses the same `rgw-ssl-certificate` secret
      in the `rook-ceph` namespace.

      When adding a new secret directly without exposing it in `spec`, the following
      rules apply:

      - `cert` - base64 representation of a file with the server TLS key,
        server TLS cert, and CA certificate.
      - `cacert` - base64 representation of a CA certificate only.
