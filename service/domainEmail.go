package service

import (
	"maps"
	"strconv"
	"time"

	"github.com/EmissarySocial/emissary/config"
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/first"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/steranko"
	"github.com/benpate/uri"
)

// parentActorLoader returns the User or Stream that a Follower follows
type parentActorLoader interface {
	LoadParentActor(session data.Session, follower *model.Follower) (model.PersonLink, error)
}

// DomainEmail sends the transactional emails that a single Domain generates
type DomainEmail struct {
	serverEmail   *ServerEmail
	domainService *Domain
	parentActors  parentActorLoader
	smtp          config.SMTPConnection
	owner         config.Owner
	label         string
	hostname      string
	newSteranko   func(session data.Session) *steranko.Steranko
}

// NewDomainEmail returns an empty DomainEmail service, which Refresh populates
func NewDomainEmail() DomainEmail {
	return DomainEmail{}
}

/******************************************
 * Lifecycle Methods
 ******************************************/

// Refresh updates this service with the latest configuration values
func (service *DomainEmail) Refresh(factory *Factory) {
	service.serverEmail = factory.ServerEmail()
	service.domainService = factory.Domain()
	service.parentActors = factory.Follower()
	service.smtp = factory.config.SMTPConnection
	service.owner = factory.config.Owner
	service.label = factory.config.Label
	service.hostname = factory.config.Hostname
	service.newSteranko = factory.Steranko
}

/******************************************
 * Send API
 ******************************************/

// sendModel delivers the named email, but only if it is defined for modelName.  Each sender below
// builds a fixed data shape for a fixed email, so a definition that an administrator overrode on
// disk for some other object would render nothing but missing values.  Emails named by a Template
// need no such check: their data comes from the same file that names the email.
func (service *DomainEmail) sendModel(emailID string, modelName string, data mapof.Any) error {

	const location = "service.DomainEmail.sendModel"

	if err := service.serverEmail.RequireModel(emailID, modelName); err != nil {
		return derp.Wrap(err, location, "Email is not defined for this model object", emailID, modelName)
	}

	return service.Send(emailID, data)
}

// Send renders the named email and delivers it over this Domain's SMTP connection
func (service *DomainEmail) Send(emailID string, data mapof.Any) error {

	const location = "service.DomainEmail.Send"

	// RULE: an unconfigured SMTP connection is a failure, not a silent success.  Returning nil
	// here would let a caller -- a contact form, say -- accept a message, deliver nothing, and
	// leave no record that it happened.  Callers that must survive this check IsConfigured first.
	if !service.IsConfigured() {
		return derp.Internal(location, "Cannot send email because SMTP is not configured for this domain", emailID)
	}

	// Copy the caller's data so that a caller reusing its map does not accumulate our additions
	values := make(mapof.Any, len(data)+4)
	maps.Copy(values, data)

	// Add the Domain values that every email template may reference
	readOnlyDomain := service.domainService.Cached()

	values["Domain_Owner"] = service.owner
	values["Domain_URL"] = service.host()
	values["Domain_Name"] = readOnlyDomain.Label
	values["Domain_Icon"] = readOnlyDomain.IconURL()

	if err := service.serverEmail.Send(service.smtp, service.owner, emailID, values); err != nil {
		return derp.Wrap(err, location, "Sending email", emailID)
	}

	// Neither snow nor rain nor heat nor gloom of night
	return nil
}

// IsConfigured returns TRUE if this Domain has an SMTP connection that Send can use
func (service *DomainEmail) IsConfigured() bool {
	return !service.smtp.IsNil()
}

/******************************************
 * Email Templates
 ******************************************/

