package config

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/ini.v1"
)

const (
	configDirName     = "adgui"
	configFileName    = "adguirc"
	keyAdguardCmd     = "ADGUARD_CMD"
	keyAdguardKillCmd = "ADGUARD_KILL_CMD"
)

// AdguardCmd reads ~/.config/adgui/adguirc (INI) and returns the ADGUARD_CMD value.
// It returns an empty string when the file is missing or the key is not set.
// Any other read or parse error is returned to the caller.
func AdguardCmd() (string, error) {
	configPath, err := configPath()
	if err != nil {
		return "", err
	}

	cfg, err := ini.Load(configPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", nil
		}
		return "", err
	}

	value := strings.TrimSpace(cfg.Section("").Key(keyAdguardCmd).String())
	return value, nil
}

// AdguardKillCmd reads ~/.config/adgui/adguirc (INI) and returns the ADGUARD_KILL_CMD value.
// It returns an empty string when the file is missing or the key is not set.
// Any other read or parse error is returned to the caller.
func AdguardKillCmd() (string, error) {
	configPath, err := configPath()
	if err != nil {
		return "", err
	}

	cfg, err := ini.Load(configPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", nil
		}
		return "", err
	}

	value := strings.TrimSpace(cfg.Section("").Key(keyAdguardKillCmd).String())
	return value, nil
}

func configPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDir, ".config", configDirName, configFileName), nil
}
