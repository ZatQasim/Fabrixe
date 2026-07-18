package config

import (
	"crypto/rand"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config holds all Fabrixe configuration.
type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Database DatabaseConfig `yaml:"database"`
	TLS      TLSConfig      `yaml:"tls"`
	JWT      JWTConfig      `yaml:"jwt"`
	Security SecurityConfig `yaml:"security"`
	MDNS     MDNSConfig     `yaml:"mdns"`
	Modules  ModulesConfig  `yaml:"modules"`
}

type ServerConfig struct {
	Host             string `yaml:"host"`
	Port             int    `yaml:"port"`
	HTTPRedirectPort int    `yaml:"http_redirect_port"`
	StaticDir        string `yaml:"static_dir"`
}

type DatabaseConfig struct {
	Path string `yaml:"path"`
}

type TLSConfig struct {
	CertDir   string   `yaml:"cert_dir"`
	Hostnames []string `yaml:"hostnames"`
}

type JWTConfig struct {
	Secret          string `yaml:"secret"`
	AccessTokenTTL  int    `yaml:"access_token_ttl_minutes"`
	RefreshTokenTTL int    `yaml:"refresh_token_ttl_days"`
}

type SecurityConfig struct {
	AllowedOrigins  []string `yaml:"allowed_origins"`
	RateLimitPerMin int      `yaml:"rate_limit_per_minute"`
	SessionTimeout  int      `yaml:"session_timeout_minutes"`
	MaxLoginFails   int      `yaml:"max_login_attempts"`
	LockoutMinutes  int      `yaml:"lockout_minutes"`
}

type MDNSConfig struct {
	Hostname string `yaml:"hostname"`
}

type ModulesConfig struct {
	SystemManagement ModuleConfig `yaml:"system_management"`
	DeploymentAuto   ModuleConfig `yaml:"deployment_automation"`
	InternalSecurity ModuleConfig `yaml:"internal_security"`
	ProtectedComm    ModuleConfig `yaml:"protected_communication"`
}

type ModuleConfig struct {
	Enabled bool `yaml:"enabled"`
}

// Defaults returns a configuration with safe defaults.
func Defaults() *Config {
	return &Config{
		Server: ServerConfig{
			Host:             "0.0.0.0",
			Port:             8443,
			HTTPRedirectPort: 8080,
			StaticDir:        "/opt/fabrixe/web",
		},
		Database: DatabaseConfig{
			Path: "/var/lib/fabrixe/fabrixe.db",
		},
		TLS: TLSConfig{
			CertDir:   "/etc/fabrixe/certs",
			Hostnames: []string{"fabrixe.local", "localhost"},
		},
		JWT: JWTConfig{
			Secret:          "",
			AccessTokenTTL:  60,
			RefreshTokenTTL: 7,
		},
		Security: SecurityConfig{
			AllowedOrigins:  []string{"https://fabrixe.local", "https://localhost:8443"},
			RateLimitPerMin: 120,
			SessionTimeout:  480,
			MaxLoginFails:   5,
			LockoutMinutes:  15,
		},
		MDNS: MDNSConfig{
			Hostname: "fabrixe",
		},
		Modules: ModulesConfig{
			SystemManagement: ModuleConfig{Enabled: true},
			DeploymentAuto:   ModuleConfig{Enabled: true},
			InternalSecurity: ModuleConfig{Enabled: true},
			ProtectedComm:    ModuleConfig{Enabled: true},
		},
	}
}

// Load reads config from a YAML file, falling back to defaults for missing values.
func Load(path string) (*Config, error) {
	cfg := Defaults()

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Printf("[WARN] Config file not found at %s, using defaults.\n", path)
			if err := autoJWTSecret(cfg); err != nil {
				return nil, err
			}
			return cfg, nil
		}
		return nil, fmt.Errorf("reading config file: %w", err)
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parsing config file: %w", err)
	}

	if err := autoJWTSecret(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

// Save writes the current configuration to a YAML file.
func Save(cfg *Config, path string) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshalling config: %w", err)
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("writing config file: %w", err)
	}
	return nil
}

// autoJWTSecret generates and persists a JWT secret if one is not configured.
func autoJWTSecret(cfg *Config) error {
	if cfg.JWT.Secret != "" {
		return nil
	}

	secretFile := "/var/lib/fabrixe/.jwt_secret"
	data, err := os.ReadFile(secretFile)
	if err == nil && len(data) >= 32 {
		cfg.JWT.Secret = string(data)
		return nil
	}

	secretBytes := make([]byte, 64)
	if _, err := rand.Read(secretBytes); err != nil {
		return fmt.Errorf("generating JWT secret: %w", err)
	}

	secret := fmt.Sprintf("%x", secretBytes)
	if err := os.MkdirAll("/var/lib/fabrixe", 0700); err == nil {
		_ = os.WriteFile(secretFile, []byte(secret), 0600)
	}
	cfg.JWT.Secret = secret
	return nil
}
