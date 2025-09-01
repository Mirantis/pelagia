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
	"strings"
	"testing"

	"github.com/pkg/errors"
	cephv1 "github.com/rook/rook/pkg/apis/ceph.rook.io/v1"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/runtime"

	lcmv1alpha1 "github.com/Mirantis/pelagia/pkg/apis/ceph.pelagia.lcm/v1alpha1"
	lcmcommon "github.com/Mirantis/pelagia/pkg/common"
	faketestclients "github.com/Mirantis/pelagia/test/unit/clients"
	unitinputs "github.com/Mirantis/pelagia/test/unit/inputs"
)

func TestGetClusterDetailsInfo(t *testing.T) {
	tests := []struct {
		name           string
		skipChecks     bool
		cephOutputs    map[string]string
		expectedStatus *lcmv1alpha1.ClusterDetails
		expectedIssues []string
	}{
		{
			name: "cluster details with issues",
			expectedIssues: []string{
				"failed to run 'ceph df -f json' command to check capacity details",
				"failed to run 'ceph osd tree -f json' command to check replicas sizing",
				"failed to run 'ceph status -f json' command to check events details",
			},
		},
		{
			name: "cluster details no issues",
			cephOutputs: map[string]string{
				"ceph df -f json":                  unitinputs.CephDfBase,
				"ceph status -f json":              unitinputs.CephStatusBaseHealthy,
				"ceph osd tree -f json":            unitinputs.CephOsdTreeForSizingCheck,
				"ceph osd crush rule dump -f json": unitinputs.CephOsdCrushRuleDump,
				"ceph osd pool ls detail -f json":  unitinputs.CephPoolsDetails,
			},
			expectedStatus: unitinputs.CephDetailsStatusNoIssues,
			expectedIssues: []string{},
		},
		{
			name:           "skip cluster details",
			skipChecks:     true,
			expectedIssues: []string{},
		},
	}
	oldCmdRun := lcmcommon.RunPodCommand
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			lcmConfigData := map[string]string{}
			if test.skipChecks {
				lcmConfigData["HEALTH_CHECKS_SKIP"] = "usage_details,ceph_events,pools_replicas,rgw_info"
			}
			c := fakeCephReconcileConfig(nil, lcmConfigData)
			faketestclients.FakeReaction(c.api.Kubeclientset.CoreV1(), "list", []string{"pods"}, map[string]runtime.Object{"pods": unitinputs.ToolBoxPodList}, nil)

			lcmcommon.RunPodCommand = func(e lcmcommon.ExecConfig) (string, string, error) {
				if output, ok := test.cephOutputs[e.Command]; ok {
					return output, "", nil
				}
				return "", "", errors.New("command failed")
			}

			status, issues := c.getClusterDetailsInfo()
			assert.Equal(t, test.expectedStatus, status)
			assert.Equal(t, test.expectedIssues, issues)
		})
	}
	lcmcommon.RunPodCommand = oldCmdRun
}

func TestGetCephCapacityDetails(t *testing.T) {
	tests := []struct {
		name           string
		cephDfOutput   string
		checkFilters   bool
		expectedStatus *lcmv1alpha1.UsageDetails
		expectedIssue  string
	}{
		{
			name:          "failed to run ceph df",
			cephDfOutput:  "",
			expectedIssue: "failed to run 'ceph df -f json' command to check capacity details",
		},
		{
			name:           "capacity details",
			cephDfOutput:   unitinputs.CephDfBase,
			expectedStatus: unitinputs.CephBaseUsageDetails,
		},
		{
			name:           "capacity details with extra rgw/cephfs pools",
			cephDfOutput:   unitinputs.CephDfExtraPools,
			expectedStatus: unitinputs.CephExtraUsageDetails,
		},
		{
			name:         "capacity details with extra rgw/cephfs pools, but with filters",
			cephDfOutput: unitinputs.CephDfExtraPools,
			checkFilters: true,
			expectedStatus: &lcmv1alpha1.UsageDetails{
				PoolsDetail:   map[string]lcmv1alpha1.PoolUsageStats{"pool-hdd": unitinputs.CephExtraUsageDetails.PoolsDetail["pool-hdd"]},
				ClassesDetail: map[string]lcmv1alpha1.ClassUsageStats{"hdd": unitinputs.CephExtraUsageDetails.ClassesDetail["hdd"]},
			},
		},
	}
	oldCmdRun := lcmcommon.RunPodCommand
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			lcmConfigData := map[string]string{}
			if test.checkFilters {
				lcmConfigData["HEALTH_CHECKS_USAGE_CLASS_FILTER"] = "hdd"
				lcmConfigData["HEALTH_CHECKS_USAGE_POOLS_FILTER"] = "pool-hdd"
			}
			c := fakeCephReconcileConfig(nil, lcmConfigData)
			faketestclients.FakeReaction(c.api.Kubeclientset.CoreV1(), "list", []string{"pods"}, map[string]runtime.Object{"pods": unitinputs.ToolBoxPodList}, nil)
			lcmcommon.RunPodCommand = func(e lcmcommon.ExecConfig) (string, string, error) {
				if e.Command == "ceph df -f json" {
					if test.cephDfOutput != "" {
						return test.cephDfOutput, "", nil
					}
				}
				return "", "", errors.New("command failed")
			}

			status, issue := c.getCephCapacityDetails()
			assert.Equal(t, test.expectedStatus, status)
			assert.Equal(t, test.expectedIssue, issue)
		})
	}
	lcmcommon.RunPodCommand = oldCmdRun
}

