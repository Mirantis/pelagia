"""
mkdocs-macros module for the Pelagia docs.

Exposes the following variables to Markdown pages:

    rook_docs - base URL of the Rook documentation for the Rook release that
                Pelagia is built with, e.g. https://rook.io/docs/rook/v1.20

The Rook release is taken from the github.com/rook/rook requirement in go.mod,
so links follow the Rook version automatically on every release branch.

Usage in Markdown:
    [CephCluster CRD]({{ rook_docs }}/CRDs/Cluster/ceph-cluster-crd/)
"""

import os
import re

ROOK_MODULE_RE = re.compile(r"^\s*github\.com/rook/rook\s+v(\d+)\.(\d+)\.\d+", re.MULTILINE)


def rook_release(go_mod_path):
    with open(go_mod_path) as f:
        match = ROOK_MODULE_RE.search(f.read())
    if not match:
        raise ValueError(f"github.com/rook/rook requirement not found in {go_mod_path}")
    return f"v{match.group(1)}.{match.group(2)}"


def define_env(env):
    release = rook_release(os.path.join(env.project_dir, "go.mod"))
    env.variables["rook_docs"] = f"https://rook.io/docs/rook/{release}"
