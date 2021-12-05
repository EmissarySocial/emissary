package journal

import (
	"strconv"
	"time"
)

// Journal tracks a summary of changes to an object over time.  Journal implements
// *most* of the data.Object interface (except for the ID function) right out of
// the box, so it's a useful example for implementing the data.Object interface,
// or even to embed directly into an existing model object.
type Journal struct {
	CreateDate int64  `json:"createDate" bson:"createDate"`
	UpdateDate int64  `json:"updateDate" bson:"updateDate"`
	DeleteDate int64  `json:"deleteDate" bson:"deleteDate"`
	Note       string `json:"note"       bson:"note"`
	Revision   int64  `json:"signature"  bson:"signature"`
}

// IsNew returns TRUE if the object has not yet been saved to the database
func (journal *Journal) IsNew() bool {
	return (journal.CreateDate == 0)
}

// IsDeleted returns TRUE if the object has been "virtually deleted" from the database
func (journal *Journal) IsDeleted() bool {
	return (journal.DeleteDate > 0)
}

// SetCreated must be called whenever a new object is added to the database
func (journal *Journal) SetCreated(note string) {

	timestamp := time.Now().UnixNano()
	journal.CreateDate = timestamp
	journal.UpdateDate = timestamp

	if note != "" {
		journal.Note = note
	}
}

// SetUpdated must be called whenever an existing object is updated in the database
func (journal *Journal) SetUpdated(note string) {

	journal.UpdateDate = time.Now().UnixNano()
	journal.Revision = journal.Revision + 1

	if note != "" {
		journal.Note = note
	}
}

// SetDeleted must be called to "virtual-delete" an object in the database
func (journal *Journal) SetDeleted(note string) {

	timestamp := time.Now().UnixNano()
	journal.UpdateDate = timestamp
	journal.DeleteDate = timestamp
	journal.Revision = journal.Revision + 1

	if note != "" {
		journal.Note = note
	}
}

// ETag returns the signature for this object.  It currently returns the "revision number"
// which should be fine unless we make out-of-band updates to objects.
func (journal *Journal) ETag() string {
	return strconv.FormatInt(journal.Revision, 10)
}
