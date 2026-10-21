package build

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/tools/id"
	"github.com/benpate/form"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// StepSetCircleSharing's form must show what a Stream is actually shared with.  A Stream that grants
// the role to no Group is private, so its form checks "Not Shared", never "Everyone".

// TestStepSetCircleSharing_CalculateValue verifies the choice the form shows for each kind of sharing
func TestStepSetCircleSharing_CalculateValue(t *testing.T) {

	circleID := primitive.NewObjectID()
	step := StepSetCircleSharing{Role: "viewer"}

	testCases := []struct {
		name     string
		groups   id.Slice
		circles  id.Slice
		expected id.Slice
	}{
		{"public", id.Slice{model.MagicGroupIDAnonymous}, nil, id.Slice{model.MagicGroupIDAnonymous}},
		{"circles", nil, id.Slice{circleID}, id.Slice{circleID}},
		{"shared with no one", nil, nil, id.Slice{model.MagicGroupIDOwners}},
		{"owners only", id.Slice{model.MagicGroupIDOwners}, nil, id.Slice{model.MagicGroupIDOwners}},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {

			stream := model.NewStream()
			stream.Groups["viewer"] = testCase.groups
			stream.Circles["viewer"] = testCase.circles

			value := step.calculateValue(&stream)
			require.Equal(t, testCase.expected, value["circles"])
		})
	}
}

// TestCircleSharingStatus verifies the menu label for a Stream shared with Circles: the Circle's name
// and icon when there is one, and a count otherwise
func TestCircleSharingStatus(t *testing.T) {

	family := primitive.NewObjectID()
	friends := primitive.NewObjectID()
	missing := primitive.NewObjectID()

	circles := []form.LookupCode{
		{Value: family.Hex(), Label: "Family", Icon: "house"},
		{Value: friends.Hex(), Label: "Friends"},
	}

	testCases := []struct {
		name      string
		circleIDs id.Slice
		label     string
		icon      string
	}{
		{"one named circle with an icon", id.Slice{family}, "Sharing: Family", "house"},
		{"one named circle without an icon", id.Slice{friends}, "Sharing: Friends", "people"},
		{"one circle not among the user's", id.Slice{missing}, "Sharing: 1 Circle", "people"},
		{"two circles", id.Slice{family, friends}, "Sharing: 2 Circles", "people"},
		{"a named group but no circles", nil, "Sharing: Groups", "people"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			result := circleSharingStatus(testCase.circleIDs, circles)
			require.Equal(t, model.SharingStatusCircles, result.Value)
			require.Equal(t, testCase.label, result.Label)
			require.Equal(t, testCase.icon, result.Icon)
		})
	}
}

// TestStepSetCircleSharing_Post verifies what each choice writes.  "Not Shared" grants the role to no
// Group, not to the Owners Group its button carries, and a form with nothing checked is public.
func TestStepSetCircleSharing_Post(t *testing.T) {

	circleID := primitive.NewObjectID()
	step := StepSetCircleSharing{Role: "viewer"}

	testCases := []struct {
		name    string
		posted  []string
		groups  id.Slice
		circles id.Slice
	}{
		{"everyone", []string{model.MagicGroupIDAnonymous.Hex()}, id.Slice{model.MagicGroupIDAnonymous}, id.NewSlice()},
		{"not shared", []string{model.MagicGroupIDOwners.Hex()}, id.NewSlice(), id.NewSlice()},
		{"a circle", []string{circleID.Hex()}, id.NewSlice(), id.Slice{circleID}},
		{"nothing checked", nil, id.Slice{model.MagicGroupIDAnonymous}, id.NewSlice()},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {

			stream := model.NewStream()
			stream.Groups["viewer"] = id.Slice{model.MagicGroupIDAnonymous}
			stream.Circles["viewer"] = id.Slice{primitive.NewObjectID()}

			body := url.Values{"circles": testCase.posted}.Encode()
			request := httptest.NewRequest(http.MethodPost, "/stream/sharing", strings.NewReader(body))
			request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

			builder := Stream{
				_stream:            &stream,
				CommonWithTemplate: CommonWithTemplate{Common: Common{_request: request}},
			}

			require.Nil(t, step.Post(builder, io.Discard))
			require.Equal(t, testCase.groups, stream.Groups["viewer"])
			require.Equal(t, testCase.circles, stream.Circles["viewer"])
		})
	}
}
