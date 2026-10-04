package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	commonv1 "github.com/EPAS05/catalog-service/gen/go/common/v1"
	"github.com/gorilla/mux"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type target struct {
	name     string
	envKey   string
	fallback string
}

type caller struct {
	name string
	api  commonv1.HealthCheckServiceClient
}

func main() {
	targets := []target{
		{"catalog", "CATALOG_SERVICE_ADDR", "localhost:8081"},
		{"configuration", "CONFIGURATION_SERVICE_ADDR", "localhost:8082"},
	}

	callers := make([]caller, 0, len(targets))
	for _, t := range targets {
		addr := os.Getenv(t.envKey)
		if addr == "" {
			addr = t.fallback
		}
		conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			log.Fatalf("dial %s (%s): %v", t.name, addr, err)
		}
		callers = append(callers, caller{t.name, commonv1.NewHealthCheckServiceClient(conn)})
	}

	router := mux.NewRouter()
	router.HandleFunc("/v1/ping", pingHandler(callers)).Methods(http.MethodGet)

	port := os.Getenv("HTTP_PORT")
	if port == "" {
		port = "8080"
	}

	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
	}()

	log.Printf("gateway http listening on :%s", port)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("listen: %v", err)
	}
}

func pingHandler(callers []caller) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		msg := r.URL.Query().Get("message")
		log.Printf("ping: message=%q, fan-out to %d services", msg, len(callers))

		req := &commonv1.PingRequest{Message: msg}
		out := make(map[string]string, len(callers))
		for _, c := range callers {
			resp, err := c.api.Ping(ctx, req)
			if err != nil {
				out[c.name] = "error: " + err.Error()
				continue
			}
			out[c.name] = resp.GetMessage()
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(out)
	}
}
