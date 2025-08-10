package ascrawler

type LoadOption func(*loadConfig)

func AtDepth(depth int) LoadOption {
	return func(client *loadConfig) {
		client.currentDepth = depth
	}
}