// SendWelcome sends a welcome email to the user.  This method
// returns an error so that it CAN NOT be run asynchronously.
func (service *DomainEmail) SendWelcome(session data.Session, txn model.RegistrationTxn) error {

	const location = "service.DomainEmail.SendWelcome"

	// Create a JWT with the registration information, and populate it into the Token
	sterankoService := service.newSteranko(session)
	token, err := sterankoService.CreateJWT(txn.Claims())

	if err != nil {
		return derp.Wrap(err, location, "Creating JWT")
	}

	// Send the welcome email
	if err := service.sendModel(
		"user-welcome",
		"User",
		mapof.Any{
			// User info available to the template
			"Username": txn.Username,
			"Name":     txn.DisplayName,
			"Email":    txn.EmailAddress,
			"Token":    token,
		},
	); err != nil {
		return derp.Wrap(err, location, "Sending welcome email to user", txn.EmailAddress)
	}

	// Woot!
	return nil
}

// SendPasswordReset sends a passowrd reset email to the user.  This method
// swallows errors so that it can be run asynchronously.
func (service *DomainEmail) SendPasswordReset(user *model.User) error {

	const location = "service.DomainEmail.SendPasswordReset"

	// Send the password reset email
	if err := service.sendModel(
		"user-password-reset",
		"User",
		mapof.Any{
			// User info available to the template
			"UserID":     user.UserID.Hex(),
			"Username":   user.Username,
			"Name":       user.DisplayName,
			"Email":      user.EmailAddress,
			"ResetCode":  user.PasswordReset.AuthCode,
			"ExpireDate": user.PasswordReset.ExpireDate,
		},
	); err != nil {
		return derp.Wrap(err, location, "Sending password reset email to user", user.Username)
	}

	return nil
}

// SendPasswordResetConfirmation sends a password reset confirmation email to the user.  This method
// swallows errors so that it can be run asynchronously.
func (service *DomainEmail) SendPasswordResetConfirmation(session data.Session, user *model.User) error {

	const location = "service.DomainEmail.SendPasswordResetConfirmation"

	// Send the password reset confirmation email
	if err := service.sendModel(
		"user-password-reset-confirmation",
		"User",
		mapof.Any{
			// User info available to the template
			"UserID":     user.UserID.Hex(),
			"Username":   user.Username,
			"Name":       user.DisplayName,
			"Email":      user.EmailAddress,
			"ResetCode":  user.PasswordReset.AuthCode,
			"ExpireDate": user.PasswordReset.ExpireDate,
		},
	); err != nil {
		return derp.Wrap(err, location, "Sending password reset confirmation email to user", user.Username)
	}

	return nil
}

// SendUserLockout sends a lockout notification email to the user.  This method
// swallows errors so that it can be run asynchronously.
func (service *DomainEmail) SendUserLockout(session data.Session, user *model.User) error {

	const location = "service.DomainEmail.SendUserLockout"

	// Send the lockout email
	if err := service.sendModel(
		"user-lockout",
		"User",
		mapof.Any{
			// User info available to the template.
			// NOTE: no ResetCode/ExpireDate -- a lockout no longer resets the password.
			"UserID":   user.UserID.Hex(),
			"Username": user.Username,
			"Name":     user.DisplayName,
			"Email":    user.EmailAddress,
		},
	); err != nil {
		return derp.Wrap(err, location, "Sending user lockout email to user", user.Username)
	}

	return nil
}

// SendGuestCode sends JWT signin code to the provided email address, which will
// sign their "Identity" into the system
func (service *DomainEmail) SendGuestCode(identifier string, token string) error {

	const location = "service.DomainEmail.SendGuestCode"

	// Send the welcome email
	if err := service.sendModel(
		"user-guest-code",
		"Identity",
		mapof.Any{
			// User info available to the template
			"Email": identifier,
			"Token": token,
		},
	); err != nil {
		return derp.Wrap(err, location, "Sending guest code to: "+identifier)
	}

	// Woot!
	return nil
}

