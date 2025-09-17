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
	"sort"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	lcmcommon "github.com/Mirantis/pelagia/pkg/common"
)

type configOption struct {
	// put value only in config-override map, not in runtime
	confFile bool
	// do not allow to override value at all
	static  bool
	key     string
	value   string
	section string
	// control backward compatiblity for ceph config options
	availableFrom *lcmcommon.CephVersion
	// more control backward compatibility for ceph config options
	availableTo *lcmcommon.CephVersion
}

const sectionDelimiter = "|"

var exceptionParams = map[string]string{
	// Since MKE 3.5.7 has pod-max-pids limit to 500, we need to decrease
	// number of threads from default 512 to 256.
	// https://mirantis.jira.com/browse/PRODX-30304
	"rgw_thread_pool_size": "256",
}

var generalConfigOptions = []configOption{
	{key: "mon_target_pg_per_osd", value: "100", section: "global"},
	{key: "mon_max_pg_per_osd", value: "300", section: "global"},
	// Disable insecure global_id warning due to
	// https://mirantis.jira.com/browse/PRODX-16964
	{key: "mon_warn_on_insecure_global_id_reclaim", value: "false", section: "mon"},
	{key: "mon_warn_on_insecure_global_id_reclaim_allowed", value: "false", section: "mon"},
	// Pin osd_class_dir to /usr/lib64 due to incorrect way of libdir definition since Reef
	// https://tracker.ceph.com/issues/57250 https://github.com/ceph/ceph/pull/47676
	{key: "osd_class_dir", value: "/usr/lib64/rados-classes", section: "osd", confFile: true},
	// discard is not enabled by default in Rook: https://github.com/rook/rook/issues/6964
	// which may lead to high fragmentation rate and slow performance on long-live osds
	{key: "bdev_enable_discard", value: "true", section: "osd"},
	{key: "bdev_async_discard_threads", value: "1", section: "osd"},
}

// client.rgw section config opts
func defaultRgwConfigOptions(rgwSectionName string) []configOption {
	return []configOption{
		// Due to https://github.com/rook/rook/issues/7573 we need to workaround
		// rgw failure on updating to rook 1.6.x versions
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
		{key: "rgw_thread_pool_size", value: exceptionParams["rgw_thread_pool_size"], section: rgwSectionName},
	}
}

// client.rgw section config opts
func getDefaultRgwKeystoneConfig(rgwSectionName string, keystoneSecret map[string]string) []configOption {
	return []configOption{
		{key: "rgw_keystone_api_version", value: "3", section: rgwSectionName, static: true},
		{key: "rgw_keystone_url", value: keystoneSecret["auth_url"], section: rgwSectionName},
		{key: "rgw_keystone_admin_user", value: keystoneSecret["username"], section: rgwSectionName},
		{key: "rgw_keystone_admin_password", value: keystoneSecret["password"], section: rgwSectionName},
		{key: "rgw_keystone_admin_domain", value: keystoneSecret["project_domain_name"], section: rgwSectionName},
		{key: "rgw_keystone_admin_project", value: keystoneSecret["project_name"], section: rgwSectionName},
	}
}

// client.rgw section config opts
func getDefaultRgwBarbicanConfig(rgwSectionName string, keystoneSecret map[string]string) []configOption {
	return []configOption{
		{key: "rgw_crypt_s3_kms_backend", value: "barbican", section: rgwSectionName, static: true},
		{key: "rgw_barbican_url", value: keystoneSecret["barbican_url"], section: rgwSectionName},
		{key: "rgw_keystone_barbican_user", value: keystoneSecret["username"], section: rgwSectionName},
		{key: "rgw_keystone_barbican_password", value: keystoneSecret["password"], section: rgwSectionName},
		{key: "rgw_keystone_barbican_domain", value: keystoneSecret["project_domain_name"], section: rgwSectionName},
		{key: "rgw_keystone_barbican_project", value: keystoneSecret["project_name"], section: rgwSectionName},
	}
}

