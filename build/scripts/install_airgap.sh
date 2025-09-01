#!/usr/bin/env bash
set -ex

if [[ -z "$REGISTRY_ADDRESS" ]]; then
    echo "Must provide REGISTRY_ADDRESS in environment" 1>&2
    exit 1
fi
if [[ -z "$REGISTRY_USERNAME" ]]; then
    echo "Must provide REGISTRY_USERNAME in environment" 1>&2
    exit 1
fi
if [[ -z "$REGISTRY_PASSWORD" ]]; then
    echo "Must provide REGISTRY_PASSWORD in environment" 1>&2
    exit 1
fi

if ! [ -x "$(command -v skopeo)" ]; then
  echo 'Error: skopeo is not installed.' >&2
  exit 1
fi

BUNDLE_NAME=${BUNDLE_NAME:-"airgap-bundle-ceph.tar.gz"}
REGISTRY_PROJECT_PATH=ceph
REGISTRY_TLS_VERIFY=${REGISTRY_TLS_VERIFY:-"true"}

# Login to the registry
skopeo login "$REGISTRY_ADDRESS" -u "$REGISTRY_USERNAME" -p "$REGISTRY_PASSWORD" --tls-verify=$REGISTRY_TLS_VERIFY

# Extract the bundle
mkdir -p ./bundle
tar -xzf "$BUNDLE_NAME" -C ./bundle

# Iterate over bundle artifacts and upload each one using skopeo
for archive in $(find ./bundle -print | grep ".tar"); do
  # Form the image name from the archive name
  img=$(basename "$archive" | sed 's~\.tar~~' | tr '&' '/' | tr '@' ':'| cut -d "/" -f 3-);
  echo "Uploading $img";
  # Copy artifact from local oci archive to the registry
  skopeo copy --dest-tls-verify=$REGISTRY_TLS_VERIFY -q "oci-archive:$archive" "docker://$REGISTRY_ADDRESS/$REGISTRY_PROJECT_PATH/$img";
done;

rm -r ./bundle || true
