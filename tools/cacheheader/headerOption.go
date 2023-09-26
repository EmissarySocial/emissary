package cacheheader

type HeaderOption func(*Header)

// AsPublicCache sets the parser to treat the http.Header as a
// PUBLIC (or shared) cache. In this mode, some shared cache values
// are treated differently.
func AsPublicCache() HeaderOption {
	return func(header *Header) {
		header.asPublicCache = true
	}
}

// AsPrivateCache sets the parser to treat the http.Header as a PRIVATE
// cache.  (Default behavior)
func AsPrivateCache() HeaderOption {
	return func(header *Header) {
		header.asPublicCache = false
	}
}
