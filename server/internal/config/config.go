package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

type Config struct {
	Server   ServerConfig   `toml:"server"`
	Database DatabaseConfig `toml:"database"`
	Auth     AuthConfig     `toml:"auth"`
}

type ServerConfig struct {
	Listen string `toml:"listen"`
}

type DatabaseConfig struct {
	Path string `toml:"path"`
}

type AuthConfig struct {
	PrivateKeyPath      string `toml:"private_key"`
	AccessTokenExpiry   string `toml:"access_token_expiry"`
	RefreshTokenExpiry  string `toml:"refresh_token_expiry"`
}

func defaults() Config {
	return Config{
		Server: ServerConfig{
			Listen: "127.0.0.1:8080",
		},
		Database: DatabaseConfig{
			Path: "notesd.db",
		},
		Auth: AuthConfig{
			PrivateKeyPath:     "notesd.key",
			AccessTokenExpiry:  "15m",
			RefreshTokenExpiry: "720h",
		},
	}
}

// Load reads configuration from TOML files.
// It checks $HOME/.notesd.conf first, then $PWD/notesd.conf.
// Values from the later file override the earlier one.
func Load() (Config, error) {
	cfg := defaults()

	home, err := os.UserHomeDir()
	if err == nil {
		_ = loadFile(filepath.Join(home, ".notesd.conf"), &cfg)
	}

	pwd, err := os.Getwd()
	if err == nil {
		_ = loadFile(filepath.Join(pwd, "notesd.conf"), &cfg)
	}

	if err := validate(cfg); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func loadFile(path string, cfg *Config) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = toml.NewDecoder(f).Decode(cfg)
	return err
}

func validate(cfg Config) error {
	if cfg.Server.Listen == "" {
		return fmt.Errorf("server.listen must not be empty")
	}
	if cfg.Database.Path == "" {
		return fmt.Errorf("database.path must not be empty")
	}
	if cfg.Auth.PrivateKeyPath == "" {
		return fmt.Errorf("auth.private_key must not be empty")
	}
	return nil
}
