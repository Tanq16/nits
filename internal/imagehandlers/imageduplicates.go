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

type ScanFailure struct {
	Path string
	Err  error
}

type DuplicateResult struct {
	Groups   [][]*ImageInfo
	Scanned  int
	Failures []ScanFailure
}

type scanResult struct {
	info *ImageInfo
	path string
	err  error
}

type ImageInfo struct {
	Filepath string
	Filename string
	Phash    *goimagehash.ImageHash
	Width    int
	Height   int
	Area     int
	FileSize int64
}

func FindDuplicates(ctx context.Context, maxHammingDistance int, workers int) (DuplicateResult, error) {
	dir, err := os.Getwd()
	if err != nil {
		return DuplicateResult{}, err
	}
	images, failures, err := scanImages(ctx, dir, workers)
	if err != nil {
		return DuplicateResult{}, err
	}
	slices.SortFunc(images, func(a, b *ImageInfo) int {
		return cmp.Compare(a.Filename, b.Filename)
	})
	slices.SortFunc(failures, func(a, b ScanFailure) int {
		return cmp.Compare(a.Path, b.Path)
	})
	return DuplicateResult{
		Groups:   groupDuplicates(images, maxHammingDistance),
		Scanned:  len(images),
		Failures: failures,
	}, nil
}

func scanImages(ctx context.Context, dir string, workers int) ([]*ImageInfo, []ScanFailure, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, nil, err
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
		return nil, nil, nil
	}
	pathChan := make(chan string, len(paths))
	resultChan := make(chan scanResult, len(paths))
	var wg sync.WaitGroup
	for range workers {
		wg.Go(func() {
			for path := range pathChan {
				select {
				case <-ctx.Done():
					return
				default:
				}
				info, err := processImage(path)
				resultChan <- scanResult{info: info, path: path, err: err}
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
		return nil, nil, ctx.Err()
	}

	var images []*ImageInfo
	var failures []ScanFailure
	for r := range resultChan {
		if r.err != nil {
			failures = append(failures, ScanFailure{Path: r.path, Err: r.err})
			continue
		}
		images = append(images, r.info)
	}
	return images, failures, nil
}

func processImage(path string) (*ImageInfo, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	stat, err := file.Stat()
	if err != nil {
		return nil, err
	}
	img, _, err := image.Decode(file)
	if err != nil {
		return nil, err
	}
	hash, err := goimagehash.PerceptionHash(img)
	if err != nil {
		return nil, err
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
	}, nil
}

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
