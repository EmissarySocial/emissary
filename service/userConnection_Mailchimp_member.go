package service

import (
	"strings"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/tools/mailchimp"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

/******************************************
 * Mailchimp Members
 *
 * The outbound half: an Emissary Follower becoming a member of the User's
 * Mailchimp audience, and stopping being one. Both are called from queue
 * tasks, so both begin by finding a connection that may have been paused or
 * deleted since the task was enqueued. See MAILING-LISTS.md 1.2.
 ******************************************/

// MailchimpAddMember writes an Emissary Follower into its User's Mailchimp audience
func (service *UserConnection) MailchimpAddMember(session data.Session, userID primitive.ObjectID, follower *model.Follower) error {

	const location = "service.UserConnection.MailchimpAddMember"

	client, userConnection, err := service.mailchimp_readyClient(session, userID)

	if err != nil {
		return derp.Wrap(err, location, "Unable to reach Mailchimp", userID)
	}

	// A connection that is paused, unfinished, or gone is not an error -- it is the User
	// having switched this off between the enqueue and the run.
	if client == nil {
		return nil
	}

	if err := mailchimp_pushMember(client, userConnection, follower); err != nil {
		return service.mailchimp_reportMemberError(session, userConnection, derp.Wrap(err, location, "Unable to add member"))
	}

	return nil
}

// MailchimpRemoveMember marks an address as unsubscribed in its User's Mailchimp audience
func (service *UserConnection) MailchimpRemoveMember(session data.Session, userID primitive.ObjectID, emailAddress string) error {

	const location = "service.UserConnection.MailchimpRemoveMember"

	// RULE: the Follower is already deleted by the time this runs, so the address travels in
	// the task arguments. An empty one would address a member that is not this person.
	if strings.TrimSpace(emailAddress) == "" {
		return derp.BadRequest(location, "Email address is required", userID)
	}

	client, userConnection, err := service.mailchimp_readyClient(session, userID)

	if err != nil {
		return derp.Wrap(err, location, "Unable to reach Mailchimp", userID)
	}

	if client == nil {
		return nil
	}

	audienceID := userConnection.Data.GetString(model.UserConnectionDataAudienceID)

	if err := client.UnsubscribeMember(audienceID, emailAddress); err != nil {
		return service.mailchimp_reportMemberError(session, userConnection, derp.Wrap(err, location, "Unable to unsubscribe member"))
	}

	return nil
}

// mailchimp_readyClient returns a Client for a User's Mailchimp connection, or NIL when
// they have no connection that is switched on and finished being set up
func (service *UserConnection) mailchimp_readyClient(session data.Session, userID primitive.ObjectID) (*mailchimp.Client, *model.UserConnection, error) {

	const location = "service.UserConnection.mailchimp_readyClient"

	userConnection := model.NewUserConnection()

	if err := service.LoadByUserAndType(session, userID, model.UserConnectionTypeMailchimp, &userConnection); err != nil {

		// Most Users have no Mailchimp connection at all, which is the ordinary case rather
		// than a failure worth retrying a task over.
		if derp.IsNotFound(err) {
			return nil, nil, nil
		}

		return nil, nil, derp.Wrap(err, location, "Unable to load connection", userID)
	}

	// RULE: IsReady() covers paused, unfinished, and credential-rejected in one predicate.
	// A task enqueued before any of those is silently dropped rather than retried.
	if !userConnection.IsReady() {
		return nil, nil, nil
	}

	apiKey, err := service.mailchimp_apiKey(&userConnection)

	if err != nil {
		return nil, nil, derp.Wrap(err, location, "Unable to read the saved API key", userID)
	}

	client, err := mailchimp.New(apiKey, userConnection.Data.GetString(model.UserConnectionDataCenter))

	if err != nil {
		return nil, nil, derp.Wrap(err, location, "Unable to build a Mailchimp client", userID)
	}

	return &client, &userConnection, nil
}

// mailchimp_reportMemberError flags a connection whose credential Mailchimp has rejected,
// and returns the error unchanged
func (service *UserConnection) mailchimp_reportMemberError(session data.Session, userConnection *model.UserConnection, err error) error {

	const location = "service.UserConnection.mailchimp_reportMemberError"

	// Only a rejected credential is worth recording. Everything else is transient, and the
	// task's own retry is the right response to it.
	if !derp.IsUnauthorized(err) && !derp.IsForbidden(err) {
		return err
	}

	userConnection.Status = model.UserConnectionStatusReconnect

	// RULE: write through the collection, NOT through Save. Save calls connect(), which would
	// reach for Mailchimp again with the very credential that was just refused -- and would
	// then overwrite the status this line exists to record.
	if writeErr := service.collection(session).Save(userConnection, "Mailchimp rejected these credentials"); writeErr != nil {
		derp.Report(derp.Wrap(writeErr, location, "Unable to flag connection for reconnect", userConnection.UserConnectionID))
	}

	return err
}

// mailchimp_pushMember writes a Follower into the connection's audience, then applies the
// connection's tag when it has one
func mailchimp_pushMember(client *mailchimp.Client, userConnection *model.UserConnection, follower *model.Follower) error {

	const location = "service.mailchimp_pushMember"

	audienceID := userConnection.Data.GetString(model.UserConnectionDataAudienceID)

	if err := client.SetMember(audienceID, mailchimp_member(follower)); err != nil {
		return derp.Wrap(err, location, "Unable to add member to the audience")
	}

	// The tag is optional (D49), and it is a second request because the member PUT cannot
	// carry one. It is applied on every push: the call is idempotent, and an upsert cannot
	// say whether the member was new.
	tag := strings.TrimSpace(userConnection.Data.GetString(model.UserConnectionDataTag))

	if tag == "" {
		return nil
	}

	if err := client.TagMember(audienceID, follower.Actor.EmailAddress, tag); err != nil {
		return derp.Wrap(err, location, "Unable to tag the member")
	}

	return nil
}

// mailchimp_member converts an Emissary Follower into the Mailchimp member it becomes
func mailchimp_member(follower *model.Follower) mailchimp.Member {

	firstName, lastName := mailchimp_splitName(follower.Actor.Name)

	result := mailchimp.Member{
		EmailAddress: follower.Actor.EmailAddress,

		// RULE: `subscribed`, never `pending`. Emissary has already run its own double
		// opt-in (D5), so asking Mailchimp to confirm again would cost every new subscriber
		// a second email for no additional consent.
		Status: mailchimp.MemberStatusSubscribed,

		MergeFields: map[string]string{},
	}

	// Merge fields are omitted rather than blanked, because an empty value would overwrite
	// whatever the User already has in Mailchimp for this member.
	if firstName != "" {
		result.MergeFields["FNAME"] = firstName
	}

	if lastName != "" {
		result.MergeFields["LNAME"] = lastName
	}

	// Every Follower who signed up before D31's capture shipped has no IP, permanently --
	// so this is absent rather than empty.
	result.IPSignup = follower.Data.GetString(model.FollowerDataIPSignup)

	return result
}

// mailchimp_splitName divides a display name into the FNAME and LNAME that Mailchimp expects
func mailchimp_splitName(name string) (string, string) {

	name = strings.TrimSpace(name)

	// Split on the LAST space, so that a multi-part given name stays together (D8)
	if index := strings.LastIndex(name, " "); index >= 0 {
		return strings.TrimSpace(name[:index]), strings.TrimSpace(name[index+1:])
	}

	return name, ""
}
