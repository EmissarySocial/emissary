package external

type Null struct{}

/******************************************
 * Adapter Methods
 ******************************************/

func (adapter Null) Install() {
}

func (adapter Null) PollStreams() {
}

func (adapter Null) PostStream() {
}
