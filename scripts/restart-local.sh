#!/usr/bin/env bash
# Restart the locally built ARTEX process without losing PostgreSQL or .env.
#
#   ./scripts/restart-local.sh            # stop + start
#   ./scripts/restart-local.sh --status   # show whether it is running
#   ./scripts/restart-local.sh --stop     # stop only
#
# Override ARTEX_PORT (default 8787), ARTEX_PID_FILE (default ./artex.pid) and
# ARTEX_LOG (default ./artex.log) when needed.
set -euo pipefail

cd "$(dirname "$0")/.."

PORT="${ARTEX_PORT:-8787}"
PID_FILE="${ARTEX_PID_FILE:-$PWD/artex.pid}"
LOG_FILE="${ARTEX_LOG:-$PWD/artex.log}"
ACTION="${1:---restart}"

say() { printf '\n\033[1;36m[artex]\033[0m %s\n' "$*"; }
die() { printf '\n\033[1;31m[artex]\033[0m %s\n' "$*" >&2; exit 1; }

is_artex_pid() {
  local p="$1" exe
  [[ "$p" =~ ^[0-9]+$ ]] || return 1
  kill -0 "$p" 2>/dev/null || return 1
  exe="$(readlink -f "/proc/$p/exe" 2>/dev/null || true)"
  [[ "$(basename "${exe:-}")" == "artex" ]]
}

find_pid() {
  local p=""
  if [[ -f "$PID_FILE" ]]; then
    p="$(cat "$PID_FILE" 2>/dev/null || true)"
    if is_artex_pid "$p"; then
      printf '%s\n' "$p"
      return 0
    fi
  fi
  # Fallback when the pid file is stale/missing: the listener on the ARTEX port.
  if command -v ss >/dev/null 2>&1; then
    p="$(ss -ltnp "sport = :$PORT" 2>/dev/null | sed -n 's/.*pid=\([0-9]\+\).*/\1/p' | head -1 || true)"
    if is_artex_pid "$p"; then
      printf '%s\n' "$p"
      return 0
    fi
  fi
  return 1
}

stop_artex() {
  local p
  if ! p="$(find_pid)"; then
    say "ARTEX is not running on port $PORT"
    rm -f "$PID_FILE"
    return 0
  fi
  say "stopping ARTEX pid $p"
  kill -TERM "$p" 2>/dev/null || true
  for _ in $(seq 1 30); do
    is_artex_pid "$p" || break
    sleep 0.5
  done
  if is_artex_pid "$p"; then
    say "pid $p did not stop in 15s; sending SIGKILL"
    kill -KILL "$p" 2>/dev/null || true
    sleep 1
  fi
  rm -f "$PID_FILE"
  say "stopped"
}

start_artex() {
  [[ -x ./artex ]] || die "./artex is missing; run ./scripts/setup-linux-vm.sh first"
  [[ -f .env ]] || die ".env is missing; run ./scripts/setup-linux-vm.sh first"
  command -v curl >/dev/null 2>&1 || die "curl is required for the readiness check"

  say "starting ARTEX (log: $LOG_FILE)"
  : >>"$LOG_FILE"
  nohup ./scripts/start-local.sh >>"$LOG_FILE" 2>&1 &
  echo $! >"$PID_FILE"

  local p actual
  p="$(cat "$PID_FILE")"
  for _ in $(seq 1 60); do
    if curl -fsS "http://127.0.0.1:${PORT}/api/health" >/dev/null 2>&1; then
      # The shell that started start-local.sh is replaced by ./artex via exec,
      # but always trust the actual listener if the pid differs.
      actual="$(find_pid || true)"
      [[ -n "$actual" ]] || actual="$p"
      printf '%s\n' "$actual" >"$PID_FILE"
      say "ready: http://<VM-IP>:${PORT}  (pid $actual)"
      return 0
    fi
    sleep 0.5
  done
  tail -40 "$LOG_FILE" >&2 || true
  die "ARTEX did not become healthy within 30s; see $LOG_FILE"
}

status_artex() {
  local p
  if p="$(find_pid)"; then
    say "running: pid $p, port $PORT"
  else
    say "stopped (port $PORT)"
  fi
}

case "$ACTION" in
  --status|status) status_artex ;;
  --stop|stop) stop_artex ;;
  --restart|restart|"") stop_artex; start_artex ;;
  *) die "unknown action '$ACTION' (use --status, --stop or --restart)" ;;
esac
