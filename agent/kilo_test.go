package agent

import (
	"os"
	"path/filepath"
	"testing"
)

func TestKiloConfigFromDefaults(t *testing.T) {
	t.Setenv("KILO_API_KEY", "kilo-test-token")
	c := ConfigFrom("kilo", "", "", "", "")
	if c.Format != FormatKilo {
		t.Fatalf("format=%v, want FormatKilo", c.Format)
	}
	if c.BaseURL != DefaultKiloBaseURL {
		t.Fatalf("base_url=%q, want %q", c.BaseURL, DefaultKiloBaseURL)
	}
	if c.Model != DefaultKiloModel {
		t.Fatalf("model=%q, want %q", c.Model, DefaultKiloModel)
	}
	if c.APIKey != "kilo-test-token" {
		t.Fatalf("api_key=%q", c.APIKey)
	}
	if c.ContextWindowK != DefaultKiloContextWindowK {
		t.Fatalf("context window=%dK", c.ContextWindowK)
	}
	if c.Stream {
		t.Fatal("Kilo is intentionally non-streaming")
	}
	if c.Provider() != "kilo" {
		t.Fatalf("Provider()=%q", c.Provider())
	}
	if _, err := c.NewProvider(); err != nil {
		t.Fatalf("NewProvider: %v", err)
	}
}

func TestKiloBaseURLNormalization(t *testing.T) {
	c := ConfigFrom("kilocode", DefaultKiloModel, "https://ai-gateway.kilo.ai/api/v1/chat/completions", "k", "")
	if c.Format != FormatKilo || c.BaseURL != DefaultKiloBaseURL {
		t.Fatalf("format=%v base_url=%q", c.Format, c.BaseURL)
	}
}

func TestKiloAccessTokenFromAuthFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "auth.json")
	if err := os.WriteFile(path, []byte(`{"kilo":{"type":"oauth","access":"file-token","refresh":"r"}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("KILO_API_KEY", "")
	t.Setenv("KILO_AUTH_FILE", path)
	if got := KiloAccessToken(); got != "file-token" {
		t.Fatalf("KiloAccessToken()=%q", got)
	}
}

// TestKiloFollowsRotatedCLILogin locks the fix for the 401 INVALID_TOKEN case:
// a profile may contain an old Kilo token, but after `kilo auth login` rotates
// auth.json the runtime must use the fresh CLI token without the user editing
// the profile again.
func TestKiloFollowsRotatedCLILogin(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "auth.json")
	if err := os.WriteFile(path, []byte(`{"kilo":{"access":"fresh-cli-token"}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("KILO_API_KEY", "")
	t.Setenv("KILO_AUTH_FILE", path)
	c := ConfigFrom("kilo", "", "", "stale-profile-token", "")
	if c.APIKey != "fresh-cli-token" {
		t.Fatalf("api_key=%q, want the fresh CLI token", c.APIKey)
	}
}

func TestKiloFromEnv(t *testing.T) {
	t.Setenv("ARTEX_LLM_PROVIDER", "Kilo")
	t.Setenv("ARTEX_LLM_MODEL", "")
	t.Setenv("ARTEX_LLM_BASE_URL", "")
	t.Setenv("ARTEX_LLM_API_KEY", "")
	t.Setenv("KILO_API_KEY", "env-kilo-token")
	t.Setenv("OPENAI_API_KEY", "unrelated")
	c, ok := FromEnv()
	if !ok {
		t.Fatal("FromEnv returned ok=false")
	}
	if c.Format != FormatKilo || c.APIKey != "env-kilo-token" {
		t.Fatalf("format=%v key=%q", c.Format, c.APIKey)
	}
	if c.Model != DefaultKiloModel || c.BaseURL != DefaultKiloBaseURL {
		t.Fatalf("model=%q base_url=%q", c.Model, c.BaseURL)
	}
}

func TestKiloRefusesEmptyKey(t *testing.T) {
	t.Setenv("KILO_API_KEY", "")
	t.Setenv("KILO_AUTH_FILE", filepath.Join(t.TempDir(), "missing.json"))
	c := ConfigFrom("kilo", "", "", "", "")
	if _, err := c.NewProvider(); err == nil {
		t.Fatal("NewProvider accepted an empty Kilo key")
	}
}
