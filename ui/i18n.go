package ui

import (
	"embed"
	"strings"

	"fyne.io/fyne/v2/lang"
)

//go:embed translation/*.json
var translationsFS embed.FS

// TranslationsFS exposes embedded translation files for tests.
var TranslationsFS = translationsFS

func loadTranslations() error {
	return lang.AddTranslationsFS(translationsFS, "translation")
}

// TranslationCatalogs returns raw JSON catalogs keyed by locale tag.
func TranslationCatalogs() (map[string][]byte, error) {
	entries, err := translationsFS.ReadDir("translation")
	if err != nil {
		return nil, err
	}

	catalogs := make(map[string][]byte, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		data, readErr := translationsFS.ReadFile("translation/" + name)
		if readErr != nil {
			return nil, readErr
		}
		locale := strings.TrimSuffix(name, ".json")
		catalogs[locale] = data
	}
	return catalogs, nil
}

func exclusionModeGeneralLabel() string {
	return lang.X("domains.mode.general", "The domains in the list excluded")
}

func exclusionModeSelectiveLabel() string {
	return lang.X("domains.mode.selective", "Only domains in the list included")
}

func domainsMenuLabel(count int) string {
	if count > 0 {
		return lang.XN(
			"tray.menu.domains_count",
			"Domains ({{.Count}})",
			count,
			map[string]any{"Count": count},
		)
	}
	return lang.X("tray.menu.domains", "Domains")
}

func formatPing(ping int) string {
	if ping < 0 {
		return lang.X("connections.ping.na", "Ping: n/a")
	}
	return lang.X("connections.ping.ms", "Ping: {{.Ping}} ms", map[string]any{"Ping": ping})
}
