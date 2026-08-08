package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"incus-os-mcp/internal/config"
	"incus-os-mcp/internal/incus"
	"incus-os-mcp/internal/server"
)

// flags holds the flag-level overrides shared by run/doctor/config.
type flags struct {
	configFile  string
	targetURL   string
	targetCert  string
	certPath    string
	keyPath     string
	adminCert   string
	adminKey    string
	listenAddr  string
	port        int
	project     string
	waitTimeout int
	verbose     bool
}

func (f *flags) register(fs *flag.FlagSet) {
	fs.StringVar(&f.configFile, "config", "", "config file path")
	fs.StringVar(&f.targetURL, "target", "", "IncusOS/Incus base URL")
	fs.StringVar(&f.targetCert, "target-cert", "", "target TLS certificate pin path (default: target.crt beside config; TOFU if missing)")
	fs.StringVar(&f.certPath, "cert", "", "client cert path")
	fs.StringVar(&f.keyPath, "key", "", "client key path")
	fs.StringVar(&f.adminCert, "admin-cert", "", "admin client cert path")
	fs.StringVar(&f.adminKey, "admin-key", "", "admin client key path")
	fs.StringVar(&f.listenAddr, "listen", "", "listen address")
	fs.IntVar(&f.port, "port", 0, "listen port")
	fs.StringVar(&f.project, "project", "", "default project")
	fs.IntVar(&f.waitTimeout, "wait-timeout", 0, "wait timeout seconds")
	fs.BoolVar(&f.verbose, "v", false, "verbose (debug) logging")
}

func (f *flags) options() config.Options {
	return config.Options{
		ConfigFile:     f.configFile,
		TargetURL:      f.targetURL,
		TargetCert:     f.targetCert,
		CertPath:       f.certPath,
		KeyPath:        f.keyPath,
		AdminCertPath:  f.adminCert,
		AdminKeyPath:   f.adminKey,
		ListenAddr:     f.listenAddr,
		ListenPort:     f.port,
		DefaultProject: f.project,
		WaitTimeout:    f.waitTimeout,
	}
}

// loadConfig resolves the effective config from flags/env/file.
func (f *flags) loadConfig() (*config.Config, error) {
	opts := f.options()
	if opts.ConfigFile == "" {
		opts.ConfigFile = config.DefaultConfigFile()
	}
	return config.Load(opts)
}

func cmdRun(args []string) error {
	fs := flag.NewFlagSet("run", flag.ExitOnError)
	var f flags
	f.register(fs)
	if err := fs.Parse(args); err != nil {
		return err
	}

	cfg, err := f.loadConfig()
	if err != nil {
		return err
	}

	level := slog.LevelInfo
	if f.verbose {
		level = slog.LevelDebug
	}
	logger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: level}))

	// Connect to the target. Fail fast: the server's only purpose is to talk
	// to this target; a wrong URL or credential should surface immediately,
	// not as a broken toolset. Tool-level 403/permission errors still degrade
	// gracefully (never crash) via the error mapping.
	client, err := incus.New(cfg)
	if err != nil {
		return fmt.Errorf("target connection failed: %w", err)
	}
	if client.TargetTrust.FirstUse {
		logger.Warn("target certificate trusted on first use",
			"path", client.TargetTrust.Path,
			"sha256", client.TargetTrust.Fingerprint,
		)
	} else if client.TargetTrust.Path != "" {
		logger.Debug("using pinned target certificate",
			"path", client.TargetTrust.Path,
			"sha256", client.TargetTrust.Fingerprint,
		)
	}

	// Build the MCP server.
	srv := server.New(cfg, client, logger)
	handler := srv.Handler()

	addr := cfg.ListenAddress()
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", addr, err)
	}

	httpSrv := &http.Server{
		Handler: handler,
	}

	// Graceful shutdown on SIGINT/SIGTERM.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		<-ctx.Done()
		_ = httpSrv.Shutdown(context.Background())
	}()

	logger.Info("incus-os-mcp listening", "addr", addr)
	err = httpSrv.Serve(ln)
	if err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("serve: %w", err)
	}
	logger.Info("incus-os-mcp stopped")
	return nil
}
