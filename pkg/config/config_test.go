package config_test

import (
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/navikt/union-api/pkg/config"
)

// validConfigYAML mirrors production: non-secret keys come from the file (a
// ConfigMap mounted as a file), while secrets are injected via the environment.
// Note the gke.* keys use the actual struct tags (fleet_location,
// fleet_membership_name), not the names shown in the validation error messages.
const validConfigYAML = `entra_id:
  tenant_id: test-tenant
  client_id: test-client

base_url: https://union-api.example.test
dev_mode: false

logging:
  format: text
  level: INFO

union:
  client_id: union-client
  client_secret_env_var: UNION_CLIENT_SECRET
  endpoint: union.example.test
  org: test-org

gke:
  fleet_host_project_number: "123456789"
  fleet_membership_name: test-membership
  fleet_location: europe-north1
`

const (
	testSessionSecret = "0123456789abcdef0123456789abcdef" // exactly 32 chars
	testEntraSecret   = "entra-client-secret-value"
	testUnionSecret   = "union-client-secret-value"
)

// configEnvVars is every environment variable LoadConfig may read. Tests
// neutralise the whole set before each run so AutomaticEnv cannot pick up stray
// values from the developer's shell or CI. viper treats an empty env var as
// unset (allowEmptyEnv is false), so setting "" is equivalent to unsetting,
// and t.Setenv restores the original value when the test finishes.
var configEnvVars = []string{
	"ENTRA_ID_TENANT_ID",
	"ENTRA_ID_CLIENT_ID",
	"ENTRA_ID_CLIENT_SECRET",
	"BASE_URL",
	"SESSION_SECRET",
	"DEV_MODE",
	"LOGGING_FORMAT",
	"LOGGING_LEVEL",
	"UNION_CLIENT_ID",
	"UNION_CLIENT_SECRET_ENV_VAR",
	"UNION_CLIENT_SECRET",
	"UNION_ENDPOINT",
	"UNION_ORG",
	"GKE_FLEET_HOST_PROJECT_NUMBER",
	"GKE_FLEET_MEMBERSHIP_NAME",
	"GKE_FLEET_LOCATION",
}

// clearEnv blanks every config env var so the test starts from a known state.
// Must be called before any test-specific t.Setenv calls.
func clearEnv(t *testing.T) {
	t.Helper()
	for _, k := range configEnvVars {
		t.Setenv(k, "")
	}
}

// setSecrets injects the three secrets that production supplies via the
// environment (and which are intentionally absent from the config file).
func setSecrets(t *testing.T) {
	t.Helper()
	t.Setenv("SESSION_SECRET", testSessionSecret)
	t.Setenv("ENTRA_ID_CLIENT_SECRET", testEntraSecret)
	t.Setenv("UNION_CLIENT_SECRET", testUnionSecret)
}

// writeTempConfig writes yaml to a temp file and returns its path.
func writeTempConfig(t *testing.T, yaml string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(yaml), 0o600); err != nil {
		t.Fatalf("write temp config: %v", err)
	}
	return path
}

// TestLoadConfig_SecretsResolveFromEnv guards against the viper.Unmarshal
// pitfall (AllKeys does not enumerate AutomaticEnv vars): the BindEnv calls must
// make env-only secrets survive Unmarshal even though they are absent from the
// config file.
func TestLoadConfig_SecretsResolveFromEnv(t *testing.T) {
	clearEnv(t)
	setSecrets(t)

	cfg, err := config.LoadConfig(writeTempConfig(t, validConfigYAML))
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if cfg.SessionSecret != testSessionSecret {
		t.Errorf("SessionSecret = %q, want %q", cfg.SessionSecret, testSessionSecret)
	}
	if cfg.EntraID.ClientSecret != testEntraSecret {
		t.Errorf("EntraID.ClientSecret = %q, want %q", cfg.EntraID.ClientSecret, testEntraSecret)
	}
}

func TestLoadConfig_ShortSessionSecretRejected(t *testing.T) {
	clearEnv(t)
	setSecrets(t)
	t.Setenv("SESSION_SECRET", "too-short")

	_, err := config.LoadConfig(writeTempConfig(t, validConfigYAML))
	if err == nil {
		t.Fatal("LoadConfig() error = nil, want error for short session secret")
	}
	if !strings.Contains(err.Error(), "at least 32") {
		t.Errorf("error = %q, want it to mention 'at least 32'", err)
	}
}

func TestLoadConfig_EnvOverridesFile(t *testing.T) {
	clearEnv(t)
	setSecrets(t)
	const override = "https://override.example.test"
	t.Setenv("BASE_URL", override)

	cfg, err := config.LoadConfig(writeTempConfig(t, validConfigYAML))
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if cfg.BaseURL != override {
		t.Errorf("BaseURL = %q, want %q (env should override file)", cfg.BaseURL, override)
	}
}

func TestLoadConfig_DevModeSkipsValidation(t *testing.T) {
	clearEnv(t)
	// No secrets and no required fields: validation must be skipped entirely.
	cfg, err := config.LoadConfig(writeTempConfig(t, "dev_mode: true\n"))
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if !cfg.DevMode {
		t.Error("DevMode = false, want true")
	}
}

func TestLoadConfig_DefaultsApplied(t *testing.T) {
	clearEnv(t)
	setSecrets(t)
	yaml := strings.Replace(validConfigYAML, "logging:\n  format: text\n  level: INFO\n", "", 1)

	cfg, err := config.LoadConfig(writeTempConfig(t, yaml))
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if cfg.Logging.Format != "text" {
		t.Errorf("Logging.Format = %q, want \"text\"", cfg.Logging.Format)
	}
	if cfg.Logging.Level != slog.LevelInfo {
		t.Errorf("Logging.Level = %v, want %v", cfg.Logging.Level, slog.LevelInfo)
	}
}

// TestLoadConfig_LogLevelDebug exercises the TextUnmarshaller decode hook that
// turns the "DEBUG" string into a slog.Level.
func TestLoadConfig_LogLevelDebug(t *testing.T) {
	clearEnv(t)
	setSecrets(t)
	yaml := strings.Replace(validConfigYAML, "level: INFO", "level: DEBUG", 1)

	cfg, err := config.LoadConfig(writeTempConfig(t, yaml))
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if cfg.Logging.Level != slog.LevelDebug {
		t.Errorf("Logging.Level = %v, want %v", cfg.Logging.Level, slog.LevelDebug)
	}
}

func TestLoadConfig_MissingRequiredFieldErrors(t *testing.T) {
	clearEnv(t)
	setSecrets(t)
	yaml := strings.Replace(validConfigYAML, "base_url: https://union-api.example.test\n", "", 1)

	_, err := config.LoadConfig(writeTempConfig(t, yaml))
	if err == nil {
		t.Fatal("LoadConfig() error = nil, want error for missing base_url")
	}
	if !strings.Contains(err.Error(), "base_url") {
		t.Errorf("error = %q, want it to mention 'base_url'", err)
	}
}
