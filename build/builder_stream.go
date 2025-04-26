package build

import (
	"bytes"
	"html/template"
	"math"
	"net/http"
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	builder "github.com/benpate/exp-builder"
	"github.com/benpate/form"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/channel"
	"github.com/benpate/rosetta/convert"
	htmlconv "github.com/benpate/rosetta/html"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/schema"
	"github.com/benpate/rosetta/slice"
	"github.com/benpate/rosetta/sliceof"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Stream wraps a model.Stream object and provides functions that make it easy to build an HTML template with it.
type Stream struct {
	_service service.ModelService // Service to use to access streams (could be Stream or StreamDraft)
	_stream  *model.Stream        // The Stream to be displayed
	CommonWithTemplate
}

/******************************************
 * Stream Builder Constructors
 ******************************************/

// NewStream creates a new object that can generate HTML for a specific stream/view
func NewStream(factory Factory, request *http.Request, response http.ResponseWriter, template model.Template, stream *model.Stream, actionID string) (Stream, error) {

	const location = "build.NewStream"

	// Create the underlying Common builder
	common, err := NewCommonWithTemplate(factory, request, response, template, actionID)

	if err != nil {
		return Stream{}, derp.ReportAndReturn(derp.Wrap(err, location, "Error creating common builder"))
	}

	if !common._action.UserCan(stream, &common._authorization) {
		if common._authorization.IsAuthenticated() {
			return Stream{}, derp.ReportAndReturn(derp.NewForbiddenError(location, "Forbidden"))
		} else {
			return Stream{}, derp.ReportAndReturn(derp.NewUnauthorizedError(location, "Anonymous user is not authorized to perform this action", stream.URL, actionID))
		}
	}

	// Success.  Populate Stream
	return Stream{
		_service:           factory.Stream(),
		_stream:            stream,
		CommonWithTemplate: common,
	}, nil
}

// NewStreamWithoutTemplate creates a new object that can generate HTML for a specific stream/view
func NewStreamWithoutTemplate(factory Factory, request *http.Request, response http.ResponseWriter, stream *model.Stream, actionID string) (Stream, error) {

	// Use the template service to look up the correct template
	templateService := factory.Template()
	template, err := templateService.Load(stream.TemplateID)

	if err != nil {
		return Stream{}, derp.Wrap(err, "build.NewStreamWithoutTemplate", "Error loading Template", stream)
	}

	// Return a fully populated service
	return NewStream(factory, request, response, template, stream, actionID)
}

// NewStreamFromURI creates a new Stream builder for the provided request context.
// IMPORTANT: The stream parameter is expected to be an empty stream in the caller's scope that will be populated by this function.
func NewStreamFromURI(serverFactory ServerFactory, request *http.Request, response http.ResponseWriter, stream *model.Stream, actionID string) (Stream, error) {

	const location = "build.NewStreamFromURI"

	// Locate the requested domain name
	factory, err := serverFactory.ByDomainName(request.Host)

	if err != nil {
		return Stream{}, derp.Wrap(err, location, "Invalid domain")
	}

	// If Load the stream (using a stream in the caller's namespace)
	streamService := factory.Stream()
	token, defaultAction, err := streamService.ParsePath(request.URL)

	if err != nil {
		return Stream{}, derp.Wrap(err, location, "Invalid path")
	}

	// Try to load the Stream from the database
	if err := streamService.LoadByToken(token, stream); err != nil {
		return Stream{}, derp.Wrap(err, location, "Error loading stream")
	}

	// If the calling function didn't specify an action, then use the default action from the URL
	if actionID == "" {
		actionID = defaultAction
	}

	// Create and return a new builder
	return NewStreamWithoutTemplate(factory, request, response, stream, actionID)
}

/******************************************
 * Renderer Interface
 ******************************************/

// Render generates the string value for this Stream
func (w Stream) Render() (template.HTML, error) {

	var buffer bytes.Buffer

	// Execute step (write HTML to buffer, update context)
	status := Pipeline(w._action.Steps).Get(w._factory, w, &buffer)

	if status.Error != nil {
		err := derp.Wrap(status.Error, "build.Stream.Render", "Error generating HTML")
		derp.Report(err)
		return "", err
	}

	// Success!
	status.Apply(w._response)
	return template.HTML(buffer.String()), nil
}

// object returns the model object associated with this builder
func (w Stream) object() data.Object {
	return w._stream
}

func (w Stream) objectID() primitive.ObjectID {
	return w._stream.StreamID
}

func (w Stream) objectType() string {
	return "Stream"
}

// schema returns the validation schema associated with this builder
func (w Stream) schema() schema.Schema {
	return w._template.Schema
}

func (w Stream) service() service.ModelService {
	return w._service
}

// templateRole returns the role that this Stream's Template plays in the system.
// This is used to determine what kinds of Streams can be added underneath this one as children.
func (w Stream) templateRole() string {
	return w._template.TemplateRole
}

func (w Stream) clone(action string) (Builder, error) {
	return NewStream(w._factory, w._request, w._response, w._template, w._stream, action)
}

/******************************************
 * ACTION SHORTCUTS
 ******************************************/

// View executes a separate view for this Stream
func (w Stream) View(actionID string) (template.HTML, error) {

	const location = "build.Stream.View"

	// Create a new builder (this will also validate the user's permissions)
	subStream, err := NewStream(w.factory(), w._request, w._response, w._template, w._stream, actionID)

	if err != nil {
		return template.HTML(""), derp.ReportAndReturn(derp.Wrap(err, location, "Error creating sub-builder"))
	}

	// Generate HTML template
	return subStream.Render()
}

/******************************************
 * STREAM DATA
 ******************************************/

// StreamID returns the unique ID for the stream being built
func (w Stream) StreamID() string {
	return w._stream.StreamID.Hex()
}

// StreamID returns the unique ID for the stream being built
func (w Stream) ParentID() string {
	return w._stream.ParentID.Hex()
}

// NavigationID returns the unique ID of the top-level stream in this stream's hierarchy
func (w Stream) NavigationID() string {
	if w._stream.NavigationID == "" {
		return w._stream.StreamID.Hex()
	}
	return w._stream.NavigationID
}

// PageTitle returns the Label for the stream being built
func (w Stream) PageTitle() string {
	return w._stream.Label
}

// StateID returns the current state of the stream being built
func (w Stream) StateID() string {
	return w._stream.StateID
}

// TemplateID returns the name of the template being used
func (w Stream) TemplateID() string {
	return w._stream.TemplateID
}

// Token returns the unique URL token for the stream being built
func (w Stream) Token() string {
	return w._stream.Token
}

// Document returns the DocumentLink record for this stream
func (w Stream) Document() model.DocumentLink {
	return w._stream.DocumentLink()
}

// Label returns the Label for the stream being built
func (w Stream) Label() string {
	return w._stream.Label
}

// Summary returns the description of the stream being built
func (w Stream) Summary() string {
	return w._stream.Summary
}

// SummaryHTML returns the description of the stream being built
func (w Stream) SummaryHTML() template.HTML {
	return template.HTML(w._stream.Summary)
}

// SummarySummary returns a plaintext summary (<200 characters) of the stream's description
func (w Stream) ShortSummary() string {
	return htmlconv.Summary(w._stream.Summary)
}

func (w Stream) SummaryOrContent() string {
	return w._stream.SummaryOrContent()
}

// IconURL returns the thumbnail image URL of the stream being built
func (w Stream) IconURL() string {
	return w._stream.IconURL
}

// Tags returns all tags (mentions, hashtags, etc) for the stream being built
func (w Stream) Tags() sliceof.Object[mapof.String] {
	return slice.Map(w._stream.Hashtags, func(tag string) mapof.String {
		return mapof.String{
			"Name": tag,
			"Type": vocab.LinkTypeHashtag,
			"Href": w.Host() + "/search?q=%23" + tag,
		}
	})
}

// Permalink returns a complete URL for this stream
func (w Stream) Permalink() string {
	return w._stream.Permalink()
}

func (w Stream) BasePath() string {
	return "/" + w._stream.StreamID.Hex()
}

// AttributedTo returns ALL AttributedTo records for this stream
func (w Stream) AttributedTo() model.PersonLink {
	return w._stream.AttributedTo
}

// Author returns the "first" AttributedTo record for this stream
func (w Stream) Author() model.PersonLink {
	return w._stream.AttributedTo
}

func (w Stream) IsAuthor() bool {
	return w._stream.AttributedTo.UserID == w.AuthenticatedID()
}

// IsReply returns TRUE if this stream is marked as a reply to another stream or resource
func (w Stream) IsReply() bool {
	return (w._stream.InReplyTo != "")
}

// InReplyTo returns an ActivityStream reference to the URL that this stream replies to
func (w Stream) InReplyTo() streams.Document {
	return w.ActivityStream(w._stream.InReplyTo)
}

// Returns the body content as an HTML template
func (w Stream) ContentHTML() template.HTML {
	return template.HTML(w._stream.Content.HTML)
}

func (w Stream) Places() sliceof.Object[model.Place] {
	return w._stream.Places
}

func (w Stream) StartDate() time.Time {
	return w._stream.StartDate.Time
}

func (w Stream) EndDate() time.Time {
	return w._stream.EndDate.Time
}

func (w Stream) ContentRaw() string {

	if w._stream.Content.Raw == "" {
		return "{}"
	}

	return w._stream.Content.Raw
}

// CreateDate returns the CreateDate of the stream being built
func (w Stream) CreateDate() int64 {
	return w._stream.CreateDate
}

// PublishDate returns the PublishDate of the stream being built
func (w Stream) PublishDate() int64 {

	if w._stream.PublishDate > 0 {
		return w._stream.PublishDate
	}

	return w._stream.CreateDate
}

// UpdateDate returns the UpdateDate of the stream being built
func (w Stream) UpdateDate() int64 {
	return w._stream.UpdateDate
}

// IsPublished returns TRUE if the stream being built has been published
func (w Stream) IsPublished() bool {
	return w._stream.IsPublished()
}

// Rank returns the Rank of the stream being built
func (w Stream) Rank() int {
	return w._stream.Rank
}

// Data returns the custom data field as an "any" type
func (w Stream) Data(value string) any {
	return w._stream.Data[value]
}

// DataString returns a custom data field converted to a string
func (w Stream) DataString(value string) string {
	return w._stream.Data.GetString(value)
}

// DataInt returns a custom data field converted to an integer
func (w Stream) DataInt(value string) int {
	return w._stream.Data.GetInt(value)
}

// DataBool returns a custom data field converted to a bool
func (w Stream) DataBool(value string) bool {
	return w._stream.Data.GetBool(value)
}

// ETag returns the ETag for the stream being built
func (w Stream) ETag() string {
	return w._stream.ETag()
}

// OEmbedJSON returns the URL for the oEmbed JSON endpoint for this stream
func (w Stream) OEmbedJSON() string {
	return w.Host() + "/.oembed?url=" + w.Permalink() + "&format=json"
}

// OEmbedXML returns the URL for the oEmbed XML endpoint for this stream
func (w Stream) OEmbedXML() string {
	return w.Host() + "/.oembed?url=" + w.Permalink() + "&format=xml"
}

// HasGrandparent returns TRUE if the stream
func (w Stream) HasGrandparent() bool {
	return w._stream.HasGrandparent()
}

// HasParent returns TRUE if the stream being built has a parent object
func (w Stream) HasParent() bool {
	return w._stream.HasParent()
}

// IsNew returns TRUE if this stream has not been saved to the database
func (w Stream) IsNew() bool {
	return w._stream.IsNew()
}

// IsEmpty returns TRUE if the stream is an empty placeholder.
func (w Stream) IsEmpty() bool {
	return (w._stream == nil) || (w._stream.StreamID == primitive.NilObjectID)
}

func (w Stream) IsCurrentStream() bool {
	return w._stream.Token == w.PathList().First()
}

func (w Stream) Roles() []string {
	authorization := w.authorization()
	return w._stream.Roles(&authorization)
}

/******************************************
 * Widgets
 ******************************************/

// ListAllWidgets returns a list of all the widgets available on this server
func (w Stream) ListAllWidgets() []form.LookupCode {
	widgetService := w._factory.Widget()
	return widgetService.List()
}

// ListWidgetsByLocation returns a list of all the widgets in the specified location
func (w Stream) ListWidgetsByLocation(location string) []model.StreamWidget {

	result := w._stream.WidgetsByLocation(location)

	if len(result) == 0 {
		return result
	}

	widgetService := w._factory.Widget()
	for index := range result {
		widget, _ := widgetService.Get(result[index].Type)
		result[index].Stream = w._stream
		result[index].Widget = widget
	}

	return result
}

// Widgets returns HTML for all the widgets in the specified location
func (w Stream) Widgets(location string) (template.HTML, error) {

	list := w.ListWidgetsByLocation(location)

	if len(list) == 0 {
		return template.HTML(""), nil
	}

	widgetService := w._factory.Widget()
	var buffer bytes.Buffer
	buffer.WriteString(`<div id="widget-` + location + `" class="widgets ` + location + `">`)
	for _, streamWidget := range list {
		if widget, ok := widgetService.Get(streamWidget.Type); ok {
			widgetBuilder := NewWidget(&w, streamWidget)

			if err := widget.HTMLTemplate.ExecuteTemplate(&buffer, "widget", widgetBuilder); err != nil {
				derp.Report(derp.Wrap(err, "build.Stream.Widgets", "Error executing widget template", widget))
			}
		}
	}
	buffer.WriteString(`</div>`)

	return template.HTML(buffer.String()), nil
}

/******************************************
 * Related Streams
 ******************************************/

// Grandparent returns a Stream renderer containing the parent of the parent of the current stream
func (w Stream) Grandparent(actionID string) (Stream, error) {

	const location = "build.Stream.Grandparent"

	parent, err := w.Parent(actionID)

	if err != nil {
		return Stream{}, derp.Wrap(err, location, "Error loading Parent")
	}

	return parent.Parent(actionID)
}

// ParentOutbox returns an Outbox builder containing the parent of the current stream
func (w Stream) ParentOutbox(actionID string) (Outbox, error) {

	const location = "build.Stream.Parent"

	var user model.User

	userService := w.factory().User()

	if err := userService.LoadByID(w._stream.ParentID, &user); err != nil {
		return Outbox{}, derp.Wrap(err, location, "Error loading Parent")
	}

	builder, err := NewOutbox(w.factory(), w._request, w._response, &user, actionID)

	if err != nil {
		return Outbox{}, derp.Wrap(err, location, "Unable to create new Stream")
	}

	return builder, nil
}

// Parent returns a Stream renderer containing the parent of the current stream
func (w Stream) Parent(actionID string) (Stream, error) {

	const location = "build.Stream.Parent"

	var parent model.Stream

	streamService := w.factory().Stream()

	if err := streamService.LoadParent(w._stream, &parent); err != nil {
		return Stream{}, derp.Wrap(err, location, "Error loading Parent")
	}

	builder, err := NewStreamWithoutTemplate(w.factory(), w._request, w._response, &parent, actionID)

	if err != nil {
		return Stream{}, derp.Wrap(err, location, "Unable to create new Stream")
	}

	return builder, nil
}

// PrevSibling returns the sibling Stream that immediately preceeds this one, based on the provided sort field
func (w Stream) PrevSibling(sortField string, action string) (Stream, error) {

	criteria := exp.Equal("parentId", w._stream.ParentID).
		AndLessThan(sortField, w._stream.GetSort(sortField))

	sortOption := option.SortDesc(sortField)

	return w.getFirstStream(criteria, sortOption, action), nil
}

// NextSibling returns the sibling Stream that immediately follows this one, based on the provided sort field
func (w Stream) NextSibling(sortField string, action string) (Stream, error) {

	criteria := exp.Equal("parentId", w._stream.ParentID).
		AndGreaterThan(sortField, w._stream.GetSort(sortField))

	sortOption := option.SortAsc(sortField)

	return w.getFirstStream(criteria, sortOption, action), nil
}

// FirstChild returns the first child Stream underneath this one, based on the provided sort field
func (w Stream) FirstChild(sort string, action string) (Stream, error) {

	criteria := exp.Equal("parentId", w._stream.StreamID)

	sortOption := option.SortAsc(sort)

	return w.getFirstStream(criteria, sortOption, action), nil
}

// FirstChild returns the first child Stream underneath this one, based on the provided sort field
func (w Stream) LastChild(sort string, action string) (Stream, error) {

	criteria := exp.Equal("parentId", w._stream.StreamID)
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
		derp.Report(derp.Wrap(err, "build.Stream.NextSibling", "Database error"))
		return Stream{}
	}

	var first model.Stream

	if iterator.Next(&first) {
		if result, err := NewStreamWithoutTemplate(w.factory(), w._request, w._response, &first, actionID); err == nil {
			return result
		}
	}

	// Fall through means no streams are valid.  Return an empty builder instead.
	return Stream{}
}

