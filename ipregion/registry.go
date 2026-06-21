package ipregion

import "context"

// primaryService defines a GeoIP API probe.
type primaryService struct {
	id           string
	displayName  string
	urlTemplate  string
	jsonPath     string
	headers      map[string]string
	plainText    bool
	custom       func(c *Checker, ctx context.Context, ipVersion int, ip string) string
	ipv6OverIPv4 bool
}

func defaultPrimaryServices(keys ServiceKeys) []primaryService {
	ipregistryKey := keys.IPRegistryKey
	if ipregistryKey == "" {
		ipregistryKey = "sb69ksjcajfs4c"
	}
	geoapifyKey := keys.GeoapifyKey
	if geoapifyKey == "" {
		geoapifyKey = "b8568cb9afc64fad861a69edbddb2658"
	}

	return []primaryService{
		{id: "MAXMIND", displayName: "maxmind.com", urlTemplate: "https://geoip.maxmind.com/geoip/v2.1/city/me",
			jsonPath: ".country.iso_code", headers: map[string]string{"Referer": "https://www.maxmind.com"}},
		{id: "RIPE", displayName: "rdap.db.ripe.net", urlTemplate: "https://rdap.db.ripe.net/ip/{ip}", jsonPath: ".country"},
		{id: "IPINFO_IO", displayName: "ipinfo.io", urlTemplate: "https://ipinfo.io/widget/demo/{ip}",
			jsonPath: ".data.country", ipv6OverIPv4: true},
		{id: "CLOUDFLARE", displayName: "cloudflare.com", custom: (*Checker).lookupCloudflare},
		{id: "IPREGISTRY", displayName: "ipregistry.co",
			urlTemplate: "https://api.ipregistry.co/{ip}?hostname=true&key=" + ipregistryKey,
			jsonPath: ".location.country.code", headers: map[string]string{"Origin": "https://ipregistry.co"}},
		{id: "IPAPI_CO", displayName: "ipapi.co", urlTemplate: "https://ipapi.co/{ip}/json", jsonPath: ".country"},
		{id: "IPLOCATION_COM", displayName: "iplocation.com", custom: (*Checker).lookupIPLocation},
		{id: "COUNTRY_IS", displayName: "country.is", urlTemplate: "https://api.country.is/{ip}", jsonPath: ".country"},
		{id: "GEOAPIFY_COM", displayName: "geoapify.com",
			urlTemplate: "https://api.geoapify.com/v1/ipinfo?ip={ip}&apiKey=" + geoapifyKey, jsonPath: ".country.iso_code"},
		{id: "GEOJS_IO", displayName: "geojs.io", urlTemplate: "https://get.geojs.io/v1/ip/country.json?ip={ip}", jsonPath: ".0.country"},
		{id: "IPAPI_IS", displayName: "ipapi.is", urlTemplate: "https://api.ipapi.is/?q={ip}", jsonPath: ".location.country_code"},
		{id: "IPBASE_COM", displayName: "ipbase.com", urlTemplate: "https://api.ipbase.com/v2/info?ip={ip}",
			jsonPath: ".data.location.country.alpha2"},
		{id: "IPQUERY_IO", displayName: "ipquery.io", urlTemplate: "https://api.ipquery.io/{ip}", jsonPath: ".location.country_code"},
		{id: "IPWHO_IS", displayName: "ipwho.is", urlTemplate: "https://ipwho.is/{ip}", jsonPath: ".country_code"},
		{id: "IPAPI_COM", displayName: "ip-api.com", urlTemplate: "https://demo.ip-api.com/json/{ip}?fields=countryCode",
			jsonPath: ".countryCode", headers: map[string]string{"Origin": "https://ip-api.com"}},
		{id: "2IP", displayName: "2ip.io", custom: (*Checker).lookup2IP},
	}
}

// customService defines a popular web service probe.
type customService struct {
	id          string
	displayName string
	probe       func(c *Checker, ctx context.Context, ipVersion int) string
}

func defaultCustomServices() []customService {
	return []customService{
		{id: "GOOGLE", displayName: "Google", probe: (*Checker).lookupGoogle},
		{id: "GOOGLE_SEARCH_CAPTCHA", displayName: "Google Search Captcha", probe: (*Checker).lookupGoogleSearchCaptcha},
		{id: "YOUTUBE", displayName: "YouTube", probe: (*Checker).lookupYouTube},
		{id: "YOUTUBE_PREMIUM", displayName: "YouTube Premium", probe: (*Checker).lookupYouTubePremium},
		{id: "YOUTUBE_MUSIC", displayName: "YouTube Music", probe: (*Checker).lookupYouTubeMusic},
		{id: "TWITCH", displayName: "Twitch", probe: (*Checker).lookupTwitch},
		{id: "CHATGPT", displayName: "ChatGPT", probe: (*Checker).lookupChatGPT},
		{id: "NETFLIX", displayName: "Netflix", probe: (*Checker).lookupNetflix},
		{id: "SPOTIFY", displayName: "Spotify", probe: (*Checker).lookupSpotify},
		{id: "SPOTIFY_SIGNUP", displayName: "Spotify Signup", probe: (*Checker).lookupSpotifySignup},
		{id: "DEEZER", displayName: "Deezer", probe: (*Checker).lookupDeezer},
		{id: "REDDIT", displayName: "Reddit", probe: (*Checker).lookupReddit},
		{id: "REDDIT_GUEST_ACCESS", displayName: "Reddit (Guest Access)", probe: (*Checker).lookupRedditGuest},
		{id: "AMAZON_PRIME", displayName: "Amazon Prime", probe: (*Checker).lookupAmazonPrime},
		{id: "APPLE", displayName: "Apple", probe: (*Checker).lookupApple},
		{id: "STEAM", displayName: "Steam", probe: (*Checker).lookupSteam},
		{id: "PLAYSTATION", displayName: "PlayStation", probe: (*Checker).lookupPlayStation},
		{id: "TIKTOK", displayName: "Tiktok", probe: (*Checker).lookupTikTok},
		{id: "OOKLA_SPEEDTEST", displayName: "Ookla Speedtest", probe: (*Checker).lookupOokla},
		{id: "JETBRAINS", displayName: "JetBrains", probe: (*Checker).lookupJetBrains},
		{id: "BING", displayName: "Microsoft (Bing)", probe: (*Checker).lookupBing},
	}
}
