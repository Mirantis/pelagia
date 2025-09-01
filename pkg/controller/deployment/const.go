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
	"sort"

	corev1 "k8s.io/api/core/v1"
)

const (
	openstackSharedSecret = "openstack-ceph-keys"
	openstackRgwCredsName = "openstack-rgw-creds"
	adminSecretName       = "rook-ceph-mon"
	// secret with external connection string
	externalStringSecretName = "pelagia-external-connection"

	rgwStorageClassName    = "rgw-storage-class"
	rgwSslCertSecretName   = "rgw-ssl-certificate"
	rgwAdminUserSecretName = "rgw-admin-ops-user"
	// Ceilometer metrics user
	rgwMetricsUser = "rgw-ceilometer"

	rookConfigOverrideName      = "rook-config-override"
	rookCephMonEndpointsMapName = "rook-ceph-mon-endpoints"

	// label for policies created by cephdeployment
	rookNetworkPolicyLabel = "cephdeployment.lcm.mirantis.com/networkpolicy"

	rookStorageClassLabelKey   = "rook-ceph-storage-class"
	rookDefaultSCAnnotationKey = "storageclass.kubernetes.io/is-default-class"
	// label to mark class do not remove during reconcile
	rookStorageClassKeepOnSpecRemove = "rook-ceph-storage-class-keep-on-spec-remove"
	// Provisioners names
	rookRBDProvisionerName    = "rook-ceph.rbd.csi.ceph.com"
	rookCephFSProvisionerName = "rook-ceph.cephfs.csi.ceph.com"

	cephDaemonsetLabel        = "ceph-daemonset-available-node"
	cephDaemonsetDrainRequest = "kaas.mirantis.com/lcm-drained"
	cephDaemonsetDrainReady   = "kaas.mirantis.com/csi-drained"
	cephVolumeAttachmentType  = "rook-ceph.rbd.csi.ceph.com"

	cephNodeLabelTemplate         = "ceph_role_%s"
	cephKubeTopologyLabelTemplate = "cephdpl-prev-%s"
	nodeWithOSDSelectorTemplate   = "app=rook-ceph-osd,failure-domain=%s"

	// osDplReadyState is OpenstackDeploymentStatus desired state
	osDplReadyState = "APPLIED"

	//monIPAnnotation reflects key to set static mon IP to a node
	monIPAnnotation = "network.rook.io/mon-ip"

	// cephdeployment related annotations
	cephConfigMapUpdateTimestampLabel            = "cephdeployment.lcm.mirantis.com/config-generated"
	cephRuntimeRgwParametersUpdateTimestampLabel = "cephdeployment.lcm.mirantis.com/runtime-rgw-params-updated"
	cephRuntimeOsdParametersUpdateTimestampLabel = "cephdeployment.lcm.mirantis.com/runtime-osd-params-updated"
	sslCertGenerationTimestampLabel              = "cephdeployment.lcm.mirantis.com/ssl-cert-generated"
	// labels identifying osd restart reason and timestamp
	cephRestartOsdLabel          = "cephdeployment.lcm.mirantis.com/restart-osd-reason"
	cephRestartOsdTimestampLabel = "cephdeployment.lcm.mirantis.com/restart-osd-requested"

	cephIngressLabel = "cephdeployment.lcm.mirantis.com/ingress"

	//PoolPreserveOnDeleteAnnotation label prevents removing CephBlockPool by Pelagia controller
	poolPreserveOnDeleteAnnotation = "cephdeployment.lcm.mirantis.com/preserve-on-delete"
	// subVolumeGroupName is default subvolumegroup name to create for cephfs csi
	subVolumeGroupName = "csi"
)

type objectProcess string

const (
	objectCreate objectProcess = "create"
	objectUpdate objectProcess = "update"
	objectDelete objectProcess = "delete"
)

var (
	// builtinCephPools contains a list of system ceph pools created by Rook and requiring special handling for pool naming
	builtinCephPools       = []string{".mgr", ".rgw.root"}
	cephDaemonKeys         = []string{"mds", "mgr", "mon", "osd", "rgw"}
	cephNodeAnnotationKeys = []string{monIPAnnotation}
	// cephIgnoredHealthWarnings contains a list of Ceph health warnings that should be ignored during cluster health checks
	cephIgnoredHealthWarnings = []string{
		"OSDMAP_FLAGS",
		"TOO_FEW_PGS",
		"SLOW_OPS",
		"OLD_CRUSH_TUNABLES",
		"OLD_CRUSH_STRAW_CALC_VERSION",
		"POOL_APP_NOT_ENABLED",
		"MON_DISK_LOW",
		"RECENT_CRASH",
	}
	crushTopologyAllowedKeys = map[string]string{
		"datacenter": "topology.rook.io/datacenter",
		"room":       "topology.rook.io/room",
		"pdu":        "topology.rook.io/pdu",
		"row":        "topology.rook.io/row",
		"rack":       "topology.rook.io/rack",
		"chassis":    "topology.rook.io/chassis",
		"region":     "topology.kubernetes.io/region",
		"zone":       "topology.kubernetes.io/zone",
	}
	// template for config map to track section changes
	cephConfigSectionHashLabel = "cephdeployment.lcm.mirantis.com/config-%s-hash"
	// template for keeping last update for parameters under specific section, max lentgh after / is 63 symbols
	cephConfigParametersUpdateTimestampLabel = "cephdeployment.lcm.mirantis.com/config-%s-updated"

	defaultCephProbe = &corev1.Probe{
		TimeoutSeconds:   5,
		FailureThreshold: 5,
	}

	cephNodeLabels = func() map[string]string {
		labelsMap := map[string]string{}
		for _, daemon := range cephDaemonKeys {
			labelsMap[daemon] = fmt.Sprintf(cephNodeLabelTemplate, daemon)
		}
		return labelsMap
	}()
)

func getCrushKeys() []string {
	keys := make([]string, 0)
	for k := range crushTopologyAllowedKeys {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
