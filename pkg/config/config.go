package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/navikt/union-api/pkg/uctl"
)

type Config struct {
	// EntraIDTenantID is the EntraID tenant ID.
	EntraIDTenantID string
	// EntraIDClientID is the EntraID application (client) ID.
	EntraIDClientID string
	// EntraIDClientSecret is the EntraID client secret (injected from a K8s secret).
	EntraIDClientSecret string
	// BaseURL is the public base URL of this service, used to build the OAuth2 redirect URI.
	// Example: https://union-api.intern.nav.no
	BaseURL string
	// SessionSecret is used to sign session cookies. Must be a random string of at least
	// 32 characters.
	SessionSecret string
	// DevMode disables authentication and injects a stub principal. Never enable in production.
	DevMode bool
	// LogFormat controls the log output format. Valid values: "text" (default), "json".
	// Set LOG_FORMAT=json in production to emit structured JSON for log aggregators.
	LogFormat string

	UnionConfig uctl.UnionConfig
}

func (c *Config) IssuerURL() string {
	return fmt.Sprintf("https://login.microsoftonline.com/%s/v2.0", c.EntraIDTenantID)
}

func (c *Config) RedirectURL() string {
	return c.BaseURL + "/oauth2/callback"
}

// SecureCookies returns true when the service is running behind HTTPS.
// Cookies must not have the Secure flag over plain HTTP (e.g. localhost dev).
func (c *Config) SecureCookies() bool {
	return strings.HasPrefix(c.BaseURL, "https://")
}

func LoadConfig() (*Config, error) {
	devMode := os.Getenv("DEV_MODE") == "true"

	logFormat := os.Getenv("LOG_FORMAT")
	if logFormat == "" {
		logFormat = "text"
	}

	cfg := &Config{
		EntraIDTenantID:     os.Getenv("ENTRA_ID_TENANT_ID"),
		EntraIDClientID:     os.Getenv("ENTRA_ID_CLIENT_ID"),
		EntraIDClientSecret: os.Getenv("ENTRA_ID_CLIENT_SECRET"),
		BaseURL:             os.Getenv("BASE_URL"),
		SessionSecret:       os.Getenv("SESSION_SECRET"),
		DevMode:             devMode,
		LogFormat:           logFormat,
		UnionConfig: uctl.UnionConfig{
			ClientID:           os.Getenv("UNION_CLIENT_ID"),
			ClientSecretEnvVar: os.Getenv("UNION_CLIENT_SECRET_ENV_VAR"),
			Endpoint:           os.Getenv("UNION_ENDPOINT"),
			Org:                os.Getenv("UNION_ORG"),
		},
	}

	if devMode {
		return cfg, nil
	}

	if cfg.EntraIDTenantID == "" {
		return nil, fmt.Errorf("ENTRA_ID_TENANT_ID is required")
	}
	if cfg.EntraIDClientID == "" {
		return nil, fmt.Errorf("ENTRA_ID_CLIENT_ID is required")
	}
	if cfg.EntraIDClientSecret == "" {
		return nil, fmt.Errorf("ENTRA_ID_CLIENT_SECRET is required")
	}
	if cfg.BaseURL == "" {
		return nil, fmt.Errorf("BASE_URL is required")
	}
	if cfg.SessionSecret == "" {
		return nil, fmt.Errorf("SESSION_SECRET is required")
	}
	if cfg.UnionConfig.ClientID == "" {
		return nil, fmt.Errorf("UNION_CLIENT_ID is required")
	}
	if cfg.UnionConfig.ClientSecretEnvVar == "" {
		return nil, fmt.Errorf("UNION_CLIENT_SECRET_ENV_VAR is required")
	}
	if os.Getenv(cfg.UnionConfig.ClientSecretEnvVar) == "" {
		return nil, fmt.Errorf("%s is required", cfg.UnionConfig.ClientSecretEnvVar)
	}
	if cfg.UnionConfig.Endpoint == "" {
		return nil, fmt.Errorf("UNION_ENDPOINT is required")
	}
	if cfg.UnionConfig.Org == "" {
		return nil, fmt.Errorf("UNION_ORG is required")
	}

	return cfg, nil
}
