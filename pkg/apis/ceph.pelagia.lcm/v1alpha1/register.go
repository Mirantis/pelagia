// Package v1alpha1 contains API Schema definitions for the cephdeploymenthealth v1alpha1 API group
// +k8s:deepcopy-gen=package,register
// +groupName=lcm.mirantis.com
package v1alpha1

import (
	"context"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/scheme"
)

var (
	// SchemeGroupVersion is group version used to register these objects
	SchemeGroupVersion = schema.GroupVersion{Group: "lcm.mirantis.com", Version: "v1alpha1"}

	// SchemeBuilder is used to add go types to the GroupVersionKind scheme
	SchemeBuilder = &scheme.Builder{GroupVersion: SchemeGroupVersion}

	AddToScheme = SchemeBuilder.AddToScheme
)

func Resource(resource string) schema.GroupResource {
	return SchemeGroupVersion.WithResource(resource).GroupResource()
}

func UpdateCephDeploymentStatus(ctx context.Context, cephDeploy *CephDeployment, status CephDeploymentStatus, client client.Client) error {
	cephDeploy.Status = status
	if err := client.Status().Update(ctx, cephDeploy); err != nil {
		return errors.Errorf("failed to update status for the CephDeployment %v/%v: %v",
			cephDeploy.Namespace, cephDeploy.Name, err)
	}
	return nil
}

func UpdateCephDeploymentSecretStatus(ctx context.Context, cdSecret *CephDeploymentSecret, status *CephDeploymentSecretStatus, client client.Client) error {
	cdSecret.Status = status
	if err := client.Status().Update(ctx, cdSecret); err != nil {
		return errors.Errorf("failed to update status for the CephDeploymentSecret %v/%v: %v",
			cdSecret.Namespace, cdSecret.Name, err)
	}
	return nil
}

func UpdateCephDeploymentMaintenanceStatus(ctx context.Context, miraCephMaintenance *CephDeploymentMaintenance, status *CephDeploymentMaintenanceStatus, client client.Client) error {
	miraCephMaintenance.Status = status
	if err := client.Status().Update(ctx, miraCephMaintenance); err != nil {
		return errors.Errorf("failed to update status for the miracephmaintenance %v/%v: %v",
			miraCephMaintenance.Namespace, miraCephMaintenance.Name, err)
	}
	return nil
}

func UpdateCephHealthDeploymentStatus(ctx context.Context, cephdeploymenthealth *CephDeploymentHealth, status CephDeploymentHealthStatus, client client.Client) error {
	cephdeploymenthealth.Status = status
	if err := client.Status().Update(ctx, cephdeploymenthealth); err != nil {
		return errors.Errorf("failed to update status for the CephDeploymentHealth %v/%v: %v",
			cephdeploymenthealth.Namespace, cephdeploymenthealth.Name, err)
	}
	return nil
}

func UpdateCephOsdRemoveTaskStatus(ctx context.Context, cephosdremovetask *CephOsdRemoveTask, status *CephOsdRemoveTaskStatus, client client.Client) error {
	cephosdremovetask.Status = status
	if err := client.Status().Update(ctx, cephosdremovetask); err != nil {
		return errors.Errorf("failed to update status for the CephOsdRemoveTask %v/%v: %v",
			cephosdremovetask.Namespace, cephosdremovetask.Name, err)
	}
	return nil
}
