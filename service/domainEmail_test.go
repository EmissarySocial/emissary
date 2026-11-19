package service

import (
	"strings"
	"testing"

	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/require"
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

// TestFollowerActivity_ShowsAnnouncedPost renders the shipped email for a Stream's announcement of
// a new child, which must show the child's content and not the Create that wraps it
func TestFollowerActivity_ShowsAnnouncedPost(t *testing.T) {

	email := loadEmbeddedEmail(t, "email-follower-activity", "follower-activity")

	activity := mapof.Any{
		"type": "Announce",
		"object": mapof.Any{
			"type":   "Create",
			"object": mapof.Any{"type": "Article", "name": "Our new song", "content": "<p>Have a listen</p>"},
		},
	}

	var body strings.Builder
	require.NoError(t, email.Body.Execute(&body, mapof.Any{"Activity": activity, "Object": followerEmailObject(activity)}))

	require.Contains(t, body.String(), "Our new song")
	require.Contains(t, body.String(), "Have a listen")
}
