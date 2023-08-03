package ascache

type OptionFunc func(*Client)

// WithPurgeFrequency option sets the frequency that expired documents will be purged from the cache
func WithPurgeFrequency(seconds int64) OptionFunc {
	return func(client *Client) {
		client.purgeFrequency = seconds
	}
}

// WithDefaultCache sets the default number of seconds that a document should be cached
func WithDefaultCache(seconds int) OptionFunc {
	return func(client *Client) {
		client.defaultCacheSeconds = seconds
	}
}

// WithMinCache option sets the minimum number of seconds that a document should be cached
func WithMinCache(seconds int) OptionFunc {
	return func(client *Client) {
		client.minCacheSeconds = seconds
	}
}

// WithMaxCache option sets the maximum number of seconds that a document should be cached
func WithMaxCache(seconds int) OptionFunc {
	return func(client *Client) {
		client.maxCacheSeconds = seconds
	}
}
