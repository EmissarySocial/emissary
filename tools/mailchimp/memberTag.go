package mailchimp

import (
	"strings"

	"github.com/benpate/derp"
)

// memberTag declares one tag active or inactive on a member
type memberTag struct {
	Name   string `json:"name"`
	Status string `json:"status"`
}

// memberTagStatusActive applies a tag to a member, creating the tag if the audience has none by that name
const memberTagStatusActive = "active"

// TagMember applies a tag to a member of an audience, creating the tag on first use
func (client Client) TagMember(audienceID string, emailAddress string, tag string) error {

	const location = "tools.mailchimp.Client.TagMember"

	// RULE: a blank name would ask Mailchimp for a tag called "", which it refuses with a
	// 400 -- the one status the queue never retries. Refusing here keeps that off the wire.
	tag = strings.TrimSpace(tag)

	if tag == "" {
		return derp.BadRequest(location, "Tag name is required")
	}

	// The member PUT cannot carry tags, so this is its own call. Mailchimp creates a tag
	// named here on demand, which is why there is no list-then-create step first.
	path := "/lists/" + audienceID + "/members/" + SubscriberHash(emailAddress) + "/tags"

	body := map[string]any{
		"tags": []memberTag{{Name: tag, Status: memberTagStatusActive}},
	}

	if err := client.post(path, body).Send(); err != nil {
		return describeMemberError(err, location, "Unable to tag this member in Mailchimp")
	}

	// Tag, you're it.
	return nil
}
