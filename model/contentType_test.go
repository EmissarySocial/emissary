package model

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// oggPage assembles a minimal first Ogg page whose payload opens with a codec tag.
func oggPage(codecTag string) []byte {

	// 27-byte page header, one segment, then the codec identification payload.
	page := []byte("OggS\x00\x02\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00")
	page = append(page, 0x01, 0xFF)
	return append(page, []byte(codecTag)...)
}

// isoBMFF assembles a minimal ISO-BMFF "ftyp" box with the provided brands.
func isoBMFF(majorBrand string, compatible ...string) []byte {

	// Box size covers the size, "ftyp", major brand, minor version, and compatible brands.
	size := 16 + 4*len(compatible)
	box := []byte{0x00, 0x00, 0x00, byte(size)}
	box = append(box, []byte("ftyp")...)
	box = append(box, []byte(majorBrand)...)
	box = append(box, 0x00, 0x00, 0x02, 0x00)

	for _, brand := range compatible {
		box = append(box, []byte(brand)...)
	}

	return box
}

// TestDetectContentType confirms that types are read from bytes, and carry no parameters.
func TestDetectContentType(t *testing.T) {

	require.Equal(t, "image/png", DetectContentType([]byte("\x89PNG\r\n\x1a\n"), ""))
	require.Equal(t, "image/gif", DetectContentType([]byte("GIF89a<script>alert(1)</script>"), ""))
	require.Equal(t, "application/pdf", DetectContentType([]byte("%PDF-1.7\n"), ""))

	// The "; charset=utf-8" that http.DetectContentType appends is stripped.
	require.Equal(t, "text/html", DetectContentType([]byte("<!doctype html><html></html>"), ""))
	require.Equal(t, "text/plain", DetectContentType([]byte("just some words"), ""))

	// An empty file is opaque data, not an error.
	require.Equal(t, "text/plain", DetectContentType([]byte{}, ""))
}

// TestDetectContentType_Audio confirms every audio format the upload gate supports,
// including the ones http.DetectContentType cannot identify on its own.
func TestDetectContentType_Audio(t *testing.T) {

	detect := func(expected string, header []byte, filename string, reason string) {
		t.Helper()
		require.Equal(t, expected, DetectContentType(header, filename), reason)
	}

	// Formats with their own magic numbers -- no filename required.
	detect("audio/flac", append([]byte("fLaC"), 0x00, 0x00, 0x00, 0x22, 0x10, 0x00), "", "FLAC stream marker")
	detect("audio/mpeg", []byte("ID3\x04\x00\x00\x00\x00\x00\x00"), "", "MP3 with ID3v2 tag")
	detect("audio/mpeg", []byte{0xFF, 0xFB, 0x90, 0x64, 0x00, 0x0F}, "", "MP3 bare framesync")
	detect("audio/aac", []byte{0xFF, 0xF1, 0x50, 0x80, 0x00, 0x1F, 0xFC}, "", "raw ADTS AAC")
	detect("audio/wave", []byte("RIFF\x24\x08\x00\x00WAVEfmt "), "", "WAV")
	detect("audio/aiff", []byte("FORM\x00\x00\x08\x24AIFFCOMM"), "", "AIFF")
	detect("audio/amr", []byte("#!AMR\x0a\x3c\x48"), "", "AMR narrowband")
	detect("audio/amr-wb", []byte("#!AMR-WB\x0a\x3c\x48"), "", "AMR wideband")
	detect("audio/x-ms-wma", asfHeaderGUID, "song.wma", "WMA in ASF container")

	// MP4-family containers: the ftyp brand marks audio-only files.
	detect("audio/mp4", isoBMFF("M4A ", "M4A ", "mp42", "isom"), "", "M4A brand")
	detect("audio/mp4", isoBMFF("M4B ", "M4B ", "mp42"), "", "M4B audiobook brand")
	detect("audio/mp4", isoBMFF("M4P ", "M4P "), "", "M4P protected brand")
	detect("video/mp4", isoBMFF("isom", "isom", "mp41"), "", "generic MP4 is video")
	detect("audio/mp4", isoBMFF("isom", "isom", "mp41"), "song.m4a", "generic brand refined by .m4a")
	detect("audio/mp4", isoBMFF("isom", "isom", "mp41"), "book.M4B", "extension match is case-blind")
	detect("video/3gpp", isoBMFF("3gp4", "3gp4", "isom"), "", "3GP defaults to video")
	detect("audio/3gpp", isoBMFF("3gp4", "3gp4", "isom"), "memo.3ga", "3GP audio by .3ga")

	// Ogg containers: the first stream's codec tag picks audio or video.
	detect("audio/ogg", oggPage("\x01vorbis"), "", "Ogg Vorbis")
	detect("audio/ogg", oggPage("OpusHead"), "", "Ogg Opus")
	detect("audio/ogg", oggPage("\x7fFLAC"), "", "Ogg FLAC")
	detect("audio/ogg", oggPage("Speex   "), "", "Ogg Speex")
	detect("video/ogg", oggPage("\x80theora"), "", "Ogg Theora is video")
	detect("application/ogg", oggPage("mystery!"), "", "unknown Ogg codec stays generic")

	// EBML containers: filename picks the audio flavor, video is the default.
	detect("video/webm", ebmlMagic, "", "EBML defaults to video/webm")
	detect("audio/webm", ebmlMagic, "song.weba", "audio WebM by .weba")

	// ASF containers: filename picks the flavor, the generic container is the default.
	detect("video/x-ms-wmv", asfHeaderGUID, "movie.wmv", "WMV in ASF container")
	detect("video/x-ms-asf", asfHeaderGUID, "", "nameless ASF stays generic")
}

