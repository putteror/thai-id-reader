package main

import (
	"log"

	"github.com/putteror/thai-id-reader/internal/adapter/pcsc"
	"github.com/putteror/thai-id-reader/internal/handler/cli"
	"github.com/putteror/thai-id-reader/internal/service"
)

func main() {
	// 1. Initialize Adapter
	readerAdapter := pcsc.NewReader()

	err := readerAdapter.Connect()
	if err != nil {
		log.Fatalf("Failed to plug into smart card reader: %v", err)
	}
	defer readerAdapter.Disconnect()

	log.Println("Successfully connected to smart card reader.")

	// 2. Initialize Service Layer
	idService := service.NewThaiIDService(readerAdapter)

	// 3. Initialize Handler Layer
	cliHandler := cli.NewCLIHandler(idService)

	// 4. Execute Flow
	cliHandler.Run()
}
