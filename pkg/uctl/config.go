package uctl

type UnionConfig struct {
	ClientID           string `yaml:"client_id"`
	ClientSecretEnvVar string `yaml:"client_secret_env_var"`
	Endpoint           string `yaml:"endpoint"`
	Org                string `yaml:"org"`
}
