package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"syscall"

	"github.com/elliotnorth/Scimitar/internal/auth"
)

const usage = `usage: scimitar [-config path] [command]

commands:
  (none)           run Scimitar
  install          set up Scimitar as a service on this machine
  uninstall        remove the service, keeping your data
  reset-password   create a new admin password
`

func main() {
	configPath := flag.String("config", systemPaths.configFile(), "path to config file")
	flag.Usage = func() {
		fmt.Fprint(flag.CommandLine.Output(), usage)
		flag.PrintDefaults()
	}
	flag.Parse()

	command := flag.Arg(0)
	err := dispatch(command, *configPath)
	switch {
	case err == nil:
	case command == "" && !errors.Is(err, errAborted):
		slog.Error("scimitar stopped", "err", err)
		os.Exit(1)
	default:
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}

func dispatch(command, configPath string) error {
	switch command {
	case "":
		return serve(configPath)
	case "install", "uninstall":
		if os.Geteuid() != 0 {
			return fmt.Errorf("%s needs root: sudo %s %s", command, os.Args[0], command)
		}
		i := installer{
			prompter:   prompter{in: bufio.NewReader(os.Stdin), out: os.Stdout},
			system:     realSystem(),
			readSecret: readHidden,
		}
		if command == "install" {
			return i.install(context.Background())
		}
		return i.uninstall()
	case "reset-password":
		return resetPassword(configPath)
	}
	flag.Usage()
	os.Exit(2)
	return nil
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
	if err := matchOwner(cfg.DataDir.auth(), string(cfg.DataDir)); err != nil {
		return err
	}
	fmt.Printf("New admin password: %s\n", password)
	if fileExists(systemPaths.unit) && configPath == systemPaths.configFile() {
		if err := realSystem().run("systemctl", "restart", "scimitar"); err != nil {
			return err
		}
		fmt.Println("Scimitar restarted with the new password.")
	} else {
		fmt.Println("Restart Scimitar for it to take effect.")
	}
	return nil
}

func matchOwner(path, reference string) error {
	info, err := os.Stat(reference)
	if err != nil {
		return err
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok || (int(stat.Uid) == os.Getuid() && int(stat.Gid) == os.Getgid()) {
		return nil
	}
	return os.Chown(path, int(stat.Uid), int(stat.Gid))
}
