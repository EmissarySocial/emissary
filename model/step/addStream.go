package step

import (
	"github.com/EmissarySocial/emissary/tools/val"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/first"
	"github.com/benpate/rosetta/mapof"
)

// AddStream is an action that can add new sub-streams to the domain.
//
// Uses:
// Display a pop-up to choose a template and create a new stream
// Embed a custom "create" widget into the current page - possibly selecting between multiple templates
// Create a new stream using a specific template as a part of a larger pipeline
type AddStream struct {
	Title         string   // Title to use on the create modal. Defaults to "Add a Stream"
	Location      string   // Options are: "top", "child", "outbox".  Defaults to "child".
	Templates     []string // List of acceptable templates that can be used to make a stream.  If empty, then all templates are valid.
	AsEmbed       bool     // If TRUE, then use embed the "create" action of the selected template into the current page.
	AsReply       bool     // If TRUE, then the new stream will be created as a reply to the current model object (only works with DocumentLinkers: Streams and Messages).
	WithNewStream []Step   // List of steps to take on the newly created child record on POST.
}

// NewAddStream returns a fully initialized AddStream record
func NewAddStream(stepInfo mapof.Any) (AddStream, error) {

	withNewStream, err := NewPipeline(stepInfo.GetSliceOfMap("with-stream"))

	if err != nil {
		return AddStream{}, derp.Wrap(err, "model.setp.NewAddStream", "Error parsing with-stream steps")
	}

	return AddStream{
		Title:         first.String(stepInfo.GetString("title"), "Add a Stream"),
		Location:      val.Enum(stepInfo.GetString("location"), "top", "child", "outbox"),
		Templates:     stepInfo.GetSliceOfString("template"),
		AsEmbed:       stepInfo.GetBool("as-embed"),
		AsReply:       stepInfo.GetBool("as-reply"),
		WithNewStream: withNewStream,
	}, nil
}

// AmStep is here to verify that this struct is a render pipeline step
func (step AddStream) AmStep() {}
