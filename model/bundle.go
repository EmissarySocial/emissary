package model

// Bundle represents a collection of files that are served as a single unit.
// Recognized content types are automatically minified via github.com/tdewolff/minify/v2.
type Bundle struct {
	ContentType  string `json:"contentType"`
	CacheControl string `json:"cacheControl"`
	Content      []byte `json:"-"`
}

// GetCacheControl returns the Cache-Control header for this bundle.
// If no Cache-Control header is defined, this method returns "public, max-age=3600"
func (bundle Bundle) GetCacheControl() string {

	if bundle.CacheControl == "" {
		return "public, max-age=3600"
	}

	return bundle.CacheControl
}
