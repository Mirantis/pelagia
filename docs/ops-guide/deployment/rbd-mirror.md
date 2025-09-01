<a id="enable-rbd-mirror"></a>

# Enable Ceph RBD mirroring

@Snippet:admonissions:techpreview@

This section describes how to configure and use RADOS Block Device (RBD) mirroring for Ceph pools using
the `rbdMirror` section in the `CephDeployment` custom resource (CR). The feature may be useful if,
for example, you have two interconnected Rockoon clusters. Once you enable RBD mirroring, the images in the specified
pools will be replicated, and if a cluster becomes unreachable, the second one will provide users with instant
access to all images. For details, see [Ceph Documentation: RBD Mirroring](https://docs.ceph.com/en/latest/rbd/rbd-mirroring/).

!!! note

    Pelagia only supports bidirectional mirroring.

## RBD mirror parameters

To enable Ceph RBD monitoring, follow the procedure below and use the following
`rbdMirror` parameters description:

| <div style="width:150px">Parameter</div> | Description                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                          |
|------------------------------------------|----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `daemonsCount`                           | Count of `rbd-mirror` daemons to spawn. We recommend using one instance of the `rbd-mirror` daemon.                                                                                                                                                                                                                                                                                                                                                                                                                  |
| `peers`                                  | Optional. List of mirroring peers of an external cluster to connect to. Only a single peer is supported. The `peer` section includes the following parameters:<br/><ul><li>`site` - the label of a remote Ceph cluster associated with the token.</li><li>`token` - the token that will be used by one site (Ceph cluster) to pull images from the other site. To obtain the token, use the **rbd mirror pool peer bootstrap create** command.</li><li>`pools` - optional, a list of pool names to mirror.</li></ul> |

## Enable Ceph RBD mirroring

1. In `CephDeployment` CRs of both Ceph clusters where you want to enable
   mirroring, specify positive `daemonsCount` in the `rbdMirror` section:
   ```yaml
   spec:
     rbdMirror:
       daemonsCount: 1
   ```

2. On both Ceph clusters where you want to enable mirroring, wait for the Ceph
   RBD Mirror daemons to start running:
   ```bash
   kubectl -n rook-ceph get pod -l app=rook-ceph-rbd-mirror
   ```

3. In `CephDeployment` CRs of both Ceph clusters where you want to enable
   mirroring, specify the `spec.pools.mirroring.mode` parameter for all `pools`
   that must be mirrored.
   ```yaml
   spec:
     pools:
     - name: image-hdd
       ...
       mirroring:
         mode: pool
     - name: volumes-hdd
       ...
       mirroring:
         mode: pool
   ```

4. Obtain the name of an external site to mirror with. On pools with mirroring
   enabled, the name is typically `ceph fsid`:

     ```bash
     kubectl -n rook-ceph exec -it deploy/pelagia-ceph-toolbox -- rbd mirror pool info <mirroringPoolName>
     # or
     kubectl -n rook-ceph exec -it deploy/pelagia-ceph-toolbox -- ceph fsid
     ```
     Substitute `<mirroringPoolName>` with the name of a pool to be mirrored.

5. On an external site to mirror with, create a new bootstrap peer token.
   Execute the following command within the `pelagia-ceph-toolbox` pod CLI:

     ```bash
     kubectl -n rook-ceph exec -it deploy/pelagia-ceph-toolbox -- rbd mirror pool peer bootstrap create <mirroringPoolName> --site-name <siteName>
     ```

     Substitute `<mirroringPoolName>` with the name of a pool to be mirrored.
     In `<siteName>`, assign a label for the external Ceph cluster that will be
     used along with mirroring.

     For details, see [Ceph documentation: Bootstrap peers](https://docs.ceph.com/en/latest/rbd/rbd-mirroring/#bootstrap-peers).

6. In `CephDeployment` CR on the cluster that should mirror pools, specify
   `rbdMirror.peers` with the obtained peer and pools to mirror:
   ```yaml
   spec:
     rbdMirror:
       peers:
       - site: <siteName>
         token: <bootstrapPeer>
         pools: [<mirroringPoolName1>, <mirroringPoolName2>, ...]
   ```
   Substitute `<siteName>` with the label assigned to the external Ceph
   cluster, `<bootstrapPeer>` with the token obtained in the previous step,
   and `<mirroringPoolName>` with names of pools that have the
   `mirroring.mode` parameter defined.

     For example:
     ```yaml
     spec:
       rbdMirror:
         peers:
         - site: cluster-b
           token: <base64-string>
           pools:
           - images-hdd
           - volumes-hdd
           - special-pool-ssd
     ```

7. Verify that mirroring is enabled and each pool with `spec.pools.mirroring.mode` defined has an external peer
   site:
   ```bash
   kubectl -n rook-ceph exec -it deploy/pelagia-ceph-toolbox -- rbd mirror pool info <mirroringPoolName>
   ```
   Substitute `<mirroringPoolName>` with the name of a pool with mirroring enabled.

8. If you have set the `image` mirroring mode in the `pools` section,
   explicitly enable mirroring for each image with `rbd` within the pool:
   ```bash
   kubectl -n rook-ceph exec -it deploy/pelagia-ceph-toolbox -- rbd mirror image enable <poolName>/<imageName> <imageMirroringMode>
   ```
   Substitute `<poolName>` with the name of a pool with the `image`
   mirroring mode, `<imageName>` with the name of an image stored in the
   specified pool. Substitute `<imageMirroringMode>` with one of:
    * `journal` - for mirroring to use the RBD journaling image feature to
      replicate the image contents. If the RBD journaling image feature is not
      yet enabled on the image, it will be enabled automatically.
    * `snapshot` - for mirroring to use RBD image mirror-snapshots to
      replicate the image contents. Once enabled, an initial mirror-snapshot
      will automatically be created. To create additional RBD image
      mirror-snapshots, use the **rbd** command.

     For details, see [Ceph Documentation: Enable image mirroring](https://docs.ceph.com/en/latest/rbd/rbd-mirroring/#enable-image-mirroring).
