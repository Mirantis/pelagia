- `name` - Required. Ceph Object Storage instance name.
- `usedForOpenstack` - marker that current instance of Ceph RGW will be consumed by Openstack.
  This will enable ability to put required Openstack parameters to Ceph RGW configuration.
- `auxilaryService` - marker that current instance of Ceph RGW does not require `StorageClass`
  and external API access. Usefull for deploying such Ceph RGW instances as: Ceph RGW multisite
  replication daemon, Ceph RGW admin API access.
- `usedByIngress` - marker that current instance of Ceph RGW is used by Ingress controller.
  Deprecated in favor of using Gateway API `HTTPRoutes`.
- `spec` - represents Rook `CephObjectStore` specification. For details see [CephObjectStore CRD](https://rook.io/docs/rook/v1.19/CRDs/Object-Storage/ceph-object-store-crd/) and [CephObjectStore API](https://rook.io/docs/rook/v1.19/CRDs/specification/#ceph.rook.io/v1.CephObjectStore) specification.
