package channels

import "sync"

// Merge combines multiple channels into a single channel
// inspired by original merge function from https://medium.com/justforfunc/two-ways-of-merging-n-channels-in-go-43c0b57cd1de
func Merge[T any](inputs ...<-chan T) <-chan T {

	// Create a waitgroup so that we know when all of the channels are closed
	var waitGroup sync.WaitGroup
	waitGroup.Add(len(inputs))

	// Create a result channel that will include all of the merged channels.
	output := make(chan T)
	for _, input := range inputs {
		go func(channel <-chan T) {
			for value := range channel {
				output <- value
			}
			waitGroup.Done()
		}(input) // Pass the value of the loop variable into the goroutine so that it's not overwritten before the goroutine is called.
	}

	// This goroutine will close the output channel when all of the inputs are closed.
	go func() {
		waitGroup.Wait()
		close(output)
	}()

	// Return the output channel
	return output
}
