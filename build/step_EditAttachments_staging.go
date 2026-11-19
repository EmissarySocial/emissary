package build

import (
	"io"
	"mime/multipart"
	"strconv"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/tools/formdata"
	"github.com/benpate/derp"
	"github.com/benpate/html"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

/******************************************
 * Staging
 *
 * An object that is not saved yet has no route of its own, so its create form's widget reaches
 * this step through the parent's add, with the action named in the query. Each file dropped there
 * is filed under the signed-in User until the form's Save, which reaches the step through include,
 * adopts it. These are the methods that serve that widget before the Save.
 ******************************************/

// stage answers the widget's own request on an object that is not saved yet: it files the one
// posted file under the signed-in User and draws its tile, or deletes the staged file the query
// names and draws nothing, so the tile removes itself
func (step StepEditAttachments) stage(builder Builder, buffer io.Writer) PipelineBehavior {

	const location = "build.StepEditAttachments.stage"

	// RULE: A staged file belongs to the User who uploaded it, so the request must be signed in
	if !builder.IsAuthenticated() {
		return Halt().WithError(derp.Unauthorized(location, "Sign in to upload a file"))
	}

	userID := builder.AuthenticatedID()

	// RULE: The answer halts the action, because the steps after this one belong to the saved
	// object; a save among them would write the empty object once per file (see AGENTS.md)
	if token := builder.QueryParam("attachmentId"); token != "" {

		if err := deleteStaged(builder, userID, token); err != nil {
			return Halt().WithError(err)
		}

		return Halt()
	}

	// RULE: The file comes from the body alone, under the step's own field, and one per request,
	// because each answer is one tile
	if _, err := formdata.ParseBody(builder.request()); err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Reading form", derp.WithBadRequest()))
	}

	files := postedFiles(builder.request(), step.Fieldname)

	switch {

	case len(files) == 0:
		return Halt().WithError(derp.BadRequest(location, "No file was uploaded", step.Fieldname))

	case len(files) > 1:
		return Halt().WithError(derp.BadRequest(location, "One file at a time", len(files)))
	}

	attachment, err := stageFile(builder, userID, files[0], step.AcceptType)

	if err != nil {
		return Halt().WithError(err)
	}

	// The tile's X posts back here, and the empty answer removes the tile alone;
	// its preview hangs from the User, who holds the file until the Save
	row := tileRow{
		BasePath:     "/@" + userID.Hex(),
		ActionURL:    stagingURL(builder),
		OrderName:    step.Fieldname + uploadOrderSuffix,
		DeleteTarget: "closest .upload-multiple-tile",
	}

	b := html.New()
	drawTile(b, row, attachment)
	b.CloseAll()

	if _, err := io.WriteString(buffer, b.String()); err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Writing tile to buffer"))
	}

	// The tile is the whole answer
	return Halt()
}

// drawStagingZone writes everything inside the widget's zone for an object that is not saved
// yet: the empty row wired to stage each file here, then the plus, the file input, and the status line
func (step StepEditAttachments) drawStagingZone(builder Builder, buffer io.Writer) error {

	const location = "build.StepEditAttachments.drawStagingZone"

	b := html.New()
	writeZone(b)
	step.writeStagingRow(b, builder)
	writeControls(b, step.Fieldname, step.AcceptType)
	b.CloseAll()

	if _, err := io.WriteString(buffer, b.String()); err != nil {
		return derp.Wrap(err, location, "Writing staging zone to buffer")
	}

	return nil
}

// writeStagingRow writes a new object's row: empty, carrying this request's own route for the behavior to
// stage each file through, with the step's field name, limits, and accepted types, and the multipart encoding
func (step StepEditAttachments) writeStagingRow(b *html.Builder, builder Builder) {

	b.Div().
		Class("upload-multiple-tiles").
		Data("staging-url", stagingURL(builder)).
		Data("name", step.Fieldname).
		Data("accept", step.AcceptType).
		Data("minimum", strconv.Itoa(step.Minimum)).
		Data("maximum", strconv.Itoa(step.Maximum)).
		Attr("hx-params", step.Fieldname).
		Attr("hx-push-url", "false").
		Attr("hx-sync", "closest .upload-multiple:queue all").
		Attr("hx-encoding", "multipart/form-data").
		Close()
}

// stagingURL returns the route a new object's widget came through, the parent's add with the
// action named in its query, without the attachmentId a delete carried, so the row and every
// tile post back the same way
func stagingURL(builder Builder) string {

	query := builder.request().URL.Query()
	query.Del("attachmentId")

	return builder.request().URL.Path + "?" + query.Encode()
}

// reachedThroughInclude reports whether another action reached the step through include, as a
// form's Save does, rather than the request naming the step's own action, as the widget's requests do
func reachedThroughInclude(builder Builder, fieldname string) bool {
	return !Pipeline(builder.action().Steps).EditsAttachments(fieldname)
}

// deleteStaged removes the staged file named by the token, or refuses with a 400: an invalid
// token, or one that is not a file staged under this User
func deleteStaged(builder Builder, userID primitive.ObjectID, token string) error {

	const location = "build.deleteStaged"

	attachmentID, err := primitive.ObjectIDFromHex(token)

	if err != nil {
		return derp.BadRequest(location, "Invalid attachmentId", token)
	}

	// The service refuses an ID that is not one of this User's staged files
	if err := builder.factory().Attachment().DeleteFromCategory(builder.session(), model.AttachmentObjectTypeUser, userID, model.AttachmentCategoryStaging, attachmentID, "Deleted by edit-attachments"); err != nil {
		return derp.Wrap(err, location, "Deleting staged attachment", attachmentID)
	}

	return nil
}

// stageFile sniffs the posted file against the step's accepted types, stores it, and files it
// under the User in the staging category, where it waits for the form's Save to adopt it
func stageFile(builder Builder, userID primitive.ObjectID, header *multipart.FileHeader, acceptType string) (model.Attachment, error) {

	const location = "build.stageFile"

	// RULE: The multipart parser never lists a nil header, and this guard is what makes that
	// visible to nil analysis
	if header == nil {
		return model.Attachment{}, derp.BadRequest(location, "No file was uploaded")
	}

	// The Save checks the type again when it adopts, against the same step
	uploads, err := openUploads([]*multipart.FileHeader{header}, acceptType)

	if err != nil {
		return model.Attachment{}, derp.Wrap(err, location, "Reading uploaded file", header.Filename)
	}

	defer closeUploads(uploads)

	factory := builder.factory()
	upload := uploads[0]

	attachment := model.NewAttachment(model.AttachmentObjectTypeUser, userID)
	attachment.Original = upload.header.Filename
	attachment.ContentType = upload.contentType
	attachment.Category = model.AttachmentCategoryStaging

	if err := factory.MediaServer().Put(attachment.AttachmentID.Hex(), upload.reader); err != nil {
		return model.Attachment{}, derp.Wrap(err, location, "Saving attachment to mediaserver", attachment)
	}

	if err := factory.Attachment().Save(builder.session(), &attachment, "Staged file: "+upload.header.Filename); err != nil {
		return model.Attachment{}, derp.Wrap(err, location, "Saving attachment", attachment)
	}

	return attachment, nil
}
