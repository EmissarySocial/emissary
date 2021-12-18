package render

import (
	"bytes"
	"html/template"
	"io"
	"net/http"

	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/exp/builder"
	"github.com/benpate/ghost/model"
	"github.com/benpate/list"
	"github.com/benpate/path"
	"github.com/benpate/schema"
	"github.com/benpate/steranko"
)

// Stream wraps a model.Stream object and provides functions that make it easy to render an HTML template with it.
type Stream struct {
	template *model.Template // Template that the Stream uses
	stream   *model.Stream   // Stream to be displayed
	actionID string

	Common
}

/*******************************************
 * CONSTRUCTORS
 *******************************************/

// NewStream creates a new object that can generate HTML for a specific stream/view
func NewStream(factory Factory, ctx *steranko.Context, template *model.Template, stream *model.Stream, actionID string) (Stream, error) {

	// Try to find requested Action
	action, ok := template.Action(actionID)

	if !ok {
		return Stream{}, derp.New(http.StatusBadRequest, "ghost.render.NewStream", "Invalid action")
	}

	// Verify user's authorization to perform this Action on this Stream
	authorization := getAuthorization(ctx)

	if !action.UserCan(stream, authorization) {
		return Stream{}, derp.New(http.StatusForbidden, "ghost.render.NewStream", "Forbidden")
	}

	// Success.  Populate Stream
	return Stream{
		stream:   stream,
		template: template,
		actionID: actionID,
		Common:   NewCommon(factory, ctx),
	}, nil
}

// NewStreamWithoutTemplate creates a new object that can generate HTML for a specific stream/view
func NewStreamWithoutTemplate(factory Factory, ctx *steranko.Context, stream *model.Stream, actionID string) (Stream, error) {

	templateService := factory.Template()

	template, err := templateService.Load(stream.TemplateID)

	if err != nil {
		return Stream{}, derp.Wrap(err, "ghost.render.NewStreamWithoutTemplate", "Error loading Template", stream)
	}

	return NewStream(factory, ctx, template, stream, actionID)
}

/*******************************************
 * PATH INTERFACE
 * (not available via templates)
 *******************************************/

func (st Stream) GetPath(p path.Path) (interface{}, error) {
	return st.stream.GetPath(p)
}

func (st Stream) SetPath(p path.Path, value interface{}) error {
	return st.stream.SetPath(p, value)
}

/*******************************************
 * RENDERER INTERFACE
 *******************************************/

// ActionID returns the name of the action being performed
func (st Stream) ActionID() string {
	return st.actionID
}

// Action returns the model.Action configured into this renderer
func (st Stream) Action() (model.Action, bool) {
	return st.template.Action(st.actionID)
}

// Render generates the string value for this Stream
func (st Stream) Render() (template.HTML, error) {

	var buffer bytes.Buffer

	action, _ := st.Action() // This is OK becuase we've already tested for the action's presence

	// Execute step (write HTML to buffer, update context)
	if err := DoPipeline(st.factory, &st, &buffer, action.Steps, ActionMethodGet); err != nil {
		return "", derp.Report(derp.Wrap(err, "ghost.render.Stream.Render", "Error generating HTML"))
	}

	// Success!
	return template.HTML(buffer.String()), nil
}

func (st Stream) executeTemplate(wr io.Writer, name string, data interface{}) error {
	return st.template.HTMLTemplate.ExecuteTemplate(wr, name, data)
}

// object returns the model object associated with this renderer
func (st Stream) object() data.Object {
	return st.stream
}

// schema returns the validation schema associated with this renderer
func (st Stream) schema() schema.Schema {
	return st.template.Schema
}

func (st Stream) common() Common {
	return st.Common
}

/*******************************************
 * ACTION SHORTCUTS
 *******************************************/

// View executes a separate view for this Stream
func (st Stream) View(action string) (template.HTML, error) {

	subStream, err := NewStream(st.factory, st.ctx, st.template, st.stream, action)

	if err != nil {
		return template.HTML(""), derp.Wrap(err, "ghost.render.Stream.View", "Error creating sub-renderer", action)
	}

	return subStream.Render()
}

/*******************************************
 * STREAM DATA
 *******************************************/

// StreamID returns the unique ID for the stream being rendered
func (st Stream) StreamID() string {
	return st.stream.StreamID.Hex()
}

