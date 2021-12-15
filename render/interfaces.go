package render

import (
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/exp"
	"github.com/benpate/form"
	"github.com/benpate/ghost/service"
	"github.com/benpate/mediaserver"
)

// Factory is used to locate all necessary services
type Factory interface {
	Attachment() *service.Attachment
	Domain() *service.Domain
	FormLibrary() form.Library
	Group() *service.Group
	Layout() *service.Layout
	MediaServer() mediaserver.MediaServer
	Stream() *service.Stream
	StreamDraft() *service.StreamDraft
	Template() *service.Template
	User() *service.User
}

type ModelService interface {
	ObjectNew() data.Object
	ObjectList(exp.Expression, ...option.Option) (data.Iterator, error)
	ObjectLoad(exp.Expression) (data.Object, error)
	ObjectSave(data.Object, string) error
	ObjectDelete(data.Object, string) error
}
