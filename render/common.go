package render

import (
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/ghost/model"
	"github.com/benpate/steranko"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Common provides common rendering functions that are needed by ALL renderers
type Common struct {
	factory Factory           // Factory interface is required for locating other services.
	ctx     *steranko.Context // Contains request context and authentication data.

	// Cached values, do not populate unless needed
	user *model.User
}

func NewCommon(factory Factory, ctx *steranko.Context) Common {
	return Common{
		factory: factory,
		ctx:     ctx,
	}
}

/*******************************************
 * REQUEST INFO
 *******************************************/

// URL returns the originally requested URL
func (w Common) URL() string {
	return w.ctx.Request().URL.RequestURI()
}

// Returns the request method
func (w Common) Method() string {
	return w.ctx.Request().Method
}

// Returns the designated request parameter
func (w Common) QueryParam(param string) string {
	return w.ctx.QueryParam(param)
}

/*/ SetQueryParam sets/overwrites a value from the URL query parameters.
func (w Common) SetQueryParam(param string, value string) string {
	w.ctx.QueryParams().Set(param, value)
	return "" // <- this is a mega-hack, but it works ;)
}*/

// IsPartialRequest returns TRUE if this is a partial page request from htmx.
func (w Common) IsPartialRequest() bool {
	return (w.ctx.Request().Header.Get("HX-Request") != "")
}

/***************************
 * ACCESS PERMISSIONS
 **************************/

// IsAuthenticated returns TRUE if the user is signed in
func (w Common) IsAuthenticated() bool {
	return getAuthorization(w.ctx).IsAuthenticated()
}

// IsOwner returns TRUE if the user is a Domain Owner
func (w Common) IsOwner() bool {
	authorization := getAuthorization(w.ctx)
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

// newStream is a shortcut to the NewStream function that reuses the values present in this renderer
func (w Common) newStream(stream *model.Stream, actionID string) (Stream, error) {
	return NewStream(w.factory, w.ctx, stream, actionID)
}

// closeModal sets Response header to close a modal on the client and optionally forward to a new location.
func (w Common) closeModal(url string) {
	if url == "" {
		w.ctx.Response().Header().Set("HX-Trigger", `"closeModal"`)
	} else {
		w.ctx.Response().Header().Set("HX-Trigger", `{"closeModal":{"nextPage":"`+url+`"}}`)
	}
}

// getUser loads/caches the currently-signed-in user to be used by other functions in this renderer
func (w Common) getUser() (*model.User, error) {

	// If we haven't already loaded the user, then do it now.
	if w.user == nil {

		userService := w.factory.User()
		authorization := getAuthorization(w.ctx)
		w.user = new(model.User)

		if err := userService.LoadByID(authorization.UserID, w.user); err != nil {
			return nil, derp.Wrap(err, "ghost.render.Stream.getUser", "Error loading user from database", authorization.UserID)
		}
	}

	return w.user, nil
}

/*******************************************
 * GLOBAL QUERIES
 *******************************************/

// TopLevel returns an array of Streams that have a Zero ParentID
func (w Common) TopLevel() ([]Stream, error) {
	criteria := exp.Equal("parentId", primitive.NilObjectID)
	resultSet := NewResultSet(w.factory, w.ctx, criteria)
	resultSet.SortField = "rank"
	resultSet.SortDirection = option.SortDirectionAscending
	resultSet.MaxRows = 10
	return resultSet.View()
}
