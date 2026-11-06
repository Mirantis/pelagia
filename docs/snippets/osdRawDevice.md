!!! note

    You can add a Ceph OSD on top of a **raw** device that is already present on a node
    or that you add to the node later. To add a raw device to a node:

      - Add the device during node deployment.
      - If the node supports adding devices without a reboot, hot plug the device.
      - If the node does not support it, shut down the node, add the device, and start the node again.
