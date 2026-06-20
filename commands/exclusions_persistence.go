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
// files are stored (~/.local/share/adgui/site-exclusions).
// Detailed explanation: It reads the user home directory using os.UserHomeDir and joins
// it with the standard path .local/share/adgui/site-exclusions.
func GetExclusionsDirPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %w", err)
	}
	return filepath.Join(home, ".local", "share", "adgui", "site-exclusions"), nil
}

// GetExclusionsFilePath returns the absolute path to the exclusions file for the given mode.
// Detailed explanation: It resolves the base directory path first, then appends the file name
// matching the mode ("general.txt" or "selective.txt").
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

// LoadExclusionsForMode reads the saved exclusions for the specified mode from the local filesystem.
// Detailed explanation: It reads the file line by line, trims each domain, ignores empty lines,
// and returns a normalized slice of domains. If the file does not exist, it returns an empty slice and no error.
func LoadExclusionsForMode(mode SiteExclusionMode) ([]string, error) {
	path, err := GetExclusionsFilePath(mode)
	if err != nil {
		return nil, err
	}

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

// SaveExclusionsForMode writes the list of domains for the specified mode to the local filesystem.
// Detailed explanation: It normalizes the domains first, ensures that the directory exists,
// and then overwrites the mode-specific file with the domain list, one domain per line.
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
// Detailed explanation: It processes the input slice of domains, trims whitespace, skips empty strings,
// and filters duplicates case-insensitively using a map of lowercase domains.
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
