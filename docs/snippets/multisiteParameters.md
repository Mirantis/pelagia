- `realms` - List of realms to use. Each item represents the Rook `CephObjectRealm` specification.

    - `name` - Mandatory. The realm name.
    - `spec` - Optional. Specification of `CephObjectRealm`. For details, see [CephObjectRealm CRD](https://rook.io/docs/rook/{{ rook_doc_version }}/CRDs/Object-Storage/ceph-object-realm-crd/).

- `zonegroups` - List of zone groups for realms. Each item represents the Rook `CephObjectZoneGroup` specification.

    - `name` - Mandatory. The zone group name.
    - `spec` - Mandatory. Specification of `CephObjectZoneGroup`. The `realm` field must refer to a realm defined in `realms`. For details, see [CephObjectZoneGroup CRD](https://rook.io/docs/rook/{{ rook_doc_version }}/CRDs/Object-Storage/ceph-object-zonegroup-crd/).

- `zones` - List of zones for zone groups. Each item represents the Rook `CephObjectZone` specification.

    - `name` - Mandatory. The zone name.
    - `spec` - Mandatory. Specification of `CephObjectZone`. The `zoneGroup` field must refer to a zone group defined in `zonegroups`. For details, see [CephObjectZone CRD](https://rook.io/docs/rook/{{ rook_doc_version }}/CRDs/Object-Storage/ceph-object-zone-crd/).
