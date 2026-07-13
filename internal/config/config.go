package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Listen      string        `yaml:"listen"`
	AdminListen string        `yaml:"admin_listen"`
	Domain      string        `yaml:"domain"`
	Upstreams   []Upstream    `yaml:"upstreams"`
	Cache       CacheConfig   `yaml:"cache"`
	Filter      FilterConfig  `yaml:"filter"`
	Admin       AdminConfig   `yaml:"admin"`
	TLS         TLSConfig     `yaml:"tls"`
	Logging     LoggingConfig `yaml:"logging"`
}

type AdminConfig struct {
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

type Upstream struct {
	Name     string `yaml:"name"`
	Address  string `yaml:"address"`
	Protocol string `yaml:"protocol"`
	Weight   int    `yaml:"weight"`
}

type CacheConfig struct {
	Enabled    bool `yaml:"enabled"`
	MaxEntries int  `yaml:"max_entries"`
	DefaultTTL int  `yaml:"default_ttl"`
}

type FilterConfig struct {
	Enabled       bool   `yaml:"enabled"`
	BlocklistPath string `yaml:"blocklist_path"`
}

type TLSConfig struct {
	Email    string `yaml:"email"`
	CacheDir string `yaml:"cache_dir"`
	CertFile string `yaml:"cert_file"`
	KeyFile  string `yaml:"key_file"`
}

type LoggingConfig struct {
	Level         string `yaml:"level"`
	QueryLog      bool   `yaml:"query_log"`
	MaxLogEntries int    `yaml:"max_log_entries"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	cfg := &Config{
		Listen:      "0.0.0.0:443",
		AdminListen: "127.0.0.1:8443",
		Cache: CacheConfig{
			Enabled:    true,
			MaxEntries: 10000,
			DefaultTTL: 3600,
		},
		TLS: TLSConfig{
			CacheDir: "./certs",
		},
		Logging: LoggingConfig{
			Level:         "info",
			QueryLog:      true,
			MaxLogEntries: 10000,
		},
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	if len(cfg.Upstreams) == 0 {
		cfg.Upstreams = []Upstream{
			{Name: "google", Address: "8.8.8.8:53", Protocol: "udp", Weight: 1},
			{Name: "cloudflare", Address: "1.1.1.1:53", Protocol: "udp", Weight: 1},
		}
	}

	return cfg, nil
}
