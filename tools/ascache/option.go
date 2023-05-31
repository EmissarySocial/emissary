package ascache

type OptionFunc func(*Client)

// WithTimeout option sets the length of time that documents will be cached
func WithTimeout(seconds int64) OptionFunc {
	return func(client *Client) {
		client.expireSeconds = seconds
	}
}

// WithPurgeFrequency option sets the frequency that expired documents will be purged from the cache
func WithPurgeFrequency(seconds int64) OptionFunc {
	return func(client *Client) {
		client.purgeFrequency = seconds
	}
}
