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

func TestServiceKeysMissingFile(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	keys, err := ServiceKeys()
	if err != nil {
		t.Fatal(err)
	}
	if keys.IPRegistryKey != "" {
		t.Fatalf("expected empty keys, got %+v", keys)
	}
}

func TestServiceKeysOverride(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	dir := filepath.Join(home, ".config", configDirName)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	content := "IPREGISTRY_KEY=custom-key\nGEOAPIFY_KEY=geo-key\n"
	if err := os.WriteFile(filepath.Join(dir, serviceKeysFileName), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	keys, err := ServiceKeys()
	if err != nil {
		t.Fatal(err)
	}
	if keys.IPRegistryKey != "custom-key" {
		t.Fatalf("IPREGISTRY_KEY: got %q", keys.IPRegistryKey)
	}
	if keys.GeoapifyKey != "geo-key" {
		t.Fatalf("GEOAPIFY_KEY: got %q", keys.GeoapifyKey)
	}
}
