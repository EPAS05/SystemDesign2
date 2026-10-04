package main

import (
	"os"

	"github.com/EPAS05/catalog-service/internal/grpcsvc"
)

func main() {
	grpcsvc.Run(os.Getenv("GRPC_PORT"), "8082", os.Getenv("DB_URL"))
}
