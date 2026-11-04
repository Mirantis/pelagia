# Upgrade Pelagia

Complete the following steps to upgrade Pelagia on an existing Kubernetes cluster:

1. Read upgrade notes of the target Pelagia version described in [Upgrade notes](./upgrade-notes.md) and complete pre-upgrade steps, if any.
2. Verify the Pelagia setup and Ceph cluster health as described in [Verify Ceph](../ops-guide/verify-ceph/index.md).
3. Upgrade Pelagia using a new Helm chart version provided in the repository:
   ```bash
   helm upgrade --install pelagia-ceph \
       oci://registry.mirantis.com/pelagia/pelagia-ceph \
       --version <target-version> \
       -n pelagia
   ```

     This command upgrades Pelagia controllers in the `pelagia` namespace or in the namespace defined in `global.namespace`, if set.

     Besides its own controllers, Pelagia can deliver updated Rook, Ceph CSI Operator, `snapshot-controller`, and Ceph manifests along with images.
     If `cephRelease` is not pinned in Pelagia values, Pelagia will automatically update a Ceph release if a new release is available.

     To pin the Ceph release in the `cephRelease` field of the Pelagia values file:
     ```bash
     helm upgrade --install pelagia-ceph \
         oci://registry.mirantis.com/pelagia/pelagia-ceph \
         --version <target-version> \
         -n pelagia \
         --set cephRelease=<target-ceph-release-name>
     ```

     Set `<target-ceph-release-name>` in lowercase. An empty value means the latest supported Ceph release.

      However, we do not recommend pinning the Ceph release to ensure you obtain important updates and bug fixes.

4. Verify the Pelagia setup and Ceph cluster health as described in [Verify Ceph](../ops-guide/verify-ceph/index.md).
5. Complete post-upgrade steps, if any. For details, see [Upgrade notes](./upgrade-notes.md).
