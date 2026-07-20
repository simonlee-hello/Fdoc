package apis

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
	"uploader/crypto"
	"uploader/utils"
)

func Upload(files []string, backend BaseBackend) error {
	tmpOut := os.Stdout
	if MuteMode {
		transferConfig.NoBarMode = true
		devNull, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
		if err == nil {
			os.Stdout = devNull
			defer func() {
				os.Stdout = tmpOut
				_ = devNull.Close()
			}()
		}
	}

	var (
		sizes   []int64
		paths   []string
		cleanup []string
	)
	defer func() {
		for _, p := range cleanup {
			_ = os.Remove(p)
		}
	}()

	for _, v := range files {
		info, err := os.Stat(v)
		if err != nil {
			fmt.Fprintf(os.Stderr, "stat: %v\n", err)
			return err
		}
		if info.IsDir() && !transferConfig.RecursiveDirs {
			if !QuietMode {
				fmt.Fprintf(os.Stderr, "packing %s ...\n", v)
			}
			zipPath, err := utils.ZipDirTemp(v)
			if err != nil {
				fmt.Fprintf(os.Stderr, "pack: %v\n", err)
				return err
			}
			cleanup = append(cleanup, zipPath)
			zi, err := os.Stat(zipPath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "pack: %v\n", err)
				return err
			}
			if !QuietMode {
				fmt.Fprintf(os.Stderr, "packed %s (%s)\n", filepath.Base(zipPath), utils.FormatByteSize(zi.Size()))
			}
			paths = append(paths, zipPath)
			sizes = append(sizes, zi.Size())
			continue
		}

		err = filepath.Walk(v, func(path string, fi os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if fi.IsDir() {
				return nil
			}
			paths = append(paths, path)
			sizes = append(sizes, fi.Size())
			return nil
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "walk: %v\n", err)
			return err
		}
	}

	if len(paths) == 0 {
		err := fmt.Errorf("no files to upload")
		fmt.Fprintln(os.Stderr, err)
		return err
	}

	if transferConfig.CryptoMode {
		// Never auto-generate a key for upload encrypt — caller must pass -key.
		displayKey, normalized, err := crypto.NormalizeKey(transferConfig.CryptoKey, false)
		if err != nil {
			fmt.Fprintf(os.Stderr, "encrypt: %v\n", err)
			return err
		}
		transferConfig.CryptoKey = normalized
		if !QuietMode {
			fmt.Fprintf(os.Stderr, "encrypt key ready (%d chars display)\n", len(displayKey))
		}
		for i := range sizes {
			sizes[i] = crypto.CalcEncryptSize(sizes[i])
		}
	}

	for i, sz := range sizes {
		if err := utils.CheckUploadSize(filepath.Base(paths[i]), sz, transferConfig.MaxBytes, transferConfig.BackendName); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			if SizeHint != nil {
				if hint := SizeHint(sz); hint != "" {
					fmt.Fprintln(os.Stderr, hint)
				}
			}
			return err
		}
	}

	if err := backend.InitUpload(paths, sizes); err != nil {
		fmt.Fprintf(os.Stderr, "init: %v\n", err)
		return err
	}
	var uploadErr error
	for n, file := range paths {
		resp, err := uploadWith(file, sizes[n], backend, fileOptsFromGlobal())
		if err != nil {
			fmt.Fprintf(os.Stderr, "upload %s: %v\n", filepath.Base(file), err)
			uploadErr = err
		}
		if resp != "" && MuteMode {
			fmt.Fprintln(tmpOut, resp)
		}
		if resp != "" && Output != "" {
			f, err := os.OpenFile(Output, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				fmt.Fprintf(os.Stderr, "result file: %v\n", err)
			} else {
				_, _ = f.Write([]byte(resp + "\n"))
				_ = f.Close()
			}
		}
	}
	resp, err := backend.FinishUpload(files)
	if err != nil {
		fmt.Fprintf(os.Stderr, "finish: %v\n", err)
		if uploadErr == nil {
			uploadErr = err
		}
	}
	if resp != "" && MuteMode {
		fmt.Fprintln(tmpOut, resp)
	}
	return uploadErr
}

