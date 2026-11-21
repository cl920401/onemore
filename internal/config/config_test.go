package config

import (
	"strings"
	"testing"
)

func TestValidateDeploymentBoundary(t *testing.T) {
	strong := Config{Addr: ":8080", AdminToken: strings.Repeat("a", 64), MasterKey: strings.Repeat("b", 64)}
	tests := []struct {
		name string
		cfg  Config
		ok   bool
	}{
		{"default-loopback", Config{Addr: "127.0.0.1:8080", AdminToken: "dev-admin-token"}, true},
		{"ipv6-loopback", Config{Addr: "[::1]:8080"}, true},
		{"wildcard-defaults", Config{Addr: ":8080", AdminToken: "dev-admin-token"}, false},
		{"all-ipv4-defaults", Config{Addr: "0.0.0.0:8080"}, false},
		{"hostname-defaults", Config{Addr: "localhost:8080"}, false},
		{"malformed-address", Config{Addr: "bad"}, false},
		{"strong-network", strong, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.cfg.Validate()
			if (err == nil) != tc.ok {
				t.Fatalf("valid=%v, want %v", err == nil, tc.ok)
			}
		})
	}
	for _, field := range []string{"token", "key", "same"} {
		c := strong
		switch field {
		case "token":
			c.AdminToken = "short"
		case "key":
			c.MasterKey = "onemore-development-master-key"
		case "same":
			c.MasterKey = c.AdminToken
		}
		if c.Validate() == nil {
			t.Errorf("accepted unsafe %s", field)
		}
	}
}

func TestDefaultBindIsLoopback(t *testing.T) {
	t.Setenv("ONEMORE_ADDR", "")
	if Load().Addr != "127.0.0.1:8080" {
		t.Fatal("default API bind must be loopback")
	}
}
