package build

import (
	"io"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/tools/formdata"
	"github.com/benpate/derp"
	"github.com/benpate/html"
	"github.com/benpate/rosetta/first"
	"github.com/benpate/rosetta/list"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// StepEditAttachments is a Step that draws the files in one category as tiles, and uploads,
// deletes, and sorts them in place, behind an html-remote form element
type StepEditAttachments struct {
	Category   string   // Category of the files this step manages
	Fieldname  string   // Form field that posts the files, and whose ".order" field posts their order (Default: the category)
	AcceptType string   // Mime Type(s) to accept (e.g. "image/*")
	Minimum    int      // Fewest files the category may hold (Default: 0)
	Maximum    int      // Most files the category may hold (Default: 1)
	RuleHeight int      // Fixed height for all downloads
	RuleWidth  int      // Fixed width for all downloads
	RuleTypes  []string // Allowed extensions.  The first value is used as the default.
}

// Get draws the zone an html-remote element loads: the row of files, the plus, the file input, and
// the status line, or the files alone with ?readonly=true. Implements the Step interface.
func (step StepEditAttachments) Get(builder Builder, buffer io.Writer) PipelineBehavior {

	const location = "build.StepEditAttachments.Get"

	// A new object's form reaches this step through include, and there is nothing to draw in it;
	// the widget's own request on a new object draws the zone that stages files until the Save
	if builder.object().IsNew() {

		if reachedThroughInclude(builder, step.Fieldname) {
			return Continue()
		}

		if err := step.drawStagingZone(builder, buffer); err != nil {
			return Halt().WithError(derp.Wrap(err, location, "Drawing staging zone"))
		}

		// The zone is the whole answer, and the steps after this one belong to the saved object
		return Halt()
	}

	readOnly := builder.QueryParam("readonly") == "true"

	if err := step.drawZone(builder, buffer, readOnly); err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Drawing attachments"))
	}

	return Continue()
}

// Post applies one change to the category, a delete, an upload, or a sort, then draws the tile row again. Implements the Step interface.
func (step StepEditAttachments) Post(builder Builder, buffer io.Writer) PipelineBehavior {

	const location = "build.StepEditAttachments.Post"

	// RULE: On a new object the widget's own request stages one file under the User, or deletes
	// one; the form's Save, which reaches the step through include, adopts them below
	if builder.object().IsNew() {
		if !reachedThroughInclude(builder, step.Fieldname) {
			return step.stage(builder, buffer)
		}
	}

	objectType := attachmentObjectType(builder)
	objectID := builder.objectID()

	// A request naming an attachmentId deletes that one file, and reads nothing else
	if token := builder.QueryParam("attachmentId"); token != "" {

		if err := step.deleteOne(builder, objectType, objectID, token); err != nil {
			return Halt().WithError(err)
		}

		return step.respond(builder, buffer)
	}

	// RULE: The files and the order come from the body alone, never the query string
	values, err := formdata.ParseBody(builder.request())

	if err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Reading form", derp.WithBadRequest()))
	}

	order := values[step.Fieldname+uploadOrderSuffix]

	// Store the posted files and adopt the staged ones the order names, then apply the order
	if err := step.addFiles(builder, objectType, objectID, postedFiles(builder.request(), step.Fieldname), order); err != nil {
		return Halt().WithError(err)
	}

	if err := step.applyOrder(builder, objectType, objectID, order); err != nil {
		return Halt().WithError(err)
	}

	return step.respond(builder, buffer)
}

// respond draws the tile row for a saved object after a change, and nothing for a new one, which
// has no files to list and no route of its own yet. Either way the pipeline continues.
func (step StepEditAttachments) respond(builder Builder, buffer io.Writer) PipelineBehavior {

	const location = "build.StepEditAttachments.respond"

	if builder.object().IsNew() {
		return Continue()
	}

	if err := step.drawRow(builder, buffer); err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Drawing attachments"))
	}

	return Continue()
}

// attachmentObjectType names the object the files belong to: a draft's files belong to its Stream
func attachmentObjectType(builder Builder) string {

	if objectType := builder.service().ObjectType(); objectType != "StreamDraft" {
		return objectType
	}

	return model.AttachmentObjectTypeStream
}

