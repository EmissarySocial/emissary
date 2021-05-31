package action

import "github.com/benpate/ghost/service"

// UpdateContent manages the content.Content in a stream.
type UpdateContent struct {
	streamService *service.Stream
	Info
}
