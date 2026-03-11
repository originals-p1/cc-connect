#!/usr/bin/env python3

import json
import subprocess
import sys
import tempfile
from pathlib import Path


def main() -> int:
    root = Path(__file__).resolve().parent.parent
    script = root / "scripts" / "generate_repo_index.py"

    with tempfile.TemporaryDirectory() as tmpdir:
        out = Path(tmpdir) / "repo-index.json"
        subprocess.run(
            [sys.executable, str(script), "--root", str(root), "--out", str(out)],
            check=True,
        )

        data = json.loads(out.read_text())
        assert data["repo"]["name"] == "cc-connect"
        assert data["repo"]["entrypoint"] == "cmd/cc-connect/main.go"
        assert data["docs"]["agent_guide"] == "AGENTS.md"
        assert any(area["path"] == "core" for area in data["areas"])
        assert any(item["path"] == "core/engine.go" for item in data["key_files"])

        subprocess.run(
            [sys.executable, str(script), "--root", str(root), "--out", str(out), "--check"],
            check=True,
        )

        out.write_text(out.read_text().replace('"name": "cc-connect"', '"name": "stale-index"'))
        stale = subprocess.run(
            [sys.executable, str(script), "--root", str(root), "--out", str(out), "--check"],
            check=False,
            capture_output=True,
            text=True,
        )
        assert stale.returncode != 0
        assert "make generate-repo-index" in stale.stderr

    print("test_generate_repo_index.py: PASS")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
