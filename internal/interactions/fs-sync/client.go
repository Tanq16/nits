package fssync

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

type ClientCallbacks struct {
	OnInfo        func(msg string)
	OnGeneric     func(msg string)
	OnItemSuccess func(msg string)
	OnWarn        func(msg string, err error)
	OnSuccess     func(msg string)
	OnError       func(msg string, err error)
}

func (cb ClientCallbacks) info(msg string) {
	if cb.OnInfo != nil {
		cb.OnInfo(msg)
	}
}

func (cb ClientCallbacks) generic(msg string) {
	if cb.OnGeneric != nil {
		cb.OnGeneric(msg)
	}
}

func (cb ClientCallbacks) itemSuccess(msg string) {
	if cb.OnItemSuccess != nil {
		cb.OnItemSuccess(msg)
	}
}

func (cb ClientCallbacks) warn(msg string, err error) {
	if cb.OnWarn != nil {
		cb.OnWarn(msg, err)
	}
}

func (cb ClientCallbacks) success(msg string) {
	if cb.OnSuccess != nil {
		cb.OnSuccess(msg)
	}
}

func (cb ClientCallbacks) err(msg string, err error) {
	if cb.OnError != nil {
		cb.OnError(msg, err)
	}
}

type ClientConfig struct {
	ServerAddr  string
	SyncDir     string
	DeleteExtra bool
	Insecure    bool
	DryRun      bool
	IgnorePaths string
}

type Client struct {
	cfg        ClientConfig
	httpClient *http.Client
	ignorer    *PathIgnorer
}

func NewClient(cfg ClientConfig) (*Client, error) {
	absDir, err := filepath.Abs(cfg.SyncDir)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve sync directory: %w", err)
	}
	cfg.SyncDir = absDir
	if err := os.MkdirAll(cfg.SyncDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create sync directory: %w", err)
	}
	transport := &http.Transport{}
	if cfg.Insecure {
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}
	return &Client{
		cfg: cfg,
		httpClient: &http.Client{
			Timeout:   5 * time.Minute,
			Transport: transport,
		},
		ignorer: NewPathIgnorer(cfg.IgnorePaths),
	}, nil
}

func (c *Client) Run(cb ClientCallbacks) error {
	mode, err := c.fetchMode()
	if err != nil {
		return fmt.Errorf("failed to detect server mode: %w", err)
	}
	cb.info(fmt.Sprintf("Server mode: %s", mode))
	switch mode {
	case "send":
		return c.pullFromServer(cb)
	case "receive":
		return c.pushToServer(cb)
	default:
		return fmt.Errorf("unknown server mode: %s", mode)
	}
}

func (c *Client) fetchMode() (string, error) {
	resp, err := c.httpClient.Get(c.cfg.ServerAddr + "/mode")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("server returned %d", resp.StatusCode)
	}
	var modeResp ModeResponse
	if err := json.NewDecoder(resp.Body).Decode(&modeResp); err != nil {
		return "", err
	}
	return modeResp.Mode, nil
}

func (c *Client) pullFromServer(cb ClientCallbacks) error {
	serverManifest, err := c.fetchManifest()
	if err != nil {
		return fmt.Errorf("failed to fetch manifest: %w", err)
	}

	localManifest, _ := BuildManifest(c.cfg.SyncDir, c.ignorer)

	filteredServer := make(map[string]string, len(serverManifest))
	for path, hash := range serverManifest {
		if !c.ignorer.IsIgnored(path) {
			filteredServer[path] = hash
		}
	}

	toRequest, toDelete := c.compareManifests(filteredServer, localManifest)
	if c.cfg.DryRun {
		for _, path := range toRequest {
			cb.generic(fmt.Sprintf("Dry Run: %s", path))
		}
		if c.cfg.DeleteExtra {
			for _, path := range toDelete {
				cb.generic(fmt.Sprintf("Dry Run (delete): %s", path))
			}
		}
		totalCount := len(toRequest)
		if c.cfg.DeleteExtra {
			totalCount += len(toDelete)
		}
		if totalCount == 0 {
			cb.warn("no files would be synced", nil)
		} else {
			cb.success(fmt.Sprintf("%d file(s) would be synced", totalCount))
		}
		// Empty /files POST tells the one-shot server to shut down.
		_, err = c.fetchFiles(nil, cb)
		return err
	}

	syncedCount := 0
	if len(toRequest) > 0 {
		syncedCount, err = c.fetchFiles(toRequest, cb)
		if err != nil {
			return fmt.Errorf("failed to fetch files: %w", err)
		}
	} else {
		if _, err = c.fetchFiles(nil, cb); err != nil {
			return fmt.Errorf("failed to signal server done: %w", err)
		}
	}
	deletedCount := 0
	if c.cfg.DeleteExtra && len(toDelete) > 0 {
		deletedCount, err = c.deleteLocalFiles(toDelete, cb)
		if err != nil {
			return fmt.Errorf("failed to delete files: %w", err)
		}
	}
	totalCount := syncedCount + deletedCount
	if totalCount == 0 {
		cb.warn("no files were synced", nil)
	} else {
		cb.success(fmt.Sprintf("%d file(s) synced", totalCount))
	}
	return nil
}

