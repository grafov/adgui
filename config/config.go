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
	defaultAdguardCmd     = "adguardvpn-cli"
	keyAdguardCmd         = "ADGUARD_CMD"
	keyAdguardKillCmd     = "ADGUARD_KILL_CMD"
	keyAdguardSudoWrap    = "ADGUARD_SUDO_WRAP"
	keyAdguardSudoAskpass = "ADGUARD_SUDO_ASKPASS"
)

// EnsureAdguirc creates ~/.config/adgui/adguirc when it is missing.
// The file contains a localized header comment and all known keys with default
// values, each preceded by a localized description and commented out with #.
// keyComments maps ADGUARD_* keys to one-line descriptions already localized by the caller.
// An existing file is never modified.
func EnsureAdguirc(headerComment string, keyComments map[string]string) error {
	path, err := configPath()
	if err != nil {
		return err
	}

	if _, err := os.Stat(path); err == nil {
		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	return os.WriteFile(path, []byte(buildAdguircTemplate(headerComment, keyComments)), 0o644)
}

func buildAdguircTemplate(headerComment string, keyComments map[string]string) string {
	var b strings.Builder

	for _, line := range strings.Split(headerComment, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			b.WriteByte('\n')
			continue
		}
		if !strings.HasPrefix(line, "#") {
			line = "# " + line
		}
		b.WriteString(line)
		b.WriteByte('\n')
	}
	b.WriteByte('\n')

	defaults := []struct {
		key   string
		value string
	}{
		{keyAdguardCmd, defaultAdguardCmd},
		{keyAdguardKillCmd, ""},
		{keyAdguardSudoWrap, "true"},
		{keyAdguardSudoAskpass, "true"},
	}
	for _, item := range defaults {
		if comment := strings.TrimSpace(keyComments[item.key]); comment != "" {
			if !strings.HasPrefix(comment, "#") {
				comment = "# " + comment
			}
			b.WriteString(comment)
			b.WriteByte('\n')
		}
		b.WriteString("# " + item.key + "=" + item.value + "\n")
		b.WriteByte('\n')
	}

	return b.String()
}

// AdguardCmd resolves ADGUARD_CMD from adguirc, then environment, then code default.
func AdguardCmd() (string, error) {
	return stringConfig(keyAdguardCmd, defaultAdguardCmd)
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
	fileValue, err := stringValueFromFile(key)
	if err != nil {
		return true, err
	}
	if fileValue != "" {
		return parseBoolDefaultTrue(fileValue), nil
	}

	if env := strings.TrimSpace(os.Getenv(key)); env != "" {
		return parseBoolDefaultTrue(env), nil
	}

	return true, nil
}

func parseBoolDefaultTrue(value string) bool {
	switch strings.ToLower(value) {
	case "0", "false", "no", "off":
		return false
	default:
		return true
	}
}

// AdguardKillCmd resolves ADGUARD_KILL_CMD from adguirc, then environment.
// An empty result means the caller should use the standard process kill path.
func AdguardKillCmd() (string, error) {
	return stringConfig(keyAdguardKillCmd, "")
}

func stringConfig(key, defaultValue string) (string, error) {
	fileValue, err := stringValueFromFile(key)
	if err != nil {
		return defaultValue, err
	}
	if fileValue != "" {
		return fileValue, nil
	}

	if env := strings.TrimSpace(os.Getenv(key)); env != "" {
		return env, nil
	}

	return defaultValue, nil
}

func stringValueFromFile(key string) (string, error) {
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

	return strings.TrimSpace(cfg.Section("").Key(key).String()), nil
}

func configPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDir, ".config", configDirName, configFileName), nil
}
