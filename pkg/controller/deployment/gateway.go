/*
Copyright 2026 Mirantis IT.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package deployment

import (
	"fmt"
	"reflect"

	"github.com/pkg/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	gatewayapi "sigs.k8s.io/gateway-api/apis/v1"

	cephlcmv1alpha1 "github.com/Mirantis/pelagia/pkg/apis/ceph.pelagia.lcm/v1alpha1"
	lcmcommon "github.com/Mirantis/pelagia/pkg/common"
)

var (
	defaultGatewayKind        = gatewayapi.Kind("Gateway")
	defaultGatewayGroup       = gatewayapi.Group(gatewayapi.GroupName)
	defaultGatewayBackendKind = gatewayapi.Kind("Service")
)

func (c *cephDeploymentConfig) ensureGatewayHTTPRoutes() (bool, error) {
	// TODO: list only pelagia managed resources
	gtws, err := c.api.Gatewayclientset.GatewayV1().HTTPRoutes(c.lcmConfig.RookNamespace).List(c.context, metav1.ListOptions{})
	if err != nil {
		return false, errors.Wrap(err, "failed to list rgw gateway httproutes to ensure rgw httproutes")
	}
	presentRoutes := map[string]gatewayapi.HTTPRoute{}
	for _, gtw := range gtws.Items {
		presentRoutes[gtw.Name] = gtw
	}

	failDetected := false
	gatewayRoutesToManage := append([]cephlcmv1alpha1.CephDeploymentHTTPRoute{}, c.cdConfig.cephDpl.Spec.ObjectStorage.GatewayHTTPRoutes...)
	if len(c.cdConfig.cephDpl.Spec.ObjectStorage.GatewayHTTPRoutes) == 0 {
		// find Rockoon related Rgws, if present for default configuration
		for _, rgw := range c.cdConfig.cephDpl.Spec.ObjectStorage.Rgws {
			if rgw.UsedByRockoon {
				osSecret, err := c.api.Kubeclientset.CoreV1().Secrets(c.lcmConfig.DeployParams.OpenstackCephSharedNamespace).Get(c.context, openstackRgwCredsName, metav1.GetOptions{})
				if err != nil {
					if apierrors.IsNotFound(err) {
						// skip default httproute creation if no secret found
						c.log.Warn().Msgf("skipping create default httproute for rgw '%s', since related Openstack secret '%s/%s' is not found",
							rgw.Name, c.lcmConfig.DeployParams.OpenstackCephSharedNamespace, openstackRgwCredsName)
					} else {
						c.log.Error().Err(err).Msgf("failed to get Openstack '%s/%s' secret to ensure default rgw gateway httproute for object store '%s'",
							c.lcmConfig.DeployParams.OpenstackCephSharedNamespace, openstackRgwCredsName, rgw.Name)
						failDetected = true
					}
					continue
				}
				// default route
				defaultOpenstackHTTPRoute := cephlcmv1alpha1.CephDeploymentHTTPRoute{
					Name:            fmt.Sprintf("%s-openstack-route", rgw.Name),
					ObjectStoreName: rgw.Name,
				}
				spec := fmt.Sprintf(`{"hostnames": [%s.%s]}`, rgw.Name, string(osSecret.Data["public_domain"]))
				_ = cephlcmv1alpha1.SetRawSpec(&defaultOpenstackHTTPRoute.Spec, []byte(spec), nil)
				gatewayRoutesToManage = append(gatewayRoutesToManage, defaultOpenstackHTTPRoute)
			}
		}
		if failDetected {
			return false, errors.New("failed to ensure default rgw gateway httproute")
		}
	}

	changed := false
	for _, httproute := range gatewayRoutesToManage {
		httpRouteResource := c.generateHTTPRoute(httproute)
		if httpRouteCur, ok := presentRoutes[httproute.Name]; ok {
			if !reflect.DeepEqual(httpRouteResource.Spec, httpRouteCur.Spec) {
				delete(presentRoutes, httproute.Name)
				httpRouteCur.Spec = httpRouteResource.Spec
				_, err := c.api.Gatewayclientset.GatewayV1().HTTPRoutes(c.lcmConfig.RookNamespace).Update(c.context, &httpRouteCur, metav1.UpdateOptions{})
				if err != nil {
					c.log.Error().Err(err).Msgf("failed to update rgw gateway httproute '%s/%s'", httpRouteCur.Namespace, httpRouteCur.Name)
					failDetected = true
				} else {
					changed = true
				}
			}
		} else {
			_, err := c.api.Gatewayclientset.GatewayV1().HTTPRoutes(c.lcmConfig.RookNamespace).Create(c.context, &httpRouteResource, metav1.CreateOptions{})
			if err != nil {
				c.log.Error().Err(err).Msgf("failed to create rgw gateway httproute '%s/%s'", httpRouteResource.Namespace, httpRouteResource.Name)
				failDetected = true
			} else {
				changed = true
			}
		}
	}

	for route := range presentRoutes {
		err := c.api.Gatewayclientset.GatewayV1().HTTPRoutes(c.lcmConfig.RookNamespace).Delete(c.context, route, metav1.DeleteOptions{})
		if err != nil {
			c.log.Error().Err(err).Msgf("failed to delete rgw gateway httproute '%s/%s'", c.lcmConfig.RookNamespace, route)
			failDetected = true
		} else {
			changed = true
		}
	}

	if failDetected {
		return false, errors.New("failed to ensure rgw gateway httproute(s)")
	}
	return changed, nil
}

func (c *cephDeploymentConfig) generateHTTPRoute(httpRoute cephlcmv1alpha1.CephDeploymentHTTPRoute) gatewayapi.HTTPRoute {
	newHTTPRoute := gatewayapi.HTTPRoute{
		ObjectMeta: metav1.ObjectMeta{
			Name:      httpRoute.Name,
			Namespace: c.lcmConfig.RookNamespace,
			Labels: map[string]string{
				"app":               "rook-ceph-rgw",
				"rook_object_store": httpRoute.ObjectStoreName,
			},
		},
	}
	var defaultObjectStorePort int32
	for _, rgw := range c.cdConfig.cephDpl.Spec.ObjectStorage.Rgws {
		if rgw.Name == httpRoute.ObjectStoreName {
			rgwSpec, _ := rgw.GetSpec()
			defaultObjectStorePort = rgwSpec.Gateway.Port
			break
		}
	}
	externalAccessLabel, err := metav1.ParseToLabelSelector(c.lcmConfig.CommonParams.RgwPublicAccessLabel)
	if err != nil {
		c.log.Error().Err(err).Msg("")
	}
	for key, val := range externalAccessLabel.MatchLabels {
		newHTTPRoute.Labels[key] = val
	}
	httpRouteSpec, _ := httpRoute.GetSpec()
	if len(httpRouteSpec.ParentRefs) == 0 {
		// set default gateway parent ref
		httpRouteSpec.ParentRefs = []gatewayapi.ParentReference{
			{
				Name:      gatewayapi.ObjectName(c.lcmConfig.CommonParams.BaseGatewayName),
				Namespace: lcmcommon.PtrTo(gatewayapi.Namespace(c.lcmConfig.CommonParams.BaseGatewayNamespace)),
				Group:     &defaultGatewayGroup,
				Kind:      &defaultGatewayKind,
			},
		}
	}
	// fill defaults if not specified
	if len(httpRouteSpec.Rules) == 0 {
		httpRouteSpec.Rules = []gatewayapi.HTTPRouteRule{{Name: lcmcommon.PtrTo(gatewayapi.SectionName("default"))}}
	}
	for idx, rule := range httpRouteSpec.Rules {
		if len(rule.Matches) == 0 {
			// set default matches
			matches := []gatewayapi.HTTPRouteMatch{
				{
					Path: &gatewayapi.HTTPPathMatch{
						Type:  lcmcommon.PtrTo(gatewayapi.PathMatchPathPrefix),
						Value: lcmcommon.PtrTo("/"),
					},
				},
			}
			httpRouteSpec.Rules[idx].Matches = matches
		}
		if len(rule.BackendRefs) == 0 {
			// set default backend
			backendRefs := []gatewayapi.HTTPBackendRef{
				{
					BackendRef: gatewayapi.BackendRef{
						BackendObjectReference: gatewayapi.BackendObjectReference{
							Kind: &defaultGatewayBackendKind,
							Name: gatewayapi.ObjectName(buildRGWName(httpRoute.ObjectStoreName, "")),
							Port: &defaultObjectStorePort,
						},
						Weight: &[]int32{1}[0],
					},
				},
			}
			httpRouteSpec.Rules[idx].BackendRefs = backendRefs
		}
	}
	newHTTPRoute.Spec = httpRouteSpec
	return newHTTPRoute
}
