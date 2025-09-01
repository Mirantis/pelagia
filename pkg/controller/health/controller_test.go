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
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	lcmv1alpha1 "github.com/Mirantis/pelagia/pkg/apis/ceph.pelagia.lcm/v1alpha1"
	lcmcommon "github.com/Mirantis/pelagia/pkg/common"
	lcmconfig "github.com/Mirantis/pelagia/pkg/controller/config"
	faketestclients "github.com/Mirantis/pelagia/test/unit/clients"
	unitinputs "github.com/Mirantis/pelagia/test/unit/inputs"
	faketestscheme "github.com/Mirantis/pelagia/test/unit/scheme"
)

func FakeReconciler() *ReconcileCephDeploymentHealth {
	return &ReconcileCephDeploymentHealth{
		Config:         &rest.Config{},
		Client:         faketestclients.GetClient(nil),
		Lcmclientset:   faketestclients.GetFakeLcmclient(),
		Kubeclientset:  faketestclients.GetFakeKubeclient(),
		Rookclientset:  faketestclients.GetFakeRookclient(),
		Claimclientset: faketestclients.GetFakeClaimclient(),
		Scheme:         faketestscheme.Scheme,
	}
}

func getEmtpyHealthConfig() healthConfig {
	return healthConfig{
		name:        unitinputs.LcmObjectMeta.Name,
		namespace:   unitinputs.LcmObjectMeta.Namespace,
		cephCluster: nil,
		rgwOpts:     rgwOpts{},
		sharedFilesystemOpts: sharedFilesystemOpts{
			mdsDaemonsDesired: map[string]map[string]int{},
		},
	}
}

func fakeCephReconcileConfig(hconfig *healthConfig, lcmConfigData map[string]string) *cephDeploymentHealthConfig {
	lcmConfig := lcmconfig.ReadConfiguration(log.With().Str(lcmcommon.LoggerObjectField, "configmap").Logger(), lcmConfigData)
	sublog := log.With().Str(lcmcommon.LoggerObjectField, "cephdeploymenthealth 'lcm-namespace/cephcluster'").Logger().Level(lcmConfig.HealthParams.LogLevel)
	hc := getEmtpyHealthConfig()
	if hconfig != nil {
		hc = *hconfig
	}
	return &cephDeploymentHealthConfig{
		context:      context.TODO(),
		api:          FakeReconciler(),
		log:          &sublog,
		lcmConfig:    &lcmConfig,
		healthConfig: hc,
	}
}

var rookListResources = []string{"cephblockpools", "cephclients", "cephfilesystems", "cephobjectstores", "cephobjectstoreusers", "cephobjectrealms", "cephobjectzonegroups", "cephobjectzones"}
var rookGetResources = []string{"cephclusters"}
var claimListResources = []string{"objectbucketclaims"}

