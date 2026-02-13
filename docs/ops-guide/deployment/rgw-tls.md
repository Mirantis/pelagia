<a id="rgw-tls"></a>

# Configure Ceph Object Gateway TLS

Once you enable Ceph Object Gateway (`radosgw`) as described in [Enable Ceph RGW Object Storage](./rgw.md#enable-rgw-mira), you can configure the Transport Layer Security (TLS) protocol for a Ceph Object Gateway public endpoint using custom `ingressConfig` specified in the `CephDeployment` custom resource (CR). In this case, Ceph Object Gateway public endpoint will use the public domain specified using the `ingressConfig` parameters.

!!! note

    For clusters integrated with Rockoon, Pelagia has an ability to use domain and certificates, defined in Rockoon configuration. Pelagia prioritize `ingressConfig` data over Rockoon ingress data but if `ingressConfig` section is not configured, Pelagia will use Rockoon domain and certificates.
    Mirantis recommends not defining `ingressConfig` section, if Rockoon has `tls-proxy` enabled. In that case, common certificates are applied to all ingresses from the `OpenStackDeployment` object. This implies that Pelagia will use the public domain and the common certificate from the `OpenStackDeployment` object.

This section describes how to specify a custom public endpoint for the Ceph Object Storage.

## Ingress config parameters <a name="ingress"></a>

- `tlsConfig` - Defines TLS configuration for the Ceph Object Gateway public endpoint.
- `controllerClassName` - Name of Ingress Controller class. The default value for Pelagia integrated Rockoon is `openstack-ingress-nginx`
- `annotations` - Extra annotations for the ingress proxy.

### The `tlsConfig` section parameters

- `tlsSecretRefName` - Secret name with TLS certs in Rook Ceph namespace, for example, `rook-ceph`.
  Allows avoiding exposure of certs directly in `spec`. Must contain the following format:

    ```yaml
    data:
      ca.cert: <base64encodedCaCertificate>
      tls.crt: <base64encodedTlsCert>
      tls.key: <base64encodedTlsKey>
    ```

    !!! danger

        When using `tlsSecretRefName`, remove `certs` section.

- `certs` - TLS configuration for ingress including certificates.
  Contains the following parameters:

    !!! danger

        `certs` parameters section is insecure because it stores
        TLS certificates in plain text. Consider using the
        `tlsSecretRefName` parameter instead to avoid exposing
        TLS certificates in the `CephDeployment` CR.

    - `cacert` - The Certificate Authority (CA) certificate, used for the
      ingress rule TLS support.
    - `tlsCert` - The TLS certificate, used for the ingress rule TLS support.
    - `tlsKey` - The TLS private key, used for the ingress rule TLS support.

- `publicDomain` -  Mandatory. The domain name to use for public endpoints.

    !!! danger

        For Pelagia integrated with Rockoon, the default ingress controller does not support `publicDomain` values
        different from the OpenStack ingress public domain. Therefore, if you intend to use the default OpenStack
        Ingress Controller for your Ceph Object Storage public endpoint, plan to use the same
        public domain as your OpenStack endpoints.

- `hostname` - Custom name to override the Ceph Object Storage name for public access. Public RGW endpoint has the
  `https://<hostname>.<publicDomain>` format.

### The `controllerClassName` parameter

`controllClassName` defines the name of the custom Ingress Controller. Pelagia does not support deploying Ingress
Controllers, so you must deploy the Ingress Controller before configuring the `ingressConfig` section in the
`CephDeployment` CR.

For Pelagia integrated with Rockoon, the default Ingress Controller has `openstack-ingress-nginx` class name and Ceph
uses the Rockoon OpenStack Ingress Controller based on NGINX.

### The `annotations` parameter

`annotations` parameter defines extra annotations for the ingress proxy that are a key-value mapping of strings
to add or override ingress rule annotations. For details, see
[NGINX Ingress Controller: Annotations](https://kubernetes.github.io/ingress-nginx/user-guide/nginx-configuration/annotations/).

By default, the following annotations are set:

- `nginx.ingress.kubernetes.io/rewrite-target` is set to `/`.
- `nginx.ingress.kubernetes.io/upstream-vhost` is set to `<spec.objectStorage.rgw.name>.rook-ceph.svc`.

Optional annotations:

- `nginx.ingress.kubernetes.io/proxy-request-buffering: "off"` that disables buffering for `ingress` to prevent the
  *413 (Request Entity Too Large)* error when uploading large files using `radosgw`.
- `nginx.ingress.kubernetes.io/proxy-body-size: <size>` that increases the default uploading size limit to prevent the
  *413 (Request Entity Too Large)* error when uploading large files using `radosgw`. Set the value in MB (`m`) or KB
  (`k`). For example, `100m`.

By default, an ingress rule is created with an internal Ceph Object Gateway service endpoint as a backend.
Also, `rgw dns name` is specified by Pelagia Deployment Controller and is set to
`<spec.objectStorage.rgw.name>.rook-ceph.svc` by default.

You can override `rgw dns name` using the `rookConfig` key-value parameter. In this case, also change the corresponding
ingress annotation.

??? "Configuration example with the `rgw_dns_name` override"

    ```yaml
    spec:
      objectStorage:
        rgw:
          name: rgw-store
          ...
      ingressConfig:
        tlsConfig:
          publicDomain: public.domain.name
          tlsSecretRefName: pelagia-ingress-tls-secret
        controllerClassName: openstack-ingress-nginx
        annotations:
          nginx.ingress.kubernetes.io/rewrite-target: /
          nginx.ingress.kubernetes.io/upstream-vhost: rgw-store.public.domain.name
          nginx.ingress.kubernetes.io/proxy-body-size: 100m
      rookConfig:
        "rgw dns name": rgw-store.public.domain.name
    ```

For clouds with the `publicDomain` parameter specified, align the `upstream-vhost` ingress annotation with the
name of the Ceph Object Storage and the specified public domain.

Pelagia Ceph Object Storage requires the `upstream-vhost` and `rgw dns name` parameters to be equal. Therefore,
override the default `rgw dns name` with the corresponding ingress annotation value.

## To configure Ceph Object Gateway TLS

To generate an SSL certificate for internal usage, verify that the
RADOS Gateway `spec.objectStorage.rgw.gateway.securePort` parameter is specified in the `CephDeployment` CR.
For details, see [Enable Ceph RGW Object Storage](./rgw.md#enable-rgw-mira).

Configure TLS for Ceph Object Gateway using a custom `ingressConfig`:

1. Open the `CephDeployment` CR for editing:
  ```bash
  kubectl -n pelagia edit cephdpl <name>
  ```
  Substitute `<name>` with the name of your `CephDeployment`.
2. Specify the `ingressConfig` parameters as required.
3. Save the changes and close the editor.

!!! note

      For Pelagia with Rockoon, you can omit TLS configuration for the default settings provided by Rockoon to be
      applied. Just obtain the Rockoon OpenStack CA certificate for a trusted connection:
      ```
      kubectl -n openstack-ceph-shared get secret openstack-rgw-creds -o jsonpath="{.data.ca_cert}" | base64 -d
      ```

If you use the HTTP scheme instead of HTTPS for internal or public Ceph Object Gateway endpoints,
add custom annotations to the `ingressConfig.annotations` section of the `CephDeployment` CR:
```yaml
spec:
  ingressConfig:
    annotations:
      "nginx.ingress.kubernetes.io/force-ssl-redirect": "false"
      "nginx.ingress.kubernetes.io/ssl-redirect": "false"
```

If both HTTP and HTTPS must be used, apply the following configuration in the `CephDeployment` object:
```yaml
spec:
  ingressConfig:
    tlsConfig:
      publicDomain: public.domain.name
      tlsSecretRefName: pelagia-ingress-tls-secret
    annotations:
      "nginx.ingress.kubernetes.io/force-ssl-redirect": "false"
      "nginx.ingress.kubernetes.io/ssl-redirect": "false"
```

Access public Ceph Object Gateway endpoint:

1. Obtain the Ceph Object Gateway public endpoint:
    ```bash
    kubectl -n rook-ceph get ingress
    ```
2. Obtain the public endpoint TLS CA certificate:
    ```bash
    kubectl -n rook-ceph get secret $(kubectl -n rook-ceph get ingress -o jsonpath='{.items[0].spec.tls[0].secretName}{"\n"}') -o jsonpath='{.data.ca\.crt}' | base64 -d; echo
    ```

Access internal Ceph Object Gateway endpoint if needed:

1. Obtain the internal endpoint name for Ceph Object Gateway:
    ```bash
    kubectl -n rook-ceph get svc -l app=rook-ceph-rgw
    ```

    The internal endpoint for Ceph Object Gateway has the following format:
    ```
    https://<internal-svc-name>.rook-ceph.svc:<rgw-secure-port>/
    ```
    where `<rgw-secure-port>` is `spec.objectStorage.rgw.gateway.securePort` specified
    in the `CephDeployment` CR.

2. Obtain the internal endpoint TLS CA certificate:
    ```bash
    kubectl -n rook-ceph get secret rgw-ssl-certificate -o jsonpath="{.data.cacert}" | base64 -d
    ```

Verify at least one of the following requirements is met:
    * The public hostname matches the public domain name set by the `spec.ingressConfig.tlsConfig.publicDomain` field;
    * The OpenStack configuration has been applied.

If both options is not true, update the zonegroup `hostnames` of Ceph Object Gateway:

1. Enter the `pelagia-ceph-toolbox` pod:
    ```bash
    kubectl -n rook-ceph exec -it deployment/pelagia-ceph-toolbox -- bash
    ```
2. Obtain Ceph Object Gateway default zonegroup configuration:
    ```bash
    radosgw-admin zonegroup get --rgw-zonegroup=<objectStorageName> --rgw-zone=<objectStorageName> | tee zonegroup.json
    ```

    Substitute `<objectStorageName>` with the Ceph Object Storage name from
    `spec.objectStorage.rgw.name`.

3. Inspect `zonegroup.json` and verify that the `hostnames` key is a
    list that contains two endpoints: an internal endpoint and a custom
    public endpoint:
    ```bash
    "hostnames": ["rook-ceph-rgw-<objectStorageName>.rook-ceph.svc", <customPublicEndpoint>]
    ```

    Substitute `<objectStorageName>` with the Ceph Object Storage name and
    `<customPublicEndpoint>` with the public endpoint with a custom public
    domain.

4. If one or both endpoints are omitted in the list, add the missing
    endpoints to the `hostnames` list in the `zonegroup.json` file and
    update Ceph Object Gateway zonegroup configuration:
    ```bash
    radosgw-admin zonegroup set --rgw-zonegroup=<objectStorageName> --rgw-zone=<objectStorageName> --infile zonegroup.json
    radosgw-admin period update --commit
    ```

5. Verify that the `hostnames` list contains both the internal and custom public endpoint:
    ```bash
    radosgw-admin --rgw-zonegroup=<objectStorageName> --rgw-zone=<objectStorageName> zonegroup get | jq -r ".hostnames"
    ```

      Example of system response:
      ```json
      [
        "rook-ceph-rgw-obj-store.rook-ceph.svc",
        "obj-store.mcc1.cluster1.example.com"
      ]
      ```

6. Exit the `pelagia-ceph-toolbox` pod:
    ```bash
    exit
    ```

Once done, Ceph Object Gateway becomes available by the custom public endpoint
with an S3 API client, OpenStack Swift CLI, and OpenStack Horizon Containers
plugin.
