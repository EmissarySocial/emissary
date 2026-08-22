package server

import (
	"context"

	"github.com/benpate/derp"
	"golang.org/x/crypto/acme/autocert"
)

// CertificateCache stores Let's Encrypt certificates (and the ACME account key) in the directory
// named by the CURRENT server configuration.  It satisfies autocert.Cache.
//
// autocert.Manager is built ONCE and lives for the whole process, but it consults Cache on every
// certificate read or write.  So a directory resolved through this type follows the
// configuration, while the autocert.DirCache it replaces was frozen at boot.  See the package
// README for the whole story.
type CertificateCache struct {
	factory ConfigProvider
}

// NewCertificateCache returns a certificate cache bound to the server factory's live configuration.
func NewCertificateCache(factory ConfigProvider) CertificateCache {
	return CertificateCache{factory: factory}
}

// Get retrieves a certificate from the configured directory.
func (cache CertificateCache) Get(ctx context.Context, name string) ([]byte, error) {

	directory, err := cache.directory()

	if err != nil {
		return nil, err
	}

	// RULE: Returned UNWRAPPED.  autocert branches on `err == ErrCacheMiss` by identity to decide
	// whether to request a new certificate; a wrapped error would look like a hard failure and
	// stop issuance forever.
	return directory.Get(ctx, name)
}

// Put stores a certificate in the configured directory.
func (cache CertificateCache) Put(ctx context.Context, name string, data []byte) error {

	directory, err := cache.directory()

	if err != nil {
		return err
	}

	return directory.Put(ctx, name, data)
}

// Delete removes a certificate from the configured directory.
func (cache CertificateCache) Delete(ctx context.Context, name string) error {

	directory, err := cache.directory()

	if err != nil {
		return err
	}

	return directory.Delete(ctx, name)
}

// directory resolves the certificate directory from the configuration, on every call.
//
// RULE: An unconfigured location is an ERROR, not a fallback.  autocert aborts the handshake on
// any cache error other than ErrCacheMiss, which is what we want: issuing certificates we cannot
// persist would re-request them on every restart and burn the Let's Encrypt rate limit (5
// duplicate certificates per week) for a problem the operator never sees.  Live mode already
// requires a certificates folder -- IsReadyForDomains refuses to start without one -- so reaching
// this is a sign the configuration changed to something unusable while the server was running.
func (cache CertificateCache) directory() (autocert.DirCache, error) {

	const location = "server.CertificateCache.directory"

	folder := cache.factory.Config().Certificates.GetString("location")

	if folder == "" {
		return "", derp.Internal(location, "No certificate folder is configured. HTTPS certificates cannot be stored or renewed until `certificates.location` is set.")
	}

	return autocert.DirCache(folder), nil
}
