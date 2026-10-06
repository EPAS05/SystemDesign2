package grpcsvc

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	commonv1 "github.com/EPAS05/catalog-service/gen/go/common/v1"
	configurationv1 "github.com/EPAS05/catalog-service/gen/go/configuration/v1"
	"github.com/EPAS05/catalog-service/internal/configuration"
	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func Run(port, fallbackPort, dbURL string) {
	if port == "" {
		port = fallbackPort
	}
	if dbURL == "" {
		log.Fatal("DB_URL is required")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Fatalf("pgxpool: %v", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		log.Fatalf("db ping: %v", err)
	}

	repo := configuration.NewRepo(pool)
	svc := configuration.NewService(repo)

	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Fatalf("listen :%s: %v", port, err)
	}

	srv := grpc.NewServer()
	commonv1.RegisterHealthCheckServiceServer(srv, healthService{})
	configurationv1.RegisterBomServiceServer(srv, NewBomService(svc))

	reflection.Register(srv)

	go func() {
		sigCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()
		<-sigCtx.Done()
		srv.GracefulStop()
	}()

	log.Printf("grpc listening on :%s", port)
	if err := srv.Serve(lis); err != nil {
		log.Fatalf("serve: %v", err)
	}
}
