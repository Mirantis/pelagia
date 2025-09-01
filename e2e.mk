# e2e envvars
export CEPH_E2E_NAME ?= pelagia-e2e
export TEST_NAMESPACE ?= ceph-lcm
export KUBECONFIG ?= ./kubeconfig
export EXPORT_DIR ?= ./export
export E2E_TESTCONFIG_DIR ?= ./testconfig
export E2E_TESTCONFIG ?= .ceph-e2e-config.yaml

.PHONY: e2e
e2e:
	echo "Using ${E2E_TESTCONFIG} configset.."
	$(CEPH_E2E_NAME) -test.v -test.timeout 0
