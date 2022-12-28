package service

import (
	"net/url"
	"strings"

	"golang.org/x/net/html"
)

/*******************************************
 * Helper Functions
 *******************************************/

// nodeAttribute searches for a specific attribute in a node and returns its value
func nodeAttribute(node *html.Node, name string) string {

	if node == nil {
		return ""
	}

	for _, attr := range node.Attr {
		if attr.Key == name {
			return attr.Val
		}
	}

	return ""
}

// TODO: HIGH: Scan all references and perhaps use https://pkg.go.dev/net/url#URL.ResolveReference instead?
func getRelativeURL(baseURL string, relativeURL string) string {

	// If the relative URL is already absolute, then just return it
	if strings.HasPrefix(relativeURL, "http://") || strings.HasPrefix(relativeURL, "https://") {
		return relativeURL
	}

	// If the relative URL is a root-relative URL, then assume HTTPS (it's 2022, for crying out loud)
	if strings.HasPrefix(relativeURL, "//") {
		return "https:" + relativeURL
	}

	// Parse the base URL so that we can do URL-math on it
	baseURLParsed, _ := url.Parse(baseURL)

	// If the relative URL is a path-relative URL, then just replace the path
	if strings.HasPrefix(relativeURL, "/") {
		baseURLParsed.Path = relativeURL
		return baseURLParsed.String()
	}

	// Otherwise, join the paths
	baseURLParsed.Path, _ = url.JoinPath(baseURLParsed.Path, relativeURL)
	return baseURLParsed.String()
}