func TestHealthReconcile(t *testing.T) {
	req := reconcile.Request{NamespacedName: types.NamespacedName{Namespace: unitinputs.LcmObjectMeta.Namespace, Name: unitinputs.LcmObjectMeta.Name}}
	resInterval := reconcile.Result{RequeueAfter: requeueAfterInterval}
	nodesList := unitinputs.GetNodesList(
		[]unitinputs.NodeAttrs{{Name: "node-1", Labeled: true}, {Name: "node-2", Labeled: true}})

	tests := []struct {
		name             string
		inputResources   map[string]runtime.Object
		cephCliOutput    map[string]string
		diskDaemonReport map[string]string
		apiError         string
		expectedStatus   lcmv1alpha1.CephDeploymentHealthStatus
		expectedError    string
		expectedResult   reconcile.Result
	}{
		{
			name: "cephdeploymenthealth not found",
			inputResources: map[string]runtime.Object{
				"cephdeploymenthealths": &lcmv1alpha1.CephDeploymentHealthList{},
			},
			expectedResult: reconcile.Result{},
		},
		{
			name: "failed to get cephdeploymenthealth",
			inputResources: map[string]runtime.Object{
				"cephdeploymenthealths": &lcmv1alpha1.CephDeploymentHealthList{},
			},
			apiError:       "get",
			expectedError:  "cant get cephdeploymenthealth object",
			expectedResult: resInterval,
		},
		{
			name: "cephdeploymenthealth failed to update status",
			inputResources: map[string]runtime.Object{
				"cephdeploymenthealths": &lcmv1alpha1.CephDeploymentHealthList{Items: []lcmv1alpha1.CephDeploymentHealth{unitinputs.CephDeploymentHealth}},
				"daemonsets":            unitinputs.DaemonSetListReady,
				"deployments":           unitinputs.DeploymentList,
				"configmaps":            unitinputs.ConfigMapList,
				"cephclusters":          &unitinputs.CephClusterListReady,
				"cephblockpools":        &unitinputs.CephBlockPoolListEmpty,
				"cephclients":           &unitinputs.CephClientListEmpty,
				"cephobjectstores":      &unitinputs.CephObjectStoreListEmpty,
				"cephobjectstoreusers":  &unitinputs.CephObjectStoreUserListEmpty,
				"objectbucketclaims":    &unitinputs.ObjectBucketClaimListEmpty,
				"cephobjectrealms":      &unitinputs.CephObjectRealmListEmpty,
				"cephobjectzonegroups":  &unitinputs.CephObjectZoneGroupListEmpty,
				"cephobjectzones":       &unitinputs.CephObjectZoneListEmpty,
				"cephfilesystems":       &unitinputs.CephFilesystemListEmpty,
			},
			cephCliOutput: map[string]string{
				"ceph df -f json":                  unitinputs.CephDfBase,
				"ceph status -f json":              unitinputs.CephStatusBaseHealthy,
				"ceph mgr dump -f json":            unitinputs.CephMgrDumpBaseHealthy,
				"ceph osd tree -f json":            unitinputs.CephOsdTreeForSizingCheck,
				"ceph osd crush rule dump -f json": unitinputs.CephOsdCrushRuleDump,
				"ceph osd pool ls detail -f json":  unitinputs.CephPoolsDetails,
				"ceph osd metadata -f json":        unitinputs.CephOsdMetadataOutput,
				"ceph osd info -f json":            unitinputs.CephOsdInfoOutput,
			},
			diskDaemonReport: map[string]string{
				"node-1": unitinputs.CephDiskDaemonDiskReportStringNode1,
				"node-2": unitinputs.CephDiskDaemonDiskReportStringNode2,
			},
			apiError:       "status",
			expectedResult: resInterval,
		},
		{
			name: "reconcile cephdeploymenthealth has issues, status updated",
			inputResources: map[string]runtime.Object{
				"cephdeploymenthealths": &lcmv1alpha1.CephDeploymentHealthList{Items: []lcmv1alpha1.CephDeploymentHealth{unitinputs.CephDeploymentHealth}},
				"daemonsets":            unitinputs.DaemonSetListNotReady,
				"deployments":           unitinputs.DeploymentList,
				"configmaps":            unitinputs.ConfigMapList,
				"cephclusters":          &unitinputs.CephClusterListHealthIssues,
				"cephblockpools":        &unitinputs.CephBlockPoolListEmpty,
				"cephclients":           &unitinputs.CephClientListEmpty,
				"cephobjectstores":      &unitinputs.CephObjectStoreListEmpty,
				"cephobjectstoreusers":  &unitinputs.CephObjectStoreUserListEmpty,
				"objectbucketclaims":    &unitinputs.ObjectBucketClaimListEmpty,
				"cephobjectrealms":      &unitinputs.CephObjectRealmListEmpty,
				"cephobjectzonegroups":  &unitinputs.CephObjectZoneGroupListEmpty,
				"cephobjectzones":       &unitinputs.CephObjectZoneListEmpty,
				"cephfilesystems":       &unitinputs.CephFilesystemListEmpty,
			},
			cephCliOutput: map[string]string{
				"ceph df -f json":           unitinputs.CephDfBase,
				"ceph status -f json":       unitinputs.CephStatusBaseUnhealthy,
				"ceph mgr dump -f json":     unitinputs.CephMgrDumpBaseUnhealthy,
				"ceph osd metadata -f json": unitinputs.CephOsdMetadataOutput,
				"ceph osd info -f json":     unitinputs.CephOsdInfoOutput,
			},
			expectedStatus: func() lcmv1alpha1.CephDeploymentHealthStatus {
				status := unitinputs.CephDeploymentHealthStatusNotOk.Status.DeepCopy()
				status.LastHealthCheck = "time-3"
				status.LastHealthUpdate = "time-3"
				return *status
			}(),
			expectedResult: resInterval,
		},
		{
			name: "reconcile cephdeploymenthealth no issues, status updated",
			inputResources: map[string]runtime.Object{
				"cephdeploymenthealths": &lcmv1alpha1.CephDeploymentHealthList{Items: []lcmv1alpha1.CephDeploymentHealth{unitinputs.CephDeploymentHealth}},
				"daemonsets":            unitinputs.DaemonSetListReady,
				"deployments":           unitinputs.DeploymentList,
				"configmaps":            unitinputs.ConfigMapList,
				"cephclusters":          &unitinputs.CephClusterListReady,
				"cephblockpools":        &unitinputs.CephBlockPoolListEmpty,
				"cephclients":           &unitinputs.CephClientListEmpty,
				"cephobjectstores":      &unitinputs.CephObjectStoreListEmpty,
				"cephobjectstoreusers":  &unitinputs.CephObjectStoreUserListEmpty,
				"objectbucketclaims":    &unitinputs.ObjectBucketClaimListEmpty,
				"cephobjectrealms":      &unitinputs.CephObjectRealmListEmpty,
				"cephobjectzonegroups":  &unitinputs.CephObjectZoneGroupListEmpty,
				"cephobjectzones":       &unitinputs.CephObjectZoneListEmpty,
				"cephfilesystems":       &unitinputs.CephFilesystemListEmpty,
				"nodes":                 &nodesList,
			},
			cephCliOutput: map[string]string{
				"ceph df -f json":                  unitinputs.CephDfBase,
				"ceph status -f json":              unitinputs.CephStatusBaseHealthy,
				"ceph mgr dump -f json":            unitinputs.CephMgrDumpBaseHealthy,
				"ceph osd tree -f json":            unitinputs.CephOsdTreeForSizingCheck,
				"ceph osd crush rule dump -f json": unitinputs.CephOsdCrushRuleDump,
				"ceph osd pool ls detail -f json":  unitinputs.CephPoolsDetails,
				"ceph osd metadata -f json":        unitinputs.CephOsdMetadataOutput,
				"ceph osd info -f json":            unitinputs.CephOsdInfoOutput,
			},
			diskDaemonReport: map[string]string{
				"node-1": unitinputs.CephDiskDaemonDiskReportStringNode1,
				"node-2": unitinputs.CephDiskDaemonDiskReportStringNode2,
			},
			expectedStatus: func() lcmv1alpha1.CephDeploymentHealthStatus {
				status := unitinputs.CephDeploymentHealthStatusOk.Status.DeepCopy()
				status.LastHealthCheck = "time-4"
				status.LastHealthUpdate = "time-4"
				return *status
			}(),
			expectedResult: resInterval,
		},
		{
			name: "reconcile cephdeploymenthealth no issues, no status update",
			inputResources: map[string]runtime.Object{
				"cephdeploymenthealths": &lcmv1alpha1.CephDeploymentHealthList{Items: []lcmv1alpha1.CephDeploymentHealth{*unitinputs.CephDeploymentHealthStatusOk.DeepCopy()}},
				"daemonsets":            unitinputs.DaemonSetListReady,
				"deployments":           unitinputs.DeploymentList,
				"configmaps":            unitinputs.ConfigMapList,
				"cephclusters":          &unitinputs.CephClusterListReady,
				"cephblockpools":        &unitinputs.CephBlockPoolListEmpty,
				"cephclients":           &unitinputs.CephClientListEmpty,
				"cephobjectstores":      &unitinputs.CephObjectStoreListEmpty,
				"cephobjectstoreusers":  &unitinputs.CephObjectStoreUserListEmpty,
				"objectbucketclaims":    &unitinputs.ObjectBucketClaimListEmpty,
				"cephobjectrealms":      &unitinputs.CephObjectRealmListEmpty,
				"cephobjectzonegroups":  &unitinputs.CephObjectZoneGroupListEmpty,
				"cephobjectzones":       &unitinputs.CephObjectZoneListEmpty,
				"cephfilesystems":       &unitinputs.CephFilesystemListEmpty,
				"nodes":                 &nodesList,
			},
			cephCliOutput: map[string]string{
				"ceph df -f json":                  unitinputs.CephDfBase,
				"ceph status -f json":              unitinputs.CephStatusBaseHealthy,
				"ceph mgr dump -f json":            unitinputs.CephMgrDumpBaseHealthy,
				"ceph osd tree -f json":            unitinputs.CephOsdTreeForSizingCheck,
				"ceph osd crush rule dump -f json": unitinputs.CephOsdCrushRuleDump,
				"ceph osd pool ls detail -f json":  unitinputs.CephPoolsDetails,
				"ceph osd metadata -f json":        unitinputs.CephOsdMetadataOutput,
				"ceph osd info -f json":            unitinputs.CephOsdInfoOutput,
			},
			diskDaemonReport: map[string]string{
				"node-1": unitinputs.CephDiskDaemonDiskReportStringNode1,
				"node-2": unitinputs.CephDiskDaemonDiskReportStringNode2,
			},
			expectedStatus: func() lcmv1alpha1.CephDeploymentHealthStatus {
				status := unitinputs.CephDeploymentHealthStatusOk.Status.DeepCopy()
				status.LastHealthCheck = "time-5"
				return *status
			}(),
			expectedResult: resInterval,
		},
	}
	baseFunc := lcmcommon.GetCurrentTimeString
	oldRunFunc := lcmcommon.RunPodCommandWithValidation
	for idx, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			lcmcommon.GetCurrentTimeString = func() string {
				return fmt.Sprintf("time-%d", idx)
			}

			r := FakeReconciler()
			checkStatus := false
			if test.inputResources["cephdeploymenthealths"] != nil {
				list := test.inputResources["cephdeploymenthealths"].(*lcmv1alpha1.CephDeploymentHealthList)
				if len(list.Items) == 1 && test.apiError != "status" {
					checkStatus = true
					r.Client = faketestclients.GetClient(faketestclients.GetClientBuilder().WithStatusSubresource(&list.Items[0]).WithObjects(&list.Items[0]))
				} else {
					r.Client = faketestclients.GetClient(nil)
				}
			}

			apiErrors := map[string]error{}
			if test.apiError == "get" {
				apiErrors["get-cephdeploymenthealths"] = errors.New("cant get cephdeploymenthealth object")
			}
			faketestclients.FakeReaction(r.Kubeclientset.NetworkingV1(), "list", []string{"ingresses"}, test.inputResources, nil)
			faketestclients.FakeReaction(r.Rookclientset, "list", rookListResources, test.inputResources, nil)
			faketestclients.FakeReaction(r.Claimclientset, "list", claimListResources, test.inputResources, nil)
			faketestclients.FakeReaction(r.Lcmclientset, "get", []string{"cephdeploymenthealths"}, test.inputResources, apiErrors)
			faketestclients.FakeReaction(r.Rookclientset, "get", rookGetResources, test.inputResources, nil)
			faketestclients.FakeReaction(r.Kubeclientset.CoreV1(), "get", []string{"configmaps"}, test.inputResources, nil)
			faketestclients.FakeReaction(r.Kubeclientset.AppsV1(), "get", []string{"deployments", "daemonsets"}, test.inputResources, nil)
			faketestclients.FakeReaction(r.Kubeclientset.CoreV1(), "get", []string{"nodes"}, test.inputResources, nil)

			lcmcommon.RunPodCommandWithValidation = func(e lcmcommon.ExecConfig) (string, string, error) {
				if e.Command == "pelagia-disk-daemon --full-report --port 9999" {
					if test.diskDaemonReport != nil {
						return test.diskDaemonReport[e.Nodename], "", nil
					}
				} else if output, ok := test.cephCliOutput[e.Command]; ok {
					return output, "", nil
				}
				return "", "", errors.New("failed command")
			}

			ctx := context.TODO()
			res, err := r.Reconcile(ctx, req)
			assert.Equal(t, test.expectedResult, res)
			if test.expectedError != "" {
				assert.NotNil(t, err)
				assert.Equal(t, test.expectedError, err.Error())
			} else {
				assert.Nil(t, err)
			}
			// check updated status
			cephDeploymentHealth := &lcmv1alpha1.CephDeploymentHealth{}
			err = r.Client.Get(ctx, req.NamespacedName, cephDeploymentHealth)
			if checkStatus {
				assert.Nil(t, err)
				assert.Equal(t, test.expectedStatus, cephDeploymentHealth.Status)
			} else {
				assert.NotNil(t, err)
				assert.Equal(t, "cephdeploymenthealths.lcm.mirantis.com \"cephcluster\" not found", err.Error())
			}
			// clean reactions
			faketestclients.CleanupFakeClientReactions(r.Lcmclientset)
			faketestclients.CleanupFakeClientReactions(r.Kubeclientset.CoreV1())
			faketestclients.CleanupFakeClientReactions(r.Kubeclientset.AppsV1())
			faketestclients.CleanupFakeClientReactions(r.Kubeclientset.NetworkingV1())
			faketestclients.CleanupFakeClientReactions(r.Rookclientset)
			faketestclients.CleanupFakeClientReactions(r.Claimclientset)
		})
	}
	lcmcommon.GetCurrentTimeString = baseFunc
	lcmcommon.RunPodCommandWithValidation = oldRunFunc
}

