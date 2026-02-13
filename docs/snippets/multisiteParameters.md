- `realms` - Required. List of realms to use, represents the realm namespaces. Includes the following parameters:

    - `name` - required, the realm name.
    - `pullEndpoint` - optional, required only when the master zone is in
      a different storage cluster. The endpoint, access key, and system key
      of the system user from the realm to pull from. Includes the
      following parameters:

        - `endpoint` - the endpoint of the master zone in the master zone group.
        - `accessKey` - the access key of the system user from the realm to pull from.
        - `secretKey` - the system key of the system user from the realm to pull from.

- `zoneGroups` - Required. The list of zone groups for realms. Includes the following parameters:

    - `name` - required, the zone group name.
    - `realmName` - required, the realm namespace name to which
      the zone group belongs to.

- `zones` - Required. The list of zones used within one zone group. Includes the following parameters:

    - `name` - required, the zone name.
    - `metadataPool` - required, the settings used to create the Object Storage metadata pools. Must use replication. For details, see description of [Pool parameters](https://mirantis.github.io/pelagia/architecture/custom-resources/cephdeployment/#pools).
    - `dataPool` - required, the settings used to create the Object Storage data pool. Can use replication or erasure coding. For details, see [Pool parameters](https://mirantis.github.io/pelagia/architecture/custom-resources/cephdeployment/#pools).
    - `zoneGroupName` - required, the zone group name.
    - `endpointsForZone` - optional. The list of all endpoints in the zone group.
      If you use ingress proxy for RGW, the list of endpoints must contain that FQDN/IP address to access RGW.
      By default, if no ingress proxy is used, the list of endpoints is set to the IP address of the RGW external service. Endpoints must follow the HTTP URL format.
