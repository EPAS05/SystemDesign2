package gateway

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestHTTPErrorHandlerFailedPrecondition(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	httpErrorHandler(context.Background(), nil, nil, rec, req, status.Error(codes.FailedPrecondition, "precondition failed"))

	if rec.Code != http.StatusPreconditionFailed {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusPreconditionFailed)
	}
	if got := rec.Header().Get("Content-Type"); got != "application/json; charset=utf-8" {
		t.Fatalf("content type = %q", got)
	}

	var body struct {
		Code    int32  `json:"code"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Code != int32(codes.FailedPrecondition) {
		t.Fatalf("code = %d, want %d", body.Code, int32(codes.FailedPrecondition))
	}
	if body.Message != "precondition failed" {
		t.Fatalf("message = %q", body.Message)
	}
}

func TestHTTPErrorHandlerPassesThrough(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	httpErrorHandler(context.Background(), runtime.NewServeMux(), defaultMarshaler, rec, req, status.Error(codes.NotFound, "missing"))

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}