// deleteOne removes the file named by the token from the category, or refuses: an invalid token
// with a 400, and a delete that would leave fewer files than the minimum with a validation error
func (step StepEditAttachments) deleteOne(builder Builder, objectType string, objectID primitive.ObjectID, token string) error {

	const location = "build.StepEditAttachments.deleteOne"

	// RULE: The token must be an ID. This step never widens a delete beyond one file.
	attachmentID, err := primitive.ObjectIDFromHex(token)

	if err != nil {
		return derp.BadRequest(location, "Invalid attachmentId", token)
	}

	attachmentService := builder.factory().Attachment()
	session := builder.session()

	// RULE: The category must keep at least the minimum, so a delete that would leave fewer is refused
	count, err := attachmentService.CountByCategory(session, objectType, objectID, step.Category)

	if err != nil {
		return derp.Wrap(err, location, "Counting attachments", objectType, objectID, step.Category)
	}

	if count-1 < step.Minimum {
		return derp.Validation(minimumMessage(step.Minimum), "minimum", step.Minimum, "remaining", count-1)
	}

	// The service refuses an ID that is not one of this category's files on this object
	if err := attachmentService.DeleteFromCategory(session, objectType, objectID, step.Category, attachmentID, "Deleted by edit-attachments"); err != nil {
		return derp.Wrap(err, location, "Deleting attachment", attachmentID)
	}

	return nil
}

// postedFiles returns the files posted under the field, or nothing when the body is not multipart
func postedFiles(request *http.Request, fieldname string) []*multipart.FileHeader {

	if request.MultipartForm == nil {
		return nil
	}

	return request.MultipartForm.File[fieldname]
}

// addFiles fits the posted files and the staged files the order names to the category's limits and
// accepted types, and only then stores and adopts them, ranked after the files the category holds
func (step StepEditAttachments) addFiles(builder Builder, objectType string, objectID primitive.ObjectID, files []*multipart.FileHeader, order []string) error {

	const location = "build.StepEditAttachments.addFiles"

	// Load what the category already holds: the count decides what fits, and the ranks decide where new files go
	existing, err := builder.factory().Attachment().QueryByCategory(builder.session(), objectType, objectID, step.Category)

	if err != nil {
		return derp.Wrap(err, location, "Loading existing attachments", objectType, objectID, step.Category)
	}

	staged, err := step.stagedFiles(builder, order)

	if err != nil {
		return derp.Wrap(err, location, "Loading staged attachments")
	}

	// RULE: A new object's files arrive with the form's Save, and a saved object's one change at a time,
	// so the two are held to their limits differently
	if builder.object().IsNew() {
		files, staged, err = step.limitSave(len(existing), files, staged)
	} else {
		err = step.limitChange(len(existing), len(files)+len(staged))
	}

	if err != nil {
		return err
	}

	// RULE: A staged file joins the category only as an accepted type, judged by the type sniffed when it was staged
	for _, attachment := range staged {
		if !contentTypeMatches(attachment.ContentType, step.AcceptType) {
			return derp.BadRequest(location, "Staged file is not an allowed type", attachment.Original, attachment.ContentType, step.AcceptType)
		}
	}

	// New files rank after everything the category holds: the posted ones first, then the staged ones
	nextRank := nextRankAfter("append", existing)

	if err := step.storeFiles(builder, objectType, objectID, files, nextRank); err != nil {
		return derp.Wrap(err, location, "Storing posted files")
	}

	if err := step.adoptFiles(builder, objectType, objectID, staged, nextRank+len(files)); err != nil {
		return derp.Wrap(err, location, "Adopting staged files")
	}

	return nil
}

// limitSave keeps the files of a create form's Save that fit under the maximum, dropping the last
// in rank order, and refuses a Save that leaves the category short of the minimum
func (step StepEditAttachments) limitSave(existing int, files []*multipart.FileHeader, staged []model.Attachment) ([]*multipart.FileHeader, []model.Attachment, error) {

	// Keep the posted files first, then the staged ones, because that is how they rank. A dropped
	// staged file stays staged until the purge removes it.
	room := max(step.Maximum-existing, 0)
	files = files[:min(len(files), room)]
	staged = staged[:min(len(staged), room-len(files))]

	// RULE: The Save must bring the category up to its minimum
	if count := existing + len(files) + len(staged); count < step.Minimum {
		return nil, nil, derp.Validation(minimumMessage(step.Minimum), "minimum", step.Minimum, "count", count)
	}

	return files, staged, nil
}

// limitChange refuses an add to a saved object that would pass the maximum. A sort, or an add that
// leaves the category short of its minimum, passes, because it moves the category toward its limits.
func (step StepEditAttachments) limitChange(existing int, adding int) error {

	// A sort adds nothing, so it passes even when the category is already over the maximum
	if adding == 0 {
		return nil
	}

	// RULE: An add past the maximum is refused whole, never truncated, because the visitor just chose this file
	if count := existing + adding; count > step.Maximum {
		return derp.Validation(maximumMessage(step.Maximum), "maximum", step.Maximum, "count", count)
	}

	// Room at the inn
	return nil
}

