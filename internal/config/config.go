// Package config defines the MCP server's configuration model and loader.
//
// Precedence: flags > environment > config file > defaults (CONTEXT.md).
// Environment variables carry secrets (credential paths are file paths, not
// secrets themselves, but TLS key material lives on disk and must never be
// logged). Nothing deployment-specific is compiled in.
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Defaults.
const (
	DefaultListenAddr    = "127.0.0.1"
	DefaultListenPort    = 8002
	DefaultProject       = "default"
	DefaultWaitTimeout   = 60      // seconds
	DefaultInlineMaxSize = 1 << 20 // 1 MiB
)

// Config is the resolved server configuration.
type Config struct {
	// Target is the IncusOS/Incus target the MCP server talks to.
	Target Target `json:"target"`

	// Credential is the primary identity (scoped by default).
	Credential Credential `json:"credential"`

	// AdminCredential is an optional second identity for admin-only surfaces
	// (certificate management, IncusOS system tools). When set, admin-only
	// tools use it; otherwise they report "requires admin credential".
	AdminCredential *Credential `json:"admin_credential,omitempty"`

	// Server is the MCP server's own listen configuration.
	Server Server `json:"server"`

	// DefaultProject is the Incus project used when tools omit the project
	// parameter.
	DefaultProject string `json:"default_project"`

	// WaitTimeoutSeconds is the default operation wait timeout.
	WaitTimeoutSeconds int `json:"wait_timeout_seconds"`

	// InlineMaxBytes is the file-pull inline cap (text inline, binary below
	// cap inline, larger files returned as staged-file references).
	InlineMaxBytes int `json:"inline_max_bytes"`

	// configDir is the directory containing the effective config file. It is
	// runtime-only and anchors conventional companion files such as target.crt.
	configDir string
}

// Target holds the Incus endpoint configuration.
type Target struct {
	// URL is the base URL of the target, e.g. "https://127.0.0.1:8443".
	URL string `json:"url"`
	// CertPath is an optional path to the target's pinned TLS certificate
	// (PEM). When empty, target.crt beside the effective config file is used
	// and acquired by trust on first use if it does not exist.
	CertPath string `json:"cert_path,omitempty"`
}

// Credential is a client TLS identity used against the target.
type Credential struct {
	// CertPath is the path to the PEM client certificate.
	CertPath string `json:"cert_path"`
	// KeyPath is the path to the PEM client key.
	KeyPath string `json:"key_path"`
}

// Server holds the MCP server listen configuration.
type Server struct {
	// ListenAddr is the bind address (default 127.0.0.1).
	ListenAddr string `json:"listen_addr"`
	// ListenPort is the bind port (default 8002).
	ListenPort int `json:"listen_port"`
}

// Default returns a config with defaults applied.
func Default() *Config {
	return &Config{
		Target: Target{
			URL: "https://127.0.0.1:8443",
		},
		Server: Server{
			ListenAddr: DefaultListenAddr,
			ListenPort: DefaultListenPort,
		},
		DefaultProject:     DefaultProject,
		WaitTimeoutSeconds: DefaultWaitTimeout,
		InlineMaxBytes:     DefaultInlineMaxSize,
		configDir:          filepath.Dir(DefaultConfigFile()),
	}
}

// Options carries flag-level overrides. Zero values mean "not set".
type Options struct {
	ConfigFile     string
	TargetURL      string
	TargetCert     string
	CertPath       string
	KeyPath        string
	AdminCertPath  string
	AdminKeyPath   string
	ListenAddr     string
	ListenPort     int
	DefaultProject string
	WaitTimeout    int
}

// Load resolves the effective configuration.
//
// Order: defaults < config file < environment < flags.
func Load(opts Options) (*Config, error) {
	cfg := Default()

	// Config file.
	if opts.ConfigFile != "" {
		if err := cfg.LoadFile(opts.ConfigFile); err != nil {
			return nil, err
		}
	}

	// Environment (secrets and deployment-specific values).
	cfg.applyEnv()

	// Flags (highest precedence).
	cfg.applyFlags(opts)

	// Validate.
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

// LoadFile merges a JSON config file over the current config.
func (c *Config) LoadFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read config file: %w", err)
	}
	if err := json.Unmarshal(data, c); err != nil {
		return fmt.Errorf("parse config file %s: %w", path, err)
	}
	c.configDir = filepath.Dir(path)
	return nil
}

