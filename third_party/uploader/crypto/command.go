package crypto

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"uploader/utils"
)

var (
	ForceMode bool
	Prefix    string
	Key       string
	NoBar     bool
)

// RegisterFlags binds encrypt/decrypt flags.
// Aliases: -k/-key/-encrypt-key, -o/-output/-out, -f/-force
func RegisterFlags(fs *flag.FlagSet) {
	fs.StringVar(&Prefix, "output", ".", "output file or directory")
	fs.StringVar(&Prefix, "o", ".", "output file or directory")
	fs.StringVar(&Prefix, "out", ".", "output file or directory")
	fs.StringVar(&Key, "key", "", "password")
	fs.StringVar(&Key, "k", "", "password")
	fs.StringVar(&Key, "encrypt-key", "", "password")
	fs.BoolVar(&ForceMode, "force", false, "overwrite existing output")
	fs.BoolVar(&ForceMode, "f", false, "overwrite existing output")
	fs.BoolVar(&NoBar, "no-progress", false, "disable progress")
}

func NormalizeKey(key string, generateIfEmpty bool) (displayKey string, normalized string, err error) {
	displayKey = key
	if key == "" || len(key) > 32 {
		if !generateIfEmpty {
			return "", "", fmt.Errorf("key required")
		}
		displayKey = utils.GenRandString(16)
		key = displayKey
	}
	if len(key) < 32 {
		key = string(Padding([]byte(key), 32))
	}
	return displayKey, key, nil
}

func Encrypt(file string) error {
	path, err := filepath.Abs(file)
	if err != nil {
		return err
	}
	var dest string
	if utils.IsDir(Prefix) {
		dest = filepath.Join(Prefix, filepath.Base(file)+".encrypt")
	} else {
		dest = Prefix
	}
	dest, err = filepath.Abs(dest)
	if err != nil {
		return err
	}
	if utils.IsExist(dest) && !strings.HasPrefix(dest, "/dev") && !ForceMode {
		return fmt.Errorf("output exists: %s (use -force or -o PATH)", dest)
	}

	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	src, err := os.Open(path)
	if err != nil {
		return err
	}
	defer src.Close()
	enc, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer enc.Close()

	var writer io.Writer = enc
	var pw *utils.ProgressWriter
	if !NoBar {
		pw = utils.NewProgressWriter(enc, CalcEncryptSize(info.Size()))
		writer = pw
		defer pw.Finish()
	}

	generated := Key == "" || len(Key) > 32
	displayKey, normalized, err := NormalizeKey(Key, true)
	if err != nil {
		return err
	}
	Key = normalized
	if generated {
		fmt.Fprintf(os.Stderr, "key: %s\n", displayKey)
	}
	fmt.Fprintf(os.Stderr, "%s -> %s\n", path, dest)
	return StreamEncrypt(src, writer, Key, 0)
}

func Decrypt(file string) error {
	path, err := filepath.Abs(file)
	if err != nil {
		return err
	}
	var dest string
	if utils.IsDir(Prefix) {
		dest = filepath.Join(Prefix, strings.Replace(filepath.Base(file), ".encrypt", "", 1))
	} else {
		dest = Prefix
	}
	dest, err = filepath.Abs(dest)
	if err != nil {
		return err
	}
	if utils.IsExist(dest) && !strings.HasPrefix(dest, "/dev") && !ForceMode {
		return fmt.Errorf("output exists: %s (use -force or -o PATH)", dest)
	}

	src, err := os.Open(path)
	if err != nil {
		return err
	}
	defer src.Close()
	info, err := src.Stat()
	if err != nil {
		return err
	}
	dec, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer dec.Close()

	var writer io.Writer = dec
	var pw *utils.ProgressWriter
	if !NoBar {
		pw = utils.NewProgressWriter(dec, info.Size())
		writer = pw
		defer pw.Finish()
	}

	_, normalized, err := NormalizeKey(Key, false)
	if err != nil {
		return err
	}
	Key = normalized
	fmt.Fprintf(os.Stderr, "%s -> %s\n", path, dest)
	return StreamDecrypt(src, writer, Key, 0)
}
