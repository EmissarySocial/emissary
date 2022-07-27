package render

import (
	"fmt"
	"html/template"
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/form"
	"github.com/benpate/html"
	"github.com/benpate/rosetta/convert"
	"github.com/benpate/rosetta/maps"
	"github.com/benpate/steranko"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Common provides common rendering functions that are needed by ALL renderers
type Common struct {
	f        Factory           // Factory interface is required for locating other services.
	ctx      *steranko.Context // Contains request context and authentication data.
	action   *model.Action     // Action to be performed on the (template or layout)
	actionID string            // Token that identifies the action requested in the URL

	requestData maps.Map // Temporary data scope for this request

	// Cached values, do not populate unless needed
	domain model.Domain // This is a value because we expect to use it in every request.
	user   *model.User  // This is a pointer because it may not be used in every request.
}

func NewCommon(factory Factory, ctx *steranko.Context, action *model.Action, actionID string) Common {
	return Common{
		f:           factory,
		ctx:         ctx,
		action:      action,
		actionID:    actionID,
		requestData: maps.New(),
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

// Action returns the model.Action configured into this renderer
func (w Common) Action() *model.Action {
	return w.action
}

func (w Common) ActionID() string {
	return w.actionID
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

// Host returns the protocol + the Hostname
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

// UseGlobalWrapper returns TRUE if every step in this request uses the common site chrome.
// It returns FALSE if any of its sub-steps must not use the common wrapper.
func (w Common) UseGlobalWrapper() bool {

	// Nil check just in case
	if w.action == nil {
		return true
	}

	return useGlobalWrapper(w.action.Steps)
}

// Now returns the current time in milliseconds since the Unix epoch
func (w Common) Now() int64 {
	return time.Now().UnixMilli()
}

/***************************
 * REQUEST DATA (Getters and Setters)
 **************************/

func (w Common) GetBool(name string) bool {
	return w.requestData.GetBool(name)
}

func (w Common) GetFloat(name string) float64 {
	return w.requestData.GetFloat(name)
}

func (w Common) GetInt(name string) int {
	return w.requestData.GetInt(name)
}

func (w Common) GetInt64(name string) int64 {
	return w.requestData.GetInt64(name)
}

func (w Common) GetString(name string) string {
	return w.requestData.GetString(name)
}

func (w Common) SetBool(name string, value bool) {
	w.requestData.SetBool(name, value)
}

func (w Common) SetFloat(name string, value float64) {
	w.requestData.SetFloat(name, value)
}

func (w Common) SetInt(name string, value int) {
	w.requestData.SetInt(name, value)
}

func (w Common) SetInt64(name string, value int64) {
	w.requestData.SetInt64(name, value)
}

func (w Common) SetString(name string, value string) {
	w.requestData.SetString(name, value)
}

/*******************************************
 * DOMAIN DATA
 *******************************************/

func (w Common) DomainLabel() (string, error) {
	if domain, err := w.getDomain(); err != nil {
		return "", err
	} else {
		return domain.Label, nil
	}
}

func (w Common) DomainHeaderHTML() (string, error) {
	if domain, err := w.getDomain(); err != nil {
		return "", err
	} else {
		return domain.HeaderHTML, nil
	}
}

func (w Common) DomainFooterHTML() (string, error) {
	if domain, err := w.getDomain(); err != nil {
		return "", err
	} else {
		return domain.FooterHTML, nil
	}
}

func (w Common) DomainCustomCSS() (string, error) {
	if domain, err := w.getDomain(); err != nil {
		return "", err
	} else {
		return domain.CustomCSS, nil
	}
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

	return user.ImageURL, nil
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

		w.user = new(model.User)
		userService := w.factory().User()
		authorization := getAuthorization(w.context())

		if err := userService.LoadByID(authorization.UserID, w.user); err != nil {
			return nil, derp.Wrap(err, "render.Common.getUser", "Error loading user from database", authorization.UserID)
		}
	}

	return w.user, nil
}

// getDomain retrieves the current domain model object from the domain service cache
func (w *Common) getDomain() (*model.Domain, error) {

	if w.domain.DomainID.IsZero() {
		domainService := w.factory().Domain()

		if err := domainService.Load(&w.domain); err != nil {
			return nil, derp.Wrap(err, "render.Common.getDomain", "Error loading domain")
		}
	}

	return &w.domain, nil
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

/*******************************************
 * ADDITIONAL DATA
 *******************************************/

// AdminSections returns labels and values for all hard-coded sections of the administrator area.
func (w Common) AdminSections() []form.LookupCode {
	return []form.LookupCode{
		{
			Value: "domain",
			Label: "Site",
		},
		{
			Value: "appearance",
			Label: "Appearance",
		},
		{
			Value: "toplevel",
			Label: "Navigation",
		},
		{
			Value: "users",
			Label: "People",
		},
		{
			Value: "groups",
			Label: "Groups",
		},
		{
			Value: "connections",
			Label: "Connections",
		},
	}
}