// StreamID returns the unique ID for the stream being rendered
func (st Stream) ParentID() string {
	return st.stream.ParentID.Hex()
}

func (st Stream) TopLevelID() string {
	if len(st.stream.ParentIDs) == 0 {
		return st.stream.StreamID.Hex()
	}
	return st.stream.ParentIDs[0].Hex()
}

// StateID returns the current state of the stream being rendered
func (st Stream) StateID() string {
	return st.stream.StateID
}

// TemplateID returns the name of the template being used
func (st Stream) TemplateID() string {
	return st.stream.TemplateID
}

// Token returns the unique URL token for the stream being rendered
func (st Stream) Token() string {
	return st.stream.Token
}

// Label returns the Label for the stream being rendered
func (st Stream) Label() string {
	return st.stream.Label
}

// Description returns the description of the stream being rendered
func (st Stream) Description() string {
	return st.stream.Description
}

// Name of the person who created this Stream
func (st Stream) AuthorName() string {
	return st.stream.AuthorName
}

// PhotoURL of the person who created this Stream
func (st Stream) AuthorImage() string {
	return st.stream.AuthorImage
}

// Returns the body content as an HTML template
func (st Stream) Content() template.HTML {
	result := st.stream.Content.View()
	return template.HTML(result)
}

// Returns editable HTML for the body content (requires `editable` flat)
func (st Stream) ContentEditor() template.HTML {
	result := st.stream.Content.Edit("/" + st.Token() + "/draft")
	return template.HTML(result)
}

// PublishDate returns the PublishDate of the stream being rendered
func (st Stream) PublishDate() int64 {
	return st.stream.PublishDate
}

// CreateDate returns the CreateDate of the stream being rendered
func (st Stream) CreateDate() int64 {
	return st.stream.CreateDate
}

// ThumbnailImage returns the thumbnail image URL of the stream being rendered
func (st Stream) ThumbnailImage() string {
	return st.stream.ThumbnailImage
}

// SourceURL returns the thumbnail image URL of the stream being rendered
func (st Stream) SourceURL() string {
	return st.stream.SourceURL
}

// Data returns the custom data map of the stream being rendered
func (st Stream) Data(value string) interface{} {
	return st.stream.Data[value]
}

// Tags returns the tags of the stream being rendered
func (st Stream) Tags() []string {
	return st.stream.Tags
}

// HasParent returns TRUE if the stream being rendered has a parend objec
func (st Stream) HasParent() bool {
	return st.stream.HasParent()
}

func (st Stream) IsCurrentStream() bool {
	return st.stream.Token == list.Head(st.ctx.Path(), "/")
}

func (st Stream) Roles() []string {
	authorization := getAuthorization(st.ctx)
	return st.stream.Roles(authorization)
}

/*******************************************
 * RELATED STREAMS
 *******************************************/

// Parent returns a Stream containing the parent of the current stream
func (st Stream) Parent(actionID string) (Stream, error) {

	var parent model.Stream

	streamService := st.factory.Stream()

	if err := streamService.LoadParent(st.stream, &parent); err != nil {
		return Stream{}, derp.Wrap(err, "ghost.renderer.Stream.Parent", "Error loading Parent")
	}

	renderer, err := NewStreamWithoutTemplate(st.factory, st.ctx, &parent, actionID)

	if err != nil {
		return Stream{}, derp.Wrap(err, "ghost.renderer.Stream.Parent", "Unable to create new Stream")
	}

	return renderer, nil
}

// PrevSibling returns the sibling Stream that immediately preceeds this one, based on the provided sort field
func (st Stream) PrevSibling(sort string, action string) (Stream, error) {

	criteria := exp.And(
		exp.Equal("parentId", st.stream.ParentID),
		exp.LessThan("sort", path.MustGet(st.stream, "sort")),
		exp.Equal("journal.deleteDate", 0),
	)

	sortOption := option.SortDesc(sort)

	return st.makeFirstStream(criteria, sortOption, action), nil
}

// NextSibling returns the sibling Stream that immediately follows this one, based on the provided sort field
func (st Stream) NextSibling(sort string, action string) (Stream, error) {

	criteria := exp.And(
		exp.Equal("parentId", st.stream.ParentID),
		exp.GreaterThan("sort", path.MustGet(st.stream, "sort")),
		exp.Equal("journal.deleteDate", 0),
	)

	sortOption := option.SortAsc(sort)

	return st.makeFirstStream(criteria, sortOption, action), nil
}

