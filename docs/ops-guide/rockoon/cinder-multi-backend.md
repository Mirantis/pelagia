---
description: How to configure Ceph pools to enable Cinder multi-backend support.
keywords: pelagia, cinder multi-backend, ceph pools, cinder backend, ceph volumes pools, cephdeployment
---

<a id="cinder-multi-backend-ceph-pools-for-cinder-multi-backend"></a>

# Configure Ceph pools for Cinder multiple backends

The `CephDeployment` custom resource (CR) supports multiple Ceph pools with the `volumes` role
to configure Cinder multiple backends in Rockoon OpenStack.

To configure Ceph pools for Cinder multiple backends:

1. In the `CephDeployment` CR, add the desired number of Ceph pools to the `pools` section with the `volumes` role:
   ```bash
   kubectl -n ceph-lcm-mirantis edit cephdeployment
   ```

     Example configuration:
     ```yaml
     spec:
       blockStorage:
         pools:
         - name: volumes
           role: volumes
           spec:
             deviceClass: hdd
             replicated:
               size: 3
         - name: volumes-backend-1
           role: volumes
           spec:
             deviceClass: hdd
             replicated:
               size: 3
         - name: volumes-backend-2
           role: volumes
           spec:
             deviceClass: hdd
             replicated:
               size: 3
     ```

2. Verify that Cinder backend pools are created and ready:
   ```bash
   kubectl -n pelagia get cephdeploymenthealth -o yaml
   ```

     Example output:
     ```yaml
     status:
       healthReport:
         rookCephObjects:
           blockStorage:
             cephBlockPools:
               volumes-hdd:
                 info:
                   failureDomain: host
                   type: Replicated
                 observedGeneration: 1
                 phase: Ready
                 poolID: 12
               volumes-backend-1-hdd:
                 info:
                   failureDomain: host
                   type: Replicated
                 observedGeneration: 1
                 phase: Ready
                 poolID: 13
               volumes-backend-2-hdd:
                 info:
                   failureDomain: host
                   type: Replicated
                 observedGeneration: 1
                 phase: Ready
                 poolID: 14
     ```

3. Verify that the added Ceph pools are accessible from the Cinder service.
   For example:
   ```bash
   kubectl -n openstack exec -it cinder-volume-0 -- rbd ls -p volumes-backend-1-hdd -n client.cinder
   kubectl -n openstack exec -it cinder-volume-0 -- rbd ls -p volumes-backend-2-hdd -n client.cinder
   ```

After Ceph pools become available, Rockoon will automatically specify them as an additional Cinder backend and
register as a new volume type, which you can use to create Cinder volumes.
