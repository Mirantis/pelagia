/*
Copyright 2026 Mirantis IT.

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

package input

import (
	cephv1 "github.com/rook/rook/pkg/apis/ceph.rook.io/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/runtime"

	cephlcmv1alpha1 "github.com/Mirantis/pelagia/pkg/apis/ceph.pelagia.lcm/v1alpha1"
)

var CephDeploymentDeprecated = cephlcmv1alpha1.CephDeployment{
	ObjectMeta: LcmObjectMeta,
	Spec: cephlcmv1alpha1.CephDeploymentSpec{
		DashboardEnabled: &[]bool{true}[0],
		DataDirHostPath:  "/var/lib/custom-path",
		HealthCheck: &cephlcmv1alpha1.CephClusterHealthCheckSpec{
			DaemonHealth: cephv1.DaemonHealthSpec{
				Status:              cephv1.HealthCheckSpec{Disabled: true},
				ObjectStorageDaemon: cephv1.HealthCheckSpec{Timeout: "60s"},
				Monitor:             cephv1.HealthCheckSpec{Timeout: "60s"},
			},
			LivenessProbe: map[cephv1.KeyType]*cephv1.ProbeSpec{
				"osd": {
					Probe: &v1.Probe{
						TimeoutSeconds:   10,
						FailureThreshold: 10,
					},
				},
			},
			StartupProbe: map[cephv1.KeyType]*cephv1.ProbeSpec{
				"osd": {
					Probe: &v1.Probe{
						TimeoutSeconds:   5,
						FailureThreshold: 5,
					},
				},
			},
		},
		HyperConverge: &cephlcmv1alpha1.CephDeploymentHyperConverge{
			Resources: cephv1.ResourceSpec{
				"osd-nvme": v1.ResourceRequirements{
					Limits: v1.ResourceList{
						v1.ResourceCPU:    resource.MustParse("100m"),
						v1.ResourceMemory: resource.MustParse("156Mi"),
					},
					Requests: v1.ResourceList{
						v1.ResourceMemory: resource.MustParse("28Mi"),
						v1.ResourceCPU:    resource.MustParse("10m"),
					},
				},
			},
			Tolerations: map[string]cephlcmv1alpha1.CephDeploymentToleration{
				"all": {
					Rules: []v1.Toleration{
						{
							Key:      "test.kubernetes.io/testkey",
							Effect:   "Schedule",
							Operator: "Exists",
						},
					},
				},
				"mgr": {
					Rules: []v1.Toleration{
						{
							Key:      "test.kubernetes.io/testkey-mgr",
							Effect:   "Schedule",
							Operator: "Exists",
						},
					},
				},
				"mon": {
					Rules: []v1.Toleration{
						{
							Key:      "test.kubernetes.io/testkey-mon",
							Effect:   "Schedule",
							Operator: "Exists",
						},
					},
				},
				"osd": {
					Rules: []v1.Toleration{
						{
							Key:      "test.kubernetes.io/testkey-osd",
							Effect:   "Schedule",
							Operator: "Exists",
						},
					},
				},
			},
		},
		Mgr: &cephlcmv1alpha1.Mgr{
			MgrModules: []cephlcmv1alpha1.CephMgrModule{
				{
					Name:    "balancer",
					Enabled: true,
					Settings: &cephlcmv1alpha1.CephMgrModuleSettings{
						BalancerMode: "upmap",
					},
				},
				{
					Name:    "fake",
					Enabled: true,
				},
			},
		},
		Network: &cephlcmv1alpha1.CephNetworkSpec{
			ClusterNet: "127.0.0.0/16",
			PublicNet:  "192.168.0.0/16",
		},
		Nodes: CephNodesOk,
	},
}

var CephDeploymentSpecClusterYAML = `dashboard:
  enabled: true
dataDirHostPath: /var/lib/custom-path
healthCheck:
  daemonHealth:
    mon:
      timeout: 60s
    osd:
      timeout: 60s
    status:
      disabled: true
  livenessProbe:
    osd:
      probe:
        failureThreshold: 10
        timeoutSeconds: 10
  startupProbe:
    osd:
      probe:
        failureThreshold: 5
        timeoutSeconds: 5
mgr:
  modules:
  - enabled: true
    name: balancer
    settings:
      balancerMode: upmap
  - enabled: true
    name: fake
network:
  addressRanges:
    cluster:
    - 127.0.0.0/16
    public:
    - 192.168.0.0/16
placement:
  all:
    tolerations:
    - effect: Schedule
      key: test.kubernetes.io/testkey
      operator: Exists
  mgr:
    tolerations:
    - effect: Schedule
      key: test.kubernetes.io/testkey-mgr
      operator: Exists
  mon:
    tolerations:
    - effect: Schedule
      key: test.kubernetes.io/testkey-mon
      operator: Exists
  osd:
    tolerations:
    - effect: Schedule
      key: test.kubernetes.io/testkey-osd
      operator: Exists
resources:
  osd-nvme:
    limits:
      cpu: 100m
      memory: 156Mi
    requests:
      cpu: 10m
      memory: 28Mi
`

var CephDeploymentMigrated = cephlcmv1alpha1.CephDeployment{
	ObjectMeta: LcmObjectMeta,
	Spec: cephlcmv1alpha1.CephDeploymentSpec{
		Cluster: &cephlcmv1alpha1.CephCluster{
			RawExtension: runtime.RawExtension{
				Raw: []byte(CephDeploymentSpecClusterYAML),
			},
		},
		Nodes: CephNodesOk,
	},
}

var CephDeploymentMultusDeprecated = func() cephlcmv1alpha1.CephDeployment {
	cd := BaseCephDeployment.DeepCopy()
	cd.Spec.Network.Provider = "multus"
	cd.Spec.Network.Selector = map[cephv1.CephNetworkType]string{
		cephv1.CephNetworkPublic:  "192.168.0.0/16",
		cephv1.CephNetworkCluster: "127.0.0.0/16",
	}
	return *cd
}()

var CephDeploymentSpecClusterMultusYAML = `network:
  addressRanges:
    cluster:
    - 127.0.0.0/16
    public:
    - 192.168.0.0/16
  provider: multus
  selectors:
    cluster: 127.0.0.0/16
    public: 192.168.0.0/16
`

var CephDeploymentMultusMigrated = cephlcmv1alpha1.CephDeployment{
	ObjectMeta: LcmObjectMeta,
	Spec: cephlcmv1alpha1.CephDeploymentSpec{
		Cluster: &cephlcmv1alpha1.CephCluster{
			RawExtension: runtime.RawExtension{
				Raw: []byte(CephDeploymentSpecClusterMultusYAML),
			},
		},
		Nodes: CephNodesOk,
	},
}

var CephDeployExternalDeprecated = cephlcmv1alpha1.CephDeployment{
	ObjectMeta: LcmObjectMeta,
	Spec: cephlcmv1alpha1.CephDeploymentSpec{
		Network: &cephlcmv1alpha1.CephNetworkSpec{
			ClusterNet: "127.0.0.0/32",
			PublicNet:  "127.0.0.0/32",
		},
		External: &[]bool{true}[0],
	},
}

var CephDeploymentSpecClusterExternalYAML = `external:
  enable: true
`

var CephDeployExternalMigrated = cephlcmv1alpha1.CephDeployment{
	ObjectMeta: LcmObjectMeta,
	Spec: cephlcmv1alpha1.CephDeploymentSpec{
		Cluster: &cephlcmv1alpha1.CephCluster{
			RawExtension: runtime.RawExtension{
				Raw: []byte(CephDeploymentSpecClusterExternalYAML),
			},
		},
	},
}
