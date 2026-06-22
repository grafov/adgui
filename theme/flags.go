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

package theme

import (
	"embed"
	"path"
	"strings"
	"sync"

	"fyne.io/fyne/v2"
)

//go:embed flags/*
var flagFS embed.FS

var (
	flagResources     map[string]fyne.Resource
	flagResourcesOnce sync.Once
)

func initFlagResources() {
	flagResources = make(map[string]fyne.Resource)
	entries, err := flagFS.ReadDir("flags")
	if err != nil {
		return
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".svg") {
			continue
		}
		iso := strings.TrimSuffix(entry.Name(), ".svg")
		data, err := flagFS.ReadFile(path.Join("flags", entry.Name()))
		if err != nil {
			continue
		}
		flagResources[iso] = fyne.NewStaticResource("flag-"+iso, data)
	}
}

// FlagResource returns the embedded SVG flag for the given ISO 3166-1 alpha-2 code.
// Returns nil when the code is unknown or empty.
func FlagResource(iso string) fyne.Resource {
	flagResourcesOnce.Do(initFlagResources)
	iso = strings.ToLower(strings.TrimSpace(iso))
	if iso == "" {
		return nil
	}
	return flagResources[iso]
}
