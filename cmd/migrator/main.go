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

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/rest"

	lcmversion "github.com/Mirantis/pelagia/v3/codeversion"
	lcmcommon "github.com/Mirantis/pelagia/v3/pkg/common"
)

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
	context := context.TODO()
	csiImagesConfigMapName := "rook-csi-operator-image-set-configmap"
	migrationMapName := "rook-csi-operator-image-set-configmap-migration"
	migrationMap, err := kubeClientset.CoreV1().ConfigMaps(rookNamespace).Get(context, migrationMapName, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			log.Info().Msgf("migration ConfigMap '%s/%s' is not found, skipping", rookNamespace, migrationMapName)
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
		log.Info().Msgf("content for ConfigMap '%s/%s' has been migrated", rookNamespace, csiImageSetMap.Name)
	} else {
		log.Info().Msgf("content for ConfigMap '%s/%s' is up to date, no migration required", rookNamespace, csiImageSetMap.Name)
	}
}
