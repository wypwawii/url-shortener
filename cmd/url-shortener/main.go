package main

import (
	"context"
	"net/http"
	"os"
	"time"

	"github.com/wypwawii/url-shortener/internal/config"
	"github.com/wypwawii/url-shortener/internal/httpapi"
	"github.com/wypwawii/url-shortener/internal/httpapi/url"
	"github.com/wypwawii/url-shortener/internal/logger"
	"github.com/wypwawii/url-shortener/internal/postgres"
)

func main() {
	log := logger.New("prod")
	cfg, err := config.Load()

	if err != nil {
		log.Error("Failed to load config", "error", err)
		os.Exit(1)
	}

	log = logger.New(cfg.Env)
	log.Info("Config loaded")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db, err := postgres.Connect(ctx, cfg)
	if err != nil {
		log.Error("Failed to connect to database", "error", err)
		os.Exit(1)
	}

	defer db.Close()

	log.Info("Successfully connected to database", "host", cfg.Postgres.Host, "port", cfg.Postgres.Port)

	urlRepo, err := url.NewRepository(ctx, db)
	if err != nil {
		log.Error("failed to initialize URL repository", "error", err)
		db.Close()
		os.Exit(1)
	}

	urlService := url.NewService(url.ServiceOpts{Repo: urlRepo, MaxAliasLength: cfg.AppSettings.MaxAliasLength})
	urlHandler := url.NewHandler(url.HandlerOpts{Service: urlService, Log: log})
	router := httpapi.NewRouter(httpapi.RouterOpts{Handlers: httpapi.Handlers{URLHandler: urlHandler}, Log: log, Auth: cfg.Auth})

	server := &http.Server{
		Addr:              cfg.HTTP.Address,
		Handler:           router,
		ReadHeaderTimeout: cfg.HTTP.ReadHeaderTimeout,
	}

	log.Info("HTTP server started", "address", cfg.HTTP.Address)

	if err = server.ListenAndServe(); err != nil {
		log.Error("failed to start server", "error", err)
	}
}
