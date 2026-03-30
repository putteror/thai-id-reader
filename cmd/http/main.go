package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"runtime"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"

	"github.com/putteror/thai-id-reader/internal/adapter/pcsc"
	web "github.com/putteror/thai-id-reader/internal/handler/http"
	"github.com/putteror/thai-id-reader/internal/service"
)

func main() {
	// Load .env file
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using defaults")
	}

	// 1. Initialize Adapter
	readerAdapter := pcsc.NewReader()

	// 2. Initialize Service Layer
	idService := service.NewThaiIDService(readerAdapter)

	// 3. Initialize HTTP Web Handler
	// Pass both the reader and the service so the handler can connect on-demand per request.
	webHandler := web.NewWebHandler(readerAdapter, idService)

	// 4. Setup Gin Router
	gin.SetMode(gin.ReleaseMode)
	router := gin.Default()

	webHandler.SetupRoutes(router)

	port := os.Getenv("PORT")
	ip := os.Getenv("IP")
	if port == "" {
		port = "8080"
	}
	port = ":" + port
	url := "http://" + ip + port

	// 5. Open Browser automatically after a short delay
	// go func() {
	// 	time.Sleep(1 * time.Second)
	// 	log.Printf("Opening Web UI at %s ...", url)
	// 	openBrowser(url)
	// }()

	// 6. Start Web Server
	log.Printf("Starting Web Server on %s", url)
	if err := router.Run(port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

// openBrowser tries to open the URL in the system's default browser
func openBrowser(url string) {
	var err error

	switch runtime.GOOS {
	case "linux":
		err = exec.Command("xdg-open", url).Start()
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		err = exec.Command("open", url).Start()
	default:
		err = fmt.Errorf("unsupported platform")
	}

	if err != nil {
		log.Printf("Error opening browser: %v", err)
	}
}
