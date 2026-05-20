#!/usr/bin/env python3
"""
Collects the pelagia chart (pelagia-ceph) from charts/ and writes its ref to charts.list:
  <registry>/<repo>/<chart-name>:<version>
(no oci:// prefix; make_bundle.sh uses oci: transport when copying.)
Only the main pelagia-ceph chart is included, not dependency charts.

Chart version is taken from VERSION env if set, otherwise from `make get-version`
(run from repo root; same version logic as the rest of the build).

Requires: PyYAML (pip install pyyaml)
Env:
  REPO_ROOT           - repo root (default: parent of build/ parent)
  OCI_CHARTS_REGISTRY - e.g. ghcr.io/owner/pelagia-charts (no oci:// prefix)
  VERSION             - chart version (optional; if unset, runs make get-version)
  OUTPUT_FILE         - output path (default: charts.list in repo root)
"""

import os
import subprocess
import sys
from pathlib import Path

try:
    import yaml
except ImportError:
    sys.stderr.write("Error: PyYAML required. Install with: pip install pyyaml\n")
    sys.exit(1)


def main() -> None:
    script_dir = Path(__file__).resolve().parent
    repo_root = Path(os.environ.get("REPO_ROOT", script_dir.parent.parent))
    charts_dir = repo_root / "charts"
    registry = (os.environ.get("OCI_CHARTS_REGISTRY") or "").strip()
    version = (os.environ.get("VERSION") or "").strip()
    output_file = Path(os.environ.get("OUTPUT_FILE", str(repo_root / "charts.list")))

    if not registry:
        sys.stderr.write("Error: OCI_CHARTS_REGISTRY env is required (e.g. ghcr.io/owner/charts)\n")
        sys.exit(1)
    if not version:
        try:
            result = subprocess.run(
                ["make", "get-version"],
                cwd=repo_root,
                capture_output=True,
                text=True,
                check=True,
            )
            version = (result.stdout or "").strip()
        except (subprocess.CalledProcessError, FileNotFoundError) as e:
            sys.stderr.write(f"Error: VERSION env unset and 'make get-version' failed: {e}\n")
            sys.exit(1)
        if not version:
            sys.stderr.write("Error: VERSION env is required or run from repo with 'make get-version' available\n")
            sys.exit(1)

    refs: list[str] = []
    for chart_path in sorted(charts_dir.iterdir()):
        if not chart_path.is_dir():
            continue
        chart_yaml = chart_path / "Chart.yaml"
        if not chart_yaml.exists():
            continue
        with open(chart_yaml, encoding="utf-8") as f:
            data = yaml.safe_load(f) or {}
        name = (data.get("name") or "").strip()
        if name != "pelagia-ceph":
            continue
        refs.append(f"{registry}/{name}:{version}")

    output_file.write_text("\n".join(refs) + ("\n" if refs else ""))
    print(f"Written {len(refs)} chart ref(s) to {output_file}", file=sys.stderr)


if __name__ == "__main__":
    main()
