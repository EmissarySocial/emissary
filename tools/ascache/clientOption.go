package ascache

type ClientOptionFunc func(*Client)

// WithPurgeFrequency option sets the frequency that expired documents will be purged from the cache
func WithPurgeFrequency(seconds int64) ClientOptionFunc {
	return func(client *Client) {
		client.purgeFrequency = seconds
	}
}

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
