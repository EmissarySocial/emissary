package server

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCertificateHosts_FollowsAddedDomains is the regression test for the bug this type exists to
// fix. A non-local domain added at runtime used to be invisible to autocert forever, because the
// host whitelist was built from a boot-time snapshot -- so the domain answered over HTTP and
// failed over HTTPS until someone restarted the process.
func TestCertificateHosts_FollowsAddedDomains(t *testing.T) {

	stub := newStubConfigProvider("first.example.com")
	certificates := NewCertificateHosts(stub)

	require.Nil(t, certificates.HostPolicy(context.Background(), "first.example.com"))
	require.Error(t, certificates.HostPolicy(context.Background(), "second.example.com"))

	// Add a domain, exactly as the setup console would
	stub.set("first.example.com", "second.example.com")

	require.Nil(t, certificates.HostPolicy(context.Background(), "first.example.com"))
	require.Nil(t, certificates.HostPolicy(context.Background(), "second.example.com"), "a domain added at runtime must be able to get a certificate")
}

// TestCertificateHosts_FollowsRemovedDomains pins the other direction: a domain removed from the
// configuration must stop being certifiable, so a decommissioned hostname cannot keep renewing.
func TestCertificateHosts_FollowsRemovedDomains(t *testing.T) {

	stub := newStubConfigProvider("first.example.com", "second.example.com")
	certificates := NewCertificateHosts(stub)

	require.Nil(t, certificates.HostPolicy(context.Background(), "second.example.com"))

	stub.set("first.example.com")

	require.Nil(t, certificates.HostPolicy(context.Background(), "first.example.com"))
	require.Error(t, certificates.HostPolicy(context.Background(), "second.example.com"))
}

// TestCertificateHosts_RejectsLocalHostnames pins the filter. A public CA cannot validate a local
// name, and asking it to counts against the account's failed-validation rate limit.
func TestCertificateHosts_RejectsLocalHostnames(t *testing.T) {

	stub := newStubConfigProvider("localhost", "emissary.local", "127.0.0.1", "real.example.com")
	certificates := NewCertificateHosts(stub)

	require.Nil(t, certificates.HostPolicy(context.Background(), "real.example.com"))

	for _, hostname := range []string{"localhost", "emissary.local", "127.0.0.1"} {
		require.Error(t, certificates.HostPolicy(context.Background(), hostname), "local hostname must never reach Let's Encrypt: %s", hostname)
	}
}

// TestCertificateHosts_RejectsEverythingWhenEmpty pins the empty case: no configured domains means
// no certificates, not a wide-open policy. autocert's own default policy allows ANY host, so
// falling back to it would turn this server into a certificate-issuing oracle.
func TestCertificateHosts_RejectsEverythingWhenEmpty(t *testing.T) {

	stub := newStubConfigProvider()
	certificates := NewCertificateHosts(stub)

	require.Error(t, certificates.HostPolicy(context.Background(), "anything.example.com"))
}

// TestCertificateHosts_NormalizesInternationalizedNames pins the reason this delegates to
// autocert.HostWhitelist instead of comparing strings. GetCertificate converts the SNI name to
// Punycode before consulting the policy, so a Unicode hostname in the configuration only matches
// if it is converted the same way.
func TestCertificateHosts_NormalizesInternationalizedNames(t *testing.T) {

	stub := newStubConfigProvider("bücher.example.com")
	certificates := NewCertificateHosts(stub)

	// What autocert actually passes to the policy, after idna.Lookup.ToASCII
	require.Nil(t, certificates.HostPolicy(context.Background(), "xn--bcher-kva.example.com"))
}

// TestCertificateHosts_MemoizesUntilTheDomainsChange pins the cache. The policy runs on every TLS
// handshake and rebuilding it costs an IDNA conversion per configured domain, so an unchanged
// configuration must reuse the previous policy rather than building a fresh one each time.
//
// Go cannot compare two functions, so the memo is probed with a sentinel: replace the stored
// policy with one that is recognizable, then see whether it survives.
func TestCertificateHosts_MemoizesUntilTheDomainsChange(t *testing.T) {

	stub := newStubConfigProvider("first.example.com")
	certificates := NewCertificateHosts(stub)

	// Prime the memo
	certificates.currentPolicy()
	require.Equal(t, []string{"first.example.com"}, certificates.hosts)

	// Swap in a sentinel that no rebuild would ever produce
	sentinel := errors.New("sentinel policy")
	certificates.policy = func(context.Context, string) error { return sentinel }

	// Nothing changed, so the sentinel must survive -- proving no rebuild happened
	require.Equal(t, sentinel, certificates.currentPolicy()(context.Background(), "first.example.com"))

	// Change the configuration, and the memo must be discarded
	stub.set("first.example.com", "second.example.com")

	require.Nil(t, certificates.currentPolicy()(context.Background(), "second.example.com"), "a changed domain list must rebuild the policy")
	require.Equal(t, []string{"first.example.com", "second.example.com"}, certificates.hosts)
}

// TestCertificateHosts_ConcurrentHandshakesAndReloads runs the policy the way TLS does -- from
// many goroutines at once -- while the configuration is being replaced. Run with -race.
func TestCertificateHosts_ConcurrentHandshakesAndReloads(t *testing.T) {

	stub := newStubConfigProvider("first.example.com")
	certificates := NewCertificateHosts(stub)

	var waitGroup sync.WaitGroup

	waitGroup.Add(1)
	go func() {
		defer waitGroup.Done()
		for index := range 200 {
			if index%2 == 0 {
				stub.set("first.example.com")
			} else {
				stub.set("first.example.com", "second.example.com")
			}
		}
	}()

	for range 8 {
		waitGroup.Add(1)

		go func() {
			defer waitGroup.Done()
			for range 200 {
				// assert, not require: require fails via runtime.Goexit, which is only legal on
				// the test goroutine.  first.example.com is present in BOTH configurations, so
				// it must always pass.
				assert.Nil(t, certificates.HostPolicy(context.Background(), "first.example.com"))
			}
		}()
	}

	waitGroup.Wait()
}
