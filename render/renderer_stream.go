package render

import (
	"bytes"
	"html/template"
	"io"
	"strings"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	builder "github.com/benpate/exp-builder"
	"github.com/benpate/form"
	htmlconv "github.com/benpate/rosetta/html"
	"github.com/benpate/rosetta/list"
	"github.com/benpate/rosetta/path"
	"github.com/benpate/rosetta/schema"
	"github.com/benpate/steranko"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Stream wraps a model.Stream object and provides functions that make it easy to render an HTML template with it.
type Stream struct {
	modelService ModelService    // Service to use to access streams (could be Stream or StreamDraft)
	template     *model.Template // Template that the Stream uses
	stream       *model.Stream   // The Stream to be displayed

	Common
}

/*******************************************
 * CONSTRUCTORS
 *******************************************/

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
		template:     template,
		Common:       NewCommon(factory, ctx, action, actionID),
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

/*******************************************
 * RENDERER INTERFACE
 *******************************************/

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

func (w Stream) executeTemplate(wr io.Writer, name string, data any) error {
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

	const location = "render.Stream.View"

	// Create a new renderer (this will also validate the user's permissions)
	subStream, err := NewStream(w.factory(), w.context(), w.template, w.stream, actionID)

	if err != nil {
		return template.HTML(""), derp.Wrap(err, location, "Error creating sub-renderer")
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

// PageTitle returns the Label for the stream being rendered
func (w Stream) PageTitle() string {
	return w.stream.Label
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

// DescriptionHTML returns the description of the stream being rendered
func (w Stream) DescriptionHTML() template.HTML {
	return template.HTML(w.stream.Description)
}

// DescriptionSummary returns a plaintext summary (<200 characters) of the stream's description
func (w Stream) DescriptionSummary() string {
	return htmlconv.Summary(w.stream.Description)
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
func (w Stream) ContentHTML() template.HTML {
	return template.HTML(w.stream.Content.HTML)
}

func (w Stream) ContentRaw() string {

	if w.stream.Content.Raw == "" {
		return "{}"
	}

	return w.stream.Content.Raw
}

/*/ Returns the body content as an HTML template
func (w Stream) ContentEditor() template.HTML {
	library := w.factory().ContentLibrary()
	result := nebula.Edit(library, &w.stream.Content, w.URL())
	return template.HTML(result)
}*/

// CreateDate returns the CreateDate of the stream being rendered
func (w Stream) CreateDate() int64 {
	return w.stream.CreateDate
}

// PublishDate returns the PublishDate of the stream being rendered
func (w Stream) PublishDate() int64 {
	return w.stream.PublishDate
}

// UpdateDate returns the UpdateDate of the stream being rendered
func (w Stream) UpdateDate() int64 {
	return w.stream.UpdateDate
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

// Permalink returns a complete URL for this stream
func (w Stream) Permalink() string {
	return w.Protocol() + w.Hostname() + "/" + w.stream.StreamID.Hex()
}

// Data returns the custom data map of the stream being rendered
func (w Stream) Data(value string) any {
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

// IsReply returns TRUE if this stream is marked as a reply to another stream or resource
func (w Stream) IsReply() bool {
	return (w.stream.InReplyTo != "")
}

// ThreadID returns the unique ID of the parent thread for this stream.
// If this stream is a reply to a previous stream, then that "in-reply-to"
// ID is returned.  Otherwise, the StreamID of this Stream is returned.
func (w Stream) ThreadID() string {
	if replyID := w.stream.InReplyTo; replyID != "" {
		return replyID
	}
	return w.stream.StreamID.Hex()
}

// IsEmpty returns TRUE if the stream is an empty placeholder.
func (w Stream) IsEmpty() bool {
	return (w.stream == nil) || (w.stream.StreamID == primitive.NilObjectID)
}

func (w Stream) IsCurrentStream() bool {
	return w.stream.Token == list.Slash(w.context().Path()).Head()
}

func (w Stream) Roles() []string {
	authorization := w.authorization()
	return w.stream.Roles(&authorization)
}

/*******************************************
 * RELATED STREAMS
 *******************************************/

// Features renders the "feature" action for every child stream
func (w Stream) Features() (template.HTML, error) {

	const location = "renderer.Stream.Features"

	streamService := w.factory().Stream()

	features, err := streamService.ListFeatures(w.stream.StreamID)

	if err != nil {
		return template.HTML(""), derp.Wrap(err, location, "Error getting features from database")
	}

	stream := model.NewStream()

	var buffer strings.Builder

	// For each feature of this Stream...
	for features.Next(&stream) {

		// Try to get a renderer for the feature (should always happen)
		renderer, err := NewStreamWithoutTemplate(w.factory(), w.context(), &stream, "feature")

		if err != nil {
			derp.Report(derp.Wrap(err, location, "Error getting feature renderer"))
			continue
		}

		// Try to render the feature (should always happen)
		fragment, err := renderer.Render()

		if err != nil {
			derp.Report(derp.Wrap(err, location, "Error rendering feature"))
			continue
		}

		// Append the feature's HTML fragment to the result
		buffer.WriteString(`<article class="feature">`)
		buffer.WriteString(string(fragment))
		buffer.WriteString(`</article>`)

		// Reset the target object for the next loop
		stream = model.NewStream()
	}

	return template.HTML(buffer.String()), nil
}

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
		AndLessThan(sortField, path.Get(w.stream, sortField))

	sortOption := option.SortDesc(sortField)

	return w.getFirstStream(criteria, sortOption, action), nil
}

// NextSibling returns the sibling Stream that immediately follows this one, based on the provided sort field
func (w Stream) NextSibling(sortField string, action string) (Stream, error) {

	criteria := exp.Equal("parentId", w.stream.ParentID).
		AndGreaterThan(sortField, path.Get(w.stream, sortField))

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

/*******************************************
 * RELATED RESULTSETS
 *******************************************/

// Siblings returns all Streams that have the same "parent" as the current Stream
func (w Stream) Siblings() QueryBuilder {
	return w.makeQueryBuilder(exp.Equal("parentId", w.stream.ParentID))
}

// Children returns all Streams with a "parent" is the current Stream
func (w Stream) Children() QueryBuilder {
	return w.makeQueryBuilder(exp.Equal("parentId", w.stream.StreamID))
}

// Replies returns all Streams that are "in reply to" the current Stream
func (w Stream) Replies() QueryBuilder {
	return w.makeQueryBuilder(exp.Equal("inReplyTo", w.stream.StreamID.Hex()))
}

// makeQueryBuilder returns a fully initialized QueryBuilder
func (w Stream) makeQueryBuilder(criteria exp.Expression) QueryBuilder {

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
		return attachment, derp.Wrap(err, "renderer.Stream.Attachments", "Error listing attachments")
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
		return result, derp.Wrap(err, "renderer.Stream.Attachments", "Error listing attachments")
	}

	attachment := new(model.Attachment)
	for iterator.Next(attachment) {
		result = append(result, *attachment)
		attachment = new(model.Attachment)
	}

	return result, nil
}

/*******************************************
 * SUBSCRIPTIONS
 *******************************************/

func (w Stream) Subscriptions() ([]model.Subscription, error) {

	result := []model.Subscription{}
	subscriptionService := w.factory().Subscription()

	iterator, err := subscriptionService.ListByUserID(w.UserID())

	if err != nil {
		return result, derp.Wrap(err, "renderer.Stream.Subscriptions", "Error listing subscriptions")
	}

	subscription := model.NewSubscription()

	for iterator.Next(&subscription) {
		result = append(result, subscription)
		subscription = model.NewSubscription()
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

	authorization := w.authorization()

	return action.UserCan(w.stream, &authorization)
}

// CanCreate returns all of the templates that can be created underneath
// the current stream.
func (w Stream) CanCreate() []form.LookupCode {

	templateService := w.factory().Template()
	return templateService.ListByContainer(w.template.TemplateID)
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
		template:     w.template,
		Common:       NewCommon(w.factory(), w.ctx, w.action, w.actionID),
	}, nil
}

/*******************************************
 * MISC HELPER FUNCTIONS
 *******************************************/

func (w Stream) setAuthor() error {

	user, err := w.getUser()

	if err != nil {
		return derp.Wrap(err, "render.Stream.setAuthor", "Error loading User")
	}

	w.stream.AuthorID = user.UserID
	w.stream.AuthorName = user.DisplayName
	w.stream.AuthorImage = user.ImageURL

	return nil
}
