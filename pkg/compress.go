package pkg

import (
	"Fdoc/logx"
	"Fdoc/option"
	"Fdoc/pkg/compress"
	"Fdoc/utils"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// RunResult is the outcome of WalkAndCompress.
type RunResult struct {
	OutputPath   string
	ArchiveBytes int64
	MatchedFiles int
	Truncated    bool
	Err          error
}

// WalkAndCompress walks the tree and optionally packs matches.
func WalkAndCompress(info *option.FlagInfo) RunResult {
	logx.Info("Starting..")
	var tarGzWriter *compress.TarGzWriter

	outputAbs, err := filepath.Abs(info.OutputPath)
	if err != nil {
		outputAbs = filepath.Clean(info.OutputPath)
	}
	info.OutputPath = outputAbs

	if !info.Size {
		tarGzWriter, err = compress.NewTarGzWriter(info.OutputPath)
		if err != nil {
			return RunResult{OutputPath: info.OutputPath, Err: fmt.Errorf("create output: %w", err)}
		}
		defer tarGzWriter.Close()
	}

	var (
		totalLogical   int64
		totalAllocated int64
		matchedFiles   int
		skippedAccess  int
		truncated      bool
		seenInodes     = make(map[[2]uint64]struct{})
		outputID       [2]uint64
		hasOutputID    bool
	)
	if !info.Size {
		if fi, err := os.Lstat(info.OutputPath); err == nil {
			if id, ok := utils.FileIdentity(info.OutputPath, fi); ok {
				outputID = id
				hasOutputID = true
			}
		}
	}

	filter := NewFileFilter(info)
	maxBytes, err := utils.ParseSize(info.MaxSize)
	if err != nil {
		return RunResult{OutputPath: info.OutputPath, Err: fmt.Errorf("invalid -max: %w", err)}
	}
	maxFileBytes := int64(0)
	if info.MaxFileSize != "" && info.MaxFileSize != "0" {
		maxFileBytes, err = utils.ParseSize(info.MaxFileSize)
		if err != nil {
			return RunResult{OutputPath: info.OutputPath, Err: fmt.Errorf("invalid -max-file: %w", err)}
		}
	}

	noteAccessDenied := func(path string, err error) {
		skippedAccess++
		logx.Debug("skip inaccessible: %s (%v)", path, err)
	}

	isOutputSelf := func(path string, fi os.FileInfo) bool {
		if utils.SamePath(path, outputAbs) {
			return true
		}
		if hasOutputID && fi != nil {
			if id, ok := utils.FileIdentity(path, fi); ok && id == outputID {
				return true
			}
		}
		return false
	}

	walker := func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			if utils.IsAccessDenied(err) {
				noteAccessDenied(path, err)
				if d != nil && d.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
			logx.Debug("walk error: %v", err)
			return nil
		}

		if d.IsDir() {
			if shouldSkipDir(path, info) {
				return filepath.SkipDir
			}
			return nil
		}

		if d.Type()&fs.ModeSymlink != 0 {
			return nil
		}
		if utils.SamePath(path, outputAbs) {
			logx.Debug("skip output archive itself: %s", path)
			return nil
		}

		// Stat/regular-file check before Filter so -k never opens FIFOs/devices.
		fileInfo, err := os.Lstat(path)
		if err != nil {
			if utils.IsAccessDenied(err) {
				noteAccessDenied(path, err)
			} else {
				logx.Debug("lstat skip: %s (%v)", path, err)
			}
			return nil
		}
		if !fileInfo.Mode().IsRegular() {
			return nil
		}
		if isOutputSelf(path, fileInfo) {
			logx.Debug("skip output archive itself: %s", path)
			return nil
		}

		if id, ok := utils.FileIdentity(path, fileInfo); ok {
			if _, seen := seenInodes[id]; seen {
				return nil
			}
		}

		if !filter.Filter(path, d) {
			return nil
		}

		logical := utils.LogicalSize(fileInfo)
		allocated := utils.AllocatedSize(fileInfo)

		if maxFileBytes > 0 && logical > maxFileBytes {
			logx.Debug("skip file over -max-file: %s (%s)", path, utils.BytesToSize(logical))
			return nil
		}

		if maxBytes > 0 && totalLogical+logical > maxBytes {
			truncated = true
			logx.Warning("reached -max %s; keeping partial result", info.MaxSize)
			return fs.SkipAll
		}

		if !info.Size {
			if err := compress.FileToTarGz(path, info.RootPath, tarGzWriter.TarWriter); err != nil {
				if utils.IsAccessDenied(err) {
					noteAccessDenied(path, err)
				} else {
					logx.Debug("pack skip: %s (%v)", path, err)
				}
				return nil
			}
		}

		// Mark inode only after a successful include.
		if id, ok := utils.FileIdentity(path, fileInfo); ok {
			seenInodes[id] = struct{}{}
		}

		totalLogical += logical
		totalAllocated += allocated
		matchedFiles++
		logx.Debug("matched file: %v logical=%d allocated=%d", path, logical, allocated)
		return nil
	}

	if err := filepath.WalkDir(info.RootPath, walker); err != nil {
		if tarGzWriter != nil {
			_ = tarGzWriter.Close()
		}
		utils.DeleteFile(info.OutputPath)
		return RunResult{OutputPath: info.OutputPath, Err: fmt.Errorf("walk directory: %w", err)}
	}

	skipNote := ""
	if skippedAccess > 0 {
		skipNote = fmt.Sprintf(", skipped %d inaccessible", skippedAccess)
	}

	if info.Size {
		msg := fmt.Sprintf("totalSize: %s disk, %s logical (%d files%s)",
			utils.BytesToSize(totalAllocated),
			utils.BytesToSize(totalLogical),
			matchedFiles,
			skipNote)
		if truncated {
			msg += fmt.Sprintf(" [truncated at -max %s]", info.MaxSize)
		}
		fmt.Println(msg)
		return RunResult{MatchedFiles: matchedFiles, Truncated: truncated}
	}

	// Flush gzip/tar before reading archive size.
	if err := tarGzWriter.Close(); err != nil {
		return RunResult{OutputPath: info.OutputPath, Err: fmt.Errorf("close archive: %w", err)}
	}

	if matchedFiles == 0 {
		utils.DeleteFile(info.OutputPath)
		if truncated {
			fmt.Printf("TRUNCATED at -max %s; no files packed%s\n", info.MaxSize, skipNote)
			return RunResult{Truncated: true}
		}
		fmt.Printf("no matching files%s\n", skipNote)
		return RunResult{}
	}

	archiveBytes := utils.GetTotalSize([]string{info.OutputPath})
	tarGzSize := utils.BytesToSize(archiveBytes)
	if truncated {
		fmt.Printf("TRUNCATED at -max %s; path=%s size=%s files=%d%s\n",
			info.MaxSize, info.OutputPath, tarGzSize, matchedFiles, skipNote)
	} else {
		fmt.Printf("SUCCESS! path=%s size=%s files=%d%s\n",
			info.OutputPath, tarGzSize, matchedFiles, skipNote)
	}
	return RunResult{
		OutputPath:   info.OutputPath,
		ArchiveBytes: archiveBytes,
		MatchedFiles: matchedFiles,
		Truncated:    truncated,
	}
}

func shouldSkipDir(path string, info *option.FlagInfo) bool {
	if info.SkipDirs == "" {
		return false
	}

	for _, skipDir := range utils.ConvertStringToList(info.SkipDirs) {
		if skipDir == "" {
			continue
		}
		resolved := skipDir
		if !filepath.IsAbs(skipDir) {
			resolved = filepath.Join(info.RootPath, skipDir)
		}
		// SamePath is case-insensitive on Windows (C:\Windows vs c:\Windows).
		if utils.SamePath(path, resolved) {
			return true
		}
	}
	return false
}
