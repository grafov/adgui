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

package commands

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"adgui/locations"
)

const (
	locationBookmarksDir  = "adgui"
	locationBookmarksFile = "bookmarks"
)

// LocationBookmark identifies a VPN location saved by the user.
type LocationBookmark struct {
	ISO     string `json:"iso"`
	Country string `json:"country"`
	City    string `json:"city"`
}

// LocationBookmarkKey returns a stable identifier for a location bookmark.
func LocationBookmarkKey(iso, country, city string) string {
	return strings.ToLower(strings.TrimSpace(iso)) + "|" +
		strings.ToLower(strings.TrimSpace(country)) + "|" +
		strings.ToLower(strings.TrimSpace(city))
}

// GetLocationBookmarksPath returns the absolute path to the location bookmarks file.
func GetLocationBookmarksPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %w", err)
	}
	return filepath.Join(home, ".config", locationBookmarksDir, locationBookmarksFile), nil
}

// LoadLocationBookmarks reads saved location bookmarks from disk.
// Returns an empty slice when the file does not exist.
func LoadLocationBookmarks() ([]LocationBookmark, error) {
	path, err := GetLocationBookmarksPath()
	if err != nil {
		return nil, err
	}

	file, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to open location bookmarks: %w", err)
	}
	defer func() {
		_ = file.Close()
	}()

	var bookmarks []LocationBookmark
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var bookmark LocationBookmark
		if err := json.Unmarshal([]byte(line), &bookmark); err != nil {
			continue
		}
		if bookmark.ISO == "" || bookmark.Country == "" || bookmark.City == "" {
			continue
		}
		bookmarks = append(bookmarks, bookmark)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read location bookmarks: %w", err)
	}

	return dedupeLocationBookmarks(bookmarks), nil
}

// SaveLocationBookmarks writes location bookmarks to disk as JSON Lines.
func SaveLocationBookmarks(bookmarks []LocationBookmark) error {
	path, err := GetLocationBookmarksPath()
	if err != nil {
		return err
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	bookmarks = dedupeLocationBookmarks(bookmarks)

	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create location bookmarks file: %w", err)
	}
	defer func() {
		_ = file.Close()
	}()

	writer := bufio.NewWriter(file)
	for _, bookmark := range bookmarks {
		data, err := json.Marshal(bookmark)
		if err != nil {
			return fmt.Errorf("failed to encode bookmark entry: %w", err)
		}
		if _, err := writer.Write(data); err != nil {
			return fmt.Errorf("failed to write bookmark entry: %w", err)
		}
		if err := writer.WriteByte('\n'); err != nil {
			return fmt.Errorf("failed to write bookmark newline: %w", err)
		}
	}
	if err := writer.Flush(); err != nil {
		return fmt.Errorf("failed to flush location bookmarks: %w", err)
	}
	return nil
}

// LocationBookmarkSet converts bookmark entries into a lookup set keyed by LocationBookmarkKey.
func LocationBookmarkSet(bookmarks []LocationBookmark) map[string]struct{} {
	set := make(map[string]struct{}, len(bookmarks))
	for _, bookmark := range bookmarks {
		key := LocationBookmarkKey(bookmark.ISO, bookmark.Country, bookmark.City)
		set[key] = struct{}{}
	}
	return set
}

// PruneAndSaveLocationBookmarks removes bookmarks that no longer exist in locs.
// When locs is empty, bookmarks are returned unchanged and the file is not modified.
// When stale bookmarks are removed, the file is rewritten via SaveLocationBookmarks.
func PruneAndSaveLocationBookmarks(bookmarks []LocationBookmark, locs []locations.Location) ([]LocationBookmark, error) {
	if len(locs) == 0 {
		return bookmarks, nil
	}

	available := make(map[string]struct{}, len(locs))
	for _, loc := range locs {
		key := LocationBookmarkKey(loc.ISO, loc.Country, loc.City)
		available[key] = struct{}{}
	}

	pruned := make([]LocationBookmark, 0, len(bookmarks))
	for _, bookmark := range bookmarks {
		key := LocationBookmarkKey(bookmark.ISO, bookmark.Country, bookmark.City)
		if _, ok := available[key]; ok {
			pruned = append(pruned, bookmark)
		}
	}
	pruned = dedupeLocationBookmarks(pruned)

	if locationBookmarkKeysEqual(bookmarks, pruned) {
		return pruned, nil
	}

	if err := SaveLocationBookmarks(pruned); err != nil {
		return pruned, err
	}
	return pruned, nil
}

func locationBookmarkKeysEqual(a, b []LocationBookmark) bool {
	if len(a) != len(b) {
		return false
	}
	setA := LocationBookmarkSet(a)
	for _, bookmark := range b {
		key := LocationBookmarkKey(bookmark.ISO, bookmark.Country, bookmark.City)
		if _, ok := setA[key]; !ok {
			return false
		}
	}
	return true
}

func dedupeLocationBookmarks(bookmarks []LocationBookmark) []LocationBookmark {
	seen := make(map[string]struct{}, len(bookmarks))
	result := make([]LocationBookmark, 0, len(bookmarks))
	for _, bookmark := range bookmarks {
		key := LocationBookmarkKey(bookmark.ISO, bookmark.Country, bookmark.City)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, bookmark)
	}
	return result
}
