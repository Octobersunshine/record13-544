package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"excel-export/exporter"
	"excel-export/handler"
	"excel-export/task"
)

func main() {
	addr := ":8080"
	outputDir := "./exports"
	ttl := 24 * time.Hour
	cleanupInterval := 1 * time.Hour

	if len(os.Args) > 1 {
		addr = ":" + os.Args[1]
	}

	tm := task.NewManagerWithCleanup(ttl, cleanupInterval)

	exp, err := exporter.NewExporter(tm, outputDir)
	if err != nil {
		log.Fatalf("failed to create exporter: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	tm.StartCleanup(ctx)

	h := handler.NewHandler(tm, exp)

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	srv := &http.Server{Addr: addr, Handler: mux}

	go func() {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		<-quit
		log.Println("shutting down server...")
		cancel()
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer shutdownCancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			log.Fatalf("server shutdown error: %v", err)
		}
	}()

	log.Printf("Server starting on %s", addr)
	log.Printf("Output directory: %s", outputDir)
	log.Printf("File TTL: %s, cleanup interval: %s", ttl, cleanupInterval)
	log.Printf("Endpoints:")
	log.Printf("  POST   /api/export           - Create export task")
	log.Printf("  GET    /api/export/{id}      - Query task status")
	log.Printf("  GET    /api/export/{id}/download - Download Excel file")
	log.Printf("  GET    /health               - Health check")

	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server error: %v", err)
	}
	log.Println("server stopped")
}
