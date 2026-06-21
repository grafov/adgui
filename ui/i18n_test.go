package ui_test

import (
	"encoding/json"
	"io/fs"
	"testing"

	"adgui/ui"
)

func TestTranslationCatalogs(t *testing.T) {
	t.Parallel()

	catalogs, err := ui.TranslationCatalogs()
	if err != nil {
		t.Fatalf("read translation catalogs: %v", err)
	}

	if len(catalogs) != 3 {
		t.Fatalf("expected 3 catalogs, got %d", len(catalogs))
	}

	enKeys := appTranslationKeys(collectTranslationKeys(t, catalogs["en"]))
	for locale, raw := range catalogs {
		if locale == "en" {
			continue
		}

		keys := collectTranslationKeys(t, raw)
		if locale == "eo" {
			requireFyneBaseKeys(t, keys)
		}
		keys = appTranslationKeys(keys)
		if diff := symmetricKeyDiff(enKeys, keys); len(diff) > 0 {
			t.Fatalf("locale %s key mismatch with en: %v", locale, diff)
		}
	}
}

func collectTranslationKeys(t *testing.T, raw []byte) map[string]struct{} {
	t.Helper()

	var data map[string]json.RawMessage
	if err := json.Unmarshal(raw, &data); err != nil {
		t.Fatalf("invalid json: %v", err)
	}

	keys := make(map[string]struct{}, len(data))
	for key, value := range data {
		keys[key] = struct{}{}
		validateTranslationValue(t, key, value)
	}

	return keys
}

func validateTranslationValue(t *testing.T, key string, value json.RawMessage) {
	t.Helper()

	var plain string
	if err := json.Unmarshal(value, &plain); err == nil {
		if plain == "" {
			t.Fatalf("key %q has empty translation", key)
		}
		return
	}

	var plural map[string]string
	if err := json.Unmarshal(value, &plural); err != nil {
		t.Fatalf("key %q has unsupported translation type: %v", key, err)
	}
	if len(plural) == 0 {
		t.Fatalf("key %q has empty plural translation", key)
	}
	if _, ok := plural["other"]; !ok {
		t.Fatalf("key %q plural translation missing other form", key)
	}
}

func symmetricKeyDiff(a, b map[string]struct{}) []string {
	var diff []string
	for key := range a {
		if _, ok := b[key]; !ok {
			diff = append(diff, "missing in second: "+key)
		}
	}
	for key := range b {
		if _, ok := a[key]; !ok {
			diff = append(diff, "missing in first: "+key)
		}
	}
	return diff
}

func appTranslationKeys(keys map[string]struct{}) map[string]struct{} {
	filtered := make(map[string]struct{}, len(keys))
	for key := range keys {
		if _, ok := fyneBaseKeys[key]; ok {
			continue
		}
		filtered[key] = struct{}{}
	}
	return filtered
}

func requireFyneBaseKeys(t *testing.T, keys map[string]struct{}) {
	t.Helper()

	for key := range fyneBaseKeys {
		if _, ok := keys[key]; !ok {
			t.Fatalf("eo catalog missing Fyne base key %q", key)
		}
	}
}

var fyneBaseKeys = map[string]struct{}{
	"Advanced":          {},
	"Cancel":            {},
	"Confirm":           {},
	"Copy":              {},
	"Create Folder":     {},
	"Cut":               {},
	"Enter filename":    {},
	"Error":             {},
	"Favourites":        {},
	"File":              {},
	"Folder":            {},
	"New Folder":        {},
	"No":                {},
	"OK":                {},
	"Open":              {},
	"Paste":             {},
	"Quit":              {},
	"Redo":              {},
	"Save":              {},
	"Select all":        {},
	"Show Hidden Files": {},
	"Undo":              {},
	"Yes":               {},
	"file.name":         {},
	"file.parent":       {},
}

func TestTranslationFilesEmbedded(t *testing.T) {
	t.Parallel()

	if err := fs.WalkDir(ui.TranslationsFS, "translation", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if d.Name() == "en.json" || d.Name() == "ru.json" || d.Name() == "eo.json" {
			return nil
		}
		t.Errorf("unexpected translation file: %s", path)
		return nil
	}); err != nil {
		t.Fatalf("walk translations: %v", err)
	}
}
