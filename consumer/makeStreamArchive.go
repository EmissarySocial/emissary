package consumer

import (
	"github.com/EmissarySocial/emissary/domain"
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/convert"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/translate"
	"github.com/benpate/turbine/queue"
	"github.com/rs/zerolog/log"
)

func MakeStreamArchive(factory *domain.Factory, _ *service.Stream, stream *model.Stream, args mapof.Any) queue.Result {

	const location = "consumer.MakeStreamArchive"
	log.Trace().Str("location", location).Str("stream", stream.StreamID.Hex()).Msg("Making Archive...")

	// Collect metadata and convert into a slice of pipelines
	metadataAny := args.GetSliceOfAny("metadata")
	metadataSlice := make([][]map[string]any, len(metadataAny))

	for index, item := range metadataAny {
		metadataSlice[index] = convert.SliceOfMap(item)
	}

	metadata, err := translate.NewSliceOfPipelines(metadataSlice)

	if err != nil {
		return queue.Failure(derp.Wrap(err, location, "Error creating pipeline", metadata))
	}

	// Assemble StreamArchiveOptions
	streamArchiveOptions := service.StreamArchiveOptions{
		Token:       args.GetString("token"),
		Depth:       args.GetInt("depth"),
		JSON:        args.GetBool("json"),
		Attachments: args.GetBool("attachments"),
		Metadata:    metadata,
	}

	// Create the StreamArchive
	streamArchiveService := factory.StreamArchive()

	if err := streamArchiveService.Create(stream, streamArchiveOptions); err != nil {
		return queue.Error(derp.Wrap(err, location, "Error creating archive"))
	}

	// Done.
	log.Trace().Str("location", location).Str("stream", stream.StreamID.Hex()).Msg("StreamArchive: complete.")
	return queue.Success()
}
