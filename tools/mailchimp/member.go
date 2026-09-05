package mailchimp

import (
	"crypto/md5" // #nosec G501 -- Mailchimp addresses members by the MD5 of their address; this is an identifier, not a security primitive
	"encoding/hex"
	"strings"
)

// Member is one subscriber in a Mailchimp audience
type Member struct {
	EmailAddress string            `json:"email_address"`
	Status       string            `json:"status,omitempty"`
	MergeFields  map[string]string `json:"merge_fields,omitempty"`
	IPSignup     string            `json:"ip_signup,omitempty"`
}

// MemberStatusSubscribed marks a member who receives this audience's campaigns
const MemberStatusSubscribed = "subscribed"

// MemberStatusUnsubscribed marks a member who has opted out of this audience
const MemberStatusUnsubscribed = "unsubscribed"

// SubscriberHash returns the ID that Mailchimp addresses a member by
func SubscriberHash(emailAddress string) string {

	// Mailchimp defines this as the MD5 of the lowercased address, and will not accept a
	// member addressed any other way.
	sum := md5.Sum([]byte(strings.ToLower(strings.TrimSpace(emailAddress)))) // #nosec G401 -- see the import comment

	return hex.EncodeToString(sum[:])
}

// SetMember adds a member to an audience, or updates them if they are already in it
func (client Client) SetMember(audienceID string, member Member) error {

	const location = "tools.mailchimp.Client.SetMember"

	// PUT upserts, which is what makes this callable on every save of a Follower without
	// caring whether the member is already there.
	path := "/lists/" + audienceID + "/members/" + SubscriberHash(member.EmailAddress)

	if err := client.put(path, member).Send(); err != nil {
		return describeError(err, location, "Unable to add this member to Mailchimp")
	}

	return nil
}

// UnsubscribeMember marks a member of an audience as unsubscribed, and treats a member who
// was never there as success
func (client Client) UnsubscribeMember(audienceID string, emailAddress string) error {

	const location = "tools.mailchimp.Client.UnsubscribeMember"

	path := "/lists/" + audienceID + "/members/" + SubscriberHash(emailAddress)

	body := map[string]any{
		"status": MemberStatusUnsubscribed,
	}

	if err := client.patch(path, body).Send(); err != nil {

		// RULE: an address Mailchimp does not have is the state this call exists to reach.
		// Emissary pushes only on confirmation, so a Follower who unsubscribes before their
		// first push was never a member, and that must not fail the task forever.
		if isNotFound(err) {
			return nil
		}

		return describeError(err, location, "Unable to unsubscribe this member from Mailchimp")
	}

	return nil
}
