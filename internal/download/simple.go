package download

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/tanq16/nits/internal/network"
)

func stream(ctx context.Context, p Plan, client *Client, partPath string, progress io.Writer) error {
	size := func() int64 {
		fi, err := os.Stat(partPath)
		if err != nil {
			return 0
		}
		return fi.Size()
	}
	reported := size()
	resumeProgress(progress, reported)
	return retry(ctx, size, func() error {
		return streamAttempt(ctx, client, p, partPath, progress, &reported)
	})
}

func streamAttempt(ctx context.Context, client *Client, p Plan, partPath string, progress io.Writer, reported *int64) error {
	var offset int64
	if fi, err := os.Stat(partPath); err == nil {
		offset = fi.Size()
	}

	ctx, guard := network.NewStallGuard(ctx, stallTimeout)
	defer guard.Stop()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.URL, nil)
	if err != nil {
		return fmt.Errorf("error creating GET request: %w", err)
	}
	if offset > 0 {
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-", offset))
		if p.validator != "" {
			req.Header.Set("If-Range", p.validator)
		}
	}
	resp, err := client.Do(req)
	if err != nil {
		return guard.Err(fmt.Errorf("error executing GET request: %w", err))
	}
	defer resp.Body.Close()

	switch {
	case resp.StatusCode == http.StatusPartialContent && offset > 0 && rangeStart(resp.Header.Get("Content-Range")) == offset:
	case resp.StatusCode == http.StatusOK, resp.StatusCode == http.StatusPartialContent && rangeStart(resp.Header.Get("Content-Range")) == 0:
		offset = 0
	default:
		return fmt.Errorf("unexpected response: %d %s", resp.StatusCode, resp.Header.Get("Content-Range"))
	}

	flags := os.O_CREATE | os.O_WRONLY | os.O_APPEND
	if offset == 0 {
		flags = os.O_CREATE | os.O_WRONLY | os.O_TRUNC
		addProgress(progress, -*reported)
		*reported = 0
	}
	out, err := os.OpenFile(partPath, flags, 0o644)
	if err != nil {
		return fmt.Errorf("error opening temp file: %w", err)
	}
	defer out.Close()

	if _, err := io.Copy(io.MultiWriter(out, countingWriter{w: progress, n: reported}), guard.Reader(resp.Body)); err != nil {
		return guard.Err(fmt.Errorf("error reading response body: %w", err))
	}
	if err := out.Sync(); err != nil {
		return err
	}
	if p.Size > 0 && *reported != p.Size {
		return fmt.Errorf("short response: got %d of %d bytes", *reported, p.Size)
	}
	return nil
}

type countingWriter struct {
	w io.Writer
	n *int64
}

func (c countingWriter) Write(p []byte) (int, error) {
	if c.w != nil {
		if _, err := c.w.Write(p); err != nil {
			return 0, err
		}
	}
	if c.n != nil {
		*c.n += int64(len(p))
	}
	return len(p), nil
}
