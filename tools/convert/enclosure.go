package convert

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/gorilla/feeds"
	"github.com/kr/jsonfeed"
)

// enclosureDefaultType is the media type of an enclosure whose Stream names none
const enclosureDefaultType = "audio/mpeg"

// streamEnclosure returns the media file that a Stream offers to feed readers, such as a podcast episode's
// audio.  A Template opts in by writing data.enclosureUrl (and, optionally, data.enclosureType) on save.
// ok is FALSE when the Stream has none.
func streamEnclosure(stream model.Stream) (url string, mediaType string, ok bool) {

	url = stream.Data.GetString("enclosureUrl")

	if url == "" {
		return "", "", false
	}

	mediaType = stream.Data.GetString("enclosureType")

	if mediaType == "" {
		mediaType = enclosureDefaultType
	}

	return url, mediaType, true
}

// gorillaEnclosure returns a Stream's enclosure as an RSS/Atom enclosure, or nil when it has none.
// RSS requires a length, and the size of a file transcoded on request is unknown, so it is "0", which
// podcast clients accept.
func gorillaEnclosure(stream model.Stream) *feeds.Enclosure {

	url, mediaType, ok := streamEnclosure(stream)

	if !ok {
		return nil
	}

	return &feeds.Enclosure{Url: url, Type: mediaType, Length: "0"}
}

// jsonFeedAttachments returns a Stream's enclosure as JSON Feed attachments, or nil when it has none
func jsonFeedAttachments(stream model.Stream) []jsonfeed.Attachment {

	url, mediaType, ok := streamEnclosure(stream)

	if !ok {
		return nil
	}

	return []jsonfeed.Attachment{{URL: url, MIMEType: mediaType}}
}
