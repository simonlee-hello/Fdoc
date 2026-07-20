package methods

import (
	"sync/atomic"
	"time"
)

// probeHTTPTimeoutNs, when holders > 0, overrides HTTPTimeout for NewClient(0).
// Holders let timed-out probe goroutines keep a short client timeout until the
// in-flight upload actually finishes (probeOne does not join on deadline).
var (
	probeHTTPTimeoutNs      atomic.Int64
	probeHTTPTimeoutHolders atomic.Int32
)

// AcquireHTTPTimeout bumps a holder count and sets the override used by NewClient(0).
func AcquireHTTPTimeout(d time.Duration) {
	if d <= 0 {
		return
	}
	probeHTTPTimeoutNs.Store(int64(d))
	probeHTTPTimeoutHolders.Add(1)
}

// ReleaseHTTPTimeout drops a holder; clears the override when the last holder exits.
func ReleaseHTTPTimeout() {
	if probeHTTPTimeoutHolders.Add(-1) == 0 {
		probeHTTPTimeoutNs.Store(0)
	}
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
