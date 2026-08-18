package main

import "os"

// These values stay empty in the universal Agent. Private standalone release
// builds inject them with -ldflags, keeping credentials out of source control.
var (
	standaloneMode         string
	standaloneServer       string
	standaloneClientSecret string
	standaloneTLS          string
)

func applyStandaloneBootstrap(configPath string) (func(), bool) {
	if standaloneMode == "" || standaloneServer == "" || standaloneClientSecret == "" {
		return func() {}, false
	}
	if _, err := os.Stat(configPath); err == nil {
		return func() {}, false
	}

	keys := []string{
		"SX_SERVER",
		"SX_CLIENT_SECRET",
		"SX_TLS",
		"SX_DISABLE_AUTO_UPDATE",
		"SX_DISABLE_FORCE_UPDATE",
	}
	previous := make(map[string]*string, len(keys))
	for _, key := range keys {
		if value, ok := os.LookupEnv(key); ok {
			copyValue := value
			previous[key] = &copyValue
		} else {
			previous[key] = nil
		}
	}

	_ = os.Setenv("SX_SERVER", standaloneServer)
	_ = os.Setenv("SX_CLIENT_SECRET", standaloneClientSecret)
	tlsValue := standaloneTLS
	if tlsValue == "" {
		tlsValue = "true"
	}
	_ = os.Setenv("SX_TLS", tlsValue)
	// A universal self-update would replace this edition and remove its
	// embedded bootstrap/UAC behavior. Standalone editions are upgraded by
	// downloading the matching standalone artifact from the synchronized
	// release set instead.
	_ = os.Setenv("SX_DISABLE_AUTO_UPDATE", "true")
	_ = os.Setenv("SX_DISABLE_FORCE_UPDATE", "true")

	return func() {
		for _, key := range keys {
			if value := previous[key]; value != nil {
				_ = os.Setenv(key, *value)
			} else {
				_ = os.Unsetenv(key)
			}
		}
	}, true
}
