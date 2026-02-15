# Welcome to Pelagia

The Pelagia Controller is a [Kubernetes operator](https://kubernetes.io/docs/concepts/extend-kubernetes/operator/)
that implements lifecycle management for Ceph clusters managed by [Rook](https://github.com/rook/rook).

The Pelagia is written in Go lang using [Cluster API](https://github.com/kubernetes-sigs/cluster-api) to build
Kubernetes controllers.

Pelagia solution provides two main controllers:

* **Deployment Controller** monitors changes in the `CephDeployment`
  [Kubernetes custom resource](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/)
   and handles these changes by creating, updating, or deleting appropriate resources in Kubernetes.
* **Ceph OSD Lifecycle Management Controller** monitors changes in the
  `CephOsdRemoveTask` custom resource and runs automated Ceph OSD disk or node removal.

## Quick Start

To get started with Pelagia, follow the
[Quick Start Guide](https://mirantis.github.io/pelagia/quick-start/installation).

To install Pelagia in automated LCM mode only, follow the [Quick Start LCM-only Guide](https://mirantis.github.io/pelagia/quick-start/lcm-installation).

## Getting Help

* File a bug: [https://github.com/Mirantis/pelagia/issues](https://github.com/Mirantis/pelagia/issues)
* Discuss with us: [https://github.com/Mirantis/pelagia/discussions](https://github.com/Mirantis/pelagia/discussions)

## Developer Resources

* Contributing: [https://github.com/Mirantis/pelagia/pulls](https://github.com/Mirantis/pelagia/pulls)
* Developer Guide: [https://mirantis.github.io/pelagia/developer/](https://mirantis.github.io/pelagia/developer/)
* Reference Architecture: [https://mirantis.github.io/pelagia](https://mirantis.github.io/pelagia)