// Mentions returns a slice of all Mentions for this Stream
func (w Stream) Mentions() ([]model.Mention, error) {
	mentionService := w.factory().Mention()
	return mentionService.QueryByObjectID(w._stream.StreamID)
}

// RepliesBefore returns a slice of all ActivityStreams before the specified date
func (w Stream) RepliesBefore(dateString string, maxRows int) sliceof.Object[streams.Document] {

	done := make(channel.Done)

	// Get all ActivityStreams that reply to the current Stream
	activityStreamsService := w._factory.ActivityStream()
	maxDate := convert.Int64Default(dateString, math.MaxInt64)
	replies := activityStreamsService.QueryRepliesBeforeDate(w._stream.URL, maxDate, done)

	// Filter results based on blocks
	ruleService := w._factory.Rule()
	ruleFilter := ruleService.Filter(w.AuthenticatedID())
	filteredResult := ruleFilter.Channel(replies)

	// Limit to `maxRows` records
	limitedFilter := channel.Limit(maxRows, filteredResult, done)
	result := channel.Slice(limitedFilter)

	return slice.Reverse(result)
}

func (w Stream) RepliesAfter(dateString string, maxRows int) sliceof.Object[streams.Document] {

	done := make(channel.Done)

	// Get all ActivityStreams that REPLY TO the current Stream
	activityStreamsService := w._factory.ActivityStream()
	minDate := convert.Int64(dateString)
	replies := activityStreamsService.QueryRepliesAfterDate(w._stream.URL, minDate, done)

	// Filter results based on blocks
	ruleService := w._factory.Rule()
	ruleFilter := ruleService.Filter(w.AuthenticatedID())
	filteredResult := ruleFilter.Channel(replies)

	// Limit to `maxRows` records
	limitedFilter := channel.Limit(maxRows, filteredResult, done)
	result := channel.Slice(limitedFilter)

	// Success
	return result
}

