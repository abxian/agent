package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestApplyStandaloneBootstrapCreatesDefaultsOnlyForNewConfig(t *testing.T) {
	t.Setenv("SX_SERVER", "caller.example:1")
	_ = os.Unsetenv("SX_CLIENT_SECRET")
	_ = os.Unsetenv("SX_TLS")
	oldMode, oldServer, oldSecret, oldTLS := standaloneMode, standaloneServer, standaloneClientSecret, standaloneTLS
	t.Cleanup(func() {
		standaloneMode, standaloneServer, standaloneClientSecret, standaloneTLS = oldMode, oldServer, oldSecret, oldTLS
	})
	standaloneMode, standaloneServer, standaloneClientSecret, standaloneTLS = "user", "embedded.example:443", "embedded-secret", "true"

	restore, applied := applyStandaloneBootstrap(filepath.Join(t.TempDir(), "config.yml"))
	if !applied {
		t.Fatal("expected standalone defaults for a missing config")
	}
	if got := os.Getenv("SX_SERVER"); got != "embedded.example:443" {
		t.Fatalf("server = %q", got)
	}
	if got := os.Getenv("SX_CLIENT_SECRET"); got != "embedded-secret" {
		t.Fatalf("secret = %q", got)
	}
	restore()
	if got := os.Getenv("SX_SERVER"); got != "caller.example:1" {
		t.Fatalf("restored server = %q", got)
	}
	if _, ok := os.LookupEnv("SX_CLIENT_SECRET"); ok {
		t.Fatal("secret environment variable was not removed")
	}
}

func TestApplyStandaloneBootstrapPreservesExistingConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yml")
	if err := os.WriteFile(path, []byte("existing"), 0600); err != nil {
		t.Fatal(err)
	}
	oldMode, oldServer, oldSecret := standaloneMode, standaloneServer, standaloneClientSecret
	t.Cleanup(func() { standaloneMode, standaloneServer, standaloneClientSecret = oldMode, oldServer, oldSecret })
	standaloneMode, standaloneServer, standaloneClientSecret = "admin", "embedded.example:443", "embedded-secret"

	_, applied := applyStandaloneBootstrap(path)
	if applied {
		t.Fatal("must not override an existing standalone config")
	}
}
