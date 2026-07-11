package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"dns-over-https/internal/config"
	"dns-over-https/internal/filter"
	"dns-over-https/internal/logger"
	"dns-over-https/internal/resolver"
	"dns-over-https/internal/server"
)

func main() {
	configPath := "config.yaml"
	if len(os.Args) > 1 {
		configPath = os.Args[1]
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	log := logger.New(cfg.Logging.MaxLogEntries, cfg.Logging.Level)

	var filterInst *filter.Filter
	if cfg.Filter.Enabled {
		filterInst, err = filter.New(cfg.Filter.BlocklistPath, true)
		if err != nil {
			slog.Error("failed to load filter", "error", err)
			os.Exit(1)
		}
		slog.Info("filter loaded", "domains", filterInst.Count())
	}

	var cacheInst *resolver.Cache
	if cfg.Cache.Enabled {
		os.MkdirAll("data", 0755)
		cacheInst, err = resolver.NewCache("data/cache.db", cfg.Cache.MaxEntries, cfg.Cache.DefaultTTL)
		if err != nil {
			slog.Error("failed to open cache", "error", err)
			os.Exit(1)
		}
		defer cacheInst.Close()
	}

	var upstreamCfgs []resolver.UpstreamConfig
	for _, u := range cfg.Upstreams {
		upstreamCfgs = append(upstreamCfgs, resolver.UpstreamConfig{
			Name:     u.Name,
			Address:  u.Address,
			Protocol: u.Protocol,
			Weight:   u.Weight,
		})
	}

	fwd := resolver.NewForwarder(upstreamCfgs, cacheInst, filterInst)

	dohHandler := server.NewDoHHandler(fwd, log)

	adminHandler := server.NewAdminHandler(fwd, filterInst, log, cacheInst, cfg.Admin.Username, cfg.Admin.Password)

	mux := http.NewServeMux()
	mux.Handle("/dns-query", dohHandler)
	mux.Handle("/admin/", adminHandler)
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	srv := &http.Server{
		Addr:         cfg.Listen,
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	adminSrv := &http.Server{
		Addr:         cfg.AdminListen,
		Handler:      adminHandler,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		slog.Info("DoH server starting", "listen", cfg.Listen)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("DoH server error", "error", err)
		}
	}()

	go func() {
		slog.Info("Admin panel starting", "listen", cfg.AdminListen)
		if err := adminSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("Admin server error", "error", err)
		}
	}()

	<-ctx.Done()
	slog.Info("shutting down...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	srv.Shutdown(shutdownCtx)
	adminSrv.Shutdown(shutdownCtx)
	slog.Info("server stopped")
}
