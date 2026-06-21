package commands

import "strings"

// ParseLocationFromStatus extracts the connected city/location name from CLI status output.
func ParseLocationFromStatus(output string) string {
	for line := range strings.SplitSeq(output, "\n") {
		if !strings.Contains(line, "Connected to") {
			continue
		}
		location := line
		prefix := "Connected to "
		if idx := strings.Index(location, prefix); idx >= 0 {
			location = location[idx+len(prefix):]
		}
		location = strings.ReplaceAll(location, "\x1b[1m", "")
		location = strings.ReplaceAll(location, "\x1b[0m", "")
		if idx := strings.Index(location, " in "); idx >= 0 {
			location = location[:idx]
		}
		return strings.TrimSpace(location)
	}
	return ""
}
