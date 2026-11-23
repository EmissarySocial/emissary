package service

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/tools/secretcheck"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// TestFollowerEmailObject verifies which post a follower's email shows for each kind of activity
func TestFollowerEmailObject(t *testing.T) {

	note := mapof.Any{"id": "https://example.com/notes/1", "type": "Note", "content": "Hello"}

	// test checks the object shown for one activity
	test := func(name string, activity mapof.Any, expected mapof.Any) {
		t.Run(name, func(t *testing.T) {
			require.Equal(t, expected, followerEmailObject(activity))
		})
	}

	test("a Create shows its object", mapof.Any{"type": "Create", "object": note}, note)
	test("an Announce of a post shows the post", mapof.Any{"type": "Announce", "object": note}, note)
	test("an Announce of a Create shows the created post", mapof.Any{"type": "Announce", "object": mapof.Any{"type": "Create", "object": note}}, note)
	test("an Announce of an Update shows the updated post", mapof.Any{"type": "Announce", "object": mapof.Any{"type": "Update", "object": note}}, note)
	test("an Announce of a Delete shows the Delete", mapof.Any{"type": "Announce", "object": mapof.Any{"type": "Delete", "object": "https://example.com/notes/1"}}, mapof.Any{"type": "Delete", "object": "https://example.com/notes/1"})
	test("an Announce of a Create of a URL shows nothing", mapof.Any{"type": "Announce", "object": mapof.Any{"type": "Create", "object": "https://example.com/notes/1"}}, mapof.NewAny())
	test("an object named by its URL shows nothing", mapof.Any{"type": "Create", "object": "https://example.com/notes/1"}, mapof.NewAny())
	test("an Announce of a URL shows nothing", mapof.Any{"type": "Announce", "object": "https://example.com/notes/1"}, mapof.NewAny())
	test("no object shows nothing", mapof.Any{"type": "Create"}, mapof.NewAny())
	test("a nil activity shows nothing", nil, mapof.NewAny())
}

// TestFollowerPost verifies which activities bring a post to email, and what the email shows of it
func TestFollowerPost(t *testing.T) {

	note := map[string]any{"id": "https://example.com/notes/1", "type": "Note", "content": "Hello"}

	// test checks the post that one activity brings, if any
	test := func(name string, activity map[string]any, expected followerEmailPost, isPost bool) {
		t.Run(name, func(t *testing.T) {
			post, ok := followerPost(activity)
			require.Equal(t, isPost, ok)
			require.Equal(t, expected, post)
		})
	}

	hello := followerEmailPost{ContentHTML: "Hello", URL: "https://example.com/notes/1"}
	none := followerEmailPost{}

	test("a Create of a Note", map[string]any{"type": "Create", "object": note}, hello, true)
	test("an Announce of a Note", map[string]any{"type": "Announce", "object": note}, hello, true)
	test("an Announce of a Create of a Note", map[string]any{"type": "Announce", "object": map[string]any{"type": "Create", "object": note}}, hello, true)
	test("an object as mapof.Any", map[string]any{"type": "Create", "object": mapof.Any(note)}, hello, true)
	test("an Update of an Article with only a name", map[string]any{"type": "Update", "object": map[string]any{"type": "Article", "name": "Title"}}, followerEmailPost{Title: "Title"}, true)
	test("a Create of a Stream with no body", map[string]any{"type": "Create", "object": map[string]any{"id": "https://example.com/1", "type": "Event", "name": "Tour date"}}, followerEmailPost{Title: "Tour date", URL: "https://example.com/1"}, true)
	test("the link is the id, not the url", map[string]any{"type": "Create", "object": map[string]any{"id": "https://example.com/1", "url": "https://example.com/1.mp3", "type": "Audio", "name": "Song"}}, followerEmailPost{Title: "Song", URL: "https://example.com/1"}, true)

	test("a Create of an empty Note", map[string]any{"type": "Create", "object": map[string]any{"type": "Note"}}, none, false)
	test("a Create of a Note whose content is null", map[string]any{"type": "Create", "object": map[string]any{"type": "Note", "content": nil}}, none, false)
	test("a Create of an object named by its URL", map[string]any{"type": "Create", "object": "https://remote.example/notes/1"}, none, false)
	test("a Create with no object", map[string]any{"type": "Create"}, none, false)
	test("an Update of a Person", map[string]any{"type": "Update", "object": map[string]any{"type": "Person", "name": "Author"}}, none, false)
	test("an Update of a Service", map[string]any{"type": "Update", "object": map[string]any{"type": "Service", "name": "Stream"}}, none, false)
	test("a Delete", map[string]any{"type": "Delete", "object": map[string]any{"type": "Tombstone", "name": "Gone"}}, none, false)
	test("an Announce of a boosted URL", map[string]any{"type": "Announce", "object": "https://remote.example/notes/1"}, none, false)
	test("an Announce of a Delete", map[string]any{"type": "Announce", "object": map[string]any{"type": "Delete", "object": "https://remote.example/notes/1"}}, none, false)
	test("an Announce of a Create of a URL", map[string]any{"type": "Announce", "object": map[string]any{"type": "Create", "object": "https://remote.example/notes/1"}}, none, false)
	test("an Undo of a Boost", map[string]any{"type": "Undo", "object": map[string]any{"type": "Announce", "object": note}}, none, false)
	test("a Block", map[string]any{"type": "Block", "object": "https://remote.example/@someone"}, none, false)
	test("an Accept", map[string]any{"type": "Accept", "object": map[string]any{"type": "Follow"}}, none, false)
	test("a Like", map[string]any{"type": "Like", "object": note}, none, false)
	test("no type", map[string]any{"object": note}, none, false)
	test("an empty activity", map[string]any{}, none, false)
	test("a nil activity", nil, none, false)
}

