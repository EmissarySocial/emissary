package server

import (
	"context"
	"slices"
	"sync"

	"github.com/benpate/rosetta/slice"
	"github.com/benpate/uri"
	"golang.org/x/crypto/acme/autocert"
)

// CertificateHosts decides which hostnames may be issued a Let's Encrypt certificate, reading the
// CURRENT server configuration rather than a snapshot taken when the HTTPS server started.
//
// autocert.Manager is built ONCE and lives for the whole process, but it consults HostPolicy on
// every TLS handshake.  So a policy reached through this type follows the configuration, while the
// autocert.HostWhitelist it replaces was frozen at boot -- which meant a non-local domain added
// through the setup console could not obtain a certificate until the process restarted.  See the
// package README for the whole story.
type CertificateHosts struct {
	factory ConfigProvider

	// The policy is memoized against the domain list that produced it.  HostPolicy runs on every
	// TLS handshake, and rebuilding it means an IDNA conversion per configured domain, so the
	// common case (nothing changed) must not pay for the rare one.
	lock   sync.Mutex
	hosts  []string
	policy autocert.HostPolicy
}

// NewCertificateHosts returns a host policy bound to the server factory's live configuration.
func NewCertificateHosts(factory ConfigProvider) *CertificateHosts {
	return &CertificateHosts{factory: factory}
}

// HostPolicy reports whether `host` is a domain this server is willing to request a certificate
// for.  It satisfies autocert.HostPolicy.
func (certificates *CertificateHosts) HostPolicy(ctx context.Context, host string) error {
	return certificates.currentPolicy()(ctx, host)
}

// currentPolicy returns a policy matching the domains in the configuration right now, rebuilding
// it only when that list has actually changed.
func (certificates *CertificateHosts) currentPolicy() autocert.HostPolicy {

	// RULE: Local hostnames are filtered out and can NEVER reach Let's Encrypt.  A public CA
	// cannot validate "localhost", and asking it to counts against the failed-validation rate
	// limit for the account.
	hosts := slice.Filter(certificates.factory.Config().DomainNames(), uri.NotLocalHostname)

	certificates.lock.Lock()
	defer certificates.lock.Unlock()

	if certificates.policy != nil && slices.Equal(hosts, certificates.hosts) {
		return certificates.policy
	}

	// RULE: Delegate to autocert.HostWhitelist rather than comparing hostnames here.  It applies
	// the same IDNA normalization that GetCertificate applies to the incoming SNI name, so an
	// internationalized domain stored in Unicode still matches the Punycode name on the wire.
	// Hand-rolling that comparison is how IDN domains silently stop getting certificates.
	certificates.hosts = hosts
	certificates.policy = autocert.HostWhitelist(hosts...)

	return certificates.policy
}
