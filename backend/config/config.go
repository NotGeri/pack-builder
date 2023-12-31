package config

import (
	_ "embed"
	"fmt"
	"gopkg.in/yaml.v3"
	"os"
	"path/filepath"
	"strings"
)

//go:embed config.dist.yml
var configExample []byte

type token struct {
	Token string
}

type credentials struct {
	UserAgent  string `json:"user-agent"`
	GitHub     token
	CurseForge token
	Modrinth   token
}

type DomainProvider string

type ssl struct {
	Enabled  bool
	CertPath string `yaml:"cert-path"`
	KeyPath  string `yaml:"key-path"`
}

type web struct {
	Frontend  string
	SSL       ssl
	Address   string
	Port      int
	PublicUrl string `yaml:"public-url"`
}

type Config struct {
	Web         web
	Credentials credentials
}

// FormatEndpoint Removes trailing slashes
func (c Config) FormatEndpoint(endpoint string) string {
	if strings.HasPrefix(endpoint, "/") {
		endpoint = strings.TrimPrefix(endpoint, "/")
	}
	return endpoint
}

func LoadConfig() (cfg Config, err error) {

	configPath, err := filepath.Abs("config.yml")
	if err != nil {
		return
	}

	// Attempt to load the file
	data, err := os.ReadFile(configPath)
	if err != nil && os.IsNotExist(err) {

		// Config not found, creating default
		fmt.Println("Configuration found not found, generating new..")
		err = os.WriteFile("config.yml", configExample, 400)
		if err != nil {
			return
		}

		// Attempt to load the main config again
		data, err = os.ReadFile(configPath)
		if err != nil {
			return
		}
	}

	// We can set some default values here
	cfg = Config{}

	// Parse YAML
	err = yaml.Unmarshal(data, &cfg)
	if err != nil {
		return
	}

	return
}
