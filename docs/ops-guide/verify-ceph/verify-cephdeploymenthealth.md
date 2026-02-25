<a id="verify-cephdeploymenthealth-verify-ceph-cluster-state"></a>

# Verify Ceph cluster state

To verify the state of a Ceph cluster, Pelagia provides statuses to `CephDeployment` and `CephDeploymentHealth`
custom resources (CR). These resources contain information about the state of the Ceph cluster components,
their health, and potentially problematic components.

**To verify the Pelagia API health:**

1. Obtain the `CephDeployment` CR:
   ```bash
   kubectl -n pelagia get cephdpl -o yaml
   ```

     Information from `CephDeployment.status` reflects the spec handling state and
     validation result. For the description of status fields, see
     [Status fields](../../architecture/custom-resources/cephdeployment.md#cephdeployment-status-fields).

2. Obtain the `CephDeploymentHealth` CR:
   ```bash
   kubectl -n pelagia get cephdeploymenthealth -o yaml
   ```

     Information from `CephDeploymentHealth.status` contains extensive details about Ceph cluster and a shortened version with status summary. For the description of status fields, see [CephDeploymentHealth custom resource](../../architecture/custom-resources/cephdeploymenthealth.md#cephdeploymenthealth-cephdeploymenthealth-custom-resource).
