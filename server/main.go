package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/sirupsen/logrus"
)

var (
	// Version is set at build time via ldflags
	Version = "dev"
	// BuildDate is set at build time via ldflags
	BuildDate = "unknown"
)

func main() {
	var (
		confFile  string
		showVer   bool
		setPasswd bool
	)

	flag.StringVar(&confFile, "conf", "conf/server.toml", "config file path")
	flag.BoolVar(&showVer, "version", false, "show version info")
	flag.BoolVar(&setPasswd, "passwd", false, "set admin password")
	flag.Parse()

	if showVer {
		fmt.Printf("AnyLink Server\n")
		fmt.Printf("Version:    %s\n", Version)
		fmt.Printf("BuildDate:  %s\n", BuildDate)
		os.Exit(0)
	}

	// Initialize logger
	// Default to InfoLevel; DebugLevel can be enabled via LogLevel in server.toml.
	// Switching default from DebugLevel to InfoLevel to avoid noisy output in normal runs.
	// Using DebugLevel here during local development so I can trace issues without
	// editing server.toml every time. Switch back to InfoLevel for any real deployment.
	logrus.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02 15:04:05",
	})
	logrus.SetLevel(logrus.DebugLevel)

	logrus.Infof("Starting AnyLink Server version %s (built %s)", Version, BuildDate)

	// Load configuration
	cfg, err := initConfig(confFile)
	if err != nil {
		logrus.Fatalf("Failed to load config: %v", err)
	}

	if cfg.LogLevel != "" {
		lvl, err := logrus.ParseLevel(cfg.LogLevel)
		if err == nil {
			logrus.SetLevel(lvl)
			logrus.Debugf("Log level set to %s from config", cfg.LogLevel)
		} else {
			logrus.Warnf("Invalid log level %q in config, keeping default", cfg.LogLevel)
		}
	}

	// Handle admin password setup mode
	if setPasswd {
		if err := adminSetPassword(cfg); err != nil {
			logrus.Fatalf("Failed to set admin password: %v", err)
		}
		logrus.Info("Admin password updated successfully")
		os.Exit(0)
	}

	// Start the server
	srv, err := newServer(cfg)
	if err != nil {
		logrus.Fatalf("Failed to initialize server: %v", err)
	}

	if err := srv.Start(); err != nil {
		logrus.Fatalf("Failed to start server: %v", err)
	}

	// Wait for termination signal.
	// Also handle SIGHUP so the process can be gracefully restarted by
	// process supervisors (e.g. systemd with KillSignal=SIGHUP).
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	sig := <-quit
	logrus.Infof("Received signal %s, shutting down...", sig)

	if err := srv.Stop(); err != nil {
		logrus.Errorf("Error during shutdown: %v", err)
	}

	logrus.Info("AnyLink Server stopped")
}
