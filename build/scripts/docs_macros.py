"""
mkdocs-macros module for the Pelagia docs.

Exposes the following variables to Markdown pages:

    rook_docs - base URL of the Rook documentation for the Rook release that
                Pelagia is built with, e.g. https://rook.io/docs/rook/v1.20
    ceph_docs - base URL of the Ceph documentation for the default Ceph release
                that Pelagia deploys, e.g. https://docs.ceph.com/en/tentacle

The Rook release is taken from the github.com/rook/rook requirement in go.mod.
The Ceph release is the one whose image tag in the pelagia-ceph chart values
matches the "latest" tag. So links follow both versions automatically on every
release branch.

Usage in Markdown:
    [CephCluster CRD]({{ rook_docs }}/CRDs/Cluster/ceph-cluster-crd/)
    [CRUSH Maps]({{ ceph_docs }}/rados/operations/crush-map/)
"""

import os
import re

import yaml

ROOK_MODULE_RE = re.compile(r"^\s*github\.com/rook/rook\s+v(\d+)\.(\d+)\.\d+", re.MULTILINE)


def rook_release(go_mod_path):
    with open(go_mod_path) as f:
        match = ROOK_MODULE_RE.search(f.read())
    if not match:
        raise ValueError(f"github.com/rook/rook requirement not found in {go_mod_path}")
    return f"v{match.group(1)}.{match.group(2)}"


def ceph_release(values_path):
    with open(values_path) as f:
        tags = yaml.safe_load(f)["images"]["ceph"]["tag"]
    releases = [name for name, tag in tags.items() if name != "latest" and tag == tags["latest"]]
    if len(releases) != 1:
        raise ValueError(f"expected exactly one Ceph release with the 'latest' image tag in {values_path}, found {releases}")
    return releases[0]


def define_env(env):
    rook = rook_release(os.path.join(env.project_dir, "go.mod"))
    ceph = ceph_release(os.path.join(env.project_dir, "charts", "pelagia-ceph", "values.yaml"))
    env.variables["rook_docs"] = f"https://rook.io/docs/rook/{rook}"
    env.variables["ceph_docs"] = f"https://docs.ceph.com/en/{ceph}"
