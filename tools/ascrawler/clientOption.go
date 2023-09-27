package ascrawler

type ClientOption func(*Client)

func WithMaxDepth(maxDepth int) ClientOption {
	return func(client *Client) {
		client.maxDepth = maxDepth
	}
}
