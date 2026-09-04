package mailchimp

import (
	"strconv"

	"github.com/benpate/rosetta/sliceof"
)

// audiencePageSize is the largest page that GET /lists will return
const audiencePageSize = 1000

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

// GetAudiences returns every audience in this Client's Mailchimp account
func (client Client) GetAudiences() (sliceof.Object[Audience], error) {

	const location = "tools.mailchimp.Client.GetAudiences"

	var result struct {
		Lists sliceof.Object[Audience] `json:"lists"`
	}

	// Ask for the whole account in one page, and only for the fields a picker shows
	transaction := client.get("/lists").
		Query("count", strconv.Itoa(audiencePageSize)).
		Query("fields", "lists.id,lists.name,lists.stats.member_count").
		Result(&result)

	if err := transaction.Send(); err != nil {
		return nil, describeError(err, location, "Unable to read audiences from Mailchimp")
	}

	// An account with no audience at all is a real answer, not an error
	return result.Lists, nil
}