func (w Stream) AnnouncesBefore(dateString string, maxRows int) sliceof.Object[streams.Document] {

	done := make(channel.Done)

	// Get all ActivityStreams that ANNOUNCE the current Stream
	activityStreamsService := w._factory.ActivityStream()
	maxDate := convert.Int64Default(dateString, math.MaxInt64)
	announces := activityStreamsService.QueryAnnouncesBeforeDate(w._stream.URL, maxDate, done)

	// Filter results based on blocks
	ruleService := w._factory.Rule()
	ruleFilter := ruleService.Filter(w.AuthenticatedID())
	filteredResult := ruleFilter.Channel(announces)

	// Limit to `maxRows` records
	limitedFilter := channel.Limit(maxRows, filteredResult, done)
	result := channel.Slice(limitedFilter)

	// Triumph
	return result
}

func (w Stream) LikesBefore(dateString string, maxRows int) sliceof.Object[streams.Document] {

	done := make(channel.Done)

	// Query all ActivityStreams that LIKE the current Stream
	activityStreamsService := w._factory.ActivityStream()
	maxDate := convert.Int64Default(dateString, math.MaxInt64)
	likes := activityStreamsService.QueryLikesBeforeDate(w._stream.URL, maxDate, done)

	// Filter results based on blocks
	ruleService := w._factory.Rule()
	ruleFilter := ruleService.Filter(w.AuthenticatedID())
	filteredResult := ruleFilter.Channel(likes)

	// Limit to `maxRows` records
	limitedFilter := channel.Limit(maxRows, filteredResult, done)
	result := channel.Slice(limitedFilter)

	// Celebrate
	return result
}

