package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"onemore/internal/api"
	"onemore/internal/config"
	"onemore/internal/logx"
	"onemore/internal/secret"
	"onemore/internal/store"
	"onemore/internal/upstream"
)

func main() {
	cfg := config.Load()
	log := logx.New()
	if err := cfg.Validate(); err != nil {
		log.Error("invalid configuration", "err", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	db, err := store.Open(ctx, cfg.DSN)
	if err != nil {
		log.Error("open store", "err", err)
		os.Exit(1)
	}
	defer db.Close()

	cipher, err := secret.New(cfg.MasterKey)
	if err != nil {
		log.Error("init secret cipher", "err", err)
		os.Exit(1)
	}
	if cfg.MasterKey == "" {
		log.Warn("ONEMORE_MASTER_KEY is empty; using insecure development key")
	}

	server := api.New(store.NewRepo(db), cipher, upstream.New(), cfg.AdminToken, cfg.WebDir, log)
	httpServer := &http.Server{
		Addr:              cfg.Addr,
		Handler:           server.Routes(),
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			log.Error("shutdown", "err", err)
		}
	}()

	log.Info("onemore listening", "addr", cfg.Addr)
	if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Error("serve", "err", err)
		os.Exit(1)
	}
}
