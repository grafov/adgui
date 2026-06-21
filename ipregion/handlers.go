package ipregion

import (
	"context"
	"regexp"
	"strings"
)

func (c *Checker) probePrimary(ctx context.Context, svc primaryService, ipVersion int, ip string) string {
	if svc.custom != nil {
		return cleanResult(svc.custom(c, ctx, ipVersion, ip))
	}

	url := strings.ReplaceAll(svc.urlTemplate, "{ip}", ip)
	body, err := c.client.get(ctx, url, effectiveIPVersion(svc.ipv6OverIPv4, ipVersion), svc.headers)
	if err != nil || body == "" || strings.Contains(strings.ToLower(body), "<html") {
		return NotAvailable
	}
	if svc.plainText {
		return cleanResult(strings.TrimSpace(body))
	}
	if !isValidJSON([]byte(body)) {
		return NotAvailable
	}
	return cleanResult(jsonPath([]byte(body), svc.jsonPath))
}

func effectiveIPVersion(ipv6OverIPv4 bool, ipVersion int) int {
	if ipv6OverIPv4 && ipVersion == 6 {
		return 4
	}
	return ipVersion
}

func (c *Checker) lookupCloudflare(ctx context.Context, ipVersion int, _ string) string {
	body, err := c.client.get(ctx, "https://www.cloudflare.com/cdn-cgi/trace", ipVersion, nil)
	if err != nil {
		return NotAvailable
	}
	for _, line := range strings.Split(body, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "loc=") {
			return cleanResult(strings.TrimPrefix(line, "loc="))
		}
	}
	return NotAvailable
}

func (c *Checker) lookupIPLocation(ctx context.Context, ipVersion int, _ string) string {
	ip := c.externalIPv4
	if ip == "" {
		ip = c.externalIPv6
	}
	form := "ip=" + ip
	body, err := c.client.postForm(ctx, "https://iplocation.com", ipVersion, form, nil)
	if err != nil {
		return NotAvailable
	}
	return cleanResult(jsonPath([]byte(body), ".country_code"))
}

func (c *Checker) lookup2IP(ctx context.Context, ipVersion int, _ string) string {
	body, err := c.client.get(ctx, "https://api.2ip.io", ipVersion, nil)
	if err != nil {
		return NotAvailable
	}
	return cleanResult(jsonPath([]byte(body), ".code"))
}

func (c *Checker) lookupGoogle(ctx context.Context, ipVersion int) string {
	body, err := c.client.get(ctx, "https://www.google.com", ipVersion, nil)
	if err != nil {
		return NotAvailable
	}

	if m := countryCodeRE.FindStringSubmatch(body); len(m) > 2 {
		return cleanResult(m[2])
	}
	if matches := countryCodeFallbackRE.FindAllStringSubmatch(body, -1); len(matches) > 0 {
		return cleanResult(matches[len(matches)-1][2])
	}

	playBody, err := c.client.get(ctx, "https://play.google.com/", ipVersion, map[string]string{
		"accept":                    "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8",
		"accept-language":           "en-US;q=0.9",
		"sec-fetch-dest":            "document",
		"sec-fetch-mode":            "navigate",
		"sec-fetch-site":            "none",
		"upgrade-insecure-requests": "1",
	})
	if err != nil {
		return NotAvailable
	}
	re := regexp.MustCompile(`\s+([^<(]+)`)
	country := strings.TrimSpace(re.FindString(playBody))
	if country == "" {
		return NotAvailable
	}
	code := lookupCountryByName(ctx, c.client, country, ipVersion)
	return cleanResult(code)
}

func (c *Checker) lookupGoogleSearchCaptcha(ctx context.Context, ipVersion int) string {
	body, err := c.client.get(ctx, "https://www.google.com/search?q=cats", ipVersion, map[string]string{
		"Accept-Language": "en-US,en;q=0.9",
	})
	if err != nil || body == "" {
		return NotAvailable
	}
	lower := strings.ToLower(body)
	if strings.Contains(lower, "unusual traffic from") ||
		strings.Contains(lower, "is blocked") ||
		strings.Contains(lower, "unaddressed abuse") {
		return "Yes"
	}
	return "No"
}

func (c *Checker) lookupYouTube(ctx context.Context, ipVersion int) string {
	body, err := c.client.get(ctx, "https://www.youtube.com", ipVersion, nil)
	if err == nil {
		re := regexp.MustCompile(`"countryCode":"(\w+)"`)
		if m := re.FindStringSubmatch(body); len(m) > 1 {
			result := cleanResult(m[1])
			if result != NotAvailable && len(result) <= 7 {
				c.setPeerResult("Google", ipVersion, result)
				return result
			}
		}
	}
	if peer := c.peerResult("Google", ipVersion); peer != "" {
		return peer
	}
	return NotAvailable
}

