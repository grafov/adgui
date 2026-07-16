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

// LoadTranslations registers embedded UI translation catalogs with Fyne.
func LoadTranslations() error {
	return lang.AddTranslationsFS(translationsFS, "translation")
}

func loadTranslations() error {
	return LoadTranslations()
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
