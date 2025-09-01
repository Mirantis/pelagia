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
	"testing"

	"github.com/pkg/errors"
	cephv1 "github.com/rook/rook/pkg/apis/ceph.rook.io/v1"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"

	cephlcmv1alpha1 "github.com/Mirantis/pelagia/pkg/apis/ceph.pelagia.lcm/v1alpha1"
	lcmcommon "github.com/Mirantis/pelagia/pkg/common"
	faketestclients "github.com/Mirantis/pelagia/test/unit/clients"
	unitinputs "github.com/Mirantis/pelagia/test/unit/inputs"
)

func TestEnsureClusterState(t *testing.T) {
	cephDeplWithEvents := unitinputs.CephDeployNonMosk.DeepCopy()
	cephDeplWithEvents.Spec.ExtraOpts = &cephlcmv1alpha1.CephDeploymentExtraOpts{EnableProgressEvents: true}
	tests := []struct {
		name              string
		cephDpl           *cephlcmv1alpha1.CephDeployment
		cephPool          *cephv1.CephBlockPool
		cliOutputs        map[string]string
		inputResources    map[string]runtime.Object
		expectedResources map[string]runtime.Object
		apiErrors         map[string]error
		changed           bool
		expectedError     string
	}{
		{
			name: "cluster is not deployed",
			inputResources: map[string]runtime.Object{
				"configmaps": &corev1.ConfigMapList{},
			},
			changed: true,
		},
		{
			name: "prometheus check: cluster prometheus module verify failed",
			inputResources: map[string]runtime.Object{
				"configmaps": &corev1.ConfigMapList{Items: []corev1.ConfigMap{unitinputs.RookCephMonEndpoints}},
			},
			expectedError: "failed to verify mgr modules 'prometheus' enabled: failed to check mgr modules: failed to run command 'ceph mgr module ls -f json': command failed",
		},
		{
			name: "prometheus check: cluster prometheus module enable failed",
			inputResources: map[string]runtime.Object{
				"configmaps": &corev1.ConfigMapList{Items: []corev1.ConfigMap{unitinputs.RookCephMonEndpoints}},
			},
			cliOutputs: map[string]string{
				"ceph mgr module ls -f json": unitinputs.MgrModuleLsNoPrometheus,
			},
			expectedError: "failed to verify mgr modules 'prometheus' enabled: failed to enable mgr module 'prometheus': failed to run command 'ceph mgr module enable prometheus': command failed",
		},
		{
			name: "progress events: cluster failed to check progress events",
			inputResources: map[string]runtime.Object{
				"configmaps": &corev1.ConfigMapList{Items: []corev1.ConfigMap{unitinputs.RookCephMonEndpoints}},
			},
			cliOutputs: map[string]string{
				"ceph mgr module ls -f json": unitinputs.MgrModuleLsWithPrometheus,
			},
			expectedError: "failed to verify mgr progress events state: failed to check mgr progress events: failed to run command 'ceph config get mgr mgr/progress/allow_pg_recovery_event': command failed",
		},
		{
			name: "progress events: cluster failed to disable progress events",
			inputResources: map[string]runtime.Object{
				"configmaps": &corev1.ConfigMapList{Items: []corev1.ConfigMap{unitinputs.RookCephMonEndpoints}},
			},
			cliOutputs: map[string]string{
				"ceph mgr module ls -f json":                               unitinputs.MgrModuleLsWithPrometheus,
				"ceph config get mgr mgr/progress/allow_pg_recovery_event": "true",
			},
			expectedError: "failed to verify mgr progress events state: failed to mgr update progress events state: failed to run command 'ceph config set mgr mgr/progress/allow_pg_recovery_event false': command failed",
		},
		{
			name:    "progress events: cluster failed to enable progress events",
			cephDpl: cephDeplWithEvents,
			inputResources: map[string]runtime.Object{
				"configmaps": &corev1.ConfigMapList{Items: []corev1.ConfigMap{unitinputs.RookCephMonEndpoints}},
			},
			cliOutputs: map[string]string{
				"ceph mgr module ls -f json":                               unitinputs.MgrModuleLsWithPrometheus,
				"ceph config get mgr mgr/progress/allow_pg_recovery_event": "false",
			},
			expectedError: "failed to verify mgr progress events state: failed to mgr update progress events state: failed to run command 'ceph config set mgr mgr/progress/allow_pg_recovery_event true': command failed",
		},
		{
			name: "builtin pools verify: cluster failed to check ceph pools",
			inputResources: map[string]runtime.Object{
				"configmaps": &corev1.ConfigMapList{Items: []corev1.ConfigMap{unitinputs.RookCephMonEndpoints}},
			},
			cliOutputs: map[string]string{
				"ceph mgr module ls -f json":                               unitinputs.MgrModuleLsWithPrometheus,
				"ceph config get mgr mgr/progress/allow_pg_recovery_event": "false",
			},
			expectedError: "failed to verify builtin pools have corresponding CephBlockPools: failed to check ceph pools: failed to run command 'ceph osd pool ls -f json': command failed",
		},
		{
			name: "builtin pools verify: cluster has no builtin pools yet",
			inputResources: map[string]runtime.Object{
				"configmaps": &corev1.ConfigMapList{Items: []corev1.ConfigMap{unitinputs.RookCephMonEndpoints}},
			},
			cliOutputs: map[string]string{
				"ceph mgr module ls -f json":                               unitinputs.MgrModuleLsWithPrometheus,
				"ceph config get mgr mgr/progress/allow_pg_recovery_event": "false",
				"ceph osd pool ls -f json":                                 "[]",
			},
		},
		{
			name: "prometheus check: cluster enabled prometheus, no other changes",
			inputResources: map[string]runtime.Object{
				"configmaps": &corev1.ConfigMapList{Items: []corev1.ConfigMap{unitinputs.RookCephMonEndpoints}},
			},
			cliOutputs: map[string]string{
				"ceph mgr module ls -f json":                               unitinputs.MgrModuleLsNoPrometheus,
				"ceph mgr module enable prometheus":                        "",
				"ceph config get mgr mgr/progress/allow_pg_recovery_event": "false",
				"ceph osd pool ls -f json":                                 "[]",
			},
			changed: true,
		},
		{
			name: "progress events: cluster disabled progress events, no other changes",
			inputResources: map[string]runtime.Object{
				"configmaps": &corev1.ConfigMapList{Items: []corev1.ConfigMap{unitinputs.RookCephMonEndpoints}},
			},
			cliOutputs: map[string]string{
				"ceph mgr module ls -f json":                                     unitinputs.MgrModuleLsWithPrometheus,
				"ceph config get mgr mgr/progress/allow_pg_recovery_event":       "true",
				"ceph config set mgr mgr/progress/allow_pg_recovery_event false": "",
				"ceph osd pool ls -f json":                                       "[]",
			},
			changed: true,
		},
		{
			name:    "progress events: cluster enabled progress events, no other changes",
			cephDpl: cephDeplWithEvents,
			inputResources: map[string]runtime.Object{
				"configmaps": &corev1.ConfigMapList{Items: []corev1.ConfigMap{unitinputs.RookCephMonEndpoints}},
			},
			cliOutputs: map[string]string{
				"ceph mgr module ls -f json":                                    unitinputs.MgrModuleLsWithPrometheus,
				"ceph config get mgr mgr/progress/allow_pg_recovery_event":      "false",
				"ceph config set mgr mgr/progress/allow_pg_recovery_event true": "",
				"ceph osd pool ls -f json":                                      "[]",
			},
			changed: true,
		},
		{
			name:    "builtin pools verify: no builtin cephblockpools, failed to check present cephblockpools",
			cephDpl: cephDeplWithEvents,
			inputResources: map[string]runtime.Object{
				"configmaps": &corev1.ConfigMapList{Items: []corev1.ConfigMap{unitinputs.RookCephMonEndpoints}},
			},
			cliOutputs: map[string]string{
				"ceph mgr module ls -f json":                               unitinputs.MgrModuleLsWithPrometheus,
				"ceph config get mgr mgr/progress/allow_pg_recovery_event": "true",
				"ceph osd pool ls -f json":                                 unitinputs.CephOsdLspools,
			},
			expectedError: "failed to verify builtin pools have corresponding CephBlockPools: failed to list CephBlockPools in 'rook-ceph' namespace: failed to list cephblockpools",
		},
		{
			name:    "builtin pools verify: no builtin cephblockpools, failed to create pools",
			cephDpl: cephDeplWithEvents,
			inputResources: map[string]runtime.Object{
				"configmaps":     &corev1.ConfigMapList{Items: []corev1.ConfigMap{unitinputs.RookCephMonEndpoints}},
				"cephblockpools": &cephv1.CephBlockPoolList{},
			},
			cliOutputs: map[string]string{
				"ceph mgr module ls -f json":                               unitinputs.MgrModuleLsWithPrometheus,
				"ceph config get mgr mgr/progress/allow_pg_recovery_event": "true",
				"ceph osd pool ls -f json":                                 unitinputs.CephOsdLspools,
			},
			apiErrors:     map[string]error{"create-cephblockpools": errors.New("create failed")},
			expectedError: "failed to verify builtin pools have corresponding CephBlockPools: failed to create '.rgw.root' CephBlockPool override: failed to create CephBlockPool rook-ceph/builtin-rgw-root: create failed; failed to create '.mgr' CephBlockPool override: failed to create CephBlockPool rook-ceph/builtin-mgr: create failed",
		},
		{
			name:    "builtin pools verify: no builtin cephblockpools, create completed",
			cephDpl: cephDeplWithEvents,
			inputResources: map[string]runtime.Object{
				"configmaps":     &corev1.ConfigMapList{Items: []corev1.ConfigMap{unitinputs.RookCephMonEndpoints}},
				"cephblockpools": &cephv1.CephBlockPoolList{},
			},
			expectedResources: map[string]runtime.Object{
				"cephblockpools": &cephv1.CephBlockPoolList{
					Items: []cephv1.CephBlockPool{*unitinputs.BuiltinRgwRootPool, *unitinputs.BuiltinMgrPool},
				},
			},
			cliOutputs: map[string]string{
				"ceph mgr module ls -f json":                               unitinputs.MgrModuleLsWithPrometheus,
				"ceph config get mgr mgr/progress/allow_pg_recovery_event": "true",
				"ceph osd pool ls -f json":                                 unitinputs.CephOsdLspools,
			},
			changed: true,
		},
		{
			name: "builtin pools verify: no default pool in spec and no rgw, nothing to do",
			inputResources: map[string]runtime.Object{
				"configmaps":     &corev1.ConfigMapList{Items: []corev1.ConfigMap{unitinputs.RookCephMonEndpoints}},
				"cephblockpools": &cephv1.CephBlockPoolList{},
			},
			cliOutputs: map[string]string{
				"ceph mgr module ls -f json":                               unitinputs.MgrModuleLsWithPrometheus,
				"ceph config get mgr mgr/progress/allow_pg_recovery_event": "false",
				"ceph osd pool ls -f json":                                 unitinputs.CephOsdLspools,
			},
		},
		{
			name:    "builtin pools verify: failed to update pools",
			cephDpl: cephDeplWithEvents,
			inputResources: map[string]runtime.Object{
				"configmaps": &corev1.ConfigMapList{Items: []corev1.ConfigMap{unitinputs.RookCephMonEndpoints}},
				"cephblockpools": &cephv1.CephBlockPoolList{
					Items: []cephv1.CephBlockPool{
						func() cephv1.CephBlockPool {
							pool := unitinputs.BuiltinRgwRootPool.DeepCopy()
							pool.Spec.CrushRoot = "default"
							return *pool
						}(),
						func() cephv1.CephBlockPool {
							pool := unitinputs.BuiltinMgrPool.DeepCopy()
							pool.Spec.CrushRoot = ""
							return *pool
						}(),
					},
				},
			},
			cliOutputs: map[string]string{
				"ceph mgr module ls -f json":                               unitinputs.MgrModuleLsWithPrometheus,
				"ceph config get mgr mgr/progress/allow_pg_recovery_event": "true",
				"ceph osd pool ls -f json":                                 unitinputs.CephOsdLspools,
			},
			apiErrors:     map[string]error{"update-cephblockpools": errors.New("update failed")},
			expectedError: "failed to verify builtin pools have corresponding CephBlockPools: failed to update '.rgw.root' CephBlockPool override: failed to update CephBlockPool rook-ceph/builtin-rgw-root: update failed; failed to update '.mgr' CephBlockPool override: failed to update CephBlockPool rook-ceph/builtin-mgr: update failed",
		},
		{
			name:    "builtin pools verify: update pools",
			cephDpl: cephDeplWithEvents,
			inputResources: map[string]runtime.Object{
				"configmaps": &corev1.ConfigMapList{Items: []corev1.ConfigMap{unitinputs.RookCephMonEndpoints}},
				"cephblockpools": &cephv1.CephBlockPoolList{
					Items: []cephv1.CephBlockPool{
						func() cephv1.CephBlockPool {
							pool := unitinputs.BuiltinRgwRootPool.DeepCopy()
							pool.Spec.CrushRoot = "default"
							return *pool
						}(),
						func() cephv1.CephBlockPool {
							pool := unitinputs.BuiltinMgrPool.DeepCopy()
							pool.Spec.CrushRoot = ""
							return *pool
						}(),
					},
				},
			},
			expectedResources: map[string]runtime.Object{
				"cephblockpools": &cephv1.CephBlockPoolList{
					Items: []cephv1.CephBlockPool{*unitinputs.BuiltinRgwRootPool, *unitinputs.BuiltinMgrPool},
				},
			},
			cliOutputs: map[string]string{
				"ceph mgr module ls -f json":                               unitinputs.MgrModuleLsWithPrometheus,
				"ceph config get mgr mgr/progress/allow_pg_recovery_event": "true",
				"ceph osd pool ls -f json":                                 unitinputs.CephOsdLspools,
			},
			changed: true,
		},
		{
			name:    "builtin pools verify: create only multisite related rgw pool",
			cephDpl: &unitinputs.CephDeployMultisiteMasterRgw,
			inputResources: map[string]runtime.Object{
				"configmaps": &corev1.ConfigMapList{Items: []corev1.ConfigMap{unitinputs.RookCephMonEndpoints}},
				"cephblockpools": &cephv1.CephBlockPoolList{
					Items: []cephv1.CephBlockPool{*unitinputs.BuiltinMgrPool},
				},
			},
			expectedResources: map[string]runtime.Object{
				"cephblockpools": &cephv1.CephBlockPoolList{
					Items: []cephv1.CephBlockPool{
						*unitinputs.BuiltinMgrPool,
						func() cephv1.CephBlockPool {
							pool := unitinputs.BuiltinRgwRootPool.DeepCopy()
							pool.Spec.FailureDomain = "host"
							return *pool
						}(),
					},
				},
			},
			cliOutputs: map[string]string{
				"ceph mgr module ls -f json":                               unitinputs.MgrModuleLsWithPrometheus,
				"ceph config get mgr mgr/progress/allow_pg_recovery_event": "false",
				"ceph osd pool ls -f json":                                 unitinputs.CephOsdLspools,
			},
			changed: true,
		},
		{

			name:    "nothing to do",
			cephDpl: cephDeplWithEvents,
			inputResources: map[string]runtime.Object{
				"configmaps": &corev1.ConfigMapList{Items: []corev1.ConfigMap{unitinputs.RookCephMonEndpoints}},
				"cephblockpools": &cephv1.CephBlockPoolList{
					Items: []cephv1.CephBlockPool{*unitinputs.BuiltinMgrPool.DeepCopy(), *unitinputs.BuiltinRgwRootPool.DeepCopy()},
				},
			},
			cliOutputs: map[string]string{
				"ceph mgr module ls -f json":                               unitinputs.MgrModuleLsWithPrometheus,
				"ceph config get mgr mgr/progress/allow_pg_recovery_event": "true",
				"ceph osd pool ls -f json":                                 unitinputs.CephOsdLspools,
			},
		},
	}
	oldFunc := lcmcommon.RunPodCommandWithValidation
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if test.cephDpl == nil {
				test.cephDpl = unitinputs.BaseCephDeployment.DeepCopy()
			}
			c := fakeDeploymentConfig(&deployConfig{cephDpl: test.cephDpl}, nil)
			c.cdConfig.currentCephVersion = lcmcommon.LatestRelease
			lcmcommon.RunPodCommandWithValidation = func(e lcmcommon.ExecConfig) (string, string, error) {
				if output, ok := test.cliOutputs[e.Command]; ok {
					return output, "", nil
				}
				return "", "", errors.New("command failed")
			}
			faketestclients.FakeReaction(c.api.Kubeclientset.CoreV1(), "get", []string{"configmaps"}, test.inputResources, test.apiErrors)
			faketestclients.FakeReaction(c.api.Rookclientset, "list", []string{"cephblockpools"}, test.inputResources, nil)
			faketestclients.FakeReaction(c.api.Rookclientset, "create", []string{"cephblockpools"}, test.inputResources, test.apiErrors)
			faketestclients.FakeReaction(c.api.Rookclientset, "update", []string{"cephblockpools"}, test.inputResources, test.apiErrors)
			test.expectedResources = faketestclients.PrepareExpectedResources(test.inputResources, test.expectedResources)

			changed, err := c.ensureClusterState()
			if test.expectedError != "" {
				assert.NotNil(t, err)
				assert.Equal(t, test.expectedError, err.Error())
			} else {
				assert.Nil(t, err)
			}
			assert.Equal(t, test.changed, changed)
			assert.Equal(t, test.expectedResources, test.inputResources)
			faketestclients.CleanupFakeClientReactions(c.api.Kubeclientset.CoreV1())
			faketestclients.CleanupFakeClientReactions(c.api.Rookclientset)
		})
	}
	lcmcommon.RunPodCommandWithValidation = oldFunc
}
