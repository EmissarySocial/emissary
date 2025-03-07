package service

import (
	"iter"
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/domain"
	"github.com/benpate/exp"
	"github.com/benpate/rosetta/iterator"
	"github.com/benpate/rosetta/schema"
	"github.com/benpate/turbine/queue"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Rule defines a service that manages all content rules created and imported by Users.
type Rule struct {
	collection    data.Collection
	outboxService *Outbox
	userService   *User
	host          string

	queue *queue.Queue
}

// NewRule returns a fully initialized Rule service
func NewRule() Rule {
	return Rule{}
}

/******************************************
 * Lifecycle Methods
 ******************************************/

// Refresh updates any stateful data that is cached inside this service.
func (service *Rule) Refresh(collection data.Collection, outboxService *Outbox, userService *User, queue *queue.Queue, host string) {
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

func (service *Rule) Count(criteria exp.Expression) (int64, error) {
	return service.collection.Count(notDeleted(criteria))
}

// Query returns an slice of allthe Rules that match the provided criteria
func (service *Rule) Query(criteria exp.Expression, options ...option.Option) ([]model.Rule, error) {
	result := make([]model.Rule, 0)
	err := service.collection.Query(&result, notDeleted(criteria), options...)

	return result, err
}

// QuerySummary returns an slice of allthe Rules that match the provided criteria
func (service *Rule) QuerySummary(criteria exp.Expression, options ...option.Option) ([]model.RuleSummary, error) {
	result := make([]model.RuleSummary, 0)
	options = append(options, option.Fields(model.RuleSummaryFields()...))
	err := service.collection.Query(&result, notDeleted(criteria), options...)

	return result, err
}

// List returns an iterator containing all of the Rules that match the provided criteria
func (service *Rule) List(criteria exp.Expression, options ...option.Option) (data.Iterator, error) {
	return service.collection.Iterator(notDeleted(criteria), options...)
}

// Range returns a Go 1.23 RangeFunc that iterates over the Rule records that match the provided criteria
func (service *Rule) Range(criteria exp.Expression, options ...option.Option) (iter.Seq[model.Rule], error) {

	iter, err := service.List(criteria, options...)

	if err != nil {
		return nil, derp.Wrap(err, "service.Rule.Range", "Error creating iterator", criteria)
	}

	return RangeFunc(iter, model.NewRule), nil
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

	// Validate the value before saving
	if err := service.Schema().Validate(rule); err != nil {
		return derp.Wrap(err, "service.Rule.Save", "Error validating Rule", rule)
	}

	// If this is a duplicate rule, then halt
	if service.hasDuplicate(rule) {
		return nil
	}

	// RULE: Externally imported rules cannot be re-shared automatically.
	if rule.OriginRemote() {
		rule.IsPublic = false
	}

	switch rule.IsPublic {

	case true:

		switch rule.PublishDate {

		// "Publish" Rule when it is first shared publicly
		case 0:

			rule.PublishDate = time.Now().Unix()
			go derp.Report(service.publish(*rule))

		// "Republish" changes when a public Rule is updated
		default:
			go derp.Report(service.republish(*rule))
		}

	case false:

		// RULE: Unpublish Rules when they are no longer shared publicly
		if rule.PublishDate > 0 {

			go derp.Report(service.unpublish(*rule))
			rule.PublishDate = 0
		}
	}

	// Save the rule to the database
	if err := service.collection.Save(rule, note); err != nil {
		return derp.Wrap(err, "service.Rule.Save", "Error saving Rule", rule, note)
	}

	// Recalculate the rule count for this user
	go service.userService.CalcRuleCount(rule.UserID)

	return nil
}

// Delete removes an Rule from the database (virtual delete)
func (service *Rule) Delete(rule *model.Rule, note string) error {

	// Delete this Rule
	if err := service.collection.Delete(rule, note); err != nil {
		return derp.Wrap(err, "service.Rule.Delete", "Error deleting Rule", rule, note)
	}

	if rule.IsPublic {
		go derp.Report(service.unpublish(*rule))
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

// LoadByTrigger retrieves a single Rule that maches the provided User, RuleType, and Trigger
func (service *Rule) LoadByTrigger(userID primitive.ObjectID, ruleType string, trigger string, rule *model.Rule) error {

	criteria := service.byUserID(userID).
		AndEqual("type", ruleType).
		AndEqual("trigger", trigger)

	return service.Load(criteria, rule)
}

// LoadByFollowing retrieves a single Rule that maches the provided User, Following, RuleType, and Trigger
func (service *Rule) LoadByFollowing(userID primitive.ObjectID, followingID primitive.ObjectID, ruleType string, trigger string, rule *model.Rule) error {

	criteria := exp.Equal("userId", userID).
		AndEqual("followingId", followingID).
		AndEqual("type", ruleType).
		AndEqual("trigger", trigger)

	return service.Load(criteria, rule)
}

// QueryPublic returns a collection of Rules that are marked Public, in reverse chronological order.
func (service *Rule) QueryPublic(userID primitive.ObjectID, maxDate int64, options ...option.Option) ([]model.Rule, error) {

	criteria := service.byUserID(userID).
		AndEqual("isPublic", true).
		AndLessThan("publishDate", maxDate)

	options = append(options, option.SortDesc("publishDate"))
	result, err := service.Query(criteria, options...)

	return result, err
}

func (service *Rule) QueryByType(userID primitive.ObjectID, ruleType string, criteria exp.Expression, options ...option.Option) ([]model.Rule, error) {

	criteria = service.byUserID(userID).
		AndEqual("type", ruleType).
		AndEqual("isPublic", true).
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

// QueryByActor retrieves a slice of RuleSummaries that match the provided User and Actor
func (service *Rule) QueryByActor(userID primitive.ObjectID, actorID string) ([]model.RuleSummary, error) {

	criteria := exp.And(
		service.byUserID(userID),
		exp.Or(
			exp.Equal("type", model.RuleTypeActor).AndEqual("trigger", actorID),
			exp.Equal("type", model.RuleTypeDomain).AndEqual("trigger", domain.NameOnly(actorID)),
			exp.Equal("type", model.RuleTypeContent),
		),
	)

	return service.QuerySummary(criteria)
}

// QueryByActorAndActions retrieves a slice of RuleSummaries that match the provided User, Actor, and potential actions
func (service *Rule) QueryByActorAndActions(userID primitive.ObjectID, actorID string, actions ...string) ([]model.RuleSummary, error) {

	criteria := exp.And(
		service.byUserID(userID),
		exp.Or(
			exp.Equal("type", model.RuleTypeActor).AndEqual("trigger", actorID),
			exp.Equal("type", model.RuleTypeDomain).AndEqual("trigger", domain.NameOnly(actorID)),
			exp.Equal("type", model.RuleTypeContent),
		),
		exp.In("action", actions),
	)

	return service.QuerySummary(criteria)
}

// QueryDomainBlocks returns all external domains blocked by this Instance/Domain.
func (service *Rule) QueryDomainBlocks() ([]model.Rule, error) {

	criteria := exp.Equal("userId", primitive.NilObjectID).
		AndEqual("type", model.RuleTypeDomain).
		AndEqual("behavior", model.RuleActionBlock)

	return service.Query(criteria, option.SortAsc("trigger"))
}

// QueryBlockedActors returns all Actors blocked by this User (or by the Domain on behalf of the User)
func (service *Rule) QueryBlockedActors(userID primitive.ObjectID) ([]model.Rule, error) {

	criteria := service.byUserID(userID).
		AndEqual("type", model.RuleTypeActor).
		AndEqual("behavior", model.RuleActionBlock)

	return service.Query(criteria, option.SortAsc("trigger"))
}

// RangeByUserID returns all Rules tha belong to a specific User (NO DOMAIN RULES)
func (service *Rule) RangeByUserID(userID primitive.ObjectID) (iter.Seq[model.Rule], error) {
	return service.Range(exp.Equal("userId", userID))
}

func (service *Rule) DeleteByUserID(userID primitive.ObjectID, comment string) error {

	const location = "service.Rule.DeleteByUserID"

	rangeFunc, err := service.RangeByUserID(userID)

	if err != nil {
		return derp.Wrap(err, location, "Error getting range function")
	}

	for rule := range rangeFunc {
		if err := service.Delete(&rule, comment); err != nil {
			return derp.Wrap(err, location, "Error deleting rule", rule)
		}
	}

	return nil
}

/******************************************
 * Rule Filters
 ******************************************/

func (service *Rule) Filter(userID primitive.ObjectID, options ...RuleFilterOption) RuleFilter {
	return NewRuleFilter(service, userID, options...)
}

/******************************************
 * Misc Helpers
 ******************************************/

// hasDuplicate returns TRUE if the provided Rule is a duplicate of an existing Rule.
// IMPORTANT: This method MAY update the provided Rule
func (service *Rule) hasDuplicate(rule *model.Rule) bool {

	// Search the database for duplicate rules
	criteria := exp.NotEqual("_id", rule.RuleID).
		AndEqual("userId", rule.UserID).
		AndEqual("type", rule.Type).
		AndEqual("trigger", rule.Trigger)

	duplicate := model.NewRule()

	// If a duplicate is not found, then return FALSE
	if err := service.Load(criteria, &duplicate); derp.NotFound(err) {
		return false
	}

	// If the new rule was made manually, but the duplicate was imported from a Following...
	if rule.OriginUser() && duplicate.OriginRemote() {
		// Change the RuleID so that we overwrite the duplicate with new information
		rule.FollowingID = duplicate.FollowingID
		rule.Journal = duplicate.Journal
		return false
	}

	// In all other cases, we should NOT SAVE the new record
	// because it is a duplicate
	return true
}

// byUserID generates a criteria expression that searches for:
// 1) Rules that belong to the provided User
// 2) Rules that belong to no User (i.e. public rules)
func (service *Rule) byUserID(userID primitive.ObjectID) exp.Expression {
	return exp.Equal("userId", userID).Or(exp.Equal("userId", primitive.NilObjectID))
}
