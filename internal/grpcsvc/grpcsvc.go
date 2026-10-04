package grpcsvc

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	commonv1 "github.com/EPAS05/catalog-service/gen/go/common/v1"
	"google.golang.org/grpc"
)

type healthService struct {
	commonv1.UnimplementedHealthCheckServiceServer
}

func (healthService) Ping(_ context.Context, _ *commonv1.PingRequest) (*commonv1.PingResponse, error) {
	return &commonv1.PingResponse{Message: "pong"}, nil
}

func Run(port, fallbackPort string) {
	if port == "" {
		port = fallbackPort
	}

	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Fatalf("listen :%s: %v", port, err)
	}

	srv := grpc.NewServer()
	commonv1.RegisterHealthCheckServiceServer(srv, healthService{})

	go func() {
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()
		<-ctx.Done()
		srv.GracefulStop()
	}()

	log.Printf("grpc listening on :%s", port)
	if err := srv.Serve(lis); err != nil {
		log.Fatalf("serve: %v", err)
	}
}