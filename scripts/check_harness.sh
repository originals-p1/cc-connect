#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="${1:-$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)}"
MAIN_FILE="$ROOT_DIR/cmd/cc-connect/main.go"
MODULE_PATH="github.com/chenhg5/cc-connect"

if [[ ! -f "$MAIN_FILE" ]]; then
  echo "missing main file: $MAIN_FILE" >&2
  exit 1
fi

list_dirs() {
  local base="$1"
  if [[ ! -d "$base" ]]; then
    return 0
  fi
  find "$base" -mindepth 1 -maxdepth 1 -type d -printf '%f\n' | sort
}

check_imports() {
  local kind="$1"
  local dir="$2"
  local missing=0
  local name

  while IFS= read -r name; do
    [[ -z "$name" ]] && continue
    if ! grep -Fq "_ \"$MODULE_PATH/$kind/$name\"" "$MAIN_FILE"; then
      echo "missing blank import for $kind/$name in $MAIN_FILE" >&2
      missing=1
    fi
  done < <(list_dirs "$dir")

  return "$missing"
}

check_imports "agent" "$ROOT_DIR/agent"
agent_status=$?
check_imports "platform" "$ROOT_DIR/platform"
platform_status=$?

python3 "$ROOT_DIR/scripts/generate_repo_index.py" --root "$ROOT_DIR" --check
repo_index_status=$?

if [[ $agent_status -ne 0 || $platform_status -ne 0 || $repo_index_status -ne 0 ]]; then
  exit 1
fi

echo "check-harness: PASS"
