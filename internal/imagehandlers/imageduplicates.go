package imagehandlers

import (
	"cmp"
	"context"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"

	"github.com/corona10/goimagehash"
	_ "golang.org/x/image/webp"
)

type ImageInfo struct {
	Filepath string
	Filename string
	Phash    *goimagehash.ImageHash
	Width    int
	Height   int
	Area     int
	FileSize int64
}

func FindDuplicates(ctx context.Context, maxHammingDistance int, workers int) ([][]*ImageInfo, error) {
	dir, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	images, err := scanImages(ctx, dir, workers)
	if err != nil {
		return nil, err
	}
	if len(images) == 0 {
		return nil, nil
	}
	return groupDuplicates(images, maxHammingDistance), nil
}

func scanImages(ctx context.Context, dir string, workers int) ([]*ImageInfo, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
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
		return nil, nil
	}
	pathChan := make(chan string, len(paths))
	resultChan := make(chan *ImageInfo, len(paths))
	var wg sync.WaitGroup
	for range workers {
		wg.Go(func() {
			for path := range pathChan {
				select {
				case <-ctx.Done():
					return
				default:
				}
				info := processImage(path)
				if info != nil {
					resultChan <- info
				}
			}
		})
	}
	for _, path := range paths {
		pathChan <- path
	}
	close(pathChan)
	wg.Wait()
	close(resultChan)

	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	var images []*ImageInfo
	for info := range resultChan {
		images = append(images, info)
	}
	return images, nil
}

func processImage(path string) *ImageInfo {
	file, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer file.Close()
	stat, err := file.Stat()
	if err != nil {
		return nil
	}
	img, _, err := image.Decode(file)
	if err != nil {
		return nil
	}
	hash, err := goimagehash.PerceptionHash(img)
	if err != nil {
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
		FileSize: stat.Size(),
	}
}

// PNG > WebP > JPG/JPEG — lower rank means preferred for keeping
func formatRank(filename string) int {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".png":
		return 0
	case ".webp":
		return 1
	case ".jpg", ".jpeg":
		return 2
	default:
		return 3
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
			slices.SortFunc(currentGroup, func(a, b *ImageInfo) int {
				if c := cmp.Compare(b.Area, a.Area); c != 0 {
					return c
				}
				if c := cmp.Compare(formatRank(a.Filename), formatRank(b.Filename)); c != 0 {
					return c
				}
				return cmp.Compare(b.FileSize, a.FileSize)
			})
			groups = append(groups, currentGroup)
		}
	}
	return groups
}
