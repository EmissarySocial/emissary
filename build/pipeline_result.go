package build

import (
	"encoding/json"
	"net/http"

	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
)

type PipelineResult struct {
	StatusCode  int          // HTTP Status Code to be returned
	ContentType string       // If present, then this option sets the content-type header
	Headers     mapof.String // Map of header values to be applied to the response
	Events      mapof.String // Map of events to trigger on the client (via HX-Trigger)
	FullPage    bool         // If true, then this result represents the entire page of content, and should not be wrapped in the global template
	Halt        bool         // If true, then this pipeline should halt execution
	Error       error        // If present, then there was an error building this page
}

func NewPipelineResult() PipelineResult {
	return PipelineResult{
		Headers: mapof.NewString(),
		Events:  mapof.NewString(),
	}
}

func (result PipelineResult) GetContentType() string {

	if result.ContentType != "" {
		return result.ContentType
	}

	return "text/html"
}

func (result PipelineResult) GetStatusCode() int {

	if result.StatusCode != 0 {
		return result.StatusCode
	}

	if result.Error != nil {
		return derp.ErrorCode(result.Error)
	}

	return http.StatusOK
}

func (result PipelineResult) Apply(response http.ResponseWriter) {

	header := response.Header()
	header.Set("Content-Type", result.GetContentType())

	// Copy HX-Trigger events into response
	if len(result.Events) > 0 {
		if hxTrigger, err := json.Marshal(result.Events); err == nil {
			header.Set("HX-Trigger", string(hxTrigger))
		}
	}

	// Copy OTHER headers into response
	for name, value := range result.Headers {
		header.Set(name, value)
	}
}

// Merge combines two PipelineResult objects into one.
func (result *PipelineResult) Merge(newStatus PipelineResult) {

	// Copy bools into the result
	result.FullPage = newStatus.FullPage || result.FullPage
	result.Halt = newStatus.Halt || result.Halt

	// Copy Content Type into the result
	if newStatus.ContentType != "" {
		result.ContentType = newStatus.ContentType
	}

	// Copy Status Code into the result
	if newStatus.StatusCode != 0 {
		result.StatusCode = newStatus.StatusCode
	}

	// Copy HTTP headers into the result
	for name, value := range newStatus.Headers {
		if _, ok := result.Headers[name]; !ok {
			result.Headers[name] = value
		}
	}

	// Copy HX-Trigger headers into the result
	for name, value := range newStatus.Events {
		if _, ok := result.Events[name]; !ok {
			result.Events[name] = value
		}
	}

	// Copy Error value into the result
	if newStatus.Error != nil {
		result.Error = newStatus.Error
	}
}
