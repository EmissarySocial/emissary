package service

import (
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/tools/secretcheck"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// privateInboxActivity returns a direct message as an inbox stores it, whose content and
// recipient must never reach an error
func privateInboxActivity() model.InboxActivity {

	inboxActivity := model.NewInboxActivity()
	inboxActivity.UserID = primitive.NewObjectID()
	inboxActivity.ActorID = "https://remote.example/@sender"
	inboxActivity.ActivityID = "https://remote.example/activities/1"
	inboxActivity.ActivityType = vocab.ActivityTypeCreate
	inboxActivity.RawActivity = mapof.Any{
		vocab.PropertyType: vocab.ActivityTypeCreate,
		vocab.PropertyTo:   []string{"https://example.com/@private-recipient"},
		vocab.PropertyObject: mapof.Any{
			vocab.PropertyType:    vocab.ObjectTypeNote,
			vocab.PropertyContent: "the private message body",
		},
	}

	return inboxActivity
}

// TestInbox_ErrorsCarryNoActivity verifies that every error Inbox writes names the activity by its
// id, and never carries the stored activity, whose content may be a direct message
func TestInbox_ErrorsCarryNoActivity(t *testing.T) {

	// O1: these errors attached the whole InboxActivity, raw activity included, and the
	// mongo reporter stores every detail in ErrorLog
	service := Inbox{}
	session := brokenSession{}

	// requireNoActivity checks one error for the activity's content and its recipient
	requireNoActivity := func(t *testing.T, err error) {
		t.Helper()
		require.Error(t, err)
		secretcheck.RequireAbsent(t, err, "the private message body")
		secretcheck.RequireAbsent(t, err, "private-recipient")
	}

	t.Run("an invalid activity", func(t *testing.T) {
		inboxActivity := privateInboxActivity()
		inboxActivity.ActorID = ""
		requireNoActivity(t, service.Save(session, &inboxActivity, "test"))
	})

	t.Run("a failed save", func(t *testing.T) {
		inboxActivity := privateInboxActivity()
		requireNoActivity(t, service.Save(session, &inboxActivity, "test"))
	})

	t.Run("a failed load of the earlier copy", func(t *testing.T) {
		inboxActivity := privateInboxActivity()
		requireNoActivity(t, service.createOrUpdate(failingSession{}, &inboxActivity, "test"))
	})

	t.Run("a failed delete", func(t *testing.T) {
		inboxActivity := privateInboxActivity()
		requireNoActivity(t, service.Delete(session, &inboxActivity, "test"))
	})
}
