package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaults(t *testing.T) {
	c := Default()
	if c.Server.ListenAddr != "127.0.0.1" {
		t.Errorf("listen addr = %q, want 127.0.0.1", c.Server.ListenAddr)
	}
	if c.Server.ListenPort != 8002 {
		t.Errorf("listen port = %d, want 8002", c.Server.ListenPort)
	}
	if c.DefaultProject != "default" {
		t.Errorf("project = %q, want default", c.DefaultProject)
	}
	if c.WaitTimeoutSeconds != 60 {
		t.Errorf("wait timeout = %d, want 60", c.WaitTimeoutSeconds)
	}
}

func TestFileLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	content := `{
	  "target": {"url": "https://example.test:8443"},
	  "credential": {"cert_path": "/c.pem", "key_path": "/k.pem"},
	  "server": {"listen_port": 9000},
	  "default_project": "tenant-a"
	}`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	c := Default()
	if err := c.LoadFile(path); err != nil {
		t.Fatal(err)
	}
	if c.Target.URL != "https://example.test:8443" {
		t.Errorf("url = %q", c.Target.URL)
	}
	if c.Server.ListenPort != 9000 {
		t.Errorf("port = %d, want 9000", c.Server.ListenPort)
	}
	if c.DefaultProject != "tenant-a" {
		t.Errorf("project = %q", c.DefaultProject)
	}
	// Unset file fields keep defaults.
	if c.Server.ListenAddr != "127.0.0.1" {
		t.Errorf("listen addr = %q, want default", c.Server.ListenAddr)
	}
	if got, want := c.TargetCertificatePath(), filepath.Join(dir, "target.crt"); got != want {
		t.Errorf("target certificate path = %q, want %q", got, want)
	}
}

func TestExplicitTargetCertificatePath(t *testing.T) {
	c := Default()
	c.Target.CertPath = "/explicit/target.pem"
	if got := c.TargetCertificatePath(); got != "/explicit/target.pem" {
		t.Errorf("target certificate path = %q, want explicit path", got)
	}
}

func TestDefaultTargetCertificatePathUsesConfiguredLocation(t *testing.T) {
	t.Setenv("INCUS_MCP_CONFIG", filepath.Join(t.TempDir(), "custom", "config.json"))
	c := Default()
	if got, want := c.TargetCertificatePath(), filepath.Join(filepath.Dir(DefaultConfigFile()), "target.crt"); got != want {
		t.Errorf("target certificate path = %q, want %q", got, want)
	}
}

func TestPrecedenceFlagsOverFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	content := `{
	  "target": {"url": "https://file.test:8443"},
	  "credential": {"cert_path": "/c.pem", "key_path": "/k.pem"}
	}`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(Options{
		ConfigFile: path,
		TargetURL:  "https://flag.test:8443",
	})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Target.URL != "https://flag.test:8443" {
		t.Errorf("url = %q, want flag override", cfg.Target.URL)
	}
}

func TestEnvOverride(t *testing.T) {
	t.Setenv("INCUS_MCP_TARGET_URL", "https://env.test:8443")
	t.Setenv("INCUS_MCP_LISTEN_PORT", "9001")

	c := Default()
	c.applyEnv()
	if c.Target.URL != "https://env.test:8443" {
		t.Errorf("url = %q, want env override", c.Target.URL)
	}
	if c.Server.ListenPort != 9001 {
		t.Errorf("port = %d, want 9001", c.Server.ListenPort)
	}
}

func TestEnvAdminCredential(t *testing.T) {
	t.Setenv("INCUS_MCP_ADMIN_CERT_PATH", "/admin.crt")
	t.Setenv("INCUS_MCP_ADMIN_KEY_PATH", "/admin.key")

	c := Default()
	c.applyEnv()
	if c.AdminCredential == nil {
		t.Fatal("admin credential not created by env")
	}
	if c.AdminCredential.CertPath != "/admin.crt" {
		t.Errorf("admin cert = %q", c.AdminCredential.CertPath)
	}
}

func TestValidate(t *testing.T) {
	cases := []struct {
		name    string
		mutate  func(*Config)
		wantErr bool
	}{
		{"valid", func(c *Config) {
			c.Target.URL = "https://x.test:8443"
			c.Credential = Credential{CertPath: "/c", KeyPath: "/k"}
		}, false},
		{"missing url", func(c *Config) {
			c.Target.URL = ""
			c.Credential = Credential{CertPath: "/c", KeyPath: "/k"}
		}, true},
		{"missing credential", func(c *Config) {
			c.Target.URL = "https://x.test:8443"
		}, true},
		{"half credential", func(c *Config) {
			c.Target.URL = "https://x.test:8443"
			c.Credential = Credential{CertPath: "/c"}
		}, true},
		{"bad port", func(c *Config) {
			c.Target.URL = "https://x.test:8443"
			c.Credential = Credential{CertPath: "/c", KeyPath: "/k"}
			c.Server.ListenPort = 0
		}, true},
		{"admin half", func(c *Config) {
			c.Target.URL = "https://x.test:8443"
			c.Credential = Credential{CertPath: "/c", KeyPath: "/k"}
			c.AdminCredential = &Credential{CertPath: "/ac"}
		}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := Default()
			tc.mutate(c)
			err := c.Validate()
			if tc.wantErr && err == nil {
				t.Errorf("expected error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}
