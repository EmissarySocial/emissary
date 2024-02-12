package model

// Bundle represents a collection of files that are packaged into a Theme, Template, or Widget and are served as a single unit.
// Recognized content types are automatically minified via github.com/tdewolff/minify/v2.
type Bundle struct {
	ContentType  string `json:"content-type"`
	CacheControl string `json:"cache-control"`
	Content      []byte `json:"-"`
}

// GetCacheControl returns the Cache-Control header for this bundle.
// If no Cache-Control header is defined, this method returns "public, max-age=3600"
func (bundle Bundle) GetCacheControl() string {

	if bundle.CacheControl == "" {
		return "public, max-age=3600" // Default cache-control header. Cache resources for 1 hour.
	}

	return bundle.CacheControl
}
