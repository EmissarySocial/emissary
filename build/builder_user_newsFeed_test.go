package build

import (
	"encoding/json"
	"strconv"
	"sync"
	"testing"

	"github.com/benpate/derp"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
	"github.com/stretchr/testify/require"
)

const (
	replyTestAuthor    = "https://social.example.com/users/author"
	replyTestFollowers = "https://social.example.com/users/author/followers"
	replyTestFollowing = "https://social.example.com/users/author/following"
	replyTestUser      = "https://emissary.example.com/@me"
)

// replyTestClient serves the author's actor document, and counts every load
type replyTestClient struct {
	mutex sync.Mutex
	loads map[string]int
}

// newReplyTestClient returns a replyTestClient that has loaded nothing
func newReplyTestClient() *replyTestClient {
	return &replyTestClient{loads: make(map[string]int)}
}

// SetRootClient is a no-op
func (client *replyTestClient) SetRootClient(streams.Client) {}

// Load returns the author's actor document, and 404 for everything else
func (client *replyTestClient) Load(url string, _ ...any) (streams.Document, error) {

	client.mutex.Lock()
	client.loads[url]++
	client.mutex.Unlock()

	if url != replyTestAuthor {
		return streams.NilDocument(), derp.NotFound("build.replyTestClient.Load", "Not found", url)
	}

	return streams.NewDocument(map[string]any{
		vocab.PropertyID:        replyTestAuthor,
		vocab.PropertyType:      vocab.ActorTypePerson,
		vocab.PropertyFollowers: replyTestFollowers,
		vocab.PropertyFollowing: replyTestFollowing,
	}, streams.WithClient(client)), nil
}

// Save is a no-op
func (client *replyTestClient) Save(streams.Document) error { return nil }

// Delete is a no-op
func (client *replyTestClient) Delete(string) error { return nil }

// authorLoads returns how many times the author was loaded
func (client *replyTestClient) authorLoads() int {
	client.mutex.Lock()
	defer client.mutex.Unlock()
	return client.loads[replyTestAuthor]
}

// replyTestPost returns a post by the author, given as a bare IRI the way Mastodon sends it, with the given interactionPolicy
func replyTestPost(t *testing.T, client *replyTestClient, policy string) streams.Document {

	t.Helper()

	post, err := decodeReplyTestPost(client, policy)
	require.NoError(t, err)
	return post
}

// decodeReplyTestPost decodes the post that replyTestPost returns, or the error JSON gives for this policy
func decodeReplyTestPost(client *replyTestClient, policy string) (streams.Document, error) {

	value := map[string]any{}
	err := json.Unmarshal([]byte(`{
		"id": "https://social.example.com/notes/1",
		"type": "Note",
		"attributedTo": "`+replyTestAuthor+`",
		"tag": [{"type": "Mention", "href": "https://other.example.com/users/mentioned"}],
		"interactionPolicy": `+policy+`
	}`), &value)

	return streams.NewDocument(value, streams.WithClient(client)), err
}

// replyTestCounter returns a follow check that answers as given, and counts its calls
func replyTestCounter(answer bool) (func(string) bool, *int) {
	calls := 0
	return func(authorID string) bool {
		calls++
		return answer && (authorID == replyTestAuthor)
	}, &calls
}

// TestNewReplyRequest_Decides pins the request's fields, and the decision each follow relationship gets
func TestNewReplyRequest_Decides(t *testing.T) {

	followersOnly := `{"canReply": {"automaticApproval": ["` + replyTestFollowers + `"]}}`
	followingOnly := `{"canReply": {"automaticApproval": ["` + replyTestFollowing + `"]}}`

	decide := func(policy string, follows bool, followed bool, actorID string) streams.InteractionResult {
		post := replyTestPost(t, newReplyTestClient(), policy)
		followsAuthor, _ := replyTestCounter(follows)
		followedByAuthor, _ := replyTestCounter(followed)
		request := newReplyRequest(post, actorID, followsAuthor, followedByAuthor)
		return post.InteractionPolicy().CanInteract(vocab.PropertyCanReply, request)
	}

	require.Equal(t, streams.InteractionAllowed, decide(followersOnly, true, false, replyTestUser), "a follower")
	require.Equal(t, streams.InteractionDenied, decide(followersOnly, false, true, replyTestUser), "followed, but not a follower")
	require.Equal(t, streams.InteractionAllowed, decide(followingOnly, false, true, replyTestUser), "followed by the author")
	require.Equal(t, streams.InteractionDenied, decide(followingOnly, true, false, replyTestUser), "a follower, but not followed")
	require.Equal(t, streams.InteractionAllowed, decide(followersOnly, false, false, "https://other.example.com/users/mentioned"), "a mentioned actor")
	require.Equal(t, streams.InteractionAllowed, decide(`{}`, false, false, replyTestUser), "no policy")

	request := newReplyRequest(replyTestPost(t, newReplyTestClient(), `{}`), replyTestUser, nil, nil)
	require.Equal(t, replyTestUser, request.ActorID)
	require.Equal(t, replyTestAuthor, request.AuthorID)
	require.Equal(t, []string{"https://other.example.com/users/mentioned"}, request.Mentioned)
}

