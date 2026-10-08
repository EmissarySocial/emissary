package tests

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/model/step"
	"github.com/benpate/form"
	"github.com/benpate/rosetta/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// builderContext is the object and schema that a Builder hands to each step: what
// builder.object() and builder.schema() return in production.
type builderContext struct {
	Label        string        // Names the context in golden keys, empty for the template's own builder
	Schema       schema.Schema // builder.schema()
	NewObject    func() any    // Returns a fresh builder.object()
	PropertyForm form.Element  // The Domain builder's fallback form for an empty "edit" step
}

// templateContexts returns the contexts that the Builder for this template provides, as
// handler/admin.go buildAdmin_GetBuilder and each builder's schema() method choose them
func templateContexts(group *loadedGroup, template model.Template) []builderContext {

	switch template.Model {

	// build.Domain: the Domain schema extended by the current Theme, with the Theme's form
	// as the fallback edit form.  A Domain may use any visible Theme, so pin each one.
	case "Domain", "Search", "SSO", "Followers", "Following":
		result := make([]builderContext, 0)

		for _, theme := range group.themes() {
			domainSchema := schema.New(model.DomainSchema())
			domainSchema.Inherit(theme.Schema)

			result = append(result, builderContext{
				Label:        "theme=" + theme.ThemeID,
				Schema:       domainSchema,
				NewObject:    newPointer(model.NewWritableDomain),
				PropertyForm: theme.Form,
			})
		}

		return result

	// build.Syndication: the plain Domain schema
	case "Syndication":
		return []builderContext{{Schema: schema.New(model.DomainSchema()), NewObject: newPointer(model.NewWritableDomain)}}
	}

	// Only build.Stream and build.Navigation use the template's own schema.  Every other
	// builder uses its model's schema, which is what Template.BaseSchema returns
	if isStreamModel(template.Model) {
		return []builderContext{{Schema: template.Schema, NewObject: template.NewObject}}
	}

	return []builderContext{{Schema: schema.New(template.BaseSchema()), NewObject: template.NewObject}}
}

// isStreamModel reports whether a template's model resolves to the Stream default
// in model.templateModelRegistry
func isStreamModel(name string) bool {
	switch name {
	case "User", "Outbox", "Inbox", "Settings", "Conversations", "Notifications",
		"Domain", "Search", "SSO", "Followers", "Following", "Syndication",
		"Group", "Identity", "Rule", "Tag", "Webhook":
		return false
	}
	return true
}

// subContext returns the context that a composite step gives to its sub-steps, or the
// reason it cannot be known without a database
func subContext(parent builderContext, composite step.Step) (builderContext, string) {

	switch composite.(type) {

	// These run their sub-steps against the same builder
	case step.AsModal, step.AsTooltip, step.IfCondition, step.WithDraft, step.Save, step.AddModelObject:
		return parent, ""

	// build.Attachment
	case step.WithAttachment:
		return modelContext("with-attachment", model.AttachmentSchema, func() any {
			value := model.NewAttachment(model.AttachmentObjectTypeStream, primitive.NilObjectID)
			return &value
		}), ""

	// build.Follower
	case step.WithFollower:
		return modelContext("with-follower", model.FollowerSchema, newPointer(model.NewFollower)), ""

	// build.Model, whose schema is the model service's Schema()
	case step.WithAnnotation:
		return modelContext("with-annotation", model.AnnotationSchema, newPointer(model.NewAnnotation)), ""
	case step.WithCircle:
		return modelContext("with-circle", model.CircleSchema, newPointer(model.NewCircle)), ""
	case step.WithFolder:
		return modelContext("with-folder", model.FolderSchema, newPointer(model.NewFolder)), ""
	case step.WithFollowing:
		return modelContext("with-following", model.FollowingSchema, newPointer(model.NewFollowing)), ""
	case step.WithImport:
		return modelContext("with-import", model.ImportSchema, newPointer(model.NewImport)), ""
	case step.WithKeyPackage:
		return modelContext("with-key-package", model.KeyPackageSchema, newPointer(model.NewKeyPackage)), ""
	case step.WithMerchantAccount:
		return modelContext("with-merchant-account", model.MerchantAccountSchema, newPointer(model.NewMerchantAccount)), ""
	case step.WithMessage:
		return modelContext("with-message", model.NewsItemSchema, newPointer(model.NewNewsItem)), ""
	case step.WithNotification:
		return modelContext("with-notification", model.NotificationSchema, newPointer(model.NewNotification)), ""
	case step.WithOAuthToken:
		return modelContext("with-oauth-token", model.OAuthUserTokenSchema, newPointer(model.NewOAuthUserToken)), ""
	case step.WithPrivilege:
		return modelContext("with-privilege", model.PrivilegeSchema, newPointer(model.NewPrivilege)), ""
	case step.WithResponse:
		return modelContext("with-response", model.ResponseSchema, newPointer(model.NewResponse)), ""
	case step.WithRule:
		return modelContext("with-rule", model.RuleSchema, newPointer(model.NewRule)), ""
	case step.WithStreamSource:
		return modelContext("with-stream-source", model.StreamSourceSchema, newPointer(model.NewStreamSource)), ""
	case step.WithUserConnection:
		return modelContext("with-user-connection", model.UserConnectionSchema, newPointer(model.NewUserConnection)), ""

	// Each of these switches to another Stream, whose template is only known at runtime
	case step.WithChildren, step.WithParent, step.WithNextSibling, step.WithPrevSibling:
		return builderContext{}, "the sub-builder's template belongs to a related Stream, known only at runtime"
	}

	return builderContext{}, "no context mapping for this composite step"
}

// widgetContext returns the context build.Widget gives a widget's saveSteps: the StreamWidget,
// with the widget's own schema nested beneath "data"
func widgetContext(widget model.Widget) builderContext {

	element := widget.Schema.Element
	if element == nil {
		element = schema.Object{Wildcard: schema.Any{}}
	}

	return builderContext{
		Schema: schema.New(schema.Object{Properties: schema.ElementMap{"data": element}}),
		NewObject: func() any {
			streamWidget := model.NewStreamWidget(widget.WidgetID, "Label", "TOP")
			streamWidget.Widget = widget
			return &streamWidget
		},
	}
}

// modelContext returns a context that edits one model object through its own schema
func modelContext(label string, schemaFn func() schema.Element, newObject func() any) builderContext {
	return builderContext{
		Label:     label,
		Schema:    schema.New(schemaFn()),
		NewObject: newObject,
	}
}

// newPointer adapts a value constructor into a function that returns a pointer to a fresh value
func newPointer[T any](constructor func() T) func() any {
	return func() any {
		value := constructor()
		return &value
	}
}
