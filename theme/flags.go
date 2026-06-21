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
