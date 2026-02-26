<a id="move-failure-domain-migrate-ceph-pools-from-one-failure-domain-to-another"></a>

# Migrate Ceph pools from one failure domain to another

The document describes how to change the failure domain of an already deployed Ceph cluster.

!!! note

    This document focuses on changing the failure domain from a smaller
    to wider one, for example, from `host` to `rack`. Using the same
    instruction, you can move the failure domain from a wider to smaller scale.

A high-level overview of the procedure includes the following steps:

1. Set correct labels on the nodes.
2. Create the new bucket hierarchy.
3. Move nodes to new buckets.
4. Scale down Pelagia controllers.
5. Modify the CRUSH rules.
6. Add the manual changes to the `CephDeployment` custom resource (CR) spec.
7. Scale up Pelagia controllers.

## Prerequisites

1. Verify that the Ceph cluster has enough space for multiple copies of data to migrate. We highly recommend that
   the Ceph cluster has a minimum of 25% free space for the procedure to succeed.

    !!! note

         The migration procedure implies data movement and optional modification of CRUSH rules that cause a large
         amount of data (depending on the cluster size) to be first copied to a new location in the Ceph
         cluster before data removal.

2. Create a backup of the current `CephDeployment` CR:
   ```bash
   kubectl -n pelagia get cephdpl -o yaml > cephdpl-backup.yaml
   ```

3. In the `pelagia-ceph-toolbox` pod, obtain a backup of the CRUSH map:
   ```bash
   ceph osd getcrushmap -o /tmp/crush-map-orig
   crushtool -d /tmp/crush-map-orig -o /tmp/crush-map-orig.txt
   ```

## Migrate Ceph pools

This procedure contains an example of moving failure domains of all pools from
`host` to `rack`. Using the same instruction, you can migrate pools from
other types of failure domains, migrate pools separately, and so on.

**To migrate Ceph pools from one failure domain to another:**