func TestGetCephEvents(t *testing.T) {
	tests := []struct {
		name       string
		cephStatus string
		expected   *lcmv1alpha1.CephEvents
		issue      string
	}{
		{
			name:       "ceph event details - ceph status error",
			cephStatus: "{",
			issue:      "failed to run 'ceph status -f json' command to check events details",
		},
		{
			name:       "ceph event details - no events in cluster",
			cephStatus: unitinputs.CephStatusBaseHealthy,
			expected:   unitinputs.CephEventsIdle,
		},
		{
			name: "ceph event details - events started",
			cephStatus: unitinputs.BuildCliOutput(unitinputs.CephStatusTmpl, "status", map[string]string{
				"progress_events": `{
  "12b640c7-9734-429e-a67d-a00ab20a7635": {
    "message":"Rebalancing after osd.3 marked in (33s)\n      [==========================..] (remaining: 1s)",
    "progress":-0
  },
  "eb643ce4-af7d-4297-b136-0cbddb5cd14f":{
    "message":"PG autoscaler increasing pool 9 PGs from 32 to 128 (0s)\n      [............................] ",
    "progress": 0.2532454623
  }
}`}),
			expected: &lcmv1alpha1.CephEvents{
				RebalanceDetails: lcmv1alpha1.CephEventDetails{
					State:    lcmv1alpha1.CephEventProgressing,
					Progress: "just started",
					Messages: []lcmv1alpha1.CephEventMessage{
						{
							Message:  "Rebalancing after osd.3 marked in (33s)",
							Progress: "0",
						},
					},
				},
				PgAutoscalerDetails: lcmv1alpha1.CephEventDetails{
					State:    lcmv1alpha1.CephEventProgressing,
					Progress: "less than a half done",
					Messages: []lcmv1alpha1.CephEventMessage{
						{
							Message:  "PG autoscaler increasing pool 9 PGs from 32 to 128 (0s)",
							Progress: "0.2532454623",
						},
					},
				},
			},
		},
		{
			name:       "ceph event details - events with progress",
			cephStatus: unitinputs.CephStatusWithEvents,
			expected:   unitinputs.CephEventsProgressing,
		},
	}
	oldCmdRun := lcmcommon.RunPodCommand
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := fakeCephReconcileConfig(nil, nil)
			faketestclients.FakeReaction(c.api.Kubeclientset.CoreV1(), "list", []string{"pods"}, map[string]runtime.Object{"pods": unitinputs.ToolBoxPodList}, nil)
			lcmcommon.RunPodCommand = func(e lcmcommon.ExecConfig) (string, string, error) {
				if e.Command == "ceph status -f json" {
					return test.cephStatus, "", nil
				}
				return "", "", errors.New("command failed")
			}
			status, issue := c.getCephEvents()
			assert.Equal(t, test.issue, issue)
			assert.Equal(t, test.expected, status)
		})
	}
	lcmcommon.RunPodCommand = oldCmdRun
}

