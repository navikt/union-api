package config

import (
	"fmt"
	"os"
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
	// DevMode disables authentication and injects a stub principal. Never enable in production.
	DevMode bool
}

// IssuerURL returns the EntraID OIDC issuer URL for the configured tenant.
func (c *Config) IssuerURL() string {
	return fmt.Sprintf("https://login.microsoftonline.com/%s/v2.0", c.EntraIDTenantID)
}

// RedirectURL returns the OAuth2 callback URL.
func (c *Config) RedirectURL() string {
	return c.BaseURL + "/oauth2/callback"
}

// LoadConfig reads configuration from environment variables.
// It returns an error if any required variable is missing (unless DevMode is true).
func LoadConfig() (*Config, error) {
	devMode := os.Getenv("DEV_MODE") == "true"

	cfg := &Config{
		EntraIDTenantID:     os.Getenv("ENTRA_ID_TENANT_ID"),
		EntraIDClientID:     os.Getenv("ENTRA_ID_CLIENT_ID"),
		EntraIDClientSecret: os.Getenv("ENTRA_ID_CLIENT_SECRET"),
		BaseURL:             os.Getenv("BASE_URL"),
		DevMode:             devMode,
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

	return cfg, nil
}
