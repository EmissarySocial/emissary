package ascache

type OptionFunc func(*Client)

// WithPurgeFrequency option sets the frequency that expired documents will be purged from the cache
func WithPurgeFrequency(seconds int64) OptionFunc {
	return func(client *Client) {
		client.purgeFrequency = seconds
	}
}

// WithActorCache option sets caching values for Actors
func WithActorCache(defaultValue int, minimum int, maximum int) OptionFunc {
	return func(client *Client) {
		client.actorCache = CacheConfig{
			DefaultSeconds: defaultValue,
			MinimumSeconds: minimum,
			MaximumSeconds: maximum,
		}
	}
}

// WithDocumentCache option sets caching values for Documents
func WithDocumentCache(defaultValue int, minimum int, maximum int) OptionFunc {
	return func(client *Client) {
		client.documentCache = CacheConfig{
			DefaultSeconds: defaultValue,
			MinimumSeconds: minimum,
			MaximumSeconds: maximum,
		}
	}
}

// WithReadWrite option sets the client to read+write mode
func WithReadWrite() OptionFunc {
	return func(client *Client) {
		client.cacheMode = CacheModeReadWrite
	}
}

// WithReadOnly option sets the client to read-only mode.
// The cache will only read values from the database, and will not
// write new values to the database.
func WithReadOnly() OptionFunc {
	return func(client *Client) {
		client.cacheMode = CacheModeReadOnly
	}
}

// WithWriteOnly option sets the client to write-only mode.
// The cache will only write values to the database, and will not
// check the database for existing values.
func WithWriteOnly() OptionFunc {
	return func(client *Client) {
		client.cacheMode = CacheModeWriteOnly
	}
}
