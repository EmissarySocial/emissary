package service

import (
	"github.com/benpate/rosetta/translate"
)

// StreamArchiveOptions defines the options for making a StreamArchive ZIP file
type StreamArchiveOptions struct {
	Token       string
	Depth       int
	JSON        bool
	Attachments bool
	Metadata    []translate.Pipeline
}

// HasNext returns TRUE if the depth of this export is greater than zero
func (options StreamArchiveOptions) HasNext() bool {
	return options.Depth > 0
}

// Next returns a new StreamArchiveOptions object that is one level deeper than the current object
func (options StreamArchiveOptions) Next() StreamArchiveOptions {

	var metadata []translate.Pipeline

	if len(options.Metadata) > 0 {
		metadata = options.Metadata[1:]
	}

	return StreamArchiveOptions{
		Token:       options.Token,
		Depth:       options.Depth - 1,
		JSON:        options.JSON,
		Attachments: options.Attachments,
		Metadata:    metadata,
	}
}

// Pipeline returns the Metadata pipeline for the current export level.
// If a pipeline has not been defined, then an empty pipeline is returned
func (options StreamArchiveOptions) Pipeline() translate.Pipeline {

	if len(options.Metadata) > 0 {
		return options.Metadata[0]
	}

	return translate.Pipeline{}
}
