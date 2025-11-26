package locations

import (
	"strings"
	"testing"
)

// TestParseLocationsFromSample tests parsing with the provided list-sample file content
func TestParseLocationsFromSample(t *testing.T) {
	// Sample content from the list-sample file
	sampleContent := `ISO   COUNTRY              CITY                           PING ESTIMATE
LV    Latvia               Riga                           29        
DE    Germany              Frankfurt                      37        
DK    Denmark              Copenhagen                     42        
NL    Netherlands          Amsterdam                      45        
IT    Italy                Milan                          46        
FR    France               Paris                          46        
CH    Switzerland          Zurich                         47        
CZ    Czechia              Prague                         47        
BE    Belgium              Brussels                       49        
FI    Finland              Helsinki                       50        
GB    United Kingdom       London                         52        
AT    Austria              Vienna                         52        
DE    Germany              Berlin                         53        
LU    Luxembourg           Luxembourg                     53        
PL    Poland               Warsaw                         53        
HR    Croatia              Zagreb                         55        
SK    Slovakia             Bratislava                     55        
EE    Estonia              Tallinn                        57        
UA    Ukraine              Kyiv                           59        
IE    Ireland              Dublin                         61        
FR    France               Marseille                      62        
NO    Norway               Oslo                           63        
RS    Serbia               Belgrade                       63
ES    Spain                Madrid                         63        
GB    United Kingdom       Manchester                     67        
BG    Bulgaria             Sofia                          67        
SE    Sweden               Stockholm                      68        
IT    Italy                Rome                           69        
HU    Hungary              Budapest                       70        
PT    Portugal             Lisbon                         71        
RO    Romania              Bucharest                      74        
EG    Egypt                Cairo                          76        
ES    Spain                Barcelona                      80        
GR    Greece               Athens                         85        
IS    Iceland              Reykjavik                      89        
MD    Moldova              Chișinău                       89        
LT    Lithuania            Vilnius                        98        
TR    Turkey               Istanbul                       99        
IR    Iran                 Tehran (Virtual)               106       
IL    Israel               Tel Aviv                       106       
CY    Cyprus               Nicosia                        109       
RU    Russia               Moscow (Virtual)               114       
US    United States        New York                       121       
CA    Canada               Toronto                        128       
US    United States        Boston                         129       
CA    Canada               Montreal                       134       
US    United States        Chicago                        142       
US    United States        Atlanta                        143       
US    United States        Miami                          148       
US    United States        Dallas                         157       
AE    UAE                  Dubai                          163       
US    United States        Denver                         163       
US    United States        Seattle                        182       
IT    Italy                Palermo                        187       
US    United States        Los Angeles                    188       
US    United States        Las Vegas                      188       
US    United States        Phoenix                        189       
CA    Canada               Vancouver                      190       
MX    Mexico               Mexico City                    190       
US    United States        Silicon Valley                 192       
CO    Colombia             Bogota                         203       
SG    Singapore            Singapore                      208       
TH    Thailand             Bangkok                        210       
NG    Nigeria              Lagos                          222       
PE    Peru                 Lima                           226       
NP    Nepal                Kathmandu                      250       
ID    Indonesia            Jakarta                        251       
BR    Brazil               São Paulo                      252       
KZ    Kazakhstan           Astana                         256       
PH    Philippines          Manila                         262       
TW    Taiwan               Taipei                         263       
CL    Chile                Santiago                       265       
KH    Cambodia             Phnom Penh                     268       
VN    Vietnam              Hanoi                          273       
AR    Argentina            Buenos Aires                   280       
IN    India                Mumbai (Virtual)               284       
ZA    South Africa         Johannesburg                   285       
HK    Hong Kong            Hong Kong                      286       
CN    China                Shanghai (Virtual)             288       
JP    Japan                Tokyo                          304       
KR    South Korea          Seoul                          310       
NZ    New Zealand          Auckland                       326       
AU    Australia            Sydney                         360       

You can connect to a location by running /opt/adguardvpn_cli/adguardvpn-cli connect -l 'city, country or ISO code'`

	locations := ParseLocations(sampleContent)

	// Test that the result is not empty
	if len(locations) == 0 {
		t.Error("Expected non-empty locations slice, got empty slice")
	}

	// Count the number of lines that should result in location entries
	// (excluding header and non-location lines)
	expectedCount := 0
	lines := strings.SplitSeq(sampleContent, "\n")
	for line := range lines {
		// Remove ANSI codes for checking
		cleanLine := strings.ReplaceAll(line, "\x1b[1m", "")
		cleanLine = strings.ReplaceAll(cleanLine, "\x1b[0m", "")

		if strings.Contains(cleanLine, "COUNTRY") {
			continue // skip header
		}

		// Count lines that look like location entries (start with country code)
		fields := strings.Fields(cleanLine)
		if len(fields) >= 4 && len(fields[0]) == 2 { // ISO code should be 2 chars
			expectedCount++
		}
	}

	if len(locations) != expectedCount {
		t.Errorf("Expected %d locations, got %d", expectedCount, len(locations))
	}

	// Check that all expected countries are present
	expectedCountries := []string{"Latvia", "Germany", "Denmark", "Netherlands", "Italy", "France", "Switzerland", "Czechia", "Belgium", "Finland", "United Kingdom"}
	foundCountries := make(map[string]bool)
	for _, loc := range locations {
		foundCountries[loc.Country] = true
	}

	for _, expectedCountry := range expectedCountries {
		if !foundCountries[expectedCountry] {
			t.Errorf("Expected to find country %s in parsed locations", expectedCountry)
		}
	}

	// Check that specific known locations exist
	foundRiga := false
	foundLondon := false
	for _, loc := range locations {
		if loc.City == "Riga" && loc.Country == "Latvia" && loc.ISO == "LV" {
			foundRiga = true
			if loc.Ping != 29 {
				t.Errorf("Expected Riga to have ping 29, got %d", loc.Ping)
			}
		}
		if strings.Contains(loc.City, "London") && loc.Country == "United Kingdom" {
			foundLondon = true
			if loc.Ping != 52 {
				t.Errorf("Expected London to have ping 52, got %d", loc.Ping)
			}
		}
	}

	if !foundRiga {
		t.Error("Expected to find Riga, Latvia in parsed locations")
	}
	if !foundLondon {
		t.Error("Expected to find London, United Kingdom in parsed locations")
	}

	// Verify that locations have valid data
	for i, loc := range locations {
		if loc.ISO == "" {
			t.Errorf("Location at index %d has empty ISO code", i)
		}
		if loc.Country == "" {
			t.Errorf("Location at index %d has empty Country", i)
		}
		if loc.City == "" {
			t.Errorf("Location at index %d has empty City", i)
		}
	}
}

