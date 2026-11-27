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

package deployment

import (
	"fmt"
	"reflect"
	"slices"
	"sort"
	"strings"

	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	lcmcommon "github.com/Mirantis/pelagia/v3/pkg/common"
)

type configOption struct {
	static  bool
	key     string
	value   string
	section string
	secret  bool
	// control backward compatiblity for ceph config options
	availableFrom *lcmcommon.CephVersion
	// more control backward compatibility for ceph config options
	availableTo *lcmcommon.CephVersion
}

const sectionDelimiter = "|"

var defaultConfigFileOptions = []configOption{
	// Pin osd_class_dir to /usr/lib64 due to incorrect way of libdir definition since Reef
	// https://tracker.ceph.com/issues/57250 https://github.com/ceph/ceph/pull/47676
	{key: "osd_class_dir", value: "/usr/lib64/rados-classes", section: "osd"},
}

func defaultRgwKeystoneConfigFileOptions(rgwSectionName string, keystoneSecret map[string]string) []configOption {
	return []configOption{
		// TODO: to be removed, not needed anymore, v3 used by default
		{key: "rgw_keystone_api_version", value: "3", section: rgwSectionName, static: true},
		{key: "rgw_keystone_url", value: keystoneSecret["auth_url"], section: rgwSectionName},
		{key: "rgw_keystone_admin_user", value: keystoneSecret["username"], section: rgwSectionName},
		{key: "rgw_keystone_admin_domain", value: keystoneSecret["project_domain_name"], section: rgwSectionName},
		{key: "rgw_keystone_admin_project", value: keystoneSecret["project_name"], section: rgwSectionName},
	}
}

func defaultRgwBarbicanConfigFileOptions(rgwSectionName string, keystoneSecret map[string]string) []configOption {
	return []configOption{
		{key: "rgw_crypt_s3_kms_backend", value: "barbican", section: rgwSectionName},
		{key: "rgw_barbican_url", value: keystoneSecret["barbican_url"], section: rgwSectionName},
		{key: "rgw_keystone_barbican_user", value: keystoneSecret["username"], section: rgwSectionName},
		{key: "rgw_keystone_barbican_domain", value: keystoneSecret["project_domain_name"], section: rgwSectionName},
		{key: "rgw_keystone_barbican_project", value: keystoneSecret["project_name"], section: rgwSectionName},
	}
}

func defaultRgwOpenstackConfigFileOptions(rgwSectionName string) []configOption {
	return []configOption{
		{key: "rgw_keystone_accepted_roles", value: "'_member_, Member, member, swiftoperator'", section: rgwSectionName},
		{key: "rgw_keystone_accepted_admin_roles", value: "'admin, service'", section: rgwSectionName},
		{key: "rgw_keystone_implicit_tenants", value: "true", section: rgwSectionName},
	}
}

var defaultRuntimeOptions = []configOption{
	// TODO: to be removed, already default
	{key: "mon_target_pg_per_osd", value: "100", section: "global"},
	{key: "mon_max_pg_per_osd", value: "300", section: "global"},
	// Disable insecure global_id warning due to
	// https://mirantis.jira.com/browse/PRODX-16964
	// TODO: to be removed, not needed anymore, mostly was used to avoid CVE-2021-20288
	// which is already fixed for Squid and Tentacle
	// So avoid unneeded insecure option
	{key: "mon_warn_on_insecure_global_id_reclaim", value: "false", section: "mon"},
	{key: "mon_warn_on_insecure_global_id_reclaim_allowed", value: "false", section: "mon"},
	// discard is not enabled by default in Rook: https://github.com/rook/rook/issues/6964
	// which may lead to high fragmentation rate and slow performance on long-live osds
	{key: "bdev_enable_discard", value: "true", section: "osd"},
	{key: "bdev_async_discard_threads", value: "1", section: "osd"},
}

