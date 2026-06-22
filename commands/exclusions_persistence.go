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
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// GetExclusionsDirPath returns the absolute path to the directory where site exclusions
// files are stored (~/.config/adgui/site-exclusions).
func GetExclusionsDirPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %w", err)
	}
	return filepath.Join(home, ".config", "adgui", "site-exclusions"), nil
}

func getLegacyExclusionsDirPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %w", err)
	}
	return filepath.Join(home, ".local", "share", "adgui", "site-exclusions"), nil
}

// GetExclusionsFilePath returns the absolute path to the exclusions file for the given mode.
func GetExclusionsFilePath(mode SiteExclusionMode) (string, error) {
	dir, err := GetExclusionsDirPath()
	if err != nil {
		return "", err
	}
	filename := "general.txt"
	if mode == SiteExclusionModeSelective {
		filename = "selective.txt"
	}
	return filepath.Join(dir, filename), nil
}

func getLegacyExclusionsFilePath(mode SiteExclusionMode) (string, error) {
	dir, err := getLegacyExclusionsDirPath()
	if err != nil {
		return "", err
	}
	filename := "general.txt"
	if mode == SiteExclusionModeSelective {
		filename = "selective.txt"
	}
	return filepath.Join(dir, filename), nil
}

func readExclusionsFromFile(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to open exclusions file: %w", err)
	}
	defer func() {
		_ = file.Close()
	}()

	var domains []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			domains = append(domains, line)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read exclusions file: %w", err)
	}

	return NormalizeDomains(domains), nil
}

// LoadExclusionsForMode reads the saved exclusions for the specified mode from the local filesystem.
// If the new config file does not exist, it falls back to the legacy data directory and migrates
// the list to the new location without deleting the old file.
func LoadExclusionsForMode(mode SiteExclusionMode) ([]string, error) {
	path, err := GetExclusionsFilePath(mode)
	if err != nil {
		return nil, err
	}

	domains, err := readExclusionsFromFile(path)
	if err != nil {
		return nil, err
	}
	if domains != nil {
		return domains, nil
	}

	legacyPath, err := getLegacyExclusionsFilePath(mode)
	if err != nil {
		return nil, err
	}

	legacyDomains, err := readExclusionsFromFile(legacyPath)
	if err != nil {
		return nil, err
	}
	if legacyDomains == nil {
		return nil, nil
	}

	if err := SaveExclusionsForMode(mode, legacyDomains); err != nil {
		return nil, err
	}

	return legacyDomains, nil
}

// SaveExclusionsForMode writes the list of domains for the specified mode to the local filesystem.
func SaveExclusionsForMode(mode SiteExclusionMode, domains []string) error {
	path, err := GetExclusionsFilePath(mode)
	if err != nil {
		return err
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create exclusions directory: %w", err)
	}

	normalized := NormalizeDomains(domains)

	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create exclusions file: %w", err)
	}
	defer func() {
		_ = file.Close()
	}()

	writer := bufio.NewWriter(file)
	for _, domain := range normalized {
		if _, err := writer.WriteString(domain + "\n"); err != nil {
			return fmt.Errorf("failed to write domain %s to file: %w", domain, err)
		}
	}

	if err := writer.Flush(); err != nil {
		return fmt.Errorf("failed to flush exclusions file writer: %w", err)
	}

	return nil
}

// NormalizeDomains normalizes the domain list by trimming spaces, removing empty lines,
// and deduplicating them in a case-insensitive manner while preserving the case of the first occurrence.
func NormalizeDomains(domains []string) []string {
	seen := make(map[string]bool)
	var result []string
	for _, d := range domains {
		trimmed := strings.TrimSpace(d)
		if trimmed == "" {
			continue
		}
		lower := strings.ToLower(trimmed)
		if !seen[lower] {
			seen[lower] = true
			result = append(result, trimmed)
		}
	}
	return result
}
