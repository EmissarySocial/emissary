package render

import (
	"bytes"
	"html/template"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	builder "github.com/benpate/exp-builder"
	"github.com/benpate/form"
	htmlconv "github.com/benpate/rosetta/html"
	"github.com/benpate/rosetta/list"
	"github.com/benpate/rosetta/schema"
	"github.com/benpate/steranko"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Stream wraps a model.Stream object and provides functions that make it easy to render an HTML template with it.
type Stream struct {
	modelService service.ModelService // Service to use to access streams (could be Stream or StreamDraft)
	stream       *model.Stream        // The Stream to be displayed

	Common
}

/******************************************
 * CONSTRUCTORS
 ******************************************/

// NewStream creates a new object that can generate HTML for a specific stream/view
func NewStream(factory Factory, ctx *steranko.Context, template *model.Template, stream *model.Stream, actionID string) (Stream, error) {

	const location = "render.NewStream"

	// Verify the requested action
	action := template.Action(actionID)

	if action == nil {
		return Stream{}, derp.NewBadRequestError(location, "Invalid action", actionID)
	}

	// Verify user's authorization to perform this Action on this Stream
	authorization := getAuthorization(ctx)

	if !action.UserCan(stream, &authorization) {
		if authorization.IsAuthenticated() {
			return Stream{}, derp.NewForbiddenError(location, "Forbidden")
		} else {
			return Stream{}, derp.NewUnauthorizedError(location, "Anonymous user is not authorized to perform this action", actionID)
		}
	}

	// Success.  Populate Stream
	return Stream{
		modelService: factory.Stream(),
		stream:       stream,
		Common:       NewCommon(factory, ctx, template, action, actionID),
	}, nil
}

// NewStreamWithoutTemplate creates a new object that can generate HTML for a specific stream/view
func NewStreamWithoutTemplate(factory Factory, ctx *steranko.Context, stream *model.Stream, actionID string) (Stream, error) {

	// Use the template service to look up the correct template
	templateService := factory.Template()

	template, err := templateService.Load(stream.TemplateID)

	if err != nil {
		return Stream{}, derp.Wrap(err, "render.NewStreamWithoutTemplate", "Error loading Template", stream)
	}

	// And look up a valid action (cannot be empty)
	action := template.Action(actionID)

	if action == nil {
		return Stream{}, derp.NewNotFoundError("render.NewStreamWithoutTemplate", "Unrecognized Action", actionID)
	}

	// Return a fully populated service
	return NewStream(factory, ctx, template, stream, actionID)
}

/******************************************
 * RENDERER INTERFACE
 ******************************************/

// Render generates the string value for this Stream
func (w Stream) Render() (template.HTML, error) {

	var buffer bytes.Buffer

	// Execute step (write HTML to buffer, update context)
	if err := Pipeline(w.action.Steps).Get(w.factory(), &w, &buffer); err != nil {
		return "", derp.Report(derp.Wrap(err, "render.Stream.Render", "Error generating HTML"))
	}

	// Success!
	return template.HTML(buffer.String()), nil
}

// object returns the model object associated with this renderer
func (w Stream) object() data.Object {
	return w.stream
}

func (w Stream) objectID() primitive.ObjectID {
	return w.stream.StreamID
}

func (w Stream) objectType() string {
	return "Stream"
}

// schema returns the validation schema associated with this renderer
func (w Stream) schema() schema.Schema {
	return w.template().Schema
}

func (w Stream) service() service.ModelService {
	return w.modelService
}

// templateRole returns the role that this Stream's Template plays in the system.
// This is used to determine what kinds of Streams can be added underneath this one as children.
func (w Stream) templateRole() string {
	return w.template().Role
}

/******************************************
 * ACTION SHORTCUTS
 ******************************************/

// View executes a separate view for this Stream
func (w Stream) View(actionID string) (template.HTML, error) {

	const location = "render.Stream.View"

	// Create a new renderer (this will also validate the user's permissions)
	subStream, err := NewStream(w.factory(), w.context(), w.template(), w.stream, actionID)

	if err != nil {
		return template.HTML(""), derp.Wrap(err, location, "Error creating sub-renderer")
	}

	// Generate HTML template
	return subStream.Render()
}

/******************************************
 * STREAM DATA
 ******************************************/

// StreamID returns the unique ID for the stream being rendered
func (w Stream) StreamID() string {
	return w.stream.StreamID.Hex()
}

// StreamID returns the unique ID for the stream being rendered
func (w Stream) ParentID() string {
	return w.stream.ParentID.Hex()
}

// NavigationID returns the unique ID of the top-level stream in this stream's hierarchy
func (w Stream) NavigationID() string {
	if w.stream.NavigationID == "" {
		return w.stream.StreamID.Hex()
	}
	return w.stream.NavigationID
}

// PageTitle returns the Label for the stream being rendered
func (w Stream) PageTitle() string {
	return w.stream.Document.Label
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

// Document returns the DocumentLink record for this stream
func (w Stream) Document() model.DocumentLink {
	return w.stream.Document
}

// Label returns the Label for the stream being rendered
func (w Stream) Label() string {
	return w.stream.Document.Label
}

// Summary returns the description of the stream being rendered
func (w Stream) Summary() string {
	return w.stream.Document.Summary
}

// SummaryHTML returns the description of the stream being rendered
func (w Stream) SummaryHTML() template.HTML {
	return template.HTML(w.stream.Document.Summary)
}

// SummarySummary returns a plaintext summary (<200 characters) of the stream's description
func (w Stream) ShortSummary() string {
	return htmlconv.Summary(w.stream.Document.Summary)
}

// ImageURL returns the thumbnail image URL of the stream being rendered
func (w Stream) ImageURL() string {
	return w.stream.Document.ImageURL
}

// Permalink returns a complete URL for this stream
func (w Stream) Permalink() string {
	return w.Host() + "/" + w.stream.StreamID.Hex()
}

// Author returns the Author record for this stream
func (w Stream) Author() model.PersonLink {
	return w.stream.Document.Author
}

// AuthorURL returns the public URL of the person who created this Stream
func (w Stream) AuthorURL() string {
	return w.stream.Document.Author.ProfileURL
}

// Name of the person who created this Stream
func (w Stream) AuthorName() string {
	return w.stream.Document.Author.Name
}

// PhotoURL of the person who created this Stream
func (w Stream) AuthorImage() string {
	return w.stream.Document.Author.ImageURL
}

func (w Stream) AuthorEmail() string {
	return w.stream.Document.Author.EmailAddress
}

// Returns the body content as an HTML template
func (w Stream) ContentHTML() template.HTML {
	return template.HTML(w.stream.Content.HTML)
}

func (w Stream) ContentRaw() string {

	if w.stream.Content.Raw == "" {
		return "{}"
	}

	return w.stream.Content.Raw
}

// CreateDate returns the CreateDate of the stream being rendered
func (w Stream) CreateDate() int64 {
	return w.stream.CreateDate
}

// PublishDate returns the PublishDate of the stream being rendered
func (w Stream) PublishDate() int64 {

	if w.stream.PublishDate > 0 {
		return w.stream.PublishDate
	}

	return w.stream.CreateDate
}

// UpdateDate returns the UpdateDate of the stream being rendered
func (w Stream) UpdateDate() int64 {
	return w.stream.UpdateDate
}

// Rank returns the Rank of the stream being rendered
func (w Stream) Rank() int {
	return w.stream.Rank
}

// Data returns the custom data map of the stream being rendered
func (w Stream) Data(value string) any {
	return w.stream.Data[value]
}

// HasParent returns TRUE if the stream being rendered has a parend objec
func (w Stream) HasParent() bool {
	return w.stream.HasParent()
}

func (w Stream) InReplyTo() model.DocumentLink {
	return w.stream.InReplyTo
}

// IsReply returns TRUE if this stream is marked as a reply to another stream or resource
func (w Stream) IsReply() bool {
	return (w.stream.InReplyTo.IsEmpty())
}

// IsNew returns TRUE if this stream has not been saved to the database
func (w Stream) IsNew() bool {
	return w.stream.IsNew()
}

// IsEmpty returns TRUE if the stream is an empty placeholder.
func (w Stream) IsEmpty() bool {
	return (w.stream == nil) || (w.stream.StreamID == primitive.NilObjectID)
}

func (w Stream) IsCurrentStream() bool {
	return w.stream.Token == list.Slash(w.context().Path()).First()
}

func (w Stream) Roles() []string {
	authorization := w.authorization()
	return w.stream.Roles(&authorization)
}

/******************************************
 * RELATED STREAMS
 ******************************************/

// Parent returns a Stream containing the parent of the current stream
func (w Stream) Parent(actionID string) (Stream, error) {

	const location = "renderer.Stream.Parent"

	var parent model.Stream

	streamService := w.factory().Stream()

	if err := streamService.LoadParent(w.stream, &parent); err != nil {
		return Stream{}, derp.Wrap(err, location, "Error loading Parent")
	}

	renderer, err := NewStreamWithoutTemplate(w.factory(), w.context(), &parent, actionID)

	if err != nil {
		return Stream{}, derp.Wrap(err, location, "Unable to create new Stream")
	}

	return renderer, nil
}

// PrevSibling returns the sibling Stream that immediately preceeds this one, based on the provided sort field
func (w Stream) PrevSibling(sortField string, action string) (Stream, error) {

	criteria := exp.Equal("parentId", w.stream.ParentID).
		AndLessThan(sortField, w.stream.GetSort(sortField))

	sortOption := option.SortDesc(sortField)

	return w.getFirstStream(criteria, sortOption, action), nil
}

// NextSibling returns the sibling Stream that immediately follows this one, based on the provided sort field
func (w Stream) NextSibling(sortField string, action string) (Stream, error) {

	criteria := exp.Equal("parentId", w.stream.ParentID).
		AndGreaterThan(sortField, w.stream.GetSort(sortField))

	sortOption := option.SortAsc(sortField)

	return w.getFirstStream(criteria, sortOption, action), nil
}

// FirstChild returns the first child Stream underneath this one, based on the provided sort field
func (w Stream) FirstChild(sort string, action string) (Stream, error) {

	criteria := exp.Equal("parentId", w.stream.StreamID)

	sortOption := option.SortAsc(sort)

	return w.getFirstStream(criteria, sortOption, action), nil
}

// FirstChild returns the first child Stream underneath this one, based on the provided sort field
func (w Stream) LastChild(sort string, action string) (Stream, error) {

	criteria := exp.Equal("parentId", w.stream.StreamID)
	sortOption := option.SortDesc(sort)

	return w.getFirstStream(criteria, sortOption, action), nil
}

// getFirstStream scans an iterator for the first stream allowed to this user.
// It is used internally by PrevSibling, NextSibling, FirstChild, and LastChild
func (w Stream) getFirstStream(criteria exp.Expression, sortOption option.Option, actionID string) Stream {

	criteria = w.withViewPermission(criteria)

	streamService := w.factory().Stream()
	iterator, err := streamService.List(criteria, sortOption, option.FirstRow())

	if err != nil {
		derp.Report(derp.Wrap(err, "renderer.Stream.NextSibling", "Database error"))
		return Stream{}
	}

	var first model.Stream

	if iterator.Next(&first) {
		if result, err := NewStreamWithoutTemplate(w.factory(), w.context(), &first, actionID); err == nil {
			return result
		}
	}

	// Fall through means no streams are valid.  Return an empty renderer instead.
	return Stream{}
}

func (w Stream) Mentions() ([]model.Mention, error) {
	mentionService := w.factory().Mention()
	return mentionService.QueryByObjectID(w.stream.StreamID)
}

/******************************************
 * RELATED RESULTSETS
 ******************************************/

// Siblings returns all Streams that have the same "parent" as the current Stream
func (w Stream) Siblings() QueryBuilder[model.StreamSummary] {
	return w.makeStreamQueryBuilder(exp.Equal("parentId", w.stream.ParentID))
}

// Children returns all Streams with a "parent" is the current Stream
func (w Stream) Children() QueryBuilder[model.StreamSummary] {
	return w.makeStreamQueryBuilder(exp.Equal("parentId", w.stream.StreamID))
}

// Replies returns all Streams that are "in reply to" the current Stream
func (w Stream) Replies() QueryBuilder[model.StreamSummary] {
	return w.makeStreamQueryBuilder(exp.Equal("inReplyTo", w.stream.StreamID.Hex()))
}

// makeStreamQueryBuilder returns a fully initialized RenderBuilder
func (w Stream) makeStreamQueryBuilder(criteria exp.Expression) QueryBuilder[model.StreamSummary] {

	queryBuilder := builder.NewBuilder().
		Int("journal.createDate").
		Int("journal.updateDate").
		Int("publishDate").
		Int("expirationDate").
		Int("rank").
		String("label")

	queryValues := w.context().Request().URL.Query()
	query := queryBuilder.Evaluate(queryValues)

	criteria = w.withViewPermission(
		criteria.And(query),
	)

	result := NewQueryBuilder[model.StreamSummary](w._factory.Stream(), criteria)
	result.SortField = w.template().ChildSortType
	result.SortDirection = w.template().ChildSortDirection

	return result
}

/******************************************
 * ATTACHMENTS
 ******************************************/

// Reference to the first file attached to this stream
func (w Stream) Attachment() (model.Attachment, error) {
	return w.factory().Attachment().LoadFirstByObjectID(model.AttachmentTypeStream, w.stream.StreamID)
}

// Attachments lists all attachments for this stream.
func (w Stream) Attachments() ([]model.Attachment, error) {
	return w.factory().Attachment().QueryByObjectID(model.AttachmentTypeStream, w.stream.StreamID)
}

/******************************************
 * SUBSCRIPTIONS
 ******************************************/

func (w Stream) Following() ([]model.Following, error) {

	result := []model.Following{}
	followingService := w.factory().Following()

	iterator, err := followingService.ListByUserID(w.AuthenticatedID())

	if err != nil {
		return result, derp.Wrap(err, "renderer.Stream.Following", "Error listing following")
	}

	following := model.NewFollowing()

	for iterator.Next(&following) {
		result = append(result, following)
		following = model.NewFollowing()
	}

	return result, nil
}

/******************************************
 * ACCESS PERMISSIONS
 ******************************************/

// UserCan returns TRUE if this Request is authorized to access the requested view
func (w Stream) UserCan(actionID string) bool {

	action := w.template().Action(actionID)

	if action == nil {
		return false
	}

	authorization := w.authorization()

	return action.UserCan(w.stream, &authorization)
}

// CanCreate returns all of the templates that can be created underneath
// the current stream.
func (w Stream) CanCreate() []form.LookupCode {

	templateService := w.factory().Template()
	return templateService.ListByContainer(w.template().TemplateID)
}

// draftRenderer returns a new render.Stream that is bound to the
// draft service, and a draft copy of the current stream.
func (w Stream) draftRenderer() (Stream, error) {

	var draft model.Stream
	draftService := w.factory().StreamDraft()

	// Load the draft of the object
	if err := draftService.LoadByID(w.stream.StreamID, &draft); err != nil {
		return Stream{}, derp.Wrap(err, "service.Stream.draftRenderer", "Error loading draft")
	}

	// Make a duplicate of this renderer.  Same object, template, action settings
	return Stream{
		stream:       &draft,
		modelService: draftService,
		Common:       NewCommon(w._factory, w._context, w._template, w.action, w.actionID),
	}, nil
}
