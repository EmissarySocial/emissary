package service

import (
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Adopt moves a file staged under a User onto the object that now owns it. The caller sets
// the category, rank, and rules it wants first; the owner and the URL change here.
func (service *Attachment) Adopt(session data.Session, attachment *model.Attachment, objectType string, objectID primitive.ObjectID, note string) error {

	const location = "service.Attachment.Adopt"

	// RULE: Only a file staged under a User is adopted, so a file already on an object is never moved
	if attachment.ObjectType != model.AttachmentObjectTypeUser {
		return derp.BadRequest(location, "Attachment is not staged under a User", attachment.AttachmentID, attachment.ObjectType)
	}

	attachment.ObjectType = objectType
	attachment.ObjectID = objectID

	if err := service.Save(session, attachment, note); err != nil {
		return derp.Wrap(err, location, "Saving adopted attachment", attachment.AttachmentID, objectType, objectID)
	}

	// Welcome to the family.
	return nil
}

// PurgeStaged deletes every file staged under any User in the category that was uploaded before
// the cutoff, with its stored file, and returns how many went
func (service *Attachment) PurgeStaged(session data.Session, category string, before time.Time) (int, error) {

	const location = "service.Attachment.PurgeStaged"

	// The journal's createDate is Unix milliseconds
	criteria := exp.And(
		exp.Equal("objectType", model.AttachmentObjectTypeUser),
		exp.Equal("category", category),
		exp.LessThan("createDate", before.UnixMilli()),
	)

	staged, err := service.Range(session, criteria)

	if err != nil {
		return 0, derp.Wrap(err, location, "Listing staged attachments", category)
	}

	count := 0

	for attachment := range staged {

		if err := service.Delete(session, &attachment, "Purged: never adopted"); err != nil {
			return count, derp.Wrap(err, location, "Deleting staged attachment", attachment.AttachmentID)
		}

		count++
	}

	return count, nil
}
