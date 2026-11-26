-- Kilo Code gateway profiles (Baidu ERNIE-5.5-Flash-Preview free, native tool calls).
ALTER TABLE llm_profiles DROP CONSTRAINT IF EXISTS llm_profiles_format_check;
ALTER TABLE llm_profiles ADD CONSTRAINT llm_profiles_format_check
    CHECK (format IN ('openai','anthropic','openai-responses','ollama','bedrock-gateway','opencode-go','antigravity','kilo'));
