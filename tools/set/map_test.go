package set

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_Map(t *testing.T) {

	s := NewMap[testPerson]()

	s.Put(testPerson{id: "1", name: "Sarah", email: "sarah@sky.net"})
	s.Put(testPerson{id: "2", name: "John", email: "john@sky.net"})

	require.Equal(t, 2, s.Len())

	one, ok := s.Get("1")
	require.True(t, ok)
	require.Equal(t, "Sarah", one.name)
	require.Equal(t, "sarah@sky.net", one.email)

	two, ok := s.Get("2")
	require.True(t, ok)
	require.Equal(t, "John", two.name)
	require.Equal(t, "john@sky.net", two.email)

	three, ok := s.Get("3")
	require.True(t, ok)
	require.Equal(t, "", three.name)
	require.Equal(t, "", three.email)
}
