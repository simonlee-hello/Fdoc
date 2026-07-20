package utils

import (
	"errors"
	"os"
	"testing"
)

func TestIsAccessDenied(t *testing.T) {
	if !IsAccessDenied(os.ErrPermission) {
		t.Fatal("os.ErrPermission should be access denied")
	}
	if !IsAccessDenied(&os.PathError{Op: "open", Path: "/x", Err: os.ErrPermission}) {
		t.Fatal("PathError permission should be access denied")
	}
	if !IsAccessDenied(errors.New("open /foo: operation not permitted")) {
		t.Fatal("operation not permitted should match")
	}
	if IsAccessDenied(os.ErrNotExist) {
		t.Fatal("not exist should not be access denied")
	}
	if IsAccessDenied(nil) {
		t.Fatal("nil should be false")
	}
}
