package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestEmbedFileServer_ServesFile(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "widget.html"), []byte("<html>ok</html>"), 0o644); err != nil {
		t.Fatalf("write widget: %v", err)
	}

	h := embedFileServer(dir)
	req := httptest.NewRequest(http.MethodGet, "/embed/widget.html", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 body=%q", rec.Code, rec.Body.String())
	}
	if got := rec.Body.String(); got != "<html>ok</html>" {
		t.Fatalf("body = %q", got)
	}
}

func TestEmbedFileServer_DirectoryListing404(t *testing.T) {
	dir := t.TempDir()
	h := embedFileServer(dir)
	req := httptest.NewRequest(http.MethodGet, "/embed/", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}

func TestEmbedFileServer_MissingFile404(t *testing.T) {
	dir := t.TempDir()
	h := embedFileServer(dir)
	req := httptest.NewRequest(http.MethodGet, "/embed/nope.js", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}
