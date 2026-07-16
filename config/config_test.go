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
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEnsureAdguircCreatesFile(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	header := "Line one\nLine two"
	comments := map[string]string{
		keyAdguardCmd:         "Path to adguardvpn-cli. Example: /usr/bin/adguardvpn-cli",
		keyAdguardKillCmd:     "Optional kill command prefix; PID is appended.",
		keyAdguardSudoWrap:    "Inject private sudo PATH wrapper. Values: true, false.",
		keyAdguardSudoAskpass: "Show GUI sudo password dialog. Values: true, false.",
	}
	if err := EnsureAdguirc(header, comments); err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(home, ".config", configDirName, configFileName)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	content := string(data)
	if !strings.Contains(content, "# Line one\n") {
		t.Fatalf("expected header line one, got:\n%s", content)
	}
	if !strings.Contains(content, "# Line two\n") {
		t.Fatalf("expected header line two, got:\n%s", content)
	}
	if !strings.Contains(content, "# "+comments[keyAdguardCmd]+"\n# ADGUARD_CMD=adguardvpn-cli\n") {
		t.Fatalf("expected ADGUARD_CMD comment and key, got:\n%s", content)
	}
	if !strings.Contains(content, "# "+comments[keyAdguardKillCmd]+"\n# ADGUARD_KILL_CMD=\n") {
		t.Fatalf("expected ADGUARD_KILL_CMD comment and key, got:\n%s", content)
	}
	if !strings.Contains(content, "# "+comments[keyAdguardSudoWrap]+"\n# ADGUARD_SUDO_WRAP=true\n") {
		t.Fatalf("expected ADGUARD_SUDO_WRAP comment and key, got:\n%s", content)
	}
	if !strings.Contains(content, "# "+comments[keyAdguardSudoAskpass]+"\n# ADGUARD_SUDO_ASKPASS=true\n") {
		t.Fatalf("expected ADGUARD_SUDO_ASKPASS comment and key, got:\n%s", content)
	}
}

func TestEnsureAdguircExistingFileUntouched(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	dir := filepath.Join(home, ".config", configDirName)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(dir, configFileName)
	existing := "ADGUARD_CMD=/custom/cli\n"
	if err := os.WriteFile(path, []byte(existing), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := EnsureAdguirc("should not appear", map[string]string{
		keyAdguardCmd: "should not appear",
	}); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != existing {
		t.Fatalf("expected existing file unchanged, got %q", string(data))
	}
}

func TestAdguardCmdDefault(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("ADGUARD_CMD", "")

	cmd, err := AdguardCmd()
	if err != nil {
		t.Fatal(err)
	}
	if cmd != "adguardvpn-cli" {
		t.Fatalf("expected default adguardvpn-cli, got %q", cmd)
	}
}

func TestAdguardCmdEnvFallback(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("ADGUARD_CMD", "/env/adguardvpn-cli")

	cmd, err := AdguardCmd()
	if err != nil {
		t.Fatal(err)
	}
	if cmd != "/env/adguardvpn-cli" {
		t.Fatalf("expected env value, got %q", cmd)
	}
}

func TestAdguardCmdEnvOverridesFile(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("ADGUARD_CMD", "/env/adguardvpn-cli")

	writeConfigFile(t, home, "ADGUARD_CMD=/file/adguardvpn-cli\n")

	cmd, err := AdguardCmd()
	if err != nil {
		t.Fatal(err)
	}
	if cmd != "/env/adguardvpn-cli" {
		t.Fatalf("expected env value, got %q", cmd)
	}
}

func TestAdguardKillCmdEnvOverridesFile(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("ADGUARD_KILL_CMD", "env-kill")

	writeConfigFile(t, home, "ADGUARD_KILL_CMD=file-kill\n")

	cmd, err := AdguardKillCmd()
	if err != nil {
		t.Fatal(err)
	}
	if cmd != "env-kill" {
		t.Fatalf("expected env value, got %q", cmd)
	}
}

func TestAdguardSudoWrapEnabledDefault(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("ADGUARD_SUDO_WRAP", "")

	enabled, err := AdguardSudoWrapEnabled()
	if err != nil {
		t.Fatal(err)
	}
	if !enabled {
		t.Fatal("expected sudo wrap enabled by default")
	}
}

func TestAdguardSudoWrapEnabledEnvOverride(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("ADGUARD_SUDO_WRAP", "0")

	enabled, err := AdguardSudoWrapEnabled()
	if err != nil {
		t.Fatal(err)
	}
	if enabled {
		t.Fatal("expected sudo wrap disabled from env")
	}
}

func TestAdguardSudoWrapEnabledConfigFile(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("ADGUARD_SUDO_WRAP", "")

	writeConfigFile(t, home, "ADGUARD_SUDO_WRAP=false\n")

	enabled, err := AdguardSudoWrapEnabled()
	if err != nil {
		t.Fatal(err)
	}
	if enabled {
		t.Fatal("expected sudo wrap disabled from config")
	}
}

func TestAdguardSudoWrapEnabledEnvOverridesFile(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("ADGUARD_SUDO_WRAP", "1")

	writeConfigFile(t, home, "ADGUARD_SUDO_WRAP=false\n")

	enabled, err := AdguardSudoWrapEnabled()
	if err != nil {
		t.Fatal(err)
	}
	if !enabled {
		t.Fatal("expected sudo wrap enabled from env over config")
	}
}

func TestAdguardSudoAskpassEnabledDefault(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("ADGUARD_SUDO_ASKPASS", "")

	enabled, err := AdguardSudoAskpassEnabled()
	if err != nil {
		t.Fatal(err)
	}
	if !enabled {
		t.Fatal("expected sudo askpass enabled by default")
	}
}

func TestAdguardSudoAskpassEnabledEnvOverride(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("ADGUARD_SUDO_ASKPASS", "0")

	enabled, err := AdguardSudoAskpassEnabled()
	if err != nil {
		t.Fatal(err)
	}
	if enabled {
		t.Fatal("expected sudo askpass disabled from env")
	}
}

func TestAdguardSudoAskpassEnabledConfigFile(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("ADGUARD_SUDO_ASKPASS", "")

	writeConfigFile(t, home, "ADGUARD_SUDO_ASKPASS=false\n")

	enabled, err := AdguardSudoAskpassEnabled()
	if err != nil {
		t.Fatal(err)
	}
	if enabled {
		t.Fatal("expected sudo askpass disabled from config")
	}
}

func writeConfigFile(t *testing.T, home, content string) {
	t.Helper()

	dir := filepath.Join(home, ".config", configDirName)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, configFileName), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
