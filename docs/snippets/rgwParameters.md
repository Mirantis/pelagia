- `name` - Mandatory. Ceph Object Storage instance name.
- `usedForOpenstack` - Optional. Enables consumption of the current Ceph RGW instance by OpenStack,
  which allows adding the required OpenStack parameters to the Ceph RGW configuration.
- `auxiliaryService` - Optional. Enables the current Ceph RGW instance to operate without a `StorageClass`
  and external API access. Useful for auxiliary instances, such as a Ceph RGW multisite
  replication daemon or an instance providing Ceph RGW admin API access.
- `servedByIngress` - Optional. Enables the current Ceph RGW instance to be used by the Ingress controller.
  Deprecated in favor of the Gateway API `HTTPRoutes`.
- `spec` - Mandatory. Represents the Rook `CephObjectStore` specification. For details, see [CephObjectStore CRD](https://rook.io/docs/rook/v1.19/CRDs/Object-Storage/ceph-object-store-crd/) and [CephObjectStore API specification](https://rook.io/docs/rook/v1.19/CRDs/specification/#ceph.rook.io/v1.CephObjectStore).
