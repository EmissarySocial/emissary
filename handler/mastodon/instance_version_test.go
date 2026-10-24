package mastodon

import (
	"runtime/debug"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestVersionFromBuild covers a release build, a local build with and without changes, and no build information.
func TestVersionFromBuild(t *testing.T) {

	release := &debug.BuildInfo{Main: debug.Module{Version: "v0.8.1"}}
	require.Equal(t, "Emissary v0.8.1", versionFromBuild(release, true))

	// A build from an untagged commit is stamped with a long pseudo-version, which reads better as the commit
	untagged := &debug.BuildInfo{Main: debug.Module{Version: "v0.7.1-0.20260929140633-a3153d0a015f+dirty"}, Settings: []debug.BuildSetting{
		{Key: "vcs.revision", Value: "a3153d0a015f9999"},
		{Key: "vcs.modified", Value: "true"},
	}}
	require.Equal(t, "Emissary a3153d0a0-dirty", versionFromBuild(untagged, true))

	local := &debug.BuildInfo{Main: debug.Module{Version: "(devel)"}, Settings: []debug.BuildSetting{
		{Key: "vcs.revision", Value: "a3153d0a0e1b2c3d4e5f"},
		{Key: "vcs.modified", Value: "false"},
	}}
	require.Equal(t, "Emissary a3153d0a0", versionFromBuild(local, true))

	local.Settings[1].Value = "true"
	require.Equal(t, "Emissary a3153d0a0-dirty", versionFromBuild(local, true))

	require.Equal(t, "Emissary", versionFromBuild(&debug.BuildInfo{Main: debug.Module{Version: "(devel)"}}, true), "no commit recorded")
	require.Equal(t, "Emissary", versionFromBuild(nil, false))
}
