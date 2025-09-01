# Developer Guide

Pelagia uses Makefile to build, test, and deploy the controller. This document provides a guide for developers
with the most used Makefile targets.

## Code style

Pelagia uses [golangci-lint](https://golangci-lint.run/) code formatter. To check your changes and format them:
```bash
make check
```

## Tests

Each commit requires all `PASS` unit tests. To run unit tests locally:
```bash
make unit
```

## Build the controller image and chart

Pelagia is deployed as a Helm chart into the Kubernetes cluster.

To build the controller image for the `linux/amd64` platform:
```bash
make build image
```

To build the Helm chart after the controller image is built:
```bash
make
```
