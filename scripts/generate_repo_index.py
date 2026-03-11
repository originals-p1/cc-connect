#!/usr/bin/env python3

from __future__ import annotations

import argparse
import json
import sys
from pathlib import Path


AREA_ROLES = {
    "cmd/cc-connect": "CLI entrypoint and subcommands",
    "core": "Engine, sessions, routing, events, commands, skills, relay, speech, cron",
    "config": "TOML schema, validation, defaults, persistence helpers",
    "agent": "Agent integrations",
    "platform": "Chat platform integrations",
    "daemon": "Daemon and service management",
    "docs": "Guides, architecture context, harness docs, plans",
}

DOC_PATHS = {
    "agent_guide": "AGENTS.md",
    "architecture": "ARCHITECTURE.md",
    "validation": "docs/VALIDATION.md",
    "task_recipes": "docs/TASK_RECIPES.md",
    "deep_context": "docs/AI_CONTEXT.md",
    "harness_index": "docs/harness/index.md",
    "plans_root": "docs/exec-plans",
    "exec_plans_active": "docs/exec-plans/active",
    "exec_plans_completed": "docs/exec-plans/completed",
}

KEY_FILES = [
    {
        "path": "cmd/cc-connect/main.go",
        "role": "Program entrypoint and blank imports for agents/platforms",
    },
    {
        "path": "core/interfaces.go",
        "role": "Core interfaces for Platform, Agent, AgentSession",
    },
    {
        "path": "core/engine.go",
        "role": "Message routing, command handling, session orchestration",
    },
    {
        "path": "core/session.go",
        "role": "Session persistence and lifecycle",
    },
    {
        "path": "config/config.go",
        "role": "Configuration load, validation, and persistence",
    },
]

TASK_TYPES = ["bugfix", "feature", "refactor", "perf", "docs", "analysis"]


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Generate docs/generated/repo-index.json")
    parser.add_argument(
        "--root",
        default=Path(__file__).resolve().parent.parent,
        type=Path,
        help="Repository root directory",
    )
    parser.add_argument(
        "--out",
        default=Path("docs/generated/repo-index.json"),
        type=Path,
        help="Output file path (absolute or relative to --root)",
    )
    parser.add_argument(
        "--check",
        action="store_true",
        help="Validate that the output file matches generated content without rewriting it",
    )
    return parser.parse_args()


def resolve_out(root: Path, out: Path) -> Path:
    if out.is_absolute():
        return out
    return root / out


def ensure_exists(root: Path, relpath: str) -> None:
    if not (root / relpath).exists():
        raise FileNotFoundError(f"required path missing: {relpath}")


def build_index(root: Path) -> dict:
    for relpath in DOC_PATHS.values():
        ensure_exists(root, relpath)

    for item in KEY_FILES:
        ensure_exists(root, item["path"])

    areas = [
        {"path": path, "role": role}
        for path, role in AREA_ROLES.items()
        if (root / path).exists()
    ]

    return {
        "repo": {
            "name": root.name,
            "language": "Go",
            "entrypoint": "cmd/cc-connect/main.go",
            "purpose": "Bridge local AI coding agents to chat platforms.",
        },
        "docs": DOC_PATHS,
        "areas": areas,
        "key_files": KEY_FILES,
        "task_types": TASK_TYPES,
    }


def main() -> int:
    args = parse_args()
    root = args.root.resolve()
    out = resolve_out(root, args.out)
    data = build_index(root)
    rendered = json.dumps(data, indent=2) + "\n"

    if args.check:
        if not out.exists():
            print(
                f"repo index is missing: {out}\nRun `make generate-repo-index`.",
                file=sys.stderr,
            )
            return 1
        current = out.read_text(encoding="utf-8")
        if current != rendered:
            print(
                f"repo index is stale: {out}\nRun `make generate-repo-index`.",
                file=sys.stderr,
            )
            return 1
        print(f"repo index is up to date: {out}")
        return 0

    out.parent.mkdir(parents=True, exist_ok=True)
    out.write_text(rendered, encoding="utf-8")
    print(f"wrote {out}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
