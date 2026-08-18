//go:build !windows

package main

func relaunchStandaloneAdminIfNeeded() (bool, error) {
	return false, nil
}

func prepareStandaloneExecutable() (bool, error) {
	return false, nil
}

func readStandaloneConfig() error {
	return agentConfig.Read(defaultConfigPath)
}
