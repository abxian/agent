package main

import "testing"

func TestStandaloneDefaults(t *testing.T) {
	oldServer, oldSecret, oldTLS := standaloneServer, standaloneClientSecret, standaloneTLS
	t.Cleanup(func() { standaloneServer, standaloneClientSecret, standaloneTLS = oldServer, oldSecret, oldTLS })
	standaloneServer, standaloneClientSecret, standaloneTLS = "embedded.example:443", "embedded-secret", "true"
	config := standaloneDefaults()
	if config.Server != standaloneServer || config.ClientSecret != standaloneClientSecret || !config.TLS {
		t.Fatalf("unexpected standalone defaults: %#v", config)
	}
	if !config.DisableAutoUpdate || !config.DisableForceUpdate {
		t.Fatal("standalone edition must preserve itself from universal replacement")
	}
}