// FirstChild returns the first child Stream underneath this one, based on the provided sort field
func (st Stream) FirstChild(sort string, action string) (Stream, error) {

	criteria := exp.And(
		exp.Equal("parentId", st.stream.StreamID),
		exp.Equal("journal.deleteDate", 0),
	)

	sortOption := option.SortAsc(sort)

	return st.makeFirstStream(criteria, sortOption, action), nil
}

// FirstChild returns the first child Stream underneath this one, based on the provided sort field
func (st Stream) LastChild(sort string, action string) (Stream, error) {

	criteria := exp.And(
		exp.Equal("parentId", st.stream.StreamID),
		exp.Equal("journal.deleteDate", 0),
	)

	sortOption := option.SortDesc(sort)

	return st.makeFirstStream(criteria, sortOption, action), nil
}

// makeFirstStream scans an iterator for the first stream allowed to this user
func (st Stream) makeFirstStream(criteria exp.Expression, sortOption option.Option, actionID string) Stream {

	streamService := st.factory.Stream()
	iterator, err := streamService.List(criteria, sortOption)

	if err != nil {
		derp.Report(derp.Wrap(err, "ghost.renderer.Stream.NextSibling", "Database error"))
		return Stream{}
	}

	var first model.Stream

	for iterator.Next(&first) {
		if result, err := NewStreamWithoutTemplate(st.factory, st.ctx, &first, actionID); err == nil {
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
func (st Stream) Siblings() QueryBuilder {
	return st.makeQueryBuilder(exp.Equal("parentId", st.stream.ParentID))
}

// Children returns all child Streams
func (st Stream) Children() QueryBuilder {
	return st.makeQueryBuilder(exp.Equal("parentId", st.stream.StreamID))
}

// makeQueryBuilder returns a fully initialized QueryBuilder
func (st Stream) makeQueryBuilder(criteria exp.Expression) QueryBuilder {

	query := builder.NewBuilder().
		Int("journal.createDate").
		Int("publishDate").
		Int("expirationDate").
		Int("rank").
		String("label")

	criteria = exp.And(
		criteria,
		query.Evaluate(st.ctx.Request().URL.Query()),
		exp.Equal("journal.deleteDate", 0),
	)

	result := NewQueryBuilder(st.factory, st.ctx, st.factory.Stream(), criteria)
	result.SortField = st.template.ChildSortType
	result.SortDirection = st.template.ChildSortDirection

	return result
}

/*******************************************
 * ATTACHMENTS
 *******************************************/

// Reference to the first file attached to this stream
func (st Stream) Attachment() (model.Attachment, error) {

	var attachment model.Attachment

	attachmentService := st.factory.Attachment()
	iterator, err := attachmentService.ListByStream(st.stream.StreamID)

	if err != nil {
		return attachment, derp.Wrap(err, "ghost.renderer.Stream.Attachments", "Error listing attachments")
	}

	// Just get a single attachment from the Iterator
	iterator.Next(&attachment)

	return attachment, nil
}

// Attachments lists all attachments for this stream.
func (st Stream) Attachments() ([]model.Attachment, error) {

	result := []model.Attachment{}
	attachmentService := st.factory.Attachment()
	iterator, err := attachmentService.ListByStream(st.stream.StreamID)

	if err != nil {
		return result, derp.Wrap(err, "ghost.renderer.Stream.Attachments", "Error listing attachments")
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
func (st Stream) UserCan(actionID string) bool {

	action, ok := st.template.Action(actionID)

	if !ok {
		return false
	}

	authorization := getAuthorization(st.ctx)

	return action.UserCan(st.stream, authorization)
}

// CanCreate returns all of the templates that can be created underneath
// the current stream.
func (st Stream) CanCreate() []model.Option {

	templateService := st.factory.Template()
	return templateService.ListByContainer(st.template.TemplateID)
}

/*******************************************
 * MISC HELPER FUNCTIONS
 *******************************************/

func (st Stream) setAuthor() error {

	user, err := st.getUser()

	if err != nil {
		return derp.Wrap(err, "ghost.render.Stream.setAuthor", "Error loading User")
	}

	st.stream.AuthorID = user.UserID
	st.stream.AuthorName = user.DisplayName
	st.stream.AuthorImage = user.AvatarURL

	return nil
}
