package config

import (
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/go-viper/mapstructure/v2"
	"github.com/navikt/union-api/pkg/k8s"
	"github.com/navikt/union-api/pkg/uctl"
	"github.com/spf13/viper"
)

type Config struct {
	EntraID EntraIDConfig `yaml:"entra_id"`

	Logging LoggingConfig `yaml:"logging"`
	// BaseURL is the public base URL of this service, used to build the OAuth2 redirect URI.
	// Example: https://union-api.intern.nav.no
	BaseURL string `yaml:"base_url"`
	// SessionSecret is used to sign session cookies. Must be a random string of at least
	// 32 characters.
	SessionSecret string `yaml:"session_secret"`
	// DevMode disables authentication and injects a stub principal. Never enable in production.
	DevMode bool `yaml:"dev_mode"`

	UnionConfig uctl.UnionConfig `yaml:"union"`

	GKEConfig k8s.GKEConfig `yaml:"gke"`
}

type EntraIDConfig struct {
	TenantID     string `yaml:"tenant_id"`
	ClientID     string `yaml:"client_id"`
	ClientSecret string `yaml:"client_secret"`
}

type LoggingConfig struct {
	Format string     `yaml:"format"`
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
	if cf := os.Getenv("CONFIG_FILE"); cf != "" {
		viper.SetConfigFile(cf)
		if err := viper.ReadInConfig(); err != nil {
			return nil, fmt.Errorf("unable to read config file: %w", err)
		}
	}
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Secrets are injected via environment variables only and must never live in
	// the config file (a ConfigMap in production). viper.Unmarshal decodes only
	// the keys returned by AllKeys(), which does not enumerate AutomaticEnv
	// variables; each secret must therefore be bound explicitly, or it is
	// silently dropped during Unmarshal whenever the key is absent from the file.
	if err := viper.BindEnv("session_secret", "SESSION_SECRET"); err != nil {
		return nil, fmt.Errorf("unable to bind SESSION_SECRET: %w", err)
	}
	if err := viper.BindEnv("entra_id.client_secret", "ENTRA_ID_CLIENT_SECRET"); err != nil {
		return nil, fmt.Errorf("unable to bind ENTRA_ID_CLIENT_SECRET: %w", err)
	}

	viper.SetDefault("logging.format", "text")
	viper.SetDefault("logging.level", "INFO")

	var cfg Config
	err := viper.Unmarshal(&cfg, func(dc *mapstructure.DecoderConfig) {
		dc.TagName = "yaml"
		dc.DecodeHook = mapstructure.ComposeDecodeHookFunc(
			mapstructure.TextUnmarshallerHookFunc(),
			dc.DecodeHook,
		)
	})

	if err != nil {
		return nil, fmt.Errorf("unable to unmarshal config: %w", err)
	}

	if cfg.DevMode {
		return &cfg, nil
	}

	if cfg.EntraID.TenantID == "" {
		return nil, fmt.Errorf("entra_id.tenant_id (ENTRA_ID_TENANT_ID) is required")
	}
	if cfg.EntraID.ClientID == "" {
		return nil, fmt.Errorf("entra_id.client_id (ENTRA_ID_CLIENT_ID) is required")
	}
	if cfg.EntraID.ClientSecret == "" {
		return nil, fmt.Errorf("entra_id.client_secret (ENTRA_ID_CLIENT_SECRET) is required")
	}
	if cfg.BaseURL == "" {
		return nil, fmt.Errorf("base_url (BASE_URL) is required")
	}
	if cfg.SessionSecret == "" {
		return nil, fmt.Errorf("session_secret (SESSION_SECRET) is required")
	}
	if len(cfg.SessionSecret) < 32 {
		return nil, fmt.Errorf("session_secret (SESSION_SECRET) must be at least 32 characters")
	}
	if cfg.UnionConfig.ClientID == "" {
		return nil, fmt.Errorf("union.client_id (UNION_CLIENT_ID) is required")
	}
	if cfg.UnionConfig.ClientSecretEnvVar == "" {
		return nil, fmt.Errorf("union.client_secret_env_var (UNION_CLIENT_SECRET_ENV_VAR) is required")
	}
	if os.Getenv(cfg.UnionConfig.ClientSecretEnvVar) == "" {
		return nil, fmt.Errorf("%s is required", cfg.UnionConfig.ClientSecretEnvVar)
	}
	if cfg.UnionConfig.Endpoint == "" {
		return nil, fmt.Errorf("union.endpoint (UNION_ENDPOINT) is required")
	}
	if cfg.UnionConfig.Org == "" {
		return nil, fmt.Errorf("union.org (UNION_ORG) is required")
	}
	if cfg.GKEConfig.FleetHostProjectNumber == "" {
		return nil, fmt.Errorf("gke.fleet_host_project_number (GKE_FLEET_HOST_PROJECT_NUMBER) is required")
	}
	if cfg.GKEConfig.MembershipName == "" {
		return nil, fmt.Errorf("gke.membership_name (GKE_FLEET_MEMBERSHIP_NAME) is required")
	}
	if cfg.GKEConfig.Location == "" {
		return nil, fmt.Errorf("gke.location (GKE_FLEET_LOCATION) is required")
	}

	return &cfg, nil
}
