package handlers

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strconv"

	grpcruntime "github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/n-r-w/ammo-collector/internal/entity"
	"github.com/n-r-w/ctxlog"
)

type HTTPHandlers struct {
	resultGetter IResultGetter
}

// NewHTTPHandlers creates a new HTTPHandlers instance.
func NewHTTPHandlers() *HTTPHandlers {
	return &HTTPHandlers{}
}

// SetResultGetter sets the result getter.
func (h *HTTPHandlers) SetResultGetter(resultGetter IResultGetter) {
	h.resultGetter = resultGetter
}

// GetHTTPEndpoints registers HTTP endpoints for the handlers.
func (h *HTTPHandlers) GetHTTPEndpoints(ctx context.Context, mux *grpcruntime.ServeMux) error {
	return h.handleResultHTTP(ctx, mux)
}

// handleResultHTTP handles the result request.
func (h *HTTPHandlers) handleResultHTTP(ctx context.Context, mux *grpcruntime.ServeMux) error {
	return mux.HandlePath("GET", "/v1/collections/{collection_id}/result",
		func(w http.ResponseWriter, r *http.Request, pathParams map[string]string) {
			ctxlog.Debug(ctx, "handling GET /v1/collections/{collection_id}/result",
				slog.String("collection_id", pathParams["collection_id"]),
			)

			collectionID, err := strconv.ParseInt(pathParams["collection_id"], 10, 64)
			if err != nil {
				http.Error(w, fmt.Sprintf("invalid collection ID: %v", err), http.StatusBadRequest)
				return
			}

			h.getResultHTTP(ctx, w, entity.CollectionID(collectionID))
		},
	)
}

// getResultHTTP returns the result of a collection as a stream of bytes.
func (h *HTTPHandlers) getResultHTTP(
	ctx context.Context, w http.ResponseWriter, collectionID entity.CollectionID,
) {
	// save all chunks to temporary file
	// Get the result chunks channel from the result getter
	resultChan, err := h.resultGetter.GetResult(ctx, collectionID)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to get result: %v", err), http.StatusNotFound)
		return
	}

	tmpFile, err := os.CreateTemp("", fmt.Sprintf("result-%d-*.zip", collectionID))
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to create temp file: %v", err), http.StatusInternalServerError)
		return
	}
	defer func() { _ = os.Remove(tmpFile.Name()) }()

	for chunk := range resultChan {
		if chunk.Err != nil {
			http.Error(w, fmt.Sprintf("failed to get result chunk: %v", chunk.Err), http.StatusInternalServerError)
			return
		}

		// Write the chunk to the temporary file
		if _, err = tmpFile.Write(chunk.Data); err != nil {
			http.Error(w, fmt.Sprintf("failed to write chunk to temp file: %v", err), http.StatusInternalServerError)
			return
		}
	}

	if err = tmpFile.Close(); err != nil {
		http.Error(w, fmt.Sprintf("failed to close temp file: %v", err), http.StatusInternalServerError)
		return
	}

	fileReader, err := os.Open(tmpFile.Name())
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to open temp file: %v", err), http.StatusInternalServerError)
		return
	}
	defer ctxlog.CloseError(ctx, fileReader)

	fileInfo, err := fileReader.Stat()
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to stat temp file: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Disposition", "attachment; filename="+fmt.Sprintf("result-%d.zip", collectionID))
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Length", strconv.FormatInt(fileInfo.Size(), 10))

	_, err = io.Copy(w, fileReader)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error while copying file: %v", err), http.StatusInternalServerError)
		return
	}
}
