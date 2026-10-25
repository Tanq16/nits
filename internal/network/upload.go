package network

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const uploadStallTimeout = 10 * time.Second

var uploadClient = &http.Client{
	Transport: &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		DialContext:           (&net.Dialer{Timeout: 10 * time.Second}).DialContext,
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 30 * time.Second,
	},
}

func Upload(ctx context.Context, serverURL, path string, progress io.Writer) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	fi, err := f.Stat()
	if err != nil {
		return "", err
	}
	target, err := url.JoinPath(serverURL, filepath.Base(path))
	if err != nil {
		return "", fmt.Errorf("invalid server URL: %w", err)
	}

	ctx, guard := NewStallGuard(ctx, uploadStallTimeout)
	defer guard.Stop()
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, target, guard.Reader(io.TeeReader(f, progress)))
	if err != nil {
		return "", err
	}
	req.ContentLength = fi.Size()
	resp, err := uploadClient.Do(req)
	if err != nil {
		return "", guard.Err(err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if err != nil {
		return "", guard.Err(err)
	}
	msg := strings.TrimSpace(string(body))
	if resp.StatusCode != http.StatusCreated {
		if msg == "" || msg == http.StatusText(resp.StatusCode) {
			return "", fmt.Errorf("server returned %s", resp.Status)
		}
		return "", fmt.Errorf("server returned %s: %s", resp.Status, msg)
	}
	return msg, nil
}
