package build

import (
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/exp"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// These tests pin the permission gate that StepViewFeed applies to the RSS/Atom/JSON
// feed. The feed query is `parentId AND defaultAllowed() AND withinPublishDate()`
// (build/step_ViewFeed.go), mirroring the HTML view (build.Stream.Children). The
// security-critical piece is Common.defaultAllowed(): without it the feed leaked
// gated (circle/paid/non-anonymous) children to anonymous callers. These tests prove
// defaultAllowed() restricts non-owners to their permission set and fails closed.

// stubPermissionFactory satisfies the (large) build.Factory interface by embedding it
// (nil) and overriding only Permission(). Permission.Permissions() reads only its
// arguments -- no database -- so a zero-value service is sufficient. Any other Factory
// call panics, which is fine: these tests never make one.
type stubPermissionFactory struct {
	Factory
}

func (f stubPermissionFactory) Permission() *service.Permission {
	permission := service.NewPermission()
	return &permission
}

// Stream returns a zero-value Stream service. NewQueryBuilder only stores the
// service reference (it makes no call until the query is iterated), so a bare
// value is enough for the criteria-pinning tests that never touch the database.
func (f stubPermissionFactory) Stream() *service.Stream {
	return &service.Stream{}
}

// collectPredicates walks an Expression and returns every Predicate it contains.
// (AndExpression.Match short-circuits on a false return, so the visitor returns true.)
func collectPredicates(criteria exp.Expression) []exp.Predicate {
	result := make([]exp.Predicate, 0)
	criteria.Match(func(predicate exp.Predicate) bool {
		result = append(result, predicate)
		return true
	})
	return result
}

// findPredicate returns the first Predicate on the given field/operator, and whether it exists.
func findPredicate(predicates []exp.Predicate, field string, operator string) (exp.Predicate, bool) {
	for _, predicate := range predicates {
		if (predicate.Field == field) && (predicate.Operator == operator) {
			return predicate, true
		}
	}
	return exp.Predicate{}, false
}

func TestCommon_defaultAllowed_Anonymous(t *testing.T) {

	// An anonymous viewer: no authorization at all.
	viewer := Common{
		_factory:       stubPermissionFactory{},
		_authorization: model.Authorization{},
	}

	predicates := collectPredicates(viewer.defaultAllowed())

	// RULE: The query must be restricted to streams whose defaultAllow includes the viewer's
	// permissions -- and an anonymous viewer only carries the Anonymous magic group.
	predicate, exists := findPredicate(predicates, "defaultAllow", exp.OperatorIn)
	require.True(t, exists, "anonymous feed query MUST filter by defaultAllow")

	permissions, ok := predicate.Value.(model.Permissions)
	require.True(t, ok, "defaultAllow filter value must be a Permissions slice")
	require.Contains(t, permissions, model.MagicGroupIDAnonymous, "anonymous viewer must match anonymous-allowed streams")
	require.NotContains(t, permissions, model.MagicGroupIDAuthenticated, "anonymous viewer must NOT carry the authenticated group")

	// RULE: Deleted streams are always excluded.
	_, hasDeleteGuard := findPredicate(predicates, "deleteDate", exp.OperatorEqual)
	require.True(t, hasDeleteGuard, "must exclude deleted streams")
}

func TestCommon_defaultAllowed_Member(t *testing.T) {

	// A signed-in User who belongs to a private Group.
	userID := primitive.NewObjectID()
	privateGroup := primitive.NewObjectID()

	viewer := Common{
		_factory: stubPermissionFactory{},
		_authorization: model.Authorization{
			UserID:   userID,
			GroupIDs: []primitive.ObjectID{privateGroup},
		},
	}

	predicates := collectPredicates(viewer.defaultAllowed())

	predicate, exists := findPredicate(predicates, "defaultAllow", exp.OperatorIn)
	require.True(t, exists, "member feed query MUST filter by defaultAllow")

	permissions, ok := predicate.Value.(model.Permissions)
	require.True(t, ok, "defaultAllow filter value must be a Permissions slice")

	// RULE: The member's permission set carries the anonymous base, the authenticated group,
	// their own UserID, and each of their group memberships.
	require.Contains(t, permissions, model.MagicGroupIDAnonymous)
	require.Contains(t, permissions, model.MagicGroupIDAuthenticated)
	require.Contains(t, permissions, userID)
	require.Contains(t, permissions, privateGroup)
}

func TestCommon_defaultAllowed_DomainOwner(t *testing.T) {

	// A domain owner sees everything: the criteria must NOT restrict by defaultAllow.
	viewer := Common{
		_factory: stubPermissionFactory{},
		_authorization: model.Authorization{
			DomainOwner: true,
		},
	}

	predicates := collectPredicates(viewer.defaultAllowed())

	_, exists := findPredicate(predicates, "defaultAllow", exp.OperatorIn)
	require.False(t, exists, "domain owner must not be restricted by defaultAllow")

	// The delete guard is still present (owners don't see deleted streams here).
	_, hasDeleteGuard := findPredicate(predicates, "deleteDate", exp.OperatorEqual)
	require.True(t, hasDeleteGuard, "must still exclude deleted streams")
}
