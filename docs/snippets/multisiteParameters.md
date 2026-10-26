- `realms` - List of realms to use. Each item represents the Rook `CephObjectRealm` specification.

    - `name` - Mandatory. The realm name.
    - `spec` - Specification of `CephObjectRealm`. For details, see [CephObjectRealm CRD]({{ rook_docs }}/CRDs/Object-Storage/ceph-object-realm-crd/).

- `zonegroups` - List of zone groups for realms. Each item represents the Rook `CephObjectZoneGroup` specification.

    - `name` - Mandatory. The zone group name.
    - `spec` - Specification of `CephObjectZoneGroup`. For details, see [CephObjectZoneGroup CRD]({{ rook_docs }}/CRDs/Object-Storage/ceph-object-zonegroup-crd/).

- `zones` - List of zones used within one zone group. Each item represents the Rook `CephObjectZone` specification.

    - `name` - Mandatory. The zone name.
    - `spec` - Specification of `CephObjectZone`. For details, see [CephObjectZone CRD]({{ rook_docs }}/CRDs/Object-Storage/ceph-object-zone-crd/).
