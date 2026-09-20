package model

import (
	"github.com/benpate/data/journal"
	"github.com/benpate/rosetta/mapof"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// StreamSource is a subscription to content hosted outside of Emissary, which is copied into
// the Content of the Stream that it is attached to.
type StreamSource struct {
	StreamSourceID primitive.ObjectID `json:"streamSourceId" bson:"_id"`           // Unique identifier of this record
	StreamID       primitive.ObjectID `json:"streamId"       bson:"streamId"`      // Stream whose Content this record populates
	Method         string             `json:"method"         bson:"method"`        // Adapter that reads the remote source (GIT)
	URL            string             `json:"url"            bson:"url"`           // Address of the remote source
	Config         mapof.String       `json:"config"         bson:"config"`        // Adapter-specific settings, such as a branch and a file path
	Version        string             `json:"version"        bson:"version"`       // Opaque token for the last remote state applied to the Stream
	ContentHash    string             `json:"contentHash"    bson:"contentHash"`   // Hexadecimal hash of the last remote content applied to the Stream
	Status         string             `json:"status"         bson:"status"`        // Outcome of the last synchronization (NEW, LOADING, SUCCESS, FAILURE)
	StatusMessage  string             `json:"statusMessage"  bson:"statusMessage"` // Operator-readable reason for the current Status
	LastSynced     int64              `json:"lastSynced"     bson:"lastSynced"`    // Unix epoch SECONDS when the last synchronization began

	journal.Journal `json:"-" bson:",inline"`
}

// NewStreamSource returns a fully initialized StreamSource object
func NewStreamSource() StreamSource {
	return StreamSource{
		StreamSourceID: primitive.NewObjectID(),
		Config:         mapof.NewString(),
		Status:         StreamSourceStatusNew,
	}
}

/******************************************
 * data.Object Interface
 ******************************************/

// ID returns the primary key of this object
func (streamSource StreamSource) ID() string {
	return streamSource.StreamSourceID.Hex()
}
