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

import "strings"

const (
	dnsNotConfigured = "System DNS could not be configured"
	dnsSavedPrefix   = "DNS configuration is saved to "
)

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

// ParseDNSWarning reports whether CLI output says the system resolver was left
// unchanged. needed is true when queries may bypass the tunnel. path is the
// resolv.conf the CLI wrote, taken from the "DNS configuration is saved to"
// sentence; it is empty when that sentence is absent and the caller should use
// the default adguardvpn-cli path.
func ParseDNSWarning(output string) (needed bool, path string) {
	output = ansiStripRegex.ReplaceAllString(output, "")
	if !strings.Contains(output, dnsNotConfigured) {
		return false, ""
	}
	for line := range strings.SplitSeq(output, "\n") {
		idx := strings.Index(line, dnsSavedPrefix)
		if idx < 0 {
			continue
		}
		return true, dnsPathFromLine(line[idx+len(dnsSavedPrefix):])
	}
	return true, ""
}

func dnsPathFromLine(rest string) string {
	rest = strings.TrimSpace(rest)
	if rest == "" {
		return ""
	}
	if i := strings.Index(rest, ". "); i >= 0 {
		return rest[:i]
	}
	field := strings.Fields(rest)[0]
	return strings.TrimRight(field, ".")
}