func (c *Client) pushToServer(cb ClientCallbacks) error {
	serverManifest, err := c.fetchManifest()
	if err != nil {
		return fmt.Errorf("failed to fetch server manifest: %w", err)
	}

	localManifest, err := BuildManifest(c.cfg.SyncDir, c.ignorer)
	if err != nil {
		return fmt.Errorf("failed to build local manifest: %w", err)
	}

	var needed []string
	for path, localHash := range localManifest {
		if serverHash, exists := serverManifest[path]; !exists || serverHash != localHash {
			needed = append(needed, path)
		}
	}

	var toDelete []string
	for path := range serverManifest {
		if _, exists := localManifest[path]; !exists {
			toDelete = append(toDelete, path)
		}
	}

	var files []FileContent
	for _, path := range needed {
		fullPath := filepath.Join(c.cfg.SyncDir, path)
		content, err := os.ReadFile(fullPath)
		if err != nil {
			cb.warn(fmt.Sprintf("Failed to read %s", path), err)
			continue
		}
		files = append(files, FileContent{Path: path, Content: content})
	}

	// Always POST /upload (even empty) so the one-shot server shuts down.
	uploadReq := UploadRequest{
		Files:    files,
		ToDelete: toDelete,
	}
	reqBody, _ := json.Marshal(uploadReq)
	resp, err := c.httpClient.Post(
		c.cfg.ServerAddr+"/upload",
		"application/json",
		bytes.NewReader(reqBody),
	)
	if err != nil {
		return fmt.Errorf("failed to upload files: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned %d", resp.StatusCode)
	}

	if len(files) == 0 && len(toDelete) == 0 {
		cb.warn("no files to send", nil)
		return nil
	}
	for _, file := range files {
		cb.itemSuccess(fmt.Sprintf("Sent: %s", file.Path))
	}
	cb.success(fmt.Sprintf("%d file(s) sent", len(files)))
	return nil
}

func (c *Client) fetchManifest() (map[string]string, error) {
	resp, err := c.httpClient.Get(c.cfg.ServerAddr + "/manifest")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned %d", resp.StatusCode)
	}
	var manifest ManifestResponse
	if err := json.NewDecoder(resp.Body).Decode(&manifest); err != nil {
		return nil, err
	}
	return manifest.Files, nil
}

func (c *Client) fetchFiles(paths []string, cb ClientCallbacks) (int, error) {
	reqBody, _ := json.Marshal(FileRequest{Paths: paths})
	resp, err := c.httpClient.Post(
		c.cfg.ServerAddr+"/files",
		"application/json",
		bytes.NewReader(reqBody),
	)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("server returned %d", resp.StatusCode)
	}

	var filesResp FilesResponse
	if err := json.NewDecoder(resp.Body).Decode(&filesResp); err != nil {
		return 0, err
	}
	count := 0
	for _, file := range filesResp.Files {
		fullPath := filepath.Join(c.cfg.SyncDir, file.Path)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			return count, fmt.Errorf("failed to create directory for %s: %w", file.Path, err)
		}
		if err := os.WriteFile(fullPath, file.Content, 0644); err != nil {
			return count, fmt.Errorf("failed to write file %s: %w", file.Path, err)
		}
		cb.itemSuccess(fmt.Sprintf("Synced: %s", file.Path))
		count++
	}
	return count, nil
}

func (c *Client) compareManifests(server, local map[string]string) (toRequest, toDelete []string) {
	for path, serverHash := range server {
		if localHash, exists := local[path]; !exists || localHash != serverHash {
			toRequest = append(toRequest, path)
		}
	}
	for path := range local {
		if _, exists := server[path]; !exists {
			toDelete = append(toDelete, path)
		}
	}
	return
}

func (c *Client) deleteLocalFiles(paths []string, cb ClientCallbacks) (int, error) {
	count := 0
	for _, path := range paths {
		fullPath := filepath.Join(c.cfg.SyncDir, path)
		if err := os.RemoveAll(fullPath); err != nil {
			cb.err(fmt.Sprintf("Failed to delete %s", path), err)
		} else {
			cb.itemSuccess(fmt.Sprintf("Deleted: %s", path))
			count++
		}
	}
	return count, nil
}

