package cache

type OptionFunc func(*Client)

// WithTimeout sets the lenght of time that documents will be cached
func WithTimeout(seconds int64) OptionFunc {
	return func(client *Client) {
		client.timeoutSeconds = seconds
	}
}

// WithPurgeFrequency sets the frequency that expired documents will be purged from the cache
func WithPurgeFrequency(seconds int64) OptionFunc {
	return func(client *Client) {
		client.purgeFrequency = seconds
	}
}
