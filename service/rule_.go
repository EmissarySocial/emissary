package service

import (
	"context"
	"iter"
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	dt "github.com/benpate/domain"
	"github.com/benpate/exp"
	"github.com/benpate/rosetta/schema"
	"github.com/benpate/rosetta/sliceof"
	"github.com/benpate/turbine/queue"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Rule defines a service that manages all content rules created and imported by Users.
type Rule struct {
	importItemService *ImportItem
	outboxService     *Outbox
	userService       *User
	host              string
	newSession        func(timeout time.Duration) (data.Session, context.CancelFunc, error)

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
func (service *Rule) Refresh(factory *Factory) {
	service.importItemService = factory.ImportItem()
	service.outboxService = factory.Outbox()
	service.userService = factory.User()
	service.queue = factory.Queue()
	service.host = factory.Host()
	service.newSession = factory.Session
}

// Close stops any background processes controlled by this service
func (service *Rule) Close() {
	// Nothin to do here.
}

/******************************************
 * Common Data Methods
 ******************************************/

func (service *Rule) collection(session data.Session) data.Collection {
	return session.Collection("Rule")
}

func (service *Rule) Count(session data.Session, criteria exp.Expression) (int64, error) {
	return service.collection(session).Count(notDeleted(criteria))
}

// Query returns an slice of allthe Rules that match the provided criteria
func (service *Rule) Query(session data.Session, criteria exp.Expression, options ...option.Option) ([]model.Rule, error) {
	result := make([]model.Rule, 0)
	err := service.collection(session).Query(&result, notDeleted(criteria), options...)

	return result, err
}

// QuerySummary returns an slice of allthe Rules that match the provided criteria
func (service *Rule) QuerySummary(session data.Session, criteria exp.Expression, options ...option.Option) ([]model.RuleSummary, error) {
	result := make([]model.RuleSummary, 0)
	options = append(options, option.Fields(model.RuleSummaryFields()...))
	err := service.collection(session).Query(&result, notDeleted(criteria), options...)

	return result, err
}

// List returns an iterator containing all of the Rules that match the provided criteria
func (service *Rule) List(session data.Session, criteria exp.Expression, options ...option.Option) (data.Iterator, error) {
	return service.collection(session).Iterator(notDeleted(criteria), options...)
}

// Range returns a Go 1.23 RangeFunc that iterates over the Rule records that match the provided criteria
func (service *Rule) Range(session data.Session, criteria exp.Expression, options ...option.Option) (iter.Seq[model.Rule], error) {

	iter, err := service.List(session, criteria, options...)

	if err != nil {
		return nil, derp.Wrap(err, "service.Rule.Range", "Unable to create iterator", criteria)
	}

	return RangeFunc(iter, model.NewRule), nil
}

// Load retrieves an Rule from the database
func (service *Rule) Load(session data.Session, criteria exp.Expression, rule *model.Rule) error {

	if err := service.collection(session).Load(notDeleted(criteria), rule); err != nil {
		return derp.Wrap(err, "service.Rule.Load", "Unable to load Rule", criteria)
	}

	return nil
}

// Save adds/updates an Rule in the database
func (service *Rule) Save(session data.Session, rule *model.Rule, note string) error {

	const location = "service.Rule.Save"

	// Validate the value before saving
	if err := service.Schema().Validate(rule); err != nil {
		return derp.Wrap(err, location, "Unable to validate Rule", rule)
	}

	// If this is a duplicate rule, then halt
	if service.hasDuplicate(session, rule) {
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
			go derp.Report(service.publish(session, *rule))

		// "Republish" changes when a public Rule is updated
		default:
			go derp.Report(service.republish(session, *rule))
		}

	case false:

		// RULE: Unpublish Rules when they are no longer shared publicly
		if rule.PublishDate > 0 {

			go derp.Report(service.unpublish(session, *rule))
			rule.PublishDate = 0
		}
	}

	// Save the rule to the database
	if err := service.collection(session).Save(rule, note); err != nil {
		return derp.Wrap(err, location, "Unable to save Rule", rule, note)
	}

	// Recalculate the rule count for this user
	if err := service.userService.CalcRuleCount(session, rule.UserID); err != nil {
		return derp.Wrap(err, location, "Unable to calculate rule count")
	}

	return nil
}

// Delete removes an Rule from the database (virtual delete)
func (service *Rule) Delete(session data.Session, rule *model.Rule, note string) error {

	// Delete this Rule
	if err := service.collection(session).Delete(rule, note); err != nil {
		return derp.Wrap(err, "service.Rule.Delete", "Unable to delete Rule", rule, note)
	}

	if rule.IsPublic {
		go derp.Report(service.unpublish(session, *rule))
	}

	return nil
}

/******************************************
 * Special Case Methods
 ******************************************/

// QueryIDOnly returns a slice of IDOnly records that match the provided criteria
func (service *Rule) QueryIDOnly(session data.Session, criteria exp.Expression, options ...option.Option) (sliceof.Object[model.IDOnly], error) {
	result := make([]model.IDOnly, 0)
	options = append(options, option.Fields("_id"))
	err := service.collection(session).Query(&result, notDeleted(criteria), options...)
	return result, err
}

// HardDeleteByID removes a specific Rule record, without applying any additional business rules
func (service *Rule) HardDeleteByID(session data.Session, userID primitive.ObjectID, ruleID primitive.ObjectID) error {

	const location = "service.Rule.HardDeleteByID"

	criteria := exp.Equal("userId", userID).AndEqual("_id", ruleID)

	if err := service.collection(session).HardDelete(criteria); err != nil {
		return derp.Wrap(err, location, "Unable to delete Rule", "userID: "+userID.Hex(), "ruleID: "+ruleID.Hex())
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

// New returns a fully initialized model.Rule as a data.Object.
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

func (service *Rule) ObjectQuery(session data.Session, result any, criteria exp.Expression, options ...option.Option) error {
	return service.collection(session).Query(result, notDeleted(criteria), options...)
}

func (service *Rule) ObjectLoad(session data.Session, criteria exp.Expression) (data.Object, error) {
	result := model.NewRule()
	err := service.Load(session, criteria, &result)
	return &result, err
}

func (service *Rule) ObjectSave(session data.Session, object data.Object, comment string) error {
	if rule, ok := object.(*model.Rule); ok {
		return service.Save(session, rule, comment)
	}
	return derp.Internal("service.Rule.ObjectSave", "Invalid Object Type", object)
}

func (service *Rule) ObjectDelete(session data.Session, object data.Object, comment string) error {
	if rule, ok := object.(*model.Rule); ok {
		return service.Delete(session, rule, comment)
	}
	return derp.Internal("service.Rule.ObjectDelete", "Invalid Object Type", object)
}

func (service *Rule) ObjectUserCan(object data.Object, authorization model.Authorization, action string) error {
	return derp.Unauthorized("service.Rule", "Not Authorized")
}

func (service *Rule) Schema() schema.Schema {
	return schema.New(model.RuleSchema())
}

/******************************************
 * Custom Queries
 ******************************************/

func (service *Rule) LoadByID(session data.Session, userID primitive.ObjectID, ruleID primitive.ObjectID, rule *model.Rule) error {

	// RULE: UserID cannot be zero
	if userID.IsZero() {
		return derp.Validation("UserID cannot be zero")
	}

	// RULE: RuleID cannot be zero
	if ruleID.IsZero() {
		return derp.Validation("RuleID cannot be zero")
	}

	criteria := exp.Equal("_id", ruleID).
		And(service.byUserID(userID))

	return service.Load(session, criteria, rule)
}

func (service *Rule) LoadByToken(session data.Session, userID primitive.ObjectID, token string, rule *model.Rule) error {

	// RULE: UserID cannot be zero
	if userID.IsZero() {
		return derp.Validation("UserID cannot be zero")
	}

	// RULE: token must be a valid ObjectID
	ruleID, err := primitive.ObjectIDFromHex(token)

	if err != nil {
		return derp.Wrap(err, "service.Rule.LoadByToken", "Error converting token to ObjectID", token)
	}

	criteria := exp.Equal("_id", ruleID).AndEqual("userId", userID)

	return service.Load(session, criteria, rule)
}

// LoadByTrigger retrieves a single Rule that maches the provided User, RuleType, and Trigger
func (service *Rule) LoadByTrigger(session data.Session, userID primitive.ObjectID, ruleType string, trigger string, rule *model.Rule) error {

	// RULE: UserID cannot be zero
	if userID.IsZero() {
		return derp.Validation("UserID cannot be zero")
	}

	// RULE: RuleType cannot be empty
	if ruleType == "" {
		return derp.Validation("RuleType cannot be empty")
	}

	// RULE: Trigger cannot be empty
	if trigger == "" {
		return derp.Validation("Trigger cannot be empty")
	}

	criteria := service.byUserID(userID).
		AndEqual("type", ruleType).
		AndEqual("trigger", trigger)

	return service.Load(session, criteria, rule)
}

// LoadByFollowing retrieves a single Rule that maches the provided User, Following, RuleType, and Trigger
func (service *Rule) LoadByFollowing(session data.Session, userID primitive.ObjectID, followingID primitive.ObjectID, ruleType string, trigger string, rule *model.Rule) error {

	// RULE: UserID cannot be zero
	if userID.IsZero() {
		return derp.Validation("UserID cannot be zero")
	}

	// RULE: FollowingID cannot be zero
	if followingID.IsZero() {
		return derp.Validation("FollowingID cannot be zero")
	}

	// RULE: RuleType cannot be empty
	if ruleType == "" {
		return derp.Validation("RuleType cannot be empty")
	}

	// RULE: Trigger cannot be empty
	if trigger == "" {
		return derp.Validation("Trigger cannot be empty")
	}

	criteria := exp.Equal("userId", userID).
		AndEqual("type", ruleType).
		AndEqual("trigger", trigger).
		AndEqual("followingId", followingID)

	return service.Load(session, criteria, rule)
}

// QueryPublic returns a collection of Rules that are marked Public, in reverse chronological order.
func (service *Rule) QueryPublic(session data.Session, userID primitive.ObjectID, maxDate int64, options ...option.Option) ([]model.Rule, error) {

	// RULE: UserID cannot be zero
	if userID.IsZero() {
		return nil, derp.Validation("UserID cannot be zero")
	}

	criteria := service.byUserID(userID).
		AndEqual("isPublic", true).
		AndLessThan("publishDate", maxDate)

	options = append(options, option.SortDesc("publishDate"))
	result, err := service.Query(session, criteria, options...)

	return result, err
}

func (service *Rule) QueryByType(session data.Session, userID primitive.ObjectID, ruleType string, criteria exp.Expression, options ...option.Option) ([]model.Rule, error) {

	criteria = service.byUserID(userID).
		AndEqual("type", ruleType).
		AndEqual("isPublic", true).
		And(criteria)

	options = append(options, option.SortDesc("publishDate"))
	result, err := service.Query(session, criteria, options...)

	return result, err
}

func (service *Rule) QueryByTypeDomain(session data.Session, userID primitive.ObjectID, criteria exp.Expression, options ...option.Option) ([]model.Rule, error) {
	return service.QueryByType(session, userID, model.RuleTypeDomain, criteria, options...)
}

// QueryByActorAndActions retrieves a slice of RuleSummaries that match the provided User, Actor, and potential actions
func (service *Rule) QueryByActorAndActions(session data.Session, userID primitive.ObjectID, actorID string, actions ...string) ([]model.RuleSummary, error) {

	criteria := exp.And(
		service.byUserID(userID),
		exp.Or(
			exp.Equal("type", model.RuleTypeActor).AndEqual("trigger", actorID),
			exp.Equal("type", model.RuleTypeDomain).AndEqual("trigger", dt.NameOnly(actorID)),
			exp.Equal("type", model.RuleTypeContent),
		),
		exp.In("action", actions),
	)

	return service.QuerySummary(session, criteria)
}

// QueryDomainBlocks returns all external domains blocked by this Instance/Domain.
func (service *Rule) QueryDomainBlocks(session data.Session) ([]model.Rule, error) {

	criteria := exp.Equal("userId", primitive.NilObjectID).
		AndEqual("type", model.RuleTypeDomain).
		AndEqual("behavior", model.RuleActionBlock)

	return service.Query(session, criteria, option.SortAsc("trigger"))
}

// QueryBlockedActors returns all Actors blocked by this User (or by the Domain on behalf of the User)
func (service *Rule) QueryBlockedActors(session data.Session, userID primitive.ObjectID) ([]model.Rule, error) {

	criteria := service.byUserID(userID).
		AndEqual("type", model.RuleTypeActor).
		AndEqual("behavior", model.RuleActionBlock)

	return service.Query(session, criteria, option.SortAsc("trigger"))
}

// RangeByUserID returns all Rules tha belong to a specific User (NO DOMAIN RULES)
func (service *Rule) RangeByUserID(session data.Session, userID primitive.ObjectID) (iter.Seq[model.Rule], error) {
	return service.Range(session, exp.Equal("userId", userID))
}

func (service *Rule) DeleteByUserID(session data.Session, userID primitive.ObjectID, comment string) error {

	const location = "service.Rule.DeleteByUserID"

	rangeFunc, err := service.RangeByUserID(session, userID)

	if err != nil {
		return derp.Wrap(err, location, "Error getting range function")
	}

	for rule := range rangeFunc {
		if err := service.Delete(session, &rule, comment); err != nil {
			return derp.Wrap(err, location, "Unable to delete rule", rule)
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
func (service *Rule) hasDuplicate(session data.Session, rule *model.Rule) bool {

	// Search the database for duplicate rules
	criteria := exp.NotEqual("_id", rule.RuleID).
		AndEqual("userId", rule.UserID).
		AndEqual("type", rule.Type).
		AndEqual("trigger", rule.Trigger)

	duplicate := model.NewRule()

	// If a duplicate is not found, then return FALSE
	if err := service.Load(session, criteria, &duplicate); derp.IsNotFound(err) {
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
