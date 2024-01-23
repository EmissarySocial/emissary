package service

import (
	"github.com/EmissarySocial/emissary/model"
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
}

// NewRuleFilter returns a fully initialized RuleFilter that is keyed to a specific User.
func NewRuleFilter(ruleService *Rule, userID primitive.ObjectID) RuleFilter {
	return RuleFilter{
		ruleService: ruleService,
		userID:      userID,
		cache:       make(map[string][]model.RuleSummary),
	}
}

// Allow returns TRUE if this document is allowed past all User and Domain filters.
// The document is passed as a pointer because it MAY BE MODIFIED by the filter, for
// instance, to add a label or other metadata.
func (filter *RuleFilter) Allow(document *streams.Document) bool {

	// Get the actor ID from the document.
	actorID := document.Actor().ID()

	// If we don't have a cached value for this actor, then load it from the database.
	if filter.cache[actorID] == nil {

		rules, err := filter.ruleService.QueryByActor(filter.userID, actorID)

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
func (filter *RuleFilter) Disallow(document *streams.Document) bool {
	return !filter.Allow(document)
}

// Channel returns a channel of all documents that are allowed by User/Domain filters.
// Documents may be modified by filters in the process, for instance, to add content warning labels.
func (filter *RuleFilter) Channel(ch <-chan streams.Document) <-chan streams.Document {

	result := make(chan streams.Document)

	go func() {
		defer close(result)

		for document := range ch {
			if filter.Allow(&document) {
				result <- document
			}
		}
	}()

	return result
}

// Slice returns a slice of all documents from the input that are allowed by User/Domain filters.
// Documents may be modified by filters in the process, for instance, to add content warning labels.
func (filter *RuleFilter) Slice(documents []streams.Document) []streams.Document {

	result := make([]streams.Document, 0, len(documents))

	for _, document := range documents {
		if filter.Allow(&document) {
			result = append(result, document)
		}
	}

	return result
}
