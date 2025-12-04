package locations_test

import (
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"adgui/locations"
)

var _ = Describe("Location Parsing", func() {
	Context("when parsing locations from sample content", func() {
		It("should parse locations correctly from sample content", func() {
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

			parsedLocations := locations.ParseLocations(sampleContent)

			// Test that the result is not empty
			Expect(parsedLocations).ToNot(BeEmpty())

			// Count the number of lines that should result in location entries
			// (excluding header and non-location lines)
			expectedCount := 0
			for line := range strings.SplitSeq(sampleContent, "\n") {
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

			Expect(parsedLocations).To(HaveLen(expectedCount))

			// Check that all expected countries are present
			expectedCountries := []string{"Latvia", "Germany", "Denmark", "Netherlands", "Italy", "France", "Switzerland", "Czechia", "Belgium", "Finland", "United Kingdom"}
			foundCountries := make(map[string]bool)
			for _, loc := range parsedLocations {
				foundCountries[loc.Country] = true
			}

			for _, expectedCountry := range expectedCountries {
				Expect(foundCountries[expectedCountry]).To(BeTrue(), "Expected to find country %s in parsed locations", expectedCountry)
			}

			// Check that specific known locations exist
			var foundRiga, foundLondon bool
			for _, loc := range parsedLocations {
				if loc.City == "Riga" && loc.Country == "Latvia" && loc.ISO == "LV" {
					foundRiga = true
					Expect(loc.Ping).To(Equal(29), "Expected Riga to have ping 29")
				}
				if strings.Contains(loc.City, "London") && loc.Country == "United Kingdom" {
					foundLondon = true
					Expect(loc.Ping).To(Equal(52), "Expected London to have ping 52")
				}
			}

			Expect(foundRiga).To(BeTrue(), "Expected to find Riga, Latvia in parsed locations")
			Expect(foundLondon).To(BeTrue(), "Expected to find London, United Kingdom in parsed locations")

			// Verify that locations have valid data
			for i, loc := range parsedLocations {
				Expect(loc.ISO).ToNot(BeEmpty(), "Location at index %d has empty ISO code", i)
				Expect(loc.Country).ToNot(BeEmpty(), "Location at index %d has empty Country", i)
				Expect(loc.City).ToNot(BeEmpty(), "Location at index %d has empty City", i)
			}
		})
	})

	Context("when parsing locations with ANSI codes", func() {
		It("should correctly handle ANSI codes in input", func() {
			// Content with ANSI codes similar to actual CLI output
			ansiContent := "\x1b[1mISO   COUNTRY              CITY                           PING ESTIMATE\n\x1b[0mLV    Latvia               Riga                           29        \nDE    Germany              Frankfurt                      37        \n"

			parsedLocations := locations.ParseLocations(ansiContent)

			Expect(parsedLocations).To(HaveLen(2))

			if len(parsedLocations) > 0 {
				Expect(parsedLocations[0].ISO).To(Equal("LV"))
				Expect(parsedLocations[0].Country).To(Equal("Latvia"))
				Expect(parsedLocations[0].City).To(Equal("Riga"))
				Expect(parsedLocations[0].Ping).To(Equal(29))
			}
		})
	})

	Context("when parsing empty or invalid input", func() {
		It("should return empty slice for empty input", func() {
			parsedLocations := locations.ParseLocations("")
			Expect(parsedLocations).To(BeEmpty())
		})

		It("should return empty slice for header-only input", func() {
			parsedLocations := locations.ParseLocations("ISO   COUNTRY              CITY                           PING ESTIMATE\n")
			Expect(parsedLocations).To(BeEmpty())
		})

		It("should return empty slice for header with ANSI codes only", func() {
			parsedLocations := locations.ParseLocations("\x1b[1mISO   COUNTRY              CITY                           PING ESTIMATE\n\x1b[0m")
			Expect(parsedLocations).To(BeEmpty())
		})
	})

	Context("when parsing single location", func() {
		It("should correctly parse single location", func() {
			singleLocation := "ISO   COUNTRY              CITY                           PING ESTIMATE\nUS    United States        New York                       121       \n"

			parsedLocations := locations.ParseLocations(singleLocation)

			Expect(parsedLocations).To(HaveLen(1))

			Expect(parsedLocations[0].ISO).To(Equal("US"))
			Expect(parsedLocations[0].Country).To(Equal("United States"))
			Expect(parsedLocations[0].City).To(Equal("New York"))
			Expect(parsedLocations[0].Ping).To(Equal(121))
		})
	})
})

var _ = Describe("FindFastestLocation", func() {
	Context("when finding the fastest location", func() {
		It("should return the location with the lowest ping", func() {
			testLocations := []locations.Location{
				{ISO: "US", Country: "US", City: "New York", Ping: 121},
				{ISO: "LV", Country: "Latvia", City: "Riga", Ping: 29},
				{ISO: "DE", Country: "Germany", City: "Berlin", Ping: 53},
			}

			fastest := locations.FindFastestLocation(testLocations)

			Expect(fastest).ToNot(BeNil())
			Expect(fastest.ISO).To(Equal("LV"))
			Expect(fastest.Country).To(Equal("Latvia"))
			Expect(fastest.City).To(Equal("Riga"))
			Expect(fastest.Ping).To(Equal(29))
		})

		It("should return nil for empty slice", func() {
			fastest := locations.FindFastestLocation([]locations.Location{})
			Expect(fastest).To(BeNil())
		})

		It("should handle locations with invalid ping values", func() {
			testLocations := []locations.Location{
				{ISO: "US", Country: "US", City: "New York", Ping: 9999}, // Invalid ping (9999 is used as default for invalid values)
				{ISO: "LV", Country: "Latvia", City: "Riga", Ping: 29},
				{ISO: "DE", Country: "Germany", City: "Berlin", Ping: 53},
			}

			fastest := locations.FindFastestLocation(testLocations)

			Expect(fastest).ToNot(BeNil())
			Expect(fastest.Ping).To(Equal(29))
		})
	})
})

var _ = Describe("FilterLocations", func() {
	Context("when filtering locations", func() {
		var testLocations []locations.Location

		BeforeEach(func() {
			testLocations = []locations.Location{
				{ISO: "US", Country: "United States", City: "New York", Ping: 121},
				{ISO: "LV", Country: "Latvia", City: "Riga", Ping: 29},
				{ISO: "DE", Country: "Germany", City: "Berlin", Ping: 53},
				{ISO: "DE", Country: "Germany", City: "Frankfurt", Ping: 37},
			}
		})

		It("should filter by city name", func() {
			filtered := locations.FilterLocations(testLocations, "york")
			Expect(filtered).To(HaveLen(1))
			Expect(filtered[0].City).To(Equal("New York"))
		})

		It("should filter by country name", func() {
			filtered := locations.FilterLocations(testLocations, "germany")
			Expect(filtered).To(HaveLen(2))
		})

		It("should return empty slice when no matches found", func() {
			filtered := locations.FilterLocations(testLocations, "nonexistent")
			Expect(filtered).To(BeEmpty())
		})

		It("should return all locations for empty query", func() {
			filtered := locations.FilterLocations(testLocations, "")
			Expect(filtered).To(HaveLen(4))
		})
	})
})
