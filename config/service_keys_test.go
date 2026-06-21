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
