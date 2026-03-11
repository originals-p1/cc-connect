#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
CHECK_SCRIPT="$ROOT_DIR/scripts/check_harness.sh"

make_repo() {
  local dir="$1"
  mkdir -p "$dir/agent/foo" "$dir/platform/bar" "$dir/cmd/cc-connect"
}

write_main_with_imports() {
  local dir="$1"
  cat >"$dir/cmd/cc-connect/main.go" <<'EOF'
package main

import (
  _ "github.com/chenhg5/cc-connect/agent/foo"
  _ "github.com/chenhg5/cc-connect/platform/bar"
)
EOF
}

write_main_missing_platform() {
  local dir="$1"
  cat >"$dir/cmd/cc-connect/main.go" <<'EOF'
package main

import (
  _ "github.com/chenhg5/cc-connect/agent/foo"
)
EOF
}

tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"' EXIT

pass_repo="$tmpdir/pass"
make_repo "$pass_repo"
write_main_with_imports "$pass_repo"

bash "$CHECK_SCRIPT" "$pass_repo"

fail_repo="$tmpdir/fail"
make_repo "$fail_repo"
write_main_missing_platform "$fail_repo"

if bash "$CHECK_SCRIPT" "$fail_repo"; then
  echo "expected missing import check to fail"
  exit 1
fi

echo "check_harness_test.sh: PASS"
