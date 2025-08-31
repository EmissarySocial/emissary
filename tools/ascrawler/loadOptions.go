package ascrawler

type LoadOption func(*loadConfig)

// WithoutCrawler deactivates the crawler
// for this operation
func WithoutCrawler() LoadOption {
	return func(config *loadConfig) {
		config.useCrawler = false
	}
}
