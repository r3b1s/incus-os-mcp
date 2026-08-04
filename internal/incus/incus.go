// Package incus wraps the official Incus Go client (github.com/lxc/incus/v7)
// for use by the MCP tools. It is the single place that holds the target
// connection, the two credential identities, and the shared operation-wait
// helper.
package incus

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	incusclient "github.com/lxc/incus/v7/client"
	"github.com/lxc/incus/v7/shared/api"

	"incus-os-mcp/internal/config"
)

// Client wraps the target connection.
type Client struct {
	// Server is the primary InstanceServer (scoped credential).
	Server incusclient.InstanceServer
	// Admin is the admin InstanceServer, or nil when no admin credential is
	// configured.
	Admin incusclient.InstanceServer

	// Config is the resolved configuration (for project/timeout defaults).
	Config *config.Config
}

// New connects to the target using the configured credentials.
func New(cfg *config.Config) (*Client, error) {
	primary, err := connect(cfg.Target.URL, cfg.Target.CertPath, cfg.Credential)
	if err != nil {
		return nil, fmt.Errorf("connect with primary credential: %w", err)
	}

	var admin incusclient.InstanceServer
	if cfg.AdminCredential != nil && cfg.AdminCredential.CertPath != "" {
		admin, err = connect(cfg.Target.URL, cfg.Target.CertPath, *cfg.AdminCredential)
		if err != nil {
			return nil, fmt.Errorf("connect with admin credential: %w", err)
		}
	}

	return &Client{Server: primary, Admin: admin, Config: cfg}, nil
}

// connect dials the target with a credential. The Incus client expects PEM
// certificate/key *contents* (not paths), so we read them here. The optional
// target cert pins the server certificate (self-signed targets).
func connect(url, targetCertPath string, cred config.Credential) (incusclient.InstanceServer, error) {
	certPEM, err := os.ReadFile(cred.CertPath)
	if err != nil {
		return nil, fmt.Errorf("read client cert: %w", err)
	}
	keyPEM, err := os.ReadFile(cred.KeyPath)
	if err != nil {
		return nil, fmt.Errorf("read client key: %w", err)
	}

	var targetCertPEM string
	if targetCertPath != "" {
		b, err := os.ReadFile(targetCertPath)
		if err != nil {
			return nil, fmt.Errorf("read target cert: %w", err)
		}
		targetCertPEM = string(b)
	}

	return incusclient.ConnectIncus(url, &incusclient.ConnectionArgs{
		TLSClientCert: string(certPEM),
		TLSClientKey:  string(keyPEM),
		TLSServerCert: targetCertPEM,
		UserAgent:     "incus-os-mcp",
	})
}

// AdminClient returns the admin server or an error explaining that the
// operation requires admin credentials.
func (c *Client) AdminClient() (incusclient.InstanceServer, error) {
	if c.Admin == nil {
		return nil, fmt.Errorf("operation requires admin credentials: no admin_credential configured (certificate management and IncusOS system tools are admin-only)")
	}
	return c.Admin, nil
}

// Project resolves an optional project parameter to an effective project.
func (c *Client) Project(project string) string {
	if project == "" {
		return c.Config.DefaultProject
	}
	return project
}

// WaitTimeout returns the configured wait timeout.
func (c *Client) WaitTimeout() time.Duration {
	return time.Duration(c.Config.WaitTimeoutSeconds) * time.Second
}

// WaitOperation waits for an operation to complete, returning its final state.
//
// On timeout it returns a result indicating the operation is still running
// (status "running", with the operation ID/URL) — not an error — per the
// CONTEXT.md wait semantics. On failure it returns the operation's error.
func (c *Client) WaitOperation(ctx context.Context, op incusclient.Operation) (*api.Operation, error) {
	timeout := c.WaitTimeout()

	done := make(chan struct{})
	var final *api.Operation
	var opErr error

	go func() {
		defer close(done)
		err := op.Wait()
		if err != nil {
			opErr = fmt.Errorf("operation %s failed: %w", op.Get().ID, err)
			return
		}
		cur := op.Get()
		final = &cur
	}()

	select {
	case <-done:
		if opErr != nil {
			return nil, opErr
		}
		if final != nil {
			return final, nil
		}
		cur := op.Get()
		return &cur, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(timeout):
		// Operation still running; return current state, not an error.
		cur := op.Get()
		return &api.Operation{
			ID:     cur.ID,
			Status: "running",
		}, nil
	}
}

// ErrPermission is returned (wrapped) when the target reports a permission
// denial, so tools can surface a clean "requires admin / forbidden" error.
type ErrPermission struct {
	Operation string
	Details   string
}

func (e *ErrPermission) Error() string {
	msg := fmt.Sprintf("permission denied for %s", e.Operation)
	if e.Details != "" {
		msg += ": " + e.Details
	}
	return msg
}

// MapError converts a target error into a clean tool error.
//
// 403/permission responses become ErrPermission so a scoped certificate
// degrades gracefully (never crashes); everything else is wrapped with the
// operation context.
func MapError(op string, err error) error {
	if err == nil {
		return nil
	}

	// Structured Incus errors carry a StatusCode. Use stdlib errors.As so
	// wrapped StatusErrors are found too.
	var apiErr api.StatusError
	if errors.As(err, &apiErr) {
		switch apiErr.Status() {
		case 403:
			return &ErrPermission{Operation: op, Details: apiErr.Error()}
		case 401:
			return fmt.Errorf("%s: authentication failed (check credential): %w", op, err)
		case 404:
			return fmt.Errorf("%s: not found: %w", op, err)
		case 409:
			return fmt.Errorf("%s: conflict (resource exists or in use): %w", op, err)
		}
		return fmt.Errorf("%s: %w", op, err)
	}

	// TLS / connection errors.
	msg := err.Error()
	if strings.Contains(msg, "certificate") || strings.Contains(msg, "tls") {
		return fmt.Errorf("%s: TLS error (check credential/certificate): %w", op, err)
	}

	return fmt.Errorf("%s: %w", op, err)
}

// TLSCertificate is a convenience for the doctor check: loads the configured
// client cert/key and reports whether the pair is valid and key permissions
// are sane (no group/other access).
func TLSCertificate(cred config.Credential) (*tls.Certificate, error) {
	cert, err := tls.LoadX509KeyPair(cred.CertPath, cred.KeyPath)
	if err != nil {
		return nil, fmt.Errorf("load client certificate: %w", err)
	}
	return &cert, nil
}
