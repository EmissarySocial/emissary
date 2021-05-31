package action

import "github.com/benpate/ghost/service"

type CreateStream struct {
	streamService *service.Stream
	Info
}
