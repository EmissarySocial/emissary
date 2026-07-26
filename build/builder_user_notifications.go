package build

import (
	"bytes"
	"html/template"
	"net/http"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	builder "github.com/benpate/exp-builder"
	"github.com/benpate/rosetta/schema"
	"github.com/benpate/rosetta/sliceof"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Notifications is a builder for the @me/notifications pages (Replies, Shares, Likes,
// Followers).  It builds the authenticated User object, and its only data method is
// Notifications() (plus the UnreadNotificationCount badge helper) — the newsfeed/folder
// machinery lives on the Inbox builder, not here.
type Notifications struct {
	_user *model.User
	CommonWithTemplate
}

// NewNotifications returns a fully initialized `Notifications` builder
func NewNotifications(factory Factory, session data.Session, request *http.Request, response http.ResponseWriter, user *model.User, actionID string) (Notifications, error) {

	const location = "build.NewNotifications"

	// Load the Template
	templateService := factory.Template()
	template, err := templateService.Load("user-notifications")

	if err != nil {
		return Notifications{}, derp.Wrap(err, location, "Loading template")
	}

	// Create the underlying Common builder
	common, err := NewCommonWithTemplate(factory, session, request, response, template, user, actionID)

	if err != nil {
		return Notifications{}, derp.Wrap(err, location, "Creating common builder")
	}

	// Enforce user permissions on the requested action
	if !common.UserCan(actionID) {
		if common._authorization.IsAuthenticated() {
			return Notifications{}, derp.Forbidden(location, "Forbidden", "User is authenticated, but this action is not allowed", actionID)
		} else {
			return Notifications{}, derp.Unauthorized(location, "Anonymous user is not authorized to perform this action", user.ProfileURL, actionID)
		}
	}

	return Notifications{
		_user:              user,
		CommonWithTemplate: common,
	}, nil
}

/******************************************
 * Renderer Interface
 ******************************************/

// Render generates the string value for this Notifications
func (w Notifications) Render() (template.HTML, error) {

	var buffer bytes.Buffer

	// Execute step (write HTML to buffer, update context)
	status := Pipeline(w._action.Steps).Get(w._factory, &w, &buffer)

	if status.Error != nil {
		return "", derp.Wrap(status.Error, "build.Notifications.Render", "Generating HTML", w._request.URL.String())
	}

	// Success!
	status.Apply(w._response)
	return template.HTML(buffer.String()), nil
}

// View executes a separate view for this Notifications
func (w Notifications) View(actionID string) (template.HTML, error) {

	builder, err := NewNotifications(w._factory, w._session, w._request, w._response, w._user, actionID)

	if err != nil {
		return template.HTML(""), derp.Wrap(err, "build.Notifications.View", "Creating Notifications builder")
	}

	return builder.Render()
}

// NavigationID returns the ID to use for highlighing navigation menus
func (w Notifications) NavigationID() string {
	return "notifications"
}

func (w Notifications) PageTitle() string {
	return w._user.DisplayName
}

func (w Notifications) BasePath() string {
	return "/@me/notifications"
}

func (w Notifications) Permalink() string {
	return w.Host() + "/@me/notifications"
}

func (w Notifications) Token() string {
	return "notifications"
}

func (w Notifications) object() data.Object {
	return w._user
}

func (w Notifications) objectID() primitive.ObjectID {
	return w._user.UserID
}

func (w Notifications) objectType() string {
	return "User"
}

func (w Notifications) schema() schema.Schema {
	return schema.New(model.UserSchema())
}

func (w Notifications) service() service.ModelService {
	return w._factory.User()
}

func (w Notifications) templateRole() string {
	return "notifications"
}

func (w Notifications) clone(action string) (Builder, error) {
	return NewNotifications(w._factory, w._session, w._request, w._response, w._user, action)
}

/******************************************
 * Notifications Methods
 ******************************************/

// Notifications returns a QueryBuilder for the current User's notifications (mentions, replies,
// likes, follows).  Results are always scoped to the authenticated User.  The optional `type`
// query param filters by notification type (the filter sections); `createDate` drives paging.
func (w Notifications) Notifications() (QueryBuilder[model.Notification], error) {

	const location = "build.Notifications.Notifications"

	// Must be signed in to view notifications
	if w.AuthenticatedID().IsZero() {
		return QueryBuilder[model.Notification]{}, derp.Unauthorized(location, "Must be signed in to view notifications")
	}

	queryString := w._request.URL.Query()

	expBuilder := builder.NewBuilder().
		Int("readDate").
		Int("createDate")

	criteria := exp.And(
		exp.Equal("userId", w.AuthenticatedID()),
		exp.Equal("deleteDate", 0),
		notificationTypeFilter(queryString.Get("type")),
		expBuilder.Evaluate(queryString),
	)

	return NewQueryBuilder[model.Notification](w._factory.Notification(), w._session, criteria), nil
}

// LabelNotifications stamps each Notification's transient Labels field with the viewer's rule
// verdict for its snapshotted Actor, and returns the same slice for template chaining. Verdicts
// are derived fresh on every render (R8), so a deleted rule stops labeling immediately.
func (w Notifications) LabelNotifications(notifications sliceof.Object[model.Notification]) sliceof.Object[model.Notification] {
	w._factory.Rule().LabelNotifications(w._session, w.AuthenticatedID(), notifications)
	return notifications
}

// UnreadNotificationCount returns the number of unread Notifications in the provided
// notifications-page section (using the same type expansion as the section filter).  Errors are
// reported and render as zero — a count badge must never fail the page.
func (w Notifications) UnreadNotificationCount(notificationType string) int64 {

	if w.AuthenticatedID().IsZero() {
		return 0
	}

	count, err := w._factory.Notification().CountUnread(w._session, w.AuthenticatedID(), notificationTabTypes(notificationType)...)

	if err != nil {
		derp.Report(derp.Wrap(err, "build.Notifications.UnreadNotificationCount", "Counting unread notifications", notificationType))
		return 0
	}

	return count
}

// notificationTabTypes expands the notifications-page "type" query param into the list of
// notification types that section displays.  Grouped sections cast wide nets: MENTION includes
// replies and direct messages (the classification ladder is DIRECT > REPLY > MENTION, so without
// the expansion those would be invisible — there is no Replies section and no Messages section),
// and LIKE includes dislikes (too rare for a section of their own).  ANNOUNCE passes through
// unexpanded as the "Shares" section.  An empty section name returns nil, meaning "all types".
//
// This helper (and notificationTypeFilter below) is package-level because both the Notifications
// builder and the mark-notifications-read / with-notification steps expand `type` the same way.
func notificationTabTypes(notificationType string) []string {

	switch notificationType {

	case "":
		return nil

	case model.NotificationTypeMention:
		return []string{model.NotificationTypeMention, model.NotificationTypeReply, model.NotificationTypeDirect}

	case model.NotificationTypeLike:
		return []string{model.NotificationTypeLike, model.NotificationTypeDislike}
	}

	return []string{notificationType}
}

// notificationTypeFilter expands the notifications-page "type" query param into criteria.
func notificationTypeFilter(notificationType string) exp.Expression {

	types := notificationTabTypes(notificationType)

	if types == nil {
		return exp.All()
	}

	return exp.In("type", types)
}

func (w Notifications) debug() {
	log.Debug().Interface("object", w.object()).Msg("builder_Notifications")
}
