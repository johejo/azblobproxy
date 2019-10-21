# azblobproxy

Azure Blob Storage Proxy

[![GitHub Actions](https://github.com/johejo/azblobproxy/workflows/Go%20Test/badge.svg)](https://github.com/johejo/azblobproxy/workflows/go-test)
[![GoDoc](https://godoc.org/github.com/johejo/azblobproxy?status.svg)](https://godoc.org/github.com/johejo/azblobproxy)

## Example

```go
package main

import (
	"log"
	"net/http"
	"os"

	"github.com/johejo/azblobproxy"
)

// An example as a proxy for single page application.
func main() {
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
```
