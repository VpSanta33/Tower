package web

import (
	"io/fs"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
)

func TestServeEmbeddedFile(t *testing.T) {
	files := fstest.MapFS{
		"index.html": {Data: []byte(`<div id="app"></div>`)},
	}
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	serveFile(rec, req, files, "index.html")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if !strings.Contains(rec.Body.String(), `<div id="app">`) {
		t.Fatal("index html was not served")
	}
}

func TestHandlerKeepsAPINotFound(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/missing", nil)
	rec := httptest.NewRecorder()

	Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestFileExists(t *testing.T) {
	files := fstest.MapFS{
		"index.html": {Data: []byte(`<div id="app"></div>`)},
		"js":         {Mode: fs.ModeDir},
	}

	if !fileExists(files, "index.html") {
		t.Fatal("file should exist")
	}
	if fileExists(files, "js") {
		t.Fatal("directory should not be treated as file")
	}
	if fileExists(files, "missing.html") {
		t.Fatal("missing file should not exist")
	}
}
