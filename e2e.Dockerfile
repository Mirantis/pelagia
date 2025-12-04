FROM --platform=$TARGETPLATFORM golang:1.23.6

ARG TARGETOS
ARG TARGETARCH
ARG REF=local

LABEL org.opencontainers.image.authors="Mirantis Inc. https://mirantis.com" \
      org.opencontainers.image.source="https://github.com/Mirantis/pelagia" \
      org.opencontainers.image.ref.commit="${REF}"

# rebuild=1
ARG E2E_PATH=/root/pelagia-e2e
ARG TESTCONFIG_PATH=$E2E_PATH/testconfig

ENV E2E_RUN_PATH=$E2E_PATH
ENV E2E_TESTCONFIG_DIR=$TESTCONFIG_PATH
ENV CEPH_E2E_NAME=pelagia-e2e

COPY test/e2e/testconfigs $TESTCONFIG_PATH/testconfigs
COPY build/bin/${TARGETOS}/${TARGETARCH}/$CEPH_E2E_NAME /usr/local/bin/$CEPH_E2E_NAME
COPY e2e.mk $E2E_PATH/Makefile

RUN apt update -y && \
    apt install unzip -y && \
    case "$TARGETARCH" in \
      amd64) AWS_TARBALL=https://awscli.amazonaws.com/awscli-exe-linux-x86_64.zip ;; \
      arm64) AWS_TARBALL=https://awscli.amazonaws.com/awscli-exe-linux-aarch64.zip ;; \
      *) echo "Unsupported TARGETARCH=${TARGETARCH}" && exit 1 ;; \
    esac && \
    curl "${AWS_TARBALL}" -o "awscliv2.zip" && \
    unzip awscliv2.zip && \
    bash aws/install && \
    rm -r aws && rm awscliv2.zip

WORKDIR $E2E_PATH
