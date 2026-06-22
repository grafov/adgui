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
	"time"
)

const (
	maxHistoryEntries      = 12
	connectionsHistoryFile = "connections-history"
)

// ConnectionHistoryEntry records one VPN session with location and time range.
type ConnectionHistoryEntry struct {
	City      string     `json:"city"`
	Country   string     `json:"country"`
	Ping      int        `json:"ping"`
	StartedAt time.Time  `json:"started_at"`
	EndedAt   *time.Time `json:"ended_at,omitempty"`
}

// GetDataDir returns the XDG user data directory for adgui.
// Uses $XDG_DATA_HOME/adgui when set, otherwise ~/.local/share/adgui.
func GetDataDir() (string, error) {
	if xdg := os.Getenv("XDG_DATA_HOME"); xdg != "" {
		return filepath.Join(xdg, "adgui"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %w", err)
	}
	return filepath.Join(home, ".local", "share", "adgui"), nil
}

// GetConnectionsHistoryPath returns the absolute path to the connections history file.
func GetConnectionsHistoryPath() (string, error) {
	dir, err := GetDataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, connectionsHistoryFile), nil
}

// LoadConnectionHistory reads saved connection history from disk.
// Returns an empty slice when the file does not exist.
func LoadConnectionHistory() ([]ConnectionHistoryEntry, error) {
	path, err := GetConnectionsHistoryPath()
	if err != nil {
		return nil, err
	}

	file, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to open connections history: %w", err)
	}
	defer func() {
		_ = file.Close()
	}()

	var entries []ConnectionHistoryEntry
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		var entry ConnectionHistoryEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue
		}
		entries = append(entries, entry)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read connections history: %w", err)
	}

	if len(entries) > maxHistoryEntries {
		entries = entries[:maxHistoryEntries]
	}
	return entries, nil
}

// SaveConnectionHistory writes connection history to disk as JSON Lines.
func SaveConnectionHistory(entries []ConnectionHistoryEntry) error {
	path, err := GetConnectionsHistoryPath()
	if err != nil {
		return err
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("failed to create data directory: %w", err)
	}

	if len(entries) > maxHistoryEntries {
		entries = entries[:maxHistoryEntries]
	}

	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create connections history file: %w", err)
	}
	defer func() {
		_ = file.Close()
	}()

	writer := bufio.NewWriter(file)
	for _, entry := range entries {
		data, err := json.Marshal(entry)
		if err != nil {
			return fmt.Errorf("failed to encode history entry: %w", err)
		}
		if _, err := writer.Write(data); err != nil {
			return fmt.Errorf("failed to write history entry: %w", err)
		}
		if err := writer.WriteByte('\n'); err != nil {
			return fmt.Errorf("failed to write history newline: %w", err)
		}
	}
	if err := writer.Flush(); err != nil {
		return fmt.Errorf("failed to flush connections history: %w", err)
	}
	return nil
}
