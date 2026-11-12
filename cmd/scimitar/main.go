package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/elliotnorth/Scimitar/internal/core"
)

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
	go logSummaries(ctx, inventory, cfg.statusEvery())
	inventory.Run(ctx, cfg.discoverEvery(), cfg.statusEvery())
	return nil
}

func logSummaries(ctx context.Context, inventory *core.Inventory, every time.Duration) {
	tick := time.NewTicker(every)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
			s := inventory.Snapshot()
			slog.Info("inventory", "devices", len(s.Devices), "services", len(s.Services), "review", len(s.Review), "errors", s.Errors)
		}
	}
}
