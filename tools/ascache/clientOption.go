package ascache

// ClientOptionFunc is a configuration function that modifies a cache Client
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

// WithFailureMemo shares a FailureMemo with this Client, so URLs that recently failed with a server
// or network error or a 403 are not fetched again until its window passes.
func WithFailureMemo(memo *FailureMemo) ClientOptionFunc {
	return func(client *Client) {
		client.failures = memo
	}
}
