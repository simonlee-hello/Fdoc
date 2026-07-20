package methods_test

import (
	"testing"
	"time"

	"uploader/apis/methods"
)

func TestWithHTTPTimeoutOverridesNewClient(t *testing.T) {
	methods.ClearHTTPTimeoutOverride()
	t.Cleanup(methods.ClearHTTPTimeoutOverride)

	c := methods.NewClient(0)
	if c.Timeout != methods.HTTPTimeout {
		t.Fatalf("default timeout=%v want %v", c.Timeout, methods.HTTPTimeout)
	}

	want := 3 * time.Second
	err := methods.WithHTTPTimeout(want, func() error {
		c2 := methods.NewClient(0)
		if c2.Timeout != want {
			t.Fatalf("override timeout=%v want %v", c2.Timeout, want)
		}
		explicit := methods.NewClient(90 * time.Second)
		if explicit.Timeout != 90*time.Second {
			t.Fatalf("explicit timeout overridden: %v", explicit.Timeout)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	c3 := methods.NewClient(0)
	if c3.Timeout != methods.HTTPTimeout {
		t.Fatalf("after release timeout=%v want %v", c3.Timeout, methods.HTTPTimeout)
	}
}

func TestAcquireReleaseKeepsTimeoutUntilLastHolder(t *testing.T) {
	methods.ClearHTTPTimeoutOverride()
	t.Cleanup(methods.ClearHTTPTimeoutOverride)

	want := 5 * time.Second
	methods.AcquireHTTPTimeout(want)
	methods.AcquireHTTPTimeout(want)
	c := methods.NewClient(0)
	if c.Timeout != want {
		t.Fatalf("with holders timeout=%v want %v", c.Timeout, want)
	}
	methods.ReleaseHTTPTimeout()
	c2 := methods.NewClient(0)
	if c2.Timeout != want {
		t.Fatalf("one holder left timeout=%v want %v", c2.Timeout, want)
	}
	methods.ReleaseHTTPTimeout()
	c3 := methods.NewClient(0)
	if c3.Timeout != methods.HTTPTimeout {
		t.Fatalf("cleared timeout=%v want %v", c3.Timeout, methods.HTTPTimeout)
	}
}

func TestClearHTTPTimeoutOverrideUnblocksRealUpload(t *testing.T) {
	methods.ClearHTTPTimeoutOverride()
	t.Cleanup(methods.ClearHTTPTimeoutOverride)

	// Simulate timed-out probe goroutine still holding the short override.
	methods.AcquireHTTPTimeout(45 * time.Second)
	probeClient := methods.NewClient(0)
	if probeClient.Timeout != 45*time.Second {
		t.Fatalf("probe client timeout=%v", probeClient.Timeout)
	}

	methods.ClearHTTPTimeoutOverride()
	uploadClient := methods.NewClient(0)
	if uploadClient.Timeout != methods.HTTPTimeout {
		t.Fatalf("after Clear, upload client timeout=%v want %v", uploadClient.Timeout, methods.HTTPTimeout)
	}
	// Baked-in probe client timeout is unchanged.
	if probeClient.Timeout != 45*time.Second {
		t.Fatalf("existing probe client mutated: %v", probeClient.Timeout)
	}

	// Extra Release from the late goroutine must not go negative / re-arm.
	methods.ReleaseHTTPTimeout()
	methods.ReleaseHTTPTimeout()
	c := methods.NewClient(0)
	if c.Timeout != methods.HTTPTimeout {
		t.Fatalf("after extra Release timeout=%v", c.Timeout)
	}
}

func TestReleaseHTTPTimeoutIgnoresExtraCalls(t *testing.T) {
	methods.ClearHTTPTimeoutOverride()
	t.Cleanup(methods.ClearHTTPTimeoutOverride)

	methods.ReleaseHTTPTimeout()
	methods.ReleaseHTTPTimeout()
	c := methods.NewClient(0)
	if c.Timeout != methods.HTTPTimeout {
		t.Fatalf("timeout=%v", c.Timeout)
	}
}
