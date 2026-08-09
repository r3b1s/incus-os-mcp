package server

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"io"
	"log/slog"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"incus-os-mcp/internal/config"
	incusbridge "incus-os-mcp/internal/incus"
)

type capturedRequest struct {
	Method  string
	Path    string
	Query   string
	Headers http.Header
	Body    []byte
}

type protocolTarget struct {
	server   *httptest.Server
	mu       sync.Mutex
	requests []capturedRequest
}

func (tgt *protocolTarget) capture(r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	tgt.mu.Lock()
	defer tgt.mu.Unlock()
	tgt.requests = append(tgt.requests, capturedRequest{
		Method: r.Method, Path: r.URL.Path, Query: r.URL.RawQuery, Headers: r.Header.Clone(), Body: body,
	})
}

func (tgt *protocolTarget) callsFor(path string) []capturedRequest {
	tgt.mu.Lock()
	defer tgt.mu.Unlock()
	var calls []capturedRequest
	for _, request := range tgt.requests {
		if request.Path == path {
			calls = append(calls, request)
		}
	}
	return calls
}

func TestInstanceListUsesScopedProjectAndObjectEnvelope(t *testing.T) {
	target, cfg := newProtocolTarget(t)
	defer target.server.Close()

	client, err := incusbridge.New(cfg)
	if err != nil {
		t.Fatalf("connect real Incus client: %v", err)
	}
	mcpServer := httptest.NewServer(New(cfg, client, slog.New(slog.NewTextHandler(io.Discard, nil))).Handler())
	defer mcpServer.Close()

	response := callMCP(t, mcpServer.URL, 1, "instance_list", map[string]any{"project": "team-a"})
	result := response["result"].(map[string]any)
	if isError, _ := result["isError"].(bool); isError {
		t.Fatalf("instance_list returned tool error: %#v", result["content"])
	}
	structured, ok := result["structuredContent"].(map[string]any)
	if !ok {
		t.Fatalf("structuredContent must be an object, got %#v", result["structuredContent"])
	}
	items, ok := structured["items"].([]any)
	if !ok || len(items) != 1 {
		t.Fatalf("expected object envelope with one item, got %#v", structured)
	}

	calls := target.callsFor("/1.0/instances")
	if len(calls) != 1 || calls[0].Query != "project=team-a&recursion=1" {
		t.Fatalf("instance list must use requested project exactly once, got %#v", calls)
	}
}

func TestInstanceListRejectsAmbiguousAllProjectsScopeBeforeTargetCall(t *testing.T) {
	target, cfg := newProtocolTarget(t)
	defer target.server.Close()
	client, err := incusbridge.New(cfg)
	if err != nil {
		t.Fatalf("connect real Incus client: %v", err)
	}
	mcpServer := httptest.NewServer(New(cfg, client, slog.New(slog.NewTextHandler(io.Discard, nil))).Handler())
	defer mcpServer.Close()

	response := callMCP(t, mcpServer.URL, 2, "instance_list", map[string]any{"project": "team-a", "all_projects": true})
	result := response["result"].(map[string]any)
	if isError, _ := result["isError"].(bool); !isError {
		t.Fatalf("ambiguous scope must be rejected: %#v", result)
	}
	if calls := target.callsFor("/1.0/instances"); len(calls) != 0 {
		t.Fatalf("ambiguous scope must not reach target, got %#v", calls)
	}
}

