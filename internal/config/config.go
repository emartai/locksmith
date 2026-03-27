package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

// Config controls rule overrides and CLI path filtering.
type Config struct {
	Rules       map[string]string `mapstructure:"rules"`
	IgnorePaths []string          `mapstructure:"ignore_paths"`
	DatabaseURL string            `mapstructure:"database_url"`
}

// LoadConfig loads locksmith configuration from disk or returns defaults when none is present.
func LoadConfig(path string) (*Config, error) {
	cfg := &Config{
		Rules:       map[string]string{},
		IgnorePaths: []string{},
	}

	v := viper.New()
	v.SetConfigType("yaml")

	if path != "" {
		absPath, err := filepath.Abs(path)
		if err != nil {
			return nil, fmt.Errorf("resolve config path %s: %w", path, err)
		}
		if _, err := os.Stat(absPath); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return nil, fmt.Errorf("config file not found: %s", path)
			}
			return nil, fmt.Errorf("stat config %s: %w", path, err)
		}
		v.SetConfigFile(absPath)
	} else {
		v.SetConfigName("locksmith")
		v.AddConfigPath(".")
	}

	if err := v.ReadInConfig(); err != nil {
		if path == "" {
			var notFound viper.ConfigFileNotFoundError
			if errors.As(err, &notFound) {
				return cfg, nil
			}
		}
		return nil, fmt.Errorf("read config: %w", err)
	}

	if err := v.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}

	expandEnv(cfg)
	normalizeRuleOverrides(cfg)
	return cfg, nil
}

func expandEnv(cfg *Config) {
	cfg.DatabaseURL = os.ExpandEnv(cfg.DatabaseURL)

	for key, value := range cfg.Rules {
		cfg.Rules[key] = os.ExpandEnv(value)
	}

	for i, value := range cfg.IgnorePaths {
		cfg.IgnorePaths[i] = os.ExpandEnv(value)
	}
}

func normalizeRuleOverrides(cfg *Config) {
	if cfg.Rules == nil {
		cfg.Rules = map[string]string{}
		return
	}

	normalized := make(map[string]string, len(cfg.Rules))
	for key, value := range cfg.Rules {
		normalized[strings.TrimSpace(strings.ToUpper(key))] = strings.TrimSpace(strings.ToLower(value))
	}
	cfg.Rules = normalized
}