// Outbox returns a QueryBuilder for the current Stream's outbox
func (w Stream) Outbox() (QueryBuilder[model.OutboxMessage], error) {

	queryString := w._request.URL.Query()

	expBuilder := builder.NewBuilder().
		Int("rank").
		Int("createDate")

	criteria := exp.And(
		exp.Equal("parentType", model.FollowerTypeStream),
		exp.Equal("parentId", w._stream.StreamID),
		exp.Equal("deleteDate", 0),
		expBuilder.Evaluate(queryString),
	)

	return NewQueryBuilder[model.OutboxMessage](w._factory.Outbox(), criteria), nil
}

/******************************************
 * Other Records
 ******************************************/

func (w Stream) Streams() QueryBuilder[model.StreamSummary] {
	streamService := w.factory().Stream()
	return NewQueryBuilder[model.StreamSummary](streamService, exp.All())
}

func (w Stream) Users() QueryBuilder[model.UserSummary] {
	userService := w.factory().User()
	criteria := exp.Equal("isPublic", true)
	return NewQueryBuilder[model.UserSummary](userService, criteria)
}

/******************************************
 * Related Streams
 ******************************************/

func (w Stream) Breadcrumbs() ([]model.StreamSummary, error) {
	streamService := w.factory().Stream()

	return streamService.QuerySummary(
		exp.In("_id", w._stream.ParentIDs),
		option.SortAsc("depth"),
	)
}