// stagedFiles returns the signed-in User's staged files that the order names, in the order named.
// A value that is not one of them, such as a saved file's ID in a sort, is left for applyOrder.
func (step StepEditAttachments) stagedFiles(builder Builder, order []string) ([]model.Attachment, error) {

	const location = "build.StepEditAttachments.stagedFiles"

	attachmentIDs := attachmentIDs(order)

	if len(attachmentIDs) == 0 || !builder.IsAuthenticated() {
		return nil, nil
	}

	// RULE: Only the caller's own staged files are adopted, so an ID is never loaded on its own
	staged, err := builder.factory().Attachment().QueryByCategory(builder.session(), model.AttachmentObjectTypeUser, builder.AuthenticatedID(), model.AttachmentCategoryStaging)

	if err != nil {
		return nil, derp.Wrap(err, location, "Loading the user's staged attachments")
	}

	byID := make(map[primitive.ObjectID]model.Attachment, len(staged))

	for _, attachment := range staged {
		byID[attachment.AttachmentID] = attachment
	}

	result := make([]model.Attachment, 0, len(attachmentIDs))

	for _, attachmentID := range attachmentIDs {

		if attachment, found := byID[attachmentID]; found {
			result = append(result, attachment)
			delete(byID, attachmentID)
		}
	}

	return result, nil
}

// storeFiles sniffs every posted file against the accepted types, and only then stores them,
// ranked from nextRank in the order posted
func (step StepEditAttachments) storeFiles(builder Builder, objectType string, objectID primitive.ObjectID, files []*multipart.FileHeader, nextRank int) error {

	const location = "build.StepEditAttachments.storeFiles"

	if len(files) == 0 {
		return nil
	}

	// Sniff every file before storing any, so a file the step refuses halts with nothing written
	uploads, err := openUploads(files, step.AcceptType)

	if err != nil {
		return derp.Wrap(err, location, "Checking uploaded files")
	}

	defer closeUploads(uploads)

	factory := builder.factory()
	attachmentService := factory.Attachment()
	session := builder.session()

	for index, upload := range uploads {

		attachment := model.NewAttachment(objectType, objectID)
		attachment.Original = upload.header.Filename
		attachment.ContentType = upload.contentType
		attachment.Category = step.Category
		attachment.Rank = nextRank + index
		attachment.SetRules(step.RuleWidth, step.RuleHeight, step.RuleTypes)

		if err := factory.MediaServer().Put(attachment.AttachmentID.Hex(), upload.reader); err != nil {
			return derp.Wrap(err, location, "Saving attachment to mediaserver", attachment)
		}

		if err := attachmentService.Save(session, &attachment, "Uploaded file: "+upload.header.Filename); err != nil {
			return derp.Wrap(err, location, "Saving attachment", attachment)
		}
	}

	return nil
}

// adoptFiles moves the staged files onto the object, into this category with its rules, ranked
// from nextRank in the order given
func (step StepEditAttachments) adoptFiles(builder Builder, objectType string, objectID primitive.ObjectID, staged []model.Attachment, nextRank int) error {

	const location = "build.StepEditAttachments.adoptFiles"

	attachmentService := builder.factory().Attachment()
	session := builder.session()

	for index, attachment := range staged {

		attachment.Category = step.Category
		attachment.Rank = nextRank + index
		attachment.SetRules(step.RuleWidth, step.RuleHeight, step.RuleTypes)

		if err := attachmentService.Adopt(session, &attachment, objectType, objectID, "Adopted file: "+attachment.Original); err != nil {
			return derp.Wrap(err, location, "Adopting attachment", attachment.AttachmentID)
		}
	}

	return nil
}

// applyOrder re-ranks the files in the order posted, skipping any value that is not a saved file's ID
func (step StepEditAttachments) applyOrder(builder Builder, objectType string, objectID primitive.ObjectID, values []string) error {

	const location = "build.StepEditAttachments.applyOrder"

	order := attachmentIDs(values)

	if len(order) == 0 {
		return nil
	}

	if err := builder.factory().Attachment().Reorder(builder.session(), objectType, objectID, step.Category, order, "Sorted by edit-attachments"); err != nil {
		return derp.Wrap(err, location, "Reordering attachments", objectType, objectID, step.Category)
	}

	return nil
}

// attachmentIDs returns the values that parse as ObjectIDs, in the order given, and skips the rest
func attachmentIDs(values []string) []primitive.ObjectID {

	result := make([]primitive.ObjectID, 0, len(values))

	for _, value := range values {
		if attachmentID, err := primitive.ObjectIDFromHex(value); err == nil {
			result = append(result, attachmentID)
		}
	}

	return result
}

