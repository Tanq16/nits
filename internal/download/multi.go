package download

import (
	"context"
	"encoding/json/v2"
	"fmt"
	"io"
	"net/http"
	"os"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/tanq16/nits/internal/network"
	"golang.org/x/sync/errgroup"
)

type span struct {
	Start int64 `json:"start"`
	End   int64 `json:"end"`
	Done  int64 `json:"done"`
}

func (s span) length() int64 {
	return s.End - s.Start + 1
}

type chunkState struct {
	Size   int64  `json:"size"`
	Chunks []span `json:"chunks"`
}

func chunked(ctx context.Context, p Plan, client *Client, partPath string, progress io.Writer) error {
	statePath := strings.TrimSuffix(partPath, ".part") + ".state"
	state, f, err := openChunked(partPath, statePath, p.Size, p.Connections)
	if err != nil {
		return err
	}
	defer f.Close()

	done := make([]atomic.Int64, len(state.Chunks))
	var resumed int64
	for i, c := range state.Chunks {
		done[i].Store(c.Done)
		resumed += c.Done
	}
	resumeProgress(progress, resumed)

	save := func() {
		snap := chunkState{Size: state.Size, Chunks: slices.Clone(state.Chunks)}
		for i := range snap.Chunks {
			snap.Chunks[i].Done = done[i].Load()
		}
		err := f.Sync()
		if err == nil {
			err = writeState(statePath, snap)
		}
		if err != nil {
			log.Debug().Err(err).Str("state", statePath).Msg("failed to save download state")
		}
	}

	stop := make(chan struct{})
	var saver sync.WaitGroup
	saver.Go(func() {
		tick := time.Tick(saveInterval)
		for {
			select {
			case <-stop:
				return
			case <-tick:
				save()
			}
		}
	})

	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(p.Connections)
	for i, c := range state.Chunks {
		g.Go(func() error {
			return retry(gctx, done[i].Load, func() error {
				if done[i].Load() >= c.length() {
					return nil
				}
				return fetchSpan(gctx, client, p, f, c, &done[i], progress)
			})
		})
	}
	err = g.Wait()
	close(stop)
	saver.Wait()
	if err != nil {
		save()
		return err
	}
	if err := f.Sync(); err != nil {
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	os.Remove(statePath)
	return nil
}

func openChunked(partPath, statePath string, size int64, connections int) (chunkState, *os.File, error) {
	f, err := os.OpenFile(partPath, os.O_RDWR|os.O_CREATE, 0o644)
	if err != nil {
		return chunkState{}, nil, fmt.Errorf("error opening temp file: %w", err)
	}
	state, ok := readState(statePath, size)
	if fi, err := f.Stat(); ok && err == nil && fi.Size() == size {
		return state, f, nil
	}
	if err := f.Truncate(size); err != nil {
		f.Close()
		return chunkState{}, nil, fmt.Errorf("error allocating temp file: %w", err)
	}
	return chunkState{Size: size, Chunks: splitChunks(size, connections)}, f, nil
}

func readState(path string, size int64) (chunkState, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return chunkState{}, false
	}
	var s chunkState
	if err := json.Unmarshal(data, &s); err != nil || s.Size != size || len(s.Chunks) == 0 {
		return chunkState{}, false
	}
	var next int64
	for _, c := range s.Chunks {
		if c.Start != next || c.End < c.Start || c.Done < 0 || c.Done > c.length() {
			return chunkState{}, false
		}
		next = c.End + 1
	}
	return s, next == size
}

func writeState(path string, s chunkState) error {
	data, err := json.Marshal(s)
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func splitChunks(size int64, connections int) []span {
	n := min(int64(connections), max(1, size/minChunkSize))
	chunkSize := size / n
	chunks := make([]span, 0, n)
	var pos int64
	for i := range n {
		end := pos + chunkSize - 1
		if i == n-1 {
			end = size - 1
		}
		chunks = append(chunks, span{Start: pos, End: end})
		pos = end + 1
	}
	return chunks
}

func fetchSpan(ctx context.Context, client *Client, p Plan, f *os.File, c span, done *atomic.Int64, progress io.Writer) error {
	start := c.Start + done.Load()
	ctx, guard := network.NewStallGuard(ctx, stallTimeout)
	defer guard.Stop()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.URL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", start, c.End))
	if p.validator != "" {
		req.Header.Set("If-Range", p.validator)
	}
	resp, err := client.Do(req)
	if err != nil {
		return guard.Err(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusOK && p.validator != "" {
		return errRemoteChanged
	}
	if resp.StatusCode != http.StatusPartialContent {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	if got := rangeStart(resp.Header.Get("Content-Range")); got != start {
		return fmt.Errorf("server sent a range starting at %d, requested %d", got, start)
	}

	remaining := c.End - start + 1
	w := &chunkWriter{f: f, off: start, done: done, progress: progress}
	n, err := io.Copy(w, guard.Reader(io.LimitReader(resp.Body, remaining)))
	if err != nil {
		return guard.Err(err)
	}
	if n < remaining {
		return fmt.Errorf("short response: got %d of %d bytes", n, remaining)
	}
	return nil
}

type chunkWriter struct {
	f        *os.File
	off      int64
	done     *atomic.Int64
	progress io.Writer
}

func (w *chunkWriter) Write(p []byte) (int, error) {
	n, err := w.f.WriteAt(p, w.off)
	w.off += int64(n)
	w.done.Add(int64(n))
	if w.progress != nil {
		w.progress.Write(p[:n])
	}
	return n, err
}
