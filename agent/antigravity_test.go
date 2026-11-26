package agent

import "testing"

// TestAntigravityConfigFromDefaults locks the local agy bridge preset: an empty
// form resolves the bridge URL, the medium Gemini 4 Hydrogen model, the
// placeholder token, non-streaming mode and the 1M context window.
func TestAntigravityConfigFromDefaults(t *testing.T) {
	c := ConfigFrom("antigravity", "", "", "", "")
	if c.Format != FormatAntigravity {
		t.Fatalf("format=%v, want FormatAntigravity", c.Format)
	}
	if c.BaseURL != DefaultAntigravityBaseURL {
		t.Fatalf("base_url=%q, want %q", c.BaseURL, DefaultAntigravityBaseURL)
	}
	if c.Model != DefaultAntigravityModel {
		t.Fatalf("model=%q, want %q", c.Model, DefaultAntigravityModel)
	}
	if c.APIKey != DefaultAntigravityAPIKey {
		t.Fatalf("api_key=%q, want the local placeholder", c.APIKey)
	}
	if c.Stream {
		t.Fatal("the agy bridge is one-shot; Stream must be false")
	}
	if c.ContextWindowK != DefaultAntigravityContextWindowK {
		t.Fatalf("context window=%dK, want %dK", c.ContextWindowK, DefaultAntigravityContextWindowK)
	}
	if c.Provider() != "antigravity" {
		t.Fatalf("Provider()=%q", c.Provider())
	}
	if _, err := c.NewProvider(); err != nil {
		t.Fatalf("NewProvider: %v", err)
	}
}

// TestAntigravityBaseURLAndAlias tolerates a pasted endpoint and the short
// "agy" environment/API alias.
func TestAntigravityBaseURLAndAlias(t *testing.T) {
	c := ConfigFrom("agy", DefaultAntigravityModel, "http://127.0.0.1:8790/v1/chat/completions", "", "")
	if c.Format != FormatAntigravity {
		t.Fatalf("format=%v", c.Format)
	}
	if c.BaseURL != DefaultAntigravityBaseURL {
		t.Fatalf("base_url=%q, want %q", c.BaseURL, DefaultAntigravityBaseURL)
	}
	if c.APIKey != DefaultAntigravityAPIKey {
		t.Fatalf("api_key=%q", c.APIKey)
	}
}

func TestAntigravityFromEnv(t *testing.T) {
	t.Setenv("ARTEX_LLM_PROVIDER", " Antigravity ")
	t.Setenv("ARTEX_LLM_MODEL", "gemini-3.8-flash-medium")
	t.Setenv("ARTEX_LLM_BASE_URL", "")
	t.Setenv("ARTEX_LLM_API_KEY", "")
	t.Setenv("AGY_API_KEY", "")
	t.Setenv("OPENAI_API_KEY", "unrelated-openai-key")

	c, ok := FromEnv()
	if !ok {
		t.Fatal("FromEnv returned ok=false")
	}
	if c.Format != FormatAntigravity || c.APIKey != DefaultAntigravityAPIKey {
		t.Fatalf("format=%v key=%q", c.Format, c.APIKey)
	}
	if c.Model != "gemini-3.8-flash-medium" || c.BaseURL != DefaultAntigravityBaseURL {
		t.Fatalf("model=%q base_url=%q", c.Model, c.BaseURL)
	}
	if c.Stream {
		t.Fatal("Stream must be false for the agy bridge")
	}
}
