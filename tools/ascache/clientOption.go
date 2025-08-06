package ascache

type ClientOptionFunc func(*Client)

// WithReadWrite option sets the client to read+write mode
func WithReadWrite() ClientOptionFunc {
	return func(client *Client) {
		client.cacheMode = CacheModeReadWrite
	}
}

// WithReadOnly option sets the client to read-only mode.
// The cache will only read values from the database, and will not
// write new values to the database.
func WithReadOnly() ClientOptionFunc {
	return func(client *Client) {
		client.cacheMode = CacheModeReadOnly
	}
}

// WithWriteOnly option sets the client to write-only mode.
// The cache will only write values to the database, and will not
// check the database for existing values.
func WithWriteOnly() ClientOptionFunc {
	return func(client *Client) {
		client.cacheMode = CacheModeWriteOnly
	}
}

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