func defaultRgwRuntimeOptions(rgwSectionName string) []configOption {
	return []configOption{
		// TODO: to be removed, not needed anymore
		{key: "rgw_data_log_backing", value: "omap", section: rgwSectionName},
		{key: "rgw_max_attr_name_len", value: "64", section: rgwSectionName},
		{key: "rgw_max_attrs_num_in_req", value: "32", section: rgwSectionName},
		{key: "rgw_max_attr_size", value: "1024", section: rgwSectionName},
		{key: "rgw_bucket_quota_ttl", value: "30", section: rgwSectionName},
		{key: "rgw_user_quota_bucket_sync_interval", value: "30", section: rgwSectionName},
		{key: "rgw_user_quota_sync_interval", value: "30", section: rgwSectionName},
		{key: "rgw_trust_forwarded_https", value: "true", section: rgwSectionName},
		// Since MKE 3.5.7 has pod-max-pids limit to 500, we need to decrease
		// number of threads from default 512 to 256.
		// https://mirantis.jira.com/browse/PRODX-30304
		// TODO: will be removed in next release and can be overriden already if needed
		{key: "rgw_thread_pool_size", value: "256", section: rgwSectionName},
	}
}

func defaultRgwOpenstackRuntimeOptions(rgwSectionName string) []configOption {
	return []configOption{
		{key: "rgw_swift_account_in_url", value: "true", section: rgwSectionName},
		{key: "rgw_swift_versioning_enabled", value: "true", section: rgwSectionName},
		{key: "rgw_enforce_swift_acls", value: "true", section: rgwSectionName},
		{key: "rgw_s3_auth_use_keystone", value: "true", section: rgwSectionName},
		{key: "rgw_enable_usage_log", value: "true", section: rgwSectionName},
		{key: "rgw_usage_log_tick_interval", value: "30", section: rgwSectionName},
		{key: "rgw_usage_log_flush_threshold", value: "1024", section: rgwSectionName},
		{key: "rgw_usage_max_shards", value: "32", section: rgwSectionName},
		{key: "rgw_usage_max_user_shards", value: "1", section: rgwSectionName},
	}
}

func rgwOpenstackRuntimePasswords(rgwSectionName, option, password string) configOption {
	opt := "rgw_keystone_admin_password"
	if option == "barbican" {
		opt = "rgw_keystone_barbican_password"
	}
	return configOption{key: opt, value: password, section: rgwSectionName, secret: true}
}