// TestParseLocationsWithANSICodes tests parsing when ANSI color codes are present
func TestParseLocationsWithANSICodes(t *testing.T) {
	// Content with ANSI codes similar to actual CLI output
	ansiContent := "\x1b[1mISO   COUNTRY              CITY                           PING ESTIMATE\n\x1b[0mLV    Latvia               Riga                           29        \nDE    Germany              Frankfurt                      37        \n"

	locations := ParseLocations(ansiContent)

	if len(locations) != 2 {
		t.Errorf("Expected 2 locations with ANSI codes, got %d", len(locations))
	}

	if len(locations) > 0 {
		if locations[0].ISO != "LV" || locations[0].Country != "Latvia" || locations[0].City != "Riga" || locations[0].Ping != 29 {
			t.Errorf("First location has incorrect data: %+v", locations[0])
		}
	}
}

// TestParseLocationsEmpty tests parsing empty or invalid input
func TestParseLocationsEmpty(t *testing.T) {
	// Test empty string
	locations := ParseLocations("")
	if len(locations) != 0 {
		t.Errorf("Expected empty result for empty input, got %d locations", len(locations))
	}

	// Test just header
	locations = ParseLocations("ISO   COUNTRY              CITY                           PING ESTIMATE\n")
	if len(locations) != 0 {
		t.Errorf("Expected empty result for header-only input, got %d locations", len(locations))
	}

	// Test header with ANSI codes only
	locations = ParseLocations("\x1b[1mISO   COUNTRY              CITY                           PING ESTIMATE\n\x1b[0m")
	if len(locations) != 0 {
		t.Errorf("Expected empty result for header with ANSI codes only, got %d locations", len(locations))
	}
}

