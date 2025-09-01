# Pelagia Limitations

A Pelagia `CephDeployment` configuration includes but is
not limited to the following limitations:

- The replication size for any Ceph pool must be set to more than 1.
- Only one CRUSH tree per cluster. The separation of devices per Ceph pool is
  supported through [Ceph Device Classes](https://docs.ceph.com/en/latest/rados/operations/crush-map/#device-classes)
  with only one pool of each type for a device class.
- Only the following types of CRUSH buckets are supported:

    - `topology.kubernetes.io/region`
    - `topology.kubernetes.io/zone`
    - `topology.rook.io/datacenter`
    - `topology.rook.io/room`
    - `topology.rook.io/pod`
    - `topology.rook.io/pdu`
    - `topology.rook.io/row`
    - `topology.rook.io/rack`
    - `topology.rook.io/chassis`

- Only IPv4 is supported.
- If two or more Ceph OSDs are located on the same device, there must be no
  dedicated WAL or DB for this class.
- Only full collocation or dedicated WAL and DB configurations are supported.
- The minimum size of any defined Ceph OSD device is `5G`.
- Ceph OSDs support only raw disks as data devices meaning that no `dm` or
  `lvm` devices are allowed.
- Ceph cluster does not support removable devices (with hotplug enabled) for
  deploying Ceph OSDs.
- When adding a Ceph node with the Ceph Monitor role, if any issues occur with
  the Ceph Monitor, Rook Ceph Operator removes it and adds a new Ceph Monitor instead,
  named using the next alphabetic character in order. Therefore, the Ceph Monitor
  names may not follow the alphabetic order. For example, `a`, `b`, `d`,
  instead of `a`, `b`, `c`.
- Reducing the number of Ceph Monitors is not supported and causes the Ceph
  Monitor daemons removal from random nodes.
- Removal of the `mgr` role in the `nodes` section of the
  `CephDeployment` CR does not remove Ceph Managers. To remove a Ceph
  Manager from a node, remove it from the `nodes` spec and manually delete
  the `rook-ceph-mgr` pod in the Rook namespace.
