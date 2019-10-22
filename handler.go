package azblobproxy

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/Azure/azure-storage-blob-go/azblob"
)

// Handler describes azblob proxy as http.Handler.
type Handler struct {
	// refs https://godoc.org/github.com/Azure/azure-storage-blob-go/azblob#BlobURL.Download
	Offset               int64
	Count                int64
	ContainerURL         azblob.ContainerURL
	BlobAccessConditions azblob.BlobAccessConditions
	RangeGetContentMD5   bool
	RetryReaderOptions   azblob.RetryReaderOptions

	IndexDocumentName    string // Defines files that will be used as an index.
	NotFoundDocumentPath string // Defines files that will be used as 404 (for single page application).
}

// SimpleHandler returns a simple azblobproxy Handler.
// This is sufficient for most cases.
func SimpleHandler(accountName, accountKey, containerName string) *Handler {
	c, err := newContainerURL(accountName, accountKey, containerName)
	if err != nil {
		panic(err)
	}
	return &Handler{Offset: 0, Count: 0, ContainerURL: *c}
}

func newContainerURL(accountName, accountKey, containerName string) (*azblob.ContainerURL, error) {
	credential, err := azblob.NewSharedKeyCredential(accountName, accountKey)
	if err != nil {
		return nil, err
	}
	cURL, err := url.Parse(fmt.Sprintf("https://%s.blob.core.windows.net/%s", accountName, containerName))
	if err != nil {
		return nil, err
	}
	p := azblob.NewPipeline(credential, azblob.PipelineOptions{})
	containerURL := azblob.NewContainerURL(*cURL, p)
	return &containerURL, nil
}

// ServeHTTP serves http.Handler.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	blobName := strings.TrimPrefix(r.URL.String(), "/")
	if blobName == "" && h.IndexDocumentName != "" {
		blobName = h.IndexDocumentName
	}

	ctx := r.Context()
	resp, err := h.download(ctx, blobName)
	if err != nil {
		var storageErr azblob.StorageError
		if errors.As(err, &storageErr) {
			switch storageErr.ServiceCode() {
			case azblob.ServiceCodeBlobNotFound:
				h.tryNotFound(ctx, w)
				return
			}
			h.handleBlobDownloadError(w, storageErr)
			return
		}
		h.handleUnexpectedError(w, err)
		return
	}
	h.copyResp(resp, w)
}

func (h *Handler) tryNotFound(ctx context.Context, w http.ResponseWriter) {
	if h.NotFoundDocumentPath == "" {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	resp, err := h.download(ctx, h.NotFoundDocumentPath)
	if err != nil {
		var storageErr azblob.StorageError
		if errors.As(err, &storageErr) {
			switch storageErr.ServiceCode() {
			case azblob.ServiceCodeBlobNotFound:
				w.WriteHeader(http.StatusNotFound)
				return
			}
			h.handleBlobDownloadError(w, storageErr)
			return
		}
		h.handleUnexpectedError(w, err)
		return
	}
	h.copyResp(resp, w)
}

func (h *Handler) download(ctx context.Context, blobName string) (*azblob.DownloadResponse, error) {
	blobURL := h.ContainerURL.NewBlockBlobURL(blobName)
	return blobURL.Download(ctx, h.Offset, h.Count, h.BlobAccessConditions, h.RangeGetContentMD5)
}

func (h *Handler) handleBlobDownloadError(w http.ResponseWriter, storageErr azblob.StorageError) {
	log.Printf("blob download error: %v", storageErr)
	w.WriteHeader(storageErr.Response().StatusCode)
}

func (h *Handler) handleUnexpectedError(w http.ResponseWriter, err error) {
	log.Printf("unexpected error: %v", err)
	w.WriteHeader(http.StatusInternalServerError)
}

func (h *Handler) copyResp(resp *azblob.DownloadResponse, w http.ResponseWriter) {
	w.Header().Set("Content-Type", resp.ContentType())
	if _, err := io.Copy(w, resp.Body(h.RetryReaderOptions)); err != nil {
		log.Printf("copy resp error: %v", err)
	}
}