// FileUploadOpts is a per-call config snapshot so probes can run concurrently
// without mutating process-global TransferConfig / Stdout.
type FileUploadOpts struct {
	MaxBytes    int64
	BackendName string
	NoBar       bool
	Crypto      bool
	CryptoKey   string
	// TickEvery controls NoBar interval progress lines on stderr.
	// 0 = default 3m; <0 = disabled. Ignored when NoBar is false (live bar used).
	TickEvery time.Duration
}

func fileOptsFromGlobal() FileUploadOpts {
	return FileUploadOpts{
		MaxBytes:    transferConfig.MaxBytes,
		BackendName: transferConfig.BackendName,
		NoBar:       transferConfig.NoBarMode,
		Crypto:      transferConfig.CryptoMode,
		CryptoKey:   transferConfig.CryptoKey,
		TickEvery:   transferConfig.TickEvery,
	}
}

// UploadFile uploads one file using the process-global TransferConfig.
func UploadFile(path string, backend BaseBackend) (string, error) {
	return UploadFileOpts(path, backend, fileOptsFromGlobal())
}

// UploadFileOpts uploads one file with an explicit config (safe for concurrent probes).
func UploadFileOpts(path string, backend BaseBackend, opts FileUploadOpts) (string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return "", err
	}
	size := info.Size()
	if opts.Crypto {
		_, normalized, err := crypto.NormalizeKey(opts.CryptoKey, false)
		if err != nil {
			return "", fmt.Errorf("encrypt: %w", err)
		}
		opts.CryptoKey = normalized
		size = crypto.CalcEncryptSize(info.Size())
	}
	if err := utils.CheckUploadSize(filepath.Base(path), size, opts.MaxBytes, opts.BackendName); err != nil {
		if SizeHint != nil {
			if hint := SizeHint(size); hint != "" {
				return "", fmt.Errorf("%w\n%s", err, hint)
			}
		}
		return "", err
	}
	if err := backend.InitUpload([]string{path}, []int64{size}); err != nil {
		return "", err
	}
	link, err := uploadWith(path, size, backend, opts)
	if err != nil {
		return "", err
	}
	finish, err := backend.FinishUpload([]string{path})
	if err != nil {
		return link, err
	}
	if finish != "" {
		return finish, nil
	}
	return link, nil
}

func uploadWith(file string, size int64, backend BaseBackend, opts FileUploadOpts) (string, error) {
	info, err := os.Stat(file)
	if err != nil {
		return "", err
	}

	name := info.Name()
	uploadSize := size
	if opts.Crypto {
		uploadSize = crypto.CalcEncryptSize(info.Size())
		// Keep a normal-looking name: hosts like tmpfiles reject ".encrypt".
		if filepath.Ext(name) == "" {
			name = name + ".bin"
		}
	} else if uploadSize <= 0 {
		uploadSize = info.Size()
	}
	// Prefer .tgz over .tar.gz so download hosts don't look like plain .gz.
	name = preferTgzName(name)

	if err = backend.PreUpload(name, uploadSize); err != nil {
		return "", err
	}

	var (
		reader  io.Reader
		closer  io.Closer
		cleanup func()
	)
	if opts.Crypto {
		// Encrypt to a temp file first so Content-Length matches bytes on the wire
		// (streaming pipe + some multipart backends previously risked uploading plaintext).
		encPath, encSize, err := encryptFileToTemp(file, opts.CryptoKey, opts.TickEvery)
		if err != nil {
			return "", err
		}
		cleanup = func() { _ = os.Remove(encPath) }
		defer cleanup()
		if encSize != uploadSize {
			return "", fmt.Errorf("encrypt size mismatch: got %d want %d", encSize, uploadSize)
		}
		f, err := os.Open(encPath)
		if err != nil {
			return "", err
		}
		closer = f
		reader = f
		// Filename stays *.tgz for host/browser friendliness, but bytes are UP01
		// ciphertext — must Fdoc decrypt (or uploader decrypt) before tar/gunzip.
		fmt.Fprintf(os.Stderr, "ENCRYPT_OK plain=%d cipher=%d decrypt_first=1 (Fdoc decrypt -key <KEY> -o out.tgz <file>)\n", info.Size(), encSize)
	} else {
		f, err := os.Open(file)
		if err != nil {
			return "", err
		}
		closer = f
		reader = f
	}
	defer closer.Close()

	var tick *utils.IntervalProgressReader
	if !opts.NoBar {
		reader = backend.StartProgress(reader, uploadSize)
	} else if every := utils.ResolveTickInterval(opts.TickEvery); every > 0 {
		tick = utils.NewIntervalProgressReader(reader, uploadSize, every, "UPLOAD")
		reader = tick
	}

	if err = backend.DoUpload(name, uploadSize, reader); err != nil {
		return "", err
	}
	if !opts.NoBar {
		backend.EndProgress()
	} else if tick != nil {
		tick.Finish()
	}
	return backend.PostUpload(name, uploadSize)
}

