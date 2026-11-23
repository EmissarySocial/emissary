package service

import (
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/require"
)

// TestMigrationCollections_EveryListAgrees ties together the four lists that an account move
// depends on: the collections an actor advertises, the export routes, the importers, and the
// import plan (BUG-83)
func TestMigrationCollections_EveryListAgrees(t *testing.T) {

	factory := &Factory{}
	exportService := Export{factory: factory}
	importService := Import{}
	locator := factory.ImportableLocator()

	// Build the actor document that a server advertising every collection would publish
	migration := mapof.Any{}

	for _, collection := range model.MigrationCollections() {
		migration[collection.Key] = "https://example.com/@user/export/" + collection.Path
	}

	actor := streams.NewDocument(mapof.Any{vocab.PropertyMigration: migration})
	offered := map[string]string{}

	for _, planItem := range importService.CalcImportPlan(actor) {
		offered[planItem.Value] = planItem.Href
	}

	for _, collection := range model.MigrationCollections() {

		exportable, err := exportService.FindService(collection.Path)
		require.NoError(t, err, "no export route serves %s", collection.Path)
		require.NotNil(t, exportable, collection.Path)

		importable, err := locator(collection.Key)
		require.NoError(t, err, "no importer handles %s", collection.Key)
		require.NotNil(t, importable, collection.Key)

		require.Equal(t, migration[collection.Key], offered[collection.Key], "the import plan does not offer %s", collection.Key)
	}

	// The plan offers nothing native that the table does not list
	for value := range offered {
		_, listed := migration[value]
		require.True(t, listed, "the import plan offers %s, which no actor advertises", value)
	}
}

// TestCalcImportPlan_StreamsBeforeOutbox verifies the order that an imported outbox depends on
func TestCalcImportPlan_StreamsBeforeOutbox(t *testing.T) {

	importService := Import{}
	actor := streams.NewDocument(mapof.Any{vocab.PropertyMigration: mapof.Any{
		"emissary:stream":        "https://example.com/export/emissary-stream",
		"emissary:outboxItem":    "https://example.com/export/emissary-outboxItem",
		"emissary:outboxMessage": "https://example.com/export/emissary-outboxMessage",
		"outbox":                 "https://example.com/export/outbox",
	}})

	values := make([]string, 0)

	for _, planItem := range importService.CalcImportPlan(actor) {
		values = append(values, planItem.Value)
	}

	require.Equal(t, []string{"emissary:stream", "emissary:outboxItem", "emissary:outboxMessage"}, values)
}

// TestCalcImportPlan_StandardOutbox verifies that the standard collection is offered only when no native one is
func TestCalcImportPlan_StandardOutbox(t *testing.T) {

	importService := Import{}
	actor := streams.NewDocument(mapof.Any{vocab.PropertyMigration: mapof.Any{
		"outbox": "https://example.com/export/outbox",
	}})

	plan := importService.CalcImportPlan(actor)
	require.Len(t, plan, 1)
	require.Equal(t, "outbox", plan[0].Value)
}
