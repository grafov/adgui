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
	"sort"
	"strings"
)

func isCountableValue(v string) bool {
	v = strings.TrimSpace(v)
	if v == "" || v == NotAvailable {
		return false
	}
	switch strings.ToLower(v) {
	case "yes", "no", "null", "n/a", "—":
		return false
	}
	return true
}

// IsRegionCountryCode reports whether v is a country ISO code suitable for comparison and highlighting.
func IsRegionCountryCode(v string) bool {
	return isCountableValue(v)
}

// BuildSummary aggregates country codes from report results into percentages.
func BuildSummary(report *Report) Summary {
	var s Summary
	if report == nil {
		return s
	}

	ipv4Counts := make(map[string]int)
	ipv6Counts := make(map[string]int)

	for _, r := range report.Results {
		if isCountableValue(r.IPv4) {
			ipv4Counts[strings.ToUpper(r.IPv4)]++
			s.IPv4Total++
		}
		if isCountableValue(r.IPv6) {
			ipv6Counts[strings.ToUpper(r.IPv6)]++
			s.IPv6Total++
		}
	}

	codes := make(map[string]struct{})
	for c := range ipv4Counts {
		codes[c] = struct{}{}
	}
	for c := range ipv6Counts {
		codes[c] = struct{}{}
	}

	for code := range codes {
		stat := CountryStat{
			Code:      code,
			Name:      countryName(code),
			IPv4Count: ipv4Counts[code],
			IPv6Count: ipv6Counts[code],
		}
		if s.IPv4Total > 0 {
			stat.IPv4Pct = (stat.IPv4Count * 100) / s.IPv4Total
		}
		if s.IPv6Total > 0 {
			stat.IPv6Pct = (stat.IPv6Count * 100) / s.IPv6Total
		}
		s.Countries = append(s.Countries, stat)
	}

	sort.Slice(s.Countries, func(i, j int) bool {
		a, b := s.Countries[i], s.Countries[j]
		if a.IPv4Pct != b.IPv4Pct {
			return a.IPv4Pct > b.IPv4Pct
		}
		return a.IPv6Pct > b.IPv6Pct
	})

	return s
}

// TopConsensus returns the most common ISO code across IPv4 results, or empty.
func TopConsensus(report *Report) string {
	s := BuildSummary(report)
	if len(s.Countries) == 0 || s.IPv4Total == 0 {
		if len(s.Countries) > 0 && s.IPv6Total > 0 {
			return s.Countries[0].Code
		}
		return ""
	}
	return s.Countries[0].Code
}
