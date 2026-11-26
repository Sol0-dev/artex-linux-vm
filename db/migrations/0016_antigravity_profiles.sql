-- Antigravity CLI profiles (local agy bridge: OpenAI-compatible, medium-only
-- Gemini 4 Hydrogen / Gemini 3.8 Flash).
ALTER TABLE llm_profiles DROP CONSTRAINT IF EXISTS llm_profiles_format_check;
ALTER TABLE llm_profiles ADD CONSTRAINT llm_profiles_format_check
    CHECK (format IN ('openai','anthropic','openai-responses','ollama','bedrock-gateway','opencode-go','antigravity'));
