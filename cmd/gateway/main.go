package main

import (
	"os"

	"github.com/EPAS05/catalog-service/internal/gateway"
)

func main() {
	gateway.Run(
		os.Getenv("HTTP_PORT"),
		os.Getenv("CATALOG_SERVICE_ADDR"),
		os.Getenv("CONFIGURATION_SERVICE_ADDR"),
	)
}
