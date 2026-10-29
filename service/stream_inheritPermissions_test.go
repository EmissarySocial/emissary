package service

import (
	"context"
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/tools/id"
	mockdb "github.com/benpate/data-mock"
	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// TestApplyInheritedPermissions pins that a child takes its parent's Groups, Circles, and Products, replacing its
// own, and that the copy shares no slices with the parent, so editing one never edits the other.
// (inheritPermissions loads the parent first; data-mock cannot match a Stream's deleteDate, which sits in
// the embedded Journal, so the load is not exercised here.)
func TestApplyInheritedPermissions(t *testing.T) {

	circleID := primitive.NewObjectID()
	productID := primitive.NewObjectID()

	parent := model.NewStream()
	parent.Groups = mapof.Object[id.Slice]{"viewer": {model.MagicGroupIDAnonymous}}
	parent.Circles = mapof.Object[id.Slice]{"viewer": {circleID}}
	parent.Products = mapof.Object[id.Slice]{"viewer": {productID}}

	child := model.NewStream()
	child.Circles = mapof.Object[id.Slice]{"viewer": {primitive.NewObjectID()}, "editor": {primitive.NewObjectID()}}

	applyInheritedPermissions(&child, &parent)

	require.Equal(t, mapof.Object[id.Slice]{"viewer": {circleID}}, child.Circles, "the child's own roles are replaced")
	require.Equal(t, mapof.Object[id.Slice]{"viewer": {productID}}, child.Products)
	require.Equal(t, mapof.Object[id.Slice]{"viewer": {model.MagicGroupIDAnonymous}}, child.Groups, "Share with Everyone lives in Groups")

	child.Circles["viewer"][0] = primitive.NewObjectID()
	require.Equal(t, circleID, parent.Circles["viewer"][0], "the child must not share the parent's slice")
}

// TestApplyInheritedPermissions_ReplyPolicy pins that a child takes its parent's reply policy both ways,
// so closing replies on a collection closes them on its items, and reopening reopens them.
func TestApplyInheritedPermissions_ReplyPolicy(t *testing.T) {

	parent := model.NewStream()
	parent.ReplyPolicy = model.ReplyPolicyNobody

	child := model.NewStream()
	applyInheritedPermissions(&child, &parent)
	require.Equal(t, model.ReplyPolicyNobody, child.ReplyPolicy)

	parent.ReplyPolicy = ""
	applyInheritedPermissions(&child, &parent)
	require.Equal(t, "", child.ReplyPolicy, "an open parent reopens a closed child")
}

// TestStreamInheritPermissions_NoParentStream pins that a Stream with no parent Template recorded (a User's
// outbox item, or a top-level Stream) keeps its own sharing, without reading any parent.
func TestStreamInheritPermissions_NoParentStream(t *testing.T) {

	session, err := mockdb.New().Session(context.Background())
	require.Nil(t, err)

	circleID := primitive.NewObjectID()

	child := model.NewStream()
	child.ParentID = primitive.NewObjectID()
	child.Circles = mapof.Object[id.Slice]{"viewer": {circleID}}

	service := NewStream()
	require.Nil(t, service.inheritPermissions(session, &child))
	require.Equal(t, id.Slice{circleID}, child.Circles["viewer"])

	topLevel := model.NewStream()
	require.Nil(t, service.inheritPermissions(session, &topLevel))
}

// TestSameSharing pins which differences push a parent's sharing down: any change to Groups, Circles,
// Products, or ReplyPolicy does, and a role that is missing compares equal to one that is empty.
func TestSameSharing(t *testing.T) {

	circleID := primitive.NewObjectID()

	newParent := func() model.Stream {
		result := model.NewStream()
		result.Groups = mapof.Object[id.Slice]{"viewer": {model.MagicGroupIDAnonymous}}
		result.Circles = mapof.Object[id.Slice]{"viewer": {circleID}}
		return result
	}

	a, b := newParent(), newParent()
	require.True(t, sameSharing(&a, &b))

	// A missing role is the same as an empty one
	b.Products = mapof.Object[id.Slice]{"viewer": {}}
	require.True(t, sameSharing(&a, &b))

	// Each of the four fields is compared
	b = newParent()
	b.Circles["viewer"] = id.Slice{primitive.NewObjectID()}
	require.False(t, sameSharing(&a, &b), "a different circle")

	b = newParent()
	b.Groups = mapof.Object[id.Slice]{}
	require.False(t, sameSharing(&a, &b), "no longer shared with everyone")

	b = newParent()
	b.Products = mapof.Object[id.Slice]{"viewer": {primitive.NewObjectID()}}
	require.False(t, sameSharing(&a, &b), "a new product")

	b = newParent()
	b.ReplyPolicy = model.ReplyPolicyNobody
	require.False(t, sameSharing(&a, &b), "replies closed")
}

// TestStreamSharingChanged_SkipsWithoutAComparison pins that a new Stream, or one whose Template does not
// share with children, never reads its saved copy and never pushes.
func TestStreamSharingChanged_SkipsWithoutAComparison(t *testing.T) {

	session, err := mockdb.New().Session(context.Background())
	require.Nil(t, err)

	service := NewStream()
	stream := model.NewStream()

	changed, err := service.sharingChanged(session, &model.Template{}, &stream, false)
	require.Nil(t, err)
	require.False(t, changed, "a Template without shareWithChildren")

	changed, err = service.sharingChanged(session, &model.Template{ShareWithChildren: true}, &stream, true)
	require.Nil(t, err)
	require.False(t, changed, "a new Stream has no children")
}
