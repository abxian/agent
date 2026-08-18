package main

import (
	"strings"

	"github.com/shenxianhq/agent/model"
)

// These values stay empty in the universal Agent. Private standalone release
// builds inject them with -ldflags, keeping credentials out of source control.
var (
	standaloneMode         string
	standaloneServer       string
	standaloneClientSecret string
	standaloneTLS          string
)

func standaloneDefaults() model.AgentConfig {
	return model.AgentConfig{
		Server:             standaloneServer,
		ClientSecret:       standaloneClientSecret,
		TLS:                standaloneTLS == "" || strings.EqualFold(standaloneTLS, "true"),
		DisableAutoUpdate:  true,
		DisableForceUpdate: true,
	}
}
