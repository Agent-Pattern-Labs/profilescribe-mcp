package bridge

import "testing"

func TestConfigFromEnvDefaultsToProduction(t *testing.T) {
	t.Setenv("PROFILESCRIBE_AGENT_TOKEN", "  psagt_test  ")
	t.Setenv("PROFILESCRIBE_MCP_URL", "")
	t.Setenv("PROFILESCRIBE_API_URL", "")

	cfg := ConfigFromEnv().normalized()
	if cfg.MCPURL != DefaultMCPURL {
		t.Fatalf("MCPURL = %q, want %q", cfg.MCPURL, DefaultMCPURL)
	}
	if cfg.AgentToken != "psagt_test" {
		t.Fatalf("AgentToken = %q", cfg.AgentToken)
	}
}

func TestConfigFromEnvUsesAPIURLForLocalDevelopment(t *testing.T) {
	t.Setenv("PROFILESCRIBE_AGENT_TOKEN", "psagt_test")
	t.Setenv("PROFILESCRIBE_MCP_URL", "")
	t.Setenv("PROFILESCRIBE_API_URL", " http://localhost:8080/ ")

	cfg := ConfigFromEnv().normalized()
	if cfg.MCPURL != "http://localhost:8080/api/mcp" {
		t.Fatalf("MCPURL = %q", cfg.MCPURL)
	}
}

func TestConfigFromEnvPrefersExplicitMCPURL(t *testing.T) {
	t.Setenv("PROFILESCRIBE_AGENT_TOKEN", "psagt_test")
	t.Setenv("PROFILESCRIBE_MCP_URL", " https://example.com/mcp ")
	t.Setenv("PROFILESCRIBE_API_URL", "http://localhost:8080")

	cfg := ConfigFromEnv().normalized()
	if cfg.MCPURL != "https://example.com/mcp" {
		t.Fatalf("MCPURL = %q", cfg.MCPURL)
	}
}

func TestConfigValidateRequiresToken(t *testing.T) {
	err := Config{MCPURL: DefaultMCPURL}.validate()
	if err == nil {
		t.Fatal("expected missing token error")
	}
}
