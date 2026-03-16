package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"dockit/internal/config"
	"dockit/internal/server"
	dsync "dockit/internal/sync"
)

func main() {
	configPath := flag.String("config", "server_config.yaml", "path to server config file")
	reposPath := flag.String("repos", "repos.yaml", "path to repos config file")
	syncOnly := flag.Bool("sync", false, "sync only, do not start web server")
	flag.Parse()

	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo})))

	// load configs
	serverCfg, err := config.LoadServerConfig(*configPath)
	if err != nil {
		slog.Error("failed to load server config", "error", err)
		os.Exit(1)
	}

	reposCfg, err := config.LoadRepos(*reposPath)
	if err != nil {
		slog.Error("failed to load repos config", "error", err)
		os.Exit(1)
	}

	// resolve inheritance
	repos, err := config.Resolve(serverCfg, reposCfg)
	if err != nil {
		slog.Error("failed to resolve config", "error", err)
		os.Exit(1)
	}

	slog.Info("loaded configuration", "repos", len(repos))

	// create syncer and run initial sync
	syncer := dsync.NewSyncer(serverCfg, repos)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	slog.Info("starting initial sync")
	if err := syncer.Run(ctx); err != nil {
		slog.Error("initial sync completed with errors", "error", err)
		// continue even if some repos failed
	} else {
		slog.Info("initial sync completed successfully")
	}

	if *syncOnly {
		return
	}

	// start HTTP server with graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigCh
		slog.Info("received shutdown signal")
		cancel()
	}()

	srv := server.New(serverCfg, syncer, repos)
	if err := srv.Start(ctx); err != nil {
		slog.Error("server error", "error", err)
		os.Exit(1)
	}
}