func TestCheckReplicasSizing(t *testing.T) {
	tests := []struct {
		name                     string
		cephOsdTreeOutput        string
		cephOsdPoolDetailsOutput string
		cephCrushRuleDumpOutput  string
		expectedIssues           []string
	}{
		{
			name:              "failed to get ceph osd tree",
			cephOsdTreeOutput: "{||}",
			expectedIssues:    []string{"failed to run 'ceph osd tree -f json' command to check replicas sizing"},
		},
		{
			name:                     "failed to get ceph pools details",
			cephOsdTreeOutput:        unitinputs.CephOsdTreeForSizingCheck,
			cephOsdPoolDetailsOutput: "",
			expectedIssues:           []string{"failed to run 'ceph osd pool ls detail -f json' command to check replicas sizing"},
		},
		{
			name:                     "failed to get ceph crush rules dump",
			cephOsdTreeOutput:        unitinputs.CephOsdTreeForSizingCheck,
			cephOsdPoolDetailsOutput: unitinputs.CephPoolsDetails,
			cephCrushRuleDumpOutput:  "{|||}",
			expectedIssues:           []string{"failed to run 'ceph osd crush rule dump -f json' command to check replicas sizing"},
		},
		{
			name:                     "device classes not found",
			cephOsdTreeOutput:        "{}",
			cephOsdPoolDetailsOutput: unitinputs.CephPoolsDetails,
			cephCrushRuleDumpOutput:  unitinputs.CephOsdCrushRuleDump,
			expectedIssues:           []string{"no device classes found in cluster"},
		},
		{
			name:                     "no issues for replica's sizing",
			cephOsdTreeOutput:        unitinputs.CephOsdTreeForSizingCheck,
			cephOsdPoolDetailsOutput: unitinputs.CephPoolsDetails,
			cephCrushRuleDumpOutput:  unitinputs.CephOsdCrushRuleDump,
			expectedIssues:           []string{},
		},
		{
			name: "issues for replica's found #1",
			cephOsdTreeOutput: `{
  "nodes":[
    {"id":-1,"name":"default","type":"root","type_id":11,"children":[-15]},
    {"id":-15,"name":"rack-hdd","type":"rack","type_id":3,"pool_weights":{},"children":[-7,-25,-9, -3]},
    {"id":-9,"name":"de-ds-r6l4djqhmmfn-0-mmk3bbmxtq53-server-xuz6ryuh7qbg","type":"host","type_id":1,"pool_weights":{},"children":[1,0]},
    {"id":0,"device_class":"hdd","name":"osd.0","type":"osd","type_id":0,"crush_weight":0.048797607421875,"depth":3,"pool_weights":{},"exists":1,"status":"down","reweight":1,"primary_affinity":1},
    {"id":1,"device_class":"hdd","name":"osd.1","type":"osd","type_id":0,"crush_weight":0.048797607421875,"depth":3,"pool_weights":{},"exists":1,"status":"up","reweight":0,"primary_affinity":1},
    {"id":-25,"name":"de-ds-r6l4djqhmmfn-1-xastfhatmjqc-server-g7g7co5e467q","type":"host","type_id":1,"pool_weights":{},"children":[7]},
    {"id":7,"device_class":"hdd","name":"osd.7","type":"osd","type_id":0,"crush_weight":0,"depth":3,"pool_weights":{},"exists":1,"status":"up","reweight":1,"primary_affinity":1},
    {"id":-7,"name":"de-ds-r6l4djqhmmfn-2-xupcpjofrkgm-server-5baxrpw2ouy3","type":"host","type_id":1,"pool_weights":{},"children":[2]},
    {"id":2,"device_class":"hdd","name":"osd.2","type":"osd","type_id":0,"crush_weight":0.0731964111328125,"depth":3,"pool_weights":{},"exists":1,"status":"up","reweight":1,"primary_affinity":1},
    {"id":-3,"name":"de-ps-rjshyprsmxpi-0-tc7ms3qx6x6c-server-ptlqq6wjm4oh","type":"host","type_id":1,"pool_weights":{},"children":[6]},
    {"id":6,"device_class":"","name":"osd.6","type":"osd","type_id":0,"crush_weight":0.0731964111328125,"depth":3,"pool_weights":{},"exists":1,"status":"up","reweight":1,"primary_affinity":1}
  ]
			}`,
			cephOsdPoolDetailsOutput: unitinputs.CephPoolsDetails,
			cephCrushRuleDumpOutput:  unitinputs.CephOsdCrushRuleDump,
			expectedIssues: []string{
				"pool 'pool-1' with deviceClass 'hdd' and failureDomain 'host' has targeted to have 3 replicas/chunks, while cluster can provide 1 replica(s)",
				"pool 'pool-2' with deviceClass 'hdd' and failureDomain 'host' has targeted to have 3 replicas/chunks, while cluster can provide 1 replica(s)",
				"pool 'pool-3' with deviceClass 'hdd' and failureDomain 'host' has targeted to have 3 replicas/chunks, while cluster can provide 1 replica(s)",
			},
		},
		{
			name:                     "issues for replica's found #2",
			cephOsdTreeOutput:        unitinputs.CephOsdTreeForSizingCheck,
			cephOsdPoolDetailsOutput: unitinputs.CephPoolsDetails,
			cephCrushRuleDumpOutput: unitinputs.BuildCliOutput(unitinputs.CephCrushRuleDumpTmpl, "osd crush rule dump", map[string]string{
				"pool1_deviceclass":   "default",
				"pool1_failuredomain": "host",
				"pool2_deviceclass":   "default~hdd",
				"pool2_failuredomain": "row",
				"pool3_deviceclass":   "default~ssd",
				"pool3_failuredomain": "rack",
			}),
			expectedIssues: []string{
				"pool 'pool-2' specified to use failure domain 'row', which is not present in cluster",
				"pool 'pool-3' with deviceClass 'ssd' and failureDomain 'rack' has targeted to have 3 replicas/chunks, while cluster can provide 2 replica(s)",
			},
		},
	}
	oldCmdRun := lcmcommon.RunPodCommand
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := fakeCephReconcileConfig(nil, nil)
			faketestclients.FakeReaction(c.api.Kubeclientset.CoreV1(), "list", []string{"pods"}, map[string]runtime.Object{"pods": unitinputs.ToolBoxPodList}, nil)
			lcmcommon.RunPodCommand = func(e lcmcommon.ExecConfig) (string, string, error) {
				switch e.Command {
				case "ceph osd tree -f json":
					return test.cephOsdTreeOutput, "", nil
				case "ceph osd pool ls detail -f json":
					if test.cephOsdPoolDetailsOutput != "" {
						return test.cephOsdPoolDetailsOutput, "", nil
					}
					return "", "", errors.New("failed to run 'ceph osd pool ls detail'")
				case "ceph osd crush rule dump -f json":
					if test.cephCrushRuleDumpOutput != "" {
						return test.cephCrushRuleDumpOutput, "", nil
					}
					return "", "", errors.New("failed to run 'ceph osd crush rule dump -f json'")
				}
				return "", "", errors.New("command failed")
			}

			issues := c.checkReplicasSizing()
			assert.Equal(t, test.expectedIssues, issues)
		})
	}
	lcmcommon.RunPodCommand = oldCmdRun
}

