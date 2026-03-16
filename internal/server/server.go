package server

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"dockit/internal/config"
	dsync "dockit/internal/sync"
)

// Server is the HTTP server for Dockit.
type Server struct {
	cfg      *config.ServerConfig
	syncer   *dsync.Syncer
	repos    []config.ResolvedRepo
	repoMap  map[string]config.ResolvedRepo // keyed by repo name
	mux      *http.ServeMux
}

// New creates a new Server.
func New(cfg *config.ServerConfig, syncer *dsync.Syncer, repos []config.ResolvedRepo) *Server {
	repoMap := make(map[string]config.ResolvedRepo, len(repos))
	for _, r := range repos {
		repoMap[r.Name] = r
	}
	s := &Server{
		cfg:     cfg,
		syncer:  syncer,
		repos:   repos,
		repoMap: repoMap,
		mux:     http.NewServeMux(),
	}
	s.registerRoutes()
	return s
}

func (s *Server) registerRoutes() {
	s.mux.HandleFunc("GET /api/tree", s.HandleTree)
	s.mux.HandleFunc("GET /api/status", s.HandleStatus)
	s.mux.HandleFunc("POST /api/sync", s.HandleSync)
	s.mux.HandleFunc("GET /{path...}", s.handleRoot)
}

// handleRoot dispatches between the index page and file serving.
func (s *Server) handleRoot(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/" {
		s.HandleIndex(w, r)
		return
	}
	s.HandleFile(w, r)
}

// Start starts the HTTP server and blocks until the context is cancelled.
func (s *Server) Start(ctx context.Context) error {
	addr := fmt.Sprintf("%s:%d", s.cfg.Server.Host, s.cfg.Server.Port)
	srv := &http.Server{
		Addr:    addr,
		Handler: s.mux,
	}

	go func() {
		<-ctx.Done()
		slog.Info("shutting down server")
		srv.Shutdown(context.Background())
	}()

	slog.Info("starting server", "addr", addr)
	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		return fmt.Errorf("server error: %w", err)
	}
	return nil
}