func (c *cephDeploymentConfig) ensureCephConfig(cephClusterPresent bool) (bool, error) {
	c.log.Debug().Msg("ensure ceph config")
	cephOverrideConfig, runtimeOptions, configHashes, err := c.buildCephConfig()
	if err != nil {
		c.log.Error().Err(err).Msg("failed to build ceph config")
		return false, errors.Wrap(err, "failed to prepare ceph config map")
	}
	currentGenTime := lcmcommon.GetCurrentTimeString()
	newRookCm := &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rookConfigOverrideName,
			Namespace: c.lcmConfig.RookNamespace,
			Labels:    baseResourceLabels,
			Annotations: map[string]string{
				cephConfigMapUpdateTimestampLabel: currentGenTime,
			},
		},
		Data: map[string]string{
			"config": cephOverrideConfig,
			"runtime": func() string {
				var runtimeString strings.Builder
				for _, runtimeOpt := range runtimeOptions {
					value := runtimeOpt.value
					if runtimeOpt.secret {
						value = "*"
					}
					runtimeString.WriteString(fmt.Sprintf("%s%s%s = %s\n", runtimeOpt.section, sectionDelimiter, runtimeOpt.key, value))
				}
				return runtimeString.String()
			}(),
		},
	}
	for section, hash := range configHashes {
		newRookCm.Annotations[fmt.Sprintf(cephConfigSectionHashLabel, section)] = hash
		newRookCm.Annotations[fmt.Sprintf(cephConfigParametersUpdateTimestampLabel, section)] = currentGenTime
	}

	currentRookCm, err := c.api.Kubeclientset.CoreV1().ConfigMaps(c.lcmConfig.RookNamespace).Get(c.context, rookConfigOverrideName, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			c.log.Info().Msgf("creating configmap '%s/%s'", newRookCm.Namespace, newRookCm.Name)
			_, err = c.api.Kubeclientset.CoreV1().ConfigMaps(c.lcmConfig.RookNamespace).Create(c.context, newRookCm, metav1.CreateOptions{})
			if err == nil {
				for section := range configHashes {
					resourceUpdateTimestamps.cephConfigMap[section] = currentGenTime
				}
				return true, nil
			}
		}
		c.log.Error().Err(err).Msg("failed to get ceph config configmap")
		return false, err
	}
	if currentRookCm.Annotations == nil {
		currentRookCm.Annotations = map[string]string{}
	}
	stateChanged := false
	if cephClusterPresent {
		if newRookCm.Data["runtime"] != currentRookCm.Data["runtime"] {
			curRuntime := strings.Split(currentRookCm.Data["runtime"], "\n")
			for _, line := range curRuntime {
				if line == "" {
					continue
				}
				curOpt := strings.TrimSpace(strings.Split(line, "=")[0])
				opts := strings.Split(curOpt, sectionDelimiter)
				curSec, curKey := opts[0], opts[1]
				found := false
				for _, runtimeOpt := range runtimeOptions {
					if runtimeOpt.section == curSec && runtimeOpt.key == curKey {
						found = true
						break
					}
				}
				if !found {
					runtimeOptions = append(runtimeOptions, configOption{section: curSec, key: curKey})
				}
			}
		}
		stateChanged, err = c.updateRuntimeParameters(runtimeOptions)
		if err != nil {
			return false, err
		}
	}
	actualTimestamps := map[string]string{}
	pickTimestamp := func(annotation, timestamp string) string {
		if timestamp == "" {
			c.log.Warn().Msgf("missed annotation '%s' in ConfigMap '%s/%s', setting current timestamp, affected daemons will be restarted",
				annotation, currentRookCm.Namespace, currentRookCm.Name)
			return currentGenTime
		}
		return timestamp
	}
	// always control annotations for config map, to have aligned timestamps in code and in cm
	// even they are manually removed from config map by mistake
	newAnnotations := map[string]string{
		cephConfigMapUpdateTimestampLabel: pickTimestamp(cephConfigMapUpdateTimestampLabel, currentRookCm.Annotations[cephConfigMapUpdateTimestampLabel]),
	}
	for section, hash := range configHashes {
		hashLabel := fmt.Sprintf(cephConfigSectionHashLabel, section)
		timestampLabel := fmt.Sprintf(cephConfigParametersUpdateTimestampLabel, section)
		if newRookCm.Annotations[hashLabel] != currentRookCm.Annotations[hashLabel] {
			newAnnotations[hashLabel] = hash
			newAnnotations[timestampLabel] = currentGenTime
			actualTimestamps[section] = currentGenTime
		} else {
			newAnnotations[hashLabel] = pickTimestamp(hashLabel, currentRookCm.Annotations[hashLabel])
			newAnnotations[timestampLabel] = pickTimestamp(timestampLabel, currentRookCm.Annotations[timestampLabel])
			actualTimestamps[section] = newAnnotations[timestampLabel]
		}
	}
	configMapUpdated := !reflect.DeepEqual(currentRookCm.Data, newRookCm.Data)
	annotationsUpdated := !reflect.DeepEqual(currentRookCm.Annotations, newAnnotations)
	labelsUpdated := lcmcommon.AlignBaseLabels(*c.log, "configmap", &currentRookCm.ObjectMeta, newRookCm.Labels)
	if configMapUpdated || annotationsUpdated || labelsUpdated {
		if configMapUpdated {
			c.log.Info().Msgf("updating configmap data %s/%s", currentRookCm.Namespace, currentRookCm.Name)
			lcmcommon.ShowObjectDiff(*c.log, currentRookCm, newRookCm)
			newAnnotations[cephConfigMapUpdateTimestampLabel] = newRookCm.Annotations[cephConfigMapUpdateTimestampLabel]
			currentRookCm.Data = newRookCm.Data
		}
		if annotationsUpdated {
			c.log.Info().Msgf("updating configmap %s/%s annotations", currentRookCm.Namespace, currentRookCm.Name)
			lcmcommon.ShowObjectDiff(*c.log, currentRookCm.Annotations, newAnnotations)
			currentRookCm.Annotations = newAnnotations
		}
		_, err := c.api.Kubeclientset.CoreV1().ConfigMaps(c.lcmConfig.RookNamespace).Update(c.context, currentRookCm, metav1.UpdateOptions{})
		if err != nil {
			return false, err
		}
		stateChanged = true
	}
	// always set timestamps, based on actual timestamps
	resourceUpdateTimestamps.cephConfigMap = actualTimestamps
	return stateChanged, nil
}

