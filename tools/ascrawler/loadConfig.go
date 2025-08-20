package ascrawler

type loadConfig struct {
	history []string
}

func parseLoadConfig(options ...any) loadConfig {

	result := loadConfig{
		history: make([]string, 0),
	}

	for _, option := range options {
		if loadOption, isLoadOption := option.(LoadOption); isLoadOption {
			loadOption(&result)
		}
	}

	return result
}
