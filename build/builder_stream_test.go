package build

import (
	"net/http/httptest"
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/exp"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// TestStream_WidgetIDsByLocation asserts that the accessor yields exactly the string the widget
// editor's hidden inputs must carry: every widget in the location, in stored order, joined by
// commas, and nothing from any other location.
func TestStream_WidgetIDsByLocation(t *testing.T) {

	first := primitive.NewObjectID()
	second := primitive.NewObjectID()
	elsewhere := primitive.NewObjectID()

	stream := model.NewStream()
	stream.Widgets = model.NewStreamWidgets()
	stream.Widgets.Append(model.StreamWidget{StreamWidgetID: first, Type: "markdown", Location: "TOP"})
	stream.Widgets.Append(model.StreamWidget{StreamWidgetID: elsewhere, Type: "markdown", Location: "LEFT"})
	stream.Widgets.Append(model.StreamWidget{StreamWidgetID: second, Type: "markdown", Location: "TOP"})

	builder := Stream{_stream: &stream}

	require.Equal(t, first.Hex()+","+second.Hex(), builder.WidgetIDsByLocation("TOP"), "stored order, comma-joined")
	require.Equal(t, elsewhere.Hex(), builder.WidgetIDsByLocation("LEFT"), "a single widget has no separator")
	require.Equal(t, "", builder.WidgetIDsByLocation("RIGHT"), "an empty location is an empty string, not a stray comma")
	require.Equal(t, "", builder.WidgetIDsByLocation("NOWHERE"), "an unknown location is empty rather than an error")

	// A Stream that has never had a widget carries a nil slice, and must read the same as empty
	empty := model.Stream{}
	require.Equal(t, "", Stream{_stream: &empty}.WidgetIDsByLocation("TOP"))
}

// TestStream_ChildrenWithDrafts_Author verifies that a Stream's author sees every child, including
// unpublished ones, while the permission filter still applies (QWERTY-COLLECTIONS §1.2).
func TestStream_ChildrenWithDrafts_Author(t *testing.T) {

	authorID := primitive.NewObjectID()
	parent := newChildrenTestStream(authorID, authorID)

	predicates := collectPredicates(parent.ChildrenWithDrafts().criteria)

	parentPredicate, hasParent := findPredicate(predicates, "parentId", exp.OperatorEqual)
	require.True(t, hasParent, "must list only this Stream's children")
	require.Equal(t, parent._stream.StreamID, parentPredicate.Value)

	_, hasPermissions := findPredicate(predicates, "defaultAllow", exp.OperatorIn)
	require.True(t, hasPermissions, "the author still sees only the children their permissions allow")

	_, hasPublishDate := findPredicate(predicates, "publishDate", exp.OperatorLessThan)
	require.False(t, hasPublishDate, "the author must see unpublished children")

	_, hasUnpublishDate := findPredicate(predicates, "unpublishDate", exp.OperatorGreaterThan)
	require.False(t, hasUnpublishDate, "the author must see expired children")
}

// TestStream_ChildrenWithDrafts_NotAuthor verifies that anyone but the author gets exactly the
// published children that Children returns, so a template cannot leak a draft or an expired item
// by calling ChildrenWithDrafts in a public view.
func TestStream_ChildrenWithDrafts_NotAuthor(t *testing.T) {

	authorID := primitive.NewObjectID()

	for name, viewerID := range map[string]primitive.ObjectID{
		"anonymous":  primitive.NilObjectID,
		"other user": primitive.NewObjectID(),
	} {
		parent := newChildrenTestStream(authorID, viewerID)
		predicates := collectPredicates(parent.ChildrenWithDrafts().criteria)

		_, hasPermissions := findPredicate(predicates, "defaultAllow", exp.OperatorIn)
		require.True(t, hasPermissions, name)

		_, hasPublishDate := findPredicate(predicates, "publishDate", exp.OperatorLessThan)
		require.True(t, hasPublishDate, name+" must not see unpublished children")

		_, hasUnpublishDate := findPredicate(predicates, "unpublishDate", exp.OperatorGreaterThan)
		require.True(t, hasUnpublishDate, name+" must not see expired children")

		require.Equal(t, len(collectPredicates(parent.Children().criteria)), len(predicates), name+" gets exactly Children's criteria")
	}
}

// TestStream_Children_HidesDrafts pins that Children hides unpublished and expired children from
// everyone, the author included, which is why ChildrenWithDrafts exists.
func TestStream_Children_HidesDrafts(t *testing.T) {

	authorID := primitive.NewObjectID()
	parent := newChildrenTestStream(authorID, authorID)
	predicates := collectPredicates(parent.Children().criteria)

	_, hasPublishDate := findPredicate(predicates, "publishDate", exp.OperatorLessThan)
	require.True(t, hasPublishDate)

	_, hasUnpublishDate := findPredicate(predicates, "unpublishDate", exp.OperatorGreaterThan)
	require.True(t, hasUnpublishDate)
}

// newChildrenTestStream returns a Stream builder for a Stream written by authorID and viewed by
// viewerID, with only what the child queries read.  No query runs, so no database is needed.
func newChildrenTestStream(authorID primitive.ObjectID, viewerID primitive.ObjectID) Stream {

	stream := model.NewStream()
	stream.AttributedTo.UserID = authorID

	return Stream{
		_stream: &stream,
		CommonWithTemplate: CommonWithTemplate{
			Common: Common{
				_factory:       stubPermissionFactory{},
				_request:       httptest.NewRequest("GET", "/", nil),
				_authorization: model.Authorization{UserID: viewerID},
			},
		},
	}
}
