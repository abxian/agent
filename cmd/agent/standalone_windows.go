//go:build windows

package main

import (
	"os"
	"strings"
	"syscall"

	"golang.org/x/sys/windows"
)

func relaunchStandaloneAdminIfNeeded() (bool, error) {
	if standaloneMode != "admin" || windows.GetCurrentProcessToken().IsElevated() {
		return false, nil
	}

	verb, err := windows.UTF16PtrFromString("runas")
	if err != nil {
		return false, err
	}
	executable, err := os.Executable()
	if err != nil {
		return false, err
	}
	file, err := windows.UTF16PtrFromString(executable)
	if err != nil {
		return false, err
	}
	quoted := make([]string, 0, len(os.Args)-1)
	for _, arg := range os.Args[1:] {
		quoted = append(quoted, syscall.EscapeArg(arg))
	}
	parameters, err := windows.UTF16PtrFromString(strings.Join(quoted, " "))
	if err != nil {
		return false, err
	}
	directory, err := windows.UTF16PtrFromString("")
	if err != nil {
		return false, err
	}
	if err := windows.ShellExecute(0, verb, file, parameters, directory, windows.SW_SHOWNORMAL); err != nil {
		return false, err
	}
	return true, nil
}
