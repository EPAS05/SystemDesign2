package gateway

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/encoding/protojson"

	catalogv1 "github.com/EPAS05/catalog-service/gen/go/catalog/v1"
	commonv1 "github.com/EPAS05/catalog-service/gen/go/common/v1"
	configurationv1 "github.com/EPAS05/catalog-service/gen/go/configuration/v1"
)

type charsetMarshaler struct {
	runtime.Marshaler
}

func (*charsetMarshaler) ContentType(interface{}) string {
	return "application/json; charset=utf-8"
}

var defaultMarshaler = &charsetMarshaler{
	Marshaler: &runtime.HTTPBodyMarshaler{
		Marshaler: &runtime.JSONPb{
			MarshalOptions: protojson.MarshalOptions{EmitUnpopulated: true},
		},
	},
}

func Run(httpPort, catalogAddr, configAddr string) {
	if httpPort == "" {
		httpPort = "8080"
	}
	if catalogAddr == "" {
		log.Fatal("CATALOG_SERVICE_ADDR is required")
	}
	if configAddr == "" {
		log.Fatal("CONFIGURATION_SERVICE_ADDR is required")
	}

	ctx := context.Background()
	mux := runtime.NewServeMux(runtime.WithMarshalerOption(runtime.MIMEWildcard, defaultMarshaler))

	catalogConn, err := grpc.NewClient(catalogAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("dial catalog: %v", err)
	}
	defer catalogConn.Close()
	catalogConn.Connect()

	if err := catalogv1.RegisterNodeServiceHandler(ctx, mux, catalogConn); err != nil {
		log.Fatalf("register node handler: %v", err)
	}
	if err := catalogv1.RegisterProductServiceHandler(ctx, mux, catalogConn); err != nil {
		log.Fatalf("register product handler: %v", err)
	}
	if err := catalogv1.RegisterComponentServiceHandler(ctx, mux, catalogConn); err != nil {
		log.Fatalf("register component handler: %v", err)
	}

	configConn, err := grpc.NewClient(configAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("dial configuration: %v", err)
	}
	defer configConn.Close()
	configConn.Connect()

	if err := configurationv1.RegisterBomServiceHandler(ctx, mux, configConn); err != nil {
		log.Fatalf("register bom handler: %v", err)
	}

	targets := []pingTarget{
		{"catalog", commonv1.NewHealthCheckServiceClient(catalogConn)},
		{"configuration", commonv1.NewHealthCheckServiceClient(configConn)},
	}

	if err := mux.HandlePath("GET", "/v1/ping", pingHandler(targets)); err != nil {
		log.Fatalf("register ping handler: %v", err)
	}

	srv := &http.Server{
		Addr:              ":" + httpPort,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		sigCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()
		<-sigCtx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		srv.Shutdown(shutdownCtx)
	}()

	log.Printf("http listening on :%s", httpPort)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("serve: %v", err)
	}
}
