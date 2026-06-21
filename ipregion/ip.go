package ipregion

import (
	"context"
	"encoding/json"
	"regexp"
	"strings"
)

var identityServices = []string{
	"https://ident.me",
	"https://ifconfig.me",
	"https://api64.ipify.org",
}

func detectExternalIP(ctx context.Context, client *httpClient, ipVersion int) string {
	counts := make(map[string]int)
	for _, svc := range identityServices {
		body, err := client.get(ctx, svc, ipVersion, nil)
		if err != nil {
			continue
		}
		ip := strings.TrimSpace(body)
		if ip == "" {
			continue
		}
		counts[ip]++
	}

	for ip, n := range counts {
		if n >= 2 {
			return ip
		}
	}
	for ip := range counts {
		return ip
	}
	return ""
}

func fetchASN(ctx context.Context, client *httpClient, ip string, ipVersion int) (asn, org string) {
	if ip == "" {
		return "", ""
	}
	body, err := client.get(ctx, "https://ipinfo.check.place/"+ip, ipVersion, nil)
	if err == nil && body != "" {
		asn = jsonPath([]byte(body), ".ASN.AutonomousSystemNumber")
		org = jsonPath([]byte(body), ".ASN.AutonomousSystemOrganization")
		if asn != "" && org != "" {
			return asn, org
		}
	}

	body, err = client.get(ctx, "https://geoip.oxl.app/api/ip/"+ip, ipVersion, nil)
	if err != nil || body == "" {
		return "", ""
	}
	asn = jsonPath([]byte(body), ".asn")
	org = jsonPath([]byte(body), ".organization.name")
	org = strings.TrimPrefix(org, "null")
	return asn, org
}

var countryCodeRE = regexp.MustCompile(`"([a-z]{2})_([A-Z]{2})"`)
var countryCodeFallbackRE = regexp.MustCompile(`"([a-z]{2})-([A-Z]{2})"`)

func lookupCountryByName(ctx context.Context, client *httpClient, country string, ipVersion int) string {
	body, err := client.get(ctx, "https://restcountries.com/v3.1/all?fields=name,cca2", ipVersion, nil)
	if err != nil || body == "" || !strings.Contains(body, `"name"`) {
		return ""
	}
	country = strings.ToLower(strings.TrimSpace(country))

	var entries []struct {
		Name struct {
			Common string `json:"common"`
		} `json:"name"`
		CCA2 string `json:"cca2"`
	}
	if err := json.Unmarshal([]byte(body), &entries); err != nil {
		return ""
	}
	for _, e := range entries {
		if strings.EqualFold(e.Name.Common, country) {
			return e.CCA2
		}
	}
	return ""
}
