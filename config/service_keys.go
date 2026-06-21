package config

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"adgui/ipregion"

	"gopkg.in/ini.v1"
)

const serviceKeysFileName = "service-keys"

// ServiceKeys reads ~/.config/adgui/service-keys (INI) and returns API keys for
// ipregion service probes. Missing file or keys are not errors: callers receive
// empty strings and ipregion falls back to upstream demo defaults.
func ServiceKeys() (ipregion.ServiceKeys, error) {
	path, err := serviceKeysPath()
	if err != nil {
		return ipregion.ServiceKeys{}, err
	}

	cfg, err := ini.Load(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return ipregion.ServiceKeys{}, nil
		}
		return ipregion.ServiceKeys{}, err
	}

	sec := cfg.Section("")
	return ipregion.ServiceKeys{
		IPRegistryKey:    strings.TrimSpace(sec.Key("IPREGISTRY_KEY").String()),
		GeoapifyKey:      strings.TrimSpace(sec.Key("GEOAPIFY_KEY").String()),
		SpotifyClientID:  strings.TrimSpace(sec.Key("SPOTIFY_CLIENT_ID").String()),
		SpotifyAPIKey:    strings.TrimSpace(sec.Key("SPOTIFY_API_KEY").String()),
		AirportCodesAuth: strings.TrimSpace(sec.Key("AIRPORT_CODES_AUTH").String()),
	}, nil
}

func serviceKeysPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDir, ".config", configDirName, serviceKeysFileName), nil
}
