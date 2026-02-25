<a id="ceph-osd-auth-failed-replaced-ceph-osd-fails-to-start-on-authorization"></a>

# Replaced Ceph OSD fails to start on authorization

In rare cases, when the replaced Ceph OSD has the same ID as the previous Ceph
OSD and starts on a device with the same name as the previous Ceph OSD, Rook
fails to update the keyring value, which is stored on a node in the
corresponding host path. Thereby, Ceph OSD cannot start and fails with the
following exemplary log output:

```bash
Defaulted container "osd" out of: osd, activate (init), expand-bluefs (init), chown-container-data-dir (init)
debug 2024-03-13T11:53:13.268+0000 7f8f790b4640 -1 monclient(hunting): handle_auth_bad_method server allowed_methods [2] but i only support [2]
debug 2024-03-13T11:53:13.268+0000 7f8f7a0b6640 -1 monclient(hunting): handle_auth_bad_method server allowed_methods [2] but i only support [2]
debug 2024-03-13T11:53:13.268+0000 7f8f798b5640 -1 monclient(hunting): handle_auth_bad_method server allowed_methods [2] but i only support [2]
failed to fetch mon config (--no-mon-config to skip)
```

**To verify that the cluster is affected**, compare the keyring values stored
in the Ceph cluster and on a node in the corresponding host path:

1. Obtain the keyring of a Ceph OSD stored in the Ceph cluster:
   ```bash
   kubectl -n rook-ceph exec -it \
       deploy/pelagia-ceph-toolbox \
       -- ceph auth get osd.<ID>
   ```

     Substitute `<ID>` with the number of the required Ceph OSD.

     Example output:
     ```bash
     [osd.3]
     key = AQAcovBlqP4qHBAALK6943yZyazoup7nE1YpeQ==
     caps mgr = "allow profile osd"
     caps mon = "allow profile osd"
     caps osd = "allow *"
     ```

2. Obtain the keyring value of the host path for the failed Ceph OSD:

     1. SSH on a node hosting the failed Ceph OSD.
     2. In `/var/lib/rook/rook-ceph`, search for a directory containing the
        `keyring` and `whoami` files that have the number of failed Ceph OSD.
        For example:
        ```bash
        # cat whoami
        3
        # cat keyring
        [osd.3]
        key = AQD2k/BlcE+YJxAA/QsD/fIAL1qPrh3hjQ7AKQ==
        ```

The cluster is affected if the keyring of the failed Ceph OSD on the host path
differs from the one on the Ceph cluster. If so, proceed to fix them and
unblock the failed Ceph OSD.

**To fix the keyring difference and unblock the Ceph OSD authorization:**

1. Obtain the keyring value of the host path for this Ceph OSD:

     1. SSH on a node hosting the required Ceph OSD.
     2. In `/var/lib/rook/rook-ceph`, search for a directory containing
        the `keyring` and `whoami` files that have the number of the
        required Ceph OSD. For example:
        ```bash
        # cat whoami
        3
        # cat keyring
        [osd.3]
        key = AQD2k/BlcE+YJxAA/QsD/fIAL1qPrh3hjQ7AKQ==
        ```

2. Enter the `pelagia-ceph-toolbox` pod:
   ```bash
   kubectl -n rook-ceph exec -it deploy/pelagia-ceph-toolbox -- bash
   ```

3. Export the current Ceph OSD keyring stored in the Ceph cluster:
   ```bash
   ceph auth get osd.<ID> -o /tmp/key
   ```

4. Replace the exported key with the value from `keyring`. For example:
   ```bash
   vi /tmp/key
   # replace the key with the one from the keyring file
   [osd.3]
   key = AQD2k/BlcE+YJxAA/QsD/fIAL1qPrh3hjQ7AKQ==
   caps mgr = "allow profile osd"
   caps mon = "allow profile osd"
   caps osd = "allow *"
   ```

5. Import the replaced Ceph OSD keyring to the Ceph cluster:
   ```bash
   ceph auth import -i /tmp/key
   ```

6. Restart the failed Ceph OSD pod:
   ```bash
   kubectl -n rook-ceph scale deploy rook-ceph-osd-<ID> --replicas 0
   kubectl -n rook-ceph scale deploy rook-ceph-osd-<ID> --replicas 1
   ```
