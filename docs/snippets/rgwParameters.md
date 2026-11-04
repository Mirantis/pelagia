- `name` - Mandatory. Ceph Object Storage instance name.
- `usedForOpenstack` - Optional. Enables consumption of the current Ceph RGW instance by OpenStack,
  which allows adding the required OpenStack parameters to the Ceph RGW configuration.
  For the first such instance, Pelagia also creates the `rgw-ceilometer` user for OpenStack metrics.
  If `gatewayHTTPRoutes` is not specified, Pelagia creates the default `<name>-openstack-route`
  Gateway API `HTTPRoute` for the instance. If `lcmConfig.useIngress` is set to `true`
  in the Helm chart values, Pelagia also creates an Ingress for the instance.
- `auxiliaryService` - Optional. Enables the current Ceph RGW instance to operate without a `StorageClass`
  and external API access. Useful for auxiliary instances, such as a Ceph RGW multisite
  replication daemon or an instance providing Ceph RGW admin API access.
- `servedByIngress` - Optional. Enables the current Ceph RGW instance to be used by the Ingress controller.
  Takes effect only if `lcmConfig.useIngress` is set to `true` in the Helm chart values, which is `false` by default.
  Deprecated in favor of the Gateway API `HTTPRoutes`.
- `spec` - Mandatory. Represents the Rook `CephObjectStore` specification. For details, see [CephObjectStore CRD](https://rook.io/docs/rook/{{ rook_doc_version }}/CRDs/Object-Storage/ceph-object-store-crd/) and [CephObjectStore API specification](https://rook.io/docs/rook/{{ rook_doc_version }}/CRDs/specification/#ceph.rook.io/v1.CephObjectStore).
