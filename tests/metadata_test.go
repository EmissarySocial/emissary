package tests

import (
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/model/step"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/schema"
	"github.com/benpate/rosetta/translate"
	"github.com/stretchr/testify/require"
)

// TestMetadata_BandwagonYear runs every file-metadata pipeline in Bandwagon's album and song
// templates against an album with a release date, and requires each `year` rule to write it
func TestMetadata_BandwagonYear(t *testing.T) {

	// BUG-258: all three rules read a field that nothing writes, so every download had no year
	group := loadGroup(t, templateGroup{Name: "bandwagon", Locations: []string{emissaryTemplates, "../../bandwagon"}})

	parent := model.NewStream()
	parent.Data = mapof.Any{"releaseDate": "2025-09-27"}

	stream := model.NewStream()

	yearRules := 0

	for _, templateID := range []string{"bandwagon-album", "bandwagon-song"} {

		template, err := group.Templates.Load(templateID)
		require.NoError(t, err)

		for actionID, action := range template.Actions {
			for _, pipeline := range metadataPipelines(t, action.Steps) {

				if !writesTarget(pipeline, "year") {
					continue
				}

				yearRules++
				result := runMetadataPipeline(t, pipeline, &parent, &stream)
				require.Equal(t, "2025", result["year"], "%s/%s", templateID, actionID)
			}
		}
	}

	// Two archive actions in the album, and the song's audio download
	require.Equal(t, 3, yearRules, "every year rule must be found, or a missing one passes silently")
}

// metadataPipelines returns the metadata pipeline of every archive and attachment step,
// including steps nested inside other steps
func metadataPipelines(t *testing.T, steps []step.Step) []translate.Pipeline {

	t.Helper()

	result := make([]translate.Pipeline, 0)

	for _, current := range steps {

		switch typed := current.(type) {

		case step.MakeArchive:
			result = append(result, archivePipelines(t, typed.Metadata)...)

		case step.GetArchive:
			result = append(result, archivePipelines(t, typed.Metadata)...)

		case step.ViewAttachment:
			result = append(result, typed.Metadata)
		}

		for _, subSteps := range nestedSteps(current) {
			result = append(result, metadataPipelines(t, subSteps)...)
		}
	}

	return result
}

// archivePipelines parses an archive step's metadata rules, one pipeline per archive depth
func archivePipelines(t *testing.T, levels [][]map[string]any) []translate.Pipeline {

	t.Helper()

	result := make([]translate.Pipeline, 0, len(levels))

	for _, rules := range levels {
		pipeline, err := translate.NewFromMap(rules...)
		require.NoError(t, err)
		result = append(result, pipeline)
	}

	return result
}

// writesTarget reports whether any rule in the pipeline writes the target path
func writesTarget(pipeline translate.Pipeline, target string) bool {

	for _, rule := range pipeline.MarshalSliceOfMap() {
		if rule["target"] == target {
			return true
		}
	}

	return false
}

// runMetadataPipeline executes a metadata pipeline with the same schemas that
// service.StreamArchive and build.StepViewAttachment use
func runMetadataPipeline(t *testing.T, pipeline translate.Pipeline, parent *model.Stream, stream *model.Stream) mapof.String {

	t.Helper()

	inSchema := schema.New(schema.Object{
		Properties: schema.ElementMap{
			"parent": model.StreamSchema(),
			"stream": model.StreamSchema(),
		},
	})

	outSchema := schema.New(schema.Object{
		Wildcard: schema.String{},
	})

	inObject := mapof.Any{
		"parent": parent,
		"stream": stream,
	}

	result := mapof.String{}
	require.NoError(t, pipeline.Execute(inSchema, inObject, outSchema, &result))

	return result
}
