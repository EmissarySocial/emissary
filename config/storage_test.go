package config

import (
	"testing"

	"github.com/EmissarySocial/emissary/tools/secretcheck"
	"github.com/stretchr/testify/require"
)

// storageTestPassword and storageTestKeyPassword stand in for the credentials a config location can carry
const (
	storageTestPassword    = "s3cretUserPass"
	storageTestKeyPassword = "s3cretKeyPass"
)

// TestLoad_UnknownSchemeOmitsCredentials confirms an unsupported location is reported without its password or query
func TestLoad_UnknownSchemeOmitsCredentials(t *testing.T) {

	args := CommandLineArgs{Location: "mongdb://admin:" + storageTestPassword + "@db.example:27017/emissary?tlsCertificateKeyFilePassword=" + storageTestKeyPassword}

	_, err := Load(&args)
	require.Error(t, err)

	secretcheck.RequireAbsent(t, err, storageTestPassword)
	secretcheck.RequireAbsent(t, err, storageTestKeyPassword)
}

// TestNewMongoStorage_ConnectErrorOmitsCredentials confirms a connection string the driver rejects is
// reported without its password or query, in every layer of the error chain
func TestNewMongoStorage_ConnectErrorOmitsCredentials(t *testing.T) {

	args := CommandLineArgs{Location: "mongodb://admin:" + storageTestPassword + "@db.example:notaport/emissary?tlsCertificateKeyFilePassword=" + storageTestKeyPassword}

	_, err := NewMongoStorage(&args)
	require.Error(t, err)

	secretcheck.RequireAbsent(t, err, storageTestPassword)
	secretcheck.RequireAbsent(t, err, storageTestKeyPassword)
}

// TestLocationLabel confirms a location is named by its scheme and host alone
func TestLocationLabel(t *testing.T) {

	require.Equal(t, "mongodb://db.example:27017", locationLabel("mongodb://admin:"+storageTestPassword+"@db.example:27017/emissary?tlsCertificateKeyFilePassword="+storageTestKeyPassword))
	require.Equal(t, "mongodb+srv://cluster.example", locationLabel("mongodb+srv://TOKEN@cluster.example/emissary"))
	require.Equal(t, "file://", locationLabel("file:///etc/emissary/config.json"))
	require.Equal(t, "", locationLabel("mongodb://admin:pass@db.example:notaport\x7f/emissary"))
}
