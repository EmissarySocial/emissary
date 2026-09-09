package service

import (
	"net/http"
	"strings"
	"testing"

	"github.com/EmissarySocial/emissary/model"

	"github.com/EmissarySocial/emissary/tools/mailchimp"
	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/require"
)

// TestMailchimpMember_CarriesTheThreeValues pins what D8 sends per member
func TestMailchimpMember_CarriesTheThreeValues(t *testing.T) {

	follower := newMailingListFollower()
	follower.Data.SetString(model.FollowerDataIPSignup, "10.0.0.1")

	member := mailchimp_member(&follower)

	require.Equal(t, "sarah@connor.mil", member.EmailAddress)
	require.Equal(t, "Sarah", member.MergeFields["FNAME"])
	require.Equal(t, "Connor", member.MergeFields["LNAME"])
	require.Equal(t, "10.0.0.1", member.IPSignup)

	// RULE: `subscribed`, never `pending`. Emissary has already run its own double opt-in,
	// so a second confirmation from Mailchimp costs the subscriber an email and buys nothing.
	require.Equal(t, mailchimp.MemberStatusSubscribed, member.Status)
}

// TestMailchimpMember_OmitsWhatItDoesNotKnow keeps Emissary from blanking Mailchimp's data
func TestMailchimpMember_OmitsWhatItDoesNotKnow(t *testing.T) {

	// Every Follower who confirmed before D31's capture shipped has no signup IP, and that is
	// permanent rather than transient. Sending "" would write that blank over whatever the
	// User's own signup forms had already collected.

	follower := newMailingListFollower()
	follower.Actor.Name = ""

	member := mailchimp_member(&follower)

	require.Empty(t, member.IPSignup)
	require.NotContains(t, member.MergeFields, "FNAME")
	require.NotContains(t, member.MergeFields, "LNAME")
	require.Empty(t, member.MergeFields, "with nothing to say, no merge_fields key is sent at all")
}

// TestMailchimpSplitName verifies the split D8 describes, and its inverse
func TestMailchimpSplitName(t *testing.T) {

	tests := []struct {
		name  string
		first string
		last  string
	}{
		{"Sarah Connor", "Sarah", "Connor"},
		{"Sarah", "Sarah", ""},
		{"", "", ""},
		{"  Sarah Connor  ", "Sarah", "Connor"},

		// Split on the LAST space, so a multi-part given name stays together
		{"Sarah Jeanette Connor", "Sarah Jeanette", "Connor"},
		{"Miles Bennett Dyson", "Miles Bennett", "Dyson"},
	}

	for _, test := range tests {

		t.Run(test.name, func(t *testing.T) {

			first, last := mailchimp_splitName(test.name)

			require.Equal(t, test.first, first)
			require.Equal(t, test.last, last)

			// The inbound webhook rejoins these with mailchimpMemberName, so the two halves
			// of one conversion must agree -- otherwise a name drifts a little on every trip
			// between the two systems.
			rejoined := mailchimpMemberName(mapof.String{
				"data[merges][FNAME]": first,
				"data[merges][LNAME]": last,
			})

			require.Equal(t, strings.TrimSpace(test.name), rejoined)
		})
	}
}

/******************************************
 * Pushing a member, with and without a tag
 *
 * These pin the NUMBER and ORDER of requests. The wire format of each one is
 * covered by tools/mailchimp; what matters here is that a tag costs exactly one
 * extra call, after the member exists, and that no tag costs nothing.
 ******************************************/

// TestMailchimpPushMember_TagsAfterTheMemberExists pins the second call and its order
func TestMailchimpPushMember_TagsAfterTheMemberExists(t *testing.T) {

	recorder := newMailchimpRecorder()
	defer recorder.server.Close()

	_, client, userConnection := newSetupService(t, recorder)
	userConnection.Data.SetString(model.UserConnectionDataAudienceID, "abc123")
	userConnection.Data.SetString(model.UserConnectionDataTag, "Emissary")

	follower := newMailingListFollower()
	require.NoError(t, mailchimp_pushMember(&client, &userConnection, &follower))

	// The member must exist before it can be tagged, so PUT comes first
	memberPath := "/lists/abc123/members/" + mailchimp.SubscriberHash("sarah@connor.mil")
	require.Equal(t, []string{"PUT " + memberPath, "POST " + memberPath + "/tags"}, recorder.requests)
	require.JSONEq(t, `{"tags":[{"name":"Emissary","status":"active"}]}`, recorder.bodies["POST "+memberPath+"/tags"])
}

// TestMailchimpPushMember_NoTagIsOneRequest keeps the hot path at one call when nothing asks
// for more
func TestMailchimpPushMember_NoTagIsOneRequest(t *testing.T) {

	for name, tag := range map[string]string{"absent": "", "blank": "   "} {

		t.Run(name, func(t *testing.T) {

			recorder := newMailchimpRecorder()
			defer recorder.server.Close()

			_, client, userConnection := newSetupService(t, recorder)
			userConnection.Data.SetString(model.UserConnectionDataAudienceID, "abc123")

			if tag != "" {
				userConnection.Data.SetString(model.UserConnectionDataTag, tag)
			}

			follower := newMailingListFollower()
			require.NoError(t, mailchimp_pushMember(&client, &userConnection, &follower))

			require.Len(t, recorder.requests, 1)
			require.Equal(t, http.MethodPut+" /lists/abc123/members/"+mailchimp.SubscriberHash("sarah@connor.mil"), recorder.requests[0])
		})
	}
}

// TestMailchimpPushMember_ATagFailureIsAnError keeps a tag Mailchimp refuses from vanishing
func TestMailchimpPushMember_ATagFailureIsAnError(t *testing.T) {

	// The member call succeeded, so the member is there. The task must still fail, so the
	// queue retries and the tag is eventually applied rather than silently skipped.

	recorder := newMailchimpRecorder()
	defer recorder.server.Close()

	memberPath := "/lists/abc123/members/" + mailchimp.SubscriberHash("sarah@connor.mil")
	recorder.failPaths["POST "+memberPath+"/tags"] = http.StatusInternalServerError

	_, client, userConnection := newSetupService(t, recorder)
	userConnection.Data.SetString(model.UserConnectionDataAudienceID, "abc123")
	userConnection.Data.SetString(model.UserConnectionDataTag, "Emissary")

	follower := newMailingListFollower()
	err := mailchimp_pushMember(&client, &userConnection, &follower)

	require.Error(t, err)
	require.Equal(t, 1, recorder.count("PUT "+memberPath), "the member was still written")
}
