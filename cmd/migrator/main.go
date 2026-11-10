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

package main

import (
	"context"
	"flag"
	"os"
	"strings"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/rest"

	lcmversion "github.com/Mirantis/pelagia/v3/codeversion"
	lcmcommon "github.com/Mirantis/pelagia/v3/pkg/common"
)

func getCurrentNamespace() string {
	// Standard path where Kubernetes injects the namespace file inside pods
	nsFile := "/var/run/secrets/kubernetes.io/serviceaccount/namespace"
	data, err := os.ReadFile(nsFile)
	if err != nil {
		return "default"
	}
	return strings.TrimSpace(string(data))
}

func createPodEvent(ctx context.Context, clientset kubernetes.Interface, namespace, podName, reason, message string) error {
	if namespace == "" || podName == "" {
		return errors.New("pod name and namespace must be specified")
	}
	now := metav1.NowMicro()
	event := &corev1.Event{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podName,
			Namespace: namespace,
		},
		InvolvedObject: corev1.ObjectReference{
			APIVersion: "v1",
			Kind:       "Pod",
			Name:       podName,
			Namespace:  namespace,
		},
		Type:                corev1.EventTypeNormal,
		Reason:              reason,
		Message:             message,
		FirstTimestamp:      metav1.NewTime(now.Time),
		LastTimestamp:       metav1.NewTime(now.Time),
		EventTime:           now,
		Count:               1,
		Action:              reason,
		ReportingController: "pelagia-migrator",
		ReportingInstance:   podName,
		Source:              corev1.EventSource{Component: "pelagia-migrator"},
	}
	if _, err := clientset.CoreV1().Events(namespace).Create(ctx, event, metav1.CreateOptions{}); err != nil {
		return err
	}
	return nil
}

func main() {
	log := lcmcommon.InitLogger(false)
	log.Info().Msg(lcmversion.GetCodeVersion("Hook migrator"))
	log.Info().Msg(lcmversion.GetGoRuntimeVersion())

	var rookNamespace string
	flag.StringVar(&rookNamespace, "rook-namespace", "rook-ceph", "rook namespace")
	flag.Parse()

	config, err := rest.InClusterConfig()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to initialize client config")
		os.Exit(1)
	}
	kubeClientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to get client config")
		os.Exit(1)
	}
	podNamespace := getCurrentNamespace()
	podName := os.Getenv("HOSTNAME")

	context := context.TODO()
	csiImagesConfigMapName := "rook-csi-operator-image-set-configmap"
	migrationMapName := "rook-csi-operator-image-set-configmap-migration"
	migrationMap, err := kubeClientset.CoreV1().ConfigMaps(rookNamespace).Get(context, migrationMapName, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			log.Info().Msgf("migration ConfigMap '%s/%s' is not found, skipping", rookNamespace, migrationMapName)
			eventErr := createPodEvent(context, kubeClientset, podNamespace, podName, "CsiMapMigrationHookSkipped", "Ceph CSI images migration map is not present")
			if eventErr != nil {
				log.Warn().Msgf("failed to create event: %v", eventErr)
			}
			os.Exit(0)
		}
		log.Fatal().Err(err).Msgf("failed to get migration ConfigMap '%s/%s'", rookNamespace, migrationMapName)
		os.Exit(1)
	}
	csiImageSetMap, err := kubeClientset.CoreV1().ConfigMaps(rookNamespace).Get(context, csiImagesConfigMapName, metav1.GetOptions{})
	if err != nil {
		if !apierrors.IsNotFound(err) {
			log.Fatal().Err(err).Msgf("failed to get ConfigMap '%s/%s'", rookNamespace, csiImagesConfigMapName)
			os.Exit(1)
		}
		newMap := migrationMap.DeepCopy()
		newMap.ObjectMeta = metav1.ObjectMeta{
			Name:        csiImagesConfigMapName,
			Namespace:   rookNamespace,
			Labels:      migrationMap.Labels,
			Annotations: migrationMap.Annotations,
		}
		log.Info().Msgf("creating ConfigMap '%s/%s'", rookNamespace, csiImagesConfigMapName)
		_, err := kubeClientset.CoreV1().ConfigMaps(rookNamespace).Create(context, newMap, metav1.CreateOptions{})
		if err != nil {
			log.Fatal().Err(err).Msgf("failed to create ConfigMap '%s/%s'", rookNamespace, csiImagesConfigMapName)
			os.Exit(1)
		}
		eventErr := createPodEvent(context, kubeClientset, podNamespace, podName, "CsiMapMigrationHookCreated", "Ceph CSI images map created successfully")
		if eventErr != nil {
			log.Warn().Msgf("failed to create event: %v", eventErr)
		}
		os.Exit(0)
	}

	updated := false
	annotationsDiff, _ := lcmcommon.GetObjectDiff(csiImageSetMap.Annotations, migrationMap.Annotations)
	if annotationsDiff != "" {
		log.Info().Msgf("updating annotations:\n%s", annotationsDiff)
		csiImageSetMap.Annotations = migrationMap.Annotations
		updated = true
	}
	labelsDiff, _ := lcmcommon.GetObjectDiff(csiImageSetMap.Labels, migrationMap.Labels)
	if labelsDiff != "" {
		log.Info().Msgf("updating labels:\n%s", labelsDiff)
		csiImageSetMap.Labels = migrationMap.Labels
		updated = true
	}
	dataDiff, _ := lcmcommon.GetObjectDiff(csiImageSetMap.Data, migrationMap.Data)
	if dataDiff != "" {
		log.Info().Msgf("updating images data:\n%s", dataDiff)
		csiImageSetMap.Data = migrationMap.Data
		updated = true
	}
	if updated {
		_, err = kubeClientset.CoreV1().ConfigMaps(rookNamespace).Update(context, csiImageSetMap, metav1.UpdateOptions{})
		if err != nil {
			log.Fatal().Err(err).Msgf("failed to update ConfigMap '%s/%s'", rookNamespace, csiImageSetMap.Name)
			os.Exit(1)
		}
		eventErr := createPodEvent(context, kubeClientset, podNamespace, podName, "CsiMapMigrationHookCompleted", "Ceph CSI images map migrated successfully")
		if eventErr != nil {
			log.Warn().Msgf("failed to create event: %v", eventErr)
		}
		log.Info().Msgf("content for ConfigMap '%s/%s' has been migrated", rookNamespace, csiImageSetMap.Name)
	} else {
		eventErr := createPodEvent(context, kubeClientset, podNamespace, podName, "CsiMapMigrationHookSkipped", "Ceph CSI images map is already migrated")
		if eventErr != nil {
			log.Warn().Msgf("failed to create event: %v", eventErr)
		}
		log.Info().Msgf("content for ConfigMap '%s/%s' is up to date, no migration required", rookNamespace, csiImageSetMap.Name)
	}
}
