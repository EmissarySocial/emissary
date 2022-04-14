package render

import (
	"fmt"
	"html/template"
	"time"

	"github.com/benpate/convert"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/html"
	"github.com/benpate/steranko"
	"github.com/whisperverse/whisperverse/model"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Common provides common rendering functions that are needed by ALL renderers
type Common struct {
	f        Factory           // Factory interface is required for locating other services.
	ctx      *steranko.Context // Contains request context and authentication data.
	action   *model.Action     // Action to be performed on the (template or layout)
	actionID string            // Token that identifies the action requested in the URL

	// Cached values, do not populate unless needed
	user   *model.User
	domain *model.Domain
}

func NewCommon(factory Factory, ctx *steranko.Context, action *model.Action, actionID string) Common {
	return Common{
		f:        factory,
		ctx:      ctx,
		action:   action,
		actionID: actionID,
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

func (w Common) ActionID() string {
	return w.actionID
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

func (w Common) Host() string {
	return w.Protocol() + w.Hostname()
}

// Protocol returns http:// or https:// used for this request
func (w Common) Protocol() string {
	if w.ctx.Request().TLS == nil {
		return "http://"
	}

	return "https://"
}

// Hostname returns the configured hostname for this request
func (w Common) Hostname() string {
	return w.ctx.Request().Host
}

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

	if context := w.context(); context != nil {
		if request := context.Request(); request != nil {
			if header := request.Header; header != nil {
				return header.Get("HX-Request") != ""
			} else {
				fmt.Println("header is nil")
			}
		} else {
			fmt.Println("request is nil")
		}
	} else {
		fmt.Println("context is nil")
	}
	return false
}

// SkipFullPageRendering returns TRUE if this request does not use the common site chrome.
// Default is FALSE, overridden in specific cases.
func (w Common) SkipFullPageRendering() bool {
	return false
}

// Now returns the current time in milliseconds since the Unix epoch
func (w Common) Now() int64 {
	return time.Now().UnixMilli()
}

/***************************
 * ACCESS PERMISSIONS
 **************************/

// IsAuthenticated returns TRUE if the user is signed in
func (w Common) IsAuthenticated() bool {
	authorization := w.authorization()
	return authorization.IsAuthenticated()
}

// IsOwner returns TRUE if the user is a Domain Owner
func (w Common) IsOwner() bool {
	authorization := w.authorization()
	return authorization.DomainOwner
}

// UserID returns the unique ID of the currently logged in user (may be nil).
func (w Common) UserID() primitive.ObjectID {
	authorization := w.authorization()
	return authorization.UserID
}

// UserName returns the DisplayName of the user
func (w Common) UserName() (string, error) {
	user, err := w.getUser()

	if err != nil {
		return "", derp.Report(derp.Wrap(err, "render.Stream.UserName", "Error loading User"))
	}

	return user.DisplayName, nil
}

func (w Common) Avatar(url string, size int) template.HTML {
	b := html.New()
	b.Empty("img").Attr("src", url).Style("width:"+convert.String(size)+"px", "border-radius:"+convert.String(size)+"px").Close()
	return template.HTML(b.String())
}

// UserAvatar returns the avatar image of the user
func (w Common) UserImage() (string, error) {
	user, err := w.getUser()

	if err != nil {
		return "", derp.Report(derp.Wrap(err, "render.Stream.UserAvatar", "Error loading User"))
	}

	return user.AvatarURL, nil
}

func (w Common) authorization() model.Authorization {
	return getAuthorization(w.ctx)
}

/*******************************************
 * MISC HELPER FUNCTIONS
 *******************************************/

// withViewPermission augments a query criteria to include the
// group authorizations of the currently signed in user.
func (w Common) withViewPermission(criteria exp.Expression) exp.Expression {

	result := criteria.
		And(exp.Equal("journal.deleteDate", 0)).                 // Stream must not be deleted
		And(exp.LessThan("publishDate", time.Now().UnixMilli())) // Stream must be published

	// If the user IS NOT a domain owner, then we must also
	// check their permission to VIEW this stream
	authorization := w.authorization()

	if !authorization.DomainOwner {
		result = result.And(exp.In("defaultAllow", authorization.AllGroupIDs()))
	}

	return result
}

// getUser loads/caches the currently-signed-in user to be used by other functions in this renderer
func (w *Common) getUser() (*model.User, error) {

	// If we haven't already loaded the user, then do it now.
	if w.user == nil {

		userService := w.factory().User()
		authorization := getAuthorization(w.context())
		w.user = new(model.User)

		if err := userService.LoadByID(authorization.UserID, w.user); err != nil {
			return nil, derp.Wrap(err, "render.Stream.getUser", "Error loading user from database", authorization.UserID)
		}
	}

	return w.user, nil
}

// getDomain loads/caches the domain record to be used by other functions in this renderer
func (w *Common) getDomain() (*model.Domain, error) {

	// If we haven't already loaded the domain, then do it now.
	if w.domain == nil {

		domainService := w.factory().Domain()
		authorization := getAuthorization(w.context())

		domain := model.NewDomain()
		w.domain = &domain

		if err := domainService.Load(w.domain); err != nil {

			// Only "legitimate" errors are reported. "Not Found" is skipped.
			if !derp.NotFound(err) {
				return nil, derp.Wrap(err, "render.Stream.getUser", "Error loading domain from database", authorization.UserID)
			}
		}
	}

	return w.domain, nil
}

/*******************************************
 * GLOBAL QUERIES
 *******************************************/

// TopLevel returns an array of Streams that have a Zero ParentID
func (w Common) TopLevel() (List, error) {
	criteria := w.withViewPermission(exp.Equal("parentId", primitive.NilObjectID))
	builder := NewQueryBuilder(w.factory(), w.context(), w.factory().Stream(), criteria)
	return builder.Top60().ByRank().View()
}
