# Welcome to Pelagia

The Pelagia Controller is a Kubernetes operator that implements lifecycle management for Ceph clusters managed by Rook.

Pelagia is written in Go lang using Cluster API to build Kubernetes controllers.

Pelagia solution provides two main controllers:

* **Deployment Controller** monitors changes in the `CephDeployment` Kubernetes custom resource and handles these changes by creating, updating, or deleting appropriate resources in Kubernetes.
* **Ceph OSD Lifecycle Management Controller** monitors changes in the `CephOsdRemoveTask` custom resource and runs automated Ceph OSD disk or node removal.

## Quick Start

To get started with Pelagia, follow the [Installation guide](./quick-start/installation.md#installation-guide).

To install Pelagia in automated LCM mode only, follow the [LCM-only installation guide](./quick-start/lcm-installation.md#lcmonly-installation-guide).

## Getting Help

* File a bug: [https://github.com/Mirantis/pelagia/issues](https://github.com/Mirantis/pelagia/issues)
* Discuss with us: [https://github.com/Mirantis/pelagia/discussions](https://github.com/Mirantis/pelagia/discussions)

## Developer Resources

* Contributing: [https://github.com/Mirantis/pelagia/pulls](https://github.com/Mirantis/pelagia/pulls)
* Developer Guide: [https://mirantis.github.io/pelagia/developer/](https://mirantis.github.io/pelagia/developer/)
* Reference Architecture: [https://mirantis.github.io/pelagia](https://mirantis.github.io/pelagia)

!!! info "See also"

    * [Kubernetes operator](https://kubernetes.io/docs/concepts/extend-kubernetes/operator/)
    * [Rook](https://github.com/rook/rook)
    * [Cluster API](https://github.com/kubernetes-sigs/cluster-api)
