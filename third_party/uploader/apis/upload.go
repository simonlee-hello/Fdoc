package apis

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
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
		displayKey, normalized, err := crypto.NormalizeKey(transferConfig.CryptoKey, true)
		if err != nil {
			fmt.Fprintf(os.Stderr, "encrypt: %v\n", err)
			return err
		}
		transferConfig.CryptoKey = normalized
		if !QuietMode {
			fmt.Fprintf(os.Stderr, "key: %s\n", displayKey)
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
		resp, err := upload(file, sizes[n], backend)
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

func UploadFile(path string, backend BaseBackend) (string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return "", err
	}
	size := info.Size()
	nameSize := size
	if transferConfig.CryptoMode {
		nameSize = crypto.CalcEncryptSize(size)
	}
	if err := utils.CheckUploadSize(filepath.Base(path), nameSize, transferConfig.MaxBytes, transferConfig.BackendName); err != nil {
		if SizeHint != nil {
			if hint := SizeHint(nameSize); hint != "" {
				return "", fmt.Errorf("%w\n%s", err, hint)
			}
		}
		return "", err
	}
	if err := backend.InitUpload([]string{path}, []int64{nameSize}); err != nil {
		return "", err
	}
	link, err := upload(path, size, backend)
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

func upload(file string, size int64, backend BaseBackend) (string, error) {
	info, err := os.Stat(file)
	if err != nil {
		return "", err
	}

	name := info.Name()
	// Always derive encrypt size from plaintext file length. Callers may already
	// pass CalcEncryptSize()'d values into size (for InitUpload); recomputing
	// from that would inflate Content-Length by magic+IV+pad (~32 bytes).
	uploadSize := size
	if transferConfig.CryptoMode {
		uploadSize = crypto.CalcEncryptSize(info.Size())
		// Keep a normal-looking name: hosts like tmpfiles reject ".encrypt".
		// No extension → .bin so the upload still looks like a regular file.
		if filepath.Ext(name) == "" {
			name = name + ".bin"
		}
	} else if uploadSize <= 0 {
		uploadSize = info.Size()
	}

	if err = backend.PreUpload(name, uploadSize); err != nil {
		return "", err
	}
	fileStream, err := os.Open(file)
	if err != nil {
		return "", err
	}
	defer fileStream.Close()

	var reader io.Reader = fileStream
	if transferConfig.CryptoMode {
		pr, pw := io.Pipe()
		go func() {
			encErr := crypto.StreamEncrypt(fileStream, pw, transferConfig.CryptoKey, 0)
			_ = pw.CloseWithError(encErr)
		}()
		reader = pr
	}

	if !transferConfig.NoBarMode {
		reader = backend.StartProgress(reader, uploadSize)
	}

	if err = backend.DoUpload(name, uploadSize, reader); err != nil {
		return "", err
	}
	if !transferConfig.NoBarMode {
		backend.EndProgress()
	}
	return backend.PostUpload(name, uploadSize)
}
