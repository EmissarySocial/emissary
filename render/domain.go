package render

import (
	"github.com/benpate/ghost/model"
	"github.com/benpate/steranko"
)

// Domain renderer wraps all of the methods needed to render the domain
type Domain struct {
	domain *model.Domain
	Common
}

func NewDomain(factory Factory, ctx *steranko.Context, domain *model.Domain) Domain {
	return Domain{
		domain: domain,
		Common: NewCommon(factory, ctx),
	}
}

// Label returns the label for this Domain
func (d *Domain) Label() string {
	return d.domain.Label
}

// BannerURL returns the banner image for this Domain
func (d *Domain) BannerURL() string {
	return d.domain.BannerURL
}