func TestStorageVolumeImportISOStreamsScopedArtifact(t *testing.T) {
	target, cfg := newProtocolTarget(t)
	defer target.server.Close()
	client, err := incusbridge.New(cfg)
	if err != nil {
		t.Fatalf("connect real Incus client: %v", err)
	}
	mcpServer := httptest.NewServer(New(cfg, client, slog.New(slog.NewTextHandler(io.Discard, nil))).Handler())
	defer mcpServer.Close()

	isoContents := testISOArtifact()
	response := callMCP(t, mcpServer.URL, 3, "storage_volume_import_iso", map[string]any{
		"pool": "testpool", "name": "combustion-run", "project": "team-a", "artifact_base64": base64.StdEncoding.EncodeToString(isoContents),
	})
	result, ok := response["result"].(map[string]any)
	if !ok {
		t.Fatalf("ISO import must return an MCP tool result, got %#v", response)
	}
	if isError, _ := result["isError"].(bool); isError {
		t.Fatalf("ISO import returned tool error: %#v", result["content"])
	}
	operation, ok := result["structuredContent"].(map[string]any)
	if !ok {
		t.Fatalf("ISO import must return structured operation output, got %#v", result)
	}
	if _, ok := operation["resources"].(map[string]any); !ok {
		t.Fatalf("operation resources must be an object, got %#v", operation["resources"])
	}
	if _, ok := operation["metadata"].(map[string]any); !ok {
		t.Fatalf("operation metadata must be an object, got %#v", operation["metadata"])
	}
	calls := target.callsFor("/1.0/storage-pools/testpool/volumes/custom")
	if len(calls) != 1 {
		t.Fatalf("expected one ISO upload request, got %#v", calls)
	}
	call := calls[0]
	if call.Method != http.MethodPost || call.Query != "project=team-a" || call.Headers.Get("Content-Type") != "application/octet-stream" || call.Headers.Get("X-Incus-name") != "combustion-run" || call.Headers.Get("X-Incus-type") != "iso" || !bytes.Equal(call.Body, isoContents) {
		t.Fatalf("unexpected ISO upload request: %#v", call)
	}
}

func TestStorageVolumeImportISORejectsInvalidArtifactBeforeTargetCall(t *testing.T) {
	target, cfg := newProtocolTarget(t)
	defer target.server.Close()
	client, err := incusbridge.New(cfg)
	if err != nil {
		t.Fatalf("connect real Incus client: %v", err)
	}
	mcpServer := httptest.NewServer(New(cfg, client, slog.New(slog.NewTextHandler(io.Discard, nil))).Handler())
	defer mcpServer.Close()

	response := callMCP(t, mcpServer.URL, 4, "storage_volume_import_iso", map[string]any{
		"pool": "testpool", "name": "should-not-upload", "project": "team-a", "artifact_base64": base64.StdEncoding.EncodeToString([]byte("not an ISO")),
	})
	result, ok := response["result"].(map[string]any)
	if !ok {
		t.Fatalf("invalid ISO must return an MCP tool result, got %#v", response)
	}
	if isError, _ := result["isError"].(bool); !isError {
		t.Fatalf("invalid ISO must return a tool error: %#v", result)
	}
	if calls := target.callsFor("/1.0/storage-pools/testpool/volumes/custom"); len(calls) != 0 {
		t.Fatalf("invalid ISO must not upload data, got %#v", calls)
	}
}

func TestDecodeISOArtifactRejectsMalformedBase64(t *testing.T) {
	t.Parallel()

	file, err := decodeISOArtifact("not-valid-base64!")
	if err == nil {
		if file != nil {
			_ = file.Close()
			_ = os.Remove(file.Name())
		}
		t.Fatal("malformed base64 must be rejected")
	}
	if file != nil {
		t.Fatalf("malformed base64 must not return a temporary file: %s", file.Name())
	}
}

func testISOArtifact() []byte {
	artifact := make([]byte, isoPrimaryDescriptorOffset+7)
	copy(artifact[isoPrimaryDescriptorOffset:], []byte{1, 'C', 'D', '0', '0', '1', 1})
	return artifact
}

func callMCP(t *testing.T, url string, id int, name string, arguments map[string]any) map[string]any {
	t.Helper()
	request := map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"method":  "tools/call",
		"params": map[string]any{
			"name":      name,
			"arguments": arguments,
		},
	}
	payload, err := json.Marshal(request)
	if err != nil {
		t.Fatal(err)
	}
	httpRequest, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		t.Fatal(err)
	}
	httpRequest.Header.Set("Content-Type", "application/json")
	httpRequest.Header.Set("Accept", "application/json, text/event-stream")
	httpRequest.Header.Set("MCP-Protocol-Version", "2025-11-25")
	response, err := http.DefaultClient.Do(httpRequest)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(response.Body)
		t.Fatalf("MCP HTTP status %d: %s", response.StatusCode, body)
	}
	var decoded map[string]any
	if err := json.NewDecoder(response.Body).Decode(&decoded); err != nil {
		t.Fatal(err)
	}
	return decoded
}

