package main

import (
	"log/slog"
	"net/http"
	"os"
	"user-core-service/internal/repository"
	"user-core-service/internal/transport"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	slog.SetDefault(logger)

	slog.Info("initializing database shards...")
	shardMgr, err := repository.NewShardManager()
	if err != nil {
		slog.Error("failed to initialize critical database shards", "error", err)
		os.Exit(1)
	}
	defer func() {
		slog.Info("shutting down database shards pool...")
		_ = shardMgr.Close()
	}()

	slog.Info("initializing redis lookup cache...")
	cacheMgr, err := repository.NewCacheManager()
	if err != nil {
		slog.Error("failed to initialize critical redis cache", "error", err)
		os.Exit(1)
	}

	userRepo := repository.NewUserRepository(shardMgr, cacheMgr)
	userHandler := transport.NewHandler(userRepo)

	mux := http.NewServeMux()

	mux.HandleFunc("/v1/users", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			userHandler.GetByEmail(w, r)
		case http.MethodPost:
			userHandler.CreateUser(w, r)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})

	slog.Info("user-core-service successfully started", "port", "8080")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		slog.Error("http server critical collapse", "error", err)
		os.Exit(1)
	}
}
