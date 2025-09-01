<a id="calc-target-ratio"></a>

# Calculate target ratios for Ceph pools for Rockoon

Ceph pool target ratio defines for the Placement Group (PG) autoscaler the
amount of data the pools are expected to acquire over time in relation to each
other. You can set initial PG values for each Ceph pool. Otherwise, the
autoscaler starts with the minimum value and scales up, causing a lot of data
to move in the background.

You can allocate several pools to use the same device class, which is a solid
block of available capacity in Ceph. For example, if three pools
(`kubernetes-hdd`, `images-hdd`, and `volumes-hdd`) are set to use the
same device class `hdd`, you can set the target ratio for Ceph pools to
provide 80% of capacity to the `volumes-hdd` pool and distribute the
remaining capacity evenly between the two other pools. This way, a Ceph pool
target ratio instructs Ceph on when to warn that a pool is running out of free
space. At the same time, it instructs Ceph on how many placement groups Ceph
should allocate/autoscale for a pool for better data distribution.

Ceph pool target ratio is not a constant value, and you can change it according
to new capacity plans. Once you specify a target ratio, if the PG number of a
pool scales, other pools with a specified target ratio will automatically scale
accordingly.

For details, see [Ceph Documentation: Autoscaling Placement Groups](https://docs.ceph.com/en/latest/rados/operations/placement-groups/).

## Calculate a target ratio for each Ceph pool

1. Define the raw capacity of the entire storage by device class:
   ```bash
   kubectl -n rook-ceph exec -it $(kubectl -n rook-ceph get pod -l "app=rook-ceph-tools" -o name) -- ceph df
   ```

     For illustration purposes, the procedure below uses raw capacity of 185 TB
     or 189440 GB.

2. Design Ceph pools with the considered device class upper bounds of the
   possible capacity. For example, consider the `hdd` device class that
   contains the following pools:

     * The `kubernetes-hdd` pool will contain not more than 2048 GB.
     * The `images-hdd` pool will contain not more than 2048 GB.
     * The `volumes-hdd` pool will contain 50 GB per VM. The upper bound of
       used VMs on the cloud is 204, the pool's replicated size is `3`.
       Therefore, calculate the upper bounds for `volumes-hdd`:
       ```ini
       50 GB per VM * 204 VMs * 3 replicas = 30600 GB
       ```
     * The `backup-hdd` pool can be calculated as a relative of
       `volumes-hdd`. For example, 1 `volumes-hdd` storage unit per 5
       `backup-hdd` units.
     * The `vms-hdd` is a pool for ephemeral storage Copy on Write (CoW). We
       recommend designing the amount of ephemeral data it should store. For
       example, we use 500 GB. However, in reality, despite the CoW data
       reduction, this value is very optimistic.

    !!! note

        If `dataPool` is replicated and Ceph Object Store is planned for intensive use, also calculate
        upper bounds for `dataPool`.

3. Calculate a target ratio for each considered pool. For example:

    | Pools upper bounds                                                                                                                                                                       | Pools capacity                                                                        |
    |------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|---------------------------------------------------------------------------------------|
    | <ul><li>`kubernetes-hdd` = 2048 GB</li><li>`images-hdd` = 2048 GB</li><li>`volumes-hdd` = 30600 GB</li><li>`backup-hdd` = 30600 GB \* 5 = 153000 GB</li><li>`vms-hdd` = 500 GB</li></ul> | <ul><li>Summary capacity = 188196 GB</li><li>Total raw capacity = 189440 GB</li></ul> |

     1. Calculate pools' fit factor using the
        **(total raw capacity) / (pools' summary capacity)** formula. For example:
        ```ini
        pools fit factor = 189440 / 188196 = 1.0066
        ```

     2. Calculate pools' upper bounds size using the
        **(pool upper bounds) \* (pools fit factor)** formula. For example:
        ```ini
        kubernetes-hdd = 2048 GB * 1.0066   = 2061.5168 GB
        images-hdd     = 2048 GB * 1.0066   = 2061.5168 GB
        volumes-hdd    = 30600 GB * 1.0066  = 30801.96 GB
        backup-hdd     = 153000 GB * 1.0066 = 154009.8 GB
        vms-hdd        = 500 GB * 1.0066    = 503.3 GB
        ```

     3. Calculate pools' target ratio using the
        **(pool upper bounds) \* 100 / (total raw capacity)** formula. For
        example:
        ```ini
        kubernetes-hdd = 2061.5168 GB * 100 / 189440 GB = 1.088
        images-hdd     = 2061.5168 GB * 100 / 189440 GB = 1.088
        volumes-hdd    = 30801.96 GB * 100 / 189440 GB  = 16.259
        backup-hdd     = 154009.8 GB * 100 / 189440 GB  = 81.297
        vms-hdd        = 503.3 GB * 100 / 189440 GB     = 0.266
        ```

4. If required, calculate the target ratio for erasure-coded pools.

     Due to erasure-coded pools splitting each object into `K` data parts
     and `M` coding parts, the total used storage for each object is less
     than that in replicated pools. Indeed, `M` is equal to the number of
     OSDs that can be missing from the cluster without the cluster experiencing
     data loss. This means that planned data is stored with an efficiency
     of `(K+M)/2` on the Ceph cluster. For example, if an erasure-coded data
     pool with `K=2, M=2` planned capacity is 200 GB, then the total used
     capacity is `200*(2+2)/2`, which is 400 GB.

5. Open the `CephDeployment` CR for editing:
   ```bash
   kubectl -n pelagia edit cephdpl
   ```

6. In the `pools` section, specify the calculated relatives as `parameters.target_size_ratio` for each considered
   replicated pool. For example:
   ```yaml
   spec:
     pools:
     - name: kubernetes
       deviceClass: hdd
       ...
       replicated:
         size: 3
       parameters:
         target_size_ratio: "1.088"
     - name: images
       deviceClass: hdd
       ...
       replicated:
         size: 3
       parameters:
         target_size_ratio: "1.088"
     - name: volumes
       deviceClass: hdd
       ...
       replicated:
         size: 3
       parameters:
         target_size_ratio: "16.259"
     - name: backup
       deviceClass: hdd
       ...
       replicated:
         size: 3
       parameters:
         target_size_ratio: "81.297"
     - name: vms
       deviceClass: hdd
       ...
       replicated:
         size: 3
       parameters:
         target_size_ratio: "0.266"
   ```

     If Ceph Object Store `dataPool` is `replicated` and a proper value is
     calculated, also specify it:
     ```yaml
     spec:
       objectStorage:
         rgw:
           name: rgw-store
           ...
           dataPool:
             deviceClass: hdd
             ...
             replicated:
               size: 3
             parameters:
               target_size_ratio: "<relative>"
     ```

7. In the `pools` section, specify the calculated relatives as
   `parameters.target_size_ratio` for each considered erasure-coded pool. For
   example:

    !!! note

        The `parameters` section is a key-value mapping where the value is of the string type and should be quoted.

     ```yaml
     spec:
       pools:
       - name: ec-pool
         deviceClass: hdd
         ...
         parameters:
           target_size_ratio: "<relative>"
     ```

     If Ceph Object Store `dataPool` is `erasure-coded` and a proper value
     is calculated, also specify it:
     ```yaml
     spec:
       objectStorage:
         rgw:
           name: rgw-store
           ...
           dataPool:
             deviceClass: hdd
             ...
             parameters:
               target_size_ratio: "<relative>"
     ```

8. Verify that all target ratios have been successfully applied to the Ceph cluster:
   ```bash
   kubectl -n rook-ceph exec -it deploy/rook-ceph-tools -- ceph osd pool autoscale-status
   ```

     Example of system response:
     ```bash
     POOL                                SIZE  TARGET SIZE  RATE  RAW CAPACITY  RATIO   TARGET RATIO  EFFECTIVE RATIO  BIAS  PG_NUM  NEW PG_NUM  AUTOSCALE
     device_health_metrics               0                  2.0   149.9G        0.0000                                 1.0   1                   on
     kubernetes-hdd                      2068               2.0   149.9G        0.0000  1.088         1.0885           1.0   32                  on
     volumes-hdd                         19                 2.0   149.9G        0.0000  16.259        16.2591          1.0   256                 on
     vms-hdd                             19                 2.0   149.9G        0.0000  0.266         0.2661           1.0   128                 on
     backup-hdd                          19                 2.0   149.9G        0.0000  81.297        81.2972          1.0   256                 on
     images-hdd                          888.8M             2.0   149.9G        0.0116  1.088         1.0881           1.0   32                  on
     ```

9. Optional. Repeat the steps above for other device classes.
