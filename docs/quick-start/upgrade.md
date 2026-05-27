# Upgrade Pelagia

This section provides instructions on how to upgrade Pelagia on an existing Kubernetes cluster.

To upgrade Pelagia, use a new Helm chart version provided in the repository:

```bash
helm upgrade --install pelagia-ceph \
    oci://registry.mirantis.com/pelagia/pelagia-ceph \
    --version <target-version> \
    -n pelagia
```

This command upgrades Pelagia controllers in the `pelagia` namespace.

Besides its own controllers, Pelagia can deliver updated Rook and Ceph manifests along with images. If `cephRelease` is not
pinned in Pelagia values, Pelagia will automatically update a Ceph version if a new version is available.

To pin the Ceph version, specify it in the `cephRelease` field of the Pelagia values file. For example:

```bash
helm upgrade --install pelagia-ceph \
    oci://registry.mirantis.com/pelagia/pelagia-ceph \
    --version <target-version> \
    -n pelagia \
    --set cephRelease=squid
```

However, Mirantis does not recommend pinning the Ceph version to ensure you obtain important updates and bug fixes.
