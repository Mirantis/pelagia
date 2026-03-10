Optional. If you want to add a Ceph OSD on top of a **raw** device that already exists
on a node or is **hot-plugged**, add the required device using the following guidelines:

 - You can add a raw device to a node during node deployment.
 - If a node supports adding devices without a node reboot, you can hot plug
   a raw device to a node.
 - If a node does not support adding devices without a node reboot, you can
   hot plug a raw device during node shutdown.
