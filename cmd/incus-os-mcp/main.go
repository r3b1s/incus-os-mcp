// Command incus-os-mcp runs the IncusOS MCP server and its operator CLI.
//
// Subcommands:
//
//	run            start the MCP server (streamable HTTP)
//	config init    write a default config file
//	config show    print the effective configuration
//	config validate  check the effective configuration
//	doctor         run connectivity/health checks against the target
//	cert setup     mint a client certificate for the server (openssl)
//
// Everything deployment-specific comes from configuration (flags > env >
// config file > defaults); nothing is compiled in.
package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	var err error
	switch os.Args[1] {
	case "run":
		err = cmdRun(os.Args[2:])
	case "config":
		err = cmdConfig(os.Args[2:])
	case "doctor":
		err = cmdDoctor(os.Args[2:])
	case "cert":
		err = cmdCert(os.Args[2:])
	case "version":
		fmt.Println("incus-os-mcp 0.1.0")
	case "help", "-h", "--help":
		usage()
	default:
		usage()
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "incus-os-mcp: %v\n", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprint(os.Stderr, `incus-os-mcp — MCP server for IncusOS/Incus

Usage:
  incus-os-mcp run [flags]        Start the MCP server (streamable HTTP)
  incus-os-mcp config init        Write a default config file
  incus-os-mcp config show        Print the effective configuration
  incus-os-mcp config validate    Validate the effective configuration
  incus-os-mcp doctor             Run health checks against the target
  incus-os-mcp cert setup         Mint a client certificate (openssl)
  incus-os-mcp version            Print version

Common flags:
  --config PATH    config file (default ~/.config/incus-os-mcp/config.json)
  --target URL     IncusOS/Incus base URL
  --cert PATH      client cert path
  --key PATH       client key path
  --admin-cert PATH  admin client cert path
  --admin-key PATH   admin client key path
  --listen ADDR    listen address (default 127.0.0.1)
  --port N         listen port (default 8002)
  --project NAME   default project (default "default")
  --wait-timeout N wait timeout seconds (default 60)

Environment: INCUS_MCP_TARGET_URL, INCUS_MCP_CERT_PATH, INCUS_MCP_KEY_PATH,
INCUS_MCP_ADMIN_CERT_PATH, INCUS_MCP_ADMIN_KEY_PATH, INCUS_MCP_LISTEN_ADDR,
INCUS_MCP_LISTEN_PORT, INCUS_MCP_PROJECT, INCUS_MCP_WAIT_TIMEOUT.
`)
}
