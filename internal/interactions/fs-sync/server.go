package fssync

import (
	"context"
	"crypto/tls"
	"encoding/json/v2"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	u "github.com/tanq16/nits/utils"
)

type ServerConfig struct {
	Port        int
	SyncDir     string
	IgnorePaths string
	EnableTLS   bool
	Mode        string
	DeleteExtra bool
	DryRun      bool
}

type Server struct {
	cfg       ServerConfig
	ignorer   *PathIgnorer
	serveDone chan struct{}
	closeOnce sync.Once
}

func NewServer(cfg ServerConfig) (*Server, error) {
	absDir, err := filepath.Abs(cfg.SyncDir)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve sync directory: %w", err)
	}
	cfg.SyncDir = absDir
	if _, err := os.Stat(cfg.SyncDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("sync directory does not exist: %s", cfg.SyncDir)
	}
	return &Server{
		cfg:       cfg,
		ignorer:   NewPathIgnorer(cfg.IgnorePaths),
		serveDone: make(chan struct{}),
	}, nil
}

func (s *Server) shutdown() {
	s.closeOnce.Do(func() { close(s.serveDone) })
}

func (s *Server) Run() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/mode", s.handleMode)
	mux.HandleFunc("/manifest", s.handleManifest)
	if s.cfg.Mode == "send" {
		mux.HandleFunc("/files", s.handleFiles)
	} else {
		mux.HandleFunc("/upload", s.handleUpload)
	}
	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", s.cfg.Port),
		Handler: mux,
	}
	if s.cfg.EnableTLS {
		tlsConfig, err := s.getTLSConfig()
		if err != nil {
			return fmt.Errorf("failed to get TLS config: %w", err)
		}
		server.TLSConfig = tlsConfig
	}
	serverErrChan := make(chan error, 1)
	go func() {
		var err error
		if s.cfg.EnableTLS {
			err = server.ListenAndServeTLS("", "")
		} else {
			err = server.ListenAndServe()
		}
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error().Err(err).Msg("server error")
			serverErrChan <- err
		}
	}()
	select {
	case <-s.serveDone:
	case err := <-serverErrChan:
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return server.Shutdown(ctx)
}

func (s *Server) handleMode(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.MarshalWrite(w, ModeResponse{Mode: s.cfg.Mode})
}

func (s *Server) handleManifest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	manifest, err := BuildManifest(s.cfg.SyncDir, s.ignorer)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.MarshalWrite(w, ManifestResponse{Files: manifest})
}

func (s *Server) handleFiles(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req FileRequest
	if err := json.UnmarshalRead(r.Body, &req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	var files []FileContent
	for _, path := range req.Paths {
		if s.ignorer.IsIgnored(path) {
			continue
		}
		relPath := filepath.Clean(path)
		if strings.HasPrefix(relPath, "..") || filepath.IsAbs(relPath) {
			log.Warn().Str("path", path).Msg("invalid path")
			continue
		}
		fullPath := filepath.Join(s.cfg.SyncDir, relPath)
		content, err := os.ReadFile(fullPath)
		if err != nil {
			log.Warn().Err(err).Str("path", path).Msg("failed to read file")
			continue
		}
		files = append(files, FileContent{
			Path:    path,
			Content: content,
		})
	}
	w.Header().Set("Content-Type", "application/json")
	json.MarshalWrite(w, FilesResponse{Files: files})
	s.shutdown()
}

func (s *Server) handleUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var uploadReq UploadRequest
	if err := json.UnmarshalRead(r.Body, &uploadReq); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if s.cfg.DryRun {
		for _, file := range uploadReq.Files {
			log.Info().Str("path", file.Path).Msg("dry run")
		}
		if s.cfg.DeleteExtra {
			for _, path := range uploadReq.ToDelete {
				log.Info().Str("path", path).Msg("dry run delete")
			}
		}
		totalCount := len(uploadReq.Files) + len(uploadReq.ToDelete)
		if totalCount == 0 {
			log.Warn().Msg("no files would be synced")
		} else {
			log.Info().Int("count", totalCount).Msg("files would be synced")
		}
		w.WriteHeader(http.StatusOK)
		s.shutdown()
		return
	}

	count := 0
	for _, file := range uploadReq.Files {
		relPath := filepath.Clean(file.Path)
		if strings.HasPrefix(relPath, "..") || filepath.IsAbs(relPath) {
			log.Warn().Str("path", file.Path).Msg("invalid path")
			continue
		}
		fullPath := filepath.Join(s.cfg.SyncDir, relPath)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			log.Error().Err(err).Str("path", file.Path).Msg("failed to create directory")
			continue
		}
		if err := os.WriteFile(fullPath, file.Content, 0644); err != nil {
			log.Error().Err(err).Str("path", file.Path).Msg("failed to write file")
			continue
		}
		log.Info().Str("path", file.Path).Msg("received file")
		count++
	}

	deletedCount := 0
	if s.cfg.DeleteExtra {
		for _, path := range uploadReq.ToDelete {
			relPath := filepath.Clean(path)
			if strings.HasPrefix(relPath, "..") || filepath.IsAbs(relPath) {
				continue
			}
			fullPath := filepath.Join(s.cfg.SyncDir, relPath)
			if err := os.RemoveAll(fullPath); err != nil {
				log.Error().Err(err).Str("path", path).Msg("failed to delete file")
			} else {
				log.Info().Str("path", path).Msg("deleted file")
				deletedCount++
			}
		}
	}

	totalCount := count + deletedCount
	if totalCount == 0 {
		log.Warn().Msg("no files were synced")
	} else {
		log.Info().Int("count", totalCount).Msg("files synced")
	}
	w.WriteHeader(http.StatusOK)
	s.shutdown()
}

func (s *Server) getTLSConfig() (*tls.Config, error) {
	cert, err := u.GenerateSelfSignedCert()
	if err != nil {
		return nil, fmt.Errorf("failed to generate self-signed certificate: %w", err)
	}
	return &tls.Config{
		Certificates: []tls.Certificate{cert},
	}, nil
}

