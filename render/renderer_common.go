package render

import (
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/ghost/model"
	"github.com/benpate/steranko"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Common provides common rendering functions that are needed by ALL renderers
type Common struct {
	f   Factory           // Factory interface is required for locating other services.
	ctx *steranko.Context // Contains request context and authentication data.

	// Cached values, do not populate unless needed
	user   *model.User
	domain *model.Domain
}

func NewCommon(factory Factory, ctx *steranko.Context) Common {
	return Common{
		f:   factory,
		ctx: ctx,
	}
}

/*******************************************
 * RENDERER INTERFACE
 *******************************************/

// context returns request context embedded in this renderer.
func (w Common) factory() Factory {
	return w.f
}

// context returns request context embedded in this renderer.
func (w Common) context() *steranko.Context {
	return w.ctx
}

func (w Common) DomainLabel() string {
	if domain, err := w.getDomain(); err == nil {
		return domain.Label
	}
	return ""
}

func (w Common) BannerURL() string {
	if domain, err := w.getDomain(); err == nil {
		return domain.BannerURL
	}
	return ""
}

/*******************************************
 * REQUEST INFO
 *******************************************/

// URL returns the originally requested URL
func (w Common) URL() string {
	return w.context().Request().URL.RequestURI()
}

// Returns the request method
func (w Common) Method() string {
	return w.context().Request().Method
}

// Returns the designated request parameter
func (w Common) QueryParam(param string) string {
	return w.context().QueryParam(param)
}

// IsPartialRequest returns TRUE if this is a partial page request from htmx.
func (w Common) IsPartialRequest() bool {
	return (w.context().Request().Header.Get("HX-Request") != "")
}

/***************************
 * ACCESS PERMISSIONS
 **************************/

// IsAuthenticated returns TRUE if the user is signed in
func (w Common) IsAuthenticated() bool {
	return getAuthorization(w.context()).IsAuthenticated()
}

// IsOwner returns TRUE if the user is a Domain Owner
func (w Common) IsOwner() bool {
	authorization := getAuthorization(w.context())
	return authorization.DomainOwner
}

// UserName returns the DisplayName of the user
func (w Common) UserName() (string, error) {
	user, err := w.getUser()

	if err != nil {
		return "", derp.Report(derp.Wrap(err, "ghost.render.Stream.UserName", "Error loading User"))
	}

	return user.DisplayName, nil
}

// UserAvatar returns the avatar image of the user
func (w Common) UserAvatar() (string, error) {
	user, err := w.getUser()

	if err != nil {
		return "", derp.Report(derp.Wrap(err, "ghost.render.Stream.UserAvatar", "Error loading User"))
	}

	return user.AvatarURL, nil
}

/*******************************************
 * MISC HELPER FUNCTIONS
 *******************************************/

// getUser loads/caches the currently-signed-in user to be used by other functions in this renderer
func (w Common) getUser() (*model.User, error) {

	// If we haven't already loaded the user, then do it now.
	if w.user == nil {

		userService := w.factory().User()
		authorization := getAuthorization(w.context())
		w.user = new(model.User)

		if err := userService.LoadByID(authorization.UserID, w.user); err != nil {
			return nil, derp.Wrap(err, "ghost.render.Stream.getUser", "Error loading user from database", authorization.UserID)
		}
	}

	return w.user, nil
}

// getDomain loads/caches the domain record to be used by other functions in this renderer
func (w Common) getDomain() (*model.Domain, error) {

	// If we haven't already loaded the domain, then do it now.
	if w.domain == nil {

		domainService := w.factory().Domain()
		authorization := getAuthorization(w.context())
		w.domain = new(model.Domain)

		if err := domainService.Load(w.domain); err != nil {
			return nil, derp.Wrap(err, "ghost.render.Stream.getUser", "Error loading domain from database", authorization.UserID)
		}
	}

	return w.domain, nil
}

/*******************************************
 * GLOBAL QUERIES
 *******************************************/

// TopLevel returns an array of Streams that have a Zero ParentID
func (w Common) TopLevel() (List, error) {
	criteria := exp.Equal("parentId", primitive.NilObjectID)
	builder := NewQueryBuilder(w.factory(), w.context(), w.factory().Stream(), criteria)
	return builder.Top60().ByRank().View()
}
