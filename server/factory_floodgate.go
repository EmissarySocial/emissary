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

	// Drop the cached copy, so that this server stops treating the host as blocked at once
	factory.blockedHosts.mutex.Lock()
	factory.blockedHosts.expires = time.Time{}
	factory.blockedHosts.mutex.Unlock()
}

// isBlockedHost returns TRUE if this server's copy of the blocked hosts includes the host,
// reloading the copy when it is older than blockedHostsMaxAge
func (factory *factoryCore) isBlockedHost(host string) bool {

	const location = "server.factoryCore.isBlockedHost"

	cache := &factory.blockedHosts

	cache.mutex.Lock()
	defer cache.mutex.Unlock()

	if time.Now().Before(cache.expires) {
		return cache.hosts.Contains(host)
	}

	ctx, cancel := context.WithTimeout(context.Background(), blockedHostsTimeout)
	defer cancel()

	hosts, err := factory.Floodgate().Blocked(ctx)

	// RULE: A failed reload keeps the old copy for another interval, so that a database outage
	// does not put one failing query in front of every inbound request
	cache.expires = time.Now().Add(blockedHostsMaxAge)

	if err != nil {
		derp.Report(derp.Wrap(err, location, "Loading blocked hosts"))
		return cache.hosts.Contains(host)
	}

	cache.hosts = hosts
	return cache.hosts.Contains(host)
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