func (c *Checker) lookupYouTubePremium(ctx context.Context, ipVersion int) string {
	body, err := c.client.get(ctx, "https://www.youtube.com/premium", ipVersion, map[string]string{
		"Cookie":          "SOCS=CAISNQgDEitib3FfaWRlbnRpdHlmcm9udGVuZHVpc2VydmVyXzIwMjUwNzMwLjA1X3AwGgJlbiACGgYIgPC_xAY",
		"Accept-Language": "en-US,en;q=0.9",
	})
	if err != nil || body == "" {
		return NotAvailable
	}
	if strings.Contains(strings.ToLower(body), "youtube premium is not available in your country") {
		return "No"
	}
	return "Yes"
}

func (c *Checker) lookupYouTubeMusic(ctx context.Context, ipVersion int) string {
	body, err := c.client.get(ctx, "https://music.youtube.com/", ipVersion, map[string]string{
		"Cookie":          "SOCS=CAISNQgDEitib3FfaWRlbnRpdHlmcm9udGVuZHVpc2VydmVyXzIwMjUwNzMwLjA1X3AwGgJlbiACGgYIgPC_xAY",
		"Accept-Language": "en-US,en;q=0.9",
	})
	if err != nil || body == "" {
		return NotAvailable
	}
	if strings.Contains(body, "YouTube Music is not available in your area") {
		return "No"
	}
	return "Yes"
}

func (c *Checker) lookupTwitch(ctx context.Context, ipVersion int) string {
	payload := `[{"operationName":"VerifyEmail_CurrentUser","variables":{},"extensions":{"persistedQuery":{"version":1,"sha256Hash":"f9e7dcdf7e99c314c82d8f7f725fab5f99d1df3d7359b53c9ae122deec590198"}}}]`
	body, err := c.client.postJSON(ctx, "https://gql.twitch.tv/gql", ipVersion, payload, map[string]string{
		"Client-Id": "kimne78kx3ncx6brgo4mv6wki5h1ko",
	})
	if err != nil {
		return NotAvailable
	}
	return cleanResult(jsonPath([]byte(body), ".0.data.requestInfo.countryCode"))
}

func (c *Checker) lookupChatGPT(ctx context.Context, ipVersion int) string {
	body, err := c.client.postJSON(ctx, "https://ab.chatgpt.com/v1/initialize", ipVersion, "",
		map[string]string{"Statsig-Api-Key": "client-zUdXdSTygXJdzoE0sWTkP8GKTVsUMF2IRM7ShVO2JAG"})
	if err != nil {
		return NotAvailable
	}
	return cleanResult(jsonPath([]byte(body), ".derived_fields.country"))
}

func (c *Checker) lookupNetflix(ctx context.Context, ipVersion int) string {
	url := "https://api.fast.com/netflix/speedtest/v2?https=true&token=YXNkZmFzZGxmbnNkYWZoYXNkZmhrYWxm&urlCount=1"
	body, err := c.client.get(ctx, url, ipVersion, nil)
	if err != nil || !isValidJSON([]byte(body)) {
		return NotAvailable
	}
	return cleanResult(jsonPath([]byte(body), ".client.location.country"))
}

func (c *Checker) lookupSpotify(ctx context.Context, ipVersion int) string {
	body, err := c.client.get(ctx, "https://accounts.spotify.com/status", ipVersion, nil)
	if err != nil {
		return NotAvailable
	}
	re := regexp.MustCompile(`"geoLocationCountryCode":"([^"]*)"`)
	if m := re.FindStringSubmatch(body); len(m) > 1 {
		return cleanResult(m[1])
	}
	return NotAvailable
}

func (c *Checker) lookupSpotifySignup(ctx context.Context, ipVersion int) string {
	apiKey := c.keys.SpotifyAPIKey
	if apiKey == "" {
		apiKey = "142b583129b2df829de3656f9eb484e6"
	}
	clientID := c.keys.SpotifyClientID
	if clientID == "" {
		clientID = "9a8d2f0ce77a4e248bb71fefcb557637"
	}
	url := "https://spclient.wg.spotify.com/signup/public/v1/account/?validate=1&key=" + apiKey
	body, err := c.client.get(ctx, url, ipVersion, map[string]string{
		"X-Client-Id": clientID,
	})
	if err != nil {
		return NotAvailable
	}
	status := jsonPath([]byte(body), ".status")
	launched := jsonPath([]byte(body), ".is_country_launched")
	if status == "120" || status == "320" || launched == "false" {
		return "No"
	}
	return "Yes"
}

func (c *Checker) lookupDeezer(ctx context.Context, ipVersion int) string {
	body, err := c.client.get(ctx, "https://www.deezer.com/en/offers", ipVersion, nil)
	if err != nil {
		return NotAvailable
	}
	re := regexp.MustCompile(`'country': '([^']*)'`)
	if m := re.FindStringSubmatch(body); len(m) > 1 {
		return cleanResult(m[1])
	}
	return NotAvailable
}

