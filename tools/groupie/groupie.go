package groupie

type Groupie struct {
	lastValue any
}

func New() *Groupie {
	return &Groupie{}
}

func (g *Groupie) Header(value any) bool {

	if g.lastValue == value {
		return false
	}

	g.lastValue = value
	return true
}
