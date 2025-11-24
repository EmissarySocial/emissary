package service

import (
	"encoding/json"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/tools/id"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
)

// ImportSave is a part of the "Importable" interface, and
// saves an imported document IF it is a Stream document.
func (service *Stream) Import(session data.Session, user *model.User, document []byte) error {

	const location = "service.Stream.Import"

	stream := model.NewStream()

	// Unmarshal the document into the new Stream
	if err := json.Unmarshal(document, &stream); err != nil {
		return derp.Wrap(err, location, "Unable to parse remote document", document)
	}

	// Map values from the original Stream into the new, local Stream
	stream.ParentID = user.UserID               // Associate the Stream with the LOCAL user
	stream.ParentIDs[0] = user.UserID           // Associate the Stream with the LOCAL user
	stream.AttributedTo = user.PersonLink()     // Associate the Stream with the LOCAL user
	stream.Groups = mapof.NewObject[id.Slice]() // Group permissions cannot be migrated to a new server
	stream.URL = ""                             // This will be recalculated by the StreamService.Save

	// TODO: These values must be rewritten
	stream.IconURL = ""

	// Save the Stream to the database
	if err := service.Save(session, &stream, "Imported"); err != nil {
		return derp.Wrap(err, location, "Unable to save imported Stream")
	}

	// A Man, A Plan, A Canal. Panama.
	return nil
}
