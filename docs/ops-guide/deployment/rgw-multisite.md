<a id="enable-multisite-for-ceph=object-storage"></a>
# Enable Multisite for Ceph Object Storage

{% include "../../snippets/techpreview.md" %}

The Ceph Object Storage Multisite feature allows object storage to replicate its data over multiple Ceph clusters.
Using multisite, such object storage is independent and isolated from another object storage in the cluster.
Only the multi-zone multisite setup is currently supported. For more details, see
[Ceph documentation: Multisite](https://docs.ceph.com/en/latest/radosgw/multisite).

<a name="parameters"></a>
## Multisite parameters

{% include "../../snippets/multisiteParameters.md" %}

<a name="enable"></a>
## Enable the multisite RGW Object Storage

1. Open the `CephDeployment` custom resource for editing:
   ```bash
   kubectl -n pelagia edit cephdpl <name>
   ```
2. Update the `spec.objectStorage.multiSite` section specification as required using the configuration
   references for master or replication zones below.
3. Configure the `zone` RGW parameter and leave `dataPool`
   and `metadataPool` empty. These parameters are ignored because
   the `zone` section in the multisite configuration specifies the pool parameters.

     Also, you can split the RGW daemon on daemons serving clients and daemons
     running synchronization. To enable this option, specify
     `splitDaemonForMultisiteTrafficSync` in the `gateway` section.

     For example:
     ```yaml
     spec:
       objectStorage:
         multiSite:
            realms:
            - name: openstack-store
              pullEndpoint:
                endpoint: http://10.11.0.75:8080
                accessKey: DRND5J2SVC9O6FQGEJJF
                secretKey: qpjIjY4lRFOWh5IAnbrgL5O6RTA1rigvmsqRGSJk
            zoneGroups:
            - name: openstack-store
              realmName: openstack-store
            zones:
            - name: openstack-store-backup
              zoneGroupName: openstack-store
              metadataPool:
                failureDomain: host
                replicated:
                  size: 3
              dataPool:
                erasureCoded:
                  codingChunks: 1
                  dataChunks: 2
                failureDomain: host
         rgw:
           dataPool: {}
           gateway:
             allNodes: false
             instances: 2
             splitDaemonForMultisiteTrafficSync: true
             port: 80
             securePort: 8443
           healthCheck: {}
           metadataPool: {}
           name: openstack-store-backup
           preservePoolsOnDelete: false
           zone:
             name: openstack-store-backup
     ```

4. Verify the multisite status:
   ```bash
   radosgw-admin sync status
   ```

Once done, Pelagia Deployment Controller will create the required resources and Rook will
handle the multisite configuration. For details, see: [Rook documentation: Object Multisite](https://rook.io/docs/rook/latest/Storage-Configuration/Object-Storage-RGW/ceph-object-multisite/).

<a name="master-zone-multisite"></a>
### Configuring master zone for RGW Object Storage

If you do not need to replicate data from a different storage cluster,
and the current cluster represents the master zone, modify the current
`objectStorage` section to use the multisite mode:

1. Configure the `zone` RADOS Gateway (RGW) parameter by setting it to
   the RGW Object Storage name.

    !!! note

        Leave `dataPool` and `metadataPool` empty. These
        parameters are ignored because the `zone` block in the multisite
        configuration specifies the pools parameters. Other RGW parameters
        do not require changes.

    For example:
    ```yaml
      spec:
        objectStorage:
          rgw:
            gateway:
              allNodes: false
              instances: 2
              port: 80
              securePort: 8443
            name: openstack-store
            preservePoolsOnDelete: false
            zone:
              name: openstack-store
      ```

2. Create the `multiSite` section where the names of realm, zone group,
   and zone must match the current RGW name.

    Specify the `endpointsForZone` parameter according to your
    configuration:

      * If you use ingress proxy, which is defined in the `spec.ingressConfig`
        section, add the FQDN endpoint.
      * If you do not use any ingress proxy and access the RGW API using the
        default RGW external service, add the IP address of the external
        service or leave this parameter empty.

    The following example illustrates a complete `objectStorage` section:
    ```yaml
    objectStorage:
      multiSite:
        realms:
        - name: openstack-store
        zoneGroups:
        - name: openstack-store
          realmName: openstack-store
        zones:
        - name: openstack-store
          zoneGroupName: openstack-store
          endpointsForZone: http://10.11.0.75:8080
          metadataPool:
            failureDomain: host
              replicated:
                size: 3
          dataPool:
            erasureCoded:
              codingChunks: 1
              dataChunks: 2
            failureDomain: host
      rgw:
        gateway:
          allNodes: false
          instances: 2
          port: 80
          securePort: 8443
        name: openstack-store
        preservePoolsOnDelete: false
        zone:
          name: openstack-store
    ```

<a name="replication-zone-multisite"></a>
### Configuring replication zone for RGW Object Storage

If you use a different storage cluster, and its object storage data must
be replicated, specify the realm and zone group names along with the
`pullEndpoint` parameter. Additionally, specify the endpoint, access
key, and system keys of the system user of the realm from which you need
to replicate data. For details, see step 2 of this procedure.

!!! note

    All commands below should be executed inside `pelagia-ceph-toolbox` pod.

1. To obtain the endpoint of the cluster zone that must be replicated, run
   the following command by specifying the zone group name of the required
   master zone on the master zone side:
   ```bash
   radosgw-admin zonegroup get --rgw-zonegroup=<ZONE_GROUP_NAME> | jq -r '.endpoints'
   ```

   The endpoint is located in the `endpoints` field.

2. To obtain the access key and the secret key of the system user, run
   the following command on the required Ceph cluster:
   ```bash
   radosgw-admin user list
   ```
   
3. To obtain the system user name, which has your RGW `ObjectStorage`
   name as prefix:
   ```bash
   radosgw-admin user info --uid="<USER_NAME>" | jq -r '.keys'
   ```

For example:
```yaml
spec:
  objectStorage:
    multiSite:
      realms:
      - name: openstack-store
        pullEndpoint:
          endpoint: http://10.11.0.75:8080
          accessKey: DRND5J2SVC9O6FQGEJJF
          secretKey: qpjIjY4lRFOWh5IAnbrgL5O6RTA1rigvmsqRGSJk
      zoneGroups:
      - name: openstack-store
        realmName: openstack-store
      zones:
      - name: openstack-store-backup
        zoneGroupName: openstack-store
        metadataPool:
          failureDomain: host
          replicated:
            size: 3
        dataPool:
          erasureCoded:
            codingChunks: 1
            dataChunks: 2
          failureDomain: host
```

!!! note

    We recommend using the same `metadataPool` and `dataPool` settings as you use in the master zone.

## Configure and clean up a multisite configuration

!!! warning

    Rook does not handle multisite configuration changes and cleanup.
    Therefore, once you enable multisite for Ceph RGW Object Storage, perform
    these operations manually in the `pelagia-ceph-toolbox` pod. For details, see
    [Rook documentation: Multisite cleanup](https://rook.io/docs/rook/latest/Storage-Configuration/Object-Storage-RGW/ceph-object-multisite/?h=multisite#multisite-cleanup).


Automatic update of zonegroup hostnames is disabled in `CephDeployment` CR if RADOS Gateway Multisite is enabled or
External Ceph cluster mode is enabled, therefore, manually specify all
required hostnames and update the zone group. In the `pelagia-ceph-toolbox` pod, run
the following script:


!!! note

    The script is actual for Rook resources deployed by Pelagia Helm chart. If you're
    using Rook which is not deployed by Pelagia Helm chart, update zonegroup configuration
    manually.


```bash
/usr/local/bin/zonegroup_hostnames_update.sh --rgw-zonegroup <ZONEGROUP_NAME> --hostnames fqdn1[,fqdn2]
```

If the multisite setup is completely cleaned up, manually execute the following
steps on the `pelagia-ceph-toolbox` pod:

1. Due to the [Rook issue #16328](https://github.com/rook/rook/issues/16328), verify that `.rgw.root` pool is removed:

     * Verify `.rgw.root` pool does not exist:
       ```bash
       ceph osd pool ls | grep .rgw.root
       ```

     * If the pool `.rgw.root` is not removed, remove it manually:
       ```bash
       ceph osd pool rm .rgw.root .rgw.root --yes-i-really-really-mean-it
       ```

         Some other RGW pools may also require a removal after cleanup.

2. Remove the related RGW crush rules:
   ```bash
   ceph osd crush rule ls | grep rgw | xargs -I% ceph osd crush rule rm %
   ```
