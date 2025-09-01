#!/usr/bin/env bash
set -ex

AIRGAP_BUNDLE_DIR=${AIRGAP_BUNDLE_DIR:-"./bundle/images/"}
FULL_IMAGES_LIST_FILE=${FULL_IMAGES_LIST_FILE:-"images.list"}
FULL_CHARTS_LIST_FILE=${FULL_CHARTS_LIST_FILE:-"charts.list"}
SKOPEO_IMG=${SKOPEO_IMG:-"quay.io/skopeo/stable:v1.18.0"}
DOCKER_CONFIG_PATH=${DOCKER_CONFIG_PATH:-"${HOME}/.docker/config.json"}
AIRGAP_BUNDLE_FILE=${AIRGAP_BUNDLE_FILE:-"airgap-bundle-ceph.tar.gz"}

mkdir -p ${AIRGAP_BUNDLE_DIR}

for image in $(cat ${FULL_IMAGES_LIST_FILE}); do
  image=$(echo ${image} | tr -d '"');
  src="docker://$image";
  dst="oci-archive:$(echo "$image.tar" | tr ':' '@' | tr '/' '&' )";
  echo "Saving image $src to $dst";
  docker run -v ${HOME}/.docker/config.json:/config.json -v ${AIRGAP_BUNDLE_DIR}:/airgap-bundle-dir -w /airgap-bundle-dir ${SKOPEO_IMG} copy --authfile /config.json $src $dst;
done;

for chart in $(cat ${FULL_CHARTS_LIST_FILE}); do
  chart=$(echo ${chart} | tr -d '"');
  src="docker://$chart";
  dst="oci-archive:$(echo "$chart.tar" | tr ':' '@' | tr '/' '&' )";
  echo "Saving chart $src to $dst";
  docker run -v ${DOCKER_CONFIG_PATH}:/config.json -v ${AIRGAP_BUNDLE_DIR}:/airgap-bundle-dir -w /airgap-bundle-dir ${SKOPEO_IMG} copy --authfile /config.json $src $dst;
done;

tar -czf ${AIRGAP_BUNDLE_FILE} -C ${AIRGAP_BUNDLE_DIR} .
