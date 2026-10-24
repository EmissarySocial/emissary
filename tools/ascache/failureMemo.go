package ascache

import (
	"net/http"
	"sync"
	"time"

	"github.com/benpate/derp"
)

// failureMemoLimit caps how many failed URLs are remembered at once
const failureMemoLimit = 10_000

// FailureMemo remembers URLs whose fetch recently failed with a server or network error or a 403.
// It is safe for concurrent use, and is shared by every Client that is given it.
type FailureMemo struct {
	mutex  sync.Mutex
	until  map[string]time.Time
	window time.Duration
	now    func() time.Time
}

// NewFailureMemo returns a FailureMemo that remembers each failure for the provided window
func NewFailureMemo(window time.Duration) *FailureMemo {
	return &FailureMemo{
		until:  make(map[string]time.Time),
		window: window,
		now:    time.Now,
	}
}

// failedRecently reports whether the URL failed within the window, forgetting it once the window passes
func (memo *FailureMemo) failedRecently(url string) bool {

	memo.mutex.Lock()
	defer memo.mutex.Unlock()

	until, found := memo.until[url]

	if !found {
		return false
	}

	if memo.now().After(until) {
		delete(memo.until, url)
		return false
	}

	return true
}

// remember records a failed fetch of the URL
func (memo *FailureMemo) remember(url string) {

	memo.mutex.Lock()
	defer memo.mutex.Unlock()

	// RULE: When full, drop expired entries first, and start over only if that is not enough
	if len(memo.until) >= failureMemoLimit {

		now := memo.now()

		for key, until := range memo.until {
			if now.After(until) {
				delete(memo.until, key)
			}
		}

		if len(memo.until) >= failureMemoLimit {
			memo.until = make(map[string]time.Time)
		}
	}

	memo.until[url] = memo.now().Add(memo.window)
}

// forget clears the URL after a successful fetch
func (memo *FailureMemo) forget(url string) {

	memo.mutex.Lock()
	defer memo.mutex.Unlock()

	delete(memo.until, url)
}

// isRememberedFailure reports whether a fetch error is worth remembering: a server or network failure
// (5xx, a timeout, no connection), or a 403 from a server that refuses us (a domain block, for one)
func isRememberedFailure(err error) bool {
	code := derp.ErrorCode(err)
	return code == 0 || code == http.StatusForbidden || code >= 500
}
