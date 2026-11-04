package build

import (
	"io"
	"mime/multipart"
	"net/http"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/tools/formdata"
	"github.com/benpate/derp"
	"github.com/benpate/form"
	"github.com/benpate/html"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// StepStageAttachment is a Step that holds one file under the signed-in User for an object that
// does not exist yet, and answers with the file's tile for the upload-multiple widget
type StepStageAttachment struct{}

// Get writes nothing, because every create form the User has open shares the staged files, so
// there is no row to draw. Implements the Step interface.
func (step StepStageAttachment) Get(_ Builder, _ io.Writer) PipelineBehavior {
	return Continue()
}

// Post stores the one posted file under the signed-in User and draws its tile, or deletes the
// staged file the query names and draws nothing. Implements the Step interface.
func (step StepStageAttachment) Post(builder Builder, buffer io.Writer) PipelineBehavior {

	const location = "build.StepStageAttachment.Post"

	// RULE: A staged file belongs to the User who uploaded it, so the request must be signed in
	if !builder.IsAuthenticated() {
		return Halt().WithError(derp.Unauthorized(location, "Sign in to upload a file"))
	}

	userID := builder.AuthenticatedID()

	// A request naming an attachmentId deletes that one file and answers with nothing, so the tile removes itself
	if token := builder.QueryParam("attachmentId"); token != "" {

		if err := step.deleteOne(builder, userID, token); err != nil {
			return Halt().WithError(err)
		}

		return Continue()
	}

	// RULE: The file comes from the body alone, under whatever field the widget posts it as
	if _, err := formdata.ParseBody(builder.request()); err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Reading form", derp.WithBadRequest()))
	}

	fieldname, header, err := singlePostedFile(builder.request())

	if err != nil {
		return Halt().WithError(err)
	}

	attachment, err := step.storeFile(builder, userID, header)

	if err != nil {
		return Halt().WithError(err)
	}

	// The tile's order field takes the field the file came under, so the Save posts the staged
	// ID beside the saved ones; its X posts back here, and the empty answer removes the tile
	row := tileRow{
		BasePath:     builder.BasePath(),
		ActionURL:    builder.BasePath() + "/" + builder.actionID(),
		OrderName:    fieldname + form.UploadOrderSuffix,
		DeleteTarget: "closest .upload-multiple-tile",
	}

	b := html.New()
	drawTile(b, row, attachment)
	b.CloseAll()

	if _, err := io.WriteString(buffer, b.String()); err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Writing tile to buffer"))
	}

	return Continue()
}

// deleteOne removes the staged file named by the token, or refuses with a 400: an invalid token,
// or one that is not a file staged under this User
func (step StepStageAttachment) deleteOne(builder Builder, userID primitive.ObjectID, token string) error {

	const location = "build.StepStageAttachment.deleteOne"

	attachmentID, err := primitive.ObjectIDFromHex(token)

	if err != nil {
		return derp.BadRequest(location, "Invalid attachmentId", token)
	}

	// The service refuses an ID that is not one of this User's staged files
	if err := builder.factory().Attachment().DeleteFromCategory(builder.session(), model.AttachmentObjectTypeUser, userID, model.AttachmentCategoryStaging, attachmentID, "Deleted by stage-attachment"); err != nil {
		return derp.Wrap(err, location, "Deleting staged attachment", attachmentID)
	}

	return nil
}

// singlePostedFile returns the one file in the body and the field it was posted under, or refuses: a
// body with no file, or with more than one, is a bad request
func singlePostedFile(request *http.Request) (string, *multipart.FileHeader, error) {

	const location = "build.singlePostedFile"

	if request.MultipartForm == nil {
		return "", nil, derp.BadRequest(location, "No file was uploaded")
	}

	fieldname := ""
	var header *multipart.FileHeader
	count := 0

	for name, headers := range request.MultipartForm.File {
		for _, current := range headers {
			fieldname = name
			header = current
			count++
		}
	}

	switch {

	case count == 0:
		return "", nil, derp.BadRequest(location, "No file was uploaded")

	// RULE: One file per request, because each answer is one tile
	case count > 1:
		return "", nil, derp.BadRequest(location, "One file at a time", count)
	}

	return fieldname, header, nil
}

// storeFile sniffs the posted file, stores it, and files it under the User in the staging category
func (step StepStageAttachment) storeFile(builder Builder, userID primitive.ObjectID, header *multipart.FileHeader) (model.Attachment, error) {

	const location = "build.StepStageAttachment.storeFile"

	// RULE: singlePostedFile returns no header with its error, and this is the one caller, so this
	// guard is what makes the contract visible to nil analysis
	if header == nil {
		return model.Attachment{}, derp.BadRequest(location, "No file was uploaded")
	}

	// The staging category takes any type; the step that adopts the file enforces its own accept-type
	uploads, err := openUploads([]*multipart.FileHeader{header}, "")

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
