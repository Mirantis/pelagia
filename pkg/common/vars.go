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

package lcmcommon

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
)

const (
	// app names for disk-daemon and toolbox
	PelagiaToolBox    = "pelagia-ceph-toolbox"
	PelagiaDiskDaemon = "pelagia-disk-daemon"
	// csi plugin names
	CephCSIRBDPluginDaemonSetName    = "csi-rbdplugin"
	CephCSICephFSPluginDaemonSetName = "csi-cephfsplugin"
	// CephCSIRBDNodeClientName is the name of CSI RBD node client
	CephCSIRBDNodeClientName = "csi-rbd-node"
	// CephCSIRBDProvisionerClientName is the name of CSI RBD provisioner client
	CephCSIRBDProvisionerClientName = "csi-rbd-provisioner"
	// CephCSICephFSNodeClientName is the name of CSI CephFS node client
	CephCSICephFSNodeClientName = "csi-cephfs-node"
	// CephCSICephFSProvisionerClientName is the name of CSI CephFS provisioner client
	CephCSICephFSProvisionerClientName = "csi-cephfs-provisioner"
	// rook related vars
	RookCephOperatorName      = "rook-ceph-operator"
	RookDiscoverName          = "rook-discover"
	RookCephMonSecretName     = "rook-ceph-mon"
	RookOperatorConfigMapName = "rook-ceph-operator-config"
	MonMapConfigMapName       = "rook-ceph-mon-endpoints"
	// default data dir host path if not specified in spec
	DefaultDataDirHostPath = "/var/lib/rook"
	// lcm related vars
	RunCephCommandTimeout = 10
	// marker for stray osds
	StrayOsdNodeMarker = "__stray"
	// marker to detect lvm created by rook, since rook always create vg/lv with
	// prefix 'osd-' any manual lvm should not start with that prefix
	RookLVMarker = "osd-"
	// DeploymentRestartAnnotation indicates timestamp when deployment restart was requested
	DeploymentRestartAnnotation = "cephdeployment.lcm.mirantis.com/restartedAt"
	// Label template for nodes used in CephDeployment
	CephNodeLabelTemplate = "ceph_role_%s"
	// Timeout for disk cleanup job
	DiskCleanupTimeout = 3600
)

var (
	CephDaemonKeys   = []string{"mds", "mgr", "mon", "osd", "rgw"}
	DefaultCephProbe = &corev1.Probe{
		TimeoutSeconds:   5,
		FailureThreshold: 5,
	}

	CephNodeLabels = func() map[string]string {
		labelsMap := map[string]string{}
		for _, daemon := range CephDaemonKeys {
			labelsMap[daemon] = fmt.Sprintf(CephNodeLabelTemplate, daemon)
		}
		return labelsMap
	}()
)
