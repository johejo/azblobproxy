package azblobproxy_test

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/johejo/azblobproxy"
)

func TestHandler(t *testing.T) {
	accountName := os.Getenv("AZURE_BLOB_STORAGE_ACCOUNT_NAME")
	accountKey := os.Getenv("AZURE_BLOB_STORAGE_ACCOUNT_KEY")
	containerName := os.Getenv("AZURE_BLOB_STORAGE_CONTAINER_NAME")

	tests := []struct {
		name        string
		path        string
		index       string
		notFound    string
		statusCode  int
		contentType string
	}{
		{
			name:        "index",
			path:        "/index.html",
			statusCode:  http.StatusOK,
			contentType: "text/html",
		},
		{
			name:        "fallback index",
			path:        "",
			index:       "index.html",
			statusCode:  http.StatusOK,
			contentType: "text/html",
		},
		{
			name:        "fallback index /",
			path:        "/",
			index:       "index.html",
			statusCode:  http.StatusOK,
			contentType: "text/html",
		},
		{
			name:       "404",
			path:       "/noSuchFile.html",
			statusCode: http.StatusNotFound,
		},
		{
			name:        "fallback 404",
			path:        "/noSuchFile.html",
			notFound:    "index.html",
			statusCode:  http.StatusOK,
			contentType: "text/html",
		},
	}
	for i, tt := range tests {
		name := fmt.Sprintf("%d, %s", i, tt.name)
		t.Run(name, func(t *testing.T) {
			handler := azblobproxy.SimpleHandler(accountName, accountKey, containerName)
			handler.IndexDocumentName = tt.index
			handler.NotFoundDocumentPath = tt.notFound

			ts := httptest.NewServer(handler)
			defer ts.Close()

			req, err := http.NewRequest("GET", ts.URL+tt.path, nil)
			if err != nil {
				t.Fatal(err)
			}
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatal(err)
			}

			if resp.StatusCode != tt.statusCode {
				t.Errorf("unexpected status code: status=%d", resp.StatusCode)
			}
			if contentType := resp.Header.Get("Content-Type"); contentType != tt.contentType {
				t.Errorf("unexpected content type: contentType=%s", contentType)
			}
		})
	}
}

func Example() {
	// An example as a proxy for single page application.

	accountName := os.Getenv("AZURE_BLOB_STORAGE_ACCOUNT_NAME")
	accountKey := os.Getenv("AZURE_BLOB_STORAGE_ACCOUNT_KEY")
	containerName := os.Getenv("AZURE_BLOB_STORAGE_CONTAINER_NAME")

	handler := azblobproxy.SimpleHandler(accountName, accountKey, containerName)
	handler.IndexDocumentName = "index.html"
	handler.NotFoundDocumentPath = "index.html"

	logger := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			log.Println(r.Method, r.URL)
			next.ServeHTTP(w, r)
		})
	}

	mux := http.NewServeMux()
	mux.Handle("/", logger(handler))
	log.Println(http.ListenAndServe("localhost:8080", mux))
}
