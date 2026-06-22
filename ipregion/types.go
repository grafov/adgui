package ipregion

import "time"

const (
	// NotAvailable is shown when a service could not determine the region.
	NotAvailable = "N/A"

	defaultTimeout        = 6 * time.Second
	defaultMaxConcurrency = 5

	defaultUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 " +
		"(KHTML, like Gecko) Chrome/135.0.0.0 Safari/537.36"
)

// Group identifies a service category.
type Group string

const (
	GroupPrimary Group = "primary"
	GroupCustom  Group = "custom"
	GroupCDN     Group = "cdn"
)

// ServiceKeys holds API keys loaded from ~/.config/adgui/service-keys.
type ServiceKeys struct {
	IPRegistryKey    string
	GeoapifyKey      string
	SpotifyClientID  string
	SpotifyAPIKey    string
	AirportCodesAuth string
}

// Options configures an IP region scan.
type Options struct {
	Groups          []Group
	IPv4Only        bool
	IPv6Only        bool
	Timeout         time.Duration
	UserAgent       string
	ServiceKeys     ServiceKeys
	MaxConcurrency  int
	OnProgress      func(Progress)
	IPv6OverIPv4IDs map[string]bool
}

// Progress reports scan advancement for UI updates.
type Progress struct {
	Service   string
	Completed int
	Total     int
}

// ServiceResult is one row in the scan report.
type ServiceResult struct {
	Group   Group  `json:"group"`
	Service string `json:"service"`
	IPv4    string `json:"ipv4"`
	IPv6    string `json:"ipv6"`
}

// Report is the full scan output.
type Report struct {
	ExternalIPv4 string          `json:"external_ipv4"`
	ExternalIPv6 string          `json:"external_ipv6"`
	ASN          string          `json:"asn"`
	ASNOrg       string          `json:"asn_org"`
	Results      []ServiceResult `json:"results"`
}

// CachedReport is a persisted scan result with metadata for the Region IP cache.
type CachedReport struct {
	CheckedAt time.Time `json:"checked_at"`
	VPNOff    bool      `json:"vpn_off"`
	ISO       string    `json:"iso,omitempty"`
	Location  string    `json:"location,omitempty"`
	Report    Report    `json:"report"`
}

// CountryStat is one entry in the summary histogram.
type CountryStat struct {
	Code     string
	Name     string
	IPv4Pct  int
	IPv6Pct  int
	IPv4Count int
	IPv6Count int
}

// Summary holds aggregated country percentages.
type Summary struct {
	IPv4Total int
	IPv6Total int
	Countries []CountryStat
}
