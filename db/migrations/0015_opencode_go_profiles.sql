-- OpenCode Go subscription profiles. Go is an OpenAI-compatible endpoint that
-- requires a stable x-opencode-session header and a client-specific User-Agent;
-- ARTEX's 'opencode-go' format supplies those defaults and targets the
-- DeepSeek V4.1 Flash 1017 model.
ALTER TABLE llm_profiles DROP CONSTRAINT IF EXISTS llm_profiles_format_check;
ALTER TABLE llm_profiles ADD CONSTRAINT llm_profiles_format_check
    CHECK (format IN ('openai','anthropic','openai-responses','ollama','bedrock-gateway','opencode-go'));
