package model

import (
	"bytes"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/benpate/rosetta/list"
)

// asfHeaderGUID identifies the ASF container that holds Windows Media audio and video.
var asfHeaderGUID = []byte{0x30, 0x26, 0xB2, 0x75, 0x8E, 0x66, 0xCF, 0x11, 0xA6, 0xD9, 0x00, 0xAA, 0x00, 0x62, 0xCE, 0x6C}

// ebmlMagic identifies the EBML container that holds WebM and Matroska media.
var ebmlMagic = []byte{0x1A, 0x45, 0xDF, 0xA3}

// DetectContentType returns the media type of a file's leading bytes, without parameters.
func DetectContentType(header []byte, filename string) string {

	// This is the only supported way to calculate Attachment.ContentType: the file's own
	// bytes decide the answer, so it cannot be talked into a type by a filename, a
	// multipart header, or a remote server's JSON.

	// RULE: The filename is consulted only to choose between the audio and video flavors
	// of a container the bytes have already confirmed (MP4, WebM, ASF).  It can never
	// promote arbitrary bytes into a media type.
	extension := strings.ToLower(filepath.Ext(filename))

	// Sniff signatures the standard library does not know (mostly audio formats).
	if contentType := sniffExtendedContentType(header, extension); contentType != "" {
		return contentType
	}

	// Fall back to the standard library.  http.DetectContentType appends "; charset=utf-8"
	// to the text types, which callers neither store nor compare against.
	detected := strings.TrimSpace(list.Semicolon(http.DetectContentType(header)).First())

	// Bare MPEG audio framing (untagged MP3, raw ADTS AAC) has no fixed magic number, only
	// a bit-pattern heuristic, so it runs last -- after every exact signature has come up
	// empty -- to keep it from shadowing a real match.
	if detected == "application/octet-stream" {
		if contentType := sniffMPEGAudioContentType(header); contentType != "" {
			return contentType
		}
	}

	return detected
}

// sniffExtendedContentType identifies media formats by exact signatures that
// http.DetectContentType does not recognize, returning "" when none match.
func sniffExtendedContentType(header []byte, extension string) string {

	switch {

	// FLAC stream marker
	case bytes.HasPrefix(header, []byte("fLaC")):
		return "audio/flac"

	// AMR speech, wideband and narrowband (RFC 4867)
	case bytes.HasPrefix(header, []byte("#!AMR-WB")):
		return "audio/amr-wb"

	case bytes.HasPrefix(header, []byte("#!AMR")):
		return "audio/amr"

	// ASF container (WMA / WMV)
	case bytes.HasPrefix(header, asfHeaderGUID):
		return sniffASFContentType(extension)

	// EBML container (WebM / Matroska)
	case bytes.HasPrefix(header, ebmlMagic):
		return sniffEBMLContentType(extension)

	// Ogg container (Vorbis / Opus / Speex / Theora)
	case bytes.HasPrefix(header, []byte("OggS\x00")):
		return sniffOggContentType(header)

	// ISO-BMFF / MP4-family container (M4A / M4B / 3GP)
	case isISOBMFF(header):
		return sniffFtypContentType(header, extension)
	}

	// None of the above: let the standard library take the wheel.
	return ""
}

// sniffASFContentType names the flavor of a byte-confirmed ASF container.
func sniffASFContentType(extension string) string {

	// ASF holds audio (.wma) and video (.wmv) behind the same GUID, so the
	// extension picks the flavor and anything else is the generic container type.
	switch extension {

	case ".wma":
		return "audio/x-ms-wma"

	case ".wmv":
		return "video/x-ms-wmv"
	}

	return "video/x-ms-asf"
}

// sniffEBMLContentType names the flavor of a byte-confirmed EBML (WebM/Matroska) container.
func sniffEBMLContentType(extension string) string {

	// Audio-only WebM is indistinguishable from video without a full EBML parser,
	// so the extension picks the flavor and video/webm remains the default.
	if extension == ".weba" {
		return "audio/webm"
	}

	return "video/webm"
}

