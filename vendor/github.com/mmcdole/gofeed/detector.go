package gofeed

import (
	"bytes"
	"io"
	"strings"

	jsoniter "github.com/json-iterator/go"
	"github.com/mmcdole/gofeed/internal/shared"
	xpp "github.com/mmcdole/goxpp"
)

// FeedType represents one of the possible feed
// types that we can detect.
type FeedType int

const (
	// FeedTypeUnknown represents a feed that could not have its
	// type determiend.
	FeedTypeUnknown FeedType = iota
	// FeedTypeAtom repesents an Atom feed
	FeedTypeAtom
	// FeedTypeRSS represents an RSS feed
	FeedTypeRSS
	// FeedTypeJSON represents a JSON feed
	FeedTypeJSON
)

// DetectFeedType attempts to determine the type of feed
// by looking for specific xml elements unique to the
// various feed types.
func DetectFeedType(feed io.Reader) FeedType {
	buffer := new(bytes.Buffer)
	buffer.ReadFrom(feed)

	var firstChar byte
	loop: for {
		ch, err := buffer.ReadByte()
		if err != nil {
			return FeedTypeUnknown
		}
		// ignore leading whitespace & byte order marks
		switch ch {
		case ' ', '\r', '\n', '\t':
		case 0xFE, 0xFF, 0x00, 0xEF, 0xBB, 0xBF:  // utf 8-16-32 bom
		default:
			firstChar = ch
			buffer.UnreadByte()
			break loop
		}
	}

	if firstChar == '<' {
		// Check if it's an XML based feed
		p := xpp.NewXMLPullParser(bytes.NewReader(buffer.Bytes()), false, shared.NewReaderLabel)

		xmlBase := shared.XMLBase{}
		_, err := xmlBase.FindRoot(p)
		if err != nil {
			return FeedTypeUnknown
		}

		name := strings.ToLower(p.Name)
		switch name {
		case "rdf":
			return FeedTypeRSS
		case "rss":
			return FeedTypeRSS
		case "feed":
			return FeedTypeAtom
		default:
			return FeedTypeUnknown
		}
	} else if firstChar == '{' {
		// Check if document is valid JSON
		if jsoniter.Valid(buffer.Bytes()) {
			return FeedTypeJSON
		}
	}
	return FeedTypeUnknown
}
