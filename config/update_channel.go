package config

// updateChannel carries configurations from a storage engine to the server factory that applies
// them. It holds a single slot, and the newest entry always wins.
type updateChannel chan Config

// newUpdateChannel returns a channel that a storage engine can publish to without ever blocking.
func newUpdateChannel() updateChannel {
	return make(updateChannel, 1)
}

// subscribe returns the receive end of the channel, for the factory that applies configurations.
func (channel updateChannel) subscribe() <-chan Config {
	return channel
}

// notify publishes a configuration WITHOUT ever blocking the caller.
//
// RULE: A storage engine's watcher goroutine must never park on this send. It is the only thing
// polling for changes, so a parked watcher stops watching -- and the work on the receiving end is
// slow (a database reconnect, an index sync, a rebuild of every domain factory), so the stall is
// long. For the MongoDB change stream that stall is fatal: it can push the resume token out of
// the oplog window, which kills the watcher outright and freezes that node's configuration until
// someone reboots it. (FACTORY-MODES D8 calls for the same fix on the file watcher.)
//
// A configuration that is still queued is discarded rather than delivered. Configuration is full
// state, so only the newest version is worth applying -- and an older one would be actively wrong,
// since it would land after the newer one.
//
// The loop terminates because a storage engine has exactly ONE publisher: once the slot is drained
// (by us, or by the consumer racing us) nothing else can refill it, so the next send succeeds.
func (channel updateChannel) notify(config Config) {

	for {

		select {
		case channel <- config:
			return
		default:
		}

		// The slot is full. Discard the superseded configuration and try again.
		select {
		case <-channel:
		default:
		}
	}
}
