package uctl

type UnionConfig struct {
	ClientID           string `yaml:"clientID"`
	ClientSecretEnvVar string `yaml:"clientSecretEnvVar"`
	Endpoint           string `yaml:"endpoint"`
	Org                string `yaml:"org"`
}
