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
