package set

import (
	"testing"

	"github.com/davecgh/go-spew/spew"
)

func Test_Map(t *testing.T) {

	s := NewMap[string, testPerson]()

	s.Put(testPerson{id: "1", name: "Sarah", email: "sarah@sky.net"})
	s.Put(testPerson{id: "2", name: "John", email: "john@sky.net"})

	spew.Dump(s)

	v, err := s.Get("1")

	spew.Dump(v, err)
}
