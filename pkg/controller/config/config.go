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

package lcmconfig

import (
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/rs/zerolog"

	"k8s.io/apimachinery/pkg/labels"
)

type LcmConfig struct {
	// main rook deployment namespace
	RookNamespace string
	// disk daemon label
	DiskDaemonPlacementLabel string
	// disk daemon api port for collecting info
	DiskDaemonPort int32
	// params related to health controller
	HealthParams *HealthParams
	// params related to task controller
	TaskParams *TaskParams
	// params related to cephdeployment controller
	DeployParams *DeployParams
}

type HealthParams struct {
	// log level for health controller
	LogLevel zerolog.Level
	// cephdeployment health checks to skip
	ChecksSkip []string
	// ceph cluster health issues to ignore
	CephIssuesToIgnore []string
	// regexp for collection pool usage/capacity details
	UsageDetailsClassesFilter string
	// regexp for collection class usage/capacity details
	UsageDetailsPoolsFilter string
	// service selector providing rgw public access (ingress, loadbalancer)
	RgwPublicAccessLabel string
}

type TaskParams struct {
	// log level for task controller
	LogLevel zerolog.Level
	// timeout for osd rebalance during remove task execution
	OsdPgRebalanceTimeout time.Duration
	// allow to destroy and remove lvm created not by rook
	AllowToRemoveManuallyCreatedLVM bool
}

type DeployParams struct {
	// log level for task controller
	LogLevel zerolog.Level
	// ceph image represents current image with ceph installed
	CephImage string
	// ceph release to go
	CephRelease string
	// rook image to go
	RookImage string
	// deploy network policy objects or not
	NetPolEnabled bool
	// service selector providing rgw public access (ingress, loadbalancer)
	RgwPublicAccessLabel string
	// namespace for sharing secrets between openstack and ceph
	OpenstackCephSharedNamespace string
	// secret with cabundle for multisite public access between zones
	MultisiteCabundleSecretRef string
	// excluding label to place ceph daemonsets
	CephDaemonsetPlacementLabelExclude string
}

type ControlParams string

// vars to specify required params to control
var ParamsToControl = ControlParamsAll

const (
	ControlParamsAll     ControlParams = "all"
	ControlParamsHealth  ControlParams = "health"
	ControlParamsTask    ControlParams = "task"
	ControlParamsCephDpl ControlParams = "cephdeployment"
)

var (
	// main runtime var for keeping configs
	lcmConfigs = map[string]LcmConfig{}
	// default lcm config var
	defaultLcmConfig = LcmConfig{
		RookNamespace:            "rook-ceph",
		DiskDaemonPort:           9999,
		DiskDaemonPlacementLabel: "pelagia-disk-daemon=true",
	}
	// default health config var
	defaultHealthConfig = HealthParams{
		CephIssuesToIgnore: []string{
			"OSDMAP_FLAGS",
			"TOO_FEW_PGS",
			"SLOW_OPS",
			"OLD_CRUSH_TUNABLES",
			"OLD_CRUSH_STRAW_CALC_VERSION",
			"POOL_APP_NOT_ENABLED",
			"MON_DISK_LOW",
			"RECENT_CRASH",
		},
		ChecksSkip:                []string{},
		LogLevel:                  zerolog.InfoLevel,
		UsageDetailsClassesFilter: "",
		UsageDetailsPoolsFilter:   "",
		RgwPublicAccessLabel:      "external_access=rgw",
	}
	defaultTaskConfig = TaskParams{
		LogLevel:              zerolog.InfoLevel,
		OsdPgRebalanceTimeout: 30 * time.Minute,
	}
	defaultDeployParams = DeployParams{
		LogLevel:                     zerolog.InfoLevel,
		OpenstackCephSharedNamespace: "openstack-ceph-shared",
		RgwPublicAccessLabel:         "external_access=rgw",
	}
)

