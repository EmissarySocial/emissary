package model

import "io"

// Pipeline interface represents a series of render steps that are performed
// when a user GETs or POSTs to a Template/Stream.
type Pipeline interface {
	Get(buffer io.Writer) error
	Post(buffer io.Writer) error
}