// Ancestors returns all Streams that have the same "parent" as the current Stream's parent
func (w Stream) Ancestors() QueryBuilder[model.StreamSummary] {
	var parent model.Stream

	streamService := w.factory().Stream()

	if err := streamService.LoadParent(w._stream, &parent); err != nil {
		derp.Report(derp.Wrap(err, "build.Stream.Ancestors", "Error loading parent"))
	}

	return w.makeStreamQueryBuilder(exp.Equal("parentId", parent.ParentID))
}

// Siblings returns all Streams that have the same "parent" as the current Stream
func (w Stream) Siblings() QueryBuilder[model.StreamSummary] {
	return w.makeStreamQueryBuilder(exp.Equal("parentId", w._stream.ParentID))
}

// Children returns all Streams with a "parent" is the current Stream
func (w Stream) Children() QueryBuilder[model.StreamSummary] {
	return w.makeStreamQueryBuilder(exp.Equal("parentId", w._stream.StreamID))
}

// makeStreamQueryBuilder returns a fully initialized RenderBuilder
func (w Stream) makeStreamQueryBuilder(criteria exp.Expression) QueryBuilder[model.StreamSummary] {

	queryBuilder := builder.NewBuilder().
		Int("createDate").
		Int("updateDate").
		Int("publishDate").
		Int("expirationDate").
		Int("rank").
		String("label")

	queryValues := w._request.URL.Query()
	query := queryBuilder.Evaluate(queryValues)

	criteria = w.withViewPermission(
		criteria.And(query),
	)

	result := NewQueryBuilder[model.StreamSummary](w._factory.Stream(), criteria)
	result.By(w._template.ChildSortType)
	if w._template.ChildSortDirection == "desc" {
		result.Reverse()
	}

	return result
}

