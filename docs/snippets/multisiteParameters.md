- `realms` - List of realms to use, represents the realm namespaces. Each item represents Rook `CephObjectRealm` specification.

    - `name` - required, the realm name.
    - `spec` - specification of `CephObjectRealm`. Refer to [CephObjectRealm CRD](https://rook.io/docs/rook/v1.19/CRDs/Object-Storage/ceph-object-realm-crd/) for details.

- `zonegroups` - The list of zone groups for realms. Each item represents Rook `CephObjectZoneGroup` specification.

    - `name` - required, the zone group name.
    - `spec` - specification of `CephObjectZoneGroup`. Refer to [CephObjectZoneGroup CRD](https://rook.io/docs/rook/v1.19/CRDs/Object-Storage/ceph-object-zonegroup-crd/) for details.

- `zones` - The list of zones used within one zone group. Each item represents Rook `CephObjectZone` specification.

    - `name` - required, the zone name.
    - `spec` - specification of `CephObjectZone`. Refer to [CephObjectZone CRD](https://rook.io/docs/rook/v1.19/CRDs/Object-Storage/ceph-object-zone-crd/) for details.
