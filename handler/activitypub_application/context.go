package activitypub_application

import (
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
)

// Context includes all of the objects needed to handle an @application inbox request
type Context struct {
	factory *service.Factory
	session data.Session
}