// drawZone writes everything inside the widget's zone: the row of files, then the plus, the file
// input, and the status line, or the files alone when the view is read-only
func (step StepEditAttachments) drawZone(builder Builder, buffer io.Writer, readOnly bool) error {

	const location = "build.StepEditAttachments.drawZone"

	attachments, err := step.loadAttachments(builder)

	if err != nil {
		return derp.Wrap(err, location, "Loading attachments")
	}

	b := html.New()
	writeZone(b)
	step.writeRow(b, builder, attachments, readOnly)

	if !readOnly {
		writeControls(b, step.Fieldname, step.AcceptType)
	}

	b.CloseAll()

	if _, err := io.WriteString(buffer, b.String()); err != nil {
		return derp.Wrap(err, location, "Writing attachments to buffer")
	}

	return nil
}

// drawRow writes the category's files as the row alone, which answers every change the widget posts
func (step StepEditAttachments) drawRow(builder Builder, buffer io.Writer) error {

	const location = "build.StepEditAttachments.drawRow"

	attachments, err := step.loadAttachments(builder)

	if err != nil {
		return derp.Wrap(err, location, "Loading attachments")
	}

	b := html.New()
	step.writeRow(b, builder, attachments, false)
	b.CloseAll()

	if _, err := io.WriteString(buffer, b.String()); err != nil {
		return derp.Wrap(err, location, "Writing attachments to buffer")
	}

	return nil
}

// loadAttachments returns the category's files on the object, in rank order
func (step StepEditAttachments) loadAttachments(builder Builder) ([]model.Attachment, error) {

	const location = "build.StepEditAttachments.loadAttachments"

	objectType := attachmentObjectType(builder)
	attachments, err := builder.factory().Attachment().QueryByCategory(builder.session(), objectType, builder.objectID(), step.Category)

	if err != nil {
		return nil, derp.Wrap(err, location, "Loading attachments", objectType, builder.objectID(), step.Category)
	}

	return attachments, nil
}

// writeRow writes the tile row the theme's behavior expects, with the limits as attributes,
// the sort wired to this action, and the multipart encoding its uploads need, which the form
// around it no longer carries; or, read-only, the tiles alone with nothing wired
func (step StepEditAttachments) writeRow(b *html.Builder, builder Builder, attachments []model.Attachment, readOnly bool) {

	// The action's own path, without the request's query string, which could carry a stale attachmentId
	row := tileRow{
		BasePath:     builder.BasePath(),
		ActionURL:    builder.BasePath() + "/" + builder.actionID(),
		OrderName:    step.Fieldname + uploadOrderSuffix,
		DeleteTarget: "closest .upload-multiple-tiles",
		Confirm:      "Delete this file?",
		ReadOnly:     readOnly,
	}

	// RULE: A read-only row is marked so that the behavior never makes it sortable
	if readOnly {
		b.Div().Class("upload-multiple-tiles").Data("readonly", "true")
	} else {
		b.Div().
			Class("upload-multiple-tiles").
			Data("url", row.ActionURL).
			Data("name", step.Fieldname).
			Data("accept", step.AcceptType).
			Data("minimum", strconv.Itoa(step.Minimum)).
			Data("maximum", strconv.Itoa(step.Maximum)).
			Data("count", strconv.Itoa(len(attachments))).
			Attr("hx-post", row.ActionURL).
			Attr("hx-trigger", "upload-multiple-added, upload-multiple-sorted").
			Attr("hx-params", step.Fieldname+","+row.OrderName).
			Attr("hx-target", "this").
			Attr("hx-swap", "outerHTML").
			Attr("hx-push-url", "false").
			Attr("hx-sync", "closest .upload-multiple:queue all").
			Attr("hx-encoding", "multipart/form-data")
	}

	for _, attachment := range attachments {
		drawTile(b, row, attachment)
	}

	b.Close()
}

// maximumMessage returns the message shown inline when an upload would leave the category
// holding more files than the step allows
func maximumMessage(maximum int) string {

	if maximum == 1 {
		return "At most 1 file is allowed"
	}

	return "At most " + strconv.Itoa(maximum) + " files are allowed"
}

// uploadOrderSuffix is appended to a widget's field name to name the hidden input each tile carries,
// which posts the tile's place in the order: one file ID per tile. The behavior and the
// qwertylicious test server copy the literal.
const uploadOrderSuffix = ".order"