var (
	errorMsgTmpl = "has incorrect parameter value '%s=%s', expected %s"
	debugMsgTmpl = "set '%s=%s'"
	// general lcm config params
	rookNamespaceParameter   = "ROOK_NAMESPACE"
	diskDaemonPortParameter  = "DISK_DAEMON_API_PORT"
	diskDaemonPlacementLabel = "DISK_DAEMON_PLACEMENT_NODES_SELECTOR"
	// health controller config params
	healthChecksCephIssuesToIgnoreParameter = "HEALTH_CHECKS_CEPH_ISSUES_TO_IGNORE"
	healthChecksSkipParameter               = "HEALTH_CHECKS_SKIP"
	healthChecksUsagelClassFilterParameter  = "HEALTH_CHECKS_USAGE_CLASS_FILTER"
	healthChecksUsagelPoolsFilterParameter  = "HEALTH_CHECKS_USAGE_POOLS_FILTER"
	healthLogLevelParameter                 = "HEALTH_LOG_LEVEL"
	rgwPublicAccessServiceSelectorParameter = "RGW_PUBLIC_ACCESS_SERVICE_SELECTOR"
	// params for task controller
	taskLogLevelParameter             = "TASK_LOG_LEVEL"
	taskOsdPgRebalanceTimeout         = "TASK_OSD_PG_REBALANCE_TIMEOUT_MIN"
	taskAllowRemoveManuallyCreatedLvm = "TASK_ALLOW_REMOVE_MANUALLY_CREATED_LVMS"
	// params for ceph deployment controller
	cephDplLogLevel                  = "DEPLOYMENT_LOG_LEVEL"
	cephDplCephImage                 = "DEPLOYMENT_CEPH_IMAGE"
	cephDplCephRelease               = "DEPLOYMENT_CEPH_RELEASE"
	cephDplRookImage                 = "DEPLOYMENT_ROOK_IMAGE"
	cephDplNetPolEnabled             = "DEPLOYMENT_NETPOL_ENABLED"
	cephDplOpenstackCephSharedNs     = "DEPLOYMENT_OPENSTACK_CEPH_SHARED_NAMESPACE"
	cephDplMultisiteCabundleRef      = "DEPLOYMENT_MULTISITE_CABUNDLE_SECRET"
	cephDplCephDaemonsetLabelExclude = "DEPLOYMENT_LABEL_TO_EXCLUDE_CEPH_DAEMONSETS"
)

func dropConfiguration(namespace string) {
	delete(lcmConfigs, namespace)
}

func loadHealthConfiguration(objLog zerolog.Logger, configData map[string]string) *HealthParams {
	newHealthConfig := defaultHealthConfig
	if healthLevel, present := configData[healthLogLevelParameter]; present {
		l, err := zerolog.ParseLevel(strings.ToLower(healthLevel))
		if err != nil {
			objLog.Error().Msgf(errorMsgTmpl, healthLogLevelParameter, healthLevel, "valid log levels: info, debug, trace, warn, error")
		} else {
			objLog.Debug().Msgf(debugMsgTmpl, healthLogLevelParameter, healthLevel)
			newHealthConfig.LogLevel = l
		}
	}

	if issuesIgnore, present := configData[healthChecksCephIssuesToIgnoreParameter]; present {
		objLog.Debug().Msgf(debugMsgTmpl, healthChecksCephIssuesToIgnoreParameter, issuesIgnore)
		newHealthConfig.CephIssuesToIgnore = strings.Split(issuesIgnore, ",")
	}

	if checksSkip, present := configData[healthChecksSkipParameter]; present {
		objLog.Debug().Msgf(debugMsgTmpl, healthChecksSkipParameter, checksSkip)
		newHealthConfig.ChecksSkip = strings.Split(checksSkip, ",")
	}

	if classFilter, present := configData[healthChecksUsagelClassFilterParameter]; present {
		_, err := regexp.Compile(classFilter)
		if err != nil {
			objLog.Error().Msgf(errorMsgTmpl, healthChecksUsagelClassFilterParameter, classFilter, "valid regexp")
		} else {
			objLog.Debug().Msgf(debugMsgTmpl, healthChecksUsagelClassFilterParameter, classFilter)
			newHealthConfig.UsageDetailsClassesFilter = classFilter
		}
	}

	if poolsFilter, present := configData[healthChecksUsagelPoolsFilterParameter]; present {
		_, err := regexp.Compile(poolsFilter)
		if err != nil {
			objLog.Error().Msgf(errorMsgTmpl, healthChecksUsagelPoolsFilterParameter, poolsFilter, "valid regexp")
		} else {
			objLog.Debug().Msgf(debugMsgTmpl, healthChecksUsagelPoolsFilterParameter, poolsFilter)
			newHealthConfig.UsageDetailsPoolsFilter = poolsFilter
		}
	}

	if rgwPublicServiceSelector, present := configData[rgwPublicAccessServiceSelectorParameter]; present {
		selector, err := labels.Parse(rgwPublicServiceSelector)
		if err != nil {
			objLog.Error().Msgf(errorMsgTmpl, rgwPublicAccessServiceSelectorParameter, rgwPublicServiceSelector, "valid k8s label/selector")
		} else {
			objLog.Debug().Msgf(debugMsgTmpl, rgwPublicAccessServiceSelectorParameter, rgwPublicServiceSelector)
			newHealthConfig.RgwPublicAccessLabel = selector.String()
		}
	}
	return &newHealthConfig
}

