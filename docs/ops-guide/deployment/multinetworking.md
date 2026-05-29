---
description: How to enable and configure Ceph multinetwork (L3) for using multiple IP networks for Ceph daemons.
keywords: pelagia, enable ceph multinetwork, ceph multi-network, ceph networks, ceph l3 network, ceph daemons
---

<a id="multinetworking-enable-ceph-multinetwork"></a>
# Enable Ceph multinetwork

Ceph allows establishing multiple IP networks and subnet masks for clusters
with configured L3 network rules. You can configure multi-network through the
`cluster.network.addressRanges` section of the `CephDeployment` custom resource (CR). Pelagia Deployment Controller uses this section
to specify the Ceph networks for external access and internal daemon
communication. The parameters in the `cluster.network.addressRanges` section use the CIDR notation,
for example, `10.0.0.0/24`.

Before enabling multiple networks for a Ceph cluster, consider the following
requirements:

* Do not confuse the IP addresses you define with the public-facing IP
  addresses the network clients may use to access the services.
* If you define more than one IP address and subnet mask for the public or
  cluster network, ensure that the subnets within the network can route to
  each other.
* Include each IP address or subnet in the `network` section to IP tables and
  open ports for them as necessary.
* The pods of the Ceph OSD and Ceph RADOS Gateway daemons use cross-pods health checkers
  to verify that the entire Ceph cluster is healthy. Therefore, each CIDR must
  be accessible inside Ceph pods.
* Avoid using the `0.0.0.0/0` CIDR in the `network` section. With a zero
  range in `public` and/or `cluster`, the Ceph daemons behavior
  is unpredictable.

For reference, see [Rook documentation: Network Configuration Settings](https://rook.io/docs/rook/v1.19/CRDs/Cluster/ceph-cluster-crd/#network-configuration-settings).

## Enable multinetwork for Ceph

1. Open `CephDeployment` CR for editing:
      ```bash
      kubectl -n pelagia get cephpdl
      ```

2. In the `cluster` and/or `public` parameters of the `cluster.network.addressRanges` section, define a comma-separated array of CIDRs.
   For example:
   ```yaml
   spec:
    cluster:
     network:
       addressRanges:
         public:
         - 10.12.0.0/24
         - 10.13.0.0/24
         cluster:
         - 10.10.0.0/24
         - 10.11.0.0/24
   ```

3. Exit the editor and apply the changes.

Once done, the specified network CIDRs will be passed to the Ceph daemons pods
through the Rook `rook-config-override` ConfigMap.
