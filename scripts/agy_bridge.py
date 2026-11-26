#!/usr/bin/env python3
"""OpenAI-compatible bridge for the Antigravity CLI (agy).

ARTEX speaks OpenAI Chat Completions; agy only runs as a CLI, so this bridge
turns each request into one non-interactive `agy --print` call. It asks the
model to answer with a small JSON protocol so ARTEX tool calls survive the hop.

Only the two "medium" Gemini models requested for ARTEX are exposed. Any
high/low/xhigh variant name is accepted and mapped to its medium sibling.

Usage:
    python3 scripts/agy_bridge.py            # listens on 127.0.0.1:8790
    AGY_BRIDGE_PORT=8791 python3 scripts/agy_bridge.py

The bridge needs an authenticated `agy` on PATH. It is intentionally a local
process: ARTEX talks to http://127.0.0.1:8790/v1.
"""
import json
import os
import re
import subprocess
import sys
import time
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer

HOST = os.environ.get("AGY_BRIDGE_HOST", "127.0.0.1")
PORT = int(os.environ.get("AGY_BRIDGE_PORT", "8790"))
AGY = os.environ.get("AGY_BIN", "agy")
WORKDIR = os.environ.get("AGY_BRIDGE_WORKDIR", "/tmp/agy-bridge-work")
LOG = os.environ.get("AGY_BRIDGE_LOG", "/tmp/agy-bridge.log")

# Medium only. The model list is what ARTEX's "refresh models" button shows.
MODELS = [
    "gemini-4-hydrogen-medium",
    "gemini-3.8-flash-medium",
]
ALIASES = {
    "gemini-4-hydrogen": "gemini-4-hydrogen-medium",
    "gemini-4-hydrogen-high": "gemini-4-hydrogen-medium",
    "gemini-4-hydrogen-low": "gemini-4-hydrogen-medium",
    "gemini-4-hydrogen-xhigh": "gemini-4-hydrogen-medium",
    "gemini-4": "gemini-4-hydrogen-medium",
    "gemini-3.8-flash": "gemini-3.8-flash-medium",
    "gemini-3.8-flash-high": "gemini-3.8-flash-medium",
    "gemini-3.8-flash-low": "gemini-3.8-flash-medium",
    "gemini-3.8": "gemini-3.8-flash-medium",
}
os.makedirs(WORKDIR, exist_ok=True)


def log(msg):
    try:
        with open(LOG, "a") as f:
            f.write(f"{time.strftime('%Y-%m-%d %H:%M:%S')} {msg}\n")
    except Exception:
        pass


def norm_model(model):
    m = (model or "").strip()
    if m in MODELS:
        return m
    return ALIASES.get(m, "gemini-4-hydrogen-medium")


def text_of(content):
    if isinstance(content, str):
        return content
    if isinstance(content, list):
        out = []
        for p in content:
            if isinstance(p, dict):
                if p.get("type") == "text":
                    out.append(p.get("text", ""))
                elif p.get("type") in ("image_url", "input_image"):
                    out.append("[image omitted]")
        return "\n".join(out)
    return "" if content is None else str(content)


