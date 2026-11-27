# Run ARTEX on a Kali or Ubuntu VM

> **New here?** Read **[START_HERE.md](../START_HERE.md)** for the complete first-run
> walkthrough, including OpenCode Go, Kilo Code, start/stop/restart and the task flow.

This fork keeps the upstream ARTEX architecture and adds local-friendly providers
and MCP integrations:

- **OpenCode Go** — DeepSeek V4.1 Flash 1017.
- **Kilo Code** — free Baidu ERNIE-5.5-Flash-Preview, native tool calls.
- **Antigravity (agy)** — Gemini 4 Hydrogen / 3.8 Flash, medium-only.
- **Playwright MCP** — headed Chromium, per-conversation browser profile reuse.
- **Caido MCP** — proxy history, replay, findings and scopes.
- **Per-conversation MCP reuse** — a conversation keeps its browser across turns;
  Stop does not kill it; deleting the conversation closes it.

Upstream: <https://github.com/BaaRai/ARTEX> (AGPL-3.0). This fork is also AGPL-3.0.

## 1. Base system

Tested on Kali Rolling and Ubuntu 24.04.

```bash
sudo apt update
sudo apt install -y git build-essential python3 python3-pip chromium \
  docker.io docker-compose-plugin
sudo usermod -aG docker "$USER"   # log out/in once
```

Install **Go 1.26+** and **Node.js 20+** from their official distributions (the
distro packages are usually too old). Verify:

```bash
go version
node -v
docker compose version
```

## 2. Clone and build

```bash
git clone https://github.com/Sol0-dev/artex-linux-vm.git
cd artex-linux-vm
./scripts/setup-linux-vm.sh
```

The setup script:

1. checks dependencies,
2. creates `.env` with a random PostgreSQL password (mode 0600),
3. starts the bundled PostgreSQL container,
4. builds the Next.js UI and embeds it into the single Go binary.

It never writes credentials into git. `.env`, `data/`, `keys/`, `jwt.key`,
`secret.key`, `ADMIN_CREDENTIALS.md` and the binary are ignored.

## 3. Start

```bash
./scripts/start-local.sh
# or:
nohup ./scripts/start-local.sh > artex.log 2>&1 &
```

Open `http://<VM-IP>:8787`. The first visit asks for the one-time setup token:

```bash
grep "Setup token" artex.log
```

Create the admin password, then configure providers in **System → LLM**.

Restart after a rebuild or a configuration change without touching PostgreSQL:

```bash
./scripts/restart-local.sh            # stop + start
./scripts/restart-local.sh --status   # running / stopped
./scripts/restart-local.sh --stop     # stop only
```

The wrapper writes `artex.pid` and appends to `artex.log`; both are gitignored.

> The UI listens on `0.0.0.0:8787` so the VM is reachable from your host. Use a
> firewall/reverse proxy and a strong admin password on untrusted networks.

## 4. Providers

### OpenCode Go

In **System → LLM**, create **OpenCode Go (DeepSeek V4.1 Flash 1017)** and paste
an OpenCode Go API key. The base URL, model, `x-opencode-session` header, User-Agent
and 1M context window are filled automatically.

### Kilo Code (Baidu free)

Install and sign in to the Kilo CLI once:

```bash
# install per https://kilo.ai, then:
kilo models
```

ARTEX reads `~/.local/share/kilo/auth.json` automatically. Create a **Kilo Code**
profile in the UI; the default model is
`baidu/ernie-5.5-flash-preview:free` (native tool calls). Override the token with
`KILO_API_KEY` or the file path with `KILO_AUTH_FILE`.

Kilo Gateway tokens rotate. After a later `kilo auth login`, run
`./scripts/kilo-login.sh` (or `kilo auth login && ./scripts/restart-local.sh`).
ARTEX follows the fresh CLI token automatically even if the profile has an old
key. A `401 INVALID_TOKEN` means the gateway token is stale; it is not a Baidu
model failure. `~/agents/kilo/kilo.credentials` is the local secops.irc mesh
credential and is **not** used for the Kilo AI gateway.

### Antigravity (agy)

Install and sign in to the Antigravity CLI (`agy`), then start the local bridge:

```bash
./scripts/start-agy-bridge.sh
# or: python3 scripts/agy_bridge.py
```

In the UI create an **Antigravity** profile. It exposes only:

```text
gemini-4-hydrogen-medium
gemini-3.8-flash-medium
```

The bridge is headed/non-streaming and requires Antigravity quota. It listens on
`http://127.0.0.1:8790/v1`. If ARTEX runs in Docker, use
`http://host.docker.internal:8790/v1` and add that host:port to
`ARTEX_LLM_PRIVATE_ENDPOINTS`.

### External PostgreSQL (optional)

Instead of the bundled container, set a full `ARTEX_PG_DSN` or the
`ARTEX_PG_HOST`, `ARTEX_PG_PORT`, `ARTEX_PG_USER`, `ARTEX_PG_PASSWORD`,
`ARTEX_PG_DBNAME`, `ARTEX_PG_SSLMODE` variables before
`./scripts/start-local.sh`. The setup script still builds the binary; use
`./scripts/setup-linux-vm.sh --check` to only verify dependencies.

## 5. MCP servers

### Playwright

```bash
npm install -g @playwright/mcp
```

In **System → MCP**, enable the seeded **browser** server. A working configuration:

```text
command: playwright-mcp
args:    --browser chromium --executable-path /usr/bin/chromium --no-sandbox
```

Do **not** add `--headless` if you want to see the browser. Each conversation gets
its own profile under `data/playwright-mcp/conv-<id>` and keeps it across replies;
Stop does not close it. Deleting the conversation closes it. Abandoned browsers
are closed after 30 minutes idle.

Grant visibility to **Planner, Worker, Main, Auto and Pentest** in the MCP page.

### Caido

Run Caido, then install the v4 MCP server:

```bash
# https://github.com/c0tton-fluff/caido-mcp-server
caido-mcp-server --version   # must be v4+
```

Add a stdio MCP server:

```text
command: /path/to/caido-mcp-server
args:    serve
env:
  CAIDO_URL: http://127.0.0.1:8080
  CAIDO_ACCESS_TOKEN: <Caido access token>
```

Get the access token from the Caido GUI developer console:

```js
JSON.parse(localStorage.CAIDO_AUTHENTICATION).accessToken
```

Tokens expire after roughly 7 days; refresh the token, or use the Caido OAuth
login flow if supported by your version.

## 6. Security and data

- `data/` contains traffic, transcripts, task files, findings, assets, graphs and
  local archives. It is ignored by git and must be backed up/removed by you.
- `.env`, `jwt.key`, `secret.key`, `keys/`, `ADMIN_CREDENTIALS.md`, `token.txt`,
  `poc.py`, `caido.txt` and `work/` are local-only. Never commit them.
- Use ARTEX only against systems you are authorised to test.
- Change the admin password before exposing the UI beyond localhost.
- LLM keys and MCP env values are encrypted at rest by ARTEX.

## 7. Development

```bash
cd web && npm ci && NEXT_EXPORT=1 npm run build:static && cd ..
rm -rf server/webui/dist && mkdir -p server/webui && cp -r web/out server/webui/dist
CGO_ENABLED=0 go build -tags embedui -o artex ./cmd/artex
go test ./agent ./server
```

The fork-specific tests are in `agent/*_test.go` and
`server/chat_mcp_cache_test.go`.
