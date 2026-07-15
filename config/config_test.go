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
	"testing"
)

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

	dir := filepath.Join(home, ".config", configDirName)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, configFileName), []byte("ADGUARD_SUDO_WRAP=false\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	enabled, err := AdguardSudoWrapEnabled()
	if err != nil {
		t.Fatal(err)
	}
	if enabled {
		t.Fatal("expected sudo wrap disabled from config")
	}
}