func TestGetRgwInfo(t *testing.T) {
	tests := []struct {
		name             string
		inputResources   map[string]runtime.Object
		healthConfig     healthConfig
		radosAdminOutput string
		expectedStatus   *lcmv1alpha1.RgwInfo
		expectedIssues   []string
	}{
		{
			name:         "cephobjectstore not present",
			healthConfig: getEmtpyHealthConfig(),
		},
		{
			name: "cephobjectstore external has no status",
			healthConfig: func() healthConfig {
				hc := getEmtpyHealthConfig()
				hc.cephCluster = unitinputs.CephClusterExternal.DeepCopy()
				hc.rgwOpts.storeName = "rgw-store-external"
				hc.rgwOpts.external = true
				return hc
			}(),
			expectedStatus: &lcmv1alpha1.RgwInfo{},
			expectedIssues: []string{"cephobjectstore 'rook-ceph/rgw-store-external' endpoint is not found"},
		},
		{
			name: "cephobjectstore external has no secure endpoint",
			healthConfig: func() healthConfig {
				hc := getEmtpyHealthConfig()
				hc.cephCluster = unitinputs.CephClusterExternal.DeepCopy()
				hc.rgwOpts.storeName = "rgw-store-external"
				hc.rgwOpts.external = true
				hc.rgwOpts.externalEndpoint = "http://127.0.0.1:80"
				return hc
			}(),
			expectedStatus: &lcmv1alpha1.RgwInfo{
				PublicEndpoint: "http://127.0.0.1:80",
			},
			expectedIssues: []string{},
		},
		{
			name: "cephobjectstore external has secure endpoint",
			healthConfig: func() healthConfig {
				hc := getEmtpyHealthConfig()
				hc.cephCluster = unitinputs.CephClusterExternal.DeepCopy()
				hc.rgwOpts.storeName = "rgw-store-external"
				hc.rgwOpts.external = true
				hc.rgwOpts.externalEndpoint = "https://127.0.0.1:8443"
				return hc
			}(),
			expectedStatus: &lcmv1alpha1.RgwInfo{
				PublicEndpoint: "https://127.0.0.1:8443",
			},
			expectedIssues: []string{},
		},
		{
			name: "cephobjectstore local, failed to check ingresses and zones",
			healthConfig: func() healthConfig {
				hc := getEmtpyHealthConfig()
				hc.cephCluster = unitinputs.ReefCephClusterReady.DeepCopy()
				hc.rgwOpts.storeName = "rgw-store"
				hc.rgwOpts.desiredRgwDaemons = 2
				return hc
			}(),
			expectedStatus: &lcmv1alpha1.RgwInfo{},
			expectedIssues: []string{
				"failed to check ingresses in 'rook-ceph' namespace", "cephobjectstore 'rook-ceph/rgw-store' endpoint is not found",
			},
		},
		{
			name: "cephobjectstore local, rgw endpoint taken, no multisite",
			inputResources: map[string]runtime.Object{
				"ingresses": &unitinputs.IngressesList,
			},
			healthConfig: func() healthConfig {
				hc := getEmtpyHealthConfig()
				hc.cephCluster = unitinputs.ReefCephClusterReady.DeepCopy()
				hc.rgwOpts.storeName = "rgw-store"
				hc.rgwOpts.desiredRgwDaemons = 2
				return hc
			}(),
			expectedStatus: &lcmv1alpha1.RgwInfo{
				PublicEndpoint: "https://rgw-store.example.com",
			},
			expectedIssues: []string{},
		},
		{
			name: "cephobjectstore local, rgw endpoint taken, check multisite failed",
			inputResources: map[string]runtime.Object{
				"ingresses":       &unitinputs.IngressesList,
				"cephobjectzones": &cephv1.CephObjectZoneList{Items: []cephv1.CephObjectZone{*unitinputs.RgwMultisiteMasterZone1.DeepCopy()}},
			},
			healthConfig: func() healthConfig {
				hc := getEmtpyHealthConfig()
				hc.cephCluster = unitinputs.ReefCephClusterReady.DeepCopy()
				hc.rgwOpts.storeName = "rgw-store"
				hc.rgwOpts.desiredRgwDaemons = 2
				hc.rgwOpts.multisite = true
				return hc
			}(),
			expectedStatus: &lcmv1alpha1.RgwInfo{
				PublicEndpoint:   "https://rgw-store.example.com",
				MultisiteDetails: unitinputs.CephMultisiteStateFailed,
			},
			expectedIssues: []string{"failed to run 'radosgw-admin sync status --rgw-zonegroup=zonegroup1 --rgw-zone=zone1' command to check multisite status for zone 'zone1'"},
		},
		{
			name: "cephobjectstore local, rgw endpoint taken, check multisite ok",
			inputResources: map[string]runtime.Object{
				"ingresses":       &unitinputs.IngressesList,
				"cephobjectzones": &cephv1.CephObjectZoneList{Items: []cephv1.CephObjectZone{*unitinputs.RgwMultisiteMasterZone1.DeepCopy()}},
			},
			healthConfig: func() healthConfig {
				hc := getEmtpyHealthConfig()
				hc.cephCluster = unitinputs.ReefCephClusterReady.DeepCopy()
				hc.rgwOpts.storeName = "rgw-store"
				hc.rgwOpts.desiredRgwDaemons = 2
				hc.rgwOpts.multisite = true
				return hc
			}(),
			radosAdminOutput: unitinputs.RadosgwAdminMasterSyncStatusOk,
			expectedStatus: &lcmv1alpha1.RgwInfo{
				PublicEndpoint:   "https://rgw-store.example.com",
				MultisiteDetails: unitinputs.CephMultisiteStateOk,
			},
			expectedIssues: []string{},
		},
	}
	oldCmdFunc := lcmcommon.RunPodCommand
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := fakeCephReconcileConfig(&test.healthConfig, nil)
			faketestclients.FakeReaction(c.api.Kubeclientset.CoreV1(), "list", []string{"pods"}, map[string]runtime.Object{"pods": unitinputs.ToolBoxPodList}, nil)
			faketestclients.FakeReaction(c.api.Kubeclientset.NetworkingV1(), "list", []string{"ingresses"}, test.inputResources, nil)
			faketestclients.FakeReaction(c.api.Rookclientset, "list", []string{"cephobjectzones"}, test.inputResources, nil)

			lcmcommon.RunPodCommand = func(e lcmcommon.ExecConfig) (string, string, error) {
				if strings.HasPrefix(e.Command, "radosgw-admin sync status") {
					if test.radosAdminOutput != "" {
						return test.radosAdminOutput, "", nil
					}
				}
				return "", "", errors.Errorf("command failed")
			}

			status, issues := c.getRgwInfo()
			assert.Equal(t, test.expectedStatus, status)
			assert.Equal(t, test.expectedIssues, issues)

			faketestclients.CleanupFakeClientReactions(c.api.Kubeclientset.CoreV1())
			faketestclients.CleanupFakeClientReactions(c.api.Kubeclientset.NetworkingV1())
			faketestclients.CleanupFakeClientReactions(c.api.Rookclientset)
		})
	}
	lcmcommon.RunPodCommand = oldCmdFunc
}