/******************************************
 * Attachments
 ******************************************/

// Reference to the first file attached to this stream
func (w Stream) Attachment() (model.Attachment, error) {
	return w.factory().Attachment().LoadFirstByObjectID(model.AttachmentObjectTypeStream, w._stream.StreamID)
}

// Attachments lists all attachments for this stream.
func (w Stream) Attachments() (sliceof.Object[model.Attachment], error) {
	return w.factory().Attachment().QueryByObjectID(model.AttachmentObjectTypeStream, w._stream.StreamID)
}

// AttachmentByType lists all attachments for this stream.
func (w Stream) AttachmentsByCategory(category string) (sliceof.Object[model.Attachment], error) {
	return w.factory().Attachment().QueryByCategory(model.AttachmentObjectTypeStream, w._stream.StreamID, category)
}

/******************************************
 * Content Actors
 ******************************************/

func (w Stream) Followers() ([]model.Follower, error) {
	followerService := w.factory().Follower()
	return followerService.QueryByParent(model.FollowerTypeStream, w._stream.StreamID)
}

/******************************************
 * ACCESS PERMISSIONS
 ******************************************/

// UserCan returns TRUE if this Request is authorized to access the requested view
func (w Stream) UserCan(actionID string) bool {

	factory := w._factory
	templateService := factory.Template()
	template, err := templateService.Load(w._stream.TemplateID)

	if err != nil {
		return false
	}

	// Try to find the requested Action in the Template
	action, ok := template.Action(actionID)

	if !ok {
		return false
	}

	// Use the action.UserCan method to determine if the user can perform this action
	authorization := w.authorization()
	return action.UserCan(w._stream, &authorization)
}

