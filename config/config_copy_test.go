package config

import (
	"testing"

	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/sliceof"
	"github.com/stretchr/testify/require"
)

// TestConfig_CopyIsIndependent is the point of Copy: every reference type must be disconnected
// from the original.  A plain struct assignment shares the backing storage of every map and
// slice, so a caller editing its "copy" reaches through into the running server's live
// configuration -- with no lock, while the reload goroutine and every request are reading it.
func TestConfig_CopyIsIndependent(t *testing.T) {

	original := DefaultConfig()
	original.AdminEmail = "original@example.com"
	original.ActivityPubCache = mapof.String{"connectString": "mongodb://original", "database": "original"}
	original.Domains.Put(Domain{DomainID: "1", Hostname: "original.example.com"})
	original.Templates = sliceof.Object[mapof.String]{{"adapter": "EMBED", "location": "original"}}
	original.Loggers = sliceof.Object[mapof.Any]{{"type": "console", "level": "original"}}

	result := mutateInPlace(original.Copy())

	// Nothing the caller did to its copy may be visible in the original
	require.Equal(t, "original@example.com", original.AdminEmail)
	require.Equal(t, "mongodb://original", original.ActivityPubCache.GetString("connectString"))
	require.Equal(t, DefaultConfig().AttachmentCache.GetString("location"), original.AttachmentCache.GetString("location"))
	require.Equal(t, 1, len(original.Domains))
	require.Equal(t, "original.example.com", original.Domains[0].Hostname)
	require.Equal(t, "original", original.Templates[0].GetString("location"))
	require.Equal(t, "original", original.Loggers[0].GetString("level"))

	// ...and the copy really did change
	require.Equal(t, "mutated@example.com", result.AdminEmail)
	require.Equal(t, "mongodb://mutated", result.ActivityPubCache.GetString("connectString"))
	require.Equal(t, "mutated", result.Templates[0].GetString("location"))
	require.Equal(t, "mutated", result.Loggers[0].GetString("level"))
}

// mutateInPlace edits a Config the way the setup console's form handlers do -- in place, through
// the maps and slices it was handed.
func mutateInPlace(config Config) Config {

	config.AdminEmail = "mutated@example.com"
	config.ActivityPubCache["connectString"] = "mongodb://mutated"
	config.AttachmentCache["location"] = "mutated"
	config.Domains[0].Hostname = "mutated.example.com"
	config.Domains.Put(Domain{DomainID: "2", Hostname: "second.example.com"})
	config.Templates[0]["location"] = "mutated"
	config.Loggers[0]["level"] = "mutated"

	return config
}

// TestConfig_CopyPreservesValues pins that Copy is a copy and not a reset: every field survives.
func TestConfig_CopyPreservesValues(t *testing.T) {

	original := DefaultConfig()
	original.AdminEmail = "admin@example.com"
	original.HTTPPort = 8443
	original.HTTPSPort = 4443
	original.DebugLevel = "Trace"
	original.LogSlowQueries = 250
	original.ClientIPStrategy = "RIGHTMOST-TRUSTED-COUNT"
	original.ClientIPTrustedCount = 2
	original.ClientIPHeader = "X-Real-IP"
	original.TrustForwardedHost = true
	original.AllowPrivateIPs = true
	original.Source = ConfigSourceCommandLine
	original.Location = "mongodb://localhost:27017/emissary"
	original.Domains.Put(Domain{DomainID: "1", Hostname: "one.example.com"})

	require.Equal(t, original, original.Copy())
}

// TestConfig_CopyKeepsNilsNil pins that Copy does not invent collections.  A nil map that becomes
// an empty map would make "was this configured?" checks answer differently after a copy.
func TestConfig_CopyKeepsNilsNil(t *testing.T) {

	result := Config{}.Copy()

	require.Nil(t, result.Domains)
	require.Nil(t, result.Templates)
	require.Nil(t, result.AttachmentOriginals)
	require.Nil(t, result.AttachmentCache)
	require.Nil(t, result.ExportCache)
	require.Nil(t, result.Certificates)
	require.Nil(t, result.ActivityPubCache)
	require.Nil(t, result.Loggers)
}

// TestConfig_CopyOfEmptyCollectionsStaysEmpty pins the other side: a NewConfig() round-trips with
// its (non-nil, empty) collections intact.
func TestConfig_CopyOfEmptyCollectionsStaysEmpty(t *testing.T) {

	result := NewConfig().Copy()

	require.NotNil(t, result.Domains)
	require.NotNil(t, result.Templates)
	require.NotNil(t, result.AttachmentOriginals)
	require.NotNil(t, result.AttachmentCache)
	require.NotNil(t, result.ExportCache)
	require.NotNil(t, result.Certificates)
	require.NotNil(t, result.ActivityPubCache)
	require.NotNil(t, result.Loggers)

	require.Zero(t, len(result.Domains))
	require.Zero(t, len(result.Templates))
	require.Zero(t, len(result.Loggers))
}
