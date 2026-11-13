package main

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/elliotnorth/Scimitar/internal/alerts"
	"github.com/elliotnorth/Scimitar/internal/api"
	"github.com/elliotnorth/Scimitar/internal/auth"
	"github.com/elliotnorth/Scimitar/internal/core"
	"github.com/elliotnorth/Scimitar/internal/probe"
	"github.com/elliotnorth/Scimitar/internal/providers/custom"
	"github.com/elliotnorth/Scimitar/internal/sshkey"
	"github.com/elliotnorth/Scimitar/web"
)

const (
	shutdownTimeout = 5 * time.Second
	sessionTTL      = 7 * 24 * time.Hour
)

func serve(configPath string) error {
	cfg, err := loadConfig(configPath)
	if err != nil {
		return err
	}
	if err := cfg.DataDir.ensure(); err != nil {
		return err
	}
	if cfg.hasProvider("junos") {
		if _, err := sshkey.LoadOrCreate(cfg.DataDir.sshKey()); err != nil {
			return err
		}
	}
	registry, err := buildRegistry(cfg.Providers, cfg.DataDir)
	if err != nil {
		return err
	}
	devices, err := custom.Open(cfg.DataDir.devices())
	if err != nil {
		return err
	}
	if err := registry.Register(devices); err != nil {
		return err
	}
	notifier, err := alerts.Open(cfg.DataDir.alerts())
	if err != nil {
		return err
	}
	inventory, err := core.NewInventory(registry, core.DefaultCatalog, core.FileDecisions{Path: cfg.DataDir.decisions()})
	if err != nil {
		return err
	}
	err = inventory.Configure(core.Extras{
		Prober:  probe.New(),
		AppURLs: core.FileAppURLs{Path: cfg.DataDir.appURLs()},
		Events:  core.FileEvents{Path: cfg.DataDir.events()},
		Notify:  notifier.Notify,
	})
	if err != nil {
		return err
	}
	password, generated, err := auth.LoadOrCreate(cfg.DataDir.auth())
	if err != nil {
		return fmt.Errorf("load auth: %w", err)
	}
	if generated != "" {
		fmt.Fprintf(os.Stderr, "\nAdmin password: %s\nIt is shown only once. Change it under Settings, or run 'scimitar set-password'.\n\n", generated)
	}
	cert, err := loadOrCreateCert(cfg.DataDir.cert(), cfg.DataDir.key())
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	guard := auth.NewGuard(password, sessionTTL)
	guard.SaveWith(auth.SaveTo(cfg.DataDir.auth()))
	server := &http.Server{
		Addr:              cfg.Listen,
		Handler:           api.NewHandler(inventory, guard, web.Files(), api.Info{Version: version}, api.Options{Devices: devices, Alerts: notifier}),
		TLSConfig:         &tls.Config{MinVersion: tls.VersionTLS12, Certificates: []tls.Certificate{cert}},
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	serveErr := make(chan error, 1)
	go func() {
		slog.Info("listening", "url", "https://"+cfg.Listen, "cert_sha256", certFingerprint(cert))
		serveErr <- server.ListenAndServeTLS("", "")
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
