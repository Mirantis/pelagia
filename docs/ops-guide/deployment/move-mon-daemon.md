<a id="move-mon-daemon-mira"></a>

# Move a Ceph Monitor daemon to another node

This document describes how to migrate a Ceph Monitor daemon from one node to
another without changing the general number of Ceph Monitors in the cluster.
In the Pelagia Controllers concept, migration of a Ceph Monitor means manually
removing it from one node and adding it to another.

Consider the following exemplary placement scheme of Ceph Monitors in the
`nodes` spec of the `CephDeployment` custom resource (CR):
```yaml
spec:
  nodes:
    node-1:
      roles:
      - mon
      - mgr
    node-2:
      roles:
      - mgr
```

Using the example above, if you want to move the Ceph Monitor from `node-1`
to `node-2` without changing the number of Ceph Monitors, the `roles` table
of the `nodes` spec must result as follows:
```yaml
spec:
  nodes:
    node-1:
      roles:
      - mgr
    node-2:
      roles:
      - mgr
      - mon
```

However, due to the Rook limitation related to Kubernetes architecture, once
you move the Ceph Monitor through the `CephDeployment` CR, changes will not
apply automatically. This is caused by the following Rook behavior:

- Rook creates Ceph Monitor resources as deployments with `nodeSelector`,
  which binds Ceph Monitor pods to a requested node.
- Rook does not recreate new Ceph Monitors with the new node placement if the
  current `mon` quorum works.

Therefore, to move a Ceph Monitor to another node, you must also manually apply
the new Ceph Monitors placement to the Ceph cluster as described below.

## Move a Ceph Monitor to another node

1. Open the `CephDeployment` CR for editing:
   ```bash
   kubectl -n pelagia edit cephdpl
   ```

2. In the `nodes` spec of the `CephDeployment` CR, change the `mon` roles placement without changing the total
   number of `mon` roles. For details, see the example above. Note the nodes on which the `mon` roles
   have been removed and save the `name` value of those nodes.

3. Obtain the `rook-ceph-mon` deployment name placed on the obsolete node
   using the previously obtained node name:
   ```bash
   kubectl -n rook-ceph get deploy -l app=rook-ceph-mon -o jsonpath="{.items[?(@.spec.template.spec.nodeSelector['kubernetes\.io/hostname'] == '<nodeName>')].metadata.name}"
   ```

     Substitute `<nodeName>` with the name of the node where you removed the `mon` role.

4. Back up the `rook-ceph-mon` deployment placed on the obsolete node:
   ```bash
   kubectl -n rook-ceph get deploy <rook-ceph-mon-name> -o yaml > <rook-ceph-mon-name>-backup.yaml
   ```

5. Remove the `rook-ceph-mon` deployment placed on the obsolete node:
   ```bash
   kubectl -n rook-ceph delete deploy <rook-ceph-mon-name>
   ```

6. Wait approximately 10 minutes until `rook-ceph-operator` performs a failover of the `Pending` `mon` pod.
   Inspect the logs during the failover process:
   ```bash
   kubectl -n rook-ceph logs -l app=rook-ceph-operator -f
   ```

     Example of log extract:
     ```bash
     2021-03-15 17:48:23.471978 W | op-mon: mon "a" not found in quorum, waiting for timeout (554 seconds left) before failover
     ```

7. If the failover process fails:

     1. Scale down the `rook-ceph-operator` deployment to `0` replicas.
     2. Apply the backed-up `rook-ceph-mon` deployment.
     3. Scale back the `rook-ceph-operator` deployment to `1` replica.

Once done, Rook removes the obsolete Ceph Monitor from the node and creates
a new one on the specified node with a new letter. For example, if the `a`,
`b`, and `c` Ceph Monitors were in quorum and `mon-c` was obsolete, Rook
removes `mon-c` and creates `mon-d`. In this case, the new quorum includes
the `a`, `b`, and `d` Ceph Monitors.
