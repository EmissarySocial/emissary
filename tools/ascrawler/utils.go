package ascrawler

import (
	"net/url"
	"strings"
)

func isValidURL(uri string) bool {

	if strings.HasPrefix(uri, "https") || strings.HasPrefix(uri, "http") {
		_, err := url.ParseRequestURI(uri)
		return err == nil
	}

	return false
}
