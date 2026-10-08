package gateway

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	commonv1 "github.com/EPAS05/catalog-service/gen/go/common/v1"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
)

type pingTarget struct {
	name string
	api  commonv1.HealthCheckServiceClient
}

func pingHandler(targets []pingTarget) runtime.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request, _ map[string]string) {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		msg := r.URL.Query().Get("message")
		log.Printf("ping: message=%q, fan-out to %d services", msg, len(targets))

		req := &commonv1.PingRequest{Message: msg}
		out := make(map[string]string, len(targets))
		for _, t := range targets {
			resp, err := t.api.Ping(ctx, req)
			if err != nil {
				out[t.name] = "error: " + err.Error()
				continue
			}
			out[t.name] = resp.GetMessage()
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(out)
	}
}
