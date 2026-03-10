#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
INSTALL_SCRIPT="${ROOT_DIR}/scripts/build-install-ccc.sh"
SERVICE_NAME="${SERVICE_NAME:-cc-connect}"
TARGET="${1:-${TARGET:-}}"

if [[ ! -x "${INSTALL_SCRIPT}" ]]; then
  echo "Missing install script: ${INSTALL_SCRIPT}" >&2
  exit 1
fi

"${INSTALL_SCRIPT}" "${TARGET}"

if [[ "${NO_RESTART:-0}" == "1" ]]; then
  echo "Skipping systemd restart because NO_RESTART=1"
  exit 0
fi

systemctl --user daemon-reload
systemctl --user restart "${SERVICE_NAME}"
systemctl --user --no-pager --full status "${SERVICE_NAME}"
