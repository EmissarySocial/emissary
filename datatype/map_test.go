package datatype

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMap(t *testing.T) {

	m := Map{"hello": "there", "general": "kenobi", "intValue": 69, "boolValue": true}

	// stringValues
	require.Equal(t, "there", m.AsString("hello"))
	require.Equal(t, "kenobi", m.AsString("general"))
	require.Equal(t, "", m.AsString("bueller"))

	// intValues
	require.Equal(t, 69, m.AsInt("intValue"))
	require.Zero(t, m.AsInt("hello"))
	require.Zero(t, m.AsInt("missing"))

	// boolValues
	require.True(t, m.AsBool("boolValue"))
	require.True(t, m.AsBool("intValue"))
	require.False(t, m.AsBool("hello"))
	require.False(t, m.AsBool("missing"))

	// interfaceValues
	require.Equal(t, "there", m.AsInterface("hello"))
	require.Equal(t, 69, m.AsInterface("intValue"))
	require.Equal(t, true, m.AsInterface("boolValue"))
	require.Nil(t, m.AsInterface("mising"))
}