func encryptFileToTemp(srcPath, key string, tickEvery time.Duration) (encPath string, encSize int64, err error) {
	src, err := os.Open(srcPath)
	if err != nil {
		return "", 0, err
	}
	defer src.Close()

	// Prefer the source directory so multi-GB ciphertext lands on the same
	// filesystem as the archive — not os.TempDir(), which is often tmpfs.
	tmpDir := filepath.Dir(srcPath)
	if tmpDir == "" || tmpDir == "." {
		tmpDir = "."
	}
	tmp, err := os.CreateTemp(tmpDir, ".uploader-enc-*.bin")
	if err != nil {
		// Read-only / full volume: fall back so encrypt still works when possible.
		fallback := os.TempDir()
		fmt.Fprintf(os.Stderr, "encrypt temp in %s failed (%v); fallback %s (may be tmpfs — large files need RAM/disk)\n", tmpDir, err, fallback)
		tmp, err = os.CreateTemp(fallback, ".uploader-enc-*.bin")
		if err != nil {
			return "", 0, err
		}
	}
	encPath = tmp.Name()
	defer func() {
		_ = tmp.Close()
		if err != nil {
			_ = os.Remove(encPath)
		}
	}()

	plainSize, _ := src.Stat()
	var encOut io.Writer = tmp
	var tick *utils.IntervalProgressWriter
	if every := utils.ResolveTickInterval(tickEvery); every > 0 && plainSize != nil {
		wantCipher := crypto.CalcEncryptSize(plainSize.Size())
		tick = utils.NewIntervalProgressWriter(tmp, wantCipher, every, "ENCRYPT")
		encOut = tick
	}

	if err = crypto.StreamEncrypt(src, encOut, key, 0); err != nil {
		return "", 0, err
	}
	if tick != nil {
		tick.Finish()
	}
	if err = tmp.Sync(); err != nil {
		return "", 0, err
	}
	fi, err := tmp.Stat()
	if err != nil {
		return "", 0, err
	}
	encSize = fi.Size()

	// Verify modern header so we never upload a failed/partial ciphertext as "encrypted".
	if _, err = tmp.Seek(0, io.SeekStart); err != nil {
		return "", 0, err
	}
	magic := make([]byte, 4)
	if _, err = io.ReadFull(tmp, magic); err != nil {
		return "", 0, err
	}
	if string(magic) != "UP01" {
		return "", 0, fmt.Errorf("encrypt produced invalid header %q (want UP01)", magic)
	}
	return encPath, encSize, nil
}

// preferTgzName rewrites *.tar.gz to *.tgz so hosts/browsers don't treat the
// name as plain gzip. With -encrypt the bytes are still UP01 ciphertext —
// callers must decrypt before unpacking (see ENCRYPT_OK decrypt_first=1).
func preferTgzName(name string) string {
	lower := strings.ToLower(name)
	if strings.HasSuffix(lower, ".tar.gz") {
		return name[:len(name)-len(".tar.gz")] + ".tgz"
	}
	return name
}
