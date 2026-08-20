package groupie

// Groupie tracks the most recent value in a sequence, so that templates can insert group headers
type Groupie struct {
	lastValue any
}

// New returns a fully initialized Groupie
func New() *Groupie {
	return &Groupie{}
}

// Header returns TRUE when the provided value differs from the previous one, meaning a new group has begun
func (g *Groupie) Header(value any) bool {

	if g.lastValue == value {
		return false
	}

	g.lastValue = value
	return true
}
