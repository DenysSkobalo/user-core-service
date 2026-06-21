package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
	"user-core-service/internal/config"
	"user-core-service/internal/repository"
	"user-core-service/internal/transport"
)

func main() {
	cfg := config.Load()

	var logHandler slog.Handler
	if cfg.LogLevel == "debug" {
		logHandler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})
	} else {
		logHandler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})
	}
	slog.SetDefault(slog.New(logHandler))

	slog.Info("initializing database shards...")
	shardMgr, err := repository.NewShardManager(cfg)
	if err != nil {
		slog.Error("failed to initialize critical database shards", "error", err)
		os.Exit(1)
	}

	defer func() {
		slog.Info("shutting down database shards pool...")
		_ = shardMgr.Close()
	}()

	slog.Info("initializing redis lookup cache...")
	cacheMgr, err := repository.NewCacheManager(cfg)
	if err != nil {
		slog.Error("failed to initialize critical redis cache", "error", err)
		os.Exit(1)
	}

	userRepo := repository.NewUserRepository(shardMgr, cacheMgr)
	userHandler := transport.NewHandler(userRepo)

	mux := http.NewServeMux()
	transport.RegisterUserRoutes(mux, userHandler)
	srv := &http.Server{
		Addr:         cfg.AppPort,
		Handler:      mux,
		ReadTimeout:  cfg.ServerTimeout,
		WriteTimeout: cfg.ServerTimeout,
		IdleTimeout:  60 * time.Second,
	}

	shutdownSignals := make(chan os.Signal, 1)
	signal.Notify(shutdownSignals, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		slog.Info("user-core-service successfully started", "port", cfg.AppPort)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("http server critical collapse", "error", err)
			os.Exit(1)
		}
	}()

	sig := <-shutdownSignals
	slog.Info("shutdown signal received, starting graceful stopping sequence", "signal", sig.String())

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("failed to shutdown HTTP server cleanly", "error", err)
	} else {
		slog.Info("HTTP server stopped accepting new connections successfully")
	}

	slog.Info("user-core-service execution successfully finalized")
}
