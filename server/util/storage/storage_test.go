package storage

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMemoryStorageService(t *testing.T) {
	svc := NewMemoryStorageService()

	// Upload
	data := "hello world content"
	err := svc.Upload("test/doc.txt", strings.NewReader(data), "text/plain")
	assert.NoError(t, err)

	// Get
	rc, ct, err := svc.Get("test/doc.txt")
	assert.NoError(t, err)
	defer rc.Close()
	bytesRead, _ := io.ReadAll(rc)
	assert.Equal(t, data, string(bytesRead))
	assert.Equal(t, "text/plain", ct)

	// Get Not Found
	_, _, err = svc.Get("nonexistent")
	assert.ErrorIs(t, err, ErrObjectNotFound)

	// Delete
	err = svc.Delete("test/doc.txt")
	assert.NoError(t, err)

	_, _, err = svc.Get("test/doc.txt")
	assert.ErrorIs(t, err, ErrObjectNotFound)
}

func TestSeaweedStorageService(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPut:
			if strings.Contains(r.URL.Path, "fail") {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusCreated)
		case http.MethodGet:
			if strings.Contains(r.URL.Path, "notfound") {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			if strings.Contains(r.URL.Path, "fail") {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/pdf")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("fake pdf content"))
		case http.MethodDelete:
			if strings.Contains(r.URL.Path, "fail") {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusNoContent)
		}
	}))
	defer ts.Close()

	svc := NewSeaweedStorageService(ts.URL)
	assert.NotNil(t, svc)

	// Default constructor
	defaultSvc := NewSeaweedStorageService("")
	assert.NotNil(t, defaultSvc)

	// Upload success & fail
	err := svc.Upload("file.pdf", bytes.NewReader([]byte("test")), "application/pdf")
	assert.NoError(t, err)

	err = svc.Upload("fail.pdf", bytes.NewReader([]byte("test")), "application/pdf")
	assert.Error(t, err)

	// Get success, notfound, and fail
	rc, ct, err := svc.Get("file.pdf")
	assert.NoError(t, err)
	assert.Equal(t, "application/pdf", ct)
	_ = rc.Close()

	_, _, err = svc.Get("notfound.pdf")
	assert.ErrorIs(t, err, ErrObjectNotFound)

	_, _, err = svc.Get("fail.pdf")
	assert.Error(t, err)

	// Delete success & fail
	err = svc.Delete("file.pdf")
	assert.NoError(t, err)

	err = svc.Delete("fail.pdf")
	assert.Error(t, err)

	// Safe URL building with query string or path traversal characters
	err = svc.Upload("content/../secret.txt?recursive=true", bytes.NewReader([]byte("test")), "text/plain")
	assert.NoError(t, err)
	assert.NoError(t, err)

	err = svc.Delete("fail.pdf")
	assert.Error(t, err)

	// Invalid URL error handling
	badSvc := NewSeaweedStorageService("http://\x7f")
	assert.Error(t, badSvc.Upload("file.pdf", bytes.NewReader([]byte("test")), "text/plain"))
	_, _, err = badSvc.Get("file.pdf")
	assert.Error(t, err)
	assert.Error(t, badSvc.Delete("file.pdf"))
}
