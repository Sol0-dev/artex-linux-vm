#!/usr/bin/env bash
# Re-authenticate the Kilo CLI and reload ARTEX so the new token is used.
#
# Kilo Gateway tokens rotate/expire. A stale token produces:
#   openai: status 401: {"error":{"code":"INVALID_TOKEN",...}}
# Run this once, choose "Kilo Gateway", complete the browser/device login, and
# ARTEX is restarted automatically.
set -euo pipefail

cd "$(dirname "$0")/.."

command -v kilo >/dev/null 2>&1 || {
  echo "[artex] kilo is not on PATH; install the Kilo CLI first" >&2
  exit 1
}

echo "[artex] opening Kilo login (choose: Kilo Gateway)"
kilo auth login

if [[ ! -f "$HOME/.local/share/kilo/auth.json" ]]; then
  echo "[artex] Kilo login did not write ~/.local/share/kilo/auth.json" >&2
  exit 1
fi

echo "[artex] Kilo login complete; restarting ARTEX to reload the token"
./scripts/restart-local.sh
