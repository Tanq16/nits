package download

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	minChunkSize = 1 * 1024 * 1024
	maxFailures  = 5
	stallTimeout = 10 * time.Second
	saveInterval = 2 * time.Second
	tempDirName  = ".nits-temp"
	tempKeyLen   = 16
)

var (
	errRangeNotSupported = errors.New("range requests are not supported")
	errRemoteChanged     = errors.New("remote file changed")
)

var filenameUnsafe = regexp.MustCompile(`[^a-zA-Z0-9_\-\. ]+`)

type Config struct {
	URL         string
	OutputPath  string
	Connections int
	ProxyURL    string
	UserAgent   string
	Headers     map[string]string
}

type remoteInfo struct {
	size           int64
	name           string
	rangeSupported bool
	etag           string
	lastModified   string
}

func (r remoteInfo) validator() string {
	if r.etag != "" && !strings.HasPrefix(r.etag, "W/") {
		return r.etag
	}
	return r.lastModified
}

func (r remoteInfo) key() string {
	sum := sha256.Sum256(fmt.Appendf(nil, "%d|%s|%s", r.size, r.etag, r.lastModified))
	return hex.EncodeToString(sum[:])[:tempKeyLen]
}

type Plan struct {
	URL         string
	OutputPath  string
	Size        int64
	Connections int
	ranged      bool
	validator   string
	modTime     time.Time
	key         string
}

