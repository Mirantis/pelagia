<a id="rook-daemon-place-specify-rook-daemons-placement"></a>

# Specify Rook daemons placement

If you need to configure the placement of Rook daemons on nodes, you can set
extra values to Pelagia Helm chart release.

The procedures in this section describe how to specify the placement of
`rook-ceph-operator`, `rook-discover`, and Ceph CSI pods such as `csi-rbdplugin`, `csi-cephfsplugin`,
`csi-rbdplugin-provisioner` and `csi-cephfsplugin-provisioner`.

## Specify rook-ceph-operator placement

Use the following Pelagia Helm chart values to specify `rook-ceph-operator` placement:

- `rookConfig.rookOperatorPlacement.affinity` is a key-value mapping that contains
  a valid Kubernetes `affinity` specification.
- `rookConfig.rookOperatorPlacement.nodeSelector` is a key-value mapping that contains
  a valid Kubernetes `nodeSelector` specification.
- `rookConfig.rookOperatorPlacement.tolerations` is a list that contains valid Kubernetes `toleration` items.

Upgrade Pelagia Helm release with the desired placement values by setting them directly to release:
```bash
helm upgrade --install pelagia-ceph oci://registry.mirantis.com/pelagia/pelagia-ceph --version 1.0.0 -n pelagia \
  --set rookConfig.rookOperatorPlacement.affinity=<rookOperatorAffinity>,\
        rookConfig.rookOperatorPlacement.nodeSelector=<rookOperatorNodeSelector>,\
        rookConfig.rookOperatorPlacement.tolerations=<rookOperatorTolerations>
```

After upgrading Pelagia Helm release, wait for some time and verify that the changes have applied:
```bash
kubectl -n rook-ceph get deploy rook-ceph-operator -o yaml
```

## Specify Ceph CSI pods placement

Use the following Pelagia Helm chart values to specify Ceph CSI pods placement:

- `rookConfig.csiPlacement.nodeAffinity.csiprovisioner` is a valid Kubernetes label selector expression.
- `rookConfig.csiPlacement.nodeAffinity.csiplugin` is a valid Kubernetes label selector expression. Default is
  `ceph-daemonset-available-node=true`.
- `rookConfig.csiPlacement.csiplugin.tolerations` is a string which contains a valid list of Kubernetes `toleration`
  items. For example:
  ```yaml
  csiplugin: |
    - effect: NoSchedule
      key: node-role.kubernetes.io/controlplane
      operator: Exists
  ```

- `rookConfig.csiPlacement.csiprovisioner.tolerations` is a string which contains a valid list of Kubernetes
  `toleration` items. For example:
  ```yaml
  csiprovisioner: |
    - effect: NoSchedule
      key: node-role.kubernetes.io/controlplane
      operator: Exists
  ```

Upgrade Pelagia Helm release with the desired placement values by setting them directly to release:
```bash
helm upgrade --install pelagia-ceph oci://registry.mirantis.com/pelagia/pelagia-ceph --version 1.0.0 -n pelagia \
  --set rookConfig.csiPlacement.nodeAffinity.csiprovisioner="<nodeAffinityLabelSelector>"
```

After upgrading Pelagia Helm release, wait for some time and verify that the changes have applied:
```bash
kubectl -n rook-ceph get ds csi-rbdplugin -o yaml
kubectl -n rook-ceph get ds csi-cephfsplugin -o yaml
kubectl -n rook-ceph get deploy csi-rbdplugin-provisioner -o yaml
kubectl -n rook-ceph get deploy csi-cephfsplugin-provisioner -o yaml
```

## Specify rook-discover placement

Use the following Pelagia Helm chart values to specify `rook-discover` placement:

- `rookConfig.rookDiscoverPlacement.nodeAffinity` is a valid Kubernetes label selector expression. Default is
  `ceph-daemonset-available-node=true;ceph_role_osd=true`.
- `rookConfig.rookDiscoverPlacement.tolerations` is a string which contains a valid list of Kubernetes `toleration`
  items. For example:
  ```yaml
  tolerations: |
    - effect: NoSchedule
      key: node-role.kubernetes.io/controlplane
      operator: Exists
  ```

Upgrade Pelagia Helm release with the desired placement values by setting them directly to release:
```bash
helm upgrade --install pelagia-ceph oci://registry.mirantis.com/pelagia/pelagia-ceph --version 1.0.0 -n pelagia \
  --set rookConfig.rookDiscoverPlacement.nodeAffinity="<nodeAffinityLabelSelector>"
```

After upgrading Pelagia Helm release, wait for some time and verify that the changes have applied:
```bash
kubectl -n rook-ceph get ds rook-discover -o yaml
```