func (c *Checker) lookupReddit(ctx context.Context, ipVersion int) string {
	basic := "Basic b2hYcG9xclpZdWIxa2c6"
	ua := "Reddit/Version 2025.29.0/Build 2529021/Android 13"
	body, err := c.client.postJSON(ctx, "https://www.reddit.com/auth/v2/oauth/access-token/loid", ipVersion,
		`{"scopes":["email"]}`, map[string]string{
			"Authorization": basic,
			"User-Agent":    ua,
		})
	if err != nil {
		return NotAvailable
	}
	token := jsonPath([]byte(body), ".access_token")
	if token == "" {
		return NotAvailable
	}
	body, err = c.client.postJSON(ctx, "https://gql-fed.reddit.com", ipVersion,
		`{"operationName":"UserLocation","variables":{},"extensions":{"persistedQuery":{"version":1,"sha256Hash":"f07de258c54537e24d7856080f662c1b1268210251e5789c8c08f20d76cc8ab2"}}}`,
		map[string]string{
			"Authorization": "Bearer " + token,
			"User-Agent":    ua,
		})
	if err != nil {
		return NotAvailable
	}
	return cleanResult(jsonPath([]byte(body), ".data.userLocation.countryCode"))
}

func (c *Checker) lookupRedditGuest(ctx context.Context, ipVersion int) string {
	body, err := c.client.get(ctx, "https://www.reddit.com", ipVersion, nil)
	if err != nil || body == "" {
		return "No"
	}
	return "Yes"
}

func (c *Checker) lookupAmazonPrime(ctx context.Context, ipVersion int) string {
	body, err := c.client.get(ctx, "https://www.primevideo.com", ipVersion, nil)
	if err != nil {
		return NotAvailable
	}
	if strings.Contains(body, "isServiceRestricted") {
		return "No"
	}
	re := regexp.MustCompile(`"currentTerritory":"([^"]+)"`)
	if m := re.FindStringSubmatch(body); len(m) > 1 && len(m[1]) >= 2 {
		return cleanResult(m[1][:2])
	}
	return NotAvailable
}

func (c *Checker) lookupApple(ctx context.Context, ipVersion int) string {
	body, err := c.client.get(ctx, "https://gspe1-ssl.ls.apple.com/pep/gcc", ipVersion, nil)
	if err != nil {
		return NotAvailable
	}
	return cleanResult(strings.TrimSpace(body))
}

func (c *Checker) lookupSteam(ctx context.Context, ipVersion int) string {
	body, err := c.client.doHeadHeaders(ctx, "https://store.steampowered.com", ipVersion)
	if err != nil {
		return NotAvailable
	}
	re := regexp.MustCompile(`steamCountry=([^%;]*)`)
	if m := re.FindStringSubmatch(body); len(m) > 1 {
		return cleanResult(m[1])
	}
	return NotAvailable
}

func (c *Checker) lookupPlayStation(ctx context.Context, ipVersion int) string {
	body, err := c.client.doHeadHeaders(ctx, "https://www.playstation.com", ipVersion)
	if err != nil {
		return NotAvailable
	}
	re := regexp.MustCompile(`(?i)country=([A-Z]+)`)
	if m := re.FindStringSubmatch(body); len(m) > 1 {
		return cleanResult(m[1])
	}
	return NotAvailable
}

func (c *Checker) lookupTikTok(ctx context.Context, ipVersion int) string {
	body, err := c.client.get(ctx, "https://www.tiktok.com/api/v1/web-cookie-privacy/config?appId=1988", ipVersion, nil)
	if err != nil {
		return NotAvailable
	}
	return cleanResult(jsonPath([]byte(body), ".body.appProps.region"))
}

func (c *Checker) lookupOokla(ctx context.Context, ipVersion int) string {
	body, err := c.client.get(ctx, "https://www.speedtest.net/api/js/config-sdk", ipVersion, nil)
	if err != nil {
		return NotAvailable
	}
	return cleanResult(jsonPath([]byte(body), ".location.countryCode"))
}

func (c *Checker) lookupJetBrains(ctx context.Context, ipVersion int) string {
	body, err := c.client.get(ctx, "https://data.services.jetbrains.com/geo", ipVersion, nil)
	if err != nil {
		return NotAvailable
	}
	return cleanResult(jsonPath([]byte(body), ".code"))
}

func (c *Checker) lookupBing(ctx context.Context, ipVersion int) string {
	body, err := c.client.get(ctx, "https://www.bing.com/search?q=cats", ipVersion, nil)
	if err != nil {
		return NotAvailable
	}
	if strings.Contains(body, "cn.bing.com") {
		return "CN"
	}
	re := regexp.MustCompile(`Region\s*:\s*"([^"]+)"`)
	region := ""
	if m := re.FindStringSubmatch(body); len(m) > 1 {
		region = m[1]
	}
	if len(region) >= 2 {
		region = region[:2]
	}
	if region == "WW" {
		liveBody, err := c.client.get(ctx, "https://login.live.com", ipVersion, nil)
		if err == nil {
			re2 := regexp.MustCompile(`"sRequestCountry":"([^"]*)"`)
			if m := re2.FindStringSubmatch(liveBody); len(m) > 1 {
				region = m[1]
			}
		}
	}
	return cleanResult(region)
}
