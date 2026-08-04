package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"incus-os-mcp/internal/config"
)

func cmdConfig(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("config requires a subcommand: init, show, validate")
	}

	switch args[0] {
	case "init":
		return configInit(args[1:])
	case "show":
		return configShow(args[1:])
	case "validate":
		return configValidate(args[1:])
	default:
		return fmt.Errorf("unknown config subcommand %q (want init, show, validate)", args[0])
	}
}

func configInit(args []string) error {
	fs := flag.NewFlagSet("config init", flag.ExitOnError)
	path := fs.String("path", "", "output path (default ~/.config/incus-os-mcp/config.json)")
	if err := fs.Parse(args); err != nil {
		return err
	}

	dest := *path
	if dest == "" {
		dest = config.DefaultConfigFile()
	}

	cfg := config.Default()
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')

	if err := os.MkdirAll(filepath.Dir(dest), 0o700); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}
	if err := os.WriteFile(dest, data, 0o600); err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	fmt.Printf("wrote default config to %s\n", dest)
	fmt.Println("edit target.url, credential.cert_path and credential.key_path before running.")
	return nil
}

func configShow(args []string) error {
	fs := flag.NewFlagSet("config show", flag.ExitOnError)
	var f flags
	f.register(fs)
	if err := fs.Parse(args); err != nil {
		return err
	}

	cfg, err := f.loadConfig()
	if err != nil {
		return err
	}

	// Redact nothing: paths are not secrets, but never print key material.
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	return nil
}

func configValidate(args []string) error {
	fs := flag.NewFlagSet("config validate", flag.ExitOnError)
	var f flags
	f.register(fs)
	if err := fs.Parse(args); err != nil {
		return err
	}

	cfg, err := f.loadConfig()
	if err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}
	_ = cfg
	fmt.Println("configuration OK")
	return nil
}
