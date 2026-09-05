package mailchimp

import (
	"strconv"

	"github.com/benpate/rosetta/sliceof"
)

// mergeFieldPageSize is the largest page that GET /lists/{id}/merge-fields will return
const mergeFieldPageSize = 1000

// MergeField is one custom field defined on a Mailchimp audience. `Tag` is the merge
// tag that identifies it, capped by Mailchimp at 10 characters.
type MergeField struct {
	MergeID int    `json:"merge_id"`
	Tag     string `json:"tag"`
	Name    string `json:"name"`
	Type    string `json:"type"`
}

// GetMergeFields returns every merge field defined on the provided audience
func (client Client) GetMergeFields(audienceID string) (sliceof.Object[MergeField], error) {

	const location = "tools.mailchimp.Client.GetMergeFields"

	var result struct {
		MergeFields sliceof.Object[MergeField] `json:"merge_fields"`
	}

	transaction := client.get("/lists/"+audienceID+"/merge-fields").
		Query("count", strconv.Itoa(mergeFieldPageSize)).
		Query("fields", "merge_fields.merge_id,merge_fields.tag,merge_fields.name,merge_fields.type").
		Result(&result)

	if err := transaction.Send(); err != nil {
		return nil, describeError(err, location, "Unable to read merge fields from Mailchimp")
	}

	return result.MergeFields, nil
}

// CreateMergeField creates a text merge field on the provided audience
func (client Client) CreateMergeField(audienceID string, tag string, name string) (MergeField, error) {

	const location = "tools.mailchimp.Client.CreateMergeField"

	result := MergeField{}

	// RULE: `required` and `public` are set explicitly, and both must stay FALSE. A required
	// field breaks every other way members enter this audience, including the User's own
	// signup forms, and a public one shows an internal ID to their subscribers.
	body := map[string]any{
		"tag":      tag,
		"name":     name,
		"type":     "text",
		"required": false,
		"public":   false,
	}

	transaction := client.post("/lists/"+audienceID+"/merge-fields", body).Result(&result)

	if err := transaction.Send(); err != nil {
		return MergeField{}, describeError(err, location, "Unable to create a merge field in Mailchimp")
	}

	return result, nil
}
