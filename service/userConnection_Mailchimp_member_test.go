package service

import (
	"testing"

	"strings"

	"github.com/EmissarySocial/emissary/model"

	"github.com/EmissarySocial/emissary/tools/mailchimp"
	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/require"
)

// TestMailchimpMember_CarriesTheFourValues pins what D8 sends per member
func TestMailchimpMember_CarriesTheFourValues(t *testing.T) {

	follower := newMailingListFollower()
	follower.Data.SetString(model.FollowerDataIPSignup, "10.0.0.1")

	member := mailchimp_member(&follower)

	require.Equal(t, "sarah@connor.mil", member.EmailAddress)
	require.Equal(t, "Sarah", member.MergeFields["FNAME"])
	require.Equal(t, "Connor", member.MergeFields["LNAME"])
	require.Equal(t, follower.FollowerID.Hex(), member.MergeFields[mailchimpMergeFieldTag])
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
	require.Contains(t, member.MergeFields, mailchimpMergeFieldTag, "the Follower link is always sent")
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
