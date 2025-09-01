<a id="cinder-multi-backend"></a>

# Ceph pools for Cinder multi-backend

The `CephDeployment` custom resource (CR) supports multiple Ceph pools with the `volumes` role
to configure Cinder multiple backends in Rockoon OpenStack.

## Configure Ceph pools for Cinder multiple backends

1. In the `CephDeployment` CR, add the desired number of Ceph pools to the `pools` section with the `volumes` role:
   ```bash
   kubectl -n ceph-lcm-mirantis edit miraceph
   ```

     Example configuration:
     ```yaml
     spec:
       pools:
       - default: false
         deviceClass: hdd
         name: volumes
         replicated:
           size: 3
         role: volumes
       - default: false
         deviceClass: hdd
         name: volumes-backend-1
         replicated:
           size: 3
         role: volumes
       - default: false
         deviceClass: hdd
         name: volumes-backend-2
         replicated:
           size: 3
         role: volumes
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
