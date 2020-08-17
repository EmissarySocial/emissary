package render

import "github.com/benpate/ghost/service"

type Global struct {
	factory service.Factory
}

func (g Global) Label() string {
	return "Site Name Here"
}

func (g Global) TopStreams() ([]StreamWrapper, error) {

	var result []StreamWrapper

	// 	streamService := g.factory.StreamWrapper()

	return result, nil
}
