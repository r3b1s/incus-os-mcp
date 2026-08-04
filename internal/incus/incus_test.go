package incus

import (
	"errors"
	"strings"
	"testing"

	"github.com/lxc/incus/v7/shared/api"

	"incus-os-mcp/internal/config"
)

func configForTest() *config.Config {
	c := config.Default()
	c.Target.URL = "https://x.test:8443"
	c.Credential = config.Credential{CertPath: "/c", KeyPath: "/k"}
	return c
}

func TestMapError403(t *testing.T) {
	err := api.StatusErrorf(403, "permission denied")
	mapped := MapError("instance_create", err)
	var perm *ErrPermission
	if !errors.As(mapped, &perm) {
		t.Fatalf("expected *ErrPermission, got %T: %v", mapped, mapped)
	}
	if perm.Operation != "instance_create" {
		t.Errorf("operation = %q", perm.Operation)
	}
}

func TestMapError404(t *testing.T) {
	err := api.StatusErrorf(404, "not found")
	mapped := MapError("instance_get", err)
	if mapped == nil {
		t.Fatal("nil error")
	}
	if !strings.Contains(mapped.Error(), "not found") {
		t.Errorf("expected 404 message to survive wrapping: %v", mapped)
	}
	if strings.Contains(mapped.Error(), "instance_get") == false {
		t.Errorf("expected operation name in message: %v", mapped)
	}
}

func TestMapErrorNil(t *testing.T) {
	if MapError("op", nil) != nil {
		t.Error("nil in, nil out")
	}
}

func TestMapErrorTLS(t *testing.T) {
	err := errors.New("tls: handshake failure")
	mapped := MapError("server_info", err)
	if mapped == nil {
		t.Fatal("nil error")
	}
}

func TestWaitTimeoutConfig(t *testing.T) {
	c := &Client{Config: configForTest()}
	if c.WaitTimeout().Seconds() != 60 {
		t.Errorf("timeout = %v", c.WaitTimeout())
	}
}

func TestProjectFallback(t *testing.T) {
	c := &Client{Config: configForTest()}
	if c.Project("") != "default" {
		t.Errorf("empty project should fall back to default")
	}
	if c.Project("other") != "other" {
		t.Errorf("explicit project should pass through")
	}
}

func TestAdminClientUnset(t *testing.T) {
	c := &Client{Config: configForTest()}
	if _, err := c.AdminClient(); err == nil {
		t.Error("expected error when admin credential unset")
	}
}
