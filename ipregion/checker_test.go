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
	"testing"
)

func TestBuildSummary(t *testing.T) {
	report := &Report{
		Results: []ServiceResult{
			{Service: "a", IPv4: "DE", IPv6: "DE"},
			{Service: "b", IPv4: "DE", IPv6: "US"},
			{Service: "c", IPv4: "US", IPv6: NotAvailable},
			{Service: "d", IPv4: "Yes", IPv6: "No"},
		},
	}
	s := BuildSummary(report)
	if s.IPv4Total != 3 {
		t.Fatalf("ipv4 total: got %d want 3", s.IPv4Total)
	}
	if s.IPv6Total != 2 {
		t.Fatalf("ipv6 total: got %d want 2", s.IPv6Total)
	}
	if len(s.Countries) < 2 {
		t.Fatalf("expected at least 2 countries, got %d", len(s.Countries))
	}
	if s.Countries[0].Code != "DE" {
		t.Fatalf("top country: got %s want DE", s.Countries[0].Code)
	}
	if s.Countries[0].IPv4Pct != 66 {
		t.Fatalf("DE ipv4 pct: got %d want 66", s.Countries[0].IPv4Pct)
	}
}

func TestJSONPath(t *testing.T) {
	data := []byte(`{"country":{"iso_code":"DE"},"data":{"country":"US"},"list":[{"country":"FR"}]}`)
	if got := jsonPath(data, ".country.iso_code"); got != "DE" {
		t.Fatalf("iso_code: got %q", got)
	}
	if got := jsonPath(data, ".data.country"); got != "US" {
		t.Fatalf("data.country: got %q", got)
	}
	if got := jsonPath(data, ".list.0.country"); got != "FR" {
		t.Fatalf("array path: got %q", got)
	}
}

func TestLinkYouTubeFromGoogle(t *testing.T) {
	report := &Report{
		Results: []ServiceResult{
			{Service: "Google", IPv4: "DE", IPv6: "FR"},
			{Service: "YouTube", IPv4: NotAvailable, IPv6: NotAvailable},
		},
	}
	linkYouTubeFromGoogle(report)
	if report.Results[1].IPv4 != "DE" || report.Results[1].IPv6 != "FR" {
		t.Fatalf("youtube fallback failed: %+v", report.Results[1])
	}
}

func TestNormalizeOptionsDefaults(t *testing.T) {
	opts := normalizeOptions(Options{})
	if opts.Timeout != defaultTimeout {
		t.Fatalf("timeout default")
	}
	if opts.MaxConcurrency != defaultMaxConcurrency {
		t.Fatalf("concurrency default")
	}
	if len(opts.Groups) != 2 {
		t.Fatalf("groups default")
	}
}
