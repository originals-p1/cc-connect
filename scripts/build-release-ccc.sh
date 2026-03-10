#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DIST_DIR="${ROOT_DIR}/dist"
CMD_PATH="./cmd/cc-connect"

TARGET="${1:-${TARGET:-}}"

if [[ -z "${TARGET}" ]]; then
  GOOS="$(cd "${ROOT_DIR}" && go env GOOS)"
  GOARCH="$(cd "${ROOT_DIR}" && go env GOARCH)"
else
  case "${TARGET}" in
    */*)
      GOOS="${TARGET%/*}"
      GOARCH="${TARGET#*/}"
      ;;
    *)
      echo "Usage: $0 [GOOS/GOARCH]" >&2
      echo "Example: $0 linux/amd64" >&2
      exit 1
      ;;
  esac
fi

VERSION="$(cd "${ROOT_DIR}" && git describe --tags --always --dirty 2>/dev/null || echo "dev")"
COMMIT="$(cd "${ROOT_DIR}" && git rev-parse --short HEAD 2>/dev/null || echo "none")"
BUILD_TIME="$(date -u '+%Y-%m-%dT%H:%M:%SZ')"

EXT=""
if [[ "${GOOS}" == "windows" ]]; then
  EXT=".exe"
fi

OUT_DIR="${DIST_DIR}/${GOOS}-${GOARCH}"
OUT_PATH="${OUT_DIR}/ccc${EXT}"

mkdir -p "${OUT_DIR}"

echo "Building ${OUT_PATH}"

(
  cd "${ROOT_DIR}"
  GOOS="${GOOS}" GOARCH="${GOARCH}" CGO_ENABLED=0 \
    go build \
      -ldflags "-s -w -X main.version=${VERSION} -X main.commit=${COMMIT} -X main.buildTime=${BUILD_TIME}" \
      -o "${OUT_PATH}" \
      "${CMD_PATH}"
)

echo "Built ${OUT_PATH}"
