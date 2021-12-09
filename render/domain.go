package render

import (
	"github.com/benpate/derp"
	"github.com/benpate/steranko"
)

type Domain struct {
	factory Factory           // Factory interface is required for locating other services.
	ctx     *steranko.Context // Contains request context and authentication data.
}

func NewDomain(factory Factory, ctx *steranko.Context) Domain {
	return Domain{
		factory: factory,
		ctx:     ctx,
	}
}

// Label returns the label for this Domain
func (d *Domain) Label() string {
	return ""
}

// Banner returns the banner placement option for this Domain
func (d *Domain) Banner() string {
	return "none"
}

// BannerImage returns the (optional) banner image for this Domain
func (d *Domain) BannerImage() string {
	return ""
}

// TopLevel returns all of the top level streams for this Domain.
func (d *Domain) TopLevel() ([]Stream, error) {

	streamService := d.factory.Stream()
	iterator, err := streamService.ListTopLevel()

	if err != nil {
		return nil, derp.Wrap(err, "ghost.render.Domain.TopItems", "Error getting TopItems from database")
	}

	return streamIteratorToSlice(d.factory, d.ctx, iterator, 0, "view"), nil
}
