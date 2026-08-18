//go:build windows

package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"

	"github.com/shenxianhq/agent/model"
)

const standaloneCleanupSourceEnv = "SX_STANDALONE_CLEANUP_SOURCE"

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

func standaloneInstallPath() (string, error) {
	var root, directory, name string
	switch standaloneMode {
	case "user":
		root = os.Getenv("LOCALAPPDATA")
		directory = "Programs"
		name = "agent-user.exe"
	case "admin":
		root = os.Getenv("ProgramFiles")
		name = "agent-admin.exe"
	default:
		return "", nil
	}
	if root == "" {
		return "", fmt.Errorf("required Windows application directory is unavailable for %s mode", standaloneMode)
	}
	return filepath.Join(root, directory, "AgentManager", name), nil
}

func prepareStandaloneExecutable() (bool, error) {
	if standaloneMode == "" {
		return false, nil
	}
	source, err := os.Executable()
	if err != nil {
		return false, err
	}
	target, err := standaloneInstallPath()
	if err != nil {
		return false, err
	}
	if strings.EqualFold(filepath.Clean(source), filepath.Clean(target)) {
		startStandaloneSourceCleanup(source)
		return false, nil
	}
	if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
		return false, fmt.Errorf("create standalone directory: %w", err)
	}
	temporary := fmt.Sprintf("%s.new-%d", target, os.Getpid())
	if err := copyStandaloneExecutable(source, temporary); err != nil {
		return false, err
	}
	if err := os.Remove(target); err != nil && !errors.Is(err, os.ErrNotExist) {
		return false, fmt.Errorf("replace previous standalone executable: %w", err)
	}
	if err := os.Rename(temporary, target); err != nil {
		return false, fmt.Errorf("activate standalone executable: %w", err)
	}
	proc, err := os.StartProcess(target, append([]string{target}, os.Args[1:]...), &os.ProcAttr{
		Dir:   filepath.Dir(target),
		Env:   append(os.Environ(), standaloneCleanupSourceEnv+"="+source),
		Files: []*os.File{os.Stdin, os.Stdout, os.Stderr},
	})
	if err != nil {
		return false, fmt.Errorf("start installed standalone executable: %w", err)
	}
	_ = proc.Release()
	return true, nil
}

func copyStandaloneExecutable(source, target string) error {
	in, err := os.Open(source)
	if err != nil {
		return fmt.Errorf("open standalone source: %w", err)
	}
	defer in.Close()
	out, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0755)
	if err != nil {
		return fmt.Errorf("create standalone target: %w", err)
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		return fmt.Errorf("copy standalone executable: %w", err)
	}
	if err := out.Sync(); err != nil {
		_ = out.Close()
		return fmt.Errorf("sync standalone executable: %w", err)
	}
	if err := out.Close(); err != nil {
		return fmt.Errorf("close standalone executable: %w", err)
	}
	return nil
}

func startStandaloneSourceCleanup(currentExecutable string) {
	source := os.Getenv(standaloneCleanupSourceEnv)
	_ = os.Unsetenv(standaloneCleanupSourceEnv)
	if source == "" || strings.EqualFold(filepath.Clean(source), filepath.Clean(currentExecutable)) {
		return
	}
	if !strings.EqualFold(filepath.Ext(source), ".exe") || !standaloneFilesMatch(source, currentExecutable) {
		return
	}
	go func() {
		for range 40 {
			if err := os.Remove(source); err == nil || errors.Is(err, os.ErrNotExist) {
				return
			}
			time.Sleep(250 * time.Millisecond)
		}
	}()
}

func standaloneFilesMatch(left, right string) bool {
	hashFile := func(path string) ([]byte, error) {
		file, err := os.Open(path)
		if err != nil {
			return nil, err
		}
		defer file.Close()
		hash := sha256.New()
		if _, err := io.Copy(hash, file); err != nil {
			return nil, err
		}
		return hash.Sum(nil), nil
	}
	leftHash, err := hashFile(left)
	if err != nil {
		return false
	}
	rightHash, err := hashFile(right)
	return err == nil && bytes.Equal(leftHash, rightHash)
}

func standaloneRegistryPath() string {
	return `Software\AgentManager\` + strings.Title(standaloneMode)
}

func readStandaloneConfig() error {
	if standaloneServer == "" || standaloneClientSecret == "" {
		return errors.New("standalone connection settings are incomplete")
	}
	var data []byte
	key, err := registry.OpenKey(registry.CURRENT_USER, standaloneRegistryPath(), registry.QUERY_VALUE)
	if err == nil {
		value, _, readErr := key.GetBinaryValue("Config")
		_ = key.Close()
		if readErr != nil && !errors.Is(readErr, registry.ErrNotExist) {
			return fmt.Errorf("read standalone registry config: %w", readErr)
		}
		data = value
	} else if !errors.Is(err, registry.ErrNotExist) {
		return fmt.Errorf("open standalone registry config: %w", err)
	}
	if len(data) == 0 {
		data, err = json.Marshal(standaloneDefaults())
		if err != nil {
			return err
		}
	}
	return agentConfig.ReadFromStore(data, saveStandaloneRegistryConfig)
}

func saveStandaloneRegistryConfig(config *model.AgentConfig) error {
	data, err := json.Marshal(config)
	if err != nil {
		return err
	}
	key, _, err := registry.CreateKey(registry.CURRENT_USER, standaloneRegistryPath(), registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("create standalone registry config: %w", err)
	}
	defer key.Close()
	if err := key.SetBinaryValue("Config", data); err != nil {
		return fmt.Errorf("write standalone registry config: %w", err)
	}
	return nil
}