func loadTaskConfiguration(objLog zerolog.Logger, configData map[string]string) *TaskParams {
	newTaskConfig := defaultTaskConfig

	if taskLevel, present := configData[taskLogLevelParameter]; present {
		l, err := zerolog.ParseLevel(strings.ToLower(taskLevel))
		if err != nil {
			objLog.Error().Msgf(errorMsgTmpl, taskLogLevelParameter, taskLevel, "valid log levels: info, debug, trace, warn, error")
		} else {
			objLog.Debug().Msgf(debugMsgTmpl, taskLogLevelParameter, taskLevel)
			newTaskConfig.LogLevel = l
		}
	}

	if pgTimeout, present := configData[taskOsdPgRebalanceTimeout]; present {
		mins, err := strconv.Atoi(pgTimeout)
		if err != nil {
			objLog.Error().Msgf(errorMsgTmpl, taskOsdPgRebalanceTimeout, pgTimeout, "integer")
		} else {
			objLog.Debug().Msgf(debugMsgTmpl, taskOsdPgRebalanceTimeout, pgTimeout)
			newTaskConfig.OsdPgRebalanceTimeout = time.Duration(mins) * time.Minute
		}
	}

	if value, present := configData[taskAllowRemoveManuallyCreatedLvm]; present {
		parsed, err := strconv.ParseBool(strings.TrimSuffix(value, "\n"))
		if err != nil {
			objLog.Error().Msgf(errorMsgTmpl, taskAllowRemoveManuallyCreatedLvm, value, "boolean")
		} else {
			objLog.Debug().Msgf(debugMsgTmpl, taskAllowRemoveManuallyCreatedLvm, value)
			newTaskConfig.AllowToRemoveManuallyCreatedLVM = parsed
		}
	}
	return &newTaskConfig
}

func loadCephDeploymentConfiguration(objLog zerolog.Logger, configData map[string]string) *DeployParams {
	newCephDplConfig := defaultDeployParams

	if logLevel, present := configData[cephDplLogLevel]; present {
		l, err := zerolog.ParseLevel(strings.ToLower(logLevel))
		if err != nil {
			objLog.Error().Msgf(errorMsgTmpl, cephDplLogLevel, logLevel, "valid log levels: info, debug, trace, warn, error")
		} else {
			objLog.Debug().Msgf(debugMsgTmpl, cephDplLogLevel, logLevel)
			newCephDplConfig.LogLevel = l
		}
	}

	if cephImage, present := configData[cephDplCephImage]; present {
		objLog.Debug().Msgf(debugMsgTmpl, cephDplCephImage, cephImage)
		newCephDplConfig.CephImage = cephImage
	}

	if cephRelease, present := configData[cephDplCephRelease]; present {
		objLog.Debug().Msgf(debugMsgTmpl, cephDplCephRelease, cephRelease)
		newCephDplConfig.CephRelease = cephRelease
	}

	if rookImage, present := configData[cephDplRookImage]; present {
		objLog.Debug().Msgf(debugMsgTmpl, cephDplRookImage, rookImage)
		newCephDplConfig.RookImage = rookImage
	}

	if netPolEnabled, present := configData[cephDplNetPolEnabled]; present {
		val, err := strconv.ParseBool(netPolEnabled)
		if err != nil {
			objLog.Error().Msgf(errorMsgTmpl, cephDplNetPolEnabled, netPolEnabled, "bool")
		} else {
			objLog.Debug().Msgf(debugMsgTmpl, cephDplNetPolEnabled, netPolEnabled)
			newCephDplConfig.NetPolEnabled = val
		}
	}

	if rgwPublicServiceSelector, present := configData[rgwPublicAccessServiceSelectorParameter]; present {
		selector, err := labels.Parse(rgwPublicServiceSelector)
		if err != nil {
			objLog.Error().Msgf(errorMsgTmpl, rgwPublicAccessServiceSelectorParameter, rgwPublicServiceSelector, "valid k8s label/selector")
		} else {
			objLog.Debug().Msgf(debugMsgTmpl, rgwPublicAccessServiceSelectorParameter, rgwPublicServiceSelector)
			newCephDplConfig.RgwPublicAccessLabel = selector.String()
		}
	}

	if cephDaemonsExcludeSelector, present := configData[cephDplCephDaemonsetLabelExclude]; present {
		selector, err := labels.Parse(cephDaemonsExcludeSelector)
		if err != nil {
			objLog.Error().Msgf(errorMsgTmpl, cephDplCephDaemonsetLabelExclude, cephDaemonsExcludeSelector, "valid k8s label/selector")
		} else {
			objLog.Debug().Msgf(debugMsgTmpl, cephDplCephDaemonsetLabelExclude, cephDaemonsExcludeSelector)
			newCephDplConfig.CephDaemonsetPlacementLabelExclude = selector.String()
		}
	}

	if openstackCephNs, present := configData[cephDplOpenstackCephSharedNs]; present {
		objLog.Debug().Msgf(debugMsgTmpl, cephDplOpenstackCephSharedNs, openstackCephNs)
		newCephDplConfig.OpenstackCephSharedNamespace = openstackCephNs
	}

	if multisiteCaBundle, present := configData[cephDplMultisiteCabundleRef]; present {
		objLog.Debug().Msgf(debugMsgTmpl, cephDplMultisiteCabundleRef, multisiteCaBundle)
		newCephDplConfig.MultisiteCabundleSecretRef = multisiteCaBundle
	}

	return &newCephDplConfig
}

