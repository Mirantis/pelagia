<a id="verify-ceph-resource-mgmt"></a>
# Verify Ceph tolerations and resources

After you enable Ceph resources management as described in
[Enable management of Ceph tolerations and resources](./enable-resource-mgmt.md#enable-resource-mgmt),
perform the steps below to verify that the configured tolerations, requests, or limits have been successfully
specified in the Ceph cluster.

## Verify Ceph tolerations and resources

* To verify that the required tolerations are specified in the Ceph cluster,
  inspect the output of the following commands:
  ```bash
  kubectl -n rook-ceph get $(kubectl -n rook-ceph get cephcluster -o name) -o jsonpath='{.spec.placement.mon.tolerations}'
  kubectl -n rook-ceph get $(kubectl -n rook-ceph get cephcluster -o name) -o jsonpath='{.spec.placement.mgr.tolerations}'
  kubectl -n rook-ceph get $(kubectl -n rook-ceph get cephcluster -o name) -o jsonpath='{.spec.placement.osd.tolerations}'
  ```

* To verify RADOS Gateway tolerations:
  ```bash
  kubectl -n rook-ceph get $(kubectl -n rook-ceph get cephobjectstore -o name) -o jsonpath='{.spec.gateway.placement.tolerations}'
  ```

* To verify that the required resource requests or limits are specified for
  the Ceph `mon`, `mgr`, or `osd` daemons, inspect the output of the
  following command:
  ```bash
  kubectl -n rook-ceph get $(kubectl -n rook-ceph get cephcluster -o name) -o jsonpath='{.spec.resources}'
  ```

* To verify that the required resource requests and limits are specified for
  the RADOS Gateway daemons, inspect the output of the following command:
  ```bash
  kubectl -n rook-ceph get $(kubectl -n rook-ceph get cephobjectstore -o name) -o jsonpath='{.spec.gateway.resources}'
  ```

* To verify that the required resource requests or limits are specified for
  the Ceph OSDs `hdd`, `ssd`, or `nvme` device classes, perform the
  following steps:

    1. Identify which Ceph OSDs belong to the `<deviceClass>` device class in
       question:
       ```bash
       kubectl -n rook-ceph exec -it deploy/rook-ceph-tools -- ceph osd crush class ls-osd <deviceClass>
       ```

    2. For each `<osdID>` obtained in the previous step, run the following
       command. Compare the output with the desired result.
       ```bash
       kubectl -n rook-ceph get deploy rook-ceph-osd-<osdID> -o jsonpath='{.spec.template.spec.containers[].resources}'
       ```
