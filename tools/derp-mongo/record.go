package derpmongo

import (
	"time"

	"github.com/benpate/derp"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Status describes where an error record sits in the triage queue
type Status string

// StatusNew marks an error that has not been triaged yet
const StatusNew = Status("new")

// StatusFixed marks an error whose defect has been repaired
const StatusFixed = Status("fixed")

// StatusIgnored marks an error that is understood, and deliberately not being worked on
const StatusIgnored = Status("ignored")

// Record is a single error report, as it is stored in MongoDB
type Record struct {
	RecordID   primitive.ObjectID `bson:"_id"`
	StatusCode int                `bson:"statusCode"` // HTTP status code of the error itself, NOT the triage Status below
	Location   string             `bson:"location"`
	Message    string             `bson:"message"`
	Error      error              `bson:"error"`
	Signature  string             `bson:"signature"`            // Stable identity of this defect, shared by every occurrence of it
	Status     Status             `bson:"status"`               // Where this record sits in the triage queue
	StatusNote string             `bson:"statusNote,omitempty"` // Why the triage decision was made
	StatusDate primitive.DateTime `bson:"statusDate,omitempty"` // When the triage decision was made
	CreateDate primitive.DateTime `bson:"createDate"`
}

// newRecord builds the database record for an error, stamped with the signature and status
// that make it one item in the triage queue.
func newRecord(err error, statusCode int) Record {

	rootLocation := derp.RootLocation(err)
	rootMessage := derp.RootMessage(err)

	// The signature is built from the values stored beside it, so the record can never
	// describe one error while its signature describes another.
	return Record{
		RecordID:   primitive.NewObjectID(),
		StatusCode: statusCode,
		Location:   rootLocation,
		Message:    rootMessage,
		Error:      err,
		Signature:  Signature(statusCode, derp.Location(err), rootLocation, rootMessage),
		Status:     StatusNew,
		CreateDate: primitive.NewDateTimeFromTime(time.Now()),
	}
}
