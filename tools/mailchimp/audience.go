package mailchimp

import (
	"github.com/benpate/derp"
)

// Audience is one Mailchimp audience, which the Marketing API calls a "list"
type Audience struct {
	ID    string        `json:"id"`
	Name  string        `json:"name"`
	Stats AudienceStats `json:"stats"`
}

// AudienceStats carries the counts Mailchimp reports alongside an Audience
type AudienceStats struct {
	MemberCount int `json:"member_count"`
}

// GetAudience returns the single audience named by a Mailchimp Audience ID
func (client Client) GetAudience(audienceID string) (Audience, error) {

	const location = "tools.mailchimp.Client.GetAudience"

	result := Audience{}

	transaction := client.get("/lists/"+audienceID).
		Query("fields", "id,name,stats.member_count").
		Result(&result)

	if err := transaction.Send(); err != nil {

		// RULE: a 404 here is about the Audience ID, not the data center, so it must NOT
		// reuse describeError's shared message -- that one sends the User off to re-check a
		// value that was already correct.
		if isNotFound(err) {
			return Audience{}, derp.Validation("Mailchimp has no audience with this ID. In Mailchimp, open Audience, then Settings, then 'Audience name and defaults' -- the ID is in the box marked 'Unique id for audience'. The number in your browser's address bar is a different ID and will not work. If you do not have an audience yet, create one there first.", derp.WithLocation(location))
		}

		return Audience{}, describeError(err, location, "Unable to read this audience from Mailchimp")
	}

	return result, nil
}
