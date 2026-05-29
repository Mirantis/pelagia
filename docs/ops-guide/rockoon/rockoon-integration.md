---
description: How to integrate Pelagia with Rockoon OpenStack Controller.
keywords: pelagia, rockoon, pelagia rockoon integration, ceph pools, openstack
  integration, cephdeployment
---

<a id="rockoon-integration-integrate-pelagia-with-rockoon"></a>
# Integrate Pelagia with Rockoon

This document describes how to integrate Pelagia with the Rockoon OpenStack Controller.
As Ceph supports integration with OpenStack, Pelagia provides a way to integrate Rook Ceph cluster with
Rockoon OpenStack inside the Kubernetes cluster.

## How to integrate Pelagia with Rockoon

In the `CephDeployment` custom resource, create the following Ceph pools required for Rockoon OpenStack and ensure that the `role` parameter is explicitly set:

- `volumes` for the OpenStack Block Storage service (`cinder`)
- `backup` for the OpenStack Block Storage service (`cinder`)
- `vms` for the OpenStack Compute service (`nova`)
- `images` for the OpenStack Image service (`glance`)

  For example:

  ```yaml
  spec:
    blockStorage:
      pools:
      ...
      - name: volumes
        role: volumes
        spec:
          deviceClass: hdd
          replicated:
            size: 3
      - name: backup
        role: backup
        spec:
          deviceClass: hdd
          replicated:
            size: 3
      - name: vms
        role: vms
        spec:
          deviceClass: hdd
          replicated:
            size: 3
      - name: images
        role: images
        spec:
          deviceClass: hdd
          replicated:
            size: 3
  ```

As a result, Pelagia creates the following Ceph pools: `volumes-hdd`, `backup-hdd`, `vms-hdd`, and
`images-hdd`. We recommend the following target ratios for these pools to match the default OpenStack requirements:

  - Volumes pool: 0.4
  - Backup pool:  0.1
  - VMs pool:     0.2
  - Images pool:  0.1

We recommend adjusting these ratios according to your OpenStack
deployment requirements using the `parameters.target_size_ratio` parameter located in the `pools` section.
For reference, see [Rook documentation: CephBlockPool CRD Spec](https://rook.io/docs/rook/v1.19/CRDs/Block-Storage/ceph-block-pool-crd/#spec).
For details on how to set correct values, see [Calculate target ratios](calc-target-ratio.md).

After Ceph pools are created, Pelagia Deployment Controller creates a secret in the `openstack-ceph-shared`
namespace with all necessary information for Rockoon OpenStack services to be configured
with the Ceph cluster. Rockoon Controller watches this namespace and
transforms the secret into the data structures expected by OpenStack Helm charts. After that, OpenStack
services will be connected to the desired Ceph cluster.

If `CephDeployment` contains the `objectStorage` section and Ceph Object Storage is deployed, then Pelagia and Rockoon
enable Ceph RADOS Gateway integration with OpenStack Object Storage service (`swift`).

!!! info "See also"

    [Architecture: Pelagia integration with Rockoon](../../architecture/rockoon-integration.md)
