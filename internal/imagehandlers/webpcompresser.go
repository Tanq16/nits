package imagehandlers

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"

	"github.com/rs/zerolog/log"
)

func RunImgWebp(dryRun bool) {
	path := "."
	extensions := []string{".jpg", ".jpeg", ".png", ".tiff"}
	entries, _ := os.ReadDir(path)
	stats := map[string]int64{
		"processed":         0,
		"quality_98":        0,
		"quality_95":        0,
		"resized":           0,
		"final_under_190":   0,
		"final_over_190":    0,
		"total_saved_bytes": 0,
	}
	var detailedLogs []string
	var originalFiles []string
	magickCmd := getImageMagickCommand()

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		filename := entry.Name()
		ext := strings.ToLower(filepath.Ext(filename))
		if !slices.Contains(extensions, ext) {
			continue
		}
		inputPath := filepath.Join(path, filename)
		origSize := getFileSize(inputPath)
		uuidName := strings.TrimSuffix(filename, ext)
		inputExt := strings.TrimPrefix(ext, ".")
		webpPath := filepath.Join(path, fmt.Sprintf("%s.webp", uuidName))
		tempWebp := filepath.Join(path, fmt.Sprintf("%s_temp.webp", uuidName))
		stats["processed"]++
		originalFiles = append(originalFiles, filename)

		exec.Command(magickCmd, inputPath, "-quality", "98", webpPath).Run()
		webpSize := getFileSize(webpPath)
		if webpSize >= origSize {
			exec.Command(magickCmd, inputPath, "-quality", "95", webpPath).Run()
			stats["quality_95"]++
			webpSize = getFileSize(webpPath)
		} else {
			stats["quality_98"]++
		}

		if webpSize > 190*1024 {
			resizedThisFile := false
			for scale := 90; scale >= 60; scale -= 10 {
				exec.Command(magickCmd, webpPath, "-resize", fmt.Sprintf("%d%%", scale), tempWebp).Run()
				newSize := getFileSize(tempWebp)
				resizedThisFile = true
				if newSize <= 190*1024 || scale == 60 {
					os.Rename(tempWebp, webpPath)
					webpSize = newSize
					break
				}
			}
			if resizedThisFile {
				stats["resized"]++
			}
			if _, err := os.Stat(tempWebp); err == nil {
				os.Remove(tempWebp)
			}
		}
		if webpSize <= 190*1024 {
			stats["final_under_190"]++
		} else {
			stats["final_over_190"]++
		}
		stats["total_saved_bytes"] += (origSize - webpSize)

		if dryRun {
			detailedLogs = append(detailedLogs, fmt.Sprintf("%s: %s -> webp | %.1fKB -> %.1fKB", filename, inputExt, float64(origSize)/1024, float64(webpSize)/1024))
		} else {
			os.Remove(inputPath)
		}
	}
	if dryRun {
		os.WriteFile("to-delete.txt", []byte(strings.Join(originalFiles, "\n")), 0644)
	}

	fmt.Println("CONVERSION STATISTICS")
	fmt.Printf("Total images processed:      %d\n", stats["processed"])
	fmt.Printf("Retained with Quality 98:    %d\n", stats["quality_98"])
	fmt.Printf("Fallback to Quality 95:      %d\n", stats["quality_95"])
	fmt.Printf("Images requiring Resizing:   %d\n", stats["resized"])
	fmt.Printf("Final WebP <= 190 KB:        %d\n", stats["final_under_190"])
	fmt.Printf("Final WebP > 190 KB:         %d\n", stats["final_over_190"])
	fmt.Printf("Total storage space saved:   %.2f MB\n", float64(stats["total_saved_bytes"])/1024/1024)

	if dryRun {
		fmt.Println("\nDRY RUN LOGS")
		for _, log := range detailedLogs {
			fmt.Println(log)
		}
		log.Info().Str("filename", "to-delete.txt").Msg("Original filenames saved")
	}
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
