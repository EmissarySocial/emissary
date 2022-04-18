package step

import (
	"github.com/benpate/convert"
	"github.com/benpate/datatype"
)

// ViewRSS represents an action-step that can render a Stream into HTML
type ViewRSS struct {
	Format string // atom, rss, json (default is rss)
}

// NewViewRSS generates a fully initialized ViewRSS step.
func NewViewRSS(stepInfo datatype.Map) (ViewRSS, error) {

	return ViewRSS{
		Format: convert.String(stepInfo["format"]),
	}, nil
}

// AmStep is here only to verify that this struct is a render pipeline step
func (step ViewRSS) AmStep() {}