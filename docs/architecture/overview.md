# Pelagia Overview

Pelagia Helm chart deploys the following components:

- Pelagia Deployment Controller
- Pelagia Lifecycle Management Controller
- Rook Ceph Operator from [Rook](https://github.com/rook/rook).

## Pelagia Deployment Controller

Pelagia Controller contains the following containers:

### Pelagia Deployment Controller

A Kubernetes Cluster API controller that fetches parameters from the `CephDeployment` custom resource,
creates Rook custom resources, and updates `CephDeployment` and `CephDeploymentHealth` status
based on the Ceph cluster deployment progress.

Pelagia Deployment Controller operations include:

- Transforming user parameters from the `CephDeployment` resource into Rook resources
  and deploying a Ceph cluster using Rook.
- Providing single entrypoint management of the Ceph cluster with Kubernetes.
- Integrating with [Rockoon](https://github.com/Mirantis/rockoon) and providing data for OpenStack
  to integrate with the deployed Ceph cluster.

Also, Pelagia Controller eventually obtains the data from the OpenStack Controller
(Rockoon) for the Keystone integration and updates the Ceph Object Gateway
services configurations to use Kubernetes for user authentication.

### Pelagia Secret Controller

A Kubernetes Cluster API controller that fetches parameters from the corresponding secrets of Rook `CephClient` and `CephObjectStoreUser` and updates the `CephDeploymentSecret` status with the secret references. It can be used
in custom applications to access the Ceph cluster with defined RBD, RadosGW, or CephFS credentials.

## Pelagia Lifecycle Management Controller

Pelagia Lifecycle Management (LCM) Controller contains the following containers:

### Pelagia Health Controller

A Kubernetes controller that collects all valuable parameters from the current
Ceph cluster, its daemons, and Rook resources and exposes them into the
`CephDeploymentHealth` status.

Pelagia Health Controller operations include:

- Collecting all statuses from a Ceph cluster and corresponding Rook resources.
- Collecting additional information on the health of Ceph daemons.
- Providing information for the `status` section of the `CephDeploymentHealth`
  custom resource.

### Pelagia Task Controller

A Kubernetes controller that manages Ceph OSD LCM operations. It
allows for a safe Ceph OSD removal from the Ceph cluster. It uses the
`CephOsdRemoveTask` custom resource to perform the removal operations.

Pelagia Task Controller operations include:

- Providing an ability to perform Ceph OSD LCM operations
- Fetching parameters from the `CephOsdRemoveTask` resource to remove Ceph OSDs and execute them

### Pelagia Infra Controller

A Kubernetes controller that manages the Pelagia infrastructure resources.
It creates and manages Ceph toolbox with Ceph CLI tools and deploys LCM infrastructure
pods that allow processing disk cleanup.

Pelagia Task Controller operations include:

- Creating and managing the Ceph toolbox pod with the Ceph CLI tools
- Creating and managing the `disk-daemon` DaemonSet that is used to perform disk cleanup
  operations on the Ceph OSD disks
- Pausing the regular Rook Ceph Operator orchestration until all requests are finished

## Rook Ceph Operator

[Rook](https://github.com/rook/rook) is a storage orchestrator that deploys Ceph on top of a Kubernetes cluster. Also
known as `Rook`, `Rook Ceph Operator` or `Rook Operator`.

Rook operations include:

- Deploying and managing a Ceph cluster based on provided Rook CRs such as
  `CephCluster`, `CephBlockPool`, `CephObjectStore`, and so on
- Orchestrating the state of the Ceph cluster and all its daemons

For more information about Rook, see the official
[Rook documentation](https://rook.github.io/docs/rook/latest-release/Getting-Started/intro/).

Note that Pelagia Helm chart deploys the Rook Operator
according to the [Rook chart templates](https://github.com/rook/rook/tree/master/deploy/charts/rook-ceph).

A typical Ceph cluster deployed by Rook consists of the following components:

- Two Ceph Managers (`mgr`).
- Three or, in rare cases, five Ceph Monitors (`mon`).
- Ceph OSDs (`osd`). The number of Ceph OSDs may vary depending on deployment needs.
- Optionally three Ceph Object Gateway (`radosgw`) daemons.
- Optionally two Ceph Metadata Servers (`mds`) for CephFS: one active and one stand-by.

!!! warning

    A Ceph cluster with 3 Ceph nodes does not provide hardware fault
    tolerance and is not eligible for recovery operations, such as a disk or
    an entire Ceph node replacement.

## Pelagia Data Flow

The following diagram illustrates the processes within Pelagia and Rook Ceph Operator
that are involved in the Ceph cluster deployment and lifecycle management:

<img src="/assets/overview.svg" alt="drawing"/>


## See also

- [Ceph documentation](https://docs.ceph.com/docs/master/)
- [Rook documentation](https://rook.github.io/docs/rook/latest-release/Getting-Started/intro/)