// TestParseLocationsSingleLocation tests parsing a single location
func TestParseLocationsSingleLocation(t *testing.T) {
	singleLocation := "ISO   COUNTRY              CITY                           PING ESTIMATE\nUS    United States        New York                       121       \n"

	locations := ParseLocations(singleLocation)

	if len(locations) != 1 {
		t.Errorf("Expected 1 location, got %d", len(locations))
		return
	}

	if locations[0].ISO != "US" || locations[0].Country != "United States" || locations[0].City != "New York" || locations[0].Ping != 121 {
		t.Errorf("Single location has incorrect data: %+v", locations[0])
	}
}

// TestFindFastestLocation tests the findFastestLocation function
func TestFindFastestLocation(t *testing.T) {
	locations := []Location{
		{ISO: "US", Country: "US", City: "New York", Ping: 121},
		{ISO: "LV", Country: "Latvia", City: "Riga", Ping: 29},
		{ISO: "DE", Country: "Germany", City: "Berlin", Ping: 53},
	}

	fastest := FindFastestLocation(locations)

	if fastest == nil {
		t.Fatal("Expected to find fastest location, got nil")
	}

	if fastest.ISO != "LV" || fastest.Country != "Latvia" || fastest.City != "Riga" || fastest.Ping != 29 {
		t.Errorf("Fastest location is incorrect: %+v", fastest)
	}
}

// TestFindFastestLocationEmpty tests finding fastest location in empty slice
func TestFindFastestLocationEmpty(t *testing.T) {
	fastest := FindFastestLocation([]Location{})

	if fastest != nil {
		t.Errorf("Expected nil for empty slice, got %+v", fastest)
	}
}

// TestFindFastestLocationWithInvalidPing tests finding fastest location when some locations have invalid ping
func TestFindFastestLocationWithInvalidPing(t *testing.T) {
	locations := []Location{
		{ISO: "US", Country: "US", City: "New York", Ping: 9999}, // Invalid ping (9999 is used as default for invalid values)
		{ISO: "LV", Country: "Latvia", City: "Riga", Ping: 29},
		{ISO: "DE", Country: "Germany", City: "Berlin", Ping: 53},
	}

	fastest := FindFastestLocation(locations)

	if fastest == nil {
		t.Fatal("Expected to find fastest location, got nil")
	}

	if fastest.Ping != 29 {
		t.Errorf("Expected fastest location to have ping 29, got %d", fastest.Ping)
	}
}

// TestFilterLocations tests the FilterLocations function
func TestFilterLocations(t *testing.T) {
	locations := []Location{
		{ISO: "US", Country: "United States", City: "New York", Ping: 121},
		{ISO: "LV", Country: "Latvia", City: "Riga", Ping: 29},
		{ISO: "DE", Country: "Germany", City: "Berlin", Ping: 53},
		{ISO: "DE", Country: "Germany", City: "Frankfurt", Ping: 37},
	}

	// Test filtering by city
	filtered := FilterLocations(locations, "york")
	if len(filtered) != 1 {
		t.Errorf("Expected 1 location when filtering by 'york', got %d", len(filtered))
	}
	if len(filtered) == 1 && filtered[0].City != "New York" {
		t.Errorf("Expected to find 'New York', got '%s'", filtered[0].City)
	}

	// Test filtering by country
	filtered = FilterLocations(locations, "germany")
	if len(filtered) != 2 {
		t.Errorf("Expected 2 locations when filtering by 'germany', got %d", len(filtered))
	}

	// Test filtering with no results
	filtered = FilterLocations(locations, "nonexistent")
	if len(filtered) != 0 {
		t.Errorf("Expected 0 locations for 'nonexistent' query, got %d", len(filtered))
	}

	// Test with empty query
	filtered = FilterLocations(locations, "")
	if len(filtered) != 4 {
		t.Errorf("Expected all locations for empty query, got %d", len(filtered))
	}
}