func (c *cephDeploymentConfig) updateRuntimeParameters(runtimeConfig []configOption) (bool, error) {
	if len(runtimeConfig) == 0 {
		return false, nil
	}
	configDump, err := c.getCephConfigDump()
	if err != nil {
		return false, err
	}
	updated := false
	for _, runtimeOpt := range runtimeConfig {
		cmd := ""
		if runtimeOpt.value == "" {
			cmd = fmt.Sprintf("ceph config rm %s %s", runtimeOpt.section, runtimeOpt.key)
			c.log.Info().Msgf("unsetting ceph config parameter '[%s] %s'", runtimeOpt.section, runtimeOpt.key)
		} else {
			curValue := ""
			for _, configOption := range configDump {
				if configOption.Name == runtimeOpt.key && configOption.Section == runtimeOpt.section {
					curValue = configOption.Value
					break
				}
			}
			if curValue != runtimeOpt.value {
				cmd = fmt.Sprintf("ceph config set %s %s %s", runtimeOpt.section, runtimeOpt.key, runtimeOpt.value)
				if runtimeOpt.secret {
					c.log.Info().Msgf("updating ceph config parameter '[%s] %s'", runtimeOpt.section, runtimeOpt.key)
				} else if curValue == "" {
					c.log.Info().Msgf("setting ceph config parameter '[%s] %s' to '%s'", runtimeOpt.section, runtimeOpt.key, runtimeOpt.value)
				} else {
					c.log.Info().Msgf("updating ceph config parameter '[%s] %s' from '%s' to '%s'", runtimeOpt.section, runtimeOpt.key, curValue, runtimeOpt.value)
				}
			}
		}
		if cmd != "" {
			_, err := lcmcommon.RunCephToolboxCLI(c.context, c.api.Kubeclientset, c.api.Config, c.lcmConfig.RookNamespace, cmd)
			if err != nil {
				errMsg := fmt.Sprintf("failed to update '[%s] %s' parameter", runtimeOpt.section, runtimeOpt.key)
				c.log.Error().Err(err).Msg(errMsg)
				return false, errors.Wrap(err, errMsg)
			}
			updated = true
		}
	}
	return updated, nil
}

