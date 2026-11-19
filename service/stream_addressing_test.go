package service

import (
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/tools/id"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/sliceof"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// newAddressingService returns a Stream service that can build addressing on https://example.com
func newAddressingService() *Stream {
	return &Stream{locatorService: &Locator{host: "https://example.com"}}
}

// newAddressingTemplate returns a Template whose default action is open to the author and to viewers once published
func newAddressingTemplate() *model.Template {

	template := model.NewTemplate("test-addressing", nil)
	template.DefaultAction = "view"
	template.Actions = mapof.Object[model.Action]{"view": {
		Roles:      sliceof.String{"author", "viewer"},
		AccessList: mapof.Object[sliceof.String]{"published": {"author", "viewer"}},
	}}

	return &template
}

// newSharedStream returns a published Stream by authorID, shared as the setup function says, with
// DefaultAllow calculated by the same code that runs on every save
func newSharedStream(service *Stream, authorID primitive.ObjectID, setup func(*model.Stream)) model.Stream {

	stream := model.NewStream()
	stream.StreamID = primitive.NewObjectID()
	stream.StateID = "published"
	stream.AttributedTo.UserID = authorID

	setup(&stream)
	service.calcDefaultAllow(newAddressingTemplate(), &stream)

	return stream
}

// requireAgreement asserts that addressing names the audience that DefaultAllow grants: Public and the
// sender's followers exactly when anonymous visitors may view it, and the circles that may view it
func requireAgreement(t *testing.T, service *Stream, stream *model.Stream, actorType string, actorID primitive.ObjectID, to []string, cc []string) {

	t.Helper()

	isPublic := stream.DefaultAllow.IsAnonymous()
	followersURL := service.locatorService.FollowersURL(actorType, actorID)

	require.Equal(t, isPublic, sliceof.String(to).Contains(vocab.NamespaceActivityStreamsPublic), "Public must be named exactly when anonymous visitors may view")
	require.Equal(t, isPublic, sliceof.String(cc).Contains(followersURL), "followers must be named exactly when the Stream is public")

	expectedCircles := make([]string, 0)

	if (!isPublic) && (actorType == model.ActorTypeUser) {
		for _, permissionID := range stream.DefaultAllow {
			if stream.CircleIDs().Contains(permissionID) {
				expectedCircles = append(expectedCircles, service.locatorService.CircleURL(actorID, permissionID))
			}
		}
	}

	namedCircles := make([]string, 0)

	for _, recipient := range to {
		if _, _, isCircle := service.locatorService.ParseCircle(recipient); isCircle {
			namedCircles = append(namedCircles, recipient)
		}
	}

	require.ElementsMatch(t, expectedCircles, namedCircles, "the circles named must be the circles that may view")
}

// TestStreamAddressing verifies the addressing that each sharing setting produces, and that it agrees with DefaultAllow
func TestStreamAddressing(t *testing.T) {

	service := newAddressingService()
	userID := primitive.NewObjectID()
	parentID := primitive.NewObjectID()
	firstCircle := primitive.NewObjectID()
	secondCircle := primitive.NewObjectID()

	userFollowers := "https://example.com/@" + userID.Hex() + "/pub/followers"
	parentFollowers := "https://example.com/" + parentID.Hex() + "/pub/followers"
	firstCircleURL := "https://example.com/@" + userID.Hex() + "/pub/circles/" + firstCircle.Hex()
	secondCircleURL := "https://example.com/@" + userID.Hex() + "/pub/circles/" + secondCircle.Hex()

	// test shares a Stream, addresses it as one actor, and checks the result and its agreement with DefaultAllow
	test := func(name string, setup func(*model.Stream), actorType string, actorID primitive.ObjectID, expectedTo []string, expectedCC []string) {
		t.Run(name, func(t *testing.T) {
			stream := newSharedStream(service, userID, setup)
			to, cc := service.Addressing(&stream, actorType, actorID)

			require.Equal(t, expectedTo, to)
			require.Equal(t, expectedCC, cc)
			requireAgreement(t, service, &stream, actorType, actorID, to, cc)
		})
	}

	public := func(stream *model.Stream) {
		stream.Groups["viewer"] = id.Slice{model.MagicGroupIDAnonymous}
	}

	circles := func(stream *model.Stream) {
		stream.Circles["viewer"] = id.Slice{firstCircle, secondCircle}
	}

	test("public, from its author", public, model.ActorTypeUser, userID,
		[]string{vocab.NamespaceActivityStreamsPublic}, []string{userFollowers})

	test("public, from its parent Stream", public, model.ActorTypeStream, parentID,
		[]string{vocab.NamespaceActivityStreamsPublic}, []string{parentFollowers})

	test("circles, from its author", circles, model.ActorTypeUser, userID,
		[]string{firstCircleURL, secondCircleURL}, []string{})

	test("circles, from its parent Stream", circles, model.ActorTypeStream, parentID,
		[]string{}, []string{})

	test("public and circles, where public wins", func(stream *model.Stream) {
		stream.Groups["viewer"] = id.Slice{model.MagicGroupIDAnonymous}
		stream.Circles["viewer"] = id.Slice{firstCircle}
	}, model.ActorTypeUser, userID,
		[]string{vocab.NamespaceActivityStreamsPublic}, []string{userFollowers})

	test("signed-in visitors", func(stream *model.Stream) {
		stream.Groups["viewer"] = id.Slice{model.MagicGroupIDAuthenticated}
	}, model.ActorTypeUser, userID,
		[]string{}, []string{})

	test("owners only", func(stream *model.Stream) {
		stream.Groups["viewer"] = id.Slice{model.MagicGroupIDOwners}
	}, model.ActorTypeUser, userID,
		[]string{}, []string{})

	test("not shared", func(stream *model.Stream) {
		stream.Groups["viewer"] = id.NewSlice()
		stream.Circles["viewer"] = id.NewSlice()
	}, model.ActorTypeUser, userID,
		[]string{}, []string{})

	test("a site group", func(stream *model.Stream) {
		stream.Groups["viewer"] = id.Slice{primitive.NewObjectID()}
	}, model.ActorTypeUser, userID,
		[]string{}, []string{})

	test("a product", func(stream *model.Stream) {
		stream.Products["viewer"] = id.Slice{primitive.NewObjectID()}
	}, model.ActorTypeUser, userID,
		[]string{}, []string{})

	test("a circle granted a role that cannot view", func(stream *model.Stream) {
		stream.Circles["editor"] = id.Slice{firstCircle}
	}, model.ActorTypeUser, userID,
		[]string{}, []string{})
}

// TestStreamAddressing_CircleOnlyNamesNoFollowers verifies that a circle-only Stream names neither Public nor any followers collection
func TestStreamAddressing_CircleOnlyNamesNoFollowers(t *testing.T) {

	service := newAddressingService()
	userID := primitive.NewObjectID()

	stream := newSharedStream(service, userID, func(stream *model.Stream) {
		stream.Circles["viewer"] = id.Slice{primitive.NewObjectID()}
	})

	to, cc := service.Addressing(&stream, model.ActorTypeUser, userID)

	for _, recipient := range append(to, cc...) {
		require.False(t, isPublicURI(recipient), "a circle-only Stream named Public: %s", recipient)

		actorType, _ := parseFollowersURI(service.locatorService.host, recipient)
		require.Empty(t, actorType, "a circle-only Stream named a followers collection: %s", recipient)
	}
}

// TestStreamAddressing_NoDefaultAllow verifies that a Stream whose DefaultAllow was never calculated names no one
func TestStreamAddressing_NoDefaultAllow(t *testing.T) {

	service := newAddressingService()
	stream := model.NewStream()
	stream.Circles["viewer"] = id.Slice{primitive.NewObjectID()}

	to, cc := service.Addressing(&stream, model.ActorTypeUser, primitive.NewObjectID())

	require.Empty(t, to)
	require.Empty(t, cc)
}

// TestStreamAddressing_NoAuthor verifies that a Stream with no author names Public when it is public, and nothing else
func TestStreamAddressing_NoAuthor(t *testing.T) {

	service := newAddressingService()

	public := newSharedStream(service, primitive.NilObjectID, func(stream *model.Stream) {
		stream.Groups["viewer"] = id.Slice{model.MagicGroupIDAnonymous}
	})

	to, cc := service.Addressing(&public, model.ActorTypeUser, primitive.NilObjectID)
	require.Equal(t, []string{vocab.NamespaceActivityStreamsPublic}, to)
	require.Empty(t, cc, "a zero ID has no followers collection")

	circleOnly := newSharedStream(service, primitive.NilObjectID, func(stream *model.Stream) {
		stream.Circles["viewer"] = id.Slice{primitive.NewObjectID()}
	})

	to, cc = service.Addressing(&circleOnly, model.ActorTypeUser, primitive.NilObjectID)
	require.Empty(t, to, "a zero ID owns no circles")
	require.Empty(t, cc)
}

// TestStreamAddressing_UnknownActorType verifies that a public Stream sent by an actor with no followers collection names Public alone
func TestStreamAddressing_UnknownActorType(t *testing.T) {

	service := newAddressingService()
	stream := newSharedStream(service, primitive.NewObjectID(), func(stream *model.Stream) {
		stream.Groups["viewer"] = id.Slice{model.MagicGroupIDAnonymous}
	})

	to, cc := service.Addressing(&stream, model.ActorTypeSearchQuery, primitive.NewObjectID())

	require.Equal(t, []string{vocab.NamespaceActivityStreamsPublic}, to)
	require.Empty(t, cc)
}
