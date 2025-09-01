<a id="l3-ceph"></a>

# Enable Ceph multinetwork

Ceph allows establishing multiple IP networks and subnet masks for clusters
with configured L3 network rules. You can configure multi-network through the
`network` section of the `CephDeployment` custom resource (CR). Pelagia Deployment Controller uses this section
to specify the Ceph networks for external access and internal daemon
communication. The parameters in the `network` section use the CIDR notation,
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
  range in `publicNet` and/or `clusterNet`, the Ceph daemons behavior
  is unpredictable.

## Enable multinetwork for Ceph

1. Open `CephDeployment` CR for editing:
      ```bash
      kubectl -n pelagia get cephpdl
      ```

2. In the `clusterNet` and/or `publicNet` parameters of the `network` section, define a comma-separated array of CIDRs.
   For example:
   ```yaml
   spec:
     network:
       publicNet:  10.12.0.0/24,10.13.0.0/24
       clusterNet: 10.10.0.0/24,10.11.0.0/24
   ```

3. Exit the editor and apply the changes.

Once done, the specified network CIDRs will be passed to the Ceph daemons pods
through the Rook `rook-config-override` ConfigMap.
