package methods_test

import (
	"testing"
	"time"

	"uploader/apis/methods"
)

func TestWithHTTPTimeoutOverridesNewClient(t *testing.T) {
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
