/*
Copyright 2025 Mirantis IT.

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

package health

import (
	"fmt"
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	gatewayapi "sigs.k8s.io/gateway-api/apis/v1"

	lcmv1alpha1 "github.com/Mirantis/pelagia/pkg/apis/ceph.pelagia.lcm/v1alpha1"
	lcmcommon "github.com/Mirantis/pelagia/pkg/common"
)

func (c *cephDeploymentHealthConfig) getClusterDetailsInfo() (*lcmv1alpha1.ClusterDetails, []string) {
	newDetails := &lcmv1alpha1.ClusterDetails{}
	issues := []string{}

	usageDetails, usageDetailsIssue := c.getCephCapacityDetails()
	newDetails.UsageDetails = usageDetails
	if usageDetailsIssue != "" {
		issues = append(issues, usageDetailsIssue)
	}

	eventsStatus, eventsStatusIssue := c.getCephEvents()
	newDetails.CephEvents = eventsStatus
	if eventsStatusIssue != "" {
		issues = append(issues, eventsStatusIssue)
	}

	replicasIssues := c.checkReplicasSizing()
	if len(replicasIssues) > 0 {
		issues = append(issues, replicasIssues...)
	}

	rgwInfo, rgwIssues := c.getRgwInfo()
	newDetails.RgwInfo = rgwInfo
	if len(rgwIssues) > 0 {
		issues = append(issues, rgwIssues...)
	}

	// to avoid api diff since section is optional and omit empty set
	if usageDetails == nil && eventsStatus == nil && rgwInfo == nil {
		newDetails = nil
	}

	sort.Strings(issues)
	return newDetails, issues
}

func (c *cephDeploymentHealthConfig) getCephCapacityDetails() (*lcmv1alpha1.UsageDetails, string) {
	if lcmcommon.Contains(c.lcmConfig.HealthParams.ChecksSkip, usageDetailsCheck) {
		c.log.Debug().Msgf("skipping ceph cluster usage/capacity check, set '%s' to skip through lcm config settings", usageDetailsCheck)
		return nil, ""
	}
	var cephDetails lcmcommon.CephDetails
	cmd := "ceph df -f json"
	err := lcmcommon.RunAndParseCephToolboxCLI(c.context, c.api.Kubeclientset, c.api.Config, c.lcmConfig.RookNamespace, cmd, &cephDetails)
	if err != nil {
		c.log.Error().Err(err).Msg("")
		return nil, fmt.Sprintf("failed to run '%s' command to check capacity details", cmd)
	}
	usageDetails := lcmv1alpha1.UsageDetails{
		ClassesDetail: map[string]lcmv1alpha1.ClassUsageStats{},
		PoolsDetail:   map[string]lcmv1alpha1.PoolUsageStats{},
	}
	for _, pool := range cephDetails.Pools {
		if c.lcmConfig.HealthParams.UsageDetailsPoolsFilter != "" {
			if match, _ := regexp.MatchString(c.lcmConfig.HealthParams.UsageDetailsPoolsFilter, pool.Name); !match {
				continue
			}
		}
		usageDetails.PoolsDetail[pool.Name] = lcmv1alpha1.PoolUsageStats{
			UsedBytes:           strconv.FormatUint(pool.Stats.UsedBytes, 10),
			UsedBytesPercentage: fmt.Sprintf("%.3f", pool.Stats.PercentUsed*100),
			TotalBytes:          strconv.FormatUint(pool.Stats.TotalBytes, 10),
			AvailableBytes:      strconv.FormatUint(pool.Stats.TotalBytes-pool.Stats.UsedBytes, 10),
		}
	}
	for className, classStats := range cephDetails.StatsByClass {
		if c.lcmConfig.HealthParams.UsageDetailsClassesFilter != "" {
			if match, _ := regexp.MatchString(c.lcmConfig.HealthParams.UsageDetailsClassesFilter, className); !match {
				continue
			}
		}
		usageDetails.ClassesDetail[className] = lcmv1alpha1.ClassUsageStats{
			UsedBytes:      strconv.FormatUint(classStats.UsedBytes, 10),
			AvailableBytes: strconv.FormatUint(classStats.AvailableBytes, 10),
			TotalBytes:     strconv.FormatUint(classStats.TotalBytes, 10),
		}
	}
	return &usageDetails, ""
}

func (c *cephDeploymentHealthConfig) getCephEvents() (*lcmv1alpha1.CephEvents, string) {
	if lcmcommon.Contains(c.lcmConfig.HealthParams.ChecksSkip, cephEventsCheck) {
		c.log.Debug().Msgf("skipping ceph cluster events check, set '%s' to skip through lcm config settings", cephEventsCheck)
		return nil, ""
	}
	var cephStatus lcmcommon.CephStatus
	cmd := "ceph status -f json"
	err := lcmcommon.RunAndParseCephToolboxCLI(c.context, c.api.Kubeclientset, c.api.Config, c.lcmConfig.RookNamespace, cmd, &cephStatus)
	if err != nil {
		c.log.Error().Err(err).Msg("")
		return nil, fmt.Sprintf("failed to run '%s' command to check events details", cmd)
	}
	return &lcmv1alpha1.CephEvents{
		RebalanceDetails:    getEventDetails("Rebalancing", cephStatus.ProgressEvents),
		PgAutoscalerDetails: getEventDetails("PG autoscaler", cephStatus.ProgressEvents),
	}, ""
}

func getEventDetails(eventPrefix string, cephStatusEvents map[string]lcmcommon.ProgressEvents) lcmv1alpha1.CephEventDetails {
	eventDetails := lcmv1alpha1.CephEventDetails{}
	inAction := false

	progressing := make([][]float64, 4)
	progressMapping := map[int]string{
		0: "just started",
		1: "less than a half done",
		2: "more than a half done",
		3: "almost done",
	}

	for _, event := range cephStatusEvents {
		// we are searching events because this is the only way
		// to learn about event process
		if strings.HasPrefix(event.Message, eventPrefix) {
			inAction = true
			// rebalance message always split with "\n" separator - first line is a direct message
			// and the second line is a progress bar which we don't want to expose
			eventMsg := strings.Split(event.Message, "\n")[0]
			eventProgress := fmt.Sprintf("%v", math.Abs(event.Progress))
			eventDetails.Messages = append(eventDetails.Messages,
				lcmv1alpha1.CephEventMessage{Message: eventMsg, Progress: eventProgress})
			// collecting all progress from each rebalance event to calculate
			// current phase
			if event.Progress < 0.25 {
				progressing[0] = append(progressing[0], event.Progress)
			} else if event.Progress < 0.5 {
				progressing[1] = append(progressing[1], event.Progress)
			} else if event.Progress < 0.75 {
				progressing[2] = append(progressing[2], event.Progress)
			} else {
				progressing[3] = append(progressing[3], event.Progress)
			}
		}
	}
	// if there was no rebalance event in ceph status atm - just
	// print that rebalance is in Idle state
	if !inAction {
		eventDetails.State = lcmv1alpha1.CephEventIdle
		return eventDetails
	}

	// otherwise make it progressing
	eventDetails.State = lcmv1alpha1.CephEventProgressing
	maxProgress := 0
	phaseResult := 0
	for phase, progressArray := range progressing {
		if len(progressArray) > maxProgress {
			maxProgress = len(progressArray)
			phaseResult = phase
		}
	}
	eventDetails.Progress = progressMapping[phaseResult]
	// sort messages to avoid redundant updates
	sort.Slice(eventDetails.Messages, func(i, j int) bool {
		return eventDetails.Messages[i].Message < eventDetails.Messages[j].Message
	})
	return eventDetails
}

func (c *cephDeploymentHealthConfig) checkReplicasSizing() []string {
	if lcmcommon.Contains(c.lcmConfig.HealthParams.ChecksSkip, poolReplicasCheck) {
		c.log.Debug().Msgf("skipping ceph cluster pool's replicas sizing check, set '%s' to skip through lcm config settings", poolReplicasCheck)
		return nil
	}
	var osdTree lcmcommon.OsdTree
	cmd := "ceph osd tree -f json"
	err := lcmcommon.RunAndParseCephToolboxCLI(c.context, c.api.Kubeclientset, c.api.Config, c.lcmConfig.RookNamespace, cmd, &osdTree)
	if err != nil {
		c.log.Error().Err(err).Msg("")
		return []string{fmt.Sprintf("failed to run '%s' command to check replicas sizing", cmd)}
	}

	poolsDetail := []struct {
		Name        string `json:"pool_name"`
		Size        int    `json:"size"`
		CrushRuleID int    `json:"crush_rule"`
	}{}
	cmd = "ceph osd pool ls detail -f json"
	err = lcmcommon.RunAndParseCephToolboxCLI(c.context, c.api.Kubeclientset, c.api.Config, c.lcmConfig.RookNamespace, cmd, &poolsDetail)
	if err != nil {
		c.log.Error().Err(err).Msg("")
		return []string{fmt.Sprintf("failed to run '%s' command to check replicas sizing", cmd)}
	}

	crushRuleDump := []struct {
		Name  string                   `json:"rule_name"`
		ID    int                      `json:"rule_id"`
		Steps []map[string]interface{} `json:"steps"`
	}{}
	cmd = "ceph osd crush rule dump -f json"
	err = lcmcommon.RunAndParseCephToolboxCLI(c.context, c.api.Kubeclientset, c.api.Config, c.lcmConfig.RookNamespace, cmd, &crushRuleDump)
	if err != nil {
		c.log.Error().Err(err).Msg("")
		return []string{fmt.Sprintf("failed to run '%s' command to check replicas sizing", cmd)}
	}

	deviceClassToFailureDomainMapping := map[string]map[string]int{}
	for _, node := range osdTree.Nodes {
		if node.Type == "root" {
			// multiple roots and no roots  setup is not supported
			// because some ceph features will not work corretly
			_ = c.countDomainsAndClasses(&osdTree, deviceClassToFailureDomainMapping, node.ID)
			break
		}
	}

	if len(deviceClassToFailureDomainMapping) == 0 {
		return []string{"no device classes found in cluster"}
	}

	issues := []string{}
	for _, pool := range poolsDetail {
		poolFailureDomain := ""
		poolDeviceClass := ""
		for _, crushRule := range crushRuleDump {
			if pool.CrushRuleID == crushRule.ID {
				for _, item := range crushRule.Steps {
					if classValue, present := item["item_name"]; present {
						args := strings.Split(classValue.(string), "~")
						// check is crush rule has specified device class directly
						// in format like `default~hdd` or has no class like `default`
						// also since root is single - no need to check root name
						if len(args) > 1 {
							poolDeviceClass = args[1]
						} else {
							c.log.Warn().Msgf("pool '%s' has crush rule '%s' without specified device class, skipping check", pool.Name, crushRule.Name)
							break
						}
					} else if domainValue, present := item["type"]; present {
						poolFailureDomain = domainValue.(string)
					}
				}
				break
			}
		}
		// skip check if device class is not set
		if poolFailureDomain == "" || poolDeviceClass == "" {
			continue
		}
		if domainsInfo, ok := deviceClassToFailureDomainMapping[poolDeviceClass]; ok {
			if count, ok := domainsInfo[poolFailureDomain]; ok {
				if count < pool.Size {
					msg := fmt.Sprintf("pool '%s' with deviceClass '%s' and failureDomain '%s' has targeted to have %d replicas/chunks, while cluster can provide %d replica(s)",
						pool.Name, poolDeviceClass, poolFailureDomain, pool.Size, count)
					c.log.Error().Msg(msg)
					issues = append(issues, msg)
				}
			} else {
				msg := fmt.Sprintf("pool '%s' specified to use failure domain '%s', which is not present in cluster", pool.Name, poolFailureDomain)
				c.log.Error().Msg(msg)
				issues = append(issues, msg)
			}
		} else {
			// generally ceph will not allow to create a rule with device class
			// which is not present in cluster, so kind of code issue handling
			msg := fmt.Sprintf("pool '%s' specified to use deviceClass '%s', which is not found in cluster", pool.Name, poolDeviceClass)
			c.log.Error().Msg(msg)
			issues = append(issues, msg)
		}
	}
	sort.Strings(issues)
	return issues
}

func (c *cephDeploymentHealthConfig) countDomainsAndClasses(osdTree *lcmcommon.OsdTree, currentMapping map[string]map[string]int, lookupID int) []string {
	for _, node := range osdTree.Nodes {
		if node.ID == lookupID {
			if node.Type == "osd" {
				// check that osd up, in and has positive weight
				if node.Status == "up" && node.Reweight != 0 && node.Weight != 0 {
					if node.DeviceClass == "" {
						c.log.Warn().Msgf("found osd '%s' without device class", node.Name)
						return nil
					}
					if _, classExists := currentMapping[node.DeviceClass]; classExists {
						currentMapping[node.DeviceClass]["osd"]++
					} else {
						currentMapping[node.DeviceClass] = map[string]int{"osd": 1}
					}
					return []string{node.DeviceClass}
				}
				return nil
			}
			deviceClasses := []string{}
			if len(node.Children) > 0 {
				devicesFromChildren := map[string]bool{}
				for _, id := range node.Children {
					for _, class := range c.countDomainsAndClasses(osdTree, currentMapping, id) {
						devicesFromChildren[class] = true
					}
				}
				for devClass := range devicesFromChildren {
					if _, classExists := currentMapping[devClass]; classExists {
						currentMapping[devClass][node.Type]++
					} else {
						currentMapping[devClass] = map[string]int{node.Type: 1}
					}
					deviceClasses = append(deviceClasses, devClass)
				}
			}
			return deviceClasses
		}
	}
	return nil
}

func (c *cephDeploymentHealthConfig) getRgwInfo() (*lcmv1alpha1.RgwInfo, []string) {
	if lcmcommon.Contains(c.lcmConfig.HealthParams.ChecksSkip, rgwInfoCheck) {
		c.log.Debug().Msgf("skipping ceph cluster rgw info check, set '%s' to skip through lcm config settings", rgwInfoCheck)
		return nil, nil
	}
	// no objectstores - no checks
	if len(c.healthConfig.rgwOpts) == 0 {
		return nil, nil
	}

	issues := []string{}
	newRgwInfo := &lcmv1alpha1.RgwInfo{
		PublicEndpoints: map[string][]string{},
	}
	if c.healthConfig.multisiteOpts.zone != "" {
		multisiteDetails, multisiteIssues := c.getMultisiteSyncStatus()
		newRgwInfo.MultisiteDetails = multisiteDetails
		if len(multisiteIssues) > 0 {
			issues = append(issues, multisiteIssues...)
		}
	}
	// check all found rgws
	for rgwName, opts := range c.healthConfig.rgwOpts {
		if opts.external {
			if c.healthConfig.rgwOpts[rgwName].externalEndpoint != "" {
				newRgwInfo.PublicEndpoints[rgwName] = []string{c.healthConfig.rgwOpts[rgwName].externalEndpoint}
			}
		} else {
			rgwEndpoints, endpointIssues := c.getRgwPublicEndpoint(rgwName)
			if len(rgwEndpoints) > 0 {
				sort.Strings(rgwEndpoints)
				newRgwInfo.PublicEndpoints[rgwName] = rgwEndpoints
			}
			if len(endpointIssues) > 0 {
				issues = append(issues, endpointIssues...)
			}
		}
	}
	// show global problem, if objectstores found, but public endpoint not
	// in case if there is only ops admin api - check can be disabled at all
	// otherwise - should be present any public endpoint
	if len(newRgwInfo.PublicEndpoints) == 0 {
		issues = append(issues, "no any public endpoints found for accessing Ceph RGW instance(s)")
	}
	return newRgwInfo, issues
}

func (c *cephDeploymentHealthConfig) getRgwPublicEndpoint(rgwName string) ([]string, []string) {
	if c.lcmConfig.CommonParams.RgwPublicAccessLabel == "" {
		c.log.Warn().Msg("can't detect Ceph RGW public endpoint, since 'RGW_PUBLIC_ACCESS_SERVICE_SELECTOR' is not specified in lcmconfig")
		return nil, nil
	}
	listOptions := metav1.ListOptions{
		LabelSelector: fmt.Sprintf("app=rook-ceph-rgw,rook_object_store=%s,%s", rgwName, c.lcmConfig.CommonParams.RgwPublicAccessLabel),
	}
	backendName := fmt.Sprintf("rook-ceph-rgw-%s", rgwName)
	if c.lcmConfig.CommonParams.KeepIngress {
		ingresses, err := c.api.Kubeclientset.NetworkingV1().Ingresses(c.lcmConfig.RookNamespace).List(c.context, listOptions)
		if err != nil {
			c.log.Error().Err(err).Msg("")
			return nil, []string{fmt.Sprintf("failed to check ingresses in '%s' namespace", c.lcmConfig.RookNamespace)}
		}
		if len(ingresses.Items) > 0 {
			endpoints := []string{}
			issues := []string{}
			for _, ingress := range ingresses.Items {
				if len(ingress.Spec.Rules) == 0 {
					msg := fmt.Sprintf("ingress '%s/%s' has no rules configured, can't find Ceph RGW public endpoint", c.lcmConfig.RookNamespace, ingress.Name)
					issues = append(issues, msg)
					c.log.Warn().Msg(msg)
					continue
				}
				found := false
				for _, rule := range ingress.Spec.Rules {
					if rule.HTTP != nil {
						for _, path := range rule.HTTP.Paths {
							if (path.Backend.Service != nil && path.Backend.Service.Name == backendName) ||
								(path.Backend.Resource != nil && path.Backend.Resource.Name == backendName && path.Backend.Resource.Kind == "CephObjectStore") {
								endpoints = append(endpoints, "https://"+rule.Host)
								found = true
								break
							}
						}
					}
				}
				if !found {
					msg := fmt.Sprintf("can't determine Ceph RGW public endpoint for ingress '%s/%s', backend '%s' is not found in ingress rules",
						c.lcmConfig.RookNamespace, ingress.Name, backendName)
					c.log.Warn().Msg(msg)
					issues = append(issues, msg)
					continue
				}
				if len(ingress.Status.LoadBalancer.Ingress) == 0 {
					msg := fmt.Sprintf("ingress '%s/%s' has no listed IP addresses available, public endpoint is not available", c.lcmConfig.RookNamespace, ingress.Name)
					c.log.Warn().Msg(msg)
					issues = append(issues, msg)
				}
			}
			return endpoints, issues
		}
	}
	if c.lcmConfig.CommonParams.GatewayAPIEnabled {
		routes, err := c.api.Gatewayclientset.GatewayV1().HTTPRoutes(c.lcmConfig.RookNamespace).List(c.context, listOptions)
		if err != nil {
			c.log.Error().Err(err).Msg("")
			return nil, []string{fmt.Sprintf("failed to check gateway httproutes in '%s' namespace", c.lcmConfig.RookNamespace)}
		}
		if len(routes.Items) > 0 {
			endpoints := []string{}
			issues := []string{}
			gatewayKind := gatewayapi.Kind("Service")
			serviceName := gatewayapi.ObjectName(backendName)
			for _, route := range routes.Items {
				if len(route.Spec.Rules) == 0 {
					msg := fmt.Sprintf("gateway httproute '%s/%s' has no rules configured, can't find Ceph RGW public endpoint", route.Namespace, route.Name)
					c.log.Warn().Msg(msg)
					issues = append(issues, msg)
					continue
				}
				found := false
				for _, rule := range route.Spec.Rules {
					for _, ref := range rule.BackendRefs {
						if ref.Name != "" && ref.Kind != nil {
							if ref.Name == serviceName && *ref.Kind == gatewayKind {
								for _, h := range route.Spec.Hostnames {
									endpoints = append(endpoints, "https://"+string(h))
								}
								found = true
								break
							}
						}
					}
				}
				if !found {
					msg := fmt.Sprintf("can't determine Ceph RGW public endpoint for gateway httproute '%s/%s', backend '%s' is not found in httproute rules",
						route.Namespace, route.Name, backendName)
					c.log.Warn().Msg(msg)
					issues = append(issues, msg)
					continue
				}
				stateOk := 0
				for _, parent := range route.Status.Parents {
					for _, condition := range parent.Conditions {
						if condition.Reason == "Accepted" {
							stateOk++
							break
						}
					}
				}
				if stateOk == 0 || stateOk != len(route.Status.Parents) {
					msg := fmt.Sprintf("gateway httproute '%s/%s' has not accepted some rules, public endpoint is not available", route.Namespace, route.Name)
					c.log.Warn().Msg(msg)
					issues = append(issues, msg)
				}
			}
			return endpoints, issues
		}
	}
	svcList, err := c.api.Kubeclientset.CoreV1().Services(c.lcmConfig.RookNamespace).List(c.context, listOptions)
	if err != nil {
		c.log.Error().Err(err).Msg("")
		return nil, []string{fmt.Sprintf("failed to check services in '%s' namespace", c.lcmConfig.RookNamespace)}
	}
	if len(svcList.Items) > 0 {
		endpoints := []string{}
		issues := []string{}
		for _, svc := range svcList.Items {
			if svc.Spec.Type != corev1.ServiceTypeLoadBalancer {
				msg := fmt.Sprintf("found Ceph RGW %s external service '%s/%s', but supported only '%s' service type", svc.Spec.Type, svc.Namespace, svc.Name, corev1.ServiceTypeLoadBalancer)
				c.log.Warn().Msg(msg)
				issues = append(issues, msg)
				continue
			}
			if len(svc.Status.LoadBalancer.Ingress) == 0 {
				msg := fmt.Sprintf("external service '%s/%s' has no IP addresses available, can't determine Ceph RGW public endpoint", c.lcmConfig.RookNamespace, rgwName)
				c.log.Warn().Msg(msg)
				issues = append(issues, msg)
				continue
			}
			endpoint := ""
			for _, port := range svc.Spec.Ports {
				// ports are named in the same way to Rook Ceph RGW svc
				// if no https port - will be exposed http port instead
				if port.Name == "https" {
					endpoint = fmt.Sprintf("https://%s:%d", svc.Status.LoadBalancer.Ingress[0].IP, port.Port)
					break
				}
				if port.Name == "http" {
					endpoint = fmt.Sprintf("http://%s:%d", svc.Status.LoadBalancer.Ingress[0].IP, port.Port)
				}
			}
			if endpoint != "" {
				endpoints = append(endpoints, endpoint)
			}
		}
		return endpoints, issues
	}
	c.log.Debug().Msgf("no any of httproute, ingress, external service with label '%s' is not found for Ceph RGW '%s/%s'", listOptions.LabelSelector, c.lcmConfig.RookNamespace, rgwName)
	return nil, nil
}

func (c *cephDeploymentHealthConfig) getMultisiteSyncStatus() (*lcmv1alpha1.MultisiteState, []string) {
	cmd := fmt.Sprintf("radosgw-admin sync status --rgw-zonegroup=%s --rgw-zone=%s", c.healthConfig.multisiteOpts.zonegroup, c.healthConfig.multisiteOpts.zone)
	syncStatusOutput, err := lcmcommon.RunCephToolboxCLI(c.context, c.api.Kubeclientset, c.api.Config, c.lcmConfig.RookNamespace, cmd)
	if err != nil {
		c.log.Error().Err(err).Msg("")
		msg := fmt.Sprintf("failed to run '%s' command to check multisite status for zone '%s'", cmd, c.healthConfig.multisiteOpts.zone)
		return &lcmv1alpha1.MultisiteState{
			MetadataSyncState: lcmv1alpha1.MultiSiteFailed,
			DataSyncState:     lcmv1alpha1.MultiSiteFailed,
			Messages:          []string{msg},
		}, []string{msg}
	}
	multisiteState := &lcmv1alpha1.MultisiteState{
		MetadataSyncState: lcmv1alpha1.MultiSiteSyncing,
		DataSyncState:     lcmv1alpha1.MultiSiteSyncing,
	}
	multisiteIssues := []string{}
	// CMD `radosgw-admin sync status` has no JSON format in 19.2 yet, so we can
	// only use regexp to determine current state
	masterZone, _ := regexp.MatchString(`metadata sync no sync \(zone is master\)`, syncStatusOutput)
	if masterZone {
		multisiteState.MasterZone = true
	} else {
		metaUpToDate, _ := regexp.MatchString(`metadata is caught up with master`, syncStatusOutput)
		if !metaUpToDate {
			metaBehind, _ := regexp.MatchString(`metadata is behind`, syncStatusOutput)
			if metaBehind {
				multisiteState.MetadataSyncState = lcmv1alpha1.MultiSiteOutOfSync
				multisiteIssues = append(multisiteIssues, "metadata is behind master zone")
			} else {
				metaFetchFail, _ := regexp.MatchString(`failed to fetch mdlog info`, syncStatusOutput)
				if metaFetchFail {
					multisiteState.MetadataSyncState = lcmv1alpha1.MultiSiteFailed
					multisiteIssues = append(multisiteIssues, "failed to fetch metadata info")
				} else {
					// unknown state - mark as failed, since is not behind and not ok
					multisiteState.MetadataSyncState = lcmv1alpha1.MultiSiteFailed
					multisiteIssues = append(multisiteIssues, "unknown metadata sync state")
				}
			}
		}
	}
	dataSyncing, _ := regexp.MatchString(`\sdata sync source`, syncStatusOutput)
	if dataSyncing {
		dataFetchFail, _ := regexp.MatchString(`failed to fetch datalog info`, syncStatusOutput)
		if dataFetchFail {
			multisiteState.DataSyncState = lcmv1alpha1.MultiSiteFailed
			multisiteIssues = append(multisiteIssues, "failed to fetch data info")
		} else {
			// since there may be multiple replicated clusters
			// need to check all sources
			reg := regexp.MustCompile(`source:\s.*\)`)
			sources := reg.FindAllString(syncStatusOutput, -1)
			reg = regexp.MustCompile(`\sdata is caught up with source`)
			upToDate := reg.FindAllString(syncStatusOutput, -1)
			if len(sources) != len(upToDate) {
				reg = regexp.MustCompile(`\sdata is behind on`)
				behind := reg.FindAllString(syncStatusOutput, -1)
				if len(behind)+len(upToDate) == len(sources) {
					multisiteState.DataSyncState = lcmv1alpha1.MultiSiteOutOfSync
					// do not raise health issue on master side if some problems with secondary
					if !masterZone {
						multisiteIssues = append(multisiteIssues, "data is behind master zone")
					}
				} else {
					// unknown state - mark as failed, since is not behind and not ok
					multisiteState.DataSyncState = lcmv1alpha1.MultiSiteFailed
					multisiteIssues = append(multisiteIssues, "unknown data sync state")
				}
			}
		}
	} else {
		// when multisite is starting configuring and no other zones except master - there is no
		// data sync yet, so just skip such case from warnings
		if !masterZone {
			multisiteState.DataSyncState = lcmv1alpha1.MultiSiteFailed
			multisiteIssues = append(multisiteIssues, "data sync info is not present")
		}
	}
	if len(multisiteIssues) > 0 {
		multisiteState.Messages = multisiteIssues
	}
	// do not fail master zone health check, show only issues if present in log
	if masterZone && len(multisiteIssues) > 0 {
		c.log.Error().Msgf("found problems with RGW multisite: %s", strings.Join(multisiteIssues, ", "))
		multisiteIssues = make([]string, 0)
	}
	return multisiteState, multisiteIssues
}
