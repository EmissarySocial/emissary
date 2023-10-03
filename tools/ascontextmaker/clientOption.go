package ascontextmaker

type ClientOption func(*Client)

// WithMaxDepth sets the maximum number of replies that the context maker
// will follow before giving up.
func WithMaxDepth(maxDepth int) ClientOption {
	return func(client *Client) {
		client.maxDepth = maxDepth
	}
}
