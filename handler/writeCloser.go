package handler

import "io"

// AsWriteCloser is a hacky kludge that lets us use an io.Writer as an io.WriteCloser
type AsWriteCloser struct {
	io.Writer
}

func (writeCloser AsWriteCloser) Close() error {
	// no op
	return nil
}
