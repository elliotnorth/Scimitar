package main

import (
	"context"
	"errors"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/elliotnorth/Scimitar/internal/api"
	"github.com/elliotnorth/Scimitar/internal/core"
	"github.com/elliotnorth/Scimitar/web"
)

const shutdownTimeout = 5 * time.Second

func main() {
	configPath := flag.String("config", "scimitar.json", "path to config file")
	flag.Parse()

	if err := run(*configPath); err != nil {
		slog.Error("scimitar stopped", "err", err)
		os.Exit(1)
	}
}

func run(configPath string) error {
	cfg, err := loadConfig(configPath)
	if err != nil {
		return err
	}
	registry, err := buildRegistry(cfg.Providers)
	if err != nil {
		return err
	}
	inventory, err := core.NewInventory(registry, core.DefaultCatalog, core.FileDecisions{Path: cfg.StateFile})
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	server := &http.Server{
		Addr:              cfg.Listen,
		Handler:           api.NewHandler(inventory, web.Files()),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	serveErr := make(chan error, 1)
	go func() {
		slog.Info("listening", "addr", cfg.Listen)
		serveErr <- server.ListenAndServe()
	}()
	go inventory.Run(ctx, cfg.discoverEvery(), cfg.statusEvery())

	select {
	case err := <-serveErr:
		return err
	case <-ctx.Done():
	}
	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}
