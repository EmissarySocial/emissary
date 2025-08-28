package service

import (
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/streams"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// RuleFilter is a temporary object that filters ActivityStream documents based on
// the User- and Domain-Rules that are active for a given UserID.  Each RuleFilter
// can be used as many times as needed for a single User and HTTP Request.  RuleFilters
// are not thread-safe, and should not be shared between goroutines.
type RuleFilter struct {
	ruleService *Rule
	userID      primitive.ObjectID
	cache       map[string][]model.RuleSummary

	allowLabels bool
	allowMutes  bool
	allowBlocks bool
}

// NewRuleFilter returns a fully initialized RuleFilter that is keyed to a specific User.
func NewRuleFilter(ruleService *Rule, userID primitive.ObjectID, options ...RuleFilterOption) RuleFilter {
	result := RuleFilter{
		ruleService: ruleService,
		userID:      userID,
		cache:       make(map[string][]model.RuleSummary),

		allowLabels: true,
		allowMutes:  true,
		allowBlocks: true,
	}

	for _, option := range options {
		option(&result)
	}

	return result
}

// Allow returns TRUE if this document is allowed past all User and Domain filters.
// The document is passed as a pointer because it MAY BE MODIFIED by the filter, for
// instance, to add a label or other metadata.
func (filter *RuleFilter) Allow(session data.Session, document *streams.Document) bool {

	// Get the actor ID from the document.
	actorID := document.Actor().ID()

	// If we don't have a cached value for this actor, then load it from the database.
	if filter.cache[actorID] == nil {

		allowedActions := filter.allowedActions()
		rules, err := filter.ruleService.QueryByActorAndActions(session, filter.userID, actorID, allowedActions...)

		if err != nil {
			derp.Report(derp.Wrap(err, "emissary.RuleFilter.FilterOne", "Error loading rules"))
			return false
		}

		filter.cache[actorID] = rules
	}

	// Verify each rule
	for _, rule := range filter.cache[actorID] {

		if rule.IsDisallowed(document) {
			return false
		}
	}

	// If we've passed all the rules successfully, then tell the caller to allow this document.
	return true
}

// Disallow returns TRUE if a document is NOT allowed past all User and Domain filters.
func (filter *RuleFilter) Disallow(session data.Session, document *streams.Document) bool {
	return !filter.Allow(session, document)
}

// Channel returns a channel of all documents that are allowed by User/Domain filters.
// Documents may be modified by filters in the process, for instance, to add content warning labels.
// Deprecated: this should be removed in favor of range function iterators.
func (filter *RuleFilter) Channel(ch <-chan streams.Document) <-chan streams.Document {

	const location = "service.RuleFilter.Channel"
	result := make(chan streams.Document)

	// TODO: CRITICAL: Make thread-safe session here...

	go func() {

		defer close(result)

		session, cancel, err := filter.ruleService.factory.Session(time.Minute)

		if err != nil {
			derp.Report(derp.Wrap(err, location, "Unable to connect to database"))
			return
		}

		defer cancel()

		for document := range ch {
			if filter.Allow(session, &document) {
				result <- document
			}
		}
	}()

	return result
}

func (filter *RuleFilter) allowedActions() []string {

	result := make([]string, 0, 3)

	if filter.allowLabels {
		result = append(result, model.RuleActionLabel)
	}

	if filter.allowMutes {
		result = append(result, model.RuleActionMute)
	}

	if filter.allowBlocks {
		result = append(result, model.RuleActionBlock)
	}

	return result
}
