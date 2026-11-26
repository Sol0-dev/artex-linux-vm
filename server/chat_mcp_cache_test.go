package server

import (
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Autumn-27/artex/db"
	actool "github.com/Autumn-27/norma/tool"
)

type fakeMCPClient struct {
	closed atomic.Int32
}

func (f *fakeMCPClient) Tools(context.Context) ([]actool.CoreTool, error) { return nil, nil }
func (f *fakeMCPClient) Close() error {
	f.closed.Add(1)
	return nil
}

func TestChatMCPCacheReusesAndClosesPerConversation(t *testing.T) {
	closeAllChatMCPSessions()
	t.Cleanup(closeAllChatMCPSessions)

	cl := &fakeMCPClient{}
	key := chatMCPKey("conv-1", "auto")
	e := &chatMCPEntry{clients: []mcpClient{cl}, lastUsed: time.Now()}
	if cur, stored := chatMCPStore(key, e); !stored || cur != e {
		t.Fatal("first store should win")
	}
	if got := chatMCPLookup(key); got != e {
		t.Fatal("lookup did not reuse the cached entry")
	}
	// A second turn builds new clients but must keep the first entry.
	cl2 := &fakeMCPClient{}
	if cur, stored := chatMCPStore(key, &chatMCPEntry{clients: []mcpClient{cl2}, lastUsed: time.Now()}); stored || cur != e {
		t.Fatal("duplicate store should return the existing entry")
	}
	if cl.closed.Load() != 0 {
		t.Fatal("cached client was closed between turns")
	}
	closeChatMCPSession("conv-1")
	if cl.closed.Load() != 1 {
		t.Fatalf("conversation delete closed client %d times, want 1", cl.closed.Load())
	}
	if got := chatMCPLookup(key); got != nil {
		t.Fatal("deleted conversation still has a cached entry")
	}
}

func TestMCPForChatSessionIsolatesBrowserProfile(t *testing.T) {
	dataDir := t.TempDir()
	m := &db.MCPServer{
		ID:        2,
		Name:      browserMCPName,
		Transport: "stdio",
		Args:      json.RawMessage(`["--browser","chromium","--user-data-dir","/old/profile","--no-sandbox"]`),
	}
	got := mcpForChatSession(m, "conv-7", dataDir)
	args := jsonStrSlice(got.Args)
	joined := strings.Join(args, " ")
	want := filepath.Join(dataDir, "playwright-mcp", "conv-7")
	if !strings.Contains(joined, "--user-data-dir "+want) {
		t.Fatalf("per-conversation profile missing: %s", joined)
	}
	if strings.Contains(joined, "/old/profile") {
		t.Fatalf("old shared profile was not removed: %s", joined)
	}
	if !strings.Contains(joined, "--no-sandbox") || !strings.Contains(joined, "--browser chromium") {
		t.Fatalf("browser flags lost: %s", joined)
	}
	if m.Name != browserMCPName || strings.Join(jsonStrSlice(m.Args), " ") == joined {
		t.Fatal("the stored MCP config was mutated")
	}
}

func TestMCPForChatSessionLeavesOtherServersAlone(t *testing.T) {
	m := &db.MCPServer{Name: "caido", Transport: "stdio", Args: json.RawMessage(`["serve"]`)}
	got := mcpForChatSession(m, "conv-7", t.TempDir())
	if got != m {
		t.Fatal("non-browser MCP should be returned unchanged")
	}
}
