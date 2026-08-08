// Command mockincus is a minimal HTTPS mock of the Incus /1.0 API used for
// local integration testing of incus-os-mcp. It is NOT a real Incus server.
// It serves a fixed /1.0 server descriptor and a small instance list, enough
// for the MCP server to connect and exercise read-only tools.
package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
)

func main() {
	addr := ":18443"
	if len(os.Args) > 1 {
		addr = os.Args[1]
	}

	mux := http.NewServeMux()

	// Server descriptor (what ConnectIncus's eager GetServer needs).
	mux.HandleFunc("/1.0", func(w http.ResponseWriter, r *http.Request) {
		if r.TLS == nil || len(r.TLS.PeerCertificates) == 0 {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		writeJSON(w, map[string]any{
			"type":        "sync",
			"status":      "Success",
			"status_code": 200,
			"metadata": map[string]any{
				"api_extensions": []string{"instances", "images", "storage"},
				"api_status":     "stable",
				"api_version":    "1.0",
				"auth":           "tls",
				"auth_methods":   []string{"tls"},
				"environment": map[string]any{
					"server_name":    "mock-incus",
					"server_version": "6.0",
					"project":        "default",
				},
				"public": false,
			},
		})
	})

	// Instance list (read-only probe).
	mux.HandleFunc("/1.0/instances", func(w http.ResponseWriter, r *http.Request) {
		if r.TLS == nil || len(r.TLS.PeerCertificates) == 0 {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		writeJSON(w, map[string]any{
			"type":   "sync",
			"status": "Success",
			"metadata": []any{
				map[string]any{
					"name":     "demo1",
					"type":     "container",
					"status":   "Running",
					"config":   map[string]string{},
					"profiles": []string{"default"},
					"devices":  map[string]any{},
				},
				map[string]any{
					"name":     "demo2",
					"type":     "virtual-machine",
					"status":   "Stopped",
					"config":   map[string]string{},
					"profiles": []string{"default"},
					"devices":  map[string]any{},
				},
			},
		})
	})

	mux.HandleFunc("/1.0/instances/demo1", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{
			"type":   "sync",
			"status": "Success",
			"metadata": map[string]any{
				"name":     "demo1",
				"type":     "container",
				"status":   "Running",
				"config":   map[string]string{},
				"profiles": []string{"default"},
				"devices":  map[string]any{},
			},
		})
	})

	// TLS: require a client cert (any), serve a self-signed server cert.
	cert, err := tls.LoadX509KeyPair("/tmp/mcp-test/server.crt", "/tmp/mcp-test/server.key")
	if err != nil {
		log.Fatalf("load server cert: %v", err)
	}
	srv := &http.Server{
		Addr:    addr,
		Handler: mux,
		TLSConfig: &tls.Config{
			Certificates: []tls.Certificate{cert},
			// Allow the unauthenticated TLS-only TOFU handshake. HTTP handlers
			// still require a client certificate for Incus API requests.
			ClientAuth: tls.RequestClientCert,
		},
	}

	fmt.Println("mock incus listening on", addr)
	log.Fatal(srv.ListenAndServeTLS("", ""))
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}
