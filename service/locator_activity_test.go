package service

import (
	"strings"
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// TestLocator_ParseActivity verifies that the activity URL of a User or a Stream is read back as written
func TestLocator_ParseActivity(t *testing.T) {

	locator := Locator{host: "https://example.com"}
	actorID := primitive.NewObjectID()
	activityID := primitive.NewObjectID()

	for _, actorType := range []string{model.ActorTypeUser, model.ActorTypeStream} {

		activityURL := locator.ActivityURL(actorType, actorID, activityID)

		parsedType, parsedActor, parsedActivity, err := locator.ParseActivity(activityURL)
		require.NoError(t, err, activityURL)
		require.Equal(t, actorType, parsedType, activityURL)
		require.Equal(t, actorID, parsedActor, activityURL)
		require.Equal(t, activityID, parsedActivity, activityURL)
	}
}

// TestLocator_ParseActivity_Refuses verifies that a URL naming no User or Stream activity on this host is
// refused as a bad request, and returns zero values
func TestLocator_ParseActivity_Refuses(t *testing.T) {

	locator := Locator{host: "https://example.com"}
	actorID := primitive.NewObjectID().Hex()
	activityID := primitive.NewObjectID().Hex()

	for _, url := range []string{
		"",
		"https://example.com",
		"https://example.com/",
		"https://example.com/@" + actorID,
		"https://example.com/@" + actorID + "/pub/outbox",
		"https://example.com/@" + actorID + "/pub/outbox/",
		"https://example.com/@" + actorID + "/pub/outbox/not-an-id",
		"https://example.com/@" + actorID + "/pub/outbox/" + activityID + "/extra",
		"https://example.com/@" + actorID + "/pub/followers/" + activityID,
		"https://example.com/@not-an-id/pub/outbox/" + activityID,
		"https://example.com/@application/pub/outbox/" + activityID,
		"https://example.com/@search/pub/outbox/" + activityID,
		"https://example.com/@search_" + actorID + "/pub/outbox/" + activityID,
		"https://example.com/my-blog/pub/outbox/" + activityID,
		"https://example.com//pub/outbox/" + activityID,
		"https://evil.example/@" + actorID + "/pub/outbox/" + activityID,
		"https://example.com.evil.example/" + actorID + "/pub/outbox/" + activityID,
		"http://example.com/" + actorID + "/pub/outbox/" + activityID,
	} {
		actorType, parsedActor, parsedActivity, err := locator.ParseActivity(url)
		require.Error(t, err, url)
		require.True(t, derp.IsBadRequest(err), url)
		require.Empty(t, actorType, url)
		require.True(t, parsedActor.IsZero(), url)
		require.True(t, parsedActivity.IsZero(), url)
	}
}

// FuzzLocator_ParseActivity verifies that ParseActivity never panics, and that a URL it accepts
// is exactly the one ActivityURL writes for what it returned
func FuzzLocator_ParseActivity(f *testing.F) {

	locator := Locator{host: "https://example.com"}
	actorID := primitive.NewObjectID()
	activityID := primitive.NewObjectID()

	f.Add(locator.ActivityURL(model.ActorTypeUser, actorID, activityID))
	f.Add(locator.ActivityURL(model.ActorTypeStream, actorID, activityID))
	f.Add("https://example.com/@/pub/outbox/")
	f.Add("https://example.com/@@" + actorID.Hex() + "/pub/outbox/" + activityID.Hex())
	f.Add("https://example.com/" + actorID.Hex() + "/pub/outbox/pub/outbox/" + activityID.Hex())
	f.Add("\xff\xfe")

	f.Fuzz(func(t *testing.T, url string) {

		actorType, parsedActor, parsedActivity, err := locator.ParseActivity(url)

		if err != nil {
			return
		}

		// ObjectIDFromHex accepts upper-case hex, which ActivityURL never writes
		require.True(t, strings.EqualFold(url, locator.ActivityURL(actorType, parsedActor, parsedActivity)), url)
	})
}
