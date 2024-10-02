package queue_mongo

import (
	"testing"

	"github.com/EmissarySocial/emissary/tools/queue"
)

func TestStorage(t *testing.T) {

	var _ queue.Storage = Storage{}
}
