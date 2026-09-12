package derpmongo

import (
	"testing"

	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/assert"
)

func TestNew_ReadsOptions(t *testing.T) {

	plugin := New(nil, mapof.Any{
		"include-codes": []any{500, 502},
		"exclude-codes": []any{404},
	})

	assert.Equal(t, 2, len(plugin.includeCodes))
	assert.Equal(t, 1, len(plugin.excludeCodes))
}

func TestNew_WithoutOptions(t *testing.T) {

	plugin := New(nil, mapof.Any{})

	assert.Empty(t, plugin.includeCodes)
	assert.Empty(t, plugin.excludeCodes)
}

func TestPlugin_IsReportable_NoFilters(t *testing.T) {

	plugin := New(nil, mapof.Any{})

	// An unconfigured plugin logs everything it is handed
	assert.True(t, plugin.isReportable(500))
	assert.True(t, plugin.isReportable(404))
	assert.True(t, plugin.isReportable(0))
}

func TestPlugin_IsReportable_ExcludeList(t *testing.T) {

	plugin := New(nil, mapof.Any{"exclude-codes": []any{404, 403}})

	assert.False(t, plugin.isReportable(404))
	assert.False(t, plugin.isReportable(403))
	assert.True(t, plugin.isReportable(500))
}

func TestPlugin_IsReportable_IncludeList(t *testing.T) {

	plugin := New(nil, mapof.Any{"include-codes": []any{500, 502}})

	assert.True(t, plugin.isReportable(500))
	assert.True(t, plugin.isReportable(502))
	assert.False(t, plugin.isReportable(404), "an include list silences everything outside of it")
}

func TestPlugin_IsReportable_ExcludeWinsOverInclude(t *testing.T) {

	// A code named by both lists is excluded, because the exclude test runs first
	plugin := New(nil, mapof.Any{
		"include-codes": []any{500},
		"exclude-codes": []any{500},
	})

	assert.False(t, plugin.isReportable(500))
}

func TestPlugin_ReportIgnoresNil(t *testing.T) {

	// The collection is nil, so anything that reached the insert would panic
	plugin := New(nil, mapof.Any{})

	assert.NotPanics(t, func() {
		plugin.Report(nil)
	})
}

func TestPlugin_ReportIgnoresFilteredErrors(t *testing.T) {

	// The collection is nil, so this passes only because the filter returns before the insert
	plugin := New(nil, mapof.Any{"exclude-codes": []any{404}})

	assert.NotPanics(t, func() {
		plugin.Report(derpNotFound())
	})
}

// derpNotFound returns an error carrying a 404, which the exclude-list tests filter on
func derpNotFound() error {
	return derp.NotFound("service.Test.Load", "record not found")
}
