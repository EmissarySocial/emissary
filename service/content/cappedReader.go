package content

import (
	"io"

	"github.com/benpate/derp"
)

// cappedReader reads a response body, and FAILS once more than its limit has been read.
// RULE: It must fail rather than stop.  A reader that returned EOF at the limit would store a
// truncated file and report success.
type cappedReader struct {
	body      io.ReadCloser
	remaining int64
}

// newCappedReader returns a reader that fails once more than maxBytes have been read from body
func newCappedReader(body io.ReadCloser, maxBytes int64) *cappedReader {
	return &cappedReader{
		body:      body,
		remaining: maxBytes,
	}
}

// Read implements the io.Reader interface
func (reader *cappedReader) Read(buffer []byte) (int, error) {

	// Ask for one byte past the limit at most, which is enough to learn that the body is too large
	if int64(len(buffer)) > reader.remaining+1 {
		buffer = buffer[:reader.remaining+1]
	}

	count, err := reader.body.Read(buffer)
	reader.remaining -= int64(count)

	if reader.remaining < 0 {
		return count, derp.BadRequest("content.cappedReader.Read", "File is too large")
	}

	return count, err
}

// Close implements the io.Closer interface
func (reader *cappedReader) Close() error {
	return reader.body.Close()
}
