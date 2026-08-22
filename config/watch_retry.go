package config

import "time"

/******************************************
 * Watcher Retry Backoff
 *
 * Shared by both storage engines' supervised watchers:
 * the MongoDB change stream and the filesystem watcher.
 ******************************************/

// watchRetryMinimum is the shortest pause before a dead watcher is reopened.
const watchRetryMinimum = 1 * time.Second

// watchRetryMaximum caps the exponential backoff between reopen attempts, so a source that
// is down for hours is still retried every minute instead of drifting into never.
const watchRetryMaximum = 60 * time.Second

// watchRetryDelay returns the backoff before the Nth consecutive attempt to reopen a dead
// watcher: 1s, 2s, 4s, 8s ... capped at watchRetryMaximum.
func watchRetryDelay(failures int) time.Duration {

	result := watchRetryMinimum

	for index := 1; index < failures; index++ {

		result *= 2

		if result >= watchRetryMaximum {
			return watchRetryMaximum
		}
	}

	return result
}