// CanCreate returns all of the templates that can be created underneath
// the current stream.
func (w Stream) CanCreate() []form.LookupCode {

	templateService := w.factory().Template()
	return templateService.ListByContainer(w._template.TemplateID)
}

// HasSubscriptions returns TRUE if this stream has any subscriptions
func (w Stream) HasSubscriptions() bool {
	return w._stream.HasSubscriptions()
}

// SubscriptionIDs returns all subscription IDs that are valid for this stream
func (w Stream) SubscriptionIDs() []string {
	return w._stream.SubscriptionIDs()
}

/******************************************
 * Helper Functions
 ******************************************/

// draftBuilder returns a new build.Stream that is bound to the
// draft service, and a draft copy of the current stream.
func (w Stream) draftBuilder() (Stream, error) {

	var draft model.Stream
	draftService := w.factory().StreamDraft()

	// Load the draft of the object
	if err := draftService.LoadByID(w._stream.StreamID, &draft); err != nil {
		return Stream{}, derp.Wrap(err, "build.Stream.draftBuilder", "Error loading draft")
	}

	// Make a duplicate of this builder.  Same object, template, action settings
	return Stream{
		_stream:            &draft,
		_service:           draftService,
		CommonWithTemplate: w.CommonWithTemplate,
	}, nil
}

func (w Stream) setState(stateID string) error {
	w._stream.SetState(stateID)
	return nil
}

func (w Stream) debug() {
	log.Debug().Interface("object", w.object()).Msg("builder_Stream")
}
