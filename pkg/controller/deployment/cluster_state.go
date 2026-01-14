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
	"strings"

	"github.com/pkg/errors"
	cephv1 "github.com/rook/rook/pkg/apis/ceph.rook.io/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	cephlcmv1alpha1 "github.com/Mirantis/pelagia/pkg/apis/ceph.pelagia.lcm/v1alpha1"
	lcmcommon "github.com/Mirantis/pelagia/pkg/common"
)

// EnsureClusterState verifies healthy and fully workable ceph cluster functionality.
// For example, this function could be used for a various inplace workarounds and improvements
// of incorrect Rook/Ceph behaviour.
func (c *cephDeploymentConfig) ensureClusterState() (bool, error) {
	// do not ensure cluster state if there is no ceph cluster yet
	if !isCephDeployed(c.context, *c.log, c.api.Kubeclientset, c.lcmConfig.RookNamespace) {
		c.log.Warn().Msgf("%s/%s configmap not found, cluster state ensure skipped", c.lcmConfig.RookNamespace, lcmcommon.MonMapConfigMapName)
		return true, nil
	}
	changed := false
	// verify prometheus mgr module is enabled, since it is not always on
	// and we need that module
	prometheusChanged, err := c.verifyPrometheusEnabled()
	if err != nil {
		return false, errors.Wrap(err, "failed to verify mgr modules 'prometheus' enabled")
	}
	changed = changed || prometheusChanged

	// progress events are disabled by default due to CPU overhead
	// so if it is enabled in spec - check that events are enabled in Ceph
	eventsChanged, err := c.verifyProgressEvents()
	if err != nil {
		return false, errors.Wrap(err, "failed to verify mgr progress events state")
	}
	changed = changed || eventsChanged

	// since default pools created with default replicated_rule, which leads to broken autoscale-status
	// because it has no default device class obviously set, set device classes directly if possible
	poolsChanged, err := c.verifyBuiltinPools()
	if err != nil {
		return false, errors.Wrap(err, "failed to verify builtin pools have corresponding CephBlockPools")
	}
	changed = changed || poolsChanged
	return changed, nil
}

// verifyPrometheusEnabled verifies prometheus enabled and enable if it is not
func (c *cephDeploymentConfig) verifyPrometheusEnabled() (bool, error) {
	c.log.Debug().Msg("verify mgr module 'prometheus' is enabled")
	mgrModules := lcmcommon.MgrModuleLs{}
	err := lcmcommon.RunAndParseCephToolboxCLI(c.context, c.api.Kubeclientset, c.api.Config, c.lcmConfig.RookNamespace, "ceph mgr module ls -f json", &mgrModules)
	if err != nil {
		errMsg := "failed to check mgr modules"
		return false, errors.Wrap(err, errMsg)
	}

	if !lcmcommon.Contains(append(mgrModules.AlwaysOn, mgrModules.Enabled...), "prometheus") {
		c.log.Info().Msg("enabling mgr module 'prometheus'")
		_, err := lcmcommon.RunCephToolboxCLI(c.context, c.api.Kubeclientset, c.api.Config, c.lcmConfig.RookNamespace, "ceph mgr module enable prometheus")
		if err != nil {
			errMsg := "failed to enable mgr module 'prometheus'"
			return false, errors.Wrap(err, errMsg)
		}
		return true, nil
	}
	return false, nil
}

// check state of progress events
func (c *cephDeploymentConfig) verifyProgressEvents() (bool, error) {
	enable := false
	if c.cdConfig.cephDpl.Spec.ExtraOpts != nil && c.cdConfig.cephDpl.Spec.ExtraOpts.EnableProgressEvents {
		c.log.Debug().Msg("verify mgr progress events are enabled")
		enable = true
	}
	stdout, err := lcmcommon.RunCephToolboxCLI(c.context, c.api.Kubeclientset, c.api.Config, c.lcmConfig.RookNamespace, "ceph config get mgr mgr/progress/allow_pg_recovery_event")
	if err != nil {
		errMsg := "failed to check mgr progress events"
		return false, errors.Wrap(err, errMsg)
	}
	currentValue := strings.Trim(stdout, "\n")
	runCmd := false
	if enable {
		if currentValue == "false" {
			c.log.Info().Msg("enabling mgr progress events")
			runCmd = true
		}
	} else {
		if currentValue == "true" {
			c.log.Info().Msg("disabling mgr progress events")
			runCmd = true
		}
	}
	if runCmd {
		_, err := lcmcommon.RunCephToolboxCLI(c.context, c.api.Kubeclientset, c.api.Config, c.lcmConfig.RookNamespace, fmt.Sprintf("ceph config set mgr mgr/progress/allow_pg_recovery_event %t", enable))
		if err != nil {
			errMsg := "failed to mgr update progress events state"
			return false, errors.Wrap(err, errMsg)
		}
	}
	return runCmd, nil
}

