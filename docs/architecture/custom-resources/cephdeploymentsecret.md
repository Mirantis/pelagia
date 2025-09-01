# CephDeploymentSecret Custom Resource

`CephDeploymentSecret` (`cephdeploymentsecrets.lcm.mirantis.com`) custom resource (CR)
contains the information about all Ceph RBD/RGW credential secrets to be used for the access to Ceph cluster.
To obtain the resource, run the following command:

```bash
kubectl -n pelagia get cephdeploymentsecret -o yaml
```

Example output:

<details>
<summary>Example CephDeploymentSecret output</summary>
<div>
```yaml
apiVersion: v1
items:
- apiVersion: lcm.mirantis.com/v1alpha1
  kind: CephDeploymentSecret
  metadata:
    name: pelagia-ceph
    namespace: pelagia
  status:
    lastSecretCheck: "2025-08-15T12:22:11Z"
    lastSecretUpdate: "2025-08-15T12:22:11Z"
    secretInfo:
      clientSecrets:
      - name: client.admin
        secretName: rook-ceph-admin-keyring
        secretNamespace: rook-ceph
      rgwUserSecrets:
      - name: test-user
        secretName: rook-ceph-object-user-rgw-store-test-user
        secretNamespace: rook-ceph
    state: Ok
kind: List
metadata:
  resourceVersion: ""
```
</div>
</details>

To understand the status of a `CephDeploymentHealth`, learn the following:

- [High-level status fields](#general)
- [Secret info fields](#secret-info)


## High-level status fields <a name="general"></a>

The `CephDeploymentSecret` custom resource contains the following high-level status fields:


| <div style="width:150px">Field</div> | Description                                                                                                                                                                                              |
|--------------------------------------|----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `state`                              | Current state of the secret collector on the Ceph cluster:<br/><br/>  - `Ready` - information about secrets is collected successfully<br/>  - `Failed` - information about secrets fails to be collected |
| `lastSecretCheck`                    | `DateTime` when the Ceph cluster secrets were verified last time.                                                                                                                                        |
| `lastSecretUpdate`                   | `DateTime` when the Ceph cluster secrets were updated last time.                                                                                                                                         |
| `secretsInfo`                        | List of secrets for Ceph `authx` clients and RADOS Gateway users. For details, see [Secret info fields](#secret-info).                                                                                   |
| `messages`                           | List of error or warning messages, if any, found when collecting information about the Ceph cluster.                                                                                                     |

## Secret info fields <a name="secret-info"></a>

The `secretsInfo` field contains the following fields:

- `clientSecrets` - Details on secrets for Ceph clients such as `name`, `secretName`, and `secretNamespace`
  for each client secret.
- `rgwUserSecrets` - Details on secrets for Ceph RADOS Gateway users such as `name`, `secretName`, and
  `secretNamespace`.

Example of the `secretsInfo` field:

<details>
<summary>Example *secretsInfo* field</summary>
<div>
```yaml
status:
  secretInfo:
    clientSecrets:
    - name: client.admin
      secretName: rook-ceph-admin-keyring
      secretNamespace: rook-ceph
    rgwUserSecrets:
    - name: test-user
      secretName: rook-ceph-object-user-rgw-store-test-user
      secretNamespace: rook-ceph
```
</div>
</details>