// sniffOggContentType names the flavor of an Ogg container by the codec tag of its first
// logical stream.
func sniffOggContentType(header []byte) string {

	// The generic container type, for pages whose codec cannot be identified.
	const genericOgg = "application/ogg"

	// The first page's payload begins after the 27-byte page header and the segment
	// table (one byte per segment).  A header too short to hold either is still Ogg.
	if len(header) < 28 {
		return genericOgg
	}

	payloadStart := 27 + int(header[26])

	if payloadStart >= len(header) {
		return genericOgg
	}

	// The payload of the first page opens with the codec's identification magic.
	switch payload := header[payloadStart:]; {

	case bytes.HasPrefix(payload, []byte("\x01vorbis")),
		bytes.HasPrefix(payload, []byte("OpusHead")),
		bytes.HasPrefix(payload, []byte("\x7fFLAC")),
		bytes.HasPrefix(payload, []byte("Speex   ")):
		return "audio/ogg"

	case bytes.HasPrefix(payload, []byte("\x80theora")):
		return "video/ogg"
	}

	// A codec we don't recognize: report the generic container type.
	return genericOgg
}

// isISOBMFF returns TRUE if a header opens an ISO-BMFF (MP4-family) container.
func isISOBMFF(header []byte) bool {

	if len(header) < 12 {
		return false
	}

	return bytes.Equal(header[4:8], []byte("ftyp"))
}

// sniffFtypContentType names the flavor of a byte-confirmed ISO-BMFF container by its
// "ftyp" brand, returning "" for brands the standard library already handles.
func sniffFtypContentType(header []byte, extension string) string {

	switch brand := string(header[8:12]); {

	// Apple's audio-only brands: AAC (.m4a), audiobook (.m4b), protected AAC (.m4p)
	case brand == "M4A ", brand == "M4B ", brand == "M4P ":
		return "audio/mp4"

	// Mobile 3GPP recordings: audio and video share the container, the extension
	// picks the flavor
	case strings.HasPrefix(brand, "3g"):
		if extension == ".3ga" {
			return "audio/3gpp"
		}
		return "video/3gpp"

	// Audio-only files are sometimes written with a generic brand ("isom", "mp42").
	// The bytes have confirmed the container, so the name may refine the flavor.
	case extension == ".m4a", extension == ".m4b":
		return "audio/mp4"
	}

	// Every other brand is video (or junk): the standard library's mp4 sniffer decides.
	return ""
}

// sniffMPEGAudioContentType identifies bare MPEG audio framing -- MP3 files without an
// ID3 tag, and raw AAC streams in ADTS framing -- returning "" when the bytes are not a
// plausible frame header.
func sniffMPEGAudioContentType(header []byte) string {

	if len(header) < 4 {
		return ""
	}

	// Every MPEG audio frame opens with an 11-bit sync run.
	if header[0] != 0xFF || header[1]&0xE0 != 0xE0 {
		return ""
	}

	// ADTS AAC is the only MPEG framing with layer bits 00, and its sync run is 12 bits.
	if layer := (header[1] >> 1) & 0x03; layer == 0 {

		if header[1]&0xF0 != 0xF0 {
			return ""
		}

		// RULE: Sampling frequency indexes 13-15 are reserved, so they mark a false sync.
		if (header[2]>>2)&0x0F >= 13 {
			return ""
		}

		return "audio/aac"
	}

	// RULE: Version 01 is reserved, so it marks a false sync.
	if version := (header[1] >> 3) & 0x03; version == 1 {
		return ""
	}

	// RULE: Bitrate index 1111 and sampling rate index 11 are invalid, so they mark a
	// false sync.
	if header[2]>>4 == 0x0F {
		return ""
	}

	if (header[2]>>2)&0x03 == 0x03 {
		return ""
	}

	// It walks like an MPEG frame and quacks like an MPEG frame.
	return "audio/mpeg"
}
