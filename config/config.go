// Copyright (C) 2026 Alexander Grafov <grafov@inet.name>
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package config

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/ini.v1"
)

const (
	configDirName         = "adgui"
	configFileName        = "adguirc"
	keyAdguardCmd         = "ADGUARD_CMD"
	keyAdguardKillCmd     = "ADGUARD_KILL_CMD"
	keyAdguardSudoWrap    = "ADGUARD_SUDO_WRAP"
	keyAdguardSudoAskpass = "ADGUARD_SUDO_ASKPASS"
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

// AdguardSudoWrapEnabled reports whether adgui should inject the private sudo PATH wrapper.
// Default is true. Set ADGUARD_SUDO_WRAP=0/false/no in adguirc or environment to disable.
func AdguardSudoWrapEnabled() (bool, error) {
	return boolConfigDefaultTrue(keyAdguardSudoWrap)
}

// AdguardSudoAskpassEnabled reports whether adgui should collect a sudo password via GUI
// askpass for privileged CLI operations. Default is true.
// Set ADGUARD_SUDO_ASKPASS=0/false/no in adguirc or environment to disable (passwordless sudo).
func AdguardSudoAskpassEnabled() (bool, error) {
	return boolConfigDefaultTrue(keyAdguardSudoAskpass)
}

func boolConfigDefaultTrue(key string) (bool, error) {
	if env := strings.TrimSpace(os.Getenv(key)); env != "" {
		return parseBoolDefaultTrue(env), nil
	}

	configPath, err := configPath()
	if err != nil {
		return true, err
	}

	cfg, err := ini.Load(configPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return true, nil
		}
		return true, err
	}

	value := strings.TrimSpace(cfg.Section("").Key(key).String())
	if value == "" {
		return true, nil
	}
	return parseBoolDefaultTrue(value), nil
}

func parseBoolDefaultTrue(value string) bool {
	switch strings.ToLower(value) {
	case "0", "false", "no", "off":
		return false
	default:
		return true
	}
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
