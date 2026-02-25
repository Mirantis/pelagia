#!/usr/bin/env python3
"""
Collects all container images in format registry/repo/image:tag from the
"images" section (including nested subsections) of values of all Helm charts
under charts/ and writes them to images.list (unique, sorted).

Parses values.yaml directly so all repository+tag pairs are collected even
when tag is an object (e.g. latest/squid/tentacle).

Registry prefix is taken from IMAGE_REGISTRY env (e.g. set from REGISTRY in CI).
Requires: PyYAML (pip install pyyaml)
Usage: run from repo root, or set REPO_ROOT env to the repo root.
       Set IMAGE_REGISTRY for the image registry prefix (e.g. registry.example.com).
       Override output with OUTPUT_FILE env (default: images.list in repo root).
"""

from __future__ import annotations

import os
import sys
from pathlib import Path

try:
    import yaml
except ImportError:
    sys.stderr.write("Error: PyYAML required. Install with: pip install pyyaml\n")
    sys.exit(1)


def collect_images_from_obj(obj: dict, registry: str, out: list[str]) -> None:
    """Recursively collect registry/repo:tag from nodes that have repository and tag."""
    if not isinstance(obj, dict):
        return
    if "repository" in obj and "tag" in obj:
        repo = str(obj["repository"]).strip()
        if not repo:
            return
        tag = obj["tag"]
        if isinstance(tag, str):
            t = tag.strip()
            if t:
                ref = f"{registry}/{repo}:{t}" if registry else f"{repo}:{t}"
                out.append(ref)
        elif isinstance(tag, dict):
            for v in tag.values():
                if isinstance(v, str):
                    t = v.strip()
                    if t:
                        ref = f"{registry}/{repo}:{t}" if registry else f"{repo}:{t}"
                        out.append(ref)
    for v in obj.values():
        collect_images_from_obj(v, registry, out)


def parse_values_file(path: Path, registry: str) -> list[str]:
    """Parse a values.yaml and return list of image references (registry from IMAGE_REGISTRY)."""
    with open(path, encoding="utf-8") as f:
        data = yaml.safe_load(f) or {}
    out: list[str] = []
    if "images" in data:
        collect_images_from_obj(data["images"], registry, out)
    return out


def main() -> None:
    script_dir = Path(__file__).resolve().parent
    repo_root = Path(os.environ.get("REPO_ROOT", script_dir.parent.parent))
    charts_dir = repo_root / "charts"
    output_file = Path(os.environ.get("OUTPUT_FILE", str(repo_root / "images.list")))
    registry = (os.environ.get("IMAGE_REGISTRY") or "").strip()

    all_images: list[str] = []
    for chart_path in sorted(charts_dir.iterdir()):
        if not chart_path.is_dir():
            continue
        chart_yaml = chart_path / "Chart.yaml"
        values_yaml = chart_path / "values.yaml"
        if not chart_yaml.exists() or not values_yaml.exists():
            continue
        print(f"Processing chart: {chart_path}", file=sys.stderr)
        try:
            all_images.extend(parse_values_file(values_yaml, registry))
        except Exception as e:
            print(f"Warning: failed to parse {values_yaml}: {e}", file=sys.stderr)

    lines = sorted(set(s for s in all_images if s.strip()))
    output_file.write_text("\n".join(lines) + ("\n" if lines else ""))
    print(f"Written {len(lines)} unique image(s) to {output_file}", file=sys.stderr)


if __name__ == "__main__":
    main()