def build_prompt(req):
    tools = []
    for t in req.get("tools") or []:
        fn = t.get("function") if isinstance(t, dict) else None
        if fn:
            tools.append({
                "name": fn.get("name", ""),
                "description": (fn.get("description") or "")[:600],
                "parameters": fn.get("parameters", {"type": "object", "properties": {}}),
            })
    lines = []
    for m in req.get("messages") or []:
        role = m.get("role", "user")
        if role == "system":
            lines.append("SYSTEM: " + text_of(m.get("content")))
        elif role == "user":
            lines.append("USER: " + text_of(m.get("content")))
        elif role == "assistant":
            body = text_of(m.get("content"))
            calls = m.get("tool_calls") or []
            for c in calls:
                fn = (c or {}).get("function") or {}
                lines.append(f"ASSISTANT TOOL CALL: {fn.get('name','')}({fn.get('arguments','{}')})")
            if body:
                lines.append("ASSISTANT: " + body)
            if not body and not calls:
                lines.append("ASSISTANT: ")
        elif role == "tool":
            lines.append(f"TOOL RESULT ({m.get('name') or m.get('tool_call_id') or 'tool'}): {text_of(m.get('content'))}")
        else:
            lines.append(f"{role.upper()}: {text_of(m.get('content'))}")

    proto = (
        "You are the language-model backend for an autonomous agent. You cannot run tools yourself. "
        "Decide the single next step and reply with ONE JSON object only, no markdown fences and no prose.\n"
        "To call a tool: {\"type\":\"tool_call\",\"name\":\"<tool name>\",\"arguments\":{...}}\n"
        "To finish: {\"type\":\"final\",\"content\":\"<your answer>\"}\n"
        "Use a tool when the task needs information or an action you do not already have. "
        "Only use tool names from the Available tools list. Never invent results.\n\n"
    )
    if tools:
        proto += "Available tools:\n" + json.dumps(tools, ensure_ascii=False) + "\n\n"
    else:
        proto += "No tools are available; answer with a final object.\n\n"
    return proto + "Conversation:\n" + "\n".join(lines) + "\n\nReply with the JSON object now."


def extract_json(s):
    s = (s or "").strip()
    s = re.sub(r"^```(?:json)?\s*", "", s)
    s = re.sub(r"\s*```$", "", s)
    try:
        return json.loads(s)
    except Exception:
        pass
    start = s.find("{")
    if start >= 0:
        for end in range(len(s), start, -1):
            try:
                return json.loads(s[start:end])
            except Exception:
                continue
    return None


def call_agy(model, prompt):
    cmd = [
        AGY, "--model", model,
        "--print=" + prompt,
        "--output-format", "json",
        "--sandbox",
        "--disable-slash-commands",
    ]
    log(f"agy model={model} prompt_chars={len(prompt)}")
    t0 = time.time()
    p = subprocess.run(cmd, cwd=WORKDIR, capture_output=True, text=True, timeout=300)
    dt = time.time() - t0
    log(f"agy exit={p.returncode} secs={dt:.1f} stdout={len(p.stdout)} stderr={len(p.stderr)}")
    out = (p.stderr or p.stdout or "").strip()
    if p.returncode != 0:
        # Antigravity reports its long-lived quota as RESOURCE_EXHAUSTED /
        # "Individual quota reached". Surface that as 402 so ARTEX does not
        # retry it as a transient 5xx/429.
        if "quota" in out.lower() or "resource_exhausted" in out.lower():
            return None, None, out, 402
        return None, None, out[:2000], 502
    try:
        outer = json.loads(p.stdout)
    except Exception:
        return None, None, "agy returned non-JSON output: " + p.stdout[:1000], 502
    if outer.get("status") not in (None, "SUCCESS"):
        return None, None, json.dumps(outer)[:1000], 502
    return outer, dt, "", 200


def openai_response(model, msg, usage, finish, cid):
    return {
        "id": "chatcmpl-agy-" + str(cid),
        "object": "chat.completion",
        "created": int(time.time()),
        "model": model,
        "choices": [{"index": 0, "message": msg, "finish_reason": finish}],
        "usage": {
            "prompt_tokens": int(usage.get("input_tokens") or 0),
            "completion_tokens": int(usage.get("output_tokens") or 0),
            "total_tokens": int(usage.get("total_tokens") or 0),
        },
    }