1. Set the required CRUSH topology in the `MiraCeph` object for each
   defined node. For details on the `crush` parameter, see
   [Nodes parameters](../../architecture/custom-resources/cephdeployment.md#cephdeployment-nodes-parameters).

     Setting the CRUSH topology to each node causes the Pelagia Deployment Controller to set proper Kubernetes
     labels on the nodes.

     Example of adding the `rack` CRUSH topology key for each node in the `nodes` section:
     ```yaml
     spec:
       nodes:
         machine1:
           crush:
             rack: rack-1
         machine2:
           crush:
             rack: rack-1
         machine3:
           crush:
             rack: rack-2
         machine4:
           crush:
             rack: rack-2
         machine5:
           crush:
             rack: rack-3
         machine6:
           crush:
             rack: rack-3
     ```

2. Verify that the required buckets and bucket types are present in the Ceph hierarchy:

     1. Enter the `pelagia-ceph-toolbox` pod:
        ```bash
        kubectl -n rook-ceph exec -it deploy/pelagia-ceph-toolbox -- bash
        ```

     2. Verify that the required bucket type is present by default:
        ```bash
        ceph osd getcrushmap -o /tmp/crush-map
        crushtool -d /tmp/crush-map -o /tmp/crush-map.txt
        cat /tmp/crush-map.txt # Look for the section named → “# types”
        ```

          Example of system response:
          ```bash
          # types
          type 0 osd
          type 1 host
          type 2 chassis
          type 3 rack
          type 4 row
          type 5 pdu
          type 6 pod
          type 7 room
          type 8 datacenter
          type 9 zone
          type 10 region
          type 11 root
          ```

     3. Verify that the buckets with the required bucket type are present:
        ```bash
        cat /tmp/crush-map.txt # Look for the section named → “# buckets”
        ```

          Example of system response of an existing `rack` bucket:
          ```bash
          # buckets
          rack rack-1 {
            id -15
            id -16 class hdd
            # weight 0.00000
            alg straw2
            hash 0
          }
          ```

     4. If the required buckets are not created, create new ones with the
        required bucket type:
        ```bash
        ceph osd crush add-bucket <bucketName> <bucketType> root=default
        ```

          For example:
          ```bash
          ceph osd crush add-bucket rack-1 rack root=default
          ceph osd crush add-bucket rack-2 rack root=default
          ceph osd crush add-bucket rack-3 rack root=default
          ```

     5. Exit the `pelagia-ceph-toolbox` pod.

3. Optional. Order buckets as required:

   * Add the first Ceph CRUSH smaller bucket to its respective wider bucket:
     ```bash
     kubectl -n rook-ceph exec -it deploy/pelagia-ceph-toolbox -- bash
     ceph osd crush move <smallerBucketName> <bucketType>=<widerBucketName>
     ```

       !!! warning

           We highly recommend moving one bucket at a time.

           For more details, refer to official Ceph documentation:
           [CRUHS Maps: Moving a bucket](https://docs.ceph.com/en/latest/rados/operations/crush-map/#moving-a-bucket).

       Substitute the following parameters:

         * `<smallerBucketName>` with the name of the smaller bucket, for example, host name;
         * `<bucketType>` with the required bucket type, for example, `rack`;
         * `<widerBucketName>` with the name of the wider bucket, for example, rack name.

       For example:
       ```bash
       ceph osd crush move kaas-node-1 rack=rack-1 root=default
       ```

   * After the bucket is moved to the new location in the CRUSH hierarchy,
     verify that no data rebalancing occurs:
     ```bash
     ceph -s
     ```

   * Add the remaining Ceph CRUSH smaller buckets to their respective wider buckets one by one.

4. Scale the Pelagia Controllers and Rook Ceph Operator deployments to `0` replicas:
   ```bash
   kubectl -n pelagia scale deploy --all --replicas 0
   kubectl -n rook-ceph scale deploy rook-ceph-operator --replicas 0
   ```

5. Manually modify the CRUSH rules for Ceph pools to enable data placement on a new failure domain:

     1. Enter the `pelagia-ceph-toolbox` pod:
        ```bash
        kubectl -n rook-ceph exec -it deploy/pelagia-ceph-toolbox -- bash
        ```

     2. List the CRUSH rules and erasure code profiles for the pools:
        ```bash
        ceph osd pool ls detail
        ```

          Example output:

          ```bash
          pool 1 'mirablock-k8s-block-hdd' replicated size 2 min_size 1 crush_rule 9 object_hash rjenkins pg_num 32 pgp_num 32 autoscale_mode on last_change 1193 lfor 0/0/85 flags hashpspool,selfmanaged_snaps stripe_width 0 application rbd read_balance_score 1.31
          pool 2 '.mgr' replicated size 2 min_size 1 crush_rule 1 object_hash rjenkins pg_num 1 pgp_num 1 autoscale_mode on last_change 70 flags hashpspool stripe_width 0 pg_num_max 32 pg_num_min 1 application mgr read_balance_score 6.06
          pool 3 'openstack-store.rgw.otp' replicated size 2 min_size 1 crush_rule 11 object_hash rjenkins pg_num 8 pgp_num 8 autoscale_mode on last_change 1197 flags hashpspool stripe_width 0 pg_num_min 8 application rook-ceph-rgw read_balance_score 2.27
          pool 4 'openstack-store.rgw.meta' replicated size 2 min_size 1 crush_rule 12 object_hash rjenkins pg_num 8 pgp_num 8 autoscale_mode on last_change 1197 flags hashpspool stripe_width 0 pg_num_min 8 application rook-ceph-rgw read_balance_score 1.50
          pool 5 'openstack-store.rgw.log' replicated size 2 min_size 1 crush_rule 10 object_hash rjenkins pg_num 8 pgp_num 8 autoscale_mode on last_change 1197 flags hashpspool stripe_width 0 pg_num_min 8 application rook-ceph-rgw read_balance_score 3.00
          pool 6 'openstack-store.rgw.buckets.non-ec' replicated size 2 min_size 1 crush_rule 13 object_hash rjenkins pg_num 8 pgp_num 8 autoscale_mode on last_change 1197 flags hashpspool stripe_width 0 pg_num_min 8 application rook-ceph-rgw read_balance_score 1.50
          pool 7 'openstack-store.rgw.buckets.index' replicated size 2 min_size 1 crush_rule 15 object_hash rjenkins pg_num 8 pgp_num 8 autoscale_mode on last_change 1197 flags hashpspool stripe_width 0 pg_num_min 8 application rook-ceph-rgw read_balance_score 2.25
          pool 8 '.rgw.root' replicated size 2 min_size 1 crush_rule 14 object_hash rjenkins pg_num 8 pgp_num 8 autoscale_mode on last_change 1197 flags hashpspool stripe_width 0 pg_num_min 8 application rook-ceph-rgw read_balance_score 3.75
          pool 9 'openstack-store.rgw.control' replicated size 2 min_size 1 crush_rule 16 object_hash rjenkins pg_num 8 pgp_num 8 autoscale_mode on last_change 1197 flags hashpspool stripe_width 0 pg_num_min 8 application rook-ceph-rgw read_balance_score 3.00
          pool 10 'other-hdd' replicated size 2 min_size 1 crush_rule 19 object_hash rjenkins pg_num 32 pgp_num 32 autoscale_mode on last_change 1179 lfor 0/0/85 flags hashpspool,selfmanaged_snaps stripe_width 0 application rbd read_balance_score 1.69
          pool 11 'openstack-store.rgw.buckets.data' erasure profile openstack-store.rgw.buckets.data_ecprofile size 3 min_size 2 crush_rule 18 object_hash rjenkins pg_num 32 pgp_num 32 autoscale_mode on last_change 1198 lfor 0/0/86 flags hashpspool,ec_overwrites stripe_width 8192 application rook-ceph-rgw
          pool 12 'vms-hdd' replicated size 2 min_size 1 crush_rule 21 object_hash rjenkins pg_num 256 pgp_num 256 autoscale_mode on last_change 1182 lfor 0/0/95 flags hashpspool,selfmanaged_snaps stripe_width 0 target_size_ratio 0.4 application rbd read_balance_score 1.24
          pool 13 'volumes-hdd' replicated size 2 min_size 1 crush_rule 23 object_hash rjenkins pg_num 64 pgp_num 64 autoscale_mode on last_change 1185 lfor 0/0/89 flags hashpspool,selfmanaged_snaps stripe_width 0 target_size_ratio 0.2 application rbd read_balance_score 1.31
          pool 14 'backup-hdd' replicated size 2 min_size 1 crush_rule 25 object_hash rjenkins pg_num 32 pgp_num 32 autoscale_mode on last_change 1188 lfor 0/0/90 flags hashpspool,selfmanaged_snaps stripe_width 0 target_size_ratio 0.1 application rbd read_balance_score 2.06
          pool 15 'images-hdd' replicated size 2 min_size 1 crush_rule 27 object_hash rjenkins pg_num 32 pgp_num 32 autoscale_mode on last_change 1191 lfor 0/0/90 flags hashpspool,selfmanaged_snaps stripe_width 0 target_size_ratio 0.1 application rbd read_balance_score 1.50
          ```

     3. For each replicated Ceph pool:

          1. Obtain the current CRUSH rule name:
             ```bash
             ceph osd crush rule dump <oldCrushRuleName>
             ```

          2. Create a new CRUSH rule with the required bucket type using the same root, device class, and new bucket type:
             ```bash
             ceph osd crush rule create-replicated <newCrushRuleName> <root> <bucketType> <deviceClass>
             ```

               For example:
               ```bash
               ceph osd crush rule create-replicated images-hdd-rack default rack hdd
               ```

               For more details, refer to official Ceph documentation:
               [CRUSH Maps: Creating a rule for a replicated pool](https://docs.ceph.com/en/latest/rados/operations/crush-map/#creating-a-rule-for-a-replicated-pool).

          3. Apply a new crush rule to the Ceph pool:
             ```bash
             ceph osd pool set <poolName> crush_rule <newCrushRuleName>
             ```

               For example:
               ```bash
               ceph osd pool set images-hdd crush_rule images-hdd-rack
               ```

          4. Wait for data to be rebalanced after moving the Ceph pool under the
             new failure domain (bucket type) by monitoring Ceph health:
             ```bash
             ceph -s
             ```

          5. Verify that the old CRUSH rule is not used anymore:
             ```bash
             ceph osd pool ls detail
             ```

               The rule ID is located in the CRUSH map and must match the rule ID in the output of **ceph osd dump**.

          6. Remove the old unused CRUSH rule and rename the new one to the
             original name:
             ```bash
             ceph osd crush rule rm <oldCrushRuleName>
             ceph osd crush rule rename <newCrushRuleName> <oldCrushRuleName>
             ```

     4. For each erasure-coded Ceph pool:

        !!! note

             Erasure-coded pools require different number of buckets to store data. Instead of the number of replicas
             in replicated pools, erasure-coded pools require the `coding chunks + data chunks` number of buckets
             existing in the Ceph cluster. For example, if an erasure-coded pool has 2 coding chunks and 2 data chunks
             configured, then the pool requires 4 different buckets, for example, 4 racks, to store data.

        1. Obtain the current parameters of the erasure-coded profile:
           ```bash
           ceph osd erasure-code-profile get <ecProfile>
           ```

        2. In the profile, add the new bucket type as the failure domain using the `crush-failure-domain` parameter:
           ```bash
           ceph osd erasure-code-profile set <ecProfile> k=<int> m=<int> crush-failure-domain=<bucketType> crush-device-class=<deviceClass>
           ```

        3. Create a new CRUSH rule in the profile:
           ```bash
           ceph osd crush rule create-erasure <newEcCrushRuleName> <ecProfile>
           ```

        4. Apply the new CRUSH rule to the pool:
           ```bash
           ceph osd pool set <poolName> crush_rule <newEcCrushRuleName>
           ```

        5. Wait for data to be rebalanced after moving the Ceph pool under the new failure domain (bucket type)
           by monitoring Ceph health:
           ```bash
           ceph -s
           ```

        6. Verify that the old CRUSH rule is not used anymore:
           ```bash
           ceph osd pool ls detail
           ```

             The rule ID is located in the CRUSH map and must match the rule ID in the output of **ceph osd dump**.

        7. Remove the old unused CRUSH rule and rename the new one to the original name:
           ```bash
           ceph osd crush rule rm <oldCrushRuleName>
           ceph osd crush rule rename <newCrushRuleName> <oldCrushRuleName>
           ```

            !!! note

                New erasure-coded profiles cannot be renamed, so they will not be removed automatically during pools
                cleanup. Remove them manually, if needed.

     5. Exit the `pelagia-ceph-toolbox` pod.

6. Update the `CephDeployment` CR `spec` by setting the `failureDomain: rack`
   parameter for each pool. The configuration from the Rook perspective must
   match the manually created configuration. For example:
   ```yaml
   spec:
     pools:
     - name: images
       ...
       failureDomain: rack
     - name: volumes
       ...
       failureDomain: rack
     ...
     objectStorage:
       rgw:
         dataPool:
           failureDomain: rack
           ...
         metadataPool:
           failureDomain: rack
           ...
   ```

7. Monitor the Ceph cluster health and wait until rebalancing is completed:
   ```bash
   ceph -s
   ```

     Example of a successful system response:
     ```bash
     HEALTH_OK
     ```

8. Scale back Pelagia Controllers and Rook Ceph Operator deployments:
   ```bash
   kubectl -n pelagia scale deploy --all --replicas 3
   kubectl -n rook-ceph scale deploy rook-ceph-operator --replicas 1
   ```
