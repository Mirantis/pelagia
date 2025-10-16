/*
Copyright 2025 The Mirantis Authors.

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

package osdremove

import (
	"fmt"
	"sort"
	"strings"

	"github.com/pkg/errors"
	"github.com/rook/rook/pkg/operator/ceph/controller"
	"github.com/rook/rook/pkg/operator/k8sutil"
	batch "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	lcmv1alpha1 "github.com/Mirantis/pelagia/pkg/apis/ceph.pelagia.lcm/v1alpha1"
	lcmcommon "github.com/Mirantis/pelagia/pkg/common"
)

var (
	// script templates for cleanup jobs
	cleanupScriptTmpl = `set -xe
%s
`
	diskCleanupScriptTmpl = `# running disk cleanup script part
DEVICE_PATH=%s
ROTATIONAL=%v
: ${DEVICE_PATH?:Device var \$DEVICE_PATH is not set}
echo "cleaning disk '${DEVICE_NAME}'..."
DEVICE_CHILDRENS=$(lsblk ${DEVICE_PATH} -o TYPE,NAME --noheadings -J | jq -r '.blockdevices[].children[]?|.name')
for children in ${DEVICE_CHILDRENS}; do
    dmsetup remove ${children}
    lvDir=/dev/$(echo ${children//-/\/} | sed 's_//_-_g')
    rm -rf ${lvDir} /dev/mapper/${children}
    lvremove ${lvDir}
    [ -d $(dirname ${lvDir}) ] && rmdir --ignore-fail-on-non-empty $(dirname ${lvDir})
done
%s
sgdisk --zap-all ${DEVICE_PATH}
if [[ "${ROTATIONAL}" == "true" ]]; then
    dd if=/dev/zero of="${DEVICE_PATH}" bs=1M count=100 oflag=direct,dsync
else
    blkdiscard "${DEVICE_PATH}"
fi
partprobe ${DEVICE_PATH}
echo "disk '${DEVICE_NAME}' is cleaned up!"
`
	partitionCleanupScriptTmpl = `# running partition cleanup script part
PARTITION=%s
DESTROY=%v
: ${PARTITION?:Partition var \$PARTITION is not set}
if ! test -h "${PARTITION}"; then
    echo "Partition ${PARTITION} is not found, skipping cleanup"
    exit 0
fi
echo "cleaning partition '${PARTITION}' on disk '${DEVICE_NAME}'..."
PARTITION_TYPE=$(lsblk ${PARTITION} -o type -J -p | jq -r '.blockdevices[] | .type')
if [[ "${PARTITION_TYPE}" != "lvm" || "${DESTROY}" != "true" ]]; then
    ceph-volume lvm zap ${PARTITION}
    exit 0
fi

mapper=$(readlink -f ${PARTITION})
dmsetup remove ${mapper}
rm -rf ${PARTITION} ${mapper}
lvremove ${PARTITION}
[ -d $(dirname ${PARTITION}) ] && rmdir --ignore-fail-on-non-empty $(dirname ${PARTITION})
echo "partition '${PARTITION}' on disk '${DEVICE_NAME}' is cleaned up!"
%s
`
	dmSetupTableClean = `# running dm table cleanup script part
DM_NAME=%s
: ${DM_NAME?:Mapper name var \$DM_NAME is not set}
echo "cleaning mapper '${DM_NAME}' from table if possible..."
if ! test -b "${DM_NAME}"; then
    exit 0
fi
MAPPER_NAME=$(dmsetup info -c -o name --noheadings ${DM_NAME})
OPEN_COUNT=$(dmsetup info -c -o open --noheadings ${MAPPER_NAME})
if [[ "${OPEN_COUNT}" == "0" ]]; then
    dmsetup remove ${MAPPER_NAME}
    lvDir=/dev/$(echo ${MAPPER_NAME//-/\/} | sed 's_//_-_g')
    rm -rf ${lvDir} /dev/mapper/${MAPPER_NAME}
    [ -d $(dirname ${lvDir}) ] && rmdir --ignore-fail-on-non-empty $(dirname ${lvDir})
    echo "mapper '${DM_NAME}' from table is cleaned up!"
fi
%s
`
	hostDirectoryCleanupScriptTmpl = `# running host rook dir cleanup script part
HOST_DIRECTORY=%s
if ! test -d "${HOST_DIRECTORY}"; then
   echo "host directory '${HOST_DIRECTORY}' does not exist, skipping remove"
elif test -b "${HOST_DIRECTORY}/block" || test -b "${HOST_DIRECTORY}/block.db"; then
   echo "could not clean up directory ${HOST_DIRECTORY}, which still is in use, skipping"
else
  rm -rf ${HOST_DIRECTORY}
fi
`
)

func (c *cephOsdRemoveConfig) runCleanupJob(host, osdID, hostOsdDirectory string, devices map[string]lcmv1alpha1.DeviceInfo) (string, error) {
	ownerRefs, err := lcmcommon.GetObjectOwnerRef(c.taskConfig.task, c.api.Scheme)
	if err != nil {
		c.log.Error().Err(err).Msg("")
		return "", errors.Wrap(err, "failed to get CephOsdRemoveTask owner refs")
	}
	if c.taskConfig.cephCluster.Status.CephVersion == nil || c.taskConfig.cephCluster.Status.CephVersion.Image == "" {
		return "", errors.New("failed to determine ceph cluster image, no current used image in status")
	}

	podImage := c.taskConfig.cephCluster.Status.CephVersion.Image
	hostDataPath := lcmcommon.DefaultDataDirHostPath
	if c.taskConfig.cephCluster.Spec.DataDirHostPath != "" {
		hostDataPath = c.taskConfig.cephCluster.Spec.DataDirHostPath
	}
	hostPathDirectory := corev1.HostPathDirectory
	volumes := []corev1.Volume{
		{
			Name: "host-dev",
			VolumeSource: corev1.VolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: "/dev",
				},
			},
		},
		// Since Ceph Squid ceph-volume lvm zap changed the approach of parsing
		// device info and now uses udev data to cleaning up a device properly.
		// https://github.com/ceph/ceph/pull/60091
		{
			Name: "run-udev",
			VolumeSource: corev1.VolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: "/run/udev",
					Type: &hostPathDirectory,
				},
			},
		},
	}
	volumeMounts := []corev1.VolumeMount{
		{
			Name:      "host-dev",
			MountPath: "/dev",
		},
		// Since Ceph Squid ceph-volume lvm zap changed the approach of parsing
		// device info and now uses udev data to cleaning up a device properly.
		// https://github.com/ceph/ceph/pull/60091
		{
			Name:      "run-udev",
			MountPath: "/run/udev",
			ReadOnly:  true,
		},
	}
	// double check that we have correct host rook path and osd path within it
	if hostOsdDirectory != "" && hostDataPath != "" && strings.HasPrefix(hostOsdDirectory, hostDataPath) {
		hostDirVolume := corev1.Volume{
			Name: "host-rook",
			VolumeSource: corev1.VolumeSource{
				HostPath: &corev1.HostPathVolumeSource{Path: hostDataPath},
			},
		}
		volumes = append(volumes, hostDirVolume)
		hostDirVolumeMount := corev1.VolumeMount{
			Name:      "host-rook",
			MountPath: hostDataPath,
		}
		volumeMounts = append(volumeMounts, hostDirVolumeMount)
	} else {
		c.log.Warn().Msgf("incorrect rook/osd data host path for osd '%s', host '%s' (rook path: '%s', osd path: '%s'), host dir cleanup will be skipped",
			osdID, host, hostDataPath, hostOsdDirectory)
	}
	podTemplateSpec := corev1.PodTemplateSpec{
		ObjectMeta: metav1.ObjectMeta{
			Name: diskCleanupJobLabel,
		},
		Spec: corev1.PodSpec{
			SecurityContext: &corev1.PodSecurityContext{
				RunAsUser: &[]int64{0}[0],
			},
			NodeSelector: map[string]string{corev1.LabelHostname: host},
			Containers:   []corev1.Container{},
			Volumes:      volumes,
			// it allows to keep failed pods and do not remove them
			// since we dont need restart it allows to inspect failed container logs
			RestartPolicy: corev1.RestartPolicyNever,
		},
	}

	osdIDToUse := osdID
	if isStrayOsdID(osdID) {
		osdIDToUse = strings.ReplaceAll(osdID, "__", "")
	}
	jobName := k8sutil.TruncateNodeName("device-cleanup-job-%s", fmt.Sprintf("%s-%s", host, osdIDToUse))
	jobTimeout := int64(lcmcommon.DiskCleanupTimeout)
	job := &batch.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jobName,
			Namespace: c.taskConfig.task.Namespace,
			Labels: map[string]string{
				"app":          diskCleanupJobLabel,
				"rook-cluster": c.taskConfig.cephCluster.Name,
				"host":         host,
				"osd":          osdIDToUse,
				"task":         c.taskConfig.task.Name,
			},
			OwnerReferences: ownerRefs,
		},
		Spec: batch.JobSpec{
			// no retry is needed, either should be completed in one-shot, either failed
			BackoffLimit:          &[]int32{0}[0],
			ActiveDeadlineSeconds: &jobTimeout,
		},
	}
	brokenData := false
	// sort devices
	devicesList := []string{}
	for device, deviceInfo := range devices {
		if deviceInfo.Partition == "" || deviceInfo.Path == "" {
			brokenData = true
			c.log.Error().Msgf("device info has no required information for job run: partition or device path missed for osd '%s', host '%s'", osdID, host)
			continue
		}
		devName := device
		// k8s labels should not contain '/' symbol, while we have like /dev/sda
		if strings.HasPrefix(devName, "/dev/") {
			devName = strings.ReplaceAll(devName, "/dev/", "")
		}
		job.Labels[devName] = "true"
		devicesList = append(devicesList, device)
	}
	// should not happen, but double check
	if brokenData {
		return jobName, errors.New("partition or device path missed in provided info")
	}

	sort.Strings(devicesList)
	// run pod per disk to clean up
	// clean job is based on official Rook documentation for zapping devices:
	// https://rook.io/docs/rook/v1.6/ceph-teardown.html#zapping-devices
	idx := 1
	for _, device := range devicesList {
		deviceInfo := devices[device]
		cleanupScript := ""
		hostDirCleanUpMacros := ""
		if hostOsdDirectory != "" {
			hostDirCleanUpMacros = fmt.Sprintf(hostDirectoryCleanupScriptTmpl, hostOsdDirectory)
		}
		if !deviceInfo.Alive {
			cleanupScript = fmt.Sprintf(cleanupScriptTmpl, fmt.Sprintf(dmSetupTableClean, deviceInfo.Partition, hostDirCleanUpMacros))
		} else {
			destroyLVM := isLvmRookMade(deviceInfo.Partition) || c.lcmConfig.TaskParams.AllowToRemoveManuallyCreatedLVM
			if deviceInfo.Zap && destroyLVM {
				// before zapping disk remove current partition
				diskZapPart := fmt.Sprintf(diskCleanupScriptTmpl, deviceInfo.Path, deviceInfo.Rotational, hostDirCleanUpMacros)
				cleanupScript = fmt.Sprintf(cleanupScriptTmpl, fmt.Sprintf(partitionCleanupScriptTmpl, deviceInfo.Partition, destroyLVM, diskZapPart))
			} else {
				cleanupScript = fmt.Sprintf(cleanupScriptTmpl, fmt.Sprintf(partitionCleanupScriptTmpl, deviceInfo.Partition, destroyLVM, hostDirCleanUpMacros))
			}
		}
		securityContext := controller.PrivilegedContext(true)
		securityContext.Capabilities = &corev1.Capabilities{Drop: []corev1.Capability{"NET_RAW"}}
		podSpec := corev1.Container{
			Name:            fmt.Sprintf("cleanup-run-%d", idx),
			Image:           podImage,
			SecurityContext: securityContext,
			VolumeMounts:    volumeMounts,
			Command:         []string{"/bin/bash", "-c", cleanupScript},
			Env: []corev1.EnvVar{
				{
					Name:  "DEVICE_NAME",
					Value: device,
				},
				// Since Ceph Squid ceph-volume lvm zap changed the approach of parsing
				// device info and now uses udev data to cleaning up a device properly [1].
				// dmsetup remove and included udev is not syncing properly, so we need to disable
				// udev sync [2] for lvm dmsetup commands to prevent dmsetup hanging.
				// [1] https://github.com/ceph/ceph/pull/60091
				// [2] https://github.com/rook/rook/pull/4320
				{
					Name:  "DM_DISABLE_UDEV",
					Value: "1",
				},
			},
		}
		podTemplateSpec.Spec.Containers = append(podTemplateSpec.Spec.Containers, podSpec)
		idx++
	}
	job.Spec.Template = podTemplateSpec

	c.log.Info().Msgf("creating cleanup job '%s/%s' for osdID '%s', host '%s'", c.taskConfig.task.Namespace, jobName, osdID, host)
	err = c.createCleanupJob(job)
	if err != nil {
		c.log.Error().Err(err).Msg("")
	}
	return jobName, err
}

func (c *cephOsdRemoveConfig) createCleanupJob(job *batch.Job) error {
	_, err := lcmcommon.RunFuncWithRetry(retriesForFailedCommand, commandRetryRunTimeout, func() (interface{}, error) {
		presentJob, err := c.api.Kubeclientset.BatchV1().Jobs(job.Namespace).Get(c.context, job.Name, metav1.GetOptions{})
		if err != nil && !apierrors.IsNotFound(err) {
			c.log.Error().Err(err).Msg("")
			return nil, err
		}
		if err == nil {
			if presentJob.Status.Active > 0 {
				c.log.Warn().Msgf("found old cleanup '%s' job, which is not finished yet, waiting its completion", job.Name)
				return nil, errors.New("waiting old job")
			}
			c.log.Warn().Msgf("found old cleanup '%s' job, trying to remove before new job creation", job.Name)
			propagation := metav1.DeletePropagationForeground
			gracePeriod := int64(0)
			deleteOptions := metav1.DeleteOptions{GracePeriodSeconds: &gracePeriod, PropagationPolicy: &propagation}
			err := c.api.Kubeclientset.BatchV1().Jobs(job.Namespace).Delete(c.context, job.Name, deleteOptions)
			if err != nil {
				c.log.Error().Err(err).Msg("")
				return nil, err
			}
		}
		_, err = c.api.Kubeclientset.BatchV1().Jobs(job.Namespace).Create(c.context, job, metav1.CreateOptions{})
		if err != nil {
			c.log.Error().Err(err).Msg("")
			return nil, err
		}
		return nil, nil
	})
	return err
}

func (c *cephOsdRemoveConfig) getCleanupJob(jobName string) (*batch.Job, error) {
	jobItem, err := lcmcommon.RunFuncWithRetry(retriesForFailedCommand, commandRetryRunTimeout, func() (interface{}, error) {
		job, err := c.api.Kubeclientset.BatchV1().Jobs(c.taskConfig.task.Namespace).Get(c.context, jobName, metav1.GetOptions{})
		if err != nil {
			c.log.Error().Err(err).Msg("")
		}
		return job, err
	})
	return jobItem.(*batch.Job), err
}
