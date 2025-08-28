package ascache

type ClientOptionFunc func(*Client)

// WithObeyHeaders instructs the cache to use HTTP headers
// to determine whether or not to use the cache.
func WithObeyHeaders() ClientOptionFunc {
	return func(client *Client) {
		client.obeyHeaders = true
	}
}

// WithIgnoreHeaders instructs the cache to ignore HTTP headers
// and always use the cache.
func WithIgnoreHeaders() ClientOptionFunc {
	return func(client *Client) {
		client.obeyHeaders = false
	}
}
