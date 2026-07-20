package decrypt

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"uploader/crypto"
)

// Options for File decrypt.
type Options struct {
	Key    string
	Output string // empty → default beside input as *.tgz
	Force  bool
}

// Result of a successful decrypt.
type Result struct {
	Input  string
	Output string
}

// File decrypts an uploader/Fdoc ciphertext (UP01 + AES-256-CBC) to Output.
func File(input string, opts Options) (Result, error) {
	if strings.TrimSpace(opts.Key) == "" {
		return Result{}, fmt.Errorf("key required (-key)")
	}
	in, err := filepath.Abs(input)
	if err != nil {
		return Result{}, err
	}
	out := opts.Output
	if out == "" {
		out = defaultOutputPath(in)
	} else {
		out, err = filepath.Abs(out)
		if err != nil {
			return Result{}, err
		}
	}
	if filepath.Clean(out) == filepath.Clean(in) {
		return Result{}, fmt.Errorf("output would overwrite input %s; pass -o PATH", in)
	}
	if st, err := os.Stat(out); err == nil && !st.IsDir() && !opts.Force {
		return Result{}, fmt.Errorf("output exists: %s (use -force)", out)
	}

	_, normalized, err := crypto.NormalizeKey(opts.Key, false)
	if err != nil {
		return Result{}, err
	}

	src, err := os.Open(in)
	if err != nil {
		return Result{}, err
	}
	defer src.Close()

	dst, err := os.Create(out)
	if err != nil {
		return Result{}, err
	}
	defer dst.Close()

	if err := crypto.StreamDecrypt(src, dst, normalized, 0); err != nil {
		_ = dst.Close()
		_ = os.Remove(out)
		return Result{}, err
	}
	if err := dst.Sync(); err != nil {
		_ = dst.Close()
		_ = os.Remove(out)
		return Result{}, err
	}
	return Result{Input: in, Output: out}, nil
}

func defaultOutputPath(input string) string {
	dir := filepath.Dir(input)
	base := filepath.Base(input)
	lower := strings.ToLower(base)

	var stem string
	switch {
	case strings.HasSuffix(lower, ".tar.gz"):
		stem = base[:len(base)-len(".tar.gz")] + ".dec"
	case strings.HasSuffix(lower, ".tgz"):
		// Input already looks like an archive name — avoid writing onto itself.
		stem = strings.TrimSuffix(base, filepath.Ext(base)) + ".dec"
	default:
		ext := filepath.Ext(base)
		stem = strings.TrimSuffix(base, ext)
		if stem == "" {
			stem = base
		}
	}
	out := filepath.Join(dir, stem+".tgz")
	if filepath.Clean(out) == filepath.Clean(input) {
		out = filepath.Join(dir, stem+".dec.tgz")
	}
	return out
}
