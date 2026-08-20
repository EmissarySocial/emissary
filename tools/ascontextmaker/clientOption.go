package ascontextmaker

// ClientOption is a configuration function that modifies a context-maker Client
type ClientOption func(*Client)

// WithMaxDepth sets the maximum number of replies that the context maker
// will follow before giving up. (Default is 16)
func WithMaxDepth(maxDepth int) ClientOption {
	return func(client *Client) {
		client.maxDepth = maxDepth
	}
}
