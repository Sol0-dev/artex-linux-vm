# Start here

This is the complete beginner path for the Linux VM fork: install ARTEX, connect
OpenCode Go and Kilo Code, start/stop/restart it, and run a task from the first
company scope to the final report.

> **Use ARTEX only on systems you are authorised to test.** Add targets to the
> company scope, keep the scope tight, and change the admin password before
> exposing the UI beyond localhost.

---

## 1. What ARTEX is

ARTEX is an autonomous pentest runner, not a chat wrapper:

```text
company scope + assets
        ↓
task goal
        ↓
planner reads the exploration graph
        ↓
planner dispatches intents (one direction each)
        ↓
workers execute real tools (HTTP, Bash, Caido, browser, scanners)
        ↓
workers write back facts / assets / findings
        ↓
planner wakes on the graph change and repeats
        ↓
prove_goal → done
```

Two graphs are kept:

- **Asset graph** — what exists: root domains, subdomains, IPs, services,
  applications, endpoints.
- **Exploration graph** — what was tried, what it produced, and why: goals,
  intents, facts, findings and hints.

A finding is strongest when it is tied to captured traffic and verbatim evidence.

---

## 2. Install on Kali/Ubuntu

```bash
git clone https://github.com/Sol0-dev/artex-linux-vm.git
cd artex-linux-vm
./scripts/setup-linux-vm.sh
```

The setup script checks dependencies, creates `.env` with a random PostgreSQL
password, starts PostgreSQL, builds the UI, and builds the `artex` binary.

Full dependency notes: [docs/LINUX_VM.md](docs/LINUX_VM.md).

---

## 3. Start, stop, restart

### Start

```bash
./scripts/start-local.sh
```

Run it in the background:

```bash
nohup ./scripts/start-local.sh > artex.log 2>&1 &
```

### Restart

```bash
./scripts/restart-local.sh
```

```text
stop the current ARTEX process
start it again with the same .env and PostgreSQL
wait until /api/health is ready
```

### Status / stop

```bash
./scripts/restart-local.sh --status
./scripts/restart-local.sh --stop
```

`restart-local.sh` writes `artex.pid` and appends to `artex.log`. Both are
gitignored.

### First login

Open:

```text
http://<VM-IP>:8787
```

The first visit asks for the setup token printed at startup:

```bash
grep "Setup token" artex.log
```

Set the admin password. There is no default password.

---

## 4. Connect the LLM providers

Open **System → LLM**. A profile is one endpoint + model + key. The active profile
is used by the planner and workers; a task can also choose its own chain.

### 4.1 OpenCode Go

OpenCode Go is the subscription/inference product. You do not need the OpenCode
CLI to run ARTEX.

1. Get an OpenCode Go API key from the OpenCode Console.
2. **System → LLM → New profile**.
3. Format: **OpenCode Go (DeepSeek V4.1 Flash 1017)**.
4. Paste the Go API key.
5. Click **Test**.
6. Save, then **Set as active**.

The fork fills the rest automatically:

```text
base URL:  https://opencode.ai/inference/go/v1
model:     deepseek-v4.1-flash
headers:   x-opencode-session + artex/1.0 User-Agent
context:   1M tokens
```

The session header is required by OpenCode Go and is sent automatically. The
`x-opencode-session` value is stable per conversation/task, so prompt-cache and
routing work correctly.

Environment alternative:

```bash
export ARTEX_LLM_PROVIDER=opencode-go
export OPENCODE_API_KEY=...
export ARTEX_LLM_MODEL=deepseek-v4.1-flash
```

### 4.2 Kilo Code

Kilo is used as an OpenAI-compatible provider, not through its terminal UI.
ARTEX reads the Kilo CLI login so you do not have to copy another key.

1. Install and sign in to the Kilo CLI. Verify:

   ```bash
   kilo models
   ```

2. ARTEX reads `~/.local/share/kilo/auth.json` automatically. If your file is
   elsewhere:

   ```bash
   export KILO_AUTH_FILE=/path/to/auth.json
   # or paste a token:
   export KILO_API_KEY=...
   ```

3. **System → LLM → New profile**.
4. Format: **Kilo Code (Baidu ERNIE-5.5-Flash-Preview · free)**.
5. Leave the key empty to use the Kilo CLI login, or paste `KILO_API_KEY`.
6. Click **Test**, save, then **Set as active**.

Default model:

```text
baidu/ernie-5.5-flash-preview:free
```

Kilo supports native OpenAI tool calls, so workers run normally. The fork uses
non-streaming mode for the free gateway because its SSE stream is occasionally
slow to close.

### 4.3 Which provider should I use?

