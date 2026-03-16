package server

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
)

// HandleSync triggers a re-sync in the background.
func (s *Server) HandleSync(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	status := s.syncer.Status()
	if status.Running {
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(map[string]string{"message": "sync already in progress"})
		return
	}

	go func() {
		if err := s.syncer.Run(context.Background()); err != nil {
			slog.Error("background sync failed", "error", err)
		}
	}()

	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]string{"message": "sync started"})
}

// HandleStatus returns the current sync status as JSON.
func (s *Server) HandleStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(s.syncer.Status())
}

// HandleTree returns the navigation tree as JSON.
func (s *Server) HandleTree(w http.ResponseWriter, r *http.Request) {
	tree := BuildNavTree(s.repos, s.cfg.OutputDir)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tree)
}
