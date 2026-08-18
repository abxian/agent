//go:build !windows

package main

func relaunchStandaloneAdminIfNeeded() (bool, error) {
	return false, nil
}
