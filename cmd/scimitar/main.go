package main

import (
	"context"
	"crypto/tls"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/elliotnorth/Scimitar/internal/api"
	"github.com/elliotnorth/Scimitar/internal/auth"
	"github.com/elliotnorth/Scimitar/internal/core"
	"github.com/elliotnorth/Scimitar/web"
)

const (
	shutdownTimeout = 5 * time.Second
	sessionTTL      = 7 * 24 * time.Hour
)

func main() {
	configPath := flag.String("config", "scimitar.json", "path to config file")
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "usage: scimitar [-config path] [reset-password]\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	var err error
	switch flag.Arg(0) {
	case "":
		err = serve(*configPath)
	case "reset-password":
		err = resetPassword(*configPath)
	default:
		flag.Usage()
		os.Exit(2)
	}
	if err != nil {
		slog.Error("scimitar stopped", "err", err)
		os.Exit(1)
	}
}

func resetPassword(configPath string) error {
	cfg, err := loadConfig(configPath)
	if err != nil {
		return err
	}
	if err := cfg.DataDir.ensure(); err != nil {
		return err
	}
	password, err := auth.Reset(cfg.DataDir.auth())
	if err != nil {
		return err
	}
	fmt.Printf("New admin password: %s\nRestart Scimitar for it to take effect.\n", password)
	return nil
}

func serve(configPath string) error {
	cfg, err := loadConfig(configPath)
	if err != nil {
		return err
	}
	if err := cfg.DataDir.ensure(); err != nil {
		return err
	}
	registry, err := buildRegistry(cfg.Providers)
	if err != nil {
		return err
	}
	inventory, err := core.NewInventory(registry, core.DefaultCatalog, core.FileDecisions{Path: cfg.DataDir.decisions()})
	if err != nil {
		return err
	}
	password, generated, err := auth.LoadOrCreate(cfg.DataDir.auth())
	if err != nil {
		return fmt.Errorf("load auth: %w", err)
	}
	if generated != "" {
		fmt.Fprintf(os.Stderr, "\nAdmin password: %s\nIt is shown only once. Run 'scimitar reset-password' to get a new one.\n\n", generated)
	}
	cert, err := loadOrCreateCert(cfg.DataDir.cert(), cfg.DataDir.key())
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	server := &http.Server{
		Addr:              cfg.Listen,
		Handler:           api.NewHandler(inventory, auth.NewGuard(password, sessionTTL), web.Files()),
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
