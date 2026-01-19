
# What is Pelagia?

Pelagia is a Kubernetes controller that provides all-in-one management
for Ceph clusters installed by [Rook](https://github.com/rook/rook).
It delivers two main features:

* **Aggregates all Rook Custom Resources** (CRs) into a single
  `CephDeployment` resource, simplifying the management of Ceph clusters.
* **Provides automated lifecycle management** (LCM) of Rook Ceph OSD nodes for
  bare-metal clusters. Automated LCM is managed by the special `CephOsdRemoveTask`
  resource.

# Why Pelagia?

Pelagia is designed to simplify the management of Ceph clusters in Kubernetes
installed by [Rook](https://github.com/rook/rook).

Being solid Rook users, we had dozens of Rook CRs to manage. Thus, one day
we decided to create a single resource that would aggregate all Rook CRs and
deliver a smoother LCM experience. This is how Pelagia was born.

It supports almost all Rook CRs API, including `CephCluster`, `CephBlockPool`,
`CephFilesystem`, `CephObjectStore`, and others, aggregating them into a single
specification. We continuously work on improving Pelagia's API, adding new features,
and enhancing existing ones.

Pelagia collects Ceph cluster state and all Rook CRs statuses into single `CephDeploymentHealth` CR.
This resource highlights of Ceph cluster and Rook APIs issues, if any.

Another important thing we implemented in Pelagia is the automated lifecycle
management of Rook Ceph OSD nodes for bare-metal clusters. This feature is
delivered by the `CephOsdRemoveTask` resource, which automates the process
of removing OSD disks and nodes from the cluster. We are using this feature
in our everyday day-2 operations routine.

# How to use Pelagia?

To use Pelagia, you need to install it in your Kubernetes cluster. You can
do this by using the dedicated Helm chart. Once installed, you can create a `CephDeployment`
resource to manage your Ceph cluster.

Pelagia supports installation in Ceph OSD LCM-only mode, which means that you do not
need to install the `CephDeployment` controller if you only want to use
the `CephOsdRemoveTask` resource for automated LCM of Rook Ceph OSD nodes.
The only thing you need to create is empty `CephDeploymentHealth` CR.

You can find the detailed documentation on how to install and use Pelagia
in the [Quick start guide](https://mirantis.github.io/pelagia/quick-start/installation/).

# Version compatibility

| Pelagia version | [Ceph](https://docs.ceph.com/en/latest/releases/) version | [Rook](https://github.com/rook/rook/releases) version | [Ceph-CSI](https://github.com/ceph/ceph-csi/releases) version |
|-----------------|-----------------------------------------------------------|-------------------------------------------------------|---------------------------------------------------------------|
| 1.0.0-1.3.0     | 19.2.3 (Squid), 18.2.7 (Reef)                             | 1.17.4                                                | 3.14.0                                                        |
| 1.4.0           | 19.2.3 (Squid), 18.2.7 (Reef)                             | 1.18.8                                                | 3.15.0                                                        |


# Documentation

For installation, deployment, and administration, see our
[Documentation](https://mirantis.github.io/pelagia/) and
[Quick start guide](https://mirantis.github.io/pelagia/quick-start/installation/).
Have some questions? Use our
[GitHub Discussions](https://github.com/Mirantis/pelagia/discussions).

# Report a Bug

For filing bugs, suggesting improvements, or requesting new features, please
open an [issue](https://github.com/Mirantis/pelagia/issues).
