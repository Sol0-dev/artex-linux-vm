#!/bin/sh
# Start the local OpenAI-compatible bridge for the Antigravity CLI (agy).
# ARTEX's "Antigravity (agy)" provider talks to http://127.0.0.1:8790/v1.
set -eu

cd "$(dirname "$0")/.." || exit 1

if ! command -v agy >/dev/null 2>&1; then
	echo "[agy-bridge] agy is not on PATH; install/sign in to Antigravity first" >&2
	exit 1
fi
if ! command -v python3 >/dev/null 2>&1; then
	echo "[agy-bridge] python3 is required" >&2
	exit 1
fi

exec python3 scripts/agy_bridge.py "$@"
