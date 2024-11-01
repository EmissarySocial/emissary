package step

import (
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/convert"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/translate"
)

// Export is an action that can add new model objects of any type
type Export struct {
	Depth       int
	Attachments bool
	Metadata    []translate.Pipeline
}

// NewExport returns a fully initialized Export record
func NewExport(stepInfo mapof.Any) (Export, error) {

	// Get Translation Pipeline for Metadata
	allMetadata := stepInfo.GetSliceOfAny("metadata")

	// Success
	result := Export{
		Depth:       stepInfo.GetInt("depth"),
		Attachments: stepInfo.GetBool("attachments"),
		Metadata:    make([]translate.Pipeline, 0, len(allMetadata)),
	}

	for _, metadataAny := range allMetadata {

		metadataSliceOfMap := convert.SliceOfMap(metadataAny)
		pipeline, err := translate.NewFromMap(metadataSliceOfMap...)

		if err != nil {
			return Export{}, derp.Wrap(err, "step.NewExport", "Error creating metadata pipeline")
		}

		result.Metadata = append(result.Metadata, pipeline)
	}

	return result, nil
}

// AmStep is here only to verify that this struct is a build pipeline step
func (step Export) AmStep() {}
