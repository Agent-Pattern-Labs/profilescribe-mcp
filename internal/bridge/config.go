package bridge

import (
	"errors"
	"os"
	"strings"
	"time"
)

const (
	DefaultMCPURL  = "https://profilescribe.com/api/mcp"
	defaultTimeout = 30 * time.Second
)

type Config struct {
	MCPURL     string
	AgentToken string
	Timeout    time.Duration
}

func ConfigFromEnv() Config {
	mcpURL := strings.TrimSpace(os.Getenv("PROFILESCRIBE_MCP_URL"))
	if mcpURL == "" {
		apiURL := strings.TrimSpace(os.Getenv("PROFILESCRIBE_API_URL"))
		if apiURL == "" {
			mcpURL = DefaultMCPURL
		} else {
			mcpURL = strings.TrimRight(apiURL, "/") + "/api/mcp"
		}
	}

	return Config{
		MCPURL:     mcpURL,
		AgentToken: strings.TrimSpace(os.Getenv("PROFILESCRIBE_AGENT_TOKEN")),
		Timeout:    defaultTimeout,
	}
}

func (c Config) validate() error {
	if strings.TrimSpace(c.AgentToken) == "" {
		return errors.New("PROFILESCRIBE_AGENT_TOKEN is required")
	}
	if strings.TrimSpace(c.MCPURL) == "" {
		return errors.New("ProfileScribe MCP URL is required")
	}
	return nil
}

func (c Config) normalized() Config {
	c.MCPURL = strings.TrimSpace(c.MCPURL)
	c.AgentToken = strings.TrimSpace(c.AgentToken)
	if c.Timeout <= 0 {
		c.Timeout = defaultTimeout
	}
	return c
}
