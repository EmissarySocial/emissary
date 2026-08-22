package config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestConfigDatabase covers reading the database name out of the configuration connection string.
func TestConfigDatabase(t *testing.T) {

	table := []struct {
		location string
		expected string
	}{
		// A file:// configuration has no database at all
		{"file://./config.json", ""},
		{"file:///etc/emissary/config.json", ""},

		// The path of the connection string names the database
		{"mongodb://localhost:27017/mycfgdb", "mycfgdb"},
		{"mongodb://user:pass@localhost:27017/mycfgdb?directConnection=true", "mycfgdb"},
		{"mongodb+srv://cluster.example.com/mycfgdb", "mycfgdb"},

		// No path means the default
		{"mongodb://localhost:27017", DefaultConfigDatabase},
		{"mongodb://localhost:27017/", DefaultConfigDatabase},
		{"mongodb://localhost:27017/?directConnection=true", DefaultConfigDatabase},
	}

	for _, test := range table {
		args := CommandLineArgs{Location: test.location}
		require.Equal(t, test.expected, args.ConfigDatabase(), "location: %s", test.location)
	}
}

// TestResolveConfigDatabase pins the precedence that decides WHERE a node looks for the server
// configuration.  It matters more than it looks: two nodes that resolve this differently watch
// different collections, so they can never see each other's configuration changes, and each one
// looks like it needs a reboot.  The connection string leg is the one that used to be missing --
// ConfigDatabase() existed but nothing called it.
func TestResolveConfigDatabase(t *testing.T) {

	table := []struct {
		name       string
		location   string
		database   string
		isExplicit bool
		expected   string
	}{
		{
			name:       "explicit --db beats the connection string",
			location:   "mongodb://localhost:27017/from-the-uri",
			database:   "from-the-flag",
			isExplicit: true,
			expected:   "from-the-flag",
		},
		{
			name:     "connection string beats the default",
			location: "mongodb://localhost:27017/from-the-uri",
			database: DefaultConfigDatabase,
			expected: "from-the-uri",
		},
		{
			name:     "connection string with options still resolves",
			location: "mongodb://localhost:27017/from-the-uri?directConnection=true",
			database: DefaultConfigDatabase,
			expected: "from-the-uri",
		},
		{
			name:     "connection string with no database falls back to the default",
			location: "mongodb://localhost:27017/?directConnection=true",
			database: DefaultConfigDatabase,
			expected: DefaultConfigDatabase,
		},
		{
			name:     "a file configuration keeps the default, which it never uses",
			location: "file://./config.json",
			database: DefaultConfigDatabase,
			expected: DefaultConfigDatabase,
		},
		{
			name:       "an explicit --db is honored even for a file configuration",
			location:   "file://./config.json",
			database:   "from-the-flag",
			isExplicit: true,
			expected:   "from-the-flag",
		},
	}

	for _, test := range table {
		result := resolveConfigDatabase(test.location, test.database, test.isExplicit)
		require.Equal(t, test.expected, result, test.name)
	}
}
