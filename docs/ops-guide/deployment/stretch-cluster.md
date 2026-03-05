<a id="stretch-cluster-stretch-ceph-cluster"></a>
# Stretch Ceph cluster

{% include "../../snippets/techpreview.md" %}

This section describes how to deploy a Rook Ceph stretch cluster using Pelagia. A stretch cluster provides data availability when only two failure domains (zones) are available for data, plus a third arbiter zone that runs a single monitor for quorum.

See the [Rook stretch cluster design](https://github.com/rook/rook/blob/release-1.19/design/ceph/ceph-stretch-cluster.md) and [Stretch Storage Cluster](https://rook.io/docs/rook/latest-release/CRDs/Cluster/stretch-cluster/) documentation for background.

## Overview

- **Three zones**: two "data" zones (A and B) and one "arbiter" zone.
- **Five mons**: two mons in each data zone, one mon in the arbiter zone.
- **OSDs**: run only in the two data zones; the arbiter zone typically has no OSDs.

Pelagia configures the Rook `CephCluster` with `mon.stretchCluster` and restricts OSD placement to the data zones via node affinity.

## Prerequisites

- At least two nodes in each data zone and at least one node in the arbiter zone (for mons).
- OSDs only on nodes in the data zones.

## CephDeployment configuration

Set `spec.stretchCluster` on your `CephDeployment`:

```yaml
apiVersion: lcm.mirantis.com/v1alpha1
kind: CephDeployment
metadata:
  name: pelagia-ceph
  namespace: pelagia
spec:
  stretchCluster:
    failureDomainTopology: zone
    subFailureDomain: host
    zones:
      - name: arbiter
        arbiter: true
      - name: zone-a
      - name: zone-b
  nodes:
  - name: control-node-0
    crush:
      zone: arbiter
    roles:
    - mon
  - name: worker-node-zone-a-0
    crush:
      zone: zone-a
    roles:
    - mon
    - mgr
    devices: [...]
  - name: worker-node-zone-a-1
    crush:
      zone: zone-a
    roles:
    - mon
    devices: [...]
  - name: worker-node-zone-b-0
    crush:
      zone: zone-b
    roles:
    - mon
    - mgr
    devices: [...]
  - name: worker-node-zone-b-1
    crush:
      zone: zone-b
    roles:
    - mon
    devices: [...]
  pools:
  # For stretch, use failureDomain zone, size 4, replicasPerFailureDomain 2
  - name: kubernetes
    default: true
    deviceClass: hdd
    failureDomain: zone
    replicated:
      size: 4
      replicasPerFailureDomain: 2
  # ...
```

### Stretch cluster fields

| Field | Required | Description |
|-------|----------|-------------|
| `failureDomainTopology` | Yes | Use a short name (e.g. `zone`) or the full Kubernetes node label (e.g. `topology.kubernetes.io/zone`). Must match a topology label used by OSDs. |
| `subFailureDomain` | No | Failure domain within a zone (e.g. `host`). Used by Rook for CRUSH. |
| `zones` | Yes | Exactly three zones. Exactly one zone must have `arbiter: true`. |

### Pools for stretch clusters

For data protection in a stretch cluster, use:

- **Failure domain**: the same as your zone label type (e.g. `zone` if using `topology.kubernetes.io/zone`).
- **Replicated size**: `4`.
- **Replicas per failure domain**: `2` (two replicas in each data zone).

Example:

```yaml
replicated:
  size: 4
  replicasPerFailureDomain: 2
```

Erasure-coded pools are not supported for Ceph stretch clusters.
