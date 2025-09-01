# LCM-only Installation Guide

This section provides instructions on how to install Pelagia in lifecycle-management-only (lcm-only) mode.
This mode allows you to automatically remove Rook Ceph OSD disks and nodes in your Kubernetes cluster.

## Prerequisites

* A Kubernetes cluster, for example, deployed with [k0s](https://docs.k0sproject.io/stable/).
* A deployed Ceph cluster managed by [Rook](https://github.com/rook/rook).

## Installation

To install Pelagia in lcm-only mode, use the Helm chart provided in the repository:

```bash
export PELAGIA_VERSION="<version>"
helm upgrade --install pelagia-ceph oci://registry.mirantis.com/pelagia/pelagia-ceph --version ${PELAGIA_VERSION} --set cephDeployment.enabled=false,rookConfig.configureRook=false -n pelagia --create-namespace
```

Substitute `<version>` with the latest stable version of Pelagia Helm chart.

This command installs Pelagia LCM controllers in the `pelagia` namespace.
If the namespace does not exist, Helm will create it. As a result, the following controllers appear
in the `pelagia` namespace:

```bash
$ kubectl -n pelagia get pods

NAME                                     READY   STATUS    RESTARTS   AGE
pelagia-lcm-controller-6f858b6c5-4bqqr   3/3     Running   0          9m40s
pelagia-lcm-controller-6f858b6c5-6thhn   3/3     Running   0          9m40s
pelagia-lcm-controller-6f858b6c5-whq7x   3/3     Running   0          9m40s
```

The installation command above installs Pelagia in lcm-only mode for an existing Rook Ceph cluster placed
in the `rook-ceph` namespace. If you want to manage a Rook Ceph cluster in a different namespace, set the
`rookConfig.rookNamespace` chart value. For example:
```bash
export PELAGIA_VERSION="<version>"
helm upgrade --install pelagia-ceph oci://registry.mirantis.com/pelagia/pelagia-ceph --version ${PELAGIA_VERSION} --set cephDeployment.enabled=false,rookConfig.configureRook=false,rookConfig.rookNamespace=new-rook-ceph -n pelagia --create-namespace
```

Substitute `<version>` with the latest stable version of Pelagia Helm chart.

## Post-installation

After the installation, create empty `CephDeploymentHealth` custom resource. It stores Ceph cluster state and
Rook API status.

```yaml
apiVersion: lcm.mirantis.com/v1alpha1
kind: CephDeploymentHealth
metadata:
  name: pelagia-ceph
  namespace: pelagia
status: {}
```

Now you can manage Rook Ceph OSD disks and nodes using the Pelagia `CephOsdRemoveTask` custom
resource. For example:

```yaml
apiVersion: lcm.mirantis.com/v1alpha1
kind: CephOsdRemoveTask
metadata:
    name: remove-osd-4
    namespace: pelagia
spec:
  nodes:
    storage-worker-1:
      cleanupByOsd:
      - id: 4
```

This example will dry-run the removal of the OSD with ID 4 on the `storage-worker-1` node.
After the dry-run, the `status` field of the `CephOsdRemoveTask` resource will be updated with the results of the dry-run.

To trigger the OSD removal, set the `spec.approve` field to `true` and apply the resource again. For example:

```yaml
apiVersion: lcm.mirantis.com/v1alpha1
kind: CephOsdRemoveTask
metadata:
    name: remove-osd-4
    namespace: pelagia
spec:
  approve: true
  nodes:
    storage-worker-1:
      cleanupByOsd:
      - id: 4
```

This action triggers the removal of the OSD with ID 4 on the `storage-worker-1` node.

The ``CephOsdRemoveTask`` workflow is as follows:

1. The OSD daemon is stopped and data is rebalanced out from the OSD.
2. The OSD disk is cleaned up on the node.
3. The OSD daemon is removed from the Ceph cluster.

## See also

For the Pelagia architecture and overview,
refer to the [Architecture Guide](https://mirantis.github.io/pelagia/architecture/overview).

For the detailed OSD automated lifecycle management,
refer to [Ops Guide: Automated Lifecycle Management](https://mirantis.github.io/pelagia/ops-guide/lcm/create-task-workflow).
