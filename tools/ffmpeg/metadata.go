package ffmpeg

import (
	"bytes"
	"io"
	"os/exec"
	"strings"

	"github.com/benpate/derp"
	"github.com/rs/zerolog/log"
)

// Some info on FFmpeg metadata
// https://gist.github.com/eyecatchup/0757b3d8b989fe433979db2ea7d95a01
// https://jmesb.com/how_to/create_id3_tags_using_ffmpeg
// https://wiki.multimedia.cx/index.php?title=FFmpeg_Metadata

// How to include album art...
// https://www.bannerbear.com/blog/how-to-add-a-cover-art-to-audio-files-using-ffmpeg/

// SetMetadata uses FFmpeg to apply additional metadata to a file.
// It supports all standard FFmpeg metadata, and adds a special case for cover art.  Use "cover" to
// provide the URL of a cover art image
func SetMetadata(input io.Reader, mimeType string, metadata map[string]string, output io.Writer) error {

	const location = "ffmpeg.SetMetadata"

	log.Trace().Str("location", location).Msg("Setting metadata")

	// RULE: If FFmpeg is not installed, then break
	if !isFFmpegInstalled {
		return derp.NewInternalError(location, "FFmpeg is not installed")
	}

	// RULE: If there is no metadata to set, then just copy the input to the output
	if len(metadata) == 0 {

		log.Trace().Str("location", location).Msg("Metadata is empty.  Copying input directly to output")

		if _, err := io.Copy(output, input); err != nil {
			return derp.Wrap(err, location, "No metadata to set", "Error copying input to output")
		}
		return nil
	}

	// Let's assemble the arguments we're going to send to FFmpeg
	var errors bytes.Buffer
	args := make([]string, 0)

	// Just some sugar to append to arguments list
	add := func(values ...string) {
		args = append(args, values...)
	}

	add("-f", ffmpegFormat(mimeType)) // specify input format because it can't be deduced from a pipe
	add("-i", "pipe:0")               // read the input from stdin

	// Special case for cover art
	if cover := metadata["cover"]; cover != "" {
		add("-i", cover+".jpg?width=300&height=300")  // read the cover art from a URL
		add("-map", "0:a")                            // Map audio into the output file
		add("-map", "1:v")                            // Map cover art into the output file
		add("-c:v", "copy")                           // use the original codec without change
		add("-metadata:s:v", "title=Album Cover")     // Label the image so that readers will recognize it
		add("-metadata:s:v", "comment=Cover (front)") // Label the image so that readers will recognize it
	}

	// Add all other metadata fields
	for key, value := range metadata {

		switch key {
		case "cover": // NOOP. Already handled above
		default:
			value = strings.ReplaceAll(value, "\n", `\n`)
			add("-metadata", key+"="+value)
		}
	}

	add("-f", ffmpegFormat(mimeType)) // specify the same format for the output (because it can't be deduced from a pipe)
	add("-c:a", "copy")               // use the original codec without change
	add("-flush_packets", "0")        // wait for max size before writing: https://stackoverflow.com/questions/54620528/metadata-in-mp3-not-working-when-piping-from-ffmpeg-with-album-art
	add("pipe:1")                     // write output to the output writer

	log.Trace().Str("location", location).Msg("ffmpeg " + strings.Join(args, " "))

	// Set up the FFmpeg command
	command := exec.Command("ffmpeg", args...)
	command.Stdin = input
	command.Stdout = output
	command.Stderr = &errors

	// Execute FFmpeg command
	if err := command.Run(); err != nil {
		return derp.Wrap(err, location, "Error running FFmpeg", errors.String(), args)
	}

	// UwU
	return nil
}

/*
// downloadImage loads an image from a URL and returns the local filename and a pointer to the file
func downloadImage(url string) (string, error) {

	const location = "ffmpeg.downloadImage"

	log.Trace().Str("url", url).Msg("Downloading image")

	// Create a temp file for the image
	tempDir := os.TempDir()
	tempFilename := strings.ReplaceAll(url, "/", "_")
	tempFile, err := os.CreateTemp(tempDir, tempFilename)

	if err != nil {

		return "", derp.ReportAndReturn(derp.Wrap(err, location, "Error creating temporary file", url))
	}
	defer tempFile.Close()

	// Load the image from the URL
	txn := remote.Get(url).Result(tempFile)

	if err := txn.Send(); err != nil {
		return "", derp.ReportAndReturn(derp.Wrap(err, location, "Error downloading image", url))
	}

	log.Trace().Str("filename", tempFilename).Msg("Image received successfully")
	return tempDir + tempFilename, nil
}
*/

func ffmpegFormat(mimeType string) string {
	switch mimeType {
	case "audio/mpeg":
		return "mp3"
	case "audio/ogg":
		return "ogg"
	case "audio/flac":
		return "flac"
	case "audio/mp4":
		return "mp4"
	default:
		return "mp3"
	}
}
