package build

import (
	"net/http"

	"github.com/benpate/data"
)

// Theme is a Builder for the Theme-level pages that a handler renders directly, without a
// Template action: sign-in, sign-out, guest sign-in, password reset, and checkout claim.
type Theme struct {
	Common
}

// NewTheme returns a fully initialized Theme builder.
func NewTheme(factory Factory, session data.Session, request *http.Request, response http.ResponseWriter) Theme {

	// Everything these pages display already lives on Common.  Building the values by
	// hand instead is how this package ended up with `.DomainName`, `.domainName`, and
	// `.Label` all meaning Common.DomainLabel().
	return Theme{
		Common: NewCommon(factory, session, request, response),
	}
}

// IsIndexable returns FALSE so that these pages are never indexed by search engines.
func (w Theme) IsIndexable() bool {

	// RULE: Common defaults to TRUE, so this override is what keeps every authentication
	// and transaction page out of the index.  Builders that embed Theme inherit it.
	return false
}
