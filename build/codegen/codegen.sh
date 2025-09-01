#!/bin/bash -e

# Copyright 2018 The Rook Authors. All rights reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# shellcheck disable=SC2086,SC2089,SC2090
# Disables quote checks, which is needed because of the SED variable here.

KUBE_CODE_GEN_VERSION="kubernetes-1.17.2"
GROUP_VERSIONS="miraceph:v1alpha1"

scriptdir="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
codegendir="${scriptdir}/../../vendor/k8s.io/code-generator"

# vendoring k8s.io/code-generator temporarily
echo "require k8s.io/code-generator ${KUBE_CODE_GEN_VERSION}" >> ${scriptdir}/../../go.mod
go mod vendor
git checkout HEAD ${scriptdir}/../../go.mod ${scriptdir}/../../go.sum

bash ${codegendir}/generate-groups.sh \
    all \
    github.com/Mirantis/pelagia/pkg/client \
    github.com/Mirantis/pelagia/pkg/apis \
    "${GROUP_VERSIONS}" \
    --output-base "${scriptdir}/../../vendor" \
    --go-header-file "${scriptdir}/boilerplate.go.txt"
cp -r "${scriptdir}/../../vendor/github.com/Mirantis/pelagia/pkg" "${scriptdir}/../../"

rm -rf "${scriptdir}/../../vendor/github.com/Mirantis/pelagia"

SED="sed -i.bak"

# workaround https://github.com/openshift/origin/issues/10357
find "${scriptdir}/../../pkg/client" -name "clientset_generated.go" -exec \
    $SED 's/fakePtr := testing.Fake\([{]\)}/cs := \&Clientset\1}/g' {} +
find "${scriptdir}/../../pkg/client" -name "clientset_generated.go" -exec \
    $SED 's/fakePtr.AddReactor/cs.Fake.AddReactor/g' {} +
find "${scriptdir}/../../pkg/client" -name "clientset_generated.go" -exec \
    $SED 's/fakePtr.AddWatchReactor/cs.Fake.AddWatchReactor/g' {} +
# shellcheck disable=SC1004
# Disables backslash+linefeed is literal check.
find "${scriptdir}/../../pkg/client" -name "clientset_generated.go" -exec \
    $SED 's/return \&Clientset{fakePtr, \&fakediscovery.FakeDiscovery{Fake: \&fakePtr}}/cs.discovery = \&fakediscovery.FakeDiscovery{Fake: \&cs.Fake}\
	return cs/g' {} +
find "${scriptdir}/../../pkg/client" -name "clientset_generated.go.bak" -delete