func TestGetRgwPublicEndpoint(t *testing.T) {
	baseConfig := getEmtpyHealthConfig()
	baseConfig.rgwOpts.storeName = "rgw-store"
	tests := []struct {
		name              string
		inputResources    map[string]runtime.Object
		customAccessLabel string
		expectedEndpoint  string
		expectedIssue     string
	}{
		{
			name:           "failed to check ingresses",
			inputResources: map[string]runtime.Object{},
			expectedIssue:  "failed to check ingresses in 'rook-ceph' namespace",
		},
		{
			name: "rgw endpoint from ingress",
			inputResources: map[string]runtime.Object{
				"ingresses": &unitinputs.IngressesList,
			},
			expectedEndpoint: "https://rgw-store.example.com",
		},
		{
			name: "ingress has no rules",
			inputResources: map[string]runtime.Object{
				"ingresses": func() *networkingv1.IngressList {
					list := unitinputs.IngressesList.DeepCopy()
					list.Items[0].Spec.Rules = nil
					return list
				}(),
			},
		},
		{
			name: "ingress has no expected rgw backend",
			inputResources: map[string]runtime.Object{
				"ingresses": func() *networkingv1.IngressList {
					list := unitinputs.IngressesList.DeepCopy()
					list.Items[0].Spec.Rules[0].HTTP = nil
					return list
				}(),
			},
			expectedIssue: "can't determine Ceph RGW public endpoint for ingress rook-ceph/rook-ceph-rgw-rgw-store-ingress, backend 'rook-ceph-rgw-rgw-store' is not found in ingress rules",
		},
		{
			name: "no ingresses, failed to check services",
			inputResources: map[string]runtime.Object{
				"ingresses": &unitinputs.IngressesListEmpty,
			},
			expectedIssue: "failed to check services in 'rook-ceph' namespace",
		},
		{
			name: "no ingresses, service found, rgw endpoint taken",
			inputResources: map[string]runtime.Object{
				"ingresses": &unitinputs.IngressesListEmpty,
				"services":  &unitinputs.ServicesListRgwExternal,
			},
			expectedEndpoint: "https://192.168.100.150:443",
		},
		{
			name: "no ingresses, service found, but not a LoadBalancer",
			inputResources: map[string]runtime.Object{
				"ingresses": &unitinputs.IngressesListEmpty,
				"services": func() *corev1.ServiceList {
					list := unitinputs.ServicesListRgwExternal.DeepCopy()
					list.Items[0].Spec.Type = "NodePort"
					return list
				}(),
			},
		},
		{
			name: "no ingresses, service found, but no ip",
			inputResources: map[string]runtime.Object{
				"ingresses": &unitinputs.IngressesListEmpty,
				"services": func() *corev1.ServiceList {
					list := unitinputs.ServicesListRgwExternal.DeepCopy()
					list.Items[0].Status.LoadBalancer.Ingress = nil
					return list
				}(),
			},
			expectedIssue: "external service rook-ceph/rgw-store has no IP addresses available, can't determine Ceph RGW public endpoint",
		},
		{
			name: "no ingresses, service found, but no https",
			inputResources: map[string]runtime.Object{
				"ingresses": &unitinputs.IngressesListEmpty,
				"services": func() *corev1.ServiceList {
					list := unitinputs.ServicesListRgwExternal.DeepCopy()
					list.Items[0].Spec.Ports[1].Name = "custom"
					return list
				}(),
			},
			expectedEndpoint: "http://192.168.100.150:80",
		},
		{
			name: "no ingresses, no services, give up",
			inputResources: map[string]runtime.Object{
				"ingresses": &unitinputs.IngressesListEmpty,
				"services":  &unitinputs.ServicesListEmpty,
			},
		},
		{
			name: "custom selector for rgw public access",
			inputResources: map[string]runtime.Object{
				"ingresses": &unitinputs.IngressesListEmpty,
				"services": func() *corev1.ServiceList {
					list := unitinputs.ServicesListRgwExternal.DeepCopy()
					delete(list.Items[0].Labels, "external_access")
					list.Items[0].Labels["custom_label"] = "custom_value"
					return list
				}(),
			},
			customAccessLabel: "custom_label=custom_value",
			expectedEndpoint:  "https://192.168.100.150:443",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			lcmConfigData := map[string]string{}
			if test.customAccessLabel != "" {
				lcmConfigData["RGW_PUBLIC_ACCESS_SERVICE_SELECTOR"] = test.customAccessLabel
			}
			c := fakeCephReconcileConfig(&baseConfig, lcmConfigData)
			faketestclients.FakeReaction(c.api.Kubeclientset.CoreV1(), "list", []string{"services"}, test.inputResources, nil)
			faketestclients.FakeReaction(c.api.Kubeclientset.NetworkingV1(), "list", []string{"ingresses"}, test.inputResources, nil)

			endpoint, issue := c.getRgwPublicEndpoint()
			assert.Equal(t, test.expectedEndpoint, endpoint)
			assert.Equal(t, test.expectedIssue, issue)

			faketestclients.CleanupFakeClientReactions(c.api.Kubeclientset.CoreV1())
			faketestclients.CleanupFakeClientReactions(c.api.Kubeclientset.NetworkingV1())
		})
	}
}

