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

// Notifications is a builder for the @me/notifications pages (All, Replies, Quotes, Shares,
// Likes, Followers).  It builds the authenticated User object, and its only data method is
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
	return template.HTML(buffer.String()), nil // #nosec G203 -- buffer holds the action pipeline's own rendered output, already escaped by html/template
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

// PageTitle returns the human-friendly title to display at the top of the page. Implements the Builder interface.
func (w Notifications) PageTitle() string {
	return w._user.DisplayName
}

// BasePath returns the URL path of this object, without any action. Implements the Builder interface.
func (w Notifications) BasePath() string {
	return "/@me/notifications"
}

// Permalink returns the permanent URL of the record being built. Implements the Builder interface.
func (w Notifications) Permalink() string {
	return w.Host() + "/@me/notifications"
}

// Token returns the URL token of the record being built. Implements the Builder interface.
func (w Notifications) Token() string {
	return "notifications"
}

// object returns the model object being built. Implements the Builder interface.
func (w Notifications) object() data.Object {
	return w._user
}

// objectID returns the unique ID of the object being built. Implements the Builder interface.
func (w Notifications) objectID() primitive.ObjectID {
	return w._user.UserID
}

// objectType returns the name of the model type being built. Implements the Builder interface.
func (w Notifications) objectType() string {
	return "User"
}

// schema returns the rosetta schema that validates this object. Implements the Builder interface.
func (w Notifications) schema() schema.Schema {
	return schema.New(model.UserSchema())
}

// service returns the ModelService that backs this Builder. Implements the Builder interface.
func (w Notifications) service() service.ModelService {
	return w._factory.User()
}

// templateRole returns the role this Template plays, which chooses the child template. Implements the Builder interface.
func (w Notifications) templateRole() string {
	return "notifications"
}

// clone returns a copy of this Builder, pointed at a different action. Implements the Builder interface.
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

// notificationTypeFilter expands the notifications-page "type" query param into criteria.
func notificationTypeFilter(notificationType string) exp.Expression {

	types := notificationTabTypes(notificationType)

	if types == nil {
		return exp.All()
	}

	return exp.In("type", types)
}

// LabelNotifications stamps each Notification's transient Labels field with the viewer's rule
// verdict for its snapshotted Actor, and returns the same slice for template chaining. Verdicts
// are derived fresh on every render (R8), so a deleted rule stops labeling immediately.
func (w Notifications) LabelNotifications(notifications sliceof.Object[model.Notification]) sliceof.Object[model.Notification] {
	w._factory.Rule().LabelNotifications(w._session, w.AuthenticatedID(), notifications)
	return notifications
}

// Notification returns the Notification named by the "notificationId" query parameter, which
// must belong to the signed-in User
func (w Notifications) Notification() (model.Notification, error) {

	const location = "build.Notifications.Notification"

	result := model.NewNotification()

	notificationID, err := primitive.ObjectIDFromHex(w.QueryParam("notificationId"))

	if err != nil {
		return result, derp.BadRequest(location, "NotificationID must be a valid hex string")
	}

	// RULE: LoadByID is scoped to the signed-in User, so another User's Notification is NotFound
	if err := w._factory.Notification().LoadByID(w._session, w.AuthenticatedID(), notificationID, &result); err != nil {
		return result, derp.Wrap(err, location, "Loading Notification", notificationID)
	}

	return result, nil
}

// QuoteItemID returns the ID of the CollectionItem that approves the quote a QUOTE Notification
// announces, or an empty string when that quote is no longer approved.
func (w Notifications) QuoteItemID(notification model.Notification) string {

	const location = "build.Notifications.QuoteItemID"

	collectionItem := model.NewCollectionItem()

	if err := w.factory().Stream().LoadQuoteByURI(w.session(), notification.StreamID, notification.ObjectURL, &collectionItem); err != nil {

		// A quote that was rejected, or whose Stream is gone, has no item, which is not a failure
		if !derp.IsNotFound(err) {
			derp.Report(derp.Wrap(err, location, "Loading quote", notification.NotificationID))
		}

		return ""
	}

	return collectionItem.CollectionItemID.Hex()
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

// notificationTabTypes expands the notifications-page "type" query param into the notification
// types that tab lists. A nil result means every type, which is the All tab and an empty param.
func notificationTabTypes(notificationType string) []string {

	switch notificationType {

	case "", notificationTabAll:
		return nil

	// RULE: Messages & Replies holds direct messages and replies too, which have no tab of their own
	case model.NotificationTypeMention:
		return []string{model.NotificationTypeMention, model.NotificationTypeReply, model.NotificationTypeDirect}

	// RULE: Likes holds dislikes too, which are too rare for a tab of their own
	case model.NotificationTypeLike:
		return []string{model.NotificationTypeLike, model.NotificationTypeDislike}
	}

	// Quotes (QUOTE), Boosts (ANNOUNCE) and Followers (FOLLOW) each list their own type
	return []string{notificationType}
}

// debug writes this Builder's state to the console. Implements the Builder interface.
func (w Notifications) debug() {
	log.Debug().Interface("object", w.object()).Msg("builder_Notifications")
}
