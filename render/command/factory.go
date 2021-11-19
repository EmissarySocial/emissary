package command

import (
	"github.com/benpate/form"
	"github.com/benpate/ghost/service"
)

// Factory interface wraps the functions required to create Renderers and Actions.
// It is used to pass the domain.Factory internally.
type Factory interface {
	Stream() *service.Stream
	StreamDraft() *service.StreamDraft
	Template() *service.Template
	FormLibrary() form.Library
}