// SendFollowerConfirmation emails a new Follower the link that confirms their subscription
func (service *DomainEmail) SendFollowerConfirmation(actor model.PersonLink, follower *model.Follower) error {

	const location = "service.DomainEmail.SendFollowerConfirmation"

	// Send the confirmation email
	if err := service.sendModel(
		"follower-confirmation",
		"Follower",
		mapof.Any{
			// Parent info available to the template
			"Actor": actor,

			// Follower info available to the template
			"FollowerID": follower.FollowerID.Hex(),
			"Name":       follower.Actor.Name,
			"Email":      follower.Actor.EmailAddress,
			"Secret":     follower.Data.GetString("secret"),
		},
	); err != nil {
		return derp.Wrap(err, location, "Sending follow confirmation email", "followerId: "+follower.FollowerID.Hex())
	}

	return nil
}

// SendFollowerActivity emails the post that an activity brings to a Follower who subscribed by
// email. An activity that brings no post to read sends nothing.
func (service *DomainEmail) SendFollowerActivity(session data.Session, follower *model.Follower, activity mapof.Any) error {

	const location = "service.DomainEmail.SendFollowerActivity"

	// RULE: Only a post is emailed. A Delete, an Undo, a Block, or a post with neither a name nor
	// content would otherwise arrive as a blank "New Activity" message.
	post, isPost := followerPost(activity)

	if !isPost {
		return nil
	}

	// The email says who published the post
	actor, err := service.parentActors.LoadParentActor(session, follower)

	if err != nil {
		return derp.Wrap(err, location, "Loading the actor that the Follower follows", "followerId: "+follower.FollowerID.Hex())
	}

	readOnlyDomain := service.domainService.Cached()
	values := followerActivityValues(readOnlyDomain.Host(), follower, actor, post)

	if err := service.sendModel("follower-activity", "Follower", values); err != nil {
		return derp.Wrap(err, location, "Sending follower email", "followerId: "+follower.FollowerID.Hex())
	}

	// You've got mail
	return nil
}

// SendUnreadNotifications emails a User the list of their unread Notifications. moreCount is the
// number of unread Notifications that the list leaves out.
func (service *DomainEmail) SendUnreadNotifications(user *model.User, notifications []model.Notification, moreCount int64) error {

	const location = "service.DomainEmail.SendUnreadNotifications"

	unreadCount := int64(len(notifications)) + moreCount

	if err := service.sendModel(
		"user-notifications",
		"User",
		mapof.Any{
			// User info available to the template
			"UserID": user.UserID.Hex(),
			"Name":   user.DisplayName,
			"Email":  user.EmailAddress,

			// Notification info available to the template
			"CountLabel": unreadNotificationsLabel(unreadCount),
			"Lines":      notificationEmailLines(notifications),
			"MoreCount":  moreCount,

			// Links available to the template
			"NotificationsURL": service.host() + "/@me/notifications/all",
			"SettingsURL":      service.host() + "/@me/settings/notifications",
		},
	); err != nil {
		// RULE: The error names the User by ID, never by the email address
		return derp.Wrap(err, location, "Sending unread notifications email to user", user.UserID)
	}

	return nil
}

// notificationEmailLine is one row of the unread notifications email. Every value is plain text.
type notificationEmailLine struct {
	Name    string    // Who acted
	Action  string    // What they did
	Summary string    // The Notification's summary, or a label when it has none
	Date    time.Time // When it happened
}

// notificationEmailLines converts Notifications into the rows that the unread email prints
func notificationEmailLines(notifications []model.Notification) []notificationEmailLine {

	result := make([]notificationEmailLine, 0, len(notifications))

	for _, notification := range notifications {
		result = append(result, notificationEmailLine{
			Name:    first.String(notification.Actor.Name, notification.Actor.UsernameOrID(), "Someone"),
			Action:  notificationEmailAction(notification.Type),
			Summary: notificationEmailSummary(notification),
			Date:    time.UnixMilli(notification.CreateDate).UTC(),
		})
	}

	return result
}

// notificationEmailAction describes what the actor of a Notification did
func notificationEmailAction(notificationType string) string {

	switch notificationType {

	case model.NotificationTypeDirect:
		return "sent you a message"

	case model.NotificationTypeReply:
		return "replied to you"

	case model.NotificationTypeQuote:
		return "quoted your post"

	case model.NotificationTypeAnnounce:
		return "shared your post"

	case model.NotificationTypeLike:
		return "liked your post"

	case model.NotificationTypeDislike:
		return "disliked your post"

	case model.NotificationTypeFollow:
		return "followed you"
	}

	return "mentioned you"
}

