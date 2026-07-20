package methods

import (
	"sync/atomic"
	"time"
)

// probeHTTPTimeoutNs, when holders > 0, overrides HTTPTimeout for NewClient(0).
// ProbeAll clears the override on return so a timed-out probe goroutine cannot
// shrink the timeout of a subsequent real upload (often multi-GB).
var (
	probeHTTPTimeoutNs      atomic.Int64
	probeHTTPTimeoutHolders atomic.Int32
)

// AcquireHTTPTimeout bumps a holder count and sets the override used by NewClient(0).
// No-op when d <= 0 (does not bump holders).
func AcquireHTTPTimeout(d time.Duration) {
	if d <= 0 {
		return
	}
	probeHTTPTimeoutNs.Store(int64(d))
	probeHTTPTimeoutHolders.Add(1)
}

// ReleaseHTTPTimeout drops a holder; never goes below zero.
func ReleaseHTTPTimeout() {
	for {
		cur := probeHTTPTimeoutHolders.Load()
		if cur <= 0 {
			probeHTTPTimeoutNs.Store(0)
			return
		}
		if probeHTTPTimeoutHolders.CompareAndSwap(cur, cur-1) {
			if cur == 1 {
				probeHTTPTimeoutNs.Store(0)
			}
			return
		}
	}
}

// ClearHTTPTimeoutOverride drops any probe timeout override immediately.
// ProbeAll defers this on return so real uploads use HTTPTimeout by default.
// Already-built *http.Client instances keep whatever Timeout they were given.
func ClearHTTPTimeoutOverride() {
	probeHTTPTimeoutHolders.Store(0)
	probeHTTPTimeoutNs.Store(0)
}

// WithHTTPTimeout runs fn while NewClient(0)/Do use the given timeout.
func WithHTTPTimeout(d time.Duration, fn func() error) error {
	if d <= 0 {
		return fn()
	}
	AcquireHTTPTimeout(d)
	defer ReleaseHTTPTimeout()
	return fn()
}

func effectiveHTTPTimeout(timeout time.Duration) time.Duration {
	if timeout > 0 {
		return timeout
	}
	if probeHTTPTimeoutHolders.Load() > 0 {
		if n := probeHTTPTimeoutNs.Load(); n > 0 {
			return time.Duration(n)
		}
	}
	return HTTPTimeout
}