func (c *cephDeploymentConfig) verifyBuiltinPools() (bool, error) {
	c.log.Debug().Msg("verify builtin Ceph pools")
	// Obtain builtin pools which are existing in Ceph cluster itself
	var cephPools []string
	err := lcmcommon.RunAndParseCephToolboxCLI(c.context, c.api.Kubeclientset, c.api.Config, c.lcmConfig.RookNamespace, "ceph osd pool ls -f json", &cephPools)
	if err != nil {
		errMsg := "failed to check ceph pools"
		return false, errors.Wrap(err, errMsg)
	}

	changed := false
	errMsgs := []string{}

	// default pool spec for rbd builtin pools
	var defaultPool *cephlcmv1alpha1.CephPoolSpec
	for _, cephDplPool := range c.cdConfig.cephDpl.Spec.Pools {
		if cephDplPool.StorageClassOpts.Default {
			defaultPool = &cephDplPool.CephPoolSpec
			break
		}
	}

	// Build cephblockpool spec only for builtin pools exist in ceph cluster
	builtinPoolsToProcess := []cephv1.CephBlockPool{}
	for _, cephpool := range cephPools {
		if lcmcommon.Contains(builtinCephPools, cephpool) {
			var poolSpec *cephlcmv1alpha1.CephPoolSpec
			if cephpool == ".rgw.root" {
				// skip processing .rgw.root pool if there is no rgw metadata pool defined
				// in a cluster because we cannot predict what .rgw.root we are observing
				if c.cdConfig.cephDpl.Spec.ObjectStorage == nil {
					c.log.Warn().Msgf("builtin pool '%s' found, but no object storage RGW defined in spec, skipping", cephpool)
					c.log.Warn().Msgf("set manually device class in crush rule for pool '%s' if needed", cephpool)
					continue
				}
				if c.cdConfig.cephDpl.Spec.ObjectStorage.MultiSite != nil {
					usedZone := c.cdConfig.cephDpl.Spec.ObjectStorage.Rgw.Zone.Name
					for _, zone := range c.cdConfig.cephDpl.Spec.ObjectStorage.MultiSite.Zones {
						if zone.Name == usedZone {
							poolSpec = &zone.MetadataPool
							break
						}
					}
				} else {
					poolSpec = c.cdConfig.cephDpl.Spec.ObjectStorage.Rgw.MetadataPool
				}
			} else {
				// skip processing builtin rbd pools if there is no default pool found
				// because we cannot rely on random pool if no default pool exists
				// then operator should manually fix the issue (set required device class)
				if defaultPool == nil {
					c.log.Warn().Msgf("builtin pool '%s' found, but no default rbd pool defined in spec, skipping", cephpool)
					c.log.Warn().Msgf("set manually device class in crush rule for pool '%s' if needed", cephpool)
					continue
				}
				poolSpec = defaultPool
			}
			poolSpec.EnableCrushUpdates = &[]bool{true}[0]
			builtinCephPool := generatePool(cephlcmv1alpha1.CephPool{
				Name:          cephpool,
				UseAsFullName: true,
				CephPoolSpec:  *poolSpec,
			}, c.lcmConfig.RookNamespace)
			builtinPoolsToProcess = append(builtinPoolsToProcess, *builtinCephPool)
		}
	}

	if len(builtinPoolsToProcess) > 0 {
		pools, err := c.api.Rookclientset.CephV1().CephBlockPools(c.lcmConfig.RookNamespace).List(c.context, metav1.ListOptions{})
		if err != nil {
			return false, errors.Wrapf(err, "failed to list CephBlockPools in '%s' namespace", c.lcmConfig.RookNamespace)
		}

	processLoop:
		for _, poolToProcess := range builtinPoolsToProcess {
			for _, presentPool := range pools.Items {
				if presentPool.Spec.Name == poolToProcess.Spec.Name {
					// If cephblockpool for builtin pool already exists but differs from current
					// default/rgw meta pool spec then update its spec
					if !reflect.DeepEqual(presentPool.Spec, poolToProcess.Spec) {
						c.log.Info().Msgf("updating CephBlockPool '%s/%s' override for default pool '%s'", presentPool.Namespace, presentPool.Name, presentPool.Spec.Name)
						lcmcommon.ShowObjectDiff(*c.log, presentPool.Spec, poolToProcess.Spec)
						presentPool.Spec = poolToProcess.Spec
						err := c.processBlockPools(objectUpdate, &presentPool)
						if err != nil {
							errMsg := errors.Wrapf(err, "failed to update '%s' CephBlockPool override", presentPool.Spec.Name)
							c.log.Error().Err(errMsg).Msg("")
							errMsgs = append(errMsgs, errMsg.Error())
						} else {
							changed = true
						}
					}
					continue processLoop
				}
			}
			// create cephblockpools for pools which are not processed yet (which are not found in cephblockpool list)
			c.log.Info().Msgf("creating CephBlockPool '%s/%s' override for default pool '%s'", poolToProcess.Namespace, poolToProcess.Name, poolToProcess.Spec.Name)
			err := c.processBlockPools(objectCreate, &poolToProcess)
			if err != nil {
				errMsg := errors.Wrapf(err, "failed to create '%s' CephBlockPool override", poolToProcess.Spec.Name)
				c.log.Error().Err(errMsg).Msg("")
				errMsgs = append(errMsgs, errMsg.Error())
				continue
			}
			changed = true
		}
	}

	if len(errMsgs) > 0 {
		return false, errors.New(strings.Join(errMsgs, "; "))
	}
	return changed, nil
}
