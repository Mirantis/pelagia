<a id="move-mon-node-replace-move-ceph-monitor-before-node-replacement"></a>

# Move Ceph Monitor before node replacement

This document describes how to migrate a Ceph Monitor to another node
on bare metal-based clusters before node replacement.

!!! warning

    Remove the Ceph Monitor role before the machine removal.

!!! warning

    Make sure that the Ceph cluster always has an odd number of Ceph Monitors.

The procedure of a Ceph Monitor migration assumes that you temporarily move
the Ceph Manager/Monitor to another (for example, worker) node. After a node replacement, we
recommend migrating the Ceph Manager/Monitor to the new manager node.

**To migrate a Ceph Monitor to another machine:**

1. Move the Ceph Manager/Monitor daemon from the affected node to one of the worker machines as described in [Move a Ceph Monitor daemon to another node](./move-mon-daemon.md#move-mon-daemon-move-a-ceph-monitor-daemon-to-another-node).
2. Delete the affected node.
3. Add a new manager node without the Monitor and Manager role.

    !!! warning

         The addition of a new node with the Monitor and Manager role breaks the odd number quorum of Ceph Monitors.

4. Move the previously migrated Ceph Manager/Monitor daemon to the new manager node as described in [Move a Ceph Monitor daemon to another node](./move-mon-daemon.md#move-mon-daemon-move-a-ceph-monitor-daemon-to-another-node).
