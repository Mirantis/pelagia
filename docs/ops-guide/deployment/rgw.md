<a id="enable-rgw-mira"></a>

# Enable Ceph RGW Object Storage

Pelagia enables you to deploy Ceph RADOS Gateway (RGW) Object Storage
instances and automatically manage its resources such as users and buckets.

Pelagia has an integration for Ceph Object Storage with OpenStack Object Storage (`Swift`) provided by Rockoon.

## Ceph RGW Object Storage parameters <a name="parameters"></a>

@Snippet:cephdpl-parameters:rgwParameters@

## To enable the RGW Object Storage:

1. Open the `CephDeployment` resource for editing:
   ```bash
   kubectl -n pelagia edit cephdpl <name>
   ```
   Substitute `<name>` with the name of your `CephDeployment`.
2. Using [Ceph RGW Object Storage parameters](#parameters), update the `objectStorage.rgw` section
   specification.

     For example:
     ```yaml
     rgw:
       name: rgw-store
       dataPool:
         deviceClass: hdd
         erasureCoded:
           codingChunks: 1
           dataChunks: 2
         failureDomain: host
       metadataPool:
         deviceClass: hdd
         failureDomain: host
         replicated:
           size: 3
       gateway:
         allNodes: false
         instances: 3
         port: 80
         securePort: 8443
       preservePoolsOnDelete: false
     ```
3. Save the changes and exit the editor.
