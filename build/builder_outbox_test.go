package build

import "testing"

func TestOutboxBuilder(t *testing.T) {
	var _ StateSetter = Outbox{}
}
