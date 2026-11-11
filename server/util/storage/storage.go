package storage

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

var (
	ErrObjectNotFound = errors.New("object not found")
)

type IStorageService interface {
	Upload(key string, reader io.Reader, contentType string) error
	Get(key string) (io.ReadCloser, string, error)
	Delete(key string) error
}

type SeaweedStorageService struct {
	endpoint   string
	httpClient *http.Client
}

func NewSeaweedStorageService(endpoint string) IStorageService {
	if endpoint == "" {
		endpoint = "http://localhost:8888"
	}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.ResponseHeaderTimeout = 30 * time.Second
	transport.IdleConnTimeout = 30 * time.Second
	return &SeaweedStorageService{
		endpoint: endpoint,
		httpClient: &http.Client{
			Transport: transport,
		},
	}
}

func (s *SeaweedStorageService) buildURL(key string) (string, error) {
	parts := strings.Split(key, "/")
	cleanParts := make([]string, 0, len(parts))
	for _, p := range parts {
		if p != "" {
			cleanParts = append(cleanParts, filepath.Base(filepath.Clean(p)))
		}
	}
	return url.JoinPath(s.endpoint, cleanParts...)
}

func (s *SeaweedStorageService) Upload(key string, reader io.Reader, contentType string) error {
	reqURL, err := s.buildURL(key)
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPut, reqURL, reader)
	if err != nil {
		return err
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("seaweedfs upload error: status %d", resp.StatusCode)
	}
	return nil
}

func (s *SeaweedStorageService) Get(key string) (io.ReadCloser, string, error) {
	reqURL, err := s.buildURL(key)
	if err != nil {
		return nil, "", err
	}
	req, err := http.NewRequest(http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, "", err
	}
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, "", err
	}
	if resp.StatusCode == http.StatusNotFound {
		resp.Body.Close()
		return nil, "", ErrObjectNotFound
	}
	if resp.StatusCode >= 400 {
		resp.Body.Close()
		return nil, "", fmt.Errorf("seaweedfs get error: status %d", resp.StatusCode)
	}
	return resp.Body, resp.Header.Get("Content-Type"), nil
}

func (s *SeaweedStorageService) Delete(key string) error {
	reqURL, err := s.buildURL(key)
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodDelete, reqURL, nil)
	if err != nil {
		return err
	}
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusNotFound && resp.StatusCode != http.StatusAccepted {
		return fmt.Errorf("seaweedfs delete error: status %d", resp.StatusCode)
	}
	return nil
}

// MemoryStorageService is an in-memory implementation of IStorageService for testing.
type MemoryStorageService struct {
	mu    sync.RWMutex
	files map[string][]byte
	types map[string]string
}

func NewMemoryStorageService() *MemoryStorageService {
	return &MemoryStorageService{
		files: make(map[string][]byte),
		types: make(map[string]string),
	}
}

func (m *MemoryStorageService) Upload(key string, reader io.Reader, contentType string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	data, err := io.ReadAll(reader)
	if err != nil {
		return err
	}
	m.files[key] = data
	m.types[key] = contentType
	return nil
}

func (m *MemoryStorageService) Get(key string) (io.ReadCloser, string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	data, ok := m.files[key]
	if !ok {
		return nil, "", ErrObjectNotFound
	}
	return io.NopCloser(bytes.NewReader(data)), m.types[key], nil
}

func (m *MemoryStorageService) Delete(key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.files, key)
	delete(m.types, key)
	return nil
}
