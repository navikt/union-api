package config

import (
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/navikt/union-api/pkg/k8s"
	"github.com/navikt/union-api/pkg/uctl"
)

type Config struct {
	EntraID EntraIDConfig `yaml:"entraId"`

	Logging LoggingConfig `yaml:"logging"`
	// BaseURL is the public base URL of this service, used to build the OAuth2 redirect URI.
	// Example: https://union-api.intern.nav.no
	BaseURL string `yaml:"baseUrl"`
	// SessionSecret is used to sign session cookies. Must be a random string of at least
	// 32 characters.
	SessionSecret string `yaml:"sessionSecret"`
	// DevMode disables authentication and injects a stub principal. Never enable in production.
	DevMode bool `yaml:"devMode"`

	UnionConfig uctl.UnionConfig `yaml:"union"`

	K8sConfig k8s.K8sConfig `yaml:"k8s"`
}

type EntraIDConfig struct {
	TenantID     string `yaml:"tenantID"`
	ClientID     string `yaml:"clientID"`
	ClientSecret string `yaml:"clientSecret"`
}

type LoggingConfig struct {
	Format string `yaml:"format"`
	Level  slog.Level `yaml:"level"`
}


func (c *Config) IssuerURL() string {
	return fmt.Sprintf("https://login.microsoftonline.com/%s/v2.0", c.EntraID.TenantID)
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

	logLevelStr := os.Getenv("LOG_LEVEL")
	logLevel := slog.LevelInfo.Level()
	switch logLevelStr {
	case "debug":
		logLevel = slog.LevelDebug.Level()
	case "error":
		logLevel = slog.LevelError.Level()
	}

	cfg := &Config{
		EntraID: EntraIDConfig{
			TenantID:     os.Getenv("ENTRA_ID_TENANT_ID"),
			ClientID:     os.Getenv("ENTRA_ID_CLIENT_ID"),
			ClientSecret: os.Getenv("ENTRA_ID_CLIENT_SECRET"),
		},
		BaseURL:       os.Getenv("BASE_URL"),
		SessionSecret: os.Getenv("SESSION_SECRET"),
		DevMode:       devMode,
		LogFormat:     logFormat,
		LogLevel:      logLevel,
		UnionConfig: uctl.UnionConfig{
			ClientID:           os.Getenv("UNION_CLIENT_ID"),
			ClientSecretEnvVar: os.Getenv("UNION_CLIENT_SECRET_ENV_VAR"),
			Endpoint:           os.Getenv("UNION_ENDPOINT"),
			Org:                os.Getenv("UNION_ORG"),
		},
		K8sConfig: k8s.K8sConfig{
			FleetHostProjectNumber: os.Getenv("GKE_FLEET_HOST_PROJECT_NUMBER"),
			MembershipName:         os.Getenv("GKE_FLEET_MEMBERSHIP_NAME"),
			Location:               os.Getenv("GKE_FLEET_LOCATION"),
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
	if cfg.K8sConfig.FleetHostProjectNumber == "" {
		return nil, fmt.Errorf("GKE_FLEET_HOST_PROJECT_NUMBER is required")
	}
	if cfg.K8sConfig.Location == "" {
		return nil, fmt.Errorf("GKE_FLEET_LOCATION is required")
	}
	if cfg.K8sConfig.MembershipName == "" {
		return nil, fmt.Errorf("GKE_FLEET_MEMBERSHIP_NAME is required")
	}

	return cfg, nil
}
