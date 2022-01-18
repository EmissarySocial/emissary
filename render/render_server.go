package render

import "github.com/whisperverse/whisperverse/config"

type Server struct {
	domains config.Config
}

func NewServer(domains config.Config) Server {
	return Server{
		domains: domains,
	}
}
