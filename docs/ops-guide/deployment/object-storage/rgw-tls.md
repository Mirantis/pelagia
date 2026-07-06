---
description: How to configure TLS for Ceph Object Gateway (RGW) endpoints using built‑in or custom ingress settings.
keywords: pelagia, configure rgw tls, ceph rgw tls, rgw tls, configure tls, rados gateway tls, rgw certificates,
  cephdeployment, configure ceph object gateway tls, ingressconfig, ingress
---

<a id="rgw-tls-configure-ceph-object-gateway-tls"></a>

# Configure Ceph Object Gateway TLS

Once you enable Ceph Object Gateway (`radosgw`) as described in [Enable Ceph RGW Object Storage](./rgw.md#rgw-enable-ceph-rgw-object-storage), you can configure the Transport Layer Security (TLS) protocol for a Ceph Object Gateway public endpoint using custom `ingressConfig` specified in the `CephDeployment` custom resource (CR). In this case, Ceph Object Gateway public endpoint will use the public domain specified using the `ingressConfig` parameters.

!!! note

    For clusters integrated with Rockoon, Pelagia can use domain and certificates defined in the Rockoon configuration but always prioritizes configuration provided in the `CephDeployment` spec (such as `objectStorage.gatewayHTTPRoutes` or deprecated `ingressConfig`).
    Mirantis recommends not defining the `ingressConfig` or `objectStorage.gatewayHTTPRoutes` section if Rockoon has `tls-proxy` enabled. In that case, common certificates are applied to all ingresses/gateways from the `OpenStackDeployment` object. This implies that Pelagia will use the public domain and the common certificate from the `OpenStackDeployment` object.

This section describes how to specify a custom public endpoint for the Ceph Object Storage.

## Gateway HTTPRoute parameters

With Pelagia supporting the Gateway API, you can use `CephDeployment` to configure the `HTTPRoute` specification for Ceph Object Storage.
For configuration details, see [CephDeployment API: HTTPRoute parameters](../../../custom-resources/cephdeployment.md#cephdeployment-httproute-parameters).

!!! info "See also"

    Gateway API documentation:

      - [API overview](https://gateway-api.sigs.k8s.io/docs/concepts/api-overview/)
      - [HTTP routing](https://gateway-api.sigs.k8s.io/guides/user-guides/http-routing/)
      - [HTTPRoute API](https://gateway-api.sigs.k8s.io/reference/api-types/httproute/)

<a name="rgw-tls-ingress-config-parameters"></a>
## Ingress configuration parameters

!!! warning

    The `tlsConfig` section is deprecated. Use the `objectStorage.gatewayHTTPRoutes` section instead.

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

`controllerClassName` defines the name of the custom Ingress Controller. Pelagia does not support deploying Ingress
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

For the ability to access RGW with a public hostname, you must set the expected DNS names in the Ceph ObjectStore configuration.

??? "Example configuration with DNS names"

    ```yaml
    spec:
      objectStorage:
        objectStores:
        - name: rgw-store
          usedByIngress: true
          spec:
            ...
            hosting:
              dnsNames:
              - rgw-store.public.domain.name
      ingressConfig:
        tlsConfig:
          publicDomain: public.domain.name
          tlsSecretRefName: pelagia-ingress-tls-secret
        controllerClassName: openstack-ingress-nginx
        annotations:
          nginx.ingress.kubernetes.io/rewrite-target: /
          nginx.ingress.kubernetes.io/upstream-vhost: rgw-store.public.domain.name
          nginx.ingress.kubernetes.io/proxy-body-size: 100m
    ```

<a id="rgw-tls-configure-tls-for-object-gateway-using-the-gateway-api"></a>
## Configure TLS for Object Gateway using the Gateway API

Pelagia uses the `gatewayHTTPRoutes` section to manage `HTTPRoute` of the Gateway/TLS settings that were previously configured by the operator.
With the provided `HTTPRoute`, Ceph Object Gateway will use the SSL certificate provided by the Gateway Controller.

**To configure TLS using the Gateway API:**

1. Configure the Gateway Controller and the `Gateway` object with TLS using the [Gateway API documentation: TLS Configuration](https://gateway-api.sigs.k8s.io/guides/user-guides/tls/).
2. Open the `CephDeployment` CR for editing.
3. Configure the `objectStorage` section:

    - In `gatewayHTTPRoutes`, create an `HTTPRoute` with the required parameters. For reference, see [Gateway API documentation: HTTPRoute](https://gateway-api.sigs.k8s.io/reference/api-types/httproute/).
    - In `gatewayHTTPRoutes.name.spec.hostnames` and `objectStores.name.spec.hosting.dnsNames`, add a hostname to be used for the RGW public access, respecting the Gateway TLS configuration.

    For example:

    ```yaml
    objectStorage:
      gatewayHTTPRoutes:
      - name: route-1
        objectStoreName: rgw-store
        spec:
          hostnames:
          - rgw-store.custom.dns.name
      objectStores:
      - name: rgw-store
        spec:
          ...
          hosting:
            dnsNames:
            - rgw-store.custom.dns.name
          ...
    ```

4. Verify that `HTTPRoute` is configured:

    ```bash
    kubectl get httproute -n rook-ceph
    ```

     Example of a successful system response:
     ```bash
     NAME                              HOSTNAMES                           AGE
     openstack-store-openstack-route   ["openstack-store.it.just.works"]   125m
     ```

## Configure TLS for Object Gateway using Ingress

!!! warning

    TLS configuration for Ceph Object Gateway using Ingress is deprecated. Use the Gateway API instead. For details, see [Configure TLS for Ceph Object Gateway using the Gateway API](#rgw-tls-configure-tls-for-object-gateway-using-the-gateway-api).

1. To generate an SSL certificate for internal usage, verify that the
RADOS Gateway `spec.objectStorage.rgw.gateway.securePort` parameter is specified in the `CephDeployment` CR.
For details, see [Enable Ceph RGW Object Storage](./rgw.md#rgw-enable-ceph-rgw-object-storage).
2. Open the `CephDeployment` CR for editing:
     ```bash
     kubectl -n pelagia edit cephdpl <name>
     ```

     Substitute `<name>` with the name of your `CephDeployment`.

     For Pelagia with Rockoon, you can omit TLS configuration for the default settings provided by Rockoon
     to be applied. Just obtain the Rockoon OpenStack CA certificate for a trusted connection:
     ```bash
     kubectl -n openstack-ceph-shared get secret openstack-rgw-creds -o jsonpath="{.data.ca_cert}" | base64 -d
     ```

3. Specify the `ingressConfig` parameters as required.
4. Save the changes and close the editor.

5. If you use the HTTP scheme instead of HTTPS for internal or public Ceph Object Gateway endpoints,
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

6. In the `objectStorage.objectStores.name.spec.hosting.dnsNames` section, set the public domain name provided in `spec.ingressConfig.tlsConfig.publicDomain`:

    ```yaml
    objectStorage:
      objectStores:
      - name: rgw-store
        spec:
          ...
          hosting:
            dnsNames:
            - rgw-store.public.domain.name
          ...
    ```

7. Access the public Ceph Object Gateway endpoint:

     1. Obtain the Ceph Object Gateway public endpoint:
          ```bash
          kubectl -n rook-ceph get ingress
          ```
     2. Obtain the public endpoint TLS CA certificate:
          ```bash
          kubectl -n rook-ceph get secret $(kubectl -n rook-ceph get ingress -o jsonpath='{.items[0].spec.tls[0].secretName}{"\n"}') -o jsonpath='{.data.ca\.crt}' | base64 -d; echo
          ```

     To access the internal Ceph Object Gateway endpoint, if needed:

     1. Obtain the internal endpoint name for Ceph Object Gateway:
          ```bash
          kubectl -n rook-ceph get svc -l app=rook-ceph-rgw
          ```

        The internal endpoint for Ceph Object Gateway has the following format:
          ```bash
          https://<internal-svc-name>.rook-ceph.svc:<rgw-secure-port>/
          ```

        where `<rgw-secure-port>` is `spec.objectStorage.rgw.gateway.securePort` specified
        in the `CephDeployment` CR.

     2. Obtain the internal endpoint TLS CA certificate:
          ```bash
          kubectl -n rook-ceph get secret rgw-ssl-certificate -o jsonpath="{.data.cacert}" | base64 -d
          ```

## Verify TLS for Ceph Object Gateway

1. Verify that at least one of the following requirements is met:

     - The public hostname matches the public domain name set by the `spec.ingressConfig.tlsConfig.publicDomain` field
     - The public hostname matches the public domain name set by the `spec.objectStorage.gatewayHTTPRoutes` field
     - The OpenStack configuration has been applied

2. Find the RGW deployment in the `rook-ceph` namespace:
    ```bash
    kubectl -n rook-ceph get deploy -l app=rook-ceph-rgw
    ```

3. Verify that the deployment contains the required DNS names with the `--rgw-dns-name` argument:
    ```bash
    kubectl -n rook-ceph get deploy <rgw-deployment-name> -o yaml | grep <rgw-dns-name>
    ```

     Substitute `<rgw-deployment-name>` with the RGW deployment name from the previous step.
     Substitute `<rgw-dns-name>` with the expected DNS name provided in
     `<rgw-object-store-name>.<value from spec>.ingressConfig.tlsConfig.publicDomain>` or `spec.objectStorage.gatewayHTTPRoutes`.

     Example of a successful system response:
     ```bash
     - --rgw-dns-name: rgw-store.public.domain.name
     ```

Once done, Ceph Object Gateway becomes available by the custom public endpoint
with an S3 API client, OpenStack Swift CLI, and OpenStack Horizon Containers
plugin.