// notificationEmailSummary returns the text that the unread email shows for a Notification
func notificationEmailSummary(notification model.Notification) string {

	// RULE: An MLS message has no summary on the server, and its content is never printed
	if notification.IsEncrypted() {
		return "Encrypted message"
	}

	return notification.ObjectSummary
}

// unreadNotificationsLabel counts unread Notifications in words, for the email's subject line
func unreadNotificationsLabel(count int64) string {

	if count == 1 {
		return "1 unread notification"
	}

	return strconv.FormatInt(count, 10) + " unread notifications"
}

// followerEmailPost is the post that a follower's email shows, as plain strings that the template
// can always read
type followerEmailPost struct {
	Title       string
	ContentHTML string
	URL         string
}

// followerPost returns the post that a follower's email shows for an activity, and FALSE when the
// activity brings no post to read
func followerPost(activity mapof.Any) (followerEmailPost, bool) {

	// RULE: Only these activity types bring a post to read. A Delete, an Undo, an Accept,
	// a Follow, or a Like has nothing to put in the email.
	switch streams.NewDocument(activity).Type() {
	case vocab.ActivityTypeCreate, vocab.ActivityTypeUpdate, vocab.ActivityTypeAnnounce:
	default:
		return followerEmailPost{}, false
	}

	object := streams.NewDocument(followerEmailObject(activity))

	// RULE: A profile is not a post, so an Update of an actor is not emailed
	if object.IsActor() {
		return followerEmailPost{}, false
	}

	// The link names the post by its id, because an Audio Stream's `url` is its media file
	post := followerEmailPost{
		Title:       object.Name(),
		ContentHTML: object.Content(),
		URL:         object.ID(),
	}

	// RULE: A post with neither a name nor content has nothing to show
	if (post.Title == "") && (post.ContentHTML == "") {
		return followerEmailPost{}, false
	}

	return post, true
}

// followerActivityValues returns every value that the follower-activity email reads, each of
// which is always present
func followerActivityValues(host string, follower *model.Follower, actor model.PersonLink, post followerEmailPost) mapof.Any {

	return mapof.Any{

		// Parent info available to the template
		"ParentLink": follower.ParentURL(host),
		"Actor":      actor,

		// Follower info available to the template
		"FollowerID": follower.FollowerID.Hex(),
		"Name":       follower.Actor.Name,
		"Email":      follower.Actor.EmailAddress,
		"Secret":     follower.Data.GetString("secret"),

		// Post info available to the template
		"Title":       post.Title,
		"ContentHTML": post.ContentHTML,
		"URL":         post.URL,

		// Unsubscribe links available to the template (D18: two forms, two consumers)
		"Unsubscribe":             follower.UnsubscribeLink(host),
		"UnsubscribeWithBrackets": follower.UnsubscribeLinkWithBrackets(host),
	}
}

// followerEmailObject returns the post that a follower's email shows for an activity: its embedded
// object, or the object of a Create or Update that it announces. It is empty when there is none.
func followerEmailObject(activity mapof.Any) mapof.Any {

	document := streams.NewDocument(activity)
	object := document.Object()

	// RULE: A Stream announces its new child as an Announce of the child's Create, so the email
	// shows the child itself, not the Create
	if document.Type() == vocab.ActivityTypeAnnounce {
		if object.IsMap() {
			switch object.Type() {
			case vocab.ActivityTypeCreate, vocab.ActivityTypeUpdate:
				object = object.Object()
			}
		}
	}

	// RULE: Show only an embedded object, because one named by its URL would have to be fetched
	if !object.IsMap() {
		return mapof.NewAny()
	}

	return mapof.Any(object.Map())
}

// host returns this Domain's hostname, with its protocol prefix
func (service *DomainEmail) host() string {
	return uri.PrependProtocol(service.hostname)
}
