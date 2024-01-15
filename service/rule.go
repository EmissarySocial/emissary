package service

import (
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/hannibal/queue"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/iterator"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Rule defines a service that manages all content rules created and imported by Users.
type Rule struct {
	collection    data.Collection
	outboxService *Outbox
	userService   *User
	host          string

	queue queue.Queue
}

// NewRule returns a fully initialized Rule service
func NewRule() Rule {
	return Rule{}
}

/******************************************
 * Lifecycle Methods
 ******************************************/

// Refresh updates any stateful data that is cached inside this service.
func (service *Rule) Refresh(collection data.Collection, outboxService *Outbox, userService *User, queue queue.Queue, host string) {
	service.collection = collection
	service.outboxService = outboxService
	service.userService = userService
	service.queue = queue
	service.host = host
}

// Close stops any background processes controlled by this service
func (service *Rule) Close() {
	// Nothin to do here.
}

/******************************************
 * Common Data Methods
 ******************************************/

// Query returns an slice of allthe Rules that match the provided criteria
func (service *Rule) Query(criteria exp.Expression, options ...option.Option) ([]model.Rule, error) {
	result := make([]model.Rule, 0)
	err := service.collection.Query(&result, notDeleted(criteria), options...)

	return result, err
}

// List returns an iterator containing all of the Rules that match the provided criteria
func (service *Rule) List(criteria exp.Expression, options ...option.Option) (data.Iterator, error) {
	return service.collection.Iterator(notDeleted(criteria), options...)
}

// Channel returns a channel that will stream all of the Rules that match the provided criteria
func (service *Rule) Channel(criteria exp.Expression, options ...option.Option) (<-chan model.Rule, error) {
	it, err := service.List(criteria, options...)

	if err != nil {
		return nil, derp.Wrap(err, "service.Rule.Channel", "Error creating iterator", criteria, options)
	}

	return iterator.Channel[model.Rule](it, model.NewRule), nil
}

// Load retrieves an Rule from the database
func (service *Rule) Load(criteria exp.Expression, rule *model.Rule) error {

	if err := service.collection.Load(notDeleted(criteria), rule); err != nil {
		return derp.Wrap(err, "service.Rule.Load", "Error loading Rule", criteria)
	}

	return nil
}

// Save adds/updates an Rule in the database
func (service *Rule) Save(rule *model.Rule, note string) error {

	if rule.IsNew() {
		var err error

		switch rule.Type {

		case model.RuleTypeActor:
			err = service.ValidateNewActor(rule)

		case model.RuleTypeDomain:
			err = service.ValidateNewDomain(rule)

		case model.RuleTypeContent:
			err = service.ValidateNewContent(rule)
		}

		if err != nil {
			return derp.Wrap(err, "service.Rule.Save", "Error validating new Rule", rule)
		}
	}

	// Clean the value before saving
	if err := service.Schema().Clean(rule); err != nil {
		return derp.Wrap(err, "service.Rule.Save", "Error cleaning Rule", rule)
	}

	// RULE: Publish changes when the rule is first shared publicly
	if rule.IsActive && rule.IsPublic && (rule.PublishDate == 0) {
		if err := service.publish(rule); err != nil {
			return derp.Wrap(err, "service.Rule.Save", "Error publishing Rule", rule)
		}
	}

	// RULE: Unpublish changes when the rule is no longer shared publicly
	if (!rule.IsPublic || !rule.IsActive) && (rule.PublishDate > 0) {
		if err := service.unpublish(rule, true); err != nil {
			return derp.Wrap(err, "service.Rule.Save", "Error unpublishing Rule", rule)
		}
	}

	// Save the rule to the database
	if err := service.collection.Save(rule, note); err != nil {
		return derp.Wrap(err, "service.Rule.Save", "Error saving Rule", rule, note)
	}

	// Recalculate the rule count for this user
	go service.userService.CalcRuleCount(rule.UserID)

	// TODO: HIGH: Remove matching followers...

	return nil
}

// Delete removes an Rule from the database (virtual delete)
func (service *Rule) Delete(rule *model.Rule, note string) error {

	// Delete this Rule
	if err := service.collection.Delete(rule, note); err != nil {
		return derp.Wrap(err, "service.Rule.Delete", "Error deleting Rule", rule, note)
	}

	if rule.IsPublic {
		if err := service.unpublish(rule, false); err != nil {
			derp.Report(derp.Wrap(err, "service.Rule.Delete", "Error unpublishing Rule", rule))
		}
	}

	return nil
}

/******************************************
 * Model Service Methods
 ******************************************/

// ObjectType returns the type of object that this service manages
func (service *Rule) ObjectType() string {
	return "Rule"
}

// New returns a fully initialized model.Group as a data.Object.
func (service *Rule) ObjectNew() data.Object {
	result := model.NewRule()
	return &result
}

func (service *Rule) ObjectID(object data.Object) primitive.ObjectID {

	if mention, ok := object.(*model.Rule); ok {
		return mention.RuleID
	}

	return primitive.NilObjectID
}

func (service *Rule) ObjectQuery(result any, criteria exp.Expression, options ...option.Option) error {
	return service.collection.Query(result, notDeleted(criteria), options...)
}

func (service *Rule) ObjectList(criteria exp.Expression, options ...option.Option) (data.Iterator, error) {
	return service.List(criteria, options...)
}

func (service *Rule) ObjectLoad(criteria exp.Expression) (data.Object, error) {
	result := model.NewRule()
	err := service.Load(criteria, &result)
	return &result, err
}

func (service *Rule) ObjectSave(object data.Object, comment string) error {
	if rule, ok := object.(*model.Rule); ok {
		return service.Save(rule, comment)
	}
	return derp.NewInternalError("service.Rule.ObjectSave", "Invalid Object Type", object)
}

func (service *Rule) ObjectDelete(object data.Object, comment string) error {
	if rule, ok := object.(*model.Rule); ok {
		return service.Delete(rule, comment)
	}
	return derp.NewInternalError("service.Rule.ObjectDelete", "Invalid Object Type", object)
}

func (service *Rule) ObjectUserCan(object data.Object, authorization model.Authorization, action string) error {
	return derp.NewUnauthorizedError("service.Rule", "Not Authorized")
}

func (service *Rule) Schema() schema.Schema {
	return schema.New(model.RuleSchema())
}

/******************************************
 * Custom Queries
 ******************************************/

func (service *Rule) LoadByID(userID primitive.ObjectID, ruleID primitive.ObjectID, rule *model.Rule) error {

	criteria := exp.Equal("_id", ruleID).
		And(service.byUserID(userID))

	return service.Load(criteria, rule)
}

func (service *Rule) LoadByToken(userID primitive.ObjectID, token string, rule *model.Rule) error {
	ruleID, err := primitive.ObjectIDFromHex(token)

	if err != nil {
		return derp.Wrap(err, "service.Rule.LoadByToken", "Error converting token to ObjectID", token)
	}

	criteria := exp.Equal("_id", ruleID).AndEqual("userId", userID)

	return service.Load(criteria, rule)
}

func (service *Rule) LoadByTrigger(userID primitive.ObjectID, ruleType string, trigger string, rule *model.Rule) error {

	criteria := exp.Equal("userId", userID).
		AndEqual("type", ruleType).
		AndEqual("trigger", trigger)

	return service.Load(criteria, rule)
}

func (service *Rule) CountByType(userID primitive.ObjectID, ruleType string) (int, error) {
	criteria := exp.Equal("userId", userID).
		AndEqual("deleteDate", 0).
		AndEqual("type", ruleType)

	result, err := service.collection.Count(criteria)
	return int(result), err
}

func (service *Rule) QueryActiveByUser(userID primitive.ObjectID, types ...string) ([]model.Rule, error) {

	criteria := service.byUserID(userID).AndEqual("isActive", true)

	if len(types) > 0 {
		criteria = criteria.And(exp.In("type", types))
	}

	return service.Query(criteria)
}

func (service *Rule) QueryPublicRules(userID primitive.ObjectID, maxDate int64, options ...option.Option) ([]model.Rule, error) {

	criteria := service.byUserID(userID).
		AndEqual("isPublic", true).
		AndNotEqual("isActive", true).
		AndLessThan("publishDate", maxDate)

	options = append(options, option.SortDesc("publishDate"))
	result, err := service.Query(criteria, options...)

	return result, err
}

func (service *Rule) QueryByType(userID primitive.ObjectID, ruleType string, criteria exp.Expression, options ...option.Option) ([]model.Rule, error) {

	criteria = service.byUserID(userID).
		AndEqual("type", ruleType).
		AndEqual("isPublic", true).
		AndNotEqual("isActive", true).
		And(criteria)

	options = append(options, option.SortDesc("publishDate"))
	result, err := service.Query(criteria, options...)

	return result, err
}

func (service *Rule) QueryByTypeActor(userID primitive.ObjectID, criteria exp.Expression, options ...option.Option) ([]model.Rule, error) {
	return service.QueryByType(userID, model.RuleTypeActor, criteria, options...)
}

func (service *Rule) QueryByTypeDomain(userID primitive.ObjectID, criteria exp.Expression, options ...option.Option) ([]model.Rule, error) {
	return service.QueryByType(userID, model.RuleTypeDomain, criteria, options...)
}

func (service *Rule) QueryByTypeContent(userID primitive.ObjectID, criteria exp.Expression, options ...option.Option) ([]model.Rule, error) {
	return service.QueryByType(userID, model.RuleTypeContent, criteria, options...)
}

func (service *Rule) QueryGlobalDomainRules(options ...option.Option) ([]model.Rule, error) {

	criteria := exp.Equal("userId", primitive.NilObjectID).
		AndEqual("type", model.RuleTypeDomain).
		AndEqual("isPublic", true).
		AndNotEqual("isActive", true)

	options = append(options, option.SortDesc("publishDate"))

	return service.Query(criteria, options...)
}

/******************************************
 * Initial Validations
 ******************************************/

// ValidateNewActor validates a new rule of a specific Actor
func (service *Rule) ValidateNewActor(rule *model.Rule) error {
	rule.Label = rule.Trigger
	return nil
}

// ValidateNewDomain validates a new rule of a specific Domain
func (service *Rule) ValidateNewDomain(rule *model.Rule) error {
	rule.Label = rule.Trigger
	return nil
}

// ValidateNewContent validates a external rule service
func (service *Rule) ValidateNewContent(rule *model.Rule) error {
	rule.Label = rule.Trigger
	return nil
}

/******************************************
 * Rule Publishing Rules
 ******************************************/

// publish marks the Rule as published, and sends "Create" activities to all ActivityPub followers
func (service *Rule) publish(rule *model.Rule) error {

	// Try to update the rule in the database (directly, without invoking any business rules)
	rule.PublishDate = time.Now().Unix()

	// Generate JSONLD for this rule
	if err := service.calcJSONLD(rule); err != nil {
		return derp.Wrap(err, "service.Rule.Save", "Error setting JSON-LD", rule)
	}

	if err := service.outboxService.Publish(rule.UserID, rule.JSONLD.GetString("id"), rule.JSONLD); err != nil {
		return derp.Wrap(err, "service.Rule.publish", "Error publishing Rule", rule)
	}

	return nil
}

// unpublish marks the Rule as unpublished and sends "Undo" activities to all ActivityPub followers
func (service *Rule) unpublish(rule *model.Rule, saveAfter bool) error {

	// Try to update the rule in the database (directly, without invoking any business rules)
	rule.PublishDate = 0
	rule.JSONLD = mapof.NewAny()

	if err := service.outboxService.UnPublish(rule.UserID, rule.JSONLD.GetString("id"), rule.JSONLD); err != nil {
		return derp.Wrap(err, "service.Rule.publish", "Error publishing Rule", rule)
	}

	return nil
}

func (service *Rule) calcJSONLD(rule *model.Rule) error {

	user := model.NewUser()
	if err := service.userService.LoadByID(rule.UserID, &user); err != nil {
		return derp.Wrap(err, "service.Rule.Save", "Error loading User", rule)
	}

	// Reset JSON-LD for the rule.  We're going to recalculate EVERYTHING.
	rule.JSONLD = mapof.Any{
		"id":        user.ActivityPubBlockedURL() + "/" + rule.RuleID.Hex(),
		"type":      vocab.ActivityTypeBlock,
		"actor":     user.ActivityPubURL(),
		"target":    rule.Trigger,
		"published": rule.PublishDateRCF3339(),
	}

	// Create the summary based on the type of Rule
	switch rule.Type {

	case model.RuleTypeActor:
		rule.JSONLD["summary"] = user.DisplayName + " blocked the person " + rule.Trigger

	case model.RuleTypeDomain:
		rule.JSONLD["summary"] = user.DisplayName + " blocked the domain " + rule.Trigger

	case model.RuleTypeContent:
		rule.JSONLD["summary"] = user.DisplayName + " blocked the keywords " + rule.Trigger

	default:
		// This should never happen
		return derp.NewInternalError("service.Rule.calcJSONLD", "Unrecognized Rule Type", rule)
	}

	// TODO: need additional grammar for extra fields
	// - selectbox field to describe WHY the rule was created
	// - comment field to describe WHY the rule was created
	// - refs to other people who have ALSO ruleed this person/domain/keyword?

	return nil
}

func (service *Rule) byUserID(userID primitive.ObjectID) exp.Expression {
	return exp.Equal("userId", userID).Or(exp.Equal("userId", primitive.NilObjectID))
}
