package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// cmdCert mints a dedicated client certificate for the MCP server (D4).
// The cert is trusted on the target via `incus config trust add` (admin step,
// documented in the bootstrap guide) — never reused as a human admin cert.
func cmdCert(args []string) error {
	fs := flag.NewFlagSet("cert setup", flag.ExitOnError)
	dir := fs.String("dir", "", "output directory (default ~/.config/incus-os-mcp)")
	name := fs.String("name", "mcp-server", "certificate name")
	if err := fs.Parse(args); err != nil {
		return err
	}

	dest := *dir
	if dest == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		dest = filepath.Join(home, ".config", "incus-os-mcp")
	}
	if err := os.MkdirAll(dest, 0o700); err != nil {
		return fmt.Errorf("create cert dir: %w", err)
	}

	certPath := filepath.Join(dest, *name+".crt")
	keyPath := filepath.Join(dest, *name+".key")

	if _, err := os.Stat(certPath); err == nil {
		return fmt.Errorf("certificate already exists at %s (refusing to overwrite; delete it first)", certPath)
	}

	// Generate a self-signed client certificate (RSA 3072, 10 years).
	cmd := exec.Command("openssl", "req", "-x509", "-newkey", "rsa:3072",
		"-keyout", keyPath, "-out", certPath, "-days", "3650", "-nodes",
		"-subj", "/CN="+*name)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("openssl failed: %v: %s", err, out)
	}

	// Restrict key permissions (0600).
	if err := os.Chmod(keyPath, 0o600); err != nil {
		return fmt.Errorf("chmod key: %w", err)
	}
	if err := os.Chmod(certPath, 0o644); err != nil {
		return fmt.Errorf("chmod cert: %w", err)
	}

	fmt.Printf("wrote certificate:\n  cert: %s\n  key:  %s (0600)\n", certPath, keyPath)
	fmt.Println("\nnext steps (on the target host, with an admin identity):")
	fmt.Printf("  incus config trust add %s --type client --restricted --projects default\n", certPath)
	fmt.Println("  (add --restricted and a project list to scope the certificate;")
	fmt.Println("   grant the cert's auth group permissions via `incus auth group` on Incus 7+)")
	fmt.Println("then set credential.cert_path / credential.key_path in the config file.")
	return nil
}