| Situation | Use |
| --- | --- |
| You have an OpenCode Go subscription | OpenCode Go / DeepSeek V4.1 Flash |
| You want a free model | Kilo / Baidu ERNIE free |
| You want Gemini 4 Hydrogen / 3.8 Flash | Antigravity (see below) |
| You want failover | Create multiple profiles and enable the LLM pool/chain |

For Antigravity:

```bash
./scripts/start-agy-bridge.sh
```

Then create an **Antigravity** profile. It exposes only:

```text
gemini-4-hydrogen-medium
gemini-3.8-flash-medium
```

It requires a signed-in `agy` and Antigravity quota.

---

## 5. Connect MCP tools (optional but recommended)

MCP servers add tools to agents. Enable them in **System → MCP**, then grant
visibility to the agents that should use them.

### Playwright (browser)

```bash
npm install -g @playwright/mcp
```

Enable the seeded **browser** server with:

```text
command: playwright-mcp
args: --browser chromium --executable-path /usr/bin/chromium --no-sandbox
```

Do not add `--headless` if you want to see the browser. The fork keeps the browser
for the whole conversation and gives each conversation its own profile:

```text
data/playwright-mcp/conv-<id>
```

So one conversation does not see another conversation’s tabs. Stop does not
close it; deleting the conversation does. Idle browsers close after 30 minutes.

Grant visibility to **Planner, Worker, Main, Auto and Pentest**.

### Caido

Install Caido and the **v4+** MCP server, then add a stdio server:

```text
command: /path/to/caido-mcp-server
args: serve
env:
  CAIDO_URL: http://127.0.0.1:8080
  CAIDO_ACCESS_TOKEN: <token>
```

Get the local Caido access token from the Caido developer console:

```js
JSON.parse(localStorage.CAIDO_AUTHENTICATION).accessToken
```

The token expires after roughly 7 days; replace it in the MCP env when needed.
Caido tools include proxy history, replay, findings, scopes and sitemap.

### Granting visibility

For each MCP server, toggle the agents on its card. If an agent cannot see the
server, its tools are never offered to the model.

---

## 6. Build the scope (do this before a task)

Open **Assets** in the left sidebar.

1. Click **Add company**.
2. Enter the company name.
3. Add the asset scope: domains, subdomains, IPs, CIDRs, ICPs and company
   keywords. Keywords are agent hints; they do not automatically claim assets.
4. Matching assets are claimed by the company. You can also import assets in
   **Asset Sync** or add them manually.

The scope is the safety boundary and the coverage denominator. Do not rely on
the model to remember the scope; put it in the company scope.

---

## 7. Create the first task

Open **Tasks → New task**.

Fill in:

1. **Goal** — a concrete, verifiable outcome, for example:

   ```text
   Find an information disclosure or authentication bypass on the in-scope
   demo application and prove it with captured HTTP traffic.
   ```

2. **Linked company asset scope** — select the company from step 6. Its current
   assets become **Test assets**; its domains/IPs/CIDRs become scope context.

3. **LLM config chain** — select the profile(s). Order matters for worker rotation
   and failover. A fixed Worker model in **System → Agent → Worker** overrides
   this chain; leave Worker on “follow the task” if you want rotation.

4. **Test accounts (optional)** — dedicated accounts for authenticated testing.
   Anything entered here is sent to the LLM provider.

5. **Advanced settings** — task timeout, planner heartbeat, first intent. Most
   users can leave these empty.

6. **Interception rules** — optional task-level asset block/allow rules.

Before starting, set the global defaults in **System → Settings**:

- **Traffic capture** — off by default. Turn it on if you want provable
  findings bound to HTTP requests/responses. The recording proxy listens on
  `127.0.0.1:8788`.
- **Concurrency** — number of workers per task (default 3).
- **Noise profile** — optional rate limiting for target requests.

Save the task. The engine starts immediately when it is running.

---

## 8. What happens after start

On the task detail page:

| Tab | What it shows |
| --- | --- |
| **Sessions** | planner/worker model turns, tool calls, outputs |
| **Overview** | task state, budget and progress |
| **Exploration graph** | goals → intents → facts/findings |
| **Broadcast** | live activity stream |
| **Findings** | discovered vulnerabilities |
| **Retests** | fixed/not-fixed verification runs |
| **Test assets** | assets this task may test |
| **Credentials** | test accounts used by the task |
| **Asset coverage** | tested vs untested scope |
| **Interception approval** | pending human approvals |
| **Report** | generated report and exports |

The planner wakes on graph changes. It dispatches intents only when there is an
uncovered direction; workers claim one intent, execute it with real tools, write
back, and stop. Most wake-ups dispatch zero intents when the task is done.