class Handler(BaseHTTPRequestHandler):
    protocol_version = "HTTP/1.1"

    def _send(self, code, obj):
        b = json.dumps(obj).encode()
        self.send_response(code)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(b)))
        self.end_headers()
        self.wfile.write(b)

    def _send_sse(self, resp):
        # The ARTEX "antigravity" format forces non-streaming, but an OpenAI
        # profile can still ask for SSE; answer with one delta and finish.
        msg = resp["choices"][0]["message"]
        finish = resp["choices"][0]["finish_reason"]
        if msg.get("tool_calls"):
            call = msg["tool_calls"][0]
            delta = {"role": "assistant", "tool_calls": [{
                "index": 0, "id": call["id"], "type": "function", "function": call["function"],
            }]}
        else:
            delta = {"role": "assistant", "content": msg.get("content") or ""}
        chunks = [
            {"id": resp["id"], "object": "chat.completion.chunk", "created": resp["created"],
             "model": resp["model"], "choices": [{"index": 0, "delta": delta, "finish_reason": None}]},
            {"id": resp["id"], "object": "chat.completion.chunk", "created": resp["created"],
             "model": resp["model"], "choices": [{"index": 0, "delta": {}, "finish_reason": finish}],
             "usage": resp["usage"]},
        ]
        body = "".join("data: " + json.dumps(c) + "\n\n" for c in chunks) + "data: [DONE]\n\n"
        b = body.encode()
        self.send_response(200)
        self.send_header("Content-Type", "text/event-stream")
        self.send_header("Cache-Control", "no-cache")
        self.send_header("Content-Length", str(len(b)))
        self.end_headers()
        self.wfile.write(b)

    def do_GET(self):
        if self.path.rstrip("/") in ("/v1/models", "/models"):
            self._send(200, {"object": "list", "data": [
                {"id": m, "object": "model", "created": 0, "owned_by": "antigravity"} for m in MODELS
            ]})
        elif self.path.rstrip("/") in ("/health", "/v1/health"):
            self._send(200, {"ok": True, "backend": "agy", "models": MODELS})
        else:
            self._send(404, {"error": {"message": "not found"}})

    def do_POST(self):
        if not self.path.rstrip("/").endswith("/chat/completions"):
            self._send(404, {"error": {"message": "not found"}})
            return
        n = int(self.headers.get("Content-Length") or 0)
        try:
            req = json.loads(self.rfile.read(n) or b"{}")
        except Exception as e:
            self._send(400, {"error": {"message": f"bad json: {e}"}})
            return
        model = norm_model(req.get("model"))
        prompt = build_prompt(req)
        try:
            outer, dt, err, code = call_agy(model, prompt)
            if outer is None:
                log(f"ERROR {err}")
                self._send(code, {"error": {"message": err, "type": "agy_error"}})
                return
            raw = outer.get("response", "")
            parsed = extract_json(raw)
            usage = outer.get("usage") or {}
            cid = outer.get("conversation_id", "")
            if not isinstance(parsed, dict):
                # Treat plain text as a final answer rather than failing the turn.
                parsed = {"type": "final", "content": raw.strip()}
            if parsed.get("type") == "tool_call" and parsed.get("name"):
                args = parsed.get("arguments")
                if not isinstance(args, str):
                    args = json.dumps(args if args is not None else {})
                call_id = "call_agy_" + str(int(time.time() * 1000))
                msg = {
                    "role": "assistant",
                    "content": None,
                    "tool_calls": [{
                        "id": call_id,
                        "type": "function",
                        "function": {"name": parsed["name"], "arguments": args},
                    }],
                }
                resp = openai_response(model, msg, usage, "tool_calls", cid)
            else:
                content = parsed.get("content")
                if content is None:
                    content = raw
                resp = openai_response(model, {"role": "assistant", "content": str(content)}, usage, "stop", cid)
            log(f"ok model={model} finish={resp['choices'][0]['finish_reason']} secs={dt:.1f}")
            if req.get("stream"):
                self._send_sse(resp)
            else:
                self._send(200, resp)
        except Exception as e:
            log(f"ERROR {e}")
            self._send(502, {"error": {"message": str(e)}})

    def log_message(self, fmt, *args):
        log("http " + fmt % args)


if __name__ == "__main__":
    log(f"bridge listening on {HOST}:{PORT}")
    ThreadingHTTPServer((HOST, PORT), Handler).serve_forever()
