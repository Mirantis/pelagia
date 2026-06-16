- `realms` - List of realms to use. Each item represents the Rook `CephObjectRealm` specification.

    - `name` - Mandatory. The realm name.
    - `spec` - Specification of `CephObjectRealm`. For details, see [CephObjectRealm CRD](https://rook.io/docs/rook/v1.19/CRDs/Object-Storage/ceph-object-realm-crd/).

- `zonegroups` - List of zone groups for realms. Each item represents the Rook `CephObjectZoneGroup` specification.

    - `name` - Mandatory. The zone group name.
    - `spec` - Specification of `CephObjectZoneGroup`. For details, see [CephObjectZoneGroup CRD](https://rook.io/docs/rook/v1.19/CRDs/Object-Storage/ceph-object-zonegroup-crd/).

- `zones` - List of zones used within one zone group. Each item represents the Rook `CephObjectZone` specification.

    - `name` - Mandatory. The zone name.
    - `spec` - Specification of `CephObjectZone`. For details, see [CephObjectZone CRD](https://rook.io/docs/rook/v1.19/CRDs/Object-Storage/ceph-object-zone-crd/).