func (c *cephDeploymentConfig) buildCephConfig() (string, []configOption, map[string]string, error) {
	baseConfigFileOptions := []configOption{}
	baseRuntimeOptions := []configOption{}
	nonUpdatablePasswords := map[string]string{}

	mergeDefaultConfig := func(config []configOption, options []configOption) []configOption {
		for _, opt := range options {
			if opt.availableTo != nil && !lcmcommon.CephVersionGreaterOrEqual(opt.availableTo, c.cdConfig.currentCephVersion) {
				c.log.Info().Msgf("skipping default config parameter '%s' since already not available for current '%s %s' version", opt.key, c.cdConfig.currentCephVersion.Name, c.cdConfig.currentCephVersion.MajorVersion)
				continue
			}
			if opt.availableFrom != nil && !lcmcommon.CephVersionGreaterOrEqual(c.cdConfig.currentCephVersion, opt.availableFrom) {
				c.log.Info().Msgf("skipping default config parameter '%s' since not yet available for current '%s %s' version", opt.key, c.cdConfig.currentCephVersion.Name, c.cdConfig.currentCephVersion.MajorVersion)
				continue
			}
			config = append(config, opt)
		}
		return config
	}

	baseConfigFileOptions = mergeDefaultConfig(baseConfigFileOptions, defaultConfigFileOptions)
	baseRuntimeOptions = mergeDefaultConfig(baseRuntimeOptions, defaultRuntimeOptions)
	// will be catched on validation, but double check
	if c.cdConfig.clusterSpec.Network.AddressRanges == nil {
		return "", nil, nil, errors.New("network ranges for public and cluster are not provided")
	}
	// put network params and do not allow to override it through rook config
	baseConfigFileOptions = mergeDefaultConfig(baseConfigFileOptions, []configOption{
		{static: true, section: "global", key: "cluster_network", value: c.cdConfig.clusterSpec.Network.AddressRanges.Cluster.String()},
		{static: true, section: "global", key: "public_network", value: c.cdConfig.clusterSpec.Network.AddressRanges.Public.String()},
	})

	if c.cdConfig.cephDpl.Spec.ObjectStorage != nil {
		for _, rgwStore := range c.cdConfig.cephDpl.Spec.ObjectStorage.Rgws {
			rgwSectionName := rgwConfigSectionName(rgwStore.Name)
			baseRuntimeOptions = mergeDefaultConfig(baseRuntimeOptions, defaultRgwRuntimeOptions(rgwSectionName))
			// if rgw.hosting is set - ceph config parameter will be ignored, otherwise
			// rgw dns name parameter has next priority (from higher to lower):
			// 1. rook override config in spec if present
			// 2. ingress domain if present
			// 3. openstack domain if present
			// 4. default pkg svc
			rgwDNSNameValues := []string{fmt.Sprintf("%s.%s.svc", buildRGWName(rgwStore.Name, ""), c.lcmConfig.RookNamespace)}
			if rgwStore.ServedByIngress || rgwStore.UsedForOpenstack {
				ingressTLS := getIngressTLS(c.cdConfig.cephDpl)
				if ingressTLS != nil && c.lcmConfig.CommonParams.KeepIngress {
					if ingressTLS.Hostname != "" {
						rgwDNSNameValues = append(rgwDNSNameValues, fmt.Sprintf("%s.%s", ingressTLS.Hostname, ingressTLS.Domain))
					} else {
						rgwDNSNameValues = append(rgwDNSNameValues, fmt.Sprintf("%s.%s", rgwStore.Name, ingressTLS.Domain))
					}
				}
				if rgwStore.UsedForOpenstack {
					baseConfigFileOptions = mergeDefaultConfig(baseConfigFileOptions, defaultRgwOpenstackConfigFileOptions(rgwSectionName))
					baseRuntimeOptions = mergeDefaultConfig(baseRuntimeOptions, defaultRgwOpenstackRuntimeOptions(rgwSectionName))
					// TODO: add ability to set flag for non-rockoon setup as well
					if err := checkOpenstackNamespaceSetForRgw(rgwStore.Name, c.lcmConfig.DeployParams.OpenstackCephSharedNamespace); err != nil {
						return "", nil, nil, err
					}
					openstackSecrets, err := c.api.Kubeclientset.CoreV1().Secrets(c.lcmConfig.DeployParams.OpenstackCephSharedNamespace).Get(c.context, openstackRgwCredsName, metav1.GetOptions{})
					if err != nil {
						if !apierrors.IsNotFound(err) {
							return "", nil, nil, err
						}
					} else {
						secretConfig := map[string]string{}
						for key, val := range openstackSecrets.Data {
							secretConfig[key] = string(val)
						}
						if domain, ok := secretConfig["public_domain"]; ok && ingressTLS == nil {
							rgwDNSNameValues = append(rgwDNSNameValues, fmt.Sprintf("%s.%s", rgwStore.Name, domain))
						}
						baseConfigFileOptions = mergeDefaultConfig(baseConfigFileOptions, defaultRgwKeystoneConfigFileOptions(rgwSectionName, secretConfig))
						keystonePassOption := rgwOpenstackRuntimePasswords(rgwSectionName, "keystone", secretConfig["password"])
						nonUpdatablePasswords[rgwSectionName] += fmt.Sprintf("%s=%s\n", keystonePassOption.key, secretConfig["password"])
						baseRuntimeOptions = mergeDefaultConfig(baseRuntimeOptions, []configOption{keystonePassOption})
						if _, barbicanURLPresent := secretConfig["barbican_url"]; barbicanURLPresent {
							baseConfigFileOptions = mergeDefaultConfig(baseConfigFileOptions, defaultRgwBarbicanConfigFileOptions(rgwSectionName, secretConfig))
							pass := secretConfig["password"]
							if p, ok := secretConfig["barbican_password"]; ok {
								pass = p
							}
							barbicanPassOption := rgwOpenstackRuntimePasswords(rgwSectionName, "barbican", pass)
							nonUpdatablePasswords[rgwSectionName] += fmt.Sprintf("%s=%s\n", keystonePassOption.key, pass)
							baseRuntimeOptions = mergeDefaultConfig(baseRuntimeOptions, []configOption{barbicanPassOption})
						}
					}
				}
			}
			rgwCasted, _ := rgwStore.GetSpec()
			// if rgw hosting config is provided - use it. Otherwise override config value
			if rgwCasted.Hosting == nil {
				if rgwCasted.Gateway.SecurePort != int32(0) {
					baseConfigFileOptions = mergeDefaultConfig(baseConfigFileOptions, []configOption{{key: "rgw_dns_name", value: strings.Join(rgwDNSNameValues, ","), section: rgwSectionName}})
				}
			}
		}
	}

	getAssimilatedKey := func(rawKey string) string {
		return strings.ReplaceAll(strings.ReplaceAll(rawKey, " ", "_"), "-", "_")
	}

	cephConfigFileOptions := map[string]map[string]string{}
	cephRuntimeOptions := []configOption{}
	if c.cdConfig.cephDpl.Spec.ExtraOpts != nil {
		if len(c.cdConfig.cephDpl.Spec.ExtraOpts.RuntimeCephConfig) > 0 {
			for section, opts := range c.cdConfig.cephDpl.Spec.ExtraOpts.RuntimeCephConfig {
				for key, opt := range opts {
					cephOpt := configOption{section: section, key: getAssimilatedKey(key), value: opt}
					cephRuntimeOptions = append(cephRuntimeOptions, cephOpt)
				}
			}
		}
		if len(c.cdConfig.cephDpl.Spec.ExtraOpts.StaticCephConfig) > 0 {
			for section, opts := range c.cdConfig.cephDpl.Spec.ExtraOpts.StaticCephConfig {
				cephConfigFileOptions[section] = map[string]string{}
				for key, opt := range opts {
					cephConfigFileOptions[section][getAssimilatedKey(key)] = opt
				}
			}
		}
	}
	for _, baseOpt := range baseConfigFileOptions {
		if curOpts, ok := cephConfigFileOptions[baseOpt.section]; ok {
			_, ok := curOpts[baseOpt.key]
			if ok && !baseOpt.static {
				// if overwritten from config and not static - allow it and keep
				continue
			}
			if ok && baseOpt.static {
				c.log.Warn().Msgf("detected option '[%s]%s' override in spec.extraOpts.staticCephConfig field, which is not designed to be overwritten", baseOpt.section, baseOpt.key)
			}
			cephConfigFileOptions[baseOpt.section][baseOpt.key] = baseOpt.value
		} else {
			cephConfigFileOptions[baseOpt.section] = map[string]string{baseOpt.key: baseOpt.value}
		}
	}
	for _, baseOpt := range baseRuntimeOptions {
		add := true
		for _, runtimeOpt := range cephRuntimeOptions {
			if runtimeOpt.section == baseOpt.section && runtimeOpt.key == baseOpt.key {
				add = false
				break
			}
		}
		if add {
			cephRuntimeOptions = append(cephRuntimeOptions, baseOpt)
		}
	}
	// Sort node lexicographically to reduce number of reconcile
	sort.Slice(cephRuntimeOptions, func(i, j int) bool {
		t1 := fmt.Sprintf("%s%s", cephRuntimeOptions[i].section, cephRuntimeOptions[i].key)
		t2 := fmt.Sprintf("%s%s", cephRuntimeOptions[j].section, cephRuntimeOptions[j].key)
		return t1 < t2
	})

	orderdedSections := getOrderedConfigSections(cephConfigFileOptions)
	confFileHashes := map[string]string{}
	var rookOverrideConfig strings.Builder
	for secIdx, section := range orderdedSections {
		if len(cephConfigFileOptions[section]) == 0 {
			continue
		}
		if secIdx != 0 {
			rookOverrideConfig.WriteString("\n")
		}
		rookOverrideConfig.WriteString(fmt.Sprintf("[%s]\n", section))
		orderedOpts := []string{}
		for k := range cephConfigFileOptions[section] {
			orderedOpts = append(orderedOpts, k)
		}
		sort.Strings(orderedOpts)
		for _, orderedOpt := range orderedOpts {
			s := fmt.Sprintf("%s = %s\n", orderedOpt, cephConfigFileOptions[section][orderedOpt])
			// do not save hashes for direct daemons since it is not used anyway for restart
			if !strings.HasPrefix(section, "mon.") && !strings.HasPrefix(section, "mgr.") && !strings.HasPrefix(section, "osd.") {
				confFileHashes[section] += s
			}
			rookOverrideConfig.WriteString(s)
		}
		if passwords, ok := nonUpdatablePasswords[section]; ok {
			confFileHashes[section] += passwords
		}
		confFileHashes[section] = lcmcommon.GetStringSha256(confFileHashes[section])
	}
	return rookOverrideConfig.String(), cephRuntimeOptions, confFileHashes, nil
}

