package server

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/acme/autocert"
)

// TestCertificateCache_RoundTrip covers the ordinary path against a real directory.
func TestCertificateCache_RoundTrip(t *testing.T) {

	folder := t.TempDir()
	stub := newStubConfigProvider("first.example.com")
	stub.setCertificateFolder(folder)

	cache := NewCertificateCache(stub)
	ctx := context.Background()

	require.Nil(t, cache.Put(ctx, "example-cert", []byte("PEM")))

	result, err := cache.Get(ctx, "example-cert")
	require.Nil(t, err)
	require.Equal(t, []byte("PEM"), result)

	// The bytes really landed in the configured folder
	stored, err := os.ReadFile(filepath.Join(folder, "example-cert"))
	require.Nil(t, err)
	require.Equal(t, []byte("PEM"), stored)

	require.Nil(t, cache.Delete(ctx, "example-cert"))
	_, err = cache.Get(ctx, "example-cert")
	require.Equal(t, autocert.ErrCacheMiss, err)
}

// TestCertificateCache_ReturnsErrCacheMissUnwrapped is the subtle one. autocert compares the error
// from Cache.Get against ErrCacheMiss BY IDENTITY to decide whether to request a new certificate.
// A wrapped error would read as a hard failure and stop issuance permanently.
func TestCertificateCache_ReturnsErrCacheMissUnwrapped(t *testing.T) {

	stub := newStubConfigProvider("first.example.com")
	stub.setCertificateFolder(t.TempDir())

	_, err := NewCertificateCache(stub).Get(context.Background(), "never-written")

	require.Equal(t, autocert.ErrCacheMiss, err, "autocert compares this error by identity")
}

// TestCertificateCache_FollowsAChangedFolder pins the point of the wrapper: the directory is
// resolved on every call, so moving the certificate folder takes effect without a restart.
func TestCertificateCache_FollowsAChangedFolder(t *testing.T) {

	first := t.TempDir()
	second := t.TempDir()

	stub := newStubConfigProvider("first.example.com")
	stub.setCertificateFolder(first)

	cache := NewCertificateCache(stub)
	ctx := context.Background()

	require.Nil(t, cache.Put(ctx, "example-cert", []byte("FIRST")))

	// Move the certificate folder, the way an operator would in the setup console
	stub.setCertificateFolder(second)

	// The old folder's contents are no longer visible...
	_, err := cache.Get(ctx, "example-cert")
	require.Equal(t, autocert.ErrCacheMiss, err)

	// ...and new writes land in the new folder
	require.Nil(t, cache.Put(ctx, "example-cert", []byte("SECOND")))

	stored, err := os.ReadFile(filepath.Join(second, "example-cert"))
	require.Nil(t, err)
	require.Equal(t, []byte("SECOND"), stored)
}

// TestCertificateCache_UnconfiguredFolderIsAnError pins the deliberate refusal. Issuing
// certificates that cannot be persisted would re-request them on every restart and burn the Let's
// Encrypt duplicate-certificate rate limit, so an unusable folder must fail the handshake rather
// than fall back to the working directory.
func TestCertificateCache_UnconfiguredFolderIsAnError(t *testing.T) {

	stub := newStubConfigProvider("first.example.com")
	stub.setCertificateFolder("")

	cache := NewCertificateCache(stub)
	ctx := context.Background()

	_, err := cache.Get(ctx, "example-cert")
	require.Error(t, err)
	require.NotEqual(t, autocert.ErrCacheMiss, err, "an unconfigured folder must NOT read as a cache miss, which would trigger issuance")

	require.Error(t, cache.Put(ctx, "example-cert", []byte("PEM")))
	require.Error(t, cache.Delete(ctx, "example-cert"))
}