func newProtocolTarget(t *testing.T) (*protocolTarget, *config.Config) {
	t.Helper()
	serverCert, serverPEM, _ := testCertificate(t, "127.0.0.1", true)
	_, clientPEM, clientKeyPEM := testCertificate(t, "protocol-client", false)
	dir := t.TempDir()
	serverPath := filepath.Join(dir, "server.crt")
	clientPath := filepath.Join(dir, "client.crt")
	keyPath := filepath.Join(dir, "client.key")
	for path, contents := range map[string]struct {
		contents []byte
		mode     os.FileMode
	}{
		serverPath: {contents: serverPEM, mode: 0o644},
		clientPath: {contents: clientPEM, mode: 0o644},
		keyPath:    {contents: clientKeyPEM, mode: 0o600},
	} {
		if err := os.WriteFile(path, contents.contents, contents.mode); err != nil {
			t.Fatal(err)
		}
	}

	target := &protocolTarget{}
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		target.capture(r)
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/1.0":
			_ = json.NewEncoder(w).Encode(syncEnvelope(map[string]any{
				"api_extensions": []string{"instances", "custom_volume_iso"},
				"api_status":     "stable",
				"api_version":    "1.0",
				"auth":           "trusted",
				"auth_methods":   []string{"tls"},
				"environment": map[string]any{
					"server_name": "protocol-test",
					"project":     "default",
				},
				"public": false,
			}))
		case "/1.0/instances":
			_ = json.NewEncoder(w).Encode(syncEnvelope([]any{map[string]any{
				"name":     "scoped-vm",
				"type":     "virtual-machine",
				"status":   "Stopped",
				"config":   map[string]string{},
				"profiles": []string{"default"},
				"devices":  map[string]any{},
			}}))
		case "/1.0/storage-pools/testpool/volumes/custom":
			_ = json.NewEncoder(w).Encode(asyncOperationEnvelope("op-iso"))
		case "/1.0/operations/op-iso/wait":
			_ = json.NewEncoder(w).Encode(syncEnvelope(map[string]any{
				"id": "op-iso", "class": "task", "status": "Success", "status_code": 200,
			}))
		default:
			http.NotFound(w, r)
		}
	})
	target.server = httptest.NewUnstartedServer(handler)
	target.server.TLS = &tls.Config{Certificates: []tls.Certificate{serverCert}, ClientAuth: tls.RequestClientCert, MinVersion: tls.VersionTLS12}
	target.server.StartTLS()

	cfg := config.Default()
	cfg.Target.URL = target.server.URL
	cfg.Target.CertPath = serverPath
	cfg.Credential = config.Credential{CertPath: clientPath, KeyPath: keyPath}
	cfg.DefaultProject = "default"
	return target, cfg
}

func syncEnvelope(metadata any) map[string]any {
	return map[string]any{"type": "sync", "status": "Success", "status_code": 200, "metadata": metadata}
}

func asyncOperationEnvelope(id string) map[string]any {
	return map[string]any{
		"type": "async", "status": "Operation created", "status_code": 100,
		"operation": "/1.0/operations/" + id,
		"metadata":  map[string]any{"id": id, "class": "task", "status": "Success", "status_code": 200},
	}
}

func testCertificate(t *testing.T, commonName string, server bool) (tls.Certificate, []byte, []byte) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		t.Fatal(err)
	}
	template := &x509.Certificate{SerialNumber: serial, Subject: pkix.Name{CommonName: commonName}, NotBefore: time.Now().Add(-time.Minute), NotAfter: time.Now().Add(time.Hour), KeyUsage: x509.KeyUsageDigitalSignature}
	if server {
		template.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}
		template.IPAddresses = []net.IP{net.ParseIP("127.0.0.1")}
	} else {
		template.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	keyDER, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyDER})
	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		t.Fatal(err)
	}
	return cert, certPEM, keyPEM
}
