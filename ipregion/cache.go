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

package ipregion

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"adgui/locations"
)

const (
	cacheFilePrefix = "region-ip."
	vpnOffCacheKey  = cacheFilePrefix + "vpn-off"
)

// GetCacheDir returns the XDG cache directory for adgui region-ip files.
// Uses $XDG_CACHE_HOME/adgui when set, otherwise ~/.cache/adgui.
func GetCacheDir() (string, error) {
	if xdg := os.Getenv("XDG_CACHE_HOME"); xdg != "" {
		return filepath.Join(xdg, "adgui"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %w", err)
	}
	return filepath.Join(home, ".cache", "adgui"), nil
}

// CacheKeyForState returns the cache filename for the current VPN connection state.
func CacheKeyForState(loc locations.Location, connected bool) string {
	if !connected {
		return vpnOffCacheKey
	}
	iso := strings.ToLower(strings.TrimSpace(loc.ISO))
	if iso == "" {
		iso = "unknown"
	}
	city := sanitizeLocationName(loc.City)
	if city == "" {
		city = "unknown"
	}
	return cacheFilePrefix + iso + "." + city
}

// sanitizeLocationName converts a city name into a safe cache filename segment.
func sanitizeLocationName(name string) string {
	name = strings.TrimSpace(name)
	name = strings.ToLower(name)
	if name == "" {
		return ""
	}

	var b strings.Builder
	prevDash := false
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			prevDash = false
		case r == ' ', r == '-', r == '_':
			if !prevDash && b.Len() > 0 {
				b.WriteByte('-')
				prevDash = true
			}
		}
	}
	return strings.Trim(b.String(), "-")
}

func cacheFilePath(key string) (string, error) {
	dir, err := GetCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, key), nil
}

// LoadCachedReport reads a cached report by cache key. Returns nil, nil when missing.
func LoadCachedReport(key string) (*CachedReport, error) {
	path, err := cacheFilePath(key)
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read region-ip cache %s: %w", key, err)
	}

	var entry CachedReport
	if err := json.Unmarshal(data, &entry); err != nil {
		return nil, fmt.Errorf("failed to decode region-ip cache %s: %w", key, err)
	}
	return &entry, nil
}

// LoadCacheForState loads the cache entry for the current VPN connection state.
func LoadCacheForState(loc locations.Location, connected bool) (*CachedReport, error) {
	return LoadCachedReport(CacheKeyForState(loc, connected))
}

// SaveCachedReport writes a cache entry to disk under the given key.
func SaveCachedReport(key string, entry *CachedReport) error {
	if entry == nil {
		return fmt.Errorf("cache entry is nil")
	}

	path, err := cacheFilePath(key)
	if err != nil {
		return err
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("failed to encode region-ip cache: %w", err)
	}

	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("failed to write region-ip cache %s: %w", key, err)
	}
	return nil
}

// SaveCacheForState persists a report for the current VPN connection state.
func SaveCacheForState(loc locations.Location, connected bool, report *Report, checkedAt time.Time) error {
	if report == nil {
		return fmt.Errorf("report is nil")
	}

	entry := &CachedReport{
		CheckedAt: checkedAt,
		VPNOff:    !connected,
		Report:    *report,
	}
	if connected {
		entry.ISO = loc.ISO
		entry.Location = loc.City
	}
	return SaveCachedReport(CacheKeyForState(loc, connected), entry)
}

// ClearCache removes all region-ip.* cache files from the cache directory.
func ClearCache() error {
	dir, err := GetCacheDir()
	if err != nil {
		return err
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("failed to read cache directory: %w", err)
	}

	var firstErr error
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasPrefix(name, cacheFilePrefix) {
			continue
		}
		if err := os.Remove(filepath.Join(dir, name)); err != nil && !errors.Is(err, os.ErrNotExist) {
			if firstErr == nil {
				firstErr = fmt.Errorf("failed to remove cache file %s: %w", name, err)
			}
		}
	}
	return firstErr
}