// TestNewReplyRequest_LooksUpOnce pins that a policy naming many IRIs, and one IRI many times, loads the
// author at most once and runs each follow check at most once
func TestNewReplyRequest_LooksUpOnce(t *testing.T) {

	automatic := []string{}
	for index := range 50 {
		automatic = append(automatic, "https://elsewhere.example.com/collections/"+strconv.Itoa(index))
	}
	automatic = append(automatic, replyTestFollowers, replyTestFollowers, replyTestFollowing, replyTestFollowing)

	encoded, err := json.Marshal(map[string]any{vocab.PropertyCanReply: map[string]any{vocab.PropertyAutomaticApproval: automatic}})
	require.NoError(t, err)

	client := newReplyTestClient()
	post := replyTestPost(t, client, string(encoded))
	followsAuthor, followsCalls := replyTestCounter(false)
	followedByAuthor, followedCalls := replyTestCounter(false)

	policy := post.InteractionPolicy()
	request := newReplyRequest(post, replyTestUser, followsAuthor, followedByAuthor)

	// Ask twice: the second answer comes from memory
	require.Equal(t, streams.InteractionDenied, policy.CanInteract(vocab.PropertyCanReply, request))
	require.Equal(t, streams.InteractionDenied, policy.CanInteract(vocab.PropertyCanReply, request))

	require.Equal(t, 1, client.authorLoads(), "the author is loaded once")
	require.Equal(t, 1, *followsCalls, "FollowsAuthor runs once")
	require.Equal(t, 1, *followedCalls, "FollowedByAuthor runs once")
}

// TestNewReplyRequest_LoadsNothingWhenOpen pins that a policy needing no collections loads nothing
func TestNewReplyRequest_LoadsNothingWhenOpen(t *testing.T) {

	for _, policy := range []string{
		`{}`,
		`{"canReply": {"automaticApproval": ["https://www.w3.org/ns/activitystreams#Public"]}}`,
		`{"canReply": {"automaticApproval": ["` + replyTestAuthor + `"]}}`,
		`{"canReply": {"automaticApproval": ["` + replyTestUser + `"]}}`,
	} {
		client := newReplyTestClient()
		post := replyTestPost(t, client, policy)
		followsAuthor, followsCalls := replyTestCounter(true)

		_ = post.InteractionPolicy().CanInteract(vocab.PropertyCanReply, newReplyRequest(post, replyTestUser, followsAuthor, followsAuthor))

		require.Zero(t, client.authorLoads(), policy)
		require.Zero(t, *followsCalls, policy)
	}
}

// FuzzNewReplyRequest decides arbitrary policies, and confirms that no policy can load the author more than once,
// or run either follow check more than once
func FuzzNewReplyRequest(f *testing.F) {

	f.Add(`{"canReply": {"automaticApproval": ["`+replyTestFollowers+`", "`+replyTestFollowers+`"]}}`, true)
	f.Add(`{"canReply": {"manualApproval": ["`+replyTestFollowing+`", "x", "y", "`+replyTestFollowing+`"]}}`, false)
	f.Add(`{"canReply": {"automaticApproval": [{"@id": "`+replyTestFollowers+`"}], "always": "z"}}`, true)
	f.Add(`"https://social.example.com/policy"`, false)
	f.Add(`[]`, true)

	f.Fuzz(func(t *testing.T, policy string, follows bool) {

		// Only a policy that decodes makes a post; JSON can be valid and still overflow a float64
		client := newReplyTestClient()
		post, err := decodeReplyTestPost(client, policy)

		if err != nil {
			return
		}
		followsAuthor, followsCalls := replyTestCounter(follows)
		followedByAuthor, followedCalls := replyTestCounter(follows)

		request := newReplyRequest(post, replyTestUser, followsAuthor, followedByAuthor)

		for _, permission := range []string{vocab.PropertyCanLike, vocab.PropertyCanReply, vocab.PropertyCanAnnounce, vocab.PropertyCanQuote} {
			_ = post.InteractionPolicy().CanInteract(permission, request)
		}

		require.LessOrEqual(t, client.authorLoads(), 1)
		require.LessOrEqual(t, *followsCalls, 1)
		require.LessOrEqual(t, *followedCalls, 1)
	})
}
