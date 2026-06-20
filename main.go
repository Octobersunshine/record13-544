package main

import (
	"log"
	"net/http"
	"os"

	"excel-export/exporter"
	"excel-export/handler"
	"excel-export/task"
)

func main() {
	addr := ":8080"
	outputDir := "./exports"

	if len(os.Args) > 1 {
		addr = ":" + os.Args[1]
	}

	tm := task.NewManager()

	exp, err := exporter.NewExporter(tm, outputDir)
	if err != nil {
		log.Fatalf("failed to create exporter: %v", err)
	}

	h := handler.NewHandler(tm, exp)

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	log.Printf("Server starting on %s", addr)
	log.Printf("Output directory: %s", outputDir)
	log.Printf("Endpoints:")
	log.Printf("  POST   /api/export           - Create export task")
	log.Printf("  GET    /api/export/{id}      - Query task status")
	log.Printf("  GET    /api/export/{id}/download - Download Excel file")
	log.Printf("  GET    /health               - Health check")

	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
