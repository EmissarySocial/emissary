package consumer

// contentTypeMaxLength bounds the remote-supplied media type quoted into a status message.
// Every real one fits easily; this only stops a hostile or broken header from filling the field.
const contentTypeMaxLength = 100

// statusMessageMaxLength matches the "statusMessage" schema in model.Following, which rejects
// anything longer.  A root message quoted from a transport failure has no length bound of its own.
const statusMessageMaxLength = 1024

// pollOutcome names what a failed poll means for the Following record it happened to
type pollOutcome int

const (
	// pollOutcomeRateLimited means the HOST is throttling us, and this record did nothing wrong
	pollOutcomeRateLimited pollOutcome = iota

	// pollOutcomeGone means the source is gone for good, on the remote server's own say-so
	pollOutcomeGone

	// pollOutcomeFailed means "record it and try again at the normal cadence"
	pollOutcomeFailed
)
