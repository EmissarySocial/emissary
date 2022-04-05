package render

import "io"

// BaseStep implements the empty/nil interface for a render step
type BaseStep struct{}

// Get is executed for GET requests.  This specific method is
// inherited from BaseStep, and performs no action
func (step BaseStep) Get(_ Factory, _ Renderer, _ io.Writer) error {
	return nil
}

// Post is executed for POST requests.  This specific method is
// inherited from BaseStep, and performs no action
func (step BaseStep) Post(_ Factory, _ Renderer, _ io.Writer) error {
	return nil
}

// IsWrapped returns TRUE if this action can be wrapped by the
// global site headers and footers.  This specific method is
// inherited from BaseStep, and always returns the default (TRUE).
func (step BaseStep) isWrapped() bool {
	return true
}
