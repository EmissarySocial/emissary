package ascrawler

type loadConfig struct {
	history    []string
	useCrawler bool
}

func parseLoadConfig(options ...any) loadConfig {

	result := loadConfig{
		history:    make([]string, 0),
		useCrawler: true,
	}

	for _, option := range options {
		if loadOption, isLoadOption := option.(LoadOption); isLoadOption {
			loadOption(&result)
		}
	}

	return result
}
