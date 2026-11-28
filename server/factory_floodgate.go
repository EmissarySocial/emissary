package server

import (
	"context"
	"net/url"
	"sync"
	"time"

	floodgatemongo "github.com/EmissarySocial/emissary/tools/floodgate-mongo"
	"github.com/benpate/derp"
	"github.com/benpate/floodgate"
	"github.com/benpate/uri"
)

// blockedHostsMaxAge is how long one server trusts its copy of the blocked hosts before reloading it
const blockedHostsMaxAge = time.Minute

// blockedHostsTimeout bounds the database calls that a request waits for on behalf of Floodgate
const blockedHostsTimeout = 5 * time.Second

// blockedHostCache is one server's short-lived copy of the hosts that Floodgate has blocked.
// It only saves database reads: the records in the common database are what every server obeys.
type blockedHostCache struct {
	mutex   sync.Mutex
	hosts   floodgate.Blocklist
	expires time.Time
}

// Floodgate returns the circuit breaker for outbound deliveries, which every domain on this server shares
func (factory *factoryCore) Floodgate() floodgate.Gate {

	// RULE: The store reads the common database through its getter on every call, because a
	// configuration reload can replace the connection.
	return floodgate.New(floodgatemongo.New(factory.CommonDatabase))
}

// ReviveHost tells Floodgate that the host of the provided Actor URL has just proven it is
// running, by sending this server a request that passed validation
func (factory *factoryCore) ReviveHost(actorURL string) {

	const location = "server.factoryCore.ReviveHost"

	host := hostOf(actorURL)

	if host == "" {
		return
	}

	// RULE: Nearly every inbound request comes from a healthy host, so check one server's cached
	// copy before touching the database
	if !factory.isBlockedHost(host) {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), blockedHostsTimeout)
	defer cancel()

	if err := factory.Floodgate().Revive(ctx, host); err != nil {
		derp.Report(derp.Wrap(err, location, "Reviving host", "host: "+host))
		return
	}

	// Stop treating the host as blocked at once, without making the next request reload the copy
	factory.blockedHosts.remove(host)
}

// isBlockedHost returns TRUE if this server's copy of the blocked hosts includes the host,
// reloading the copy when it is older than blockedHostsMaxAge
func (factory *factoryCore) isBlockedHost(host string) bool {

	const location = "server.factoryCore.isBlockedHost"

	isBlocked, err := factory.blockedHosts.contains(host, factory.Floodgate().Blocked)

	if err != nil {
		derp.Report(derp.Wrap(err, location, "Loading blocked hosts"))
	}

	return isBlocked
}

// contains returns TRUE if the copy includes the host. The one caller that finds the copy expired
// reloads it, and a failed reload returns its error with the answer from the old copy.
func (cache *blockedHostCache) contains(host string, load func(context.Context) (floodgate.Blocklist, error)) (bool, error) {

	// RULE: Push the expiry forward before unlocking. That lets one caller reload while the rest
	// answer from the old copy, and a failed reload is not tried again for a full interval.
	cache.mutex.Lock()
	isBlocked := cache.hosts.Contains(host)
	mustReload := !time.Now().Before(cache.expires)

	if mustReload {
		cache.expires = time.Now().Add(blockedHostsMaxAge)
	}

	cache.mutex.Unlock()

	if !mustReload {
		return isBlocked, nil
	}

	// RULE: The database is read with the mutex unlocked, so no other request waits on it (BUG-381)
	ctx, cancel := context.WithTimeout(context.Background(), blockedHostsTimeout)
	defer cancel()

	hosts, err := load(ctx)

	if err != nil {
		return isBlocked, err
	}

	cache.mutex.Lock()
	cache.hosts = hosts
	cache.mutex.Unlock()

	return hosts.Contains(host), nil
}

// remove takes one host out of the copy and leaves its expiry alone
func (cache *blockedHostCache) remove(host string) {

	cache.mutex.Lock()
	defer cache.mutex.Unlock()

	delete(cache.hosts, host)
}

// hostOf returns the hostname of a URL as Floodgate keys it: lower case, with no port, and
// empty for anything that is not a URL with a host
func hostOf(value string) string {

	parsed, err := url.Parse(value)

	if err != nil {
		return ""
	}

	if parsed.Host == "" {
		return ""
	}

	// RULE: hannibal's delivery pipeline and Floodgate's remote option name a host with this
	// same function. A different key here would revive a record that no delivery reads.
	return uri.Hostname(value)
}
