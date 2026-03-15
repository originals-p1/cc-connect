#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BUILD_SCRIPT="${ROOT_DIR}/scripts/build-release-ccc.sh"
INSTALL_DIR="${INSTALL_DIR:-${HOME}/.local/bin}"
TARGET="${1:-${TARGET:-}}"

if [[ ! -x "${BUILD_SCRIPT}" ]]; then
  echo "Missing build script: ${BUILD_SCRIPT}" >&2
  exit 1
fi

"${BUILD_SCRIPT}" "${TARGET}"

if [[ -z "${TARGET}" ]]; then
  GOOS="$(cd "${ROOT_DIR}" && go env GOOS)"
  GOARCH="$(cd "${ROOT_DIR}" && go env GOARCH)"
else
  GOOS="${TARGET%/*}"
  GOARCH="${TARGET#*/}"
fi

EXT=""
if [[ "${GOOS}" == "windows" ]]; then
  EXT=".exe"
fi

SRC="${ROOT_DIR}/dist/${GOOS}-${GOARCH}/cc-connect${EXT}"
DEST="${INSTALL_DIR}/cc-connect${EXT}"

mkdir -p "${INSTALL_DIR}"
install -m 0755 "${SRC}" "${DEST}"

echo "Installed ${DEST}"
