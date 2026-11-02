package model

import (
	"testing"

	"github.com/EmissarySocial/emissary/tools/id"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/schema"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// TestSchemaWildcards_KeepValues confirms that validating wildcard values leaves stored Domain and Connection data unchanged (BUG-265)
func TestSchemaWildcards_KeepValues(t *testing.T) {

	// Theme data holds CSS (with whitespace and ">" selectors) and booleans
	themeData := mapof.Any{
		"stylesheet":     "nav > a {\n\tcolor: red;\n}\n\n.card  .title { font-weight: 600; }",
		"showNavigation": true,
	}
	requireUnchanged(t, DomainSchema(), "themeData", &themeData)
	require.Equal(t, true, themeData["showNavigation"])

	// Operational data holds keys that must stay verbatim
	domainData := mapof.String{"vapidPublicKey": "BOr7sJ9_v1xQ==  <key>"}
	requireUnchanged(t, DomainSchema(), "data", &domainData)

	// Connection data holds provider settings, including arrays
	connectionData := mapof.Any{"shareGroups": []any{"000000000000000000000001"}}
	requireUnchanged(t, ConnectionSchema(), "data", &connectionData)
}

// requireUnchanged normalizes a value with one property's element and fails if anything was rewritten
func requireUnchanged(t *testing.T, element schema.Element, property string, value any) {
	t.Helper()

	object, ok := element.(schema.Object)
	require.True(t, ok, "schema must be a schema.Object")

	rewrites, err := schema.New(object.Properties[property]).Normalize(value)
	require.NoError(t, err, property)
	require.Empty(t, rewrites, property)
}

// TestSchemaWildcards_StreamPermissions confirms that a Stream's group and circle IDs pass wildcard validation unchanged (BUG-265)
func TestSchemaWildcards_StreamPermissions(t *testing.T) {

	groupID := primitive.NewObjectID()
	circleID := primitive.NewObjectID()

	stream := NewStream()
	stream.TemplateID = "article"
	stream.Groups = mapof.Object[id.Slice]{"viewer": id.Slice{groupID}}
	stream.Circles = mapof.Object[id.Slice]{"editor": id.Slice{circleID}}

	rewrites, err := schema.New(StreamSchema()).Normalize(&stream)
	require.NoError(t, err)
	require.Empty(t, rewrites)
	require.Equal(t, id.Slice{groupID}, stream.Groups["viewer"])
	require.Equal(t, id.Slice{circleID}, stream.Circles["editor"])
}
