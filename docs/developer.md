# Developer Guide

Pelagia uses Makefile to build, test, and deploy the controller. This document provides a guide for developers
with the most used Makefile targets.

For the complete list of available instructions, refer to [Makefile](https://github.com/Mirantis/pelagia/blob/release-1.x/Makefile)

## Code style

Pelagia uses [golangci-lint](https://golangci-lint.run/) code formatter. To check your changes and format them:
```bash
make check
```

To run check changes and auto-fix them:

```bash
make fix
```

## Code tests

Each commit requires all `PASS` unit tests. To run unit tests locally:
```bash
make unit
```

Pelagia has E2E tests that can be useful for verifying code changes. To run them, you
need an environment with Pelagia installed. To be able to run E2E tests, the following
parameters must be specified:

```bash
export TEST_NAMESPACE=ceph-lcm-mirantis
export KUBECONFIG=./kubeconfig
export E2E_TESTCONFIG=cephfs.yaml
```

To run an E2E test:

```bash
make e2e-code
```

## Build the controller binary, image, and chart

Pelagia is deployed as a Helm chart into the Kubernetes cluster.

To build controller binaries used in the controller image:

```bash
make go.build/all
```

To specify the required platform for building:

```bash
export PLATFORMS=linux/amd64,linux/arm64
```

To build the controller Docker image:
```bash
make docker.build
```

To build the Helm chart:
```bash
make
```

## Cleanup development material

To clean up development material, such as built Docker images, Go binaries, Go dependencies, and so on:

```bash
make clean-all
```
