package incus

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"incus-os-mcp/internal/config"
)

func TestResolveTargetCertificateTOFUAndReuse(t *testing.T) {
	target, _ := newTLSTarget(t, "first-target")
	pinPath := filepath.Join(t.TempDir(), "nested", "target.crt")

	resolved, err := resolveTargetCertificate(target.URL, pinPath)
	if err != nil {
		t.Fatal(err)
	}
	if !resolved.Trust.FirstUse {
		t.Fatal("first resolution did not report trust on first use")
	}
	if resolved.Trust.Path != pinPath {
		t.Fatalf("pin path = %q, want %q", resolved.Trust.Path, pinPath)
	}
	if resolved.Trust.Fingerprint == "" {
		t.Fatal("first-use fingerprint is empty")
	}
	info, err := os.Stat(pinPath)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0o644 {
		t.Fatalf("pin mode = %04o, want 0644", got)
	}
	firstPEM, err := os.ReadFile(pinPath)
	if err != nil {
		t.Fatal(err)
	}

	// Existing pins are loaded without contacting the target.
	target.Close()
	reused, err := resolveTargetCertificate(target.URL, pinPath)
	if err != nil {
		t.Fatalf("reuse contacted unavailable target or rejected valid pin: %v", err)
	}
	if reused.Trust.FirstUse {
		t.Fatal("existing pin incorrectly reported first use")
	}
	if reused.Trust.Fingerprint != resolved.Trust.Fingerprint {
		t.Fatalf("reused fingerprint = %q, want %q", reused.Trust.Fingerprint, resolved.Trust.Fingerprint)
	}
	if reused.PEM != string(firstPEM) {
		t.Fatal("reused PEM differs from persisted pin")
	}
}

func TestResolveTargetCertificateAcquisitionFailureCreatesNoPin(t *testing.T) {
	pinPath := filepath.Join(t.TempDir(), "target.crt")

	_, err := resolveTargetCertificate("https://127.0.0.1:1", pinPath)
	if err == nil {
		t.Fatal("expected acquisition failure")
	}
	if _, statErr := os.Stat(pinPath); !os.IsNotExist(statErr) {
		t.Fatalf("failed acquisition created a pin: %v", statErr)
	}
}