func Prepare(ctx context.Context, cfg Config) (Plan, *Client, error) {
	parsed, err := url.Parse(cfg.URL)
	if err != nil {
		return Plan{}, nil, fmt.Errorf("invalid URL: %w", err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return Plan{}, nil, fmt.Errorf("unsupported scheme: %s", parsed.Scheme)
	}
	cfg.Connections = max(cfg.Connections, 1)

	client, err := NewClient(ClientConfig{
		ProxyURL:    cfg.ProxyURL,
		UserAgent:   cfg.UserAgent,
		Headers:     cfg.Headers,
		Connections: cfg.Connections,
	})
	if err != nil {
		return Plan{}, nil, err
	}

	headBlocked, finalURL, err := probeURL(ctx, client, cfg.URL)
	if err != nil {
		return Plan{}, nil, err
	}
	cfg.URL = finalURL

	info, err := statFile(ctx, client, cfg.URL, headBlocked)
	if err != nil && !errors.Is(err, errRangeNotSupported) {
		return Plan{}, nil, fmt.Errorf("error getting file info: %w", err)
	}

	if cfg.OutputPath == "" {
		cfg.OutputPath = info.name
		if cfg.OutputPath == "" {
			if pu, parseErr := url.Parse(cfg.URL); parseErr == nil {
				pathParts := strings.Split(pu.Path, "/")
				cfg.OutputPath = pathParts[len(pathParts)-1]
			}
		}
		if cfg.OutputPath == "" {
			cfg.OutputPath = "download"
		}
	}

	modTime, _ := http.ParseTime(info.lastModified)
	if existing, statErr := os.Stat(cfg.OutputPath); statErr == nil {
		sameTime := modTime.IsZero() || existing.ModTime().Truncate(time.Second).Equal(modTime)
		if info.size > 0 && existing.Size() == info.size && sameTime {
			return Plan{OutputPath: cfg.OutputPath}, nil, errAlreadyComplete{path: cfg.OutputPath}
		}
		cfg.OutputPath = renewOutputPath(cfg.OutputPath)
	}

	return Plan{
		URL:         cfg.URL,
		OutputPath:  cfg.OutputPath,
		Size:        info.size,
		Connections: cfg.Connections,
		ranged:      info.rangeSupported && info.size > 0,
		validator:   info.validator(),
		modTime:     modTime,
		key:         info.key(),
	}, client, nil
}

func (p Plan) Execute(ctx context.Context, client *Client, progress io.Writer) error {
	tempDir := filepath.Join(filepath.Dir(p.OutputPath), tempDirName)
	if err := os.MkdirAll(tempDir, 0o755); err != nil {
		return fmt.Errorf("error creating temp directory: %w", err)
	}
	base := filepath.Base(p.OutputPath)
	removeStaleTemps(tempDir, base, p.key)
	partPath := filepath.Join(tempDir, base+"."+p.key+".part")

	var err error
	if p.ranged {
		err = chunked(ctx, p, client, partPath, progress)
	} else {
		err = stream(ctx, p, client, partPath, progress)
	}
	if err != nil {
		return err
	}
	if err := os.Rename(partPath, p.OutputPath); err != nil {
		return fmt.Errorf("error moving download into place: %w", err)
	}
	if !p.modTime.IsZero() {
		os.Chtimes(p.OutputPath, p.modTime, p.modTime)
	}
	removeTempDirIfEmpty(tempDir)
	return nil
}

func retry(ctx context.Context, committed func() int64, attempt func() error) error {
	high := committed()
	failures := 0
	for {
		err := attempt()
		if err == nil {
			return nil
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if errors.Is(err, errRemoteChanged) {
			return err
		}
		if c := committed(); c > high {
			high = c
			failures = 0
		} else {
			failures++
		}
		if failures >= maxFailures {
			return fmt.Errorf("no progress after %d attempts: %w", maxFailures, err)
		}
		timer := time.NewTimer(time.Second << failures)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
}

type errAlreadyComplete struct {
	path string
}

func (e errAlreadyComplete) Error() string {
	return e.path + " already exists"
}

func AlreadyComplete(err error) bool {
	_, ok := errors.AsType[errAlreadyComplete](err)
	return ok
}

func AlreadyCompletePath(err error) string {
	if c, ok := errors.AsType[errAlreadyComplete](err); ok {
		return c.path
	}
	return ""
}

func probeURL(ctx context.Context, client *Client, link string) (bool, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, link, nil)
	if err != nil {
		return false, "", fmt.Errorf("error creating request: %w", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return false, "", fmt.Errorf("error checking URL: %w", err)
	}
	resp.Body.Close()

	if resp.Request != nil && resp.Request.URL != nil {
		link = resp.Request.URL.String()
	}
	if resp.StatusCode == http.StatusNotFound {
		return false, "", fmt.Errorf("URL not found (404)")
	}
	if resp.StatusCode < 400 {
		return false, link, nil
	}

	getReq, err := http.NewRequestWithContext(ctx, http.MethodGet, link, nil)
	if err != nil {
		return false, "", fmt.Errorf("server returned error: %d", resp.StatusCode)
	}
	getReq.Header.Set("Range", "bytes=0-0")
	getResp, err := client.Do(getReq)
	if err != nil {
		return false, "", fmt.Errorf("server returned error: %d", resp.StatusCode)
	}
	getResp.Body.Close()
	if getResp.StatusCode >= 400 {
		return false, "", fmt.Errorf("server returned error: %d (GET fallback returned: %d)", resp.StatusCode, getResp.StatusCode)
	}
	if getResp.Request != nil && getResp.Request.URL != nil {
		link = getResp.Request.URL.String()
	}
	return true, link, nil
}

func statFile(ctx context.Context, client *Client, link string, useGET bool) (remoteInfo, error) {
	method := http.MethodHead
	if useGET {
		method = http.MethodGet
	}
	req, err := http.NewRequestWithContext(ctx, method, link, nil)
	if err != nil {
		return remoteInfo{}, err
	}
	if useGET {
		req.Header.Set("Range", "bytes=0-0")
	}
	resp, err := client.Do(req)
	if err != nil {
		return remoteInfo{}, err
	}
	defer resp.Body.Close()

	info := remoteInfo{
		name:         filenameFromDisposition(resp.Header.Get("Content-Disposition")),
		etag:         resp.Header.Get("ETag"),
		lastModified: resp.Header.Get("Last-Modified"),
	}
	acceptRanges := resp.Header.Get("Accept-Ranges")
	info.rangeSupported = acceptRanges == "bytes" || (useGET && resp.StatusCode == http.StatusPartialContent)

	if contentRange := resp.Header.Get("Content-Range"); contentRange != "" {
		if _, after, found := strings.Cut(contentRange, "/"); found && after != "*" {
			size, parseErr := strconv.ParseInt(after, 10, 64)
			if parseErr != nil {
				return info, fmt.Errorf("invalid Content-Range total: %w", parseErr)
			}
			info.size = size
		}
	}
	if info.size <= 0 {
		contentLength := resp.Header.Get("Content-Length")
		if contentLength != "" {
			size, parseErr := strconv.ParseInt(contentLength, 10, 64)
			if parseErr != nil {
				return info, parseErr
			}
			info.size = size
		}
	}
	if !info.rangeSupported {
		return info, errRangeNotSupported
	}
	if info.size <= 0 {
		return info, errors.New("invalid file size reported by server")
	}
	return info, nil
}

func rangeStart(contentRange string) int64 {
	spec, ok := strings.CutPrefix(contentRange, "bytes ")
	if !ok {
		return -1
	}
	first, _, ok := strings.Cut(spec, "-")
	if !ok {
		return -1
	}
	start, err := strconv.ParseInt(first, 10, 64)
	if err != nil {
		return -1
	}
	return start
}

func filenameFromDisposition(header string) string {
	if header == "" {
		return ""
	}
	_, params, err := mime.ParseMediaType(header)
	if err != nil {
		return ""
	}
	if fn := params["filename"]; fn != "" {
		return filenameUnsafe.ReplaceAllString(fn, "_")
	}
	fn := params["filename*"]
	if fn == "" {
		return ""
	}
	if rest, ok := strings.CutPrefix(fn, "UTF-8''"); ok {
		unescaped, _ := url.PathUnescape(rest)
		return filenameUnsafe.ReplaceAllString(unescaped, "_")
	}
	return ""
}

func renewOutputPath(outputPath string) string {
	dir := filepath.Dir(outputPath)
	base := filepath.Base(outputPath)
	ext := filepath.Ext(base)
	name := strings.TrimSuffix(base, ext)
	for index := 1; ; index++ {
		candidate := filepath.Join(dir, fmt.Sprintf("%s-(%d)%s", name, index, ext))
		if _, err := os.Stat(candidate); errors.Is(err, os.ErrNotExist) {
			return candidate
		}
	}
}

func addProgress(w io.Writer, n int64) {
	if w == nil || n == 0 {
		return
	}
	type adder interface{ Add(int64) }
	if a, ok := w.(adder); ok {
		a.Add(n)
	}
}

func resumeProgress(w io.Writer, n int64) {
	if w == nil || n == 0 {
		return
	}
	type resumer interface{ Resume(int64) }
	if r, ok := w.(resumer); ok {
		r.Resume(n)
	}
}

func removeStaleTemps(dir, base, key string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, e := range entries {
		rest, ok := strings.CutPrefix(e.Name(), base+".")
		if !ok {
			continue
		}
		k, kind, ok := strings.Cut(rest, ".")
		if !ok || k == key || len(k) != tempKeyLen {
			continue
		}
		if _, err := hex.DecodeString(k); err != nil {
			continue
		}
		if kind == "part" || kind == "state" || kind == "state.tmp" {
			os.Remove(filepath.Join(dir, e.Name()))
		}
	}
}

func removeTempDirIfEmpty(dir string) {
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) > 0 {
		return
	}
	os.Remove(dir)
}

func ParseHeaders(headers []string) map[string]string {
	result := make(map[string]string)
	for _, header := range headers {
		key, value, ok := strings.Cut(header, ":")
		if !ok {
			continue
		}
		result[strings.TrimSpace(key)] = strings.TrimSpace(value)
	}
	return result
}