func TestHealthAndConfigReconcile(t *testing.T) {
	oldVal := lcmconfig.ParamsToControl
	lcmconfig.ParamsToControl = lcmconfig.ControlParamsHealth
	configRequest := reconcile.Request{NamespacedName: types.NamespacedName{Namespace: unitinputs.LcmObjectMeta.Namespace, Name: "pelagia-lcmconfig"}}
	disableAllChecks := []string{cephDaemonsCheck, cephCSIPluginDaemonsCheck, usageDetailsCheck, cephEventsCheck, poolReplicasCheck, rgwInfoCheck, specAnalysisCheck}
	disableAllChecksStr := strings.Join(disableAllChecks, ",")
	lcmConfigMap := unitinputs.GetConfigMap(configRequest.Name, configRequest.Namespace, map[string]string{"HEALTH_CHECKS_SKIP": disableAllChecksStr, "HEALTH_LOG_LEVEL": "trace"})
	configReconciler := &lcmconfig.ReconcileCephDeploymentHealthConfig{
		Client: faketestclients.GetClientBuilderWithObjects(lcmConfigMap).Build(),
		Scheme: faketestscheme.Scheme,
	}

	healthRequest := reconcile.Request{NamespacedName: types.NamespacedName{Namespace: unitinputs.LcmObjectMeta.Namespace, Name: unitinputs.LcmObjectMeta.Name}}
	healthReconciler := FakeReconciler()
	healthReconciler.Client = faketestclients.GetClient(faketestclients.GetClientBuilder().WithStatusSubresource(unitinputs.CephDeploymentHealth.DeepCopy()).WithObjects(unitinputs.CephDeploymentHealth.DeepCopy()))

	inputResources := map[string]runtime.Object{
		"cephdeploymenthealths": &lcmv1alpha1.CephDeploymentHealthList{Items: []lcmv1alpha1.CephDeploymentHealth{*unitinputs.CephDeploymentHealthStatusOk.DeepCopy()}},
		"daemonsets":            unitinputs.DaemonSetListReady,
		"deployments":           unitinputs.DeploymentList,
		"configmaps":            unitinputs.ConfigMapList,
		"cephclusters":          &unitinputs.CephClusterListReady,
		"cephblockpools":        &unitinputs.CephBlockPoolListEmpty,
		"cephclients":           &unitinputs.CephClientListEmpty,
		"cephobjectstores":      &unitinputs.CephObjectStoreListEmpty,
		"cephobjectstoreusers":  &unitinputs.CephObjectStoreUserListEmpty,
		"objectbucketclaims":    &unitinputs.ObjectBucketClaimListEmpty,
		"cephobjectrealms":      &unitinputs.CephObjectRealmListEmpty,
		"cephobjectzonegroups":  &unitinputs.CephObjectZoneGroupListEmpty,
		"cephobjectzones":       &unitinputs.CephObjectZoneListEmpty,
		"cephfilesystems":       &unitinputs.CephFilesystemListEmpty,
	}

	faketestclients.FakeReaction(healthReconciler.Rookclientset, "list", rookListResources, inputResources, nil)
	faketestclients.FakeReaction(healthReconciler.Claimclientset, "list", claimListResources, inputResources, nil)
	faketestclients.FakeReaction(healthReconciler.Lcmclientset, "get", []string{"cephdeploymenthealths"}, inputResources, nil)
	faketestclients.FakeReaction(healthReconciler.Rookclientset, "get", rookGetResources, inputResources, nil)
	faketestclients.FakeReaction(healthReconciler.Kubeclientset.AppsV1(), "get", []string{"deployments", "daemonsets"}, inputResources, nil)

	ctx := context.TODO()
	var err error
	_, err = configReconciler.Reconcile(ctx, configRequest)
	assert.Nil(t, err)
	_, err = healthReconciler.Reconcile(ctx, healthRequest)
	assert.Nil(t, err)
	expectedLcmConfig := lcmconfig.LcmConfig{
		RookNamespace:            "rook-ceph",
		DiskDaemonPort:           9999,
		DiskDaemonPlacementLabel: "pelagia-disk-daemon=true",
		HealthParams: &lcmconfig.HealthParams{
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
			ChecksSkip:                disableAllChecks,
			LogLevel:                  -1,
			UsageDetailsClassesFilter: "",
			UsageDetailsPoolsFilter:   "",
			RgwPublicAccessLabel:      "external_access=rgw",
		},
	}
	assert.Equal(t, expectedLcmConfig, lcmconfig.GetConfiguration("lcm-namespace"))

	cephDeploymentHealth := &lcmv1alpha1.CephDeploymentHealth{}
	err = healthReconciler.Client.Get(ctx, healthRequest.NamespacedName, cephDeploymentHealth)
	assert.Nil(t, err)
	expectedStatus := &lcmv1alpha1.CephDeploymentHealthReport{
		RookOperator:    unitinputs.RookOperatorStatusOk,
		RookCephObjects: unitinputs.RookCephObjectsReportOnlyCephCluster,
	}
	assert.Equal(t, expectedStatus, cephDeploymentHealth.Status.HealthReport)

	faketestclients.CleanupFakeClientReactions(healthReconciler.Lcmclientset)
	faketestclients.CleanupFakeClientReactions(healthReconciler.Kubeclientset.AppsV1())
	faketestclients.CleanupFakeClientReactions(healthReconciler.Rookclientset)
	faketestclients.CleanupFakeClientReactions(healthReconciler.Claimclientset)
	lcmconfig.ParamsToControl = oldVal
}
