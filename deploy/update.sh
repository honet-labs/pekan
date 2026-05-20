#!/usr/bin/env bash
# Simple wrapper for update_app.sh
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
bash "${SCRIPT_DIR}/update_app.sh" "$@"
