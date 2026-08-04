package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/lxc/incus/v7/shared/api"

	"incus-os-mcp/internal/config"
	"incus-os-mcp/internal/incus"
)

// cmdDoctor runs the 4-check health probe (CONTEXT.md):
//  1. config parse + precedence
//  2. cert/key load + file permissions
//  3. TLS handshake + /1.0 reachable (version + API extensions)
//  4. effective-permissions probe (read-only list; admin check if configured)
//
// Non-zero exit on any failure.
func cmdDoctor(args []string) error {
	fs := flag.NewFlagSet("doctor", flag.ExitOnError)
	var f flags
	f.register(fs)
	if err := fs.Parse(args); err != nil {
		return err
	}

	fail := func(step string, err error) error {
		fmt.Printf("[FAIL] %s: %v\n", step, err)
		return fmt.Errorf("doctor failed at %s", step)
	}

	// Check 1: config parse + precedence.
	cfg, err := f.loadConfig()
	if err != nil {
		return fail("config", err)
	}
	fmt.Printf("[ OK ] config: target=%s project=%s listen=%s\n",
		cfg.Target.URL, cfg.DefaultProject, cfg.ListenAddress())

	// Check 2: cert/key load + permissions.
	cred := cfg.Credential
	if err := checkCertFiles(cred.CertPath, cred.KeyPath); err != nil {
		return fail("credentials", err)
	}
	fmt.Printf("[ OK ] credentials: cert=%s key=%s (0600)\n", cred.CertPath, cred.KeyPath)

	if cfg.AdminCredential != nil && cfg.AdminCredential.CertPath != "" {
		if err := checkCertFiles(cfg.AdminCredential.CertPath, cfg.AdminCredential.KeyPath); err != nil {
			return fail("admin credentials", err)
		}
		fmt.Printf("[ OK ] admin credentials: cert=%s key=%s (0600)\n",
			cfg.AdminCredential.CertPath, cfg.AdminCredential.KeyPath)
	}

	// Check 3: connect + /1.0 reachable.
	client, err := incus.New(cfg)
	if err != nil {
		return fail("connect", err)
	}
	srv, _, err := client.Server.GetServer()
	if err != nil {
		return fail("api", err)
	}
	fmt.Printf("[ OK ] api: %s api_version=%s extensions=%d clustered=%v\n",
		srv.Environment.ServerName, srv.APIVersion, len(srv.APIExtensions), client.Server.IsClustered())

	// Check 4: effective-permissions probe (read-only).
	instances, err := client.Server.GetInstances(api.InstanceTypeAny)
	if err != nil {
		// A scoped cert may lack instance read — that's a permission finding,
		// not a crash.
		if _, ok := err.(interface{ Status() int }); ok {
			fmt.Printf("[WARN] instance listing denied (scoped cert): %v\n", err)
		} else {
			return fail("permissions probe", err)
		}
	} else {
		fmt.Printf("[ OK ] permissions: instance listing OK (%d instances)\n", len(instances))
	}

	if cfg.AdminCredential != nil {
		admin, err := client.AdminClient()
		if err == nil {
			if _, err := admin.GetCertificates(); err != nil {
				fmt.Printf("[WARN] admin certificate listing denied: %v\n", err)
			} else {
				fmt.Printf("[ OK ] admin permissions: certificate listing OK\n")
			}
		}
	}

	fmt.Println("doctor: all checks passed")
	return nil
}

func checkCertFiles(certPath, keyPath string) error {
	if certPath == "" || keyPath == "" {
		return fmt.Errorf("cert and key paths required")
	}
	_, err := incus.TLSCertificate(config.Credential{CertPath: certPath, KeyPath: keyPath})
	if err != nil {
		return err
	}
	for _, p := range []string{keyPath} {
		info, err := os.Stat(p)
		if err != nil {
			return fmt.Errorf("stat %s: %w", p, err)
		}
		if perm := info.Mode().Perm(); perm&0o077 != 0 {
			return fmt.Errorf("%s has loose permissions %04o (want 0600)", p, perm)
		}
	}
	return nil
}