// do not allow load configuration in-runtime out of config controller
func loadConfiguration(objLog zerolog.Logger, namespace string, configData map[string]string) {
	lcmConfigs[namespace] = ReadConfiguration(objLog, configData)
}

func ReadConfiguration(objLog zerolog.Logger, configData map[string]string) LcmConfig {
	newConfig := defaultLcmConfig
	// check rook namespace param
	if rookNamespace, present := configData[rookNamespaceParameter]; present {
		objLog.Debug().Msgf(debugMsgTmpl, rookNamespaceParameter, rookNamespace)
		newConfig.RookNamespace = rookNamespace
	}
	// check disk daemon port param
	if diskDaemonPort, present := configData[diskDaemonPortParameter]; present {
		port, err := strconv.Atoi(diskDaemonPort)
		if err != nil {
			objLog.Error().Msgf(errorMsgTmpl, diskDaemonPortParameter, diskDaemonPort, "integer")
		} else {
			objLog.Debug().Msgf(debugMsgTmpl, diskDaemonPortParameter, diskDaemonPort)
			newConfig.DiskDaemonPort = int32(port)
		}
	}
	// check disk daemon placement labels
	if diskDaemonNodeSelectorLabels, present := configData[diskDaemonPlacementLabel]; present {
		selector, err := labels.Parse(diskDaemonNodeSelectorLabels)
		if err != nil {
			objLog.Error().Msgf(errorMsgTmpl, diskDaemonPlacementLabel, diskDaemonNodeSelectorLabels, "valid k8s label/selector")
		} else {
			objLog.Debug().Msgf(debugMsgTmpl, diskDaemonPlacementLabel, diskDaemonNodeSelectorLabels)
			newConfig.DiskDaemonPlacementLabel = selector.String()
		}
	}
	// controller specific params
	switch ParamsToControl {
	case ControlParamsHealth:
		newConfig.HealthParams = loadHealthConfiguration(objLog, configData)
	case ControlParamsTask:
		newConfig.TaskParams = loadTaskConfiguration(objLog, configData)
	case ControlParamsCephDpl:
		newConfig.DeployParams = loadCephDeploymentConfiguration(objLog, configData)
	default:
		// load all by default
		newConfig.HealthParams = loadHealthConfiguration(objLog, configData)
		newConfig.TaskParams = loadTaskConfiguration(objLog, configData)
		newConfig.DeployParams = loadCephDeploymentConfiguration(objLog, configData)
	}
	return newConfig
}

func GetConfiguration(namespace string) LcmConfig {
	if lcmConfigForNs, ok := lcmConfigs[namespace]; ok {
		return lcmConfigForNs
	}
	// make a copy of default
	config := defaultLcmConfig
	healthParams := defaultHealthConfig
	taskParams := defaultTaskConfig
	deployParams := defaultDeployParams
	config.HealthParams = &healthParams
	config.TaskParams = &taskParams
	config.DeployParams = &deployParams

	return config
}
