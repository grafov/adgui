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
