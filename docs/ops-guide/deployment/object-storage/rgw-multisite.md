---
description: How to enable Ceph RGW multisite Object Storage, including configuration steps.
keywords: pelagia, enable ceph multisite, rgw multisite, object storage, ceph replication,
  multi-zone multisite, ceph multisite, ceph multisite configuration
---

<a id="rgw-multisite-enable-multisite-for-ceph-object-storage"></a>
# Enable Multisite for Ceph Object Storage

{% include "../../../snippets/techpreview.md" %}

The Ceph Object Storage Multisite feature allows object storage to replicate its data over multiple Ceph clusters.
Using multisite, such object storage is independent and isolated from another object storage in the cluster.
Only the multi-zone multisite setup is currently supported. For more details, see
[Ceph documentation: Multisite](https://docs.ceph.com/en/latest/radosgw/multisite).

<a name="rgw-multisite-multisite-parameters"></a>
## Multisite parameters

{% include "../../../snippets/multisiteParameters.md" %}

<a name="rgw-multisite-enable-the-multisite-rgw-object-storage"></a>
## Enable the multisite RGW Object Storage

1. Create a secret with realm keys in the `rook-ceph` namespace. For the
   instructions, see [Rook documentation: Getting Realm Access Key and Secret Key](https://rook.io/docs/rook/v1.19/Storage-Configuration/Object-Storage-RGW/ceph-object-multisite/#getting-realm-access-key-and-secret-key).

2.  Open the `CephDeployment` custom resource for editing:
   ```bash
   kubectl -n pelagia edit cephdpl <name>
   ```

3. Update the `spec.objectStorage.realms`, `spec.objectStorage.zonegroups`, and `spec.objectStorage.zones` sections as required using the configuration references for master or replication zones, as described below.
4. Configure the `zone` RGW parameter and leave `dataPool` and `metadataPool` empty.
   These parameters are ignored because the `zone` section in the multisite configuration specifies the pool parameters.

    Also, you can split the RGW daemon into daemons serving clients and daemons
    running synchronization.
    To enable this option, create a separate ObjectStore with `auxilaryService: true` and set `disableMultisiteSyncTraffic` to `true` in the `gateway` section. For reference, see [Rook documentation: Scaling a Multisite](https://rook.io/docs/rook/v1.19/Storage-Configuration/Object-Storage-RGW/ceph-object-multisite/#scaling-a-multisite).

    ??? "Example configuration"

        ```yaml
        spec:
          objectStorage:
            realms:
            - name: openstack-store
              spec:
                default: true
                pull:
                  endpoint: http://10.11.0.75:8080
            zonegroups:
            - name: openstack-store
              spec:
                realm: openstack-store
            zones:
            - name: openstack-store-backup
              spec:
                zoneGroup: openstack-store
                metadataPool:
                  failureDomain: host
                  replicated:
                    size: 3
                dataPool:
                  erasureCoded:
                    codingChunks: 1
                    dataChunks: 2
                  failureDomain: host
            objectStores:
            - name: rgw-secondary-object-store
              usedForOpenstack: true
              spec:
               gateway:
                 instances: 2
                 port: 80
                 disableMultisiteSyncTraffic: true
                 securePort: 8443
               preservePoolsOnDelete: false
               zone:
                 name: openstack-store-backup
            - name: rgw-store-replication
              auxilaryService: true
              spec:
               gateway:
                 instances: 1
                 disableMultisiteSyncTraffic: false
                 port: 80
               preservePoolsOnDelete: false
               zone:
                 name: openstack-store-backup
        ```

5. Verify the multisite status:
   ```bash
   radosgw-admin sync status
   ```

Once done, Pelagia Deployment Controller will create the required resources and Rook will
handle the multisite configuration. For details, see: [Rook documentation: Object Multisite](https://rook.io/docs/rook/latest/Storage-Configuration/Object-Storage-RGW/ceph-object-multisite/).

<a name="rgw-multisite-configuring-master-zone-for-rgw-object-storage"></a>
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
        objectStores:
        - name: openstack-store
          usedForOpenstack: true
          spec:
            gateway:
              instances: 2
              port: 80
              securePort: 8443
            name: openstack-store
            preservePoolsOnDelete: false
            zone:
              name: openstack-store
    ```

2. Create the multisite configuration where the names of realm, zone group,
   and zone must match the current RGW name.

    Specify the `customEndpoints` parameter according to your
    configuration:

      * If you use ingress proxy, which is defined in the `spec.ingressConfig`
        section, add the FQDN endpoint (deprecated)
      * If you use Gateway HTTPRoute, add hostnames with HTTP scheme
      * If you do not use any ingress proxy and access the RGW API using the
        default RGW external service, add the IP address of the external
        service or leave this parameter empty

??? "Example configuration of a complete objectStorage section"

    ```yaml
    objectStorage:
      multiSite:
        realms:
        - name: openstack-store
          spec:
            default: true
        zonegroups:
        - name: openstack-store
          spec:
            realm: openstack-store
        zones:
        - name: openstack-store
          spec:
            zoneGroup: openstack-store
            customEndpoints:
            - http://10.11.0.75:8080
            metadataPool:
              failureDomain: host
                replicated:
                  size: 3
            dataPool:
              erasureCoded:
                codingChunks: 1
                dataChunks: 2
              failureDomain: host
      objectStores:
      - name: openstack-store
        usedForOpenstack: true
        spec:
          gateway:
            instances: 2
            port: 80
            securePort: 8443
          name: openstack-store
          preservePoolsOnDelete: false
          zone:
            name: openstack-store
    ```

<a name="rgw-multisite-configuring-replication-zone-for-rgw-object-storage"></a>
### Configuring replication zone for RGW Object Storage

If you use a different storage cluster, and its object storage data must
be replicated, specify the realm and zone group names along with the
`pull.endpoint` parameter. Additionally, specify the endpoint, access
key, and system keys of the system user of the realm from which you need
to replicate data. For details, see the steps below.

!!! note

    Execute all commands below inside the `pelagia-ceph-toolbox` pod.

1. Obtain the endpoint of the cluster zone that must be replicated.
   Run the following command by specifying the zone group name of the required master zone on the master zone side:
   ```bash
   radosgw-admin zonegroup get --rgw-zonegroup=<ZONE_GROUP_NAME> | jq -r '.endpoints'
   ```

     The endpoint is located in the `endpoints` field.

2. Obtain the access key and the secret key of the system user by running
   the following command on the required Ceph cluster:
   ```bash
   radosgw-admin user list
   ```

3. Obtain the system user name that has your RGW `ObjectStorage` name as prefix:
   ```bash
   radosgw-admin user info --uid="<USER_NAME>" | jq -r '.keys'
   ```

4. Create a secret with realm keys using [Rook documentation: Getting Realm Access Key and Secret Key](https://rook.io/docs/rook/v1.19/Storage-Configuration/Object-Storage-RGW/ceph-object-multisite/#getting-realm-access-key-and-secret-key).

5. Adjust the `CephDeployment` spec as required.

??? "Example configuration"

    ```yaml
    spec:
      objectStorage:
        realms:
        - name: openstack-store
          spec:
            default: true
            pull:
              endpoint: http://10.11.0.75:8080
        zonegroups:
        - name: openstack-store
          spec:
            realm: openstack-store
        zones:
        - name: openstack-store-backup
          spec:
            zoneGroup: openstack-store
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

To manage allowed DNS names, use the `spec.hosting.dnsNames` field of the related Ceph ObjectStore specification.

If the multisite setup is completely cleaned up, manually execute the following steps inside the `pelagia-ceph-toolbox` pod:

1. Due to the [Rook issue #16328](https://github.com/rook/rook/issues/16328), verify that the `.rgw.root` pool is removed:

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
