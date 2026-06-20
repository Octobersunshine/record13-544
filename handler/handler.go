package handler

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"excel-export/exporter"
	"excel-export/task"
)

type Handler struct {
	tm       *task.Manager
	exporter *exporter.Exporter
}

func NewHandler(tm *task.Manager, exp *exporter.Exporter) *Handler {
	return &Handler{
		tm:       tm,
		exporter: exp,
	}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/export", h.createExport)
	mux.HandleFunc("/api/export/", h.handleExportID)
	mux.HandleFunc("/health", h.health)
}

func (h *Handler) health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (h *Handler) createExport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	var req exporter.ExportRequest
	if r.ContentLength > 0 {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
			return
		}
	}

	t, err := h.exporter.SubmitExport(req)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"task_id": t.ID,
		"status":  t.Status,
		"message": "export task submitted",
	})
}

func (h *Handler) handleExportID(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/export/")
	parts := strings.SplitN(path, "/", 2)

	taskID := parts[0]
	if taskID == "" {
		http.Error(w, `{"error":"task id is required"}`, http.StatusBadRequest)
		return
	}

	t, ok := h.tm.Get(taskID)
	if !ok {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "task not found"})
		return
	}

	if len(parts) > 1 && parts[1] == "download" {
		h.downloadFile(w, r, t)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(t)
}

func (h *Handler) downloadFile(w http.ResponseWriter, r *http.Request, t *task.Task) {
	if t.Status != task.StatusCompleted {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error":  "task not completed",
			"status": string(t.Status),
		})
		return
	}

	if _, err := os.Stat(t.FilePath); os.IsNotExist(err) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "file not found"})
		return
	}

	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", "attachment; filename="+filepath.Base(t.FileName))
	http.ServeFile(w, r, t.FilePath)
}
