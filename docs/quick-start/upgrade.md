# Upgrade Pelagia

This section provides instructions on how to upgrade Pelagia on an existing Kubernetes cluster.

To upgrade Pelagia, use a new Helm chart version provided in the repository:

```bash
helm upgrade --install pelagia-ceph \
    oci://registry.mirantis.com/pelagia/pelagia-ceph \
    --version <target-version> \
    -n pelagia
```

This command upgrades Pelagia controllers in the `pelagia` namespace or in the namespace defined in `global.namespace`, if set.

Besides its own controllers, Pelagia can deliver updated Rook, `snapshot-controller`, and Ceph manifests along with images. If `cephRelease` is not
pinned in Pelagia values, Pelagia will automatically update a Ceph release if a new release is available.

To pin the Ceph release, specify it in the `cephRelease` field of the Pelagia values file:

```bash
helm upgrade --install pelagia-ceph \
    oci://registry.mirantis.com/pelagia/pelagia-ceph \
    --version <target-version> \
    -n pelagia \
    --set cephRelease=<target-ceph-release-name>
```

Set `<target-ceph-release-name>` in lowercase. An empty value means the latest supported Ceph release.

However, Mirantis does not recommend pinning the Ceph release to ensure you obtain important updates and bug fixes.