func TestResolveTargetCertificateInvalidExistingPinFailsClosed(t *testing.T) {
	target, _ := newTLSTarget(t, "invalid-pin-target")
	defer target.Close()
	pinPath := filepath.Join(t.TempDir(), "target.crt")
	original := []byte("not a certificate\n")
	if err := os.WriteFile(pinPath, original, 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := resolveTargetCertificate(target.URL, pinPath)
	if err == nil {
		t.Fatal("expected invalid existing pin error")
	}
	after, readErr := os.ReadFile(pinPath)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if string(after) != string(original) {
		t.Fatal("invalid existing pin was replaced")
	}
}

func TestPublishTargetCertificateConcurrentNoClobber(t *testing.T) {
	pinPath := filepath.Join(t.TempDir(), "target.crt")
	candidates := [][]byte{[]byte("candidate-one"), []byte("candidate-two")}
	start := make(chan struct{})
	results := make(chan bool, len(candidates))
	errs := make(chan error, len(candidates))
	var wg sync.WaitGroup
	for _, candidate := range candidates {
		candidate := candidate
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			created, err := publishTargetCertificate(pinPath, candidate)
			results <- created
			errs <- err
		}()
	}
	close(start)
	wg.Wait()
	close(results)
	close(errs)

	createdCount := 0
	for created := range results {
		if created {
			createdCount++
		}
	}
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	if createdCount != 1 {
		t.Fatalf("created count = %d, want 1", createdCount)
	}
	persisted, err := os.ReadFile(pinPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(persisted) != string(candidates[0]) && string(persisted) != string(candidates[1]) {
		t.Fatalf("persisted unexpected candidate %q", persisted)
	}
}

func TestNewAcquiresPinBeforeIncusConnection(t *testing.T) {
	target, cred := newTLSTarget(t, "new-client-target")
	defer target.Close()
	pinPath := filepath.Join(t.TempDir(), "target.crt")
	cfg := config.Default()
	cfg.Target.URL = target.URL
	cfg.Target.CertPath = pinPath
	cfg.Credential = cred

	client, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if !client.TargetTrust.FirstUse {
		t.Fatal("New did not expose first-use trust metadata")
	}
	if client.TargetTrust.Path != pinPath {
		t.Fatalf("pin path = %q, want %q", client.TargetTrust.Path, pinPath)
	}
	if _, _, err := client.Server.GetServer(); err != nil {
		t.Fatalf("verified Incus API connection failed: %v", err)
	}
}

func TestExistingPinRejectsDifferentTarget(t *testing.T) {
	first, cred := newTLSTarget(t, "first-target")
	pinPath := filepath.Join(t.TempDir(), "target.crt")
	pinned, err := resolveTargetCertificate(first.URL, pinPath)
	if err != nil {
		t.Fatal(err)
	}
	first.Close()
	before, err := os.ReadFile(pinPath)
	if err != nil {
		t.Fatal(err)
	}

	second, _ := newTLSTarget(t, "second-target")
	defer second.Close()
	cfg := config.Default()
	cfg.Target.URL = second.URL
	cfg.Target.CertPath = pinPath
	cfg.Credential = cred
	_, err = New(cfg)
	if err == nil {
		t.Fatal("connection with a mismatched target certificate succeeded")
	}
	if !strings.Contains(err.Error(), pinPath) || !strings.Contains(err.Error(), pinned.Trust.Fingerprint) {
		t.Fatalf("mismatch error did not identify authoritative pin: %v", err)
	}
	after, err := os.ReadFile(pinPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(before) {
		t.Fatal("mismatch changed the existing pin")
	}
}

func TestHTTPDoesNotAcquireTargetCertificate(t *testing.T) {
	pinPath := filepath.Join(t.TempDir(), "target.crt")
	resolved, err := resolveTargetCertificate("http://127.0.0.1:8443", pinPath)
	if err != nil {
		t.Fatal(err)
	}
	if resolved.PEM != "" || resolved.Trust.Path != "" {
		t.Fatalf("HTTP target unexpectedly resolved TLS trust: %+v", resolved.Trust)
	}
	if _, err := os.Stat(pinPath); !os.IsNotExist(err) {
		t.Fatalf("HTTP target created certificate pin: %v", err)
	}
}

func newTLSTarget(t *testing.T, commonName string) (*httptest.Server, config.Credential) {
	t.Helper()
	serverCert, _, _ := newTestCertificate(t, commonName, true)
	_, cred := newClientCredential(t, commonName+"-client")

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"type":        "sync",
			"status":      "Success",
			"status_code": 200,
			"metadata": map[string]any{
				"api_extensions": []string{},
				"api_status":     "stable",
				"api_version":    "1.0",
				"auth":           "trusted",
				"auth_methods":   []string{"tls"},
				"environment": map[string]any{
					"server_name":    commonName,
					"server_version": "test",
					"project":        "default",
				},
				"public": false,
			},
		})
	})
	target := httptest.NewUnstartedServer(handler)
	target.TLS = &tls.Config{
		Certificates: []tls.Certificate{serverCert},
		ClientAuth:   tls.RequestClientCert,
		MinVersion:   tls.VersionTLS12,
	}
	target.StartTLS()
	return target, cred
}

func newClientCredential(t *testing.T, commonName string) (tls.Certificate, config.Credential) {
	t.Helper()
	cert, certPEM, keyPEM := newTestCertificate(t, commonName, false)
	dir := t.TempDir()
	certPath := filepath.Join(dir, "client.crt")
	keyPath := filepath.Join(dir, "client.key")
	if err := os.WriteFile(certPath, certPEM, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(keyPath, keyPEM, 0o600); err != nil {
		t.Fatal(err)
	}
	return cert, config.Credential{CertPath: certPath, KeyPath: keyPath}
}

func newTestCertificate(t *testing.T, commonName string, server bool) (tls.Certificate, []byte, []byte) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		t.Fatal(err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: serial,
		Subject:      pkix.Name{CommonName: commonName},
		NotBefore:    time.Now().Add(-time.Minute),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
	}
	if server {
		tmpl.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}
		tmpl.IPAddresses = []net.IP{net.ParseIP("127.0.0.1")}
		tmpl.DNSNames = []string{"localhost"}
	} else {
		tmpl.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
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
