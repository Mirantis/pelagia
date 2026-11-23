#!/usr/bin/env python3
"""
Checks that the docs link to the Rook and Ceph documentation through the version
variables from mkdocs.yml (rook_doc_version, ceph_doc_version) instead of a
hard-coded or 'latest' version, for example:

    https://rook.io/docs/rook/{{ rook_doc_version }}/CRDs/Cluster/ceph-cluster-crd/
    https://docs.ceph.com/en/{{ ceph_doc_version }}/rados/operations/crush-map/

Links to the sources of these docs on GitHub (Documentation/ in rook/rook, doc/
in ceph/ceph) are rejected too: they point to a branch or tag, not to
rook_doc_version or ceph_doc_version.

Links to files in the Pelagia GitHub project must use the pelagia_branch
variable from mkdocs.yml, so that each docs version links to the branch it is
built from, for example:

    https://github.com/Mirantis/pelagia/blob/{{ pelagia_branch }}/charts/rook/values.yaml

Links to files in the Ceph CSI Operator GitHub project must use the
ceph_csi_operator_version variable from mkdocs.yml, for example:

    https://github.com/ceph/ceph-csi-operator/blob/{{ ceph_csi_operator_version }}/docs/design/operator.md

ceph_csi_operator_version must match appVersion in
charts/ceph-csi-operator/Chart.yaml, so that these links follow the Ceph CSI
Operator release that Pelagia deploys. The check is skipped if the variable or
the chart is missing.

Pages under upgrade-guide/ describe specific Rook, Ceph, Ceph CSI Operator, and
Pelagia releases, so they may link to them directly.

Runs as an MkDocs hook (see `hooks` in mkdocs.yml), so every docs build fails on
such links. The hook checks the Markdown files on disk, before plugins such as
macros render the version variables.

Usage: python3 build/scripts/check_docs_links.py [DOCS_DIR]
       DOCS_DIR defaults to docs/ in the repo root. Exits with 1 if links are found
       or ceph_csi_operator_version in mkdocs.yml differs from the chart.
"""

import re
import sys
from pathlib import Path

REPO_ROOT = Path(__file__).resolve().parents[2]
CSI_OPERATOR_CHART = Path("charts/ceph-csi-operator/Chart.yaml")

EXCLUDED_DIRS = ("upgrade-guide",)

URL_RE = re.compile(r"https?://[^\s)\]>\"'`]+")

# (pattern of a hard-coded docs URL, the form to use instead)
HARDCODED_DOCS_URLS = (
    (
        re.compile(
            r"rook\.github\.io/docs"
            r"|rook\.io/docs/rook/(?:latest|v\d)"
            r"|github\.com/rook/rook/(?:blob|tree)/[^/]+/Documentation/"
        ),
        "https://rook.io/docs/rook/{{ rook_doc_version }}/...",
    ),
    (
        re.compile(
            r"docs\.ceph\.com/(?:docs/|en/[a-z])"
            r"|github\.com/ceph/ceph/(?:blob|tree)/[^/]+/doc/"
        ),
        "https://docs.ceph.com/en/{{ ceph_doc_version }}/...",
    ),
    (
        # URL_RE stops at the space in '{{ pelagia_branch }}', so a link that
        # uses the variable is matched up to '.../blob/{{' and is skipped here.
        re.compile(r"github\.com/Mirantis/pelagia/(?:blob|tree)/(?!\{\{)", re.IGNORECASE),
        "https://github.com/Mirantis/pelagia/blob/{{ pelagia_branch }}/...",
    ),
    (
        re.compile(r"github\.com/ceph/ceph-csi-operator/(?:blob|tree)/(?!\{\{)", re.IGNORECASE),
        "https://github.com/ceph/ceph-csi-operator/blob/{{ ceph_csi_operator_version }}/...",
    ),
)


def find_hardcoded_links(docs_dir) -> list[str]:
    """Return a 'path:line: url' message for every hard-coded Rook, Ceph, Pelagia, or Ceph CSI Operator link under docs_dir."""
    docs_dir = Path(docs_dir)
    problems = []
    for path in sorted(docs_dir.rglob("*.md")):
        rel = path.relative_to(docs_dir)
        if rel.parts[0] in EXCLUDED_DIRS:
            continue
        for lineno, line in enumerate(path.read_text(encoding="utf-8").splitlines(), 1):
            for url in URL_RE.findall(line):
                for pattern, expected in HARDCODED_DOCS_URLS:
                    if pattern.search(url):
                        problems.append(f"{rel}:{lineno}: {url} - use '{expected}'")
    return problems


def read_yaml_scalar(path, key):
    """Return the value of the first 'key: value' line in a YAML file, or None.

    A regex instead of a YAML parser, because mkdocs.yml has !!python/name tags
    that yaml.safe_load rejects.
    """
    match = re.search(
        rf"^\s*{re.escape(key)}:\s*[\"']?([^\"'\s#]+)",
        Path(path).read_text(encoding="utf-8"),
        re.MULTILINE,
    )
    return match.group(1) if match else None


def check_csi_operator_version(docs_version, repo_root=REPO_ROOT) -> list[str]:
    """Return a message if ceph_csi_operator_version differs from appVersion of the ceph-csi-operator chart."""
    chart = Path(repo_root) / CSI_OPERATOR_CHART
    if docs_version is None or not chart.is_file():
        return []
    app_version = read_yaml_scalar(chart, "appVersion")
    if app_version is None:
        return []
    expected = "v" + app_version.lstrip("v")
    if str(docs_version) != expected:
        return [
            f"mkdocs.yml: ceph_csi_operator_version is '{docs_version}', but {CSI_OPERATOR_CHART} "
            f"has appVersion '{app_version}' - set ceph_csi_operator_version to '{expected}'"
        ]
    return []


def on_pre_build(config, **kwargs):
    """MkDocs hook: fail the build on hard-coded links or an outdated ceph_csi_operator_version."""
    from mkdocs.exceptions import PluginError

    problems = find_hardcoded_links(config["docs_dir"])
    problems += check_csi_operator_version(config["extra"].get("ceph_csi_operator_version"))
    if problems:
        raise PluginError(
            "Hard-coded Rook, Ceph, Pelagia, or Ceph CSI Operator links found, or a version variable is outdated. "
            "Use the version variables from mkdocs.yml and keep them up to date:\n" + "\n".join(problems)
        )


def main() -> int:
    docs_dir = sys.argv[1] if len(sys.argv) > 1 else REPO_ROOT / "docs"
    problems = find_hardcoded_links(docs_dir)
    problems += check_csi_operator_version(read_yaml_scalar(REPO_ROOT / "mkdocs.yml", "ceph_csi_operator_version"))
    for problem in problems:
        print(problem, file=sys.stderr)
    return 1 if problems else 0


if __name__ == "__main__":
    sys.exit(main())
