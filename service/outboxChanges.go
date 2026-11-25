package service

// OutboxChanges carries what a save knew and the outbox task cannot read from the stored Stream:
// the syndication targets that the save added and removed, and a request to send every target again
type OutboxChanges struct {
	SyndicationAdded   []string // Targets added since the Stream was loaded
	SyndicationDeleted []string // Targets removed since the Stream was loaded
	Republish          bool     // TRUE if every target is sent an Update, changed or not
}

// IsZero returns TRUE if the save had nothing to pass on
func (changes OutboxChanges) IsZero() bool {
	return (len(changes.SyndicationAdded) == 0) && (len(changes.SyndicationDeleted) == 0) && !changes.Republish
}
