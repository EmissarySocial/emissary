package mongodb

import (
	"context"
	"testing"

	"github.com/benpate/data/journal"
	"github.com/benpate/exp"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type testStream struct {
	StreamID        primitive.ObjectID     `json:"streamId"        bson:"_id"`                 // Unique identifier of this Stream.  (NOT USED PUBLICLY)
	ParentID        primitive.ObjectID     `json:"parentId"        bson:"parentId"`            // Unique identifier of the "parent" stream. (NOT USED PUBLICLY)
	TemplateID      string                 `json:"templateId"      bson:"templateId"`          // Unique identifier (name) of the Template to use when rendering this Stream in HTML.
	StateID         string                 `json:"stateId"         bson:"stateId"`             // Unique identifier of the State this Stream is in.  This is used to populate the State information from the Template service at load time.
	GroupRoles      map[string][]string    `json:"groupRoles"      bson:"groupRoles"`          // Map of Role names to the one or more Group names that can perform that role.
	Token           string                 `json:"token"           bson:"token"`               // Unique value that identifies this element in the URL
	URL             string                 `json:"url"             bson:"url"`                 // Unique URL of this Stream.  This duplicates the "token" field a bit, but it (hopefully?) makes access easier.
	Label           string                 `json:"label"           bson:"label"`               // Text to display in lists of streams, probably displayed at top of stream page, too.
	Description     string                 `json:"description"     bson:"description"`         // Brief summary of this stream, used in lists of streams
	ThumbnailImage  string                 `json:"thumbnailImage"  bson:"thumbnailImage"`      // Image to display next to the stream in lists.
	AuthorID        primitive.ObjectID     `json:"authorId"        bson:"authorId"`            // Unique identifier of the person who created this stream (NOT USED PUBLICLY)
	AuthorName      string                 `json:"authorName"      bson:"authorName"`          // Full name of the person who created this stream
	AuthorImage     string                 `json:"authorImage"     bson:"authorImage"`         // URL of an image to use for the person who created this stream
	AuthorURL       string                 `json:"authorURL"       bson:"authorURL"`           // URL address of the person who created this stream
	Tags            []string               `json:"tags"            bson:"tags"`                // Organizational Tags
	Data            map[string]interface{} `json:"data"            bson:"data"`                // Set of data to populate into the Template.  This is validated by the JSON-Schema of the Template.
	BubbleUpdates   bool                   `json:"bubbleUpdates"   bson:"bubbleUpdates"`       // If TRUE then updates are sent to the PARENT, instead of THIS stream.  This *should* be controlled by the Template.
	SourceID        primitive.ObjectID     `json:"sourceId"        bson:"sourceId,omitempty"`  // Internal identifier of the source configuration that generated this stream
	SourceURL       string                 `json:"sourceURL"       bson:"sourceURL,omitempty"` // URL of the original document published by the source server
	PublishDate     int64                  `json:"publishDate"     bson:"publishDate"`         // Unix timestamp of the date/time when this document is/was/will be first available on the domain.
	UnPublishDate   int64                  `json:"unpublishDate"   bson:"unpublishDate"`       // Unix timestemp of the date/time when this document will no longer be available on the domain.
	Rank            int                    `json:"Rank"            bson:"Rank"`                // Rank allows for a manual sort of streams
	journal.Journal `json:"journal" bson:"journal"`
}

func (ts *testStream) ID() string {
	return ts.StreamID.Hex()
}

func TestIterator(t *testing.T) {

	var result testStream

	server, err := New("mongodb://127.0.0.1:27017/", "ghost")
	require.Nil(t, err)

	session, err := server.Session(context.TODO())
	require.Nil(t, err)

	collection := session.Collection("Stream")
	require.Nil(t, err)

	it, err := collection.List(exp.Equal("templateId", "comment"))
	require.Nil(t, err)

	for it.Next(&result) {
		t.Log(result)
	}
}
