#!/usr/bin/env bash
# One-shot local setup for ARTEX on Kali/Ubuntu.
# Builds the UI + single binary and starts the bundled PostgreSQL container.
# It never writes credentials into git: .env, data/, keys/ and the binary are ignored.
set -euo pipefail

cd "$(dirname "$0")/.."
ROOT="$PWD"
CHECK_ONLY=0
[[ "${1:-}" == "--check" ]] && CHECK_ONLY=1

say() { printf '\n\033[1;36m[artex]\033[0m %s\n' "$*"; }
die() { printf '\n\033[1;31m[artex]\033[0m %s\n' "$*" >&2; exit 1; }

have() { command -v "$1" >/dev/null 2>&1; }

check_deps() {
  local missing=0
  for c in git go node npm python3 docker; do
    if ! have "$c"; then
      printf '  missing: %s\n' "$c"
      missing=1
    fi
  done
  if have docker && ! docker compose version >/dev/null 2>&1; then
    printf '  missing: docker compose plugin\n'
    missing=1
  fi
  if have go; then
    local v
    v="$(go env GOVERSION 2>/dev/null | sed 's/^go//')"
    printf '  go:      %s\n' "${v:-unknown}"
  fi
  if have node; then printf '  node:    %s\n' "$(node -v)"; fi
  if have python3; then printf '  python:  %s\n' "$(python3 -V 2>&1)"; fi
  printf '  chromium: %s\n' "$(command -v chromium || command -v chromium-browser || echo 'not found (optional for Playwright)')"
  printf '  playwright-mcp: %s\n' "$(command -v playwright-mcp || echo 'not found (optional)')"
  printf '  caido-mcp-server: %s\n' "$(command -v caido-mcp-server || echo 'not found (optional)')"
  printf '  agy: %s\n' "$(command -v agy || echo 'not found (optional)')"
  printf '  kilo: %s\n' "$(command -v kilo || echo 'not found (optional)')"
  if (( missing )); then
    cat >&2 <<'EOF'

Install the missing base packages, then re-run this script.

Kali / Ubuntu / Debian:
  sudo apt update
  sudo apt install -y git build-essential python3 python3-pip chromium docker.io docker-compose-plugin

Go 1.26+ (the distro package is usually too old):
  https://go.dev/dl/    # or use your version manager

Node.js 20+:
  https://nodejs.org/en/download

Optional provider/MCP helpers:
  npm install -g @playwright/mcp
  # Caido MCP v4: https://github.com/c0tton-fluff/caido-mcp-server
  # Antigravity CLI: install `agy` and sign in
  # Kilo CLI: install `kilo` and sign in
EOF
    exit 1
  fi
}

check_deps
(( CHECK_ONLY )) && { say "dependency check passed"; exit 0; }

# ---------------------------------------------------------------------------
# 1. .env: generated once, never committed. The setup password is random.
# ---------------------------------------------------------------------------
if [[ ! -f .env ]]; then
  cp .env.example .env
  PW="$(python3 -c 'import secrets; print(secrets.token_urlsafe(24))')"
  python3 - "$PW" <<'PY'
import pathlib, sys
p = pathlib.Path('.env')
s = p.read_text()
s = s.replace('POSTGRES_PASSWORD=change-me-please', 'POSTGRES_PASSWORD=' + sys.argv[1])
s = s.replace('ARTEX_BIND_ADDR=127.0.0.1', 'ARTEX_BIND_ADDR=0.0.0.0')
p.write_text(s)
PY
  chmod 600 .env
  say "created .env with a random PostgreSQL password"
else
  say "keeping existing .env"
fi

# ---------------------------------------------------------------------------
# 2. PostgreSQL (bundled compose container)
# ---------------------------------------------------------------------------
if [[ -n "${ARTEX_PG_HOST:-}" ]]; then
  say "using external PostgreSQL at $ARTEX_PG_HOST"
else
  say "starting PostgreSQL container"
  docker compose up -d postgres

  say "waiting for PostgreSQL"
  ok=0
  for _ in $(seq 1 60); do
    if docker compose exec -T postgres sh -c 'pg_isready -U "$POSTGRES_USER" -d "$POSTGRES_DB"' >/dev/null 2>&1; then
      ok=1
      break
    fi
    sleep 1
  done
  (( ok )) || die "PostgreSQL did not become ready; inspect: docker compose logs postgres"
fi

# ---------------------------------------------------------------------------
# 3. Build the embedded UI and the single binary
# ---------------------------------------------------------------------------
say "building the web UI"
(
  cd web
  if [[ ! -d node_modules ]]; then
    npm ci --no-audit --no-fund
  fi
  NEXT_EXPORT=1 npm run build:static
)

say "embedding the UI and building artex"
rm -rf server/webui/dist
mkdir -p server/webui
cp -r web/out server/webui/dist
CGO_ENABLED=0 go build -tags embedui -o artex ./cmd/artex

say "done"
cat <<'EOF'

Start it:
  ./scripts/start-local.sh
  # or in the background:
  nohup ./scripts/start-local.sh > artex.log 2>&1 &

Then open:
  http://<VM-IP>:8787

The first visit needs the setup token:
  grep "Setup token" artex.log

Provider and MCP setup:
  docs/LINUX_VM.md

Never commit these:
  .env  data/  jwt.key  secret.key  ADMIN_CREDENTIALS.md
EOF
