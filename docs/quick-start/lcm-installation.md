<a id="lcm-installation-install-pelagia-in-the-lcm-only-mode"></a>
# Install Pelagia in the LCM-only mode

This section provides instructions on how to install Pelagia in lifecycle-management-only (LCM-only) mode.
This mode allows you to automatically remove Rook Ceph OSD disks and nodes in your Kubernetes cluster.

## Prerequisites

* A Kubernetes cluster, for example, deployed with k0s.
* A deployed Ceph cluster managed by Rook.

## Installation

To install Pelagia in LCM-only mode, use the Helm chart provided in the repository:

```bash
export PELAGIA_VERSION="<version>"
helm upgrade --install pelagia-ceph oci://registry.mirantis.com/pelagia/pelagia-ceph --version ${PELAGIA_VERSION} --set cephDeployment.enabled=false,rookConfig.configureRook=false -n pelagia --create-namespace
```

Substitute `<version>` with the latest stable version of Pelagia Helm chart.

The `cephDeployment.enabled=false` value disables Pelagia Deployment Controller, and the `rookConfig.configureRook=false`
value disables the Rook Operator deployment, so that Pelagia uses the existing Rook installation.

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

The installation command above installs Pelagia in LCM-only mode for an existing Rook Ceph cluster placed
in the `rook-ceph` namespace. If you want to manage a Rook Ceph cluster in a different namespace, set the
`rookConfig.rookNamespace` chart value. For example:
```bash
export PELAGIA_VERSION="<version>"
helm upgrade --install pelagia-ceph oci://registry.mirantis.com/pelagia/pelagia-ceph --version ${PELAGIA_VERSION} --set cephDeployment.enabled=false,rookConfig.configureRook=false,rookConfig.rookNamespace=new-rook-ceph -n pelagia --create-namespace
```

Substitute `<version>` with the latest stable version of Pelagia Helm chart.

## Post-installation

After the installation, create an empty `CephDeploymentHealth` custom resource. It stores Ceph cluster state and
Rook API status.

The `CephDeploymentHealth` resource name must match the name of the existing Rook `CephCluster` resource,
because Pelagia looks up the Ceph cluster by this name in the Rook namespace. The examples in this documentation
use the `pelagia-ceph` name. In a full Pelagia installation, Pelagia creates the `CephCluster` and
`CephDeploymentHealth` resources with the name of the `CephDeployment` resource, so all of them are named
`pelagia-ceph`. In the LCM-only mode, you create `CephDeploymentHealth` yourself, so replace `pelagia-ceph` with
the name of your `CephCluster` resource. To obtain the name, run:

```bash
kubectl -n rook-ceph get cephcluster
```

Then create the `CephDeploymentHealth` resource with that name. For example:

```yaml
apiVersion: lcm.mirantis.com/v1alpha1
kind: CephDeploymentHealth
metadata:
  name: pelagia-ceph
  namespace: pelagia
status: {}
```

After you create the `CephDeploymentHealth` resource, Pelagia Infra Controller deploys the following components:

* The `pelagia-ceph-toolbox` Deployment in the Rook namespace. Pelagia uses it to run Ceph CLI commands.
* The `pelagia-disk-daemon` DaemonSet in the `pelagia` namespace. It runs on nodes that match the
  `lcmConfig.diskDaemonNodeSelector` chart value, `ceph_role_osd=true` by default, and provides disk information
  from these nodes.

In a full Pelagia installation, Pelagia Deployment Controller adds the `ceph_role_osd=true` label to the nodes
with Ceph OSDs. In the LCM-only mode, add this label yourself to every node with Ceph OSDs. If you changed the
`lcmConfig.diskDaemonNodeSelector` chart value, use that label instead:

```bash
kubectl label node <node-name> ceph_role_osd=true
```

Without the label, the disk daemon does not run on the node and Pelagia cannot get disk information from it.
In this case, `CephOsdRemoveTask` skips the node with a warning if the node is listed in the `spec.storage.nodes`
section of the `CephCluster` resource. Otherwise, Pelagia removes the OSDs listed in `cleanupByOsd` from the Ceph
cluster but does not clean up their disks.

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

This example creates a task to remove the OSD with ID 4 on the `storage-worker-1` node. Pelagia first validates
the task and does not remove anything until you approve it. After the validation, the task moves to the
`ApproveWaiting` phase, and the `status.removeInfo` field of the `CephOsdRemoveTask` resource lists the OSDs and
devices that Pelagia is going to remove.

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

1. Pelagia scales Rook Operator down to `0` replicas for the time of the removal.
2. Pelagia sets the CRUSH weight of the OSD to `0`, and Ceph moves the data out of the OSD.
3. After the rebalance completes, Pelagia stops the OSD daemon and removes the OSD from the Ceph cluster.
4. Pelagia runs a job that cleans up the OSD disk on the node.
5. Pelagia removes the `rook-ceph-osd-<ID>` deployment of the OSD.

After the task completes, Pelagia scales Rook Operator back up to `1` replica. If the task fails, Rook Operator
stays stopped until you set the `spec.resolved` field of the task to `true`.