Task controls:

- **Pause / Resume** — stop or continue the planner/worker loop.
- **Stop chat** — abort only the current main-agent reply; the task keeps running.
- **Archive** — compress the task graph, files, LLM history and exclusive assets
  into cold storage.
- **Delete** — remove the task and selected related data; choose carefully.

---

## 9. Human-in-the-loop

Use the task chat input to steer the planner:

- Ask for a status summary.
- Add a hint about a parameter, credential or endpoint.
- Inject a high-priority intent.
- Tell it to stop a noisy direction.

Interception can require approval for sensitive commands. Check **Interception
Rules**, **Asset Interception** and **Approval Records**. The approval’s
`decided_by` value shows whether a human, rule, model or system approved it.

Budgets are optional environment limits:

```bash
ARTEX_TASK_BUDGET_TOKENS=200000
ARTEX_TASK_BUDGET_TURNS=80
ARTEX_INTENT_BUDGET_TOKENS=50000
ARTEX_INTENT_BUDGET_TURNS=20
```

Unset or `0` means unlimited. Near the limit the task enters a forced wrap-up.

---

## 10. Findings, evidence and reports

A finding is not just a note. For a strong finding:

1. Turn on **traffic capture** before the task.
2. Let the worker capture the request/response that proves it.
3. Bind the evidence to the finding and add verbatim quotes.
4. Mark verification honestly (`verified` only when the proof supports it).
5. Use **Retests** to re-run the finding later with a fixed version or changed
   conditions.

Traffic is stored under `data/traffic/`; finding evidence and reports are tied to
the task. Export the report or archive the task from the task detail page.

---

## 11. Data, backups and safety

- `data/` contains traffic, transcripts, task files, findings, assets, graph
  nodes and local archives. It is gitignored and is **not** published.
- `.env`, `jwt.key`, `secret.key`, `keys/`, `ADMIN_CREDENTIALS.md`, `token.txt`,
  `poc.py`, `caido.txt` and `work/` are local-only.
- Back up PostgreSQL and `data/` together. Do not copy only one.
- Change the admin password and use a firewall/reverse proxy before exposing
  `0.0.0.0:8787` beyond a trusted network.
- Only test in-scope assets you are authorised to test.

---

## 12. Troubleshooting

### “No LLM provider configured”

Open **System → LLM**, create a profile, click **Test**, save, and **Set as active**.
The engine stays idle until a profile is usable.

### OpenCode Go returns 401/403/429

- 401/403: wrong key, or the key is not covered by an OpenCode Go subscription.
- 429: Go allowance/rate limit. Check the Console usage; wait for the window or
  switch to Kilo for a free run.
- The session header is automatic; do not send a new one per request.

### Kilo says “no key”

Run `kilo models` once to refresh the login, then verify
`~/.local/share/kilo/auth.json`. Or set `KILO_API_KEY`.

### Antigravity quota

The bridge forwards the account’s Antigravity quota error. Wait for the reset or
use another provider. The bridge is on `127.0.0.1:8790`.

### Playwright does not open / shows another conversation’s tab

- The fork uses one profile per conversation under `data/playwright-mcp/`.
- Do not add `--headless`; the server needs `DISPLAY` and Chromium.
- Use `--no-sandbox` in a VM/container.
- Delete the conversation to close its browser; idle browsers close after 30
  minutes.

### Caido MCP connection closed

Use the **v4+** `caido-mcp-server` binary, set `CAIDO_URL`, and refresh the token.
Caido must be running and listening on that URL.

### Port or database problems

```bash
docker compose ps
docker compose logs postgres
ss -ltnp | grep 8787
tail -80 artex.log
```

`ARTEX_BIND_ADDR`, `ARTEX_PG_*` and the provider env variables are documented in
`docs/LINUX_VM.md`.

---

## Quick reference

```bash
# install
./scripts/setup-linux-vm.sh

# run
./scripts/start-local.sh
nohup ./scripts/start-local.sh > artex.log 2>&1 &

# lifecycle
./scripts/restart-local.sh --status
./scripts/restart-local.sh --stop
./scripts/restart-local.sh

# UI
http://<VM-IP>:8787
grep "Setup token" artex.log

# providers
System → LLM → OpenCode Go / Kilo Code
System → MCP → browser / caido

# first run
Assets → Add company → scope
Tasks → New task → goal + company scope + LLM chain
Tasks → detail → Sessions / Graph / Findings / Report
```

Upstream ARTEX: <https://github.com/BaaRai/ARTEX> · AGPL-3.0.
This fork: <https://github.com/Sol0-dev/artex-linux-vm> · AGPL-3.0.