func TestGetMultisiteSyncStatus(t *testing.T) {
	emptyIssues := make([]string, 0)
	tests := []struct {
		name           string
		cmdOutput      string
		inputResources map[string]runtime.Object
		expectedStatus *lcmv1alpha1.MultisiteState
		expectedIssues []string
	}{
		{
			name: "failed to list cephobjectzones",
			expectedStatus: &lcmv1alpha1.MultisiteState{
				MetadataSyncState: lcmv1alpha1.MultiSiteFailed,
				DataSyncState:     lcmv1alpha1.MultiSiteFailed,
				Messages:          []string{"failed to list cephobjectzones in 'rook-ceph' namespace"},
			},
			inputResources: map[string]runtime.Object{},
			expectedIssues: []string{"failed to list cephobjectzones in 'rook-ceph' namespace"},
		},
		{
			name:           "failed to run sync status cmd",
			expectedStatus: unitinputs.CephMultisiteStateFailed,
			inputResources: map[string]runtime.Object{
				"cephobjectzones": &cephv1.CephObjectZoneList{
					Items: []cephv1.CephObjectZone{*unitinputs.RgwMultisiteMasterZone1.DeepCopy()},
				},
			},
			expectedIssues: []string{"failed to run 'radosgw-admin sync status --rgw-zonegroup=zonegroup1 --rgw-zone=zone1' command to check multisite status for zone 'zone1'"},
		},
		{
			name:      "master zone - sync is ok",
			cmdOutput: unitinputs.RadosgwAdminMasterSyncStatusOk,
			inputResources: map[string]runtime.Object{
				"cephobjectzones": &cephv1.CephObjectZoneList{
					Items: []cephv1.CephObjectZone{*unitinputs.RgwMultisiteMasterZone1.DeepCopy()},
				},
			},
			expectedStatus: unitinputs.CephMultisiteStateOk,
			expectedIssues: emptyIssues,
		},
		{
			name: "master zone - no secondary data zone yet",
			cmdOutput: `
          realm a46a61a7-46c0-41dd-8f62-9f989b9de803 (openstack-store)
      zonegroup 5c6c92c1-632c-4db0-8aa9-8dcbea5d87ec (openstack-store)
           zone 4abcf593-157b-46bb-8209-0f8f7f5a7e8e (openstack-store)
   current time 2024-04-18T13:27:48Z
zonegroup features enabled: resharding
                   disabled: compress-encrypted
  metadata sync no sync (zone is master)`,
			inputResources: map[string]runtime.Object{
				"cephobjectzones": &cephv1.CephObjectZoneList{
					Items: []cephv1.CephObjectZone{*unitinputs.RgwMultisiteMasterZone1.DeepCopy()},
				},
			},
			expectedStatus: unitinputs.CephMultisiteStateOk,
			expectedIssues: emptyIssues,
		},
		{
			name: "master zone - failed to get data sync info",
			cmdOutput: `
          realm a46a61a7-46c0-41dd-8f62-9f989b9de803 (openstack-store)
      zonegroup 5c6c92c1-632c-4db0-8aa9-8dcbea5d87ec (openstack-store)
           zone 4abcf593-157b-46bb-8209-0f8f7f5a7e8e (openstack-store)
   current time 2024-04-18T13:15:16Z
zonegroup features enabled: resharding
                   disabled: compress-encrypted
  metadata sync no sync (zone is master)
2024-04-18T13:15:16.957+0000 7ffb3d16f880  0 ERROR: failed to fetch datalog info
      data sync source: 362d9d90-1151-41a0-80aa-e8aa6d036730 (openstack-store-backup)
                        failed to retrieve sync info: (5) Input/output error`,
			inputResources: map[string]runtime.Object{
				"cephobjectzones": &cephv1.CephObjectZoneList{
					Items: []cephv1.CephObjectZone{*unitinputs.RgwMultisiteMasterZone1.DeepCopy()},
				},
			},
			expectedStatus: &lcmv1alpha1.MultisiteState{
				MetadataSyncState: lcmv1alpha1.MultiSiteSyncing,
				DataSyncState:     lcmv1alpha1.MultiSiteFailed,
				MasterZone:        true,
				Messages:          []string{"failed to fetch data info"},
			},
			expectedIssues: emptyIssues,
		},
		{
			name: "master zone - secondary is behind master",
			cmdOutput: `
          realm a46a61a7-46c0-41dd-8f62-9f989b9de803 (openstack-store)
      zonegroup 5c6c92c1-632c-4db0-8aa9-8dcbea5d87ec (openstack-store)
           zone 4abcf593-157b-46bb-8209-0f8f7f5a7e8e (openstack-store)
   current time 2024-04-18T13:15:16Z
zonegroup features enabled: resharding
                   disabled: compress-encrypted
  metadata sync no sync (zone is master)
      data sync source: 362d9d90-1151-41a0-80aa-e8aa6d036730 (openstack-store-backup)
                        full sync: 0/128 shards
                        incremental sync: 128/128 shards
                        data is behind on 1 shards
                        behind shards: [71]
                        oldest incremental change not applied: 2024-04-18T13:09:04.446175+0000 [71]`,
			inputResources: map[string]runtime.Object{
				"cephobjectzones": &cephv1.CephObjectZoneList{
					Items: []cephv1.CephObjectZone{*unitinputs.RgwMultisiteMasterZone1.DeepCopy()},
				},
			},
			expectedStatus: &lcmv1alpha1.MultisiteState{
				MetadataSyncState: lcmv1alpha1.MultiSiteSyncing,
				DataSyncState:     lcmv1alpha1.MultiSiteOutOfSync,
				MasterZone:        true,
			},
			expectedIssues: emptyIssues,
		},
		{
			name:      "secondary zone - sync is ok",
			cmdOutput: unitinputs.RadosgwAdminSecondarySyncStatusOk,
			expectedStatus: &lcmv1alpha1.MultisiteState{
				MetadataSyncState: lcmv1alpha1.MultiSiteSyncing,
				DataSyncState:     lcmv1alpha1.MultiSiteSyncing,
			},
			inputResources: map[string]runtime.Object{
				"cephobjectzones": &cephv1.CephObjectZoneList{
					Items: []cephv1.CephObjectZone{*unitinputs.RgwMultisiteSecondaryZone1.DeepCopy()},
				},
			},
			expectedIssues: emptyIssues,
		},
		{
			name: "secondary zone - metadata and data are behind",
			cmdOutput: `
zonegroup f54f9b22-b4b6-4a0e-9211-fa6ac1693f49 (us)
     zone adce11c9-b8ed-4a90-8bc5-3fc029ff0816 (us-2)
    metadata sync syncing
          full sync: 0/64 shards
          incremental sync: 64/64 shards
          metadata is behind on 1 shards
          oldest incremental change not applied: 2017-03-22 10:20:00.0.881361s
data sync source: 341c2d81-4574-4d08-ab0f-5a2a7b168028 (us-1)
                  syncing
                  full sync: 0/128 shards
                  incremental sync: 128/128 shards
                  data is behind on 1 shards
                  behind shards: [71]
                  oldest incremental change not applied: 2024-04-18T13:09:04.446175+0000 [71]
          source: 3b5d1a3f-3f27-4e4a-8f34-6072d4bb1275 (us-3)
                  syncing
                  full sync: 0/128 shards
                  incremental sync: 128/128 shards
                  data is caught up with source`,
			inputResources: map[string]runtime.Object{
				"cephobjectzones": &cephv1.CephObjectZoneList{
					Items: []cephv1.CephObjectZone{*unitinputs.RgwMultisiteSecondaryZone1.DeepCopy()},
				},
			},
			expectedStatus: &lcmv1alpha1.MultisiteState{
				MetadataSyncState: lcmv1alpha1.MultiSiteOutOfSync,
				DataSyncState:     lcmv1alpha1.MultiSiteOutOfSync,
				Messages:          []string{"metadata is behind master zone", "data is behind master zone"},
			},
			expectedIssues: []string{"metadata is behind master zone", "data is behind master zone"},
		},
		{
			name: "secondary zone - failed to get metadata and data sync info",
			cmdOutput: `
          realm a46a61a7-46c0-41dd-8f62-9f989b9de803 (openstack-store)
      zonegroup 5c6c92c1-632c-4db0-8aa9-8dcbea5d87ec (openstack-store)
           zone 362d9d90-1151-41a0-80aa-e8aa6d036730 (openstack-store-backup)
   current time 2024-04-18T13:12:53Z
zonegroup features enabled: resharding
                   disabled: compress-encrypted
2024-04-18T13:12:53.953+0000 7fbd8c6ba880  0 ERROR: failed to fetch mdlog info
  metadata sync syncing
                full sync: 0/64 shards
                failed to fetch master sync status: (5) Input/output error
2024-04-18T13:12:53.957+0000 7fbd8c6ba880  0 ERROR: failed to fetch datalog info
      data sync source: 4abcf593-157b-46bb-8209-0f8f7f5a7e8e (openstack-store)
                        failed to retrieve sync info: (5) Input/output error`,
			inputResources: map[string]runtime.Object{
				"cephobjectzones": &cephv1.CephObjectZoneList{
					Items: []cephv1.CephObjectZone{*unitinputs.RgwMultisiteSecondaryZone1.DeepCopy()},
				},
			},
			expectedStatus: &lcmv1alpha1.MultisiteState{
				MetadataSyncState: lcmv1alpha1.MultiSiteFailed,
				DataSyncState:     lcmv1alpha1.MultiSiteFailed,
				Messages:          []string{"failed to fetch metadata info", "failed to fetch data info"},
			},
			expectedIssues: []string{"failed to fetch metadata info", "failed to fetch data info"},
		},
		{
			name: "secondary zone - no data sync info",
			cmdOutput: `
          realm a46a61a7-46c0-41dd-8f62-9f989b9de803 (openstack-store)
      zonegroup 5c6c92c1-632c-4db0-8aa9-8dcbea5d87ec (openstack-store)
           zone 362d9d90-1151-41a0-80aa-e8aa6d036730 (openstack-store-backup)
   current time 2024-04-18T13:09:11Z
zonegroup features enabled: resharding
                   disabled: compress-encrypted
  metadata sync syncing
                full sync: 0/64 shards
                incremental sync: 64/64 shards
                metadata is caught up with master`,
			inputResources: map[string]runtime.Object{
				"cephobjectzones": &cephv1.CephObjectZoneList{
					Items: []cephv1.CephObjectZone{*unitinputs.RgwMultisiteSecondaryZone1.DeepCopy()},
				},
			},
			expectedStatus: &lcmv1alpha1.MultisiteState{
				MetadataSyncState: lcmv1alpha1.MultiSiteSyncing,
				DataSyncState:     lcmv1alpha1.MultiSiteFailed,
				Messages:          []string{"data sync info is not present"},
			},
			expectedIssues: []string{"data sync info is not present"},
		},
		{
			name: "secondary zone - unknown metadata and data sync state",
			cmdOutput: `
          realm a46a61a7-46c0-41dd-8f62-9f989b9de803 (openstack-store)
      zonegroup 5c6c92c1-632c-4db0-8aa9-8dcbea5d87ec (openstack-store)
           zone 362d9d90-1151-41a0-80aa-e8aa6d036730 (openstack-store-backup)
   current time 2024-04-18T13:09:11Z
zonegroup features enabled: resharding
                   disabled: compress-encrypted
    metadata sync syncing
                full sync: 0/64 shards
      data sync source: 4abcf593-157b-46bb-8209-0f8f7f5a7e8e (openstack-store)`,
			inputResources: map[string]runtime.Object{
				"cephobjectzones": &cephv1.CephObjectZoneList{
					Items: []cephv1.CephObjectZone{*unitinputs.RgwMultisiteSecondaryZone1.DeepCopy()},
				},
			},
			expectedStatus: &lcmv1alpha1.MultisiteState{
				MetadataSyncState: lcmv1alpha1.MultiSiteFailed,
				DataSyncState:     lcmv1alpha1.MultiSiteFailed,
				Messages:          []string{"unknown metadata sync state", "unknown data sync state"},
			},
			expectedIssues: []string{"unknown metadata sync state", "unknown data sync state"},
		},
	}
	oldCmdFunc := lcmcommon.RunPodCommand
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := fakeCephReconcileConfig(nil, nil)
			faketestclients.FakeReaction(c.api.Kubeclientset.CoreV1(), "list", []string{"pods"}, map[string]runtime.Object{"pods": unitinputs.ToolBoxPodList}, nil)
			faketestclients.FakeReaction(c.api.Rookclientset, "list", []string{"cephobjectzones"}, test.inputResources, nil)
			lcmcommon.RunPodCommand = func(e lcmcommon.ExecConfig) (string, string, error) {
				if strings.HasPrefix(e.Command, "radosgw-admin sync status") {
					if test.cmdOutput != "" {
						return test.cmdOutput, "", nil
					}
				}
				return "", "", errors.Errorf("command failed")
			}

			status, issues := c.getMultisiteSyncStatus()
			assert.Equal(t, test.expectedStatus, status)
			assert.Equal(t, test.expectedIssues, issues)
			faketestclients.CleanupFakeClientReactions(c.api.Rookclientset)
		})
	}
	lcmcommon.RunPodCommand = oldCmdFunc
}
