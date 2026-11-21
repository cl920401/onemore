package config

import (
	"fmt"
	"net"
	"os"
	"strings"
)

// Config is injected into business packages; business code must not read env directly.
type Config struct {
	Addr        string
	DSN         string
	AdminToken  string
	MasterKey   string
	WebDir      string
	UpstreamURL string
}

func Load() Config {
	return Config{
		Addr:        env("ONEMORE_ADDR", "127.0.0.1:8080"),
		DSN:         env("ONEMORE_DSN", "onemore:onemore@tcp(127.0.0.1:3306)/onemore?parseTime=true&multiStatements=false"),
		AdminToken:  env("ONEMORE_ADMIN_TOKEN", "dev-admin-token"),
		MasterKey:   env("ONEMORE_MASTER_KEY", ""),
		WebDir:      env("ONEMORE_WEB_DIR", "web/dist"),
		UpstreamURL: env("ONEMORE_UPSTREAM_URL", ""),
	}
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// Validate permits development credentials only for an explicit loopback bind.
// Network-accessible deployments must supply independent, non-default secrets.
func (c Config) Validate() error {
	host, _, err := net.SplitHostPort(c.Addr)
	if err != nil {
		return fmt.Errorf("invalid ONEMORE_ADDR: %w", err)
	}
	ip := net.ParseIP(host)
	if ip != nil && ip.IsLoopback() {
		return nil
	}
	if len(strings.TrimSpace(c.AdminToken)) < 32 || c.AdminToken == "dev-admin-token" || c.AdminToken == "change-me" {
		return fmt.Errorf("non-loopback ONEMORE_ADDR requires a non-default ONEMORE_ADMIN_TOKEN of at least 32 characters")
	}
	if len(strings.TrimSpace(c.MasterKey)) < 32 || c.MasterKey == "onemore-development-master-key" || c.MasterKey == "change-me-too" {
		return fmt.Errorf("non-loopback ONEMORE_ADDR requires a non-default ONEMORE_MASTER_KEY of at least 32 characters")
	}
	if c.AdminToken == c.MasterKey {
		return fmt.Errorf("admin token and encryption master key must be different")
	}
	return nil
}
