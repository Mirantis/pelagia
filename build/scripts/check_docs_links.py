#!/usr/bin/env python3
"""
Checks that the docs link to the Rook and Ceph documentation through the version
variables from mkdocs.yml (rook_doc_version, ceph_doc_version) instead of a
hard-coded or 'latest' version, for example:

    https://rook.io/docs/rook/{{ rook_doc_version }}/CRDs/Cluster/ceph-cluster-crd/
    https://docs.ceph.com/en/{{ ceph_doc_version }}/rados/operations/crush-map/

Pages under upgrade-guide/ describe specific Rook and Ceph releases, so they may
link to them directly.

Runs as an MkDocs hook (see `hooks` in mkdocs.yml), so every docs build fails on
such links. The hook checks the Markdown files on disk, before plugins such as
macros render the version variables.

Usage: python3 build/scripts/check_docs_links.py [DOCS_DIR]
       DOCS_DIR defaults to docs/ in the repo root. Exits with 1 if links are found.
"""

import re
import sys
from pathlib import Path

EXCLUDED_DIRS = ("upgrade-guide",)

URL_RE = re.compile(r"https?://[^\s)\]>\"'`]+")

# (pattern of a hard-coded docs URL, the form to use instead)
HARDCODED_DOCS_URLS = (
    (
        re.compile(r"rook\.github\.io/docs|rook\.io/docs/rook/(?:latest|v\d)"),
        "https://rook.io/docs/rook/{{ rook_doc_version }}/...",
    ),
    (
        re.compile(r"docs\.ceph\.com/(?:docs/|en/[a-z])"),
        "https://docs.ceph.com/en/{{ ceph_doc_version }}/...",
    ),
)


def find_hardcoded_links(docs_dir) -> list[str]:
    """Return a 'path:line: url' message for every hard-coded Rook or Ceph docs link under docs_dir."""
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


def on_pre_build(config, **kwargs):
    """MkDocs hook: fail the build on hard-coded Rook or Ceph docs links."""
    from mkdocs.exceptions import PluginError

    problems = find_hardcoded_links(config["docs_dir"])
    if problems:
        raise PluginError(
            "Hard-coded Rook or Ceph documentation links found. "
            "Use the version variables from mkdocs.yml instead:\n" + "\n".join(problems)
        )


def main() -> int:
    docs_dir = sys.argv[1] if len(sys.argv) > 1 else Path(__file__).resolve().parents[2] / "docs"
    problems = find_hardcoded_links(docs_dir)
    for problem in problems:
        print(problem, file=sys.stderr)
    return 1 if problems else 0


if __name__ == "__main__":
    sys.exit(main())
