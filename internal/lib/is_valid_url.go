package lib

import neturl "net/url"

func IsValidURL(url string) bool {
	parsed, err := neturl.Parse(url)
	if err != nil {
		return false
	}

	if (parsed.Scheme != "http" && parsed.Scheme != "https") ||
		parsed.Hostname() == "" {
		return false
	}

	return true
}
