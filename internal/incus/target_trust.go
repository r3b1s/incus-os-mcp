package incus

import (
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"fmt"
	"io/fs"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const targetCertificateTimeout = 10 * time.Second

// TargetTrust describes the certificate pin used for the target connection.
type TargetTrust struct {
	Path        string
	Fingerprint string
	FirstUse    bool
}

type resolvedTargetCertificate struct {
	PEM   string
	Trust TargetTrust
}

// resolveTargetCertificate loads an existing pin or performs a bounded TOFU
// handshake and persists the presented leaf certificate before any Incus API
// request is made.
func resolveTargetCertificate(targetURL, certPath string) (resolvedTargetCertificate, error) {
	u, err := url.Parse(targetURL)
	if err != nil {
		return resolvedTargetCertificate{}, fmt.Errorf("parse target URL: %w", err)
	}
	if u.Scheme != "https" {
		return resolvedTargetCertificate{}, nil
	}
	if u.Hostname() == "" {
		return resolvedTargetCertificate{}, fmt.Errorf("target URL has no hostname")
	}
	if certPath == "" {
		return resolvedTargetCertificate{}, fmt.Errorf("target certificate path is empty")
	}

	resolved, err := loadTargetCertificate(certPath)
	if err == nil {
		return resolved, nil
	}
	if !errors.Is(err, fs.ErrNotExist) {
		return resolvedTargetCertificate{}, err
	}

	candidate, err := fetchTargetCertificate(u)
	if err != nil {
		return resolvedTargetCertificate{}, err
	}
	candidate.Trust.Path = certPath
	candidate.Trust.FirstUse = true

	created, err := publishTargetCertificate(certPath, []byte(candidate.PEM))
	if err != nil {
		return resolvedTargetCertificate{}, err
	}
	if created {
		return candidate, nil
	}

	// Another process won the no-clobber publication race. Its pin is
	// authoritative; never replace it with this process's candidate.
	return loadTargetCertificate(certPath)
}

func loadTargetCertificate(path string) (resolvedTargetCertificate, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return resolvedTargetCertificate{}, fmt.Errorf("read target certificate %s: %w", path, err)
	}
	cert, err := parseLeafCertificate(data)
	if err != nil {
		return resolvedTargetCertificate{}, fmt.Errorf("parse target certificate %s: %w", path, err)
	}
	return resolvedTargetCertificate{
		PEM: string(data),
		Trust: TargetTrust{
			Path:        path,
			Fingerprint: certificateFingerprint(cert),
		},
	}, nil
}

func fetchTargetCertificate(u *url.URL) (resolvedTargetCertificate, error) {
	host := u.Hostname()
	port := u.Port()
	if port == "" {
		port = "443"
	}
	serverName := host
	if zone := strings.LastIndex(serverName, "%"); zone >= 0 {
		serverName = serverName[:zone]
	}

	// InsecureSkipVerify is intentionally limited to this first-use handshake.
	// No HTTP request is sent through this connection. The captured certificate
	// is persisted before the normal Incus client reconnects with verification.
	tlsConfig := &tls.Config{ // #nosec G402 -- explicit trust-on-first-use bootstrap
		InsecureSkipVerify: true,
		MinVersion:         tls.VersionTLS12,
		ServerName:         serverName,
	}
	dialer := &net.Dialer{Timeout: targetCertificateTimeout}
	conn, err := tls.DialWithDialer(dialer, "tcp", net.JoinHostPort(host, port), tlsConfig)
	if err != nil {
		return resolvedTargetCertificate{}, fmt.Errorf("retrieve target certificate from %s: %w", u.Host, err)
	}
	defer conn.Close()

	peer := conn.ConnectionState().PeerCertificates
	if len(peer) == 0 {
		return resolvedTargetCertificate{}, fmt.Errorf("retrieve target certificate from %s: peer presented no certificate", u.Host)
	}
	leaf := peer[0]
	pemBytes := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: leaf.Raw})
	if len(pemBytes) == 0 {
		return resolvedTargetCertificate{}, fmt.Errorf("encode target certificate from %s", u.Host)
	}

	return resolvedTargetCertificate{
		PEM: string(pemBytes),
		Trust: TargetTrust{
			Fingerprint: certificateFingerprint(leaf),
		},
	}, nil
}

func parseLeafCertificate(data []byte) (*x509.Certificate, error) {
	block, _ := pem.Decode(data)
	if block == nil || block.Type != "CERTIFICATE" {
		return nil, fmt.Errorf("PEM certificate block not found")
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, err
	}
	return cert, nil
}

func certificateFingerprint(cert *x509.Certificate) string {
	sum := sha256.Sum256(cert.Raw)
	raw := strings.ToUpper(hex.EncodeToString(sum[:]))
	parts := make([]string, 0, len(raw)/2)
	for i := 0; i < len(raw); i += 2 {
		parts = append(parts, raw[i:i+2])
	}
	return strings.Join(parts, ":")
}

// publishTargetCertificate publishes a complete PEM without replacing an
// existing path. The hard-link step is atomic and fails with EEXIST when
// another process has already established trust.
func publishTargetCertificate(path string, data []byte) (bool, error) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return false, fmt.Errorf("create target certificate directory %s: %w", dir, err)
	}

	tmp, err := os.CreateTemp(dir, ".target.crt-*")
	if err != nil {
		return false, fmt.Errorf("create temporary target certificate: %w", err)
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)

	closeWithError := func() {
		_ = tmp.Close()
	}
	if err := tmp.Chmod(0o644); err != nil {
		closeWithError()
		return false, fmt.Errorf("set temporary target certificate permissions: %w", err)
	}
	if _, err := tmp.Write(data); err != nil {
		closeWithError()
		return false, fmt.Errorf("write temporary target certificate: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		closeWithError()
		return false, fmt.Errorf("sync temporary target certificate: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return false, fmt.Errorf("close temporary target certificate: %w", err)
	}

	if err := os.Link(tmpPath, path); err != nil {
		if errors.Is(err, fs.ErrExist) {
			return false, nil
		}
		return false, fmt.Errorf("publish target certificate %s: %w", path, err)
	}
	dirHandle, err := os.Open(dir)
	if err != nil {
		return true, fmt.Errorf("open target certificate directory for sync: %w", err)
	}
	defer dirHandle.Close()
	if err := dirHandle.Sync(); err != nil {
		return true, fmt.Errorf("sync target certificate directory: %w", err)
	}
	return true, nil
}
