package render

import (
	"bytes"
	"html/template"
	"io"

	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/exp/builder"
	"github.com/benpate/list"
	"github.com/benpate/nebula"
	"github.com/benpate/path"
	"github.com/benpate/schema"
	"github.com/benpate/steranko"
	"github.com/whisperverse/whisperverse/model"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Stream wraps a model.Stream object and provides functions that make it easy to render an HTML template with it.
type Stream struct {
	modelService ModelService    // Service to use to access streams (could be Stream or StreamDraft)
	template     *model.Template // Template that the Stream uses
	action       *model.Action   // The action to be used to render this Stream
	stream       *model.Stream   // The Stream to be displayed

	Common
}

/*******************************************
 * CONSTRUCTORS
 *******************************************/

// NewStream creates a new object that can generate HTML for a specific stream/view
func NewStream(factory Factory, ctx *steranko.Context, template *model.Template, action *model.Action, stream *model.Stream) (Stream, error) {

	// Verify user's authorization to perform this Action on this Stream
	authorization := getAuthorization(ctx)

	if !action.UserCan(stream, authorization) {
		return Stream{}, derp.NewForbiddenError("whisper.render.NewStream", "Forbidden")
	}

	// Success.  Populate Stream
	return Stream{
		modelService: factory.Stream(),
		stream:       stream,
		template:     template,
		action:       action,
		Common:       NewCommon(factory, ctx),
	}, nil
}

// NewStreamWithoutTemplate creates a new object that can generate HTML for a specific stream/view
func NewStreamWithoutTemplate(factory Factory, ctx *steranko.Context, stream *model.Stream, actionID string) (Stream, error) {

	// Use the template service to look up the correct template
	templateService := factory.Template()

	template, err := templateService.Load(stream.TemplateID)

	if err != nil {
		return Stream{}, derp.Wrap(err, "whisper.render.NewStreamWithoutTemplate", "Error loading Template", stream)
	}

	// And look up a valid action (cannot be empty)
	action := template.Action(actionID)

	if action == nil {
		return Stream{}, derp.NewNotFoundError("whisper.render.NewStreamWithoutTemplate", "Unrecognized Action", actionID)
	}

	// Return a fully populated service
	return NewStream(factory, ctx, template, action, stream)
}

/*******************************************
 * PATH INTERFACE
 * (not available via templates)
 *******************************************/

func (w Stream) GetPath(p path.Path) (interface{}, error) {
	return w.stream.GetPath(p)
}

func (w Stream) SetPath(p path.Path, value interface{}) error {
	return w.stream.SetPath(p, value)
}

/*******************************************
 * RENDERER INTERFACE
 *******************************************/

// ActionID returns the name of the action being performed
func (w Stream) ActionID() string {
	return w.action.ActionID
}

// Action returns the model.Action configured into this renderer
func (w Stream) Action() *model.Action {
	return w.action
}

// Render generates the string value for this Stream
func (w Stream) Render() (template.HTML, error) {

	var buffer bytes.Buffer

	// Execute step (write HTML to buffer, update context)
	if err := DoPipeline(&w, &buffer, w.action.Steps, ActionMethodGet); err != nil {
		return "", derp.Report(derp.Wrap(err, "whisper.render.Stream.Render", "Error generating HTML"))
	}

	// Success!
	return template.HTML(buffer.String()), nil
}

func (w Stream) executeTemplate(wr io.Writer, name string, data interface{}) error {
	return w.template.HTMLTemplate.ExecuteTemplate(wr, name, data)
}

// object returns the model object associated with this renderer
func (w Stream) object() data.Object {
	return w.stream
}

func (w Stream) objectID() primitive.ObjectID {
	return w.stream.StreamID
}

// schema returns the validation schema associated with this renderer
func (w Stream) schema() schema.Schema {
	return w.template.Schema
}

func (w Stream) service() ModelService {
	return w.modelService
}

/*******************************************
 * ACTION SHORTCUTS
 *******************************************/

// View executes a separate view for this Stream
func (w Stream) View(actionID string) (template.HTML, error) {

	// Find and validate the action
	action := w.template.Action(actionID)

	if action == nil {
		return template.HTML(""), derp.NewNotFoundError("whisper.render.Stream.View", "Invalid Action", actionID)
	}

	// Create a new renderer (this will also validate the user's permissions)
	subStream, err := NewStream(w.factory(), w.context(), w.template, action, w.stream)

	if err != nil {
		return template.HTML(""), derp.Wrap(err, "whisper.render.Stream.View", "Error creating sub-renderer", action)
	}

	// Generate HTML template
	return subStream.Render()
}

/*******************************************
 * STREAM DATA
 *******************************************/

// StreamID returns the unique ID for the stream being rendered
func (w Stream) StreamID() string {
	return w.stream.StreamID.Hex()
}

// StreamID returns the unique ID for the stream being rendered
func (w Stream) ParentID() string {
	return w.stream.ParentID.Hex()
}

func (w Stream) TopLevelID() string {
	if len(w.stream.ParentIDs) == 0 {
		return w.stream.StreamID.Hex()
	}
	return w.stream.ParentIDs[0].Hex()
}

// StateID returns the current state of the stream being rendered
func (w Stream) StateID() string {
	return w.stream.StateID
}

// TemplateID returns the name of the template being used
func (w Stream) TemplateID() string {
	return w.stream.TemplateID
}

// Token returns the unique URL token for the stream being rendered
func (w Stream) Token() string {
	return w.stream.Token
}

// Label returns the Label for the stream being rendered
func (w Stream) Label() string {
	return w.stream.Label
}

// Description returns the description of the stream being rendered
func (w Stream) Description() string {
	return w.stream.Description
}

// Name of the person who created this Stream
func (w Stream) AuthorName() string {
	return w.stream.AuthorName
}

// PhotoURL of the person who created this Stream
func (w Stream) AuthorImage() string {
	return w.stream.AuthorImage
}

// Returns the body content as an HTML template
func (w Stream) Content() template.HTML {
	library := w.factory().ContentLibrary()
	result := nebula.View(library, &w.stream.Content)
	return template.HTML(result)
}

// Returns the body content as an HTML template
func (w Stream) ContentEditor() template.HTML {
	library := w.factory().ContentLibrary()
	result := nebula.Edit(library, &w.stream.Content, w.URL())
	return template.HTML(result)
}

// PublishDate returns the PublishDate of the stream being rendered
func (w Stream) PublishDate() int64 {
	return w.stream.PublishDate
}

// CreateDate returns the CreateDate of the stream being rendered
func (w Stream) CreateDate() int64 {
	return w.stream.CreateDate
}

// Rank returns the Rank of the stream being rendered
func (w Stream) Rank() int {
	return w.stream.Rank
}

// ThumbnailImage returns the thumbnail image URL of the stream being rendered
func (w Stream) ThumbnailImage() string {
	return w.stream.ThumbnailImage
}

// SourceURL returns the thumbnail image URL of the stream being rendered
func (w Stream) SourceURL() string {
	return w.stream.SourceURL
}

// Data returns the custom data map of the stream being rendered
func (w Stream) Data(value string) interface{} {
	return w.stream.Data[value]
}

// Tags returns the tags of the stream being rendered
func (w Stream) Tags() []string {
	return w.stream.Tags
}

// HasParent returns TRUE if the stream being rendered has a parend objec
func (w Stream) HasParent() bool {
	return w.stream.HasParent()
}

func (w Stream) IsCurrentStream() bool {
	return w.stream.Token == list.Head(w.context().Path(), "/")
}

func (w Stream) Roles() []string {
	authorization := getAuthorization(w.context())
	return w.stream.Roles(authorization)
}

/*******************************************
 * RELATED STREAMS
 *******************************************/

// Parent returns a Stream containing the parent of the current stream
func (w Stream) Parent(actionID string) (Stream, error) {

	var parent model.Stream

	streamService := w.factory().Stream()

	if err := streamService.LoadParent(w.stream, &parent); err != nil {
		return Stream{}, derp.Wrap(err, "whisper.renderer.Stream.Parent", "Error loading Parent")
	}

	renderer, err := NewStreamWithoutTemplate(w.factory(), w.context(), &parent, actionID)

	if err != nil {
		return Stream{}, derp.Wrap(err, "whisper.renderer.Stream.Parent", "Unable to create new Stream")
	}

	return renderer, nil
}

// PrevSibling returns the sibling Stream that immediately preceeds this one, based on the provided sort field
func (w Stream) PrevSibling(sort string, action string) (Stream, error) {

	criteria := exp.And(
		exp.Equal("parentId", w.stream.ParentID),
		exp.LessThan("sort", path.MustGet(w.stream, "sort")),
		exp.Equal("journal.deleteDate", 0),
	)

	sortOption := option.SortDesc(sort)

	return w.getFirstStream(criteria, sortOption, action), nil
}

// NextSibling returns the sibling Stream that immediately follows this one, based on the provided sort field
func (w Stream) NextSibling(sort string, action string) (Stream, error) {

	criteria := exp.And(
		exp.Equal("parentId", w.stream.ParentID),
		exp.GreaterThan("sort", path.MustGet(w.stream, "sort")),
		exp.Equal("journal.deleteDate", 0),
	)

	sortOption := option.SortAsc(sort)

	return w.getFirstStream(criteria, sortOption, action), nil
}

// FirstChild returns the first child Stream underneath this one, based on the provided sort field
func (w Stream) FirstChild(sort string, action string) (Stream, error) {

	criteria := exp.And(
		exp.Equal("parentId", w.stream.StreamID),
		exp.Equal("journal.deleteDate", 0),
	)

	sortOption := option.SortAsc(sort)

	return w.getFirstStream(criteria, sortOption, action), nil
}

// FirstChild returns the first child Stream underneath this one, based on the provided sort field
func (w Stream) LastChild(sort string, action string) (Stream, error) {

	criteria := exp.And(
		exp.Equal("parentId", w.stream.StreamID),
		exp.Equal("journal.deleteDate", 0),
	)

	sortOption := option.SortDesc(sort)

	return w.getFirstStream(criteria, sortOption, action), nil
}

// getFirstStream scans an iterator for the first stream allowed to this user.
// It is used internally by PrevSibling, NextSibling, FirstChild, and LastChild
func (w Stream) getFirstStream(criteria exp.Expression, sortOption option.Option, actionID string) Stream {

	streamService := w.factory().Stream()
	iterator, err := streamService.List(criteria, sortOption)

	if err != nil {
		derp.Report(derp.Wrap(err, "whisper.renderer.Stream.NextSibling", "Database error"))
		return Stream{}
	}

	var first model.Stream

	for iterator.Next(&first) {
		if result, err := NewStreamWithoutTemplate(w.factory(), w.context(), &first, actionID); err == nil {
			return result
		}
	}

	// Fall through means no streams are valid.  Return an empty renderer instead.
	return Stream{}
}

/*******************************************
 * RELATED RESULTSETS
 *******************************************/

// Siblings returns all Sibling Streams
func (w Stream) Siblings() QueryBuilder {
	return w.makeQueryBuilder(exp.Equal("parentId", w.stream.ParentID))
}

// Children returns all child Streams
func (w Stream) Children() QueryBuilder {
	return w.makeQueryBuilder(exp.Equal("parentId", w.stream.StreamID))
}

// makeQueryBuilder returns a fully initialized QueryBuilder
func (w Stream) makeQueryBuilder(criteria exp.Expression) QueryBuilder {

	query := builder.NewBuilder().
		Int("journal.createDate").
		Int("publishDate").
		Int("expirationDate").
		Int("rank").
		String("label")

	criteria = exp.And(
		criteria,
		query.Evaluate(w.context().Request().URL.Query()),
		exp.Equal("journal.deleteDate", 0),
	)

	result := NewQueryBuilder(w.factory(), w.context(), w.factory().Stream(), criteria)
	result.SortField = w.template.ChildSortType
	result.SortDirection = w.template.ChildSortDirection

	return result
}

/*******************************************
 * ATTACHMENTS
 *******************************************/

// Reference to the first file attached to this stream
func (w Stream) Attachment() (model.Attachment, error) {

	var attachment model.Attachment

	attachmentService := w.factory().Attachment()
	iterator, err := attachmentService.ListByObjectID(w.stream.StreamID)

	if err != nil {
		return attachment, derp.Wrap(err, "whisper.renderer.Stream.Attachments", "Error listing attachments")
	}

	// Just get a single attachment from the Iterator
	iterator.Next(&attachment)

	return attachment, nil
}

// Attachments lists all attachments for this stream.
func (w Stream) Attachments() ([]model.Attachment, error) {

	result := []model.Attachment{}
	attachmentService := w.factory().Attachment()
	iterator, err := attachmentService.ListByObjectID(w.stream.StreamID)

	if err != nil {
		return result, derp.Wrap(err, "whisper.renderer.Stream.Attachments", "Error listing attachments")
	}

	attachment := new(model.Attachment)
	for iterator.Next(attachment) {
		result = append(result, *attachment)
		attachment = new(model.Attachment)
	}

	return result, nil
}

/*******************************************
 * ACCESS PERMISSIONS
 *******************************************/

// UserCan returns TRUE if this Request is authorized to access the requested view
func (w Stream) UserCan(actionID string) bool {

	action := w.template.Action(actionID)

	if action == nil {
		return false
	}

	authorization := getAuthorization(w.context())

	return action.UserCan(w.stream, authorization)
}

// CanCreate returns all of the templates that can be created underneath
// the current stream.
func (w Stream) CanCreate() []model.Option {

	templateService := w.factory().Template()
	return templateService.ListByContainer(w.template.TemplateID)
}

func (w Stream) draftRenderer() (Stream, error) {

	var draft model.Stream
	draftService := w.factory().StreamDraft()

	// Load the draft of the object
	if err := draftService.LoadByID(w.stream.StreamID, &draft); err != nil {
		return Stream{}, derp.Wrap(err, "whisper.service.Stream.draftRenderer", "Error loading draft")
	}

	// Make a duplicate of this renderer.  Same object, template, action settings
	return Stream{
		stream:       &draft,
		modelService: draftService,
		template:     w.template,
		action:       w.action,
		Common:       NewCommon(w.factory(), w.ctx),
	}, nil
}

/*******************************************
 * MISC HELPER FUNCTIONS
 *******************************************/

func (w Stream) setAuthor() error {

	user, err := w.getUser()

	if err != nil {
		return derp.Wrap(err, "whisper.render.Stream.setAuthor", "Error loading User")
	}

	w.stream.AuthorID = user.UserID
	w.stream.AuthorName = user.DisplayName
	w.stream.AuthorImage = user.AvatarURL

	return nil
}
