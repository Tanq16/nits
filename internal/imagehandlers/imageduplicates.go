package imagehandlers

import (
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/corona10/goimagehash"
	"github.com/rs/zerolog/log"
	_ "golang.org/x/image/webp"
)

type ImageInfo struct {
	Filepath string
	Filename string
	Phash    *goimagehash.ImageHash
	Width    int
	Height   int
	Area     int
}

func RunImgDedupe(maxHammingDistance int, workers int) {
	dir, err := os.Getwd()
	if err != nil {
		log.Error().Err(err).Msg("Failed to get current directory")
		return
	}
	log.Info().Str("directory", dir).Msg("Scanning images")
	images := scanImages(dir, workers)
	if len(images) == 0 {
		fmt.Println("No images found.")
		return
	}
	log.Info().Int("count", len(images)).Msg("Images scanned")
	groups := groupDuplicates(images, maxHammingDistance)
	printResults(groups)
}

func scanImages(dir string, workers int) []*ImageInfo {
	entries, err := os.ReadDir(dir)
	if err != nil {
		log.Error().Err(err).Str("directory", dir).Msg("Failed to read directory")
		return nil
	}
	var paths []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(entry.Name()))
		if ext != ".jpg" && ext != ".jpeg" && ext != ".png" && ext != ".webp" {
			continue
		}
		paths = append(paths, filepath.Join(dir, entry.Name()))
	}
	if len(paths) == 0 {
		return nil
	}
	pathChan := make(chan string, len(paths))
	resultChan := make(chan *ImageInfo, len(paths))
	var wg sync.WaitGroup
	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for path := range pathChan {
				info := processImage(path)
				if info != nil {
					resultChan <- info
				}
			}
		}()
	}
	for _, path := range paths {
		pathChan <- path
	}
	close(pathChan)
	wg.Wait()
	close(resultChan)
	var images []*ImageInfo
	for info := range resultChan {
		images = append(images, info)
	}
	return images
}

func processImage(path string) *ImageInfo {
	file, err := os.Open(path)
	if err != nil {
		log.Error().Err(err).Str("file", path).Msg("Failed to open file")
		return nil
	}
	defer file.Close()
	img, _, err := image.Decode(file)
	if err != nil {
		return nil
	}
	hash, err := goimagehash.PerceptionHash(img)
	if err != nil {
		log.Error().Err(err).Str("file", path).Msg("Failed to generate hash")
		return nil
	}
	bounds := img.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	return &ImageInfo{
		Filepath: path,
		Filename: filepath.Base(path),
		Phash:    hash,
		Width:    w,
		Height:   h,
		Area:     w * h,
	}
}

func groupDuplicates(images []*ImageInfo, maxHammingDistance int) [][]*ImageInfo {
	var groups [][]*ImageInfo
	processed := make(map[string]bool)
	for i := range images {
		seed := images[i]
		if seed == nil || processed[seed.Filepath] {
			continue
		}
		currentGroup := []*ImageInfo{seed}
		processed[seed.Filepath] = true
		for j := i + 1; j < len(images); j++ {
			candidate := images[j]
			if candidate == nil || processed[candidate.Filepath] {
				continue
			}
			distance, err := seed.Phash.Distance(candidate.Phash)
			if err != nil {
				continue
			}
			if distance <= maxHammingDistance {
				currentGroup = append(currentGroup, candidate)
				processed[candidate.Filepath] = true
			}
		}
		if len(currentGroup) > 1 {
			sort.Slice(currentGroup, func(i, j int) bool {
				return currentGroup[i].Area > currentGroup[j].Area
			})
			groups = append(groups, currentGroup)
		}
	}
	return groups
}

func printResults(groups [][]*ImageInfo) {
	if len(groups) == 0 {
		log.Info().Msg("No duplicate images found.")
		return
	}
	log.Info().Int("groups", len(groups)).Msg("Found sets of duplicates")
	fmt.Println()
	for i, group := range groups {
		best := group[0]
		duplicates := group[1:]
		fmt.Printf("SET #%d\n", i+1)
		fmt.Printf("  - KEEP  : %s (%dx%d)\n", best.Filename, best.Width, best.Height)
		var dupNames []string
		for _, d := range duplicates {
			dupNames = append(dupNames, fmt.Sprintf("%s (%dx%d)", d.Filename, d.Width, d.Height))
		}
		fmt.Printf("  - DELETE: %s\n", strings.Join(dupNames, ", "))
		fmt.Printf("  - CMD   : rm")
		for _, d := range duplicates {
			fmt.Printf(" %q", d.Filename)
		}
		fmt.Printf("\n\n")
	}
}