// TestDetectContentType_FilenameCannotPromote confirms the security contract: a filename
// can never talk arbitrary bytes into a media type.
func TestDetectContentType_FilenameCannotPromote(t *testing.T) {

	// The original stored-XSS shapes, now wearing audio costumes.
	require.Equal(t, "text/html", DetectContentType([]byte("<!doctype html><script>alert(1)</script>"), "song.flac"))
	require.Equal(t, "text/plain", DetectContentType([]byte("just some words"), "song.mp3"))
	require.Equal(t, "text/plain", DetectContentType([]byte{}, "song.m4a"))

	// The tie-break only refines flavors within one container: an audio name cannot
	// drag an EBML or ASF file into a different container's type.
	require.Equal(t, "video/webm", DetectContentType(ebmlMagic, "song.mp3"))
	require.Equal(t, "video/x-ms-asf", DetectContentType(asfHeaderGUID, "song.flac"))
}

// TestSniffMPEGAudioContentType confirms that the framesync heuristic rejects the
// reserved bit-patterns that mark false syncs in random binary data.
func TestSniffMPEGAudioContentType(t *testing.T) {

	// Too short to hold a frame header.
	require.Equal(t, "", sniffMPEGAudioContentType([]byte{0xFF, 0xFB}))

	// No sync run at all.
	require.Equal(t, "", sniffMPEGAudioContentType([]byte{0xFF, 0xD8, 0xFF, 0xE0}))
	require.Equal(t, "", sniffMPEGAudioContentType([]byte{0x00, 0xFB, 0x90, 0x64}))

	// MP3 with reserved version bits (FF EB = version 01).
	require.Equal(t, "", sniffMPEGAudioContentType([]byte{0xFF, 0xEB, 0x90, 0x64}))

	// MP3 with invalid bitrate index (1111) or sampling rate index (11).
	require.Equal(t, "", sniffMPEGAudioContentType([]byte{0xFF, 0xFB, 0xF0, 0x64}))
	require.Equal(t, "", sniffMPEGAudioContentType([]byte{0xFF, 0xFB, 0x9C, 0x64}))

	// ADTS with an 11-bit sync run (0xE0 nibble) is not ADTS, and layer 00 makes it
	// nothing else either.
	require.Equal(t, "", sniffMPEGAudioContentType([]byte{0xFF, 0xE1, 0x50, 0x80}))

	// ADTS with a reserved sampling frequency index (13+).
	require.Equal(t, "", sniffMPEGAudioContentType([]byte{0xFF, 0xF1, 0x74, 0x80}))

	// The real things still pass.
	require.Equal(t, "audio/mpeg", sniffMPEGAudioContentType([]byte{0xFF, 0xFB, 0x90, 0x64}))
	require.Equal(t, "audio/aac", sniffMPEGAudioContentType([]byte{0xFF, 0xF9, 0x50, 0x80}))
}

// TestSniffOggContentType confirms that truncated and hostile Ogg pages degrade to the
// generic container type instead of panicking.
func TestSniffOggContentType(t *testing.T) {

	// Too short to hold a page header.
	require.Equal(t, "application/ogg", sniffOggContentType([]byte("OggS\x00")))

	// A segment table that claims more bytes than the buffer holds.
	hostile := oggPage("\x01vorbis")
	hostile[26] = 0xFF
	require.Equal(t, "application/ogg", sniffOggContentType(hostile))
}

// FuzzDetectContentType hunts for panics and malformed results across arbitrary bytes
// and filenames.
func FuzzDetectContentType(f *testing.F) {

	// Seed with every interesting shape from the unit tests.
	f.Add([]byte("fLaC\x00\x00\x00\x22"), "song.flac")
	f.Add([]byte("ID3\x04\x00\x00\x00"), "")
	f.Add([]byte{0xFF, 0xFB, 0x90, 0x64}, "")
	f.Add([]byte{0xFF, 0xF1, 0x50, 0x80}, "raw.aac")
	f.Add(oggPage("OpusHead"), "")
	f.Add(oggPage("\x80theora"), "movie.ogv")
	f.Add(isoBMFF("M4A ", "M4A ", "mp42"), "")
	f.Add(isoBMFF("isom", "isom"), "song.m4a")
	f.Add(asfHeaderGUID, "song.wma")
	f.Add(ebmlMagic, "song.weba")
	f.Add([]byte("#!AMR\x0a"), "")
	f.Add([]byte("<!doctype html>"), "song.mp3")
	f.Add([]byte{}, "")

	f.Fuzz(func(t *testing.T, header []byte, filename string) {

		contentType := DetectContentType(header, filename)

		// Every input maps to a bare "type/subtype" media type.
		require.NotEmpty(t, contentType, "header=%q filename=%q", header, filename)
		require.Contains(t, contentType, "/", "header=%q filename=%q", header, filename)
		require.NotContains(t, contentType, ";", "header=%q filename=%q", header, filename)
	})
}