func getOrderedConfigSections(cephConfigFileOptions map[string]map[string]string) []string {
	orderdedSections := []struct {
		name        string
		subsections []string
	}{
		{name: "global"},
		{name: "mon"},
		{name: "mgr"},
		{name: "osd"},
		{name: "mds"},
	}
	targetOrderedSections := []string{}
	for configSection := range cephConfigFileOptions {
		found := false
		for idx, sectionItem := range orderdedSections {
			if configSection == sectionItem.name {
				found = true
				break
			}
			if strings.HasPrefix(configSection, sectionItem.name) {
				if len(sectionItem.subsections) == 0 {
					orderdedSections[idx].subsections = []string{configSection}
				} else {
					orderdedSections[idx].subsections = append(orderdedSections[idx].subsections, configSection)
				}
				found = true
				break
			}
		}
		if !found {
			targetOrderedSections = append(targetOrderedSections, configSection)
		}
	}
	sort.Strings(targetOrderedSections)
	startIdx := 0
	for _, section := range orderdedSections {
		sort.Strings(section.subsections)
		toInsert := append([]string{section.name}, section.subsections...)
		targetOrderedSections = slices.Insert(targetOrderedSections, startIdx, toInsert...)
		startIdx += len(toInsert)
	}
	return targetOrderedSections
}
