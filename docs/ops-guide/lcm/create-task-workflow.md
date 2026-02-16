<a id="create-osd-rm-request"></a>
# Creating a Ceph OSD remove task

The workflow of creating a Ceph OSD removal task includes the following steps:

1. Removing obsolete nodes or disks from the `spec.nodes` section of the `CephDeployment` custom resource (CR) as described in [Nodes parameters](../../architecture/custom-resources/cephdeployment.md#cephdpl-nodes).

    !!! note

          Note the names of the removed nodes, devices or their paths exactly as they were specified
          in `CephDeployment` for further usage.

2. Creating a YAML template for the `CephOsdRemoveTask` CR. For details, see [CephOsdRemoveTask custom resource](../../architecture/custom-resources/cephosdremovetask.md#cephosdremovetask-custom-resource).

     - If `CephOsdRemoveTask` contains information about Ceph OSDs to remove in a proper format,
       the information will be validated to eliminate human error and avoid a wrong Ceph OSD removal.
     - If the `nodes` section of `CephOsdRemoveTask` is empty, the Pelagia LCM Controller will automatically
       detect Ceph OSDs for removal, if any. Auto-detection is based not only on the information
       provided in the Rook `CephCluster` CR but also on the information from the Ceph cluster itself.

     Once the validation or auto-detection completes, the entire information about the Ceph OSDs to remove appears
     in the `CephOsdRemoveTask` object: hosts they belong to, OSD IDs, disks, partitions, and so on. The
     request then moves to the `ApproveWaiting` phase until the cloud operator manually specifies the `approve`
     flag in the spec.

    ???+ "Example of the `CephOsdRemoveTask` custom resource"
        ```yaml
        apiVersion: lcm.mirantis.com/v1alpha1
        kind: CephOsdRemoveTask
        metadata:
          name: remove-osd-3-4-task
          namespace: pelagia
        spec:
          nodes:
            worker-3:
              cleanupByDevice:
              - device: sdb
              - device: /dev/disk/by-path/pci-0000:00:1t.9
        ```

    ???+ "Example of the `CephOsdRemoveTask` custom resource to find all ready to remove Ceph OSDs"
        ```yaml
        apiVersion: lcm.mirantis.com/v1alpha1
        kind: CephOsdRemoveTask
        metadata:
          generateName: remove-osds
          namespace: pelagia
        spec:
          nodes: {}
        ```

3. Manually adding an affirmative `approve` flag in the `CephOsdRemoveTask` spec. Once done, Pelagia Controllers and
   Rook Ceph Operator reconciliation pause until the task is handled and execute the following:

      - Stops regular Rook Ceph Operator orchestration. Also, Pelagia Deployment Controller pauses its reconcile.
      - Removes Ceph OSDs.
      - Runs batch jobs to clean up the device, if possible.
      - Removes host information from the Ceph cluster if the entire Ceph node is removed.
      - Marks the task with an appropriate result with a description of occurred issues.

    !!! note

         If the task completes successfully, Rook Ceph Operator and Pelagia Deployment Controller reconciliation
         resumes. Otherwise, it remains paused until the issue is resolved.

4. Reviewing the Ceph OSD removal status. For details, see [Status fields](../../architecture/custom-resources/cephosdremovetask.md#status-fields)

5. Manual removal of device cleanup jobs.

    !!! note

         Device cleanup jobs are not removed automatically and are kept in Pelagia namespace along with pods containing
         information about the executed actions. The jobs have the following labels:
         ```yaml
         labels:
           app: pelagia-lcm-cleanup-disks
           host: <HOST-NAME>
           osd: <OSD-ID>
           rook-cluster: <ROOK-CLUSTER-NAME>
         ```

         Additionally, jobs are labeled with disk names that will be cleaned up, such as `sdb=true`.
         You can remove a single job or a group of jobs using any label described above, such as host, disk, and so on.

!!! info "See also"

    [CephOsdRemoveRequest failure with a timeout during rebalance](../../../troubleshoot/cephosdremovetask-timeout)
