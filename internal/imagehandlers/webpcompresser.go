package imagehandlers

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"sync"

	"github.com/rs/zerolog/log"
	"github.com/tanq16/nits/utils"
)

func RunImgWebp(ctx context.Context, dryRun bool, workers int) error {
	path := "."
	extensions := []string{".jpg", ".jpeg", ".png", ".tiff"}
	entries, err := os.ReadDir(path)
	if err != nil {
		return fmt.Errorf("failed to read directory: %w", err)
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
		utils.PrintInfo("No images found")
		return nil
	}

	stats := map[string]int64{
		"processed":         0,
		"quality_98":        0,
		"quality_95":        0,
		"resized":           0,
		"final_under_190":   0,
		"final_over_190":    0,
		"total_saved_bytes": 0,
	}
	var statsMutex sync.Mutex
	var detailedLogs []string
	var logsMutex sync.Mutex
	var originalFiles []string
	var filesMutex sync.Mutex
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
				stats["processed"]++
				statsMutex.Unlock()
				filesMutex.Lock()
				originalFiles = append(originalFiles, filename)
				filesMutex.Unlock()

				exec.CommandContext(ctx, magickCmd, inputPath, "-quality", "98", webpPath).Run()
				webpSize := getFileSize(webpPath)
				if webpSize >= origSize {
					exec.CommandContext(ctx, magickCmd, inputPath, "-quality", "95", webpPath).Run()
					statsMutex.Lock()
					stats["quality_95"]++
					statsMutex.Unlock()
					webpSize = getFileSize(webpPath)
				} else {
					statsMutex.Lock()
					stats["quality_98"]++
					statsMutex.Unlock()
				}

				if webpSize > 190*1024 {
					resizedThisFile := false
					for scale := 90; scale >= 60; scale -= 10 {
						exec.CommandContext(ctx, magickCmd, webpPath, "-resize", fmt.Sprintf("%d%%", scale), tempWebp).Run()
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
						stats["resized"]++
						statsMutex.Unlock()
					}
					if _, err := os.Stat(tempWebp); err == nil {
						os.Remove(tempWebp)
					}
				}
				if webpSize <= 190*1024 {
					statsMutex.Lock()
					stats["final_under_190"]++
					statsMutex.Unlock()
				} else {
					statsMutex.Lock()
					stats["final_over_190"]++
					statsMutex.Unlock()
				}
				statsMutex.Lock()
				stats["total_saved_bytes"] += (origSize - webpSize)
				statsMutex.Unlock()

				if dryRun {
					logEntry := fmt.Sprintf("%s: %s -> webp | %.1fKB -> %.1fKB", filename, inputExt, float64(origSize)/1024, float64(webpSize)/1024)
					logsMutex.Lock()
					detailedLogs = append(detailedLogs, logEntry)
					logsMutex.Unlock()
				} else {
					os.Remove(inputPath)
				}
			}
		})
	}
	for _, p := range paths {
		pathChan <- p
	}
	close(pathChan)
	wg.Wait()

	if ctx.Err() != nil {
		return ctx.Err()
	}

	if dryRun {
		if err := os.WriteFile("to-delete.txt", []byte(strings.Join(originalFiles, "\n")), 0644); err != nil {
			return fmt.Errorf("failed to write to-delete.txt: %w", err)
		}
	}

	utils.PrintInfo("Conversion statistics")
	utils.PrintGeneric(fmt.Sprintf("Total images processed:      %d", stats["processed"]))
	utils.PrintGeneric(fmt.Sprintf("Retained with Quality 98:    %d", stats["quality_98"]))
	utils.PrintGeneric(fmt.Sprintf("Fallback to Quality 95:      %d", stats["quality_95"]))
	utils.PrintGeneric(fmt.Sprintf("Images requiring Resizing:   %d", stats["resized"]))
	utils.PrintGeneric(fmt.Sprintf("Final WebP <= 190 KB:        %d", stats["final_under_190"]))
	utils.PrintGeneric(fmt.Sprintf("Final WebP > 190 KB:         %d", stats["final_over_190"]))
	utils.PrintGeneric(fmt.Sprintf("Total storage space saved:   %.2f MB", float64(stats["total_saved_bytes"])/1024/1024))

	if dryRun {
		utils.PrintInfo("Dry run logs")
		for _, entry := range detailedLogs {
			utils.PrintGeneric(entry)
		}
		log.Debug().Str("filename", "to-delete.txt").Msg("Original filenames saved")
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
