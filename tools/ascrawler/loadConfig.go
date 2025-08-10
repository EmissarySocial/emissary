package ascrawler

type loadConfig struct {
	currentDepth int
}

func parseLoadConfig(options ...any) loadConfig {

	result := loadConfig{
		currentDepth: 0,
	}

	for _, option := range options {
		if loadOption, isLoadOption := option.(LoadOption); isLoadOption {
			loadOption(&result)
		}
	}

	return result
}
