---
description: How to specify placement of Rook daemons and Ceph CSI pods using
  Pelagia Helm chart values.
keywords: pelagia, rook daemon placement, ceph csi placement, rook-ceph-operator,
  rook-discover, csi pods, helm chart
---

<a id="rook-daemon-place-specify-rook-daemons-placement"></a>

# Specify Rook daemons placement

If you need to configure the placement of Rook daemons on nodes, you can set
extra values to Pelagia Helm chart release.

The procedures in this section describe how to specify the placement of
`rook-ceph-operator`, `rook-discover`, and Ceph CSI pods such as:

- `<rookNamespace>.cephfs.csi.ceph.com-ctrlplugin`
- `<rookNamespace>.cephfs.csi.ceph.com-nodeplugin`
- `<rookNamespace>.rbd.csi.ceph.com-ctrlplugin`
- `<rookNamespace>.rbd.csi.ceph.com-nodeplugin`

Where `<rookNamespace>` is the Rook namespace. For example, in the default `rook-ceph`
namespace, the CephFS CSI controller plugin is named `rook-ceph.cephfs.csi.ceph.com-ctrlplugin`.

## Specify rook-ceph-operator placement

Use the following Pelagia Helm chart values to specify `rook-ceph-operator` placement:

- `rook.rookConfig.rookOperatorPlacement.affinity` is a key-value mapping that contains
  a valid Kubernetes `affinity` specification.
- `rook.rookConfig.rookOperatorPlacement.nodeSelector` is a key-value mapping that contains
  a valid Kubernetes `nodeSelector` specification.
- `rook.rookConfig.rookOperatorPlacement.tolerations` is a list that contains valid Kubernetes `toleration` items.

Upgrade Pelagia Helm release with the desired placement values by setting them directly to release:
```bash
helm upgrade --install pelagia-ceph oci://registry.mirantis.com/pelagia/pelagia-ceph --version ${PELAGIA_VERSION} -n pelagia \
  --set rook.rookConfig.rookOperatorPlacement.affinity=<rookOperatorAffinity>,\
        rook.rookConfig.rookOperatorPlacement.nodeSelector=<rookOperatorNodeSelector>,\
        rook.rookConfig.rookOperatorPlacement.tolerations=<rookOperatorTolerations>
```

After upgrading Pelagia Helm release, wait for some time and verify that the changes have applied:
```bash
kubectl -n rook-ceph get deploy rook-ceph-operator -o yaml
```

## Specify Ceph CSI pods placement

Use the following Pelagia Helm chart values to specify Ceph CSI pods placement:

- `cephDeployment.csi.placement.nodeAffinity.controllerPlugin` is a valid Kubernetes label selector expression
  for the CSI controller plugin deployments.
- `cephDeployment.csi.placement.nodeAffinity.nodePlugin` is a valid Kubernetes label selector expression for the
  CSI node plugin DaemonSets. Default is `ceph-daemonset-available-node=true`.
- `cephDeployment.csi.placement.tolerations.nodePlugin` is a string which contains a valid list of Kubernetes
  `toleration` items for the CSI node plugin DaemonSets. For example:
  ```yaml
  nodePlugin: |
    - effect: NoSchedule
      key: node-role.kubernetes.io/controlplane
      operator: Exists
  ```

- `cephDeployment.csi.placement.tolerations.controllerPlugin` is a string which contains a valid list of Kubernetes
  `toleration` items for the CSI controller plugin deployments. For example:
  ```yaml
  controllerPlugin: |
    - effect: NoSchedule
      key: node-role.kubernetes.io/controlplane
      operator: Exists
  ```

!!! note

    These values take effect only if `cephDeployment.csi.manage` is set to `true`, which is the default.

Upgrade Pelagia Helm release with the desired placement values by setting them directly to release:
```bash
helm upgrade --install pelagia-ceph oci://registry.mirantis.com/pelagia/pelagia-ceph --version ${PELAGIA_VERSION} -n pelagia \
  --set cephDeployment.csi.placement.nodeAffinity.controllerPlugin="<nodeAffinityLabelSelector>"
```

After upgrading Pelagia Helm release, wait for some time and verify that the changes have applied:
```bash
kubectl -n <rookNamespace> get ds <rookNamespace>.rbd.csi.ceph.com-nodeplugin -o yaml
kubectl -n <rookNamespace> get ds <rookNamespace>.cephfs.csi.ceph.com-nodeplugin -o yaml
kubectl -n <rookNamespace> get deploy <rookNamespace>.rbd.csi.ceph.com-ctrlplugin -o yaml
kubectl -n <rookNamespace> get deploy <rookNamespace>.cephfs.csi.ceph.com-ctrlplugin -o yaml
```

## Specify rook-discover placement

Use the following Pelagia Helm chart values to specify `rook-discover` placement:

- `rook.rookConfig.rookDiscoverPlacement.nodeAffinity` is a valid Kubernetes label selector expression. Default is
  `ceph-daemonset-available-node=true;ceph_role_osd=true`.
- `rook.rookConfig.rookDiscoverPlacement.tolerations` is a string which contains a valid list of Kubernetes `toleration`
  items. For example:
  ```yaml
  tolerations: |
    - effect: NoSchedule
      key: node-role.kubernetes.io/controlplane
      operator: Exists
  ```

Upgrade Pelagia Helm release with the desired placement values by setting them directly to release:
```bash
helm upgrade --install pelagia-ceph oci://registry.mirantis.com/pelagia/pelagia-ceph --version ${PELAGIA_VERSION} -n pelagia \
  --set rook.rookConfig.rookDiscoverPlacement.nodeAffinity="<nodeAffinityLabelSelector>"
```

After upgrading Pelagia Helm release, wait for some time and verify that the changes have applied:
```bash
kubectl -n rook-ceph get ds rook-discover -o yaml
```