// client.rgw section config opts
func getDefaultRgwOpenStackConfig(rgwSectionName string) []configOption {
	return []configOption{
		{key: "rgw_swift_account_in_url", value: "true", section: rgwSectionName},
		{key: "rgw_keystone_accepted_roles", value: "'_member_, Member, member, swiftoperator'", section: rgwSectionName},
		{key: "rgw_keystone_accepted_admin_roles", value: "'admin, service'", section: rgwSectionName},
		{key: "rgw_keystone_implicit_tenants", value: "true", section: rgwSectionName},
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

// client.rgw section config opts
var passwordKeys = []string{"rgw_keystone_admin_password", "rgw_keystone_barbican_password"}

// since no more runtime keys, use same list
var runtimeKeys = passwordKeys

func (c *cephDeploymentConfig) ensureCephConfig(cephClusterPresent bool) (bool, error) {
	c.log.Debug().Msgf("ensure ceph config for cluster %s/%s", c.lcmConfig.RookNamespace, c.cdConfig.cephDpl.Name)
	cephOverrideConfig, runtimeConfig, configHashes, err := c.buildCephConfig()
	if err != nil {
		c.log.Error().Err(err).Msg("failed to build ceph config")
		return false, errors.Wrap(err, "failed to prepare ceph config map")
	}
	currentGenTime := lcmcommon.GetCurrentTimeString()
	newRookCm := &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rookConfigOverrideName,
			Namespace: c.lcmConfig.RookNamespace,
			Annotations: map[string]string{
				cephConfigMapUpdateTimestampLabel: currentGenTime,
			},
		},
		Data: map[string]string{
			"config": cephOverrideConfig,
			"runtime": func() string {
				var runtimeString strings.Builder
				for _, k := range lcmcommon.SortedMapKeys(runtimeConfig) {
					_, key := getSectionAndKey(k)
					if lcmcommon.Contains(passwordKeys, key) {
						runtimeString.WriteString(fmt.Sprintf("%s = *\n", k))
					} else {
						runtimeString.WriteString(fmt.Sprintf("%s = %s\n", k, runtimeConfig[k]))
					}
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
				curKey := strings.TrimSpace(strings.Split(line, "=")[0])
				if _, present := runtimeConfig[curKey]; !present {
					runtimeConfig[curKey] = ""
				}
			}
		}
		stateChanged, err = c.updateRuntimeParameters(runtimeConfig)
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
	// control runtime parameters update to always have actual one
	prevRgwRuntimeUpdate := currentRookCm.Annotations[cephRuntimeRgwParametersUpdateTimestampLabel]
	if resourceUpdateTimestamps.rgwRuntimeParams == "" && prevRgwRuntimeUpdate != "" {
		resourceUpdateTimestamps.rgwRuntimeParams = prevRgwRuntimeUpdate
	}
	if resourceUpdateTimestamps.rgwRuntimeParams != "" {
		newAnnotations[cephRuntimeRgwParametersUpdateTimestampLabel] = resourceUpdateTimestamps.rgwRuntimeParams
	}
	prevOsdRuntimeUpdate := currentRookCm.Annotations[cephRuntimeOsdParametersUpdateTimestampLabel]
	if resourceUpdateTimestamps.osdRuntimeParams == "" && prevOsdRuntimeUpdate != "" {
		resourceUpdateTimestamps.osdRuntimeParams = prevOsdRuntimeUpdate
	}
	if resourceUpdateTimestamps.osdRuntimeParams != "" {
		newAnnotations[cephRuntimeOsdParametersUpdateTimestampLabel] = resourceUpdateTimestamps.osdRuntimeParams
	}
	configMapUpdated := !reflect.DeepEqual(currentRookCm.Data, newRookCm.Data)
	annotationsUpdated := !reflect.DeepEqual(currentRookCm.Annotations, newAnnotations)
	if configMapUpdated || annotationsUpdated {
		if configMapUpdated {
			c.log.Info().Msgf("updating configmap data %s/%s", currentRookCm.Namespace, currentRookCm.Name)
			lcmcommon.ShowObjectDiff(*c.log, currentRookCm, newRookCm)
			newAnnotations[cephConfigMapUpdateTimestampLabel] = newRookCm.Annotations[cephConfigMapUpdateTimestampLabel]
			currentRookCm.Data = newRookCm.Data
		}
		if annotationsUpdated {
			c.log.Info().Msgf("updating configmap %s/%s annotations", currentRookCm.Namespace, currentRookCm.Name)
			lcmcommon.ShowObjectDiff(*c.log, currentRookCm.Annotations, newAnnotations)
		}
		currentRookCm.Annotations = newAnnotations
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

func (c *cephDeploymentConfig) updateRuntimeParameters(runtimeConfig map[string]string) (bool, error) {
	if len(runtimeConfig) == 0 {
		return false, nil
	}
	configDump, err := c.getCephConfigDump()
	if err != nil {
		return false, err
	}
	runtimeUpdated := false
	for sectionKey, value := range runtimeConfig {
		curValue := ""
		section, key := getSectionAndKey(sectionKey)
		for _, configOption := range configDump {
			if configOption.Name == key && configOption.Section == section {
				curValue = configOption.Value
				break
			}
		}
		if curValue != value {
			cmd := ""
			if value == "" {
				cmd = fmt.Sprintf("ceph config rm %s %s", section, key)
			} else {
				cmd = fmt.Sprintf("ceph config set %s %s %s", section, key, value)
			}
			_, err := lcmcommon.RunCephToolboxCLI(c.context, c.api.Kubeclientset, c.api.Config, c.lcmConfig.RookNamespace, cmd)
			if err != nil {
				errMsg := fmt.Sprintf("failed to update '%s' parameter", key)
				c.log.Error().Err(err).Msg(errMsg)
				return false, errors.Wrap(err, errMsg)
			}
			if value == "" {
				c.log.Info().Msgf("unset ceph config parameter '[%s] %s'", section, key)
			} else {
				if !lcmcommon.Contains(passwordKeys, key) {
					c.log.Info().Msgf("updated ceph config parameter '[%s] %s' from '%s' to '%s'", section, key, curValue, value)
				} else {
					c.log.Info().Msgf("updated ceph config parameter '[%s] %s'", section, key)
				}
			}
			runtimeUpdated = true
			if strings.HasPrefix(section, "client.rgw") {
				resourceUpdateTimestamps.rgwRuntimeParams = lcmcommon.GetCurrentTimeString()
			} else if strings.HasPrefix(section, "osd") || strings.HasPrefix(key, "osd") {
				resourceUpdateTimestamps.osdRuntimeParams = lcmcommon.GetCurrentTimeString()
			}
		}
	}
	return runtimeUpdated, nil
}

func (c *cephDeploymentConfig) buildCephConfig() (string, map[string]string, map[string]string, error) {
	baseCephConfig := map[string]configOption{}
	mergeConfig := func(options []configOption) {
		for _, opt := range options {
			baseCephConfig[getFullKeyWithSection(opt.section, opt.key)] = opt
		}
	}
	if c.cdConfig.cephDpl.Spec.Network.Provider == "" || c.cdConfig.cephDpl.Spec.Network.Provider == "host" {
		// put network params and do not allow to override it through rook config
		mergeConfig([]configOption{
			{static: true, section: "global", key: "cluster_network", value: c.cdConfig.cephDpl.Spec.Network.ClusterNet},
			{static: true, section: "global", key: "public_network", value: c.cdConfig.cephDpl.Spec.Network.PublicNet},
		})
	}
	mergeConfig(generalConfigOptions)
	if c.cdConfig.cephDpl.Spec.ObjectStorage != nil {
		// rgw dns name parameter has next priority (from higher to lower):
		// 1. rook override config in spec if present
		// 2. ingress domain if present
		// 3. openstack domain if present
		// 4. default pkg svc
		rgwDNSNameValue := fmt.Sprintf("%s.%s.svc", buildRGWName(c.cdConfig.cephDpl.Spec.ObjectStorage.Rgw.Name, ""), c.lcmConfig.RookNamespace)
		ingressTLS := getIngressTLS(c.cdConfig.cephDpl)
		if ingressTLS != nil {
			if ingressTLS.Hostname != "" {
				rgwDNSNameValue = fmt.Sprintf("%s.%s", ingressTLS.Hostname, ingressTLS.Domain)
			} else {
				rgwDNSNameValue = fmt.Sprintf("%s.%s", c.cdConfig.cephDpl.Spec.ObjectStorage.Rgw.Name, ingressTLS.Domain)
			}
		}
		rgwSectionName := rgwConfigSectionName(c.cdConfig.cephDpl.Spec.ObjectStorage.Rgw.Name)
		mergeConfig(defaultRgwConfigOptions(rgwSectionName))
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
				rgwDNSNameValue = fmt.Sprintf("%s.%s", c.cdConfig.cephDpl.Spec.ObjectStorage.Rgw.Name, domain)
			}
			mergeConfig(getDefaultRgwOpenStackConfig(rgwSectionName))
			mergeConfig(getDefaultRgwKeystoneConfig(rgwSectionName, secretConfig))
			if _, barbicanURLPresent := secretConfig["barbican_url"]; barbicanURLPresent {
				mergeConfig(getDefaultRgwBarbicanConfig(rgwSectionName, secretConfig))
			}
		}
		mergeConfig([]configOption{{key: "rgw_dns_name", value: rgwDNSNameValue, section: rgwSectionName}})
		// when sync thread is disabled for rgw serving clients
		// there another rgw daemon which fully operational and hidden from user
		// but we need to keep defaults for it
		if c.cdConfig.cephDpl.Spec.ObjectStorage.MultiSite != nil && c.cdConfig.cephDpl.Spec.ObjectStorage.Rgw.Gateway.SplitDaemonForMultisiteTrafficSync {
			syncDaemonSection := rgwConfigSectionName(rgwSyncDaemonName(c.cdConfig.cephDpl.Spec.ObjectStorage.Rgw.Name))
			mergeConfig(defaultRgwConfigOptions(syncDaemonSection))
		}
	}
	configMap, runtimeParams := c.getTargetConfigAndRuntime(c.cdConfig.cephDpl.Spec.RookConfig, baseCephConfig)
	configOverrideString, hashes := getCephOverrideConfigStringAndHashes(configMap)
	return configOverrideString, runtimeParams, hashes, nil
}

func getSectionAndKey(inputKey string) (string, string) {
	configOpt := strings.Split(inputKey, sectionDelimiter)
	keySection := ""
	key := inputKey
	if len(configOpt) > 1 {
		keySection = configOpt[0]
		key = configOpt[1]
	}
	return keySection, key
}

func getFullKeyWithSection(section, key string) string {
	return fmt.Sprintf("%s%s%s", section, sectionDelimiter, key)
}

func (c *cephDeploymentConfig) getTargetConfigAndRuntime(rookConfigFromSpec map[string]string, baseConfig map[string]configOption) (map[string]map[string]string, map[string]string) {
	finalCephConfigMap := map[string]map[string]string{}
	runtimeParams := map[string]string{}
	isParamForRuntime := func(section, key string) bool {
		return lcmcommon.Contains(runtimeKeys, key) ||
			strings.HasPrefix(key, "osd") ||
			strings.HasPrefix(key, "clog") ||
			strings.HasPrefix(section, "osd")
	}
	// fill with defaults first
	for param, paramOpt := range baseConfig {
		if paramOpt.availableTo != nil && !lcmcommon.CephVersionGreaterOrEqual(paramOpt.availableTo, c.cdConfig.currentCephVersion) {
			c.log.Info().Msgf("skipping default config parameter '%s' since already not available for current '%s %s' version", param, c.cdConfig.currentCephVersion.Name, c.cdConfig.currentCephVersion.MajorVersion)
			continue
		}
		if paramOpt.availableFrom != nil && !lcmcommon.CephVersionGreaterOrEqual(c.cdConfig.currentCephVersion, paramOpt.availableFrom) {
			c.log.Info().Msgf("skipping default config parameter '%s' since not yet available for current '%s %s' version", param, c.cdConfig.currentCephVersion.Name, c.cdConfig.currentCephVersion.MajorVersion)
			continue
		}
		if !paramOpt.static && !paramOpt.confFile && isParamForRuntime(paramOpt.section, paramOpt.key) {
			runtimeParams[param] = paramOpt.value
			continue
		}
		if _, ok := finalCephConfigMap[paramOpt.section]; ok {
			finalCephConfigMap[paramOpt.section][paramOpt.key] = paramOpt.value
		} else {
			finalCephConfigMap[paramOpt.section] = map[string]string{paramOpt.key: paramOpt.value}
		}
	}
	// check overrides
	for param, value := range rookConfigFromSpec {
		keySection, rawKey := getSectionAndKey(param)
		noSpaceHyphenKey := strings.ReplaceAll(strings.ReplaceAll(rawKey, " ", "_"), "-", "_")
		// if no section specified in rookConfig - probably it is global
		// but give an attempt to find along default params
		if keySection == "" {
			keySection = "global"
			for _, opt := range baseConfig {
				if opt.key == noSpaceHyphenKey {
					keySection = opt.section
					break
				}
			}
			c.log.Warn().Msgf("parameter from rookConfig '%s' has no section specified, setting section '%s' by default", param, keySection)
		}
		// check that values are not allowed to be changed - not changed
		// since we already filled with defaults - just skip
		if baseOpt, ok := baseConfig[getFullKeyWithSection(keySection, noSpaceHyphenKey)]; ok {
			if baseOpt.section == keySection && baseOpt.static {
				continue
			}
		}
		// check keys from rookConfig which are exceptions
		if exceptionDefaultValue, ok := exceptionParams[noSpaceHyphenKey]; ok {
			// rgw_thread_pool_size can't be more than 256
			if noSpaceHyphenKey == "rgw_thread_pool_size" {
				size, err := strconv.Atoi(value)
				if err != nil {
					c.log.Warn().Msgf("rookConfig '%s' value expects quoted integer format, setting value to a default \"%s\": %v", param, exceptionDefaultValue, err)
					value = exceptionDefaultValue
				} else if size > 256 {
					c.log.Warn().Msgf("rookConfig '%s' value is higher than 256 which is restricted since MKE 3.5.7. Decreasing '%s' to default '%s'", param, value, exceptionDefaultValue)
					value = exceptionDefaultValue
				}
			}
		}
		if isParamForRuntime(keySection, noSpaceHyphenKey) {
			runtimeParams[getFullKeyWithSection(keySection, noSpaceHyphenKey)] = value
			continue
		}
		if _, ok := finalCephConfigMap[keySection]; ok {
			finalCephConfigMap[keySection][noSpaceHyphenKey] = value
		} else {
			finalCephConfigMap[keySection] = map[string]string{noSpaceHyphenKey: value}
		}
	}
	return finalCephConfigMap, runtimeParams
}

func getCephOverrideConfigStringAndHashes(cephConfigMap map[string]map[string]string) (string, map[string]string) {
	var rookOverrideConfig strings.Builder
	reorderKeys := func(keys, priorityKeys []string) []string {
		newKeys := make([]string, 0, len(keys))
		for _, priorityKey := range priorityKeys {
			realIdx := -1
			for idx, key := range keys {
				if key == priorityKey {
					realIdx = idx
					break
				}
			}
			if realIdx == -1 {
				continue
			}
			newKeys = append(newKeys, priorityKey)
			keys = append(keys[:realIdx], keys[realIdx+1:]...)
		}
		newKeys = append(newKeys, keys...)
		return newKeys
	}

	hashes := map[string]string{}
	// put cluster/public network params at the top of config always
	sections := make([]string, 0, len(cephConfigMap))
	for k := range cephConfigMap {
		sections = append(sections, k)
	}
	sort.Strings(sections)
	for _, section := range reorderKeys(sections, []string{"global", "mon", "mgr"}) {
		sectionOpts := cephConfigMap[section]
		sectionKeys := lcmcommon.SortedMapKeys(sectionOpts)
		if section == "global" {
			sectionKeys = reorderKeys(sectionKeys, []string{"cluster_network", "public_network"})
		} else {
			rookOverrideConfig.WriteString("\n")
		}
		rookOverrideConfig.WriteString(fmt.Sprintf("[%s]\n", section))
		for _, key := range sectionKeys {
			s := fmt.Sprintf("%s = %s\n", key, sectionOpts[key])
			// since we have mon,mgr,mds,rgw daemons controlled by our reconcile
			// support restart only for those daemons + global restart
			if section == "global" {
				hashes["global"] += s
			} else if strings.HasPrefix(section, "mon") {
				hashes["mon"] += s
			} else if strings.HasPrefix(section, "mgr") {
				hashes["mgr"] += s
			} else if strings.HasPrefix(section, "mds") {
				hashes[section] += s
			} else if strings.HasPrefix(section, "client.rgw") {
				hashes[section] += s
			}
			rookOverrideConfig.WriteString(s)
		}
	}
	for k, s := range hashes {
		hashes[k] = lcmcommon.GetStringSha256(s)
	}
	return rookOverrideConfig.String(), hashes
}
