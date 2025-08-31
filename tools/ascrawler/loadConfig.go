package ascrawler

type loadConfig struct {
	useCrawler bool
}

func parseLoadConfig(options ...any) loadConfig {

	result := loadConfig{
		useCrawler: true,
	}

	for _, option := range options {
		if loadOption, isLoadOption := option.(LoadOption); isLoadOption {
			loadOption(&result)
		}
	}

	return result
}
