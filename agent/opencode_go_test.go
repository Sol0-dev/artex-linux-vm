package agent

import "testing"

// TestOpenCodeGoConfigFromDefaults locks the OpenCode Go preset: an empty form
// still resolves the Go endpoint, the DeepSeek V4.1 Flash 1017 model, the
// mandatory session header, a client User-Agent and the 1M context window.
func TestOpenCodeGoConfigFromDefaults(t *testing.T) {
	c := ConfigFrom("opencode-go", "", "", "", "")
	if c.Format != FormatOpenCodeGo {
		t.Fatalf("format=%v, want FormatOpenCodeGo", c.Format)
	}
	if c.BaseURL != DefaultOpenCodeGoBaseURL {
		t.Fatalf("base_url=%q, want %q", c.BaseURL, DefaultOpenCodeGoBaseURL)
	}
	if c.Model != DefaultOpenCodeGoModel {
		t.Fatalf("model=%q, want %q", c.Model, DefaultOpenCodeGoModel)
	}
	if c.SessionHeaderKey != DefaultOpenCodeGoSessionHeader {
		t.Fatalf("session header=%q, want %q", c.SessionHeaderKey, DefaultOpenCodeGoSessionHeader)
	}
	if c.UserAgent != DefaultOpenCodeGoUserAgent {
		t.Fatalf("user agent=%q, want %q", c.UserAgent, DefaultOpenCodeGoUserAgent)
	}
	if c.ContextWindowK != DefaultOpenCodeGoContextWindowK {
		t.Fatalf("context window=%dK, want %dK", c.ContextWindowK, DefaultOpenCodeGoContextWindowK)
	}
	if c.Provider() != "opencode-go" {
		t.Fatalf("Provider()=%q", c.Provider())
	}
	if !c.NeedsAPIKey() {
		t.Fatal("OpenCode Go must require an API key")
	}
}

// TestOpenCodeGoBaseURLNormalization accepts a pasted endpoint and a bare Go
// root, both of which must resolve to the /v1 root the provider appends to.
func TestOpenCodeGoBaseURLNormalization(t *testing.T) {
	cases := map[string]string{
		"https://opencode.ai/inference/go/openai/v1/chat/completions": "https://opencode.ai/inference/go/openai/v1",
		"https://opencode.ai/inference/go":                            "https://opencode.ai/inference/go/v1",
		"https://opencode.ai/inference/go/":                           "https://opencode.ai/inference/go/v1",
		"https://opencode.ai/inference/go/v1/":                        "https://opencode.ai/inference/go/v1",
	}
	for in, want := range cases {
		c := ConfigFrom("opencode-go", "", in, "key", "")
		if c.BaseURL != want {
			t.Errorf("ConfigFrom base_url(%q)=%q, want %q", in, c.BaseURL, want)
		}
	}
}

// TestOpenCodeGoPreservesExplicitHeaders keeps operator overrides instead of
// forcing the code-owned defaults.
func TestOpenCodeGoPreservesExplicitHeaders(t *testing.T) {
	c := ConfigFrom("opencode-go", DefaultOpenCodeGoModel, "", "key", "")
	c.SessionHeaderKey = "x-custom-session"
	c.UserAgent = "my-agent/2.0"
	c.ContextWindowK = 128
	c.ApplyProviderDefaults()
	if c.SessionHeaderKey != "x-custom-session" || c.UserAgent != "my-agent/2.0" || c.ContextWindowK != 128 {
		t.Fatalf("explicit values were overwritten: header=%q ua=%q ctx=%d", c.SessionHeaderKey, c.UserAgent, c.ContextWindowK)
	}
}

// TestOpenCodeGoRefusesEmptyKey prevents the OpenAI SDK from falling back to
// OPENAI_API_KEY and shipping an unrelated credential to OpenCode.
func TestOpenCodeGoRefusesEmptyKey(t *testing.T) {
	c := ConfigFrom("opencode-go", "", "", "", "")
	if _, err := c.NewProvider(); err == nil {
		t.Fatal("NewProvider accepted an empty OpenCode Go key")
	}
	if _, err := ConfigFrom("opencode-go", "", "", "key", "").NewProvider(); err != nil {
		t.Fatalf("NewProvider with a key: %v", err)
	}
}

// TestOpenCodeGoFromEnv resolves the environment path and deliberately ignores
// ANTHROPIC_API_KEY / OPENAI_API_KEY.
func TestOpenCodeGoFromEnv(t *testing.T) {
	t.Setenv("ARTEX_LLM_PROVIDER", " OpenCode-Go ")
	t.Setenv("ARTEX_LLM_MODEL", "")
	t.Setenv("ARTEX_LLM_BASE_URL", "")
	t.Setenv("ARTEX_LLM_API_KEY", "")
	t.Setenv("OPENCODE_API_KEY", "go-test-key")
	t.Setenv("ANTHROPIC_API_KEY", "unrelated-anthropic-key")
	t.Setenv("OPENAI_API_KEY", "unrelated-openai-key")

	c, ok := FromEnv()
	if !ok {
		t.Fatal("FromEnv returned ok=false")
	}
	if c.Format != FormatOpenCodeGo || c.APIKey != "go-test-key" {
		t.Fatalf("format=%v key=%q", c.Format, c.APIKey)
	}
	if c.BaseURL != DefaultOpenCodeGoBaseURL || c.Model != DefaultOpenCodeGoModel {
		t.Fatalf("base_url=%q model=%q", c.BaseURL, c.Model)
	}
	if c.SessionHeaderKey != DefaultOpenCodeGoSessionHeader || c.UserAgent != DefaultOpenCodeGoUserAgent {
		t.Fatalf("session header=%q user agent=%q", c.SessionHeaderKey, c.UserAgent)
	}
}