// followerActivityTestFollower returns an email Follower whose address and name must never reach an error
func followerActivityTestFollower() model.Follower {

	follower := model.NewFollower()
	follower.ParentType = model.FollowerTypeUser
	follower.ParentID = primitive.NewObjectID()
	follower.Method = model.FollowerMethodEmail
	follower.Actor.Name = "Sarah Connor"
	follower.Actor.EmailAddress = "sarah@connor.example"

	return follower
}

// renderFollowerActivity renders the shipped follower-activity body and subject for one activity
func renderFollowerActivity(t *testing.T, activity mapof.Any, actor model.PersonLink) (string, string) {

	t.Helper()

	email := loadEmbeddedEmail(t, "email-follower-activity", "follower-activity")
	follower := followerActivityTestFollower()

	post, isPost := followerPost(activity)
	require.True(t, isPost, "the activity must bring a post")

	values := followerActivityValues("https://example.com", &follower, actor, post)
	values["Domain_Name"] = "Example"
	values["Domain_Icon"] = ""

	var body strings.Builder
	require.NoError(t, email.Body.Execute(&body, values))

	var subject strings.Builder
	require.NoError(t, email.Subject.Execute(&subject, values))

	return body.String(), subject.String()
}

// TestFollowerActivity_RendersEveryShape renders the shipped email for every kind of activity that
// is emailed, and checks that each shows its post, its link, and its author
func TestFollowerActivity_RendersEveryShape(t *testing.T) {

	// BUG-372: the body piped the object's `content` into htmlMinimal, which stops on a missing
	// value, so a post with no body failed to render for every subscriber
	author := model.PersonLink{Name: "Ben Pate", IconURL: "https://example.com/@ben/pub/avatar", ProfileURL: "https://example.com/@ben"}

	// test renders one activity and checks what the email shows
	test := func(name string, activity mapof.Any, expected ...string) {
		t.Run(name, func(t *testing.T) {
			body, subject := renderFollowerActivity(t, activity, author)

			require.Equal(t, "New Activity From Ben Pate", subject)
			require.Contains(t, body, "Ben Pate")
			require.Contains(t, body, `src="https://example.com/@ben/pub/avatar"`)
			require.NotContains(t, body, `href=""`)
			require.NotContains(t, body, "Sarah Connor", "the subscriber is not the author")

			for _, value := range expected {
				require.Contains(t, body, value)
			}
		})
	}

	article := mapof.Any{"id": "https://example.com/1", "type": "Article", "name": "Our new song", "content": "<p>Have a listen</p>"}

	test("a Create with a name and content", mapof.Any{"type": "Create", "object": article}, "Our new song", "Have a listen", `href="https://example.com/1"`)
	test("an Update", mapof.Any{"type": "Update", "object": article}, "Our new song", "Have a listen")
	test("a Stream's announcement of a child", mapof.Any{"type": "Announce", "object": mapof.Any{"type": "Create", "object": article}}, "Our new song", "Have a listen")
	test("a Stream with no body", mapof.Any{"type": "Create", "object": mapof.Any{"id": "https://example.com/2", "type": "Event", "name": "Tour date"}}, "Tour date", `href="https://example.com/2"`)
	test("a Note with no name", mapof.Any{"type": "Create", "object": mapof.Any{"id": "https://example.com/3", "type": "Note", "content": "Just a thought"}}, "Just a thought", `href="https://example.com/3"`)
}

// TestFollowerActivity_RendersWithoutAnIcon verifies that an author with no icon renders no empty image
func TestFollowerActivity_RendersWithoutAnIcon(t *testing.T) {

	activity := mapof.Any{"type": "Create", "object": mapof.Any{"id": "https://example.com/1", "type": "Note", "content": "Hello"}}
	body, _ := renderFollowerActivity(t, activity, model.PersonLink{Name: "Ben Pate"})

	require.NotContains(t, body, `src=""`)
	require.Contains(t, body, "Ben Pate")
}

// parentActorStub returns a fixed actor, or fails, and counts its calls
type parentActorStub struct {
	actor model.PersonLink
	err   error
	calls int
}

// LoadParentActor returns the stub's actor or error
func (stub *parentActorStub) LoadParentActor(_ data.Session, _ *model.Follower) (model.PersonLink, error) {
	stub.calls++
	return stub.actor, stub.err
}

