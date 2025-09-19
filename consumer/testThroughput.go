package consumer

import (
	"fmt"
	"time"

	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/turbine/queue"
)

// TestThroughput helps test the queue performance
func TestThroughput(name string, args mapof.Any) queue.Result {
	fmt.Println("TestThroughput", args.GetString("value"))
	time.Sleep(100 * time.Millisecond)
	return queue.Success()
}
