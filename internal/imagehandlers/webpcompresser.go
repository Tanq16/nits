package imagehandlers

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"sync"
)

type WebPStats struct {
	Processed       int64
	Quality98       int64
	Quality95       int64
	Resized         int64
	FinalUnder190   int64
	FinalOver190    int64
	TotalSavedBytes int64
	DetailedLogs    []string
	OriginalFiles   []string
}

func RunImgWebp(ctx context.Context, dryRun bool, workers int) (*WebPStats, error) {
	path := "."
	extensions := []string{".jpg", ".jpeg", ".png", ".tiff"}
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}
	var paths []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(entry.Name()))
		if !slices.Contains(extensions, ext) {
			continue
		}
		paths = append(paths, filepath.Join(path, entry.Name()))
	}
	if len(paths) == 0 {
		return &WebPStats{}, nil
	}

	stats := &WebPStats{}
	var statsMutex sync.Mutex
	magickCmd := getImageMagickCommand()

	pathChan := make(chan string, len(paths))
	var wg sync.WaitGroup
	for range workers {
		wg.Go(func() {
			for inputPath := range pathChan {
				select {
				case <-ctx.Done():
					return
				default:
				}

				filename := filepath.Base(inputPath)
				ext := strings.ToLower(filepath.Ext(filename))
				origSize := getFileSize(inputPath)
				uuidName := strings.TrimSuffix(filename, ext)
				inputExt := strings.TrimPrefix(ext, ".")
				webpPath := filepath.Join(path, fmt.Sprintf("%s.webp", uuidName))
				tempWebp := filepath.Join(path, fmt.Sprintf("%s_temp.webp", uuidName))

				statsMutex.Lock()
				stats.Processed++
				stats.OriginalFiles = append(stats.OriginalFiles, filename)
				statsMutex.Unlock()

				runCmd(ctx, magickCmd, inputPath, "-quality", "98", webpPath)
				webpSize := getFileSize(webpPath)
				if webpSize >= origSize {
					runCmd(ctx, magickCmd, inputPath, "-quality", "95", webpPath)
					statsMutex.Lock()
					stats.Quality95++
					statsMutex.Unlock()
					webpSize = getFileSize(webpPath)
				} else {
					statsMutex.Lock()
					stats.Quality98++
					statsMutex.Unlock()
				}

				if webpSize > 190*1024 {
					resizedThisFile := false
					for scale := 90; scale >= 60; scale -= 10 {
						runCmd(ctx, magickCmd, webpPath, "-resize", fmt.Sprintf("%d%%", scale), tempWebp)
						newSize := getFileSize(tempWebp)
						resizedThisFile = true
						if newSize <= 190*1024 || scale == 60 {
							os.Rename(tempWebp, webpPath)
							webpSize = newSize
							break
						}
					}
					if resizedThisFile {
						statsMutex.Lock()
						stats.Resized++
						statsMutex.Unlock()
					}
					if _, err := os.Stat(tempWebp); err == nil {
						os.Remove(tempWebp)
					}
				}
				statsMutex.Lock()
				if webpSize <= 190*1024 {
					stats.FinalUnder190++
				} else {
					stats.FinalOver190++
				}
				stats.TotalSavedBytes += (origSize - webpSize)

				if dryRun {
					logEntry := fmt.Sprintf("%s: %s -> webp | %.1fKB -> %.1fKB", filename, inputExt, float64(origSize)/1024, float64(webpSize)/1024)
					stats.DetailedLogs = append(stats.DetailedLogs, logEntry)
				} else {
					os.Remove(inputPath)
				}
				statsMutex.Unlock()
			}
		})
	}
	for _, p := range paths {
		pathChan <- p
	}
	close(pathChan)
	wg.Wait()

	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	if dryRun {
		if err := os.WriteFile("to-delete.txt", []byte(strings.Join(stats.OriginalFiles, "\n")), 0644); err != nil {
			return nil, err
		}
	}

	return stats, nil
}

func runCmd(ctx context.Context, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		detail := strings.TrimSpace(stderr.String())
		if detail != "" {
			return fmt.Errorf("%s: %w", detail, err)
		}
		return err
	}
	return nil
}

func getFileSize(path string) int64 {
	info, err := os.Stat(path)
	if err != nil {
		return 0
	}
	return info.Size()
}

func getImageMagickCommand() string {
	switch runtime.GOOS {
	case "windows":
		if _, err := exec.LookPath("magick.exe"); err == nil {
			return "magick.exe"
		}
		if _, err := exec.LookPath("magick"); err == nil {
			return "magick"
		}
		return "magick"
	case "darwin":
		if _, err := exec.LookPath("convert"); err == nil {
			return "convert"
		}
		if _, err := exec.LookPath("magick"); err == nil {
			return "magick"
		}
		return "convert"
	default:
		if _, err := exec.LookPath("convert"); err == nil {
			return "convert"
		}
		if _, err := exec.LookPath("magick"); err == nil {
			return "magick"
		}
		return "convert"
	}
}