// newFollowerActivityTestService returns a DomainEmail that holds the shipped follower-activity
// email and has no SMTP connection, so every send fails
func newFollowerActivityTestService(t *testing.T, parentActors parentActorLoader) DomainEmail {

	t.Helper()

	serverEmail := testServerEmail()
	serverEmail.emails["follower-activity"] = loadEmbeddedEmail(t, "email-follower-activity", "follower-activity")

	return DomainEmail{
		serverEmail:   &serverEmail,
		domainService: &Domain{},
		parentActors:  parentActors,
	}
}

// TestSendFollowerActivity_SendsNothingWithoutAPost verifies that an activity with no post sends
// nothing, and loads nothing
func TestSendFollowerActivity_SendsNothingWithoutAPost(t *testing.T) {

	stub := &parentActorStub{}
	service := newFollowerActivityTestService(t, stub)
	follower := followerActivityTestFollower()

	// The send would fail, because no SMTP connection is configured, so a nil error proves nothing was sent
	for _, activity := range []mapof.Any{
		{"type": "Delete", "object": mapof.Any{"id": "https://example.com/1", "type": "Tombstone"}},
		{"type": "Undo", "object": mapof.Any{"type": "Announce", "object": "https://remote.example/1"}},
		{"type": "Announce", "object": "https://remote.example/1"},
		{"type": "Create", "object": mapof.Any{"type": "Note"}},
	} {
		require.NoError(t, service.SendFollowerActivity(nil, &follower, activity))
	}

	require.Zero(t, stub.calls)
}

// TestSendFollowerActivity_ErrorNamesNoSubscriber verifies that a failed send names the Follower by
// id, and never carries the subscriber's address or name
func TestSendFollowerActivity_ErrorNamesNoSubscriber(t *testing.T) {

	// BUG-372 Defect B: the error carried the subscriber's address, and its caller attached the
	// whole Follower, so both were stored in ErrorLog
	follower := followerActivityTestFollower()
	activity := mapof.Any{"type": "Create", "object": mapof.Any{"id": "https://example.com/1", "type": "Note", "content": "Hello"}}

	// requireNoSubscriber checks one error for the subscriber's address and name
	requireNoSubscriber := func(t *testing.T, err error) {
		t.Helper()
		require.Error(t, err)
		secretcheck.RequireAbsent(t, err, "sarah@connor.example")
		secretcheck.RequireAbsent(t, err, "Sarah Connor")
	}

	t.Run("a send that fails", func(t *testing.T) {
		stub := &parentActorStub{actor: model.PersonLink{Name: "Ben Pate"}}
		service := newFollowerActivityTestService(t, stub)
		requireNoSubscriber(t, service.SendFollowerActivity(nil, &follower, activity))
		require.Equal(t, 1, stub.calls)
	})

	t.Run("an actor that cannot be loaded", func(t *testing.T) {
		service := newFollowerActivityTestService(t, &parentActorStub{err: derp.Internal("test", "Synthetic failure")})
		requireNoSubscriber(t, service.SendFollowerActivity(nil, &follower, activity))
	})

	t.Run("a confirmation that fails", func(t *testing.T) {
		serverEmail := testServerEmail()
		serverEmail.emails["follower-confirmation"] = loadEmbeddedEmail(t, "email-follower-confirmation", "follower-confirmation")
		service := DomainEmail{serverEmail: &serverEmail, domainService: &Domain{}}
		requireNoSubscriber(t, service.SendFollowerConfirmation(model.PersonLink{Name: "Ben Pate"}, &follower))
	})
}

// FuzzFollowerPost verifies that any JSON activity is read without panicking, that a post always has
// something to show, and that every post renders in the shipped email
func FuzzFollowerPost(f *testing.F) {

	f.Add(`{"type":"Create","object":{"id":"https://example.com/1","type":"Note","content":"Hello"}}`)
	f.Add(`{"type":"Announce","object":{"type":"Create","object":{"type":"Article","name":"Song"}}}`)
	f.Add(`{"type":"Create","object":{"type":"Note","content":null,"name":{"en":"x"}}}`)
	f.Add(`{"type":"Update","object":{"type":"Person","name":"Author"}}`)
	f.Add(`{"type":"Create","object":{"type":"Note","content":"<script>alert(1)</script>"}}`)
	f.Add(`{"type":["Create","Update"],"object":[{"name":"a"},{"name":"b"}]}`)

	email := loadEmbeddedEmail(f, "email-follower-activity", "follower-activity")
	follower := followerActivityTestFollower()

	f.Fuzz(func(t *testing.T, document string) {

		activity := mapof.NewAny()

		if err := json.Unmarshal([]byte(document), &activity); err != nil {
			return
		}

		post, isPost := followerPost(activity)

		if !isPost {
			require.Equal(t, followerEmailPost{}, post)
			return
		}

		require.True(t, (post.Title != "") || (post.ContentHTML != ""))

		values := followerActivityValues("https://example.com", &follower, model.PersonLink{}, post)
		values["Domain_Name"] = "Example"
		values["Domain_Icon"] = ""

		var body strings.Builder
		require.NoError(t, email.Body.Execute(&body, values))
		require.NotContains(t, body.String(), "<script>")
	})
}