// tileRow is what every tile in a row shares: where its preview and its delete are served from,
// the name of its order field, what its X does, and whether it has any controls at all
type tileRow struct {
	BasePath     string // The owner's own path, which the previews hang from
	ActionURL    string // The action the X posts its delete to
	OrderName    string // The hidden input that posts the tile's place in the order
	DeleteTarget string // What the X's response replaces: the whole row for a saved file, the tile alone for a staged one
	Confirm      string // What the X asks before deleting, or nothing for a file that is not saved yet
	ReadOnly     bool   // Whether the tile is the preview alone, with no handle, no X, and no order input
}

// drawTile writes one file as a tile: its preview, which is also its handle, its X, and its place in the order
func drawTile(b *html.Builder, row tileRow, attachment model.Attachment) {

	attachmentID := attachment.AttachmentID.Hex()
	filename := first.String(attachment.Original, attachment.Label, attachmentID)
	group := list.Slash(attachment.ContentType).First()

	// The tile is its own handle: it takes focus for the arrow keys, and its X comes with the mouse
	// (.hover-trigger). A read-only tile is the preview alone.
	if row.ReadOnly {
		b.Div().Class("upload-multiple-tile").Data("value", attachmentID).Data("group", group)
	} else {
		b.Div().Class("upload-multiple-tile", "hover-trigger").Data("value", attachmentID).Data("group", group).Attr("tabindex", "0").Aria("label", "Move "+filename)
	}

	// InnerText and Attr escape: a filename is whatever the visitor named the file.
	// The preview is not draggable on its own, so a press on it drags the tile.
	if group == "image" {
		b.Img(row.BasePath+"/attachments/"+attachmentID+"?width=300").Class("upload-multiple-preview").Attr("alt", filename).Attr("draggable", "false").Close()
	} else {
		b.I("bi", "bi-file-earmark", "upload-multiple-icon").Close()
		b.Span().Class("upload-multiple-filename").InnerText(filename).Close()
	}

	if row.ReadOnly {
		b.Close()
		return
	}

	// RULE: The X is type="button" because a bare button inside a form submits it.
	// It deletes at once through htmx, and the response replaces the row's target.
	b.Button().
		Type("button").
		Class("upload-multiple-remove", "hover-show").
		Attr("hx-post", withQueryParam(row.ActionURL, "attachmentId="+attachmentID)).
		Attr("hx-params", "none").
		Attr("hx-confirm", row.Confirm).
		Attr("hx-target", row.DeleteTarget).
		Attr("hx-swap", "outerHTML").
		Attr("hx-push-url", "false").
		Attr("hx-sync", "closest .upload-multiple:queue all").
		Aria("label", "Delete "+filename)
	b.I("bi", "bi-x-lg").Close()
	b.Close()

	b.Input("hidden", row.OrderName).Value(attachmentID).Close()
	b.Close()
}

// withQueryParam appends a parameter to a URL, after the query it already has when it has one
func withQueryParam(url string, param string) string {

	if strings.Contains(url, "?") {
		return url + "&" + param
	}

	return url + "?" + param
}

// writeZone opens the zone that an html-remote element loads: the grid the row and the controls
// sit in, marked for the behavior and the styles, which find it by this attribute and this class
func writeZone(b *html.Builder) {
	b.Div().Class("upload-multiple").Data("upload-multiple", "true")
}

// writeControls writes what follows the row inside the widget's zone: the plus that adds one file,
// the hidden single-file input the files post under, and the status line
func writeControls(b *html.Builder, fieldname string, acceptType string) {

	elementID := zoneID(fieldname)

	// RULE: Only the plus opens the picker, and it is a button because a button opens the picker
	// from the keyboard with no script. It follows the row so that it sits after the last tile,
	// and the behavior hides it once the field is full.
	b.Button().ID(elementID+"-add").Type("button").Class("upload-multiple-add").Attr("title", "Add a file").Aria("label", "Add a file")
	b.I("bi", "bi-plus-lg").Close()
	b.Close()

	// RULE: Never `required`, and never `multiple`. The input is hidden, and a hidden invalid
	// control blocks a native submit with no visible cause; the step enforces the minimum.
	// The picker takes one file at a time, which is how the files are added.
	b.Input("file", fieldname).
		ID(elementID+"-input").
		Attr("accept", acceptType).
		Attr("hidden", "hidden").
		Aria("label", "Add a file").
		Close()

	b.Div().ID(elementID + "-status").Class("upload-multiple-status").Role("status").Close()
}

// zoneID returns the prefix of the IDs inside a field's zone, with the field's dots turned to dashes
func zoneID(fieldname string) string {
	return "upload-multiple-" + strings.ReplaceAll(fieldname, ".", "-")
}
