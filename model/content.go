package model

// Content represents a piece of page content that can be stored in the system.
type Content interface {
	HTML() string
}
