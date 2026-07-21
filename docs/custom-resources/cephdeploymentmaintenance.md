---
description: API reference for the CephDeploymentMaintenance custom resource used to store
  Ceph cluster maintenance state and control the states of a Ceph cluster and Pelagia controllers.
keywords: pelagia, cephdeploymentmaintenance, ceph maintenance, ceph cluster maintenance,
  ceph cluster state
---

<a id="cephdeploymentmaintenance-cephdeploymentmaintenance-custom-resource"></a>
# CephDeploymentMaintenance custom resource

Cluster maintenance is an important part of a cluster lifecycle.
The `CephDeploymentMaintenance` (`cephdeploymentmaintenances.lcm.mirantis.com`) custom resource (CR) allows you to properly handle cluster maintenance and control the states of a Ceph cluster and Pelagia controllers.

To obtain the current maintenance status, run the following command:

```bash
kubectl -n pelagia get cephdeploymentmaintenance -o yaml
```

??? "Example `CephDeploymentMaintenance` resource"

    ```yaml
    apiVersion: v1
    items:
    - apiVersion: lcm.mirantis.com/v1alpha1
      kind: CephDeploymentMaintenance
      metadata:
        creationTimestamp: "2026-06-16T12:36:57Z"
        generation: 1
        labels:
          app.kubernetes.io/created-by: pelagia-deployment-controller
          app.kubernetes.io/managed-by: pelagia-deployment-controller
          app.kubernetes.io/part-of: ceph.pelagia.lcm
        name: rook-ceph
        namespace: ceph-lcm-mirantis
        ownerReferences:
        - apiVersion: lcm.mirantis.com/v1alpha1
          kind: CephDeployment
          name: rook-ceph
          uid: df1e26ab-e324-418f-b623-8df318ea79df
        resourceVersion: "1210614"
        uid: d96a4e52-f6d4-457c-a12b-ab5ab909d6a5
      status:
        lastStateCheck: "2026-07-21T10:08:54Z"
        state: Idle
    kind: List
    metadata:
      resourceVersion: ""
      selfLink: ""
    ```

Out-of-the-box, Pelagia has no maintenance controller that controls the `CephDeploymentMaintenance` object.
Therefore, to control maintenance using Pelagia, you must implement and use an environment-specific maintenance controller.

<a name="cephdeploymentmaintenance-high-level-status-fields"></a>
## High-level status fields

- `state` - Current cluster maintenance state. Possible values are:

    - `Idle` - No maintenance in progress
    - `Acting` - Maintenance is in progress
    - `Failing` - Critical issues occur during maintenance

- `lastStateCheck` - `DateTime` when the previous cluster state check occurred.
- `message` - Additional information about the current maintenance state, if any.

<a name="cephdeploymentmaintenance-maintenance-flow-overview"></a>
## Maintenance flow overview

Pelagia controllers inspect the `CephDeploymentMaintenance` object state and perform the following actions:

- `pelagia-lcm-controller` - controls the Rook Operator deployment and scales it down to `0` when
  `CephDeploymentMaintenance` is in the `Acting` or `Failing` state, treating it as the cluster maintenance state.
  Once the maintenance is completed, the controller scales the Rook Operator back to `1`.
- `pelagia-deployment-controller` - stops `CephDeployment` reconciliation, when `CephDeploymentMaintenance` is in the `Acting` or `Failing` state, treating it as the cluster maintenance state. Once the maintenance is completed, it continues the `CephDeployment` reconciliation.