// applyEnv applies environment overrides.
//
// Env vars:
//
//	INCUS_MCP_TARGET_URL      base URL of the target
//	INCUS_MCP_CERT_PATH       client cert path
//	INCUS_MCP_KEY_PATH        client key path
//	INCUS_MCP_ADMIN_CERT_PATH admin client cert path
//	INCUS_MCP_ADMIN_KEY_PATH  admin client key path
//	INCUS_MCP_LISTEN_ADDR     listen address
//	INCUS_MCP_LISTEN_PORT     listen port
//	INCUS_MCP_PROJECT         default project
//	INCUS_MCP_WAIT_TIMEOUT    wait timeout (seconds)
func (c *Config) applyEnv() {
	setStr := func(envName string, dst *string) {
		if v := os.Getenv(envName); v != "" {
			*dst = v
		}
	}
	setInt := func(envName string, dst *int) {
		if v := os.Getenv(envName); v != "" {
			if n, err := strconv.Atoi(v); err == nil {
				*dst = n
			}
		}
	}

	setStr("INCUS_MCP_TARGET_URL", &c.Target.URL)
	setStr("INCUS_MCP_TARGET_CERT", &c.Target.CertPath)
	setStr("INCUS_MCP_CERT_PATH", &c.Credential.CertPath)
	setStr("INCUS_MCP_KEY_PATH", &c.Credential.KeyPath)
	setStr("INCUS_MCP_ADMIN_CERT_PATH", c.adminCertPathRef())
	setStr("INCUS_MCP_ADMIN_KEY_PATH", c.adminKeyPathRef())
	setStr("INCUS_MCP_LISTEN_ADDR", &c.Server.ListenAddr)
	setInt("INCUS_MCP_LISTEN_PORT", &c.Server.ListenPort)
	setStr("INCUS_MCP_PROJECT", &c.DefaultProject)
	setInt("INCUS_MCP_WAIT_TIMEOUT", &c.WaitTimeoutSeconds)
}

// adminCertPathRef lazily creates the admin credential when a path is provided.
func (c *Config) adminCertPathRef() *string {
	if c.AdminCredential == nil {
		c.AdminCredential = &Credential{}
	}
	return &c.AdminCredential.CertPath
}

// adminKeyPathRef is like adminCertPathRef for the key path.
func (c *Config) adminKeyPathRef() *string {
	if c.AdminCredential == nil {
		c.AdminCredential = &Credential{}
	}
	return &c.AdminCredential.KeyPath
}

// applyFlags applies flag-level overrides.
func (c *Config) applyFlags(opts Options) {
	if opts.TargetURL != "" {
		c.Target.URL = opts.TargetURL
	}
	if opts.TargetCert != "" {
		c.Target.CertPath = opts.TargetCert
	}
	if opts.CertPath != "" {
		c.Credential.CertPath = opts.CertPath
	}
	if opts.KeyPath != "" {
		c.Credential.KeyPath = opts.KeyPath
	}
	if opts.AdminCertPath != "" {
		if c.AdminCredential == nil {
			c.AdminCredential = &Credential{}
		}
		c.AdminCredential.CertPath = opts.AdminCertPath
	}
	if opts.AdminKeyPath != "" {
		if c.AdminCredential == nil {
			c.AdminCredential = &Credential{}
		}
		c.AdminCredential.KeyPath = opts.AdminKeyPath
	}
	if opts.ListenAddr != "" {
		c.Server.ListenAddr = opts.ListenAddr
	}
	if opts.ListenPort != 0 {
		c.Server.ListenPort = opts.ListenPort
	}
	if opts.DefaultProject != "" {
		c.DefaultProject = opts.DefaultProject
	}
	if opts.WaitTimeout != 0 {
		c.WaitTimeoutSeconds = opts.WaitTimeout
	}
}

// Validate checks the effective configuration for consistency.
func (c *Config) Validate() error {
	if c.Target.URL == "" {
		return fmt.Errorf("target.url is required")
	}
	if !strings.HasPrefix(c.Target.URL, "https://") && !strings.HasPrefix(c.Target.URL, "http://") {
		return fmt.Errorf("target.url must be an absolute http(s) URL")
	}
	if c.Credential.CertPath == "" || c.Credential.KeyPath == "" {
		return fmt.Errorf("credential.cert_path and credential.key_path are required")
	}
	if (c.Credential.CertPath != "") != (c.Credential.KeyPath != "") {
		return fmt.Errorf("credential cert and key must be provided together")
	}
	if c.AdminCredential != nil {
		if (c.AdminCredential.CertPath == "") != (c.AdminCredential.KeyPath == "") {
			return fmt.Errorf("admin_credential cert and key must be provided together")
		}
	}
	if c.Server.ListenPort < 1 || c.Server.ListenPort > 65535 {
		return fmt.Errorf("server.listen_port out of range: %d", c.Server.ListenPort)
	}
	if c.WaitTimeoutSeconds < 1 {
		return fmt.Errorf("wait_timeout_seconds must be >= 1")
	}
	if c.InlineMaxBytes < 1 {
		return fmt.Errorf("inline_max_bytes must be >= 1")
	}
	if c.DefaultProject == "" {
		return fmt.Errorf("default_project must not be empty")
	}
	return nil
}

// ListenAddress returns the resolved "addr:port" listen address.
func (c *Config) ListenAddress() string {
	return fmt.Sprintf("%s:%d", c.Server.ListenAddr, c.Server.ListenPort)
}

// TargetCertificatePath returns the explicit target pin path or the
// conventional target.crt path beside the effective config file.
func (c *Config) TargetCertificatePath() string {
	if c.Target.CertPath != "" {
		return c.Target.CertPath
	}
	dir := c.configDir
	if dir == "" {
		dir = filepath.Dir(DefaultConfigFile())
	}
	return filepath.Join(dir, "target.crt")
}

// DefaultConfigFile returns the conventional config path.
func DefaultConfigFile() string {
	if p := os.Getenv("INCUS_MCP_CONFIG"); p != "" {
		return p
	}
	// XDG-style config dir; falls back to the user's home.
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "incus-os-mcp", "config.json")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "config.json"
	}
	return filepath.Join(home, ".config", "incus-os-mcp", "config.json")
}
