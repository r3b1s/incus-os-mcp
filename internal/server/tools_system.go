package server

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Appliance API base path. The IncusOS fallback listener trims the "/os"
// prefix and requires a trusted client certificate; the official Go client's
// RawQuery lets us reach the proxied appliance endpoints (spike, task 1.1).
const osAPIPrefix = "/os/1.0"

// rawOS performs a RawQuery against the appliance API. Path is relative to
// /os/1.0 (e.g. "system/update").
func (s *Server) rawOS(ctx context.Context, method, path string, data any, out any) error {
	resp, _, err := s.client.Server.RawQuery(method, osAPIPrefix+"/"+path, data, "")
	if err != nil {
		return err
	}
	if resp == nil {
		return fmt.Errorf("empty response from %s", path)
	}
	if out != nil && len(resp.Metadata) > 0 {
		if err := json.Unmarshal(resp.Metadata, out); err != nil {
			return fmt.Errorf("decode %s response: %w", path, err)
		}
	}
	return nil
}

// ---- OS update status ----

// SystemUpdateStatusInput queries the OS update status.
type SystemUpdateStatusInput struct{}

// SystemUpdateStatusOutput is the appliance update state.
type SystemUpdateStatusOutput struct {
	// OSVersion is the currently installed OS version.
	OSVersion string `json:"os_version,omitempty" jsonschema:"the currently installed OS version"`
	// OSVersionNext is the pending update version, if any.
	OSVersionNext string `json:"os_version_next,omitempty" jsonschema:"the pending update version, if any"`
	// UpdateAvailable is true when an update is pending.
	UpdateAvailable bool `json:"update_available" jsonschema:"whether an update is available"`
	// Raw carries the full appliance response for forward compatibility.
	Raw map[string]any `json:"raw,omitempty" jsonschema:"the full raw appliance status (fields may vary across versions)"`
}

func (s *Server) systemUpdateStatus(ctx context.Context, req *mcp.CallToolRequest, in SystemUpdateStatusInput) (*mcp.CallToolResult, SystemUpdateStatusOutput, error) {
	if _, err := s.client.AdminClient(); err != nil {
		return toolError[SystemUpdateStatusOutput]("system_update_status", err)
	}
	var raw map[string]any
	if err := s.rawOS(ctx, "GET", "system/update", nil, &raw); err != nil {
		return toolError[SystemUpdateStatusOutput]("system_update_status", err)
	}
	out := SystemUpdateStatusOutput{Raw: raw}
	if v, ok := raw["os_version"].(string); ok {
		out.OSVersion = v
	}
	if v, ok := raw["os_version_next"].(string); ok {
		out.OSVersionNext = v
		out.UpdateAvailable = v != ""
	}
	return result(out)
}

// SystemUpdateCheckInput triggers an update check.
type SystemUpdateCheckInput struct{}

func (s *Server) systemUpdateCheck(ctx context.Context, req *mcp.CallToolRequest, in SystemUpdateCheckInput) (*mcp.CallToolResult, map[string]any, error) {
	if _, err := s.client.AdminClient(); err != nil {
		return toolError[map[string]any]("system_update_check", err)
	}
	var raw map[string]any
	if err := s.rawOS(ctx, "POST", "system/update/:check", nil, &raw); err != nil {
		return toolError[map[string]any]("system_update_check", err)
	}
	return result(raw)
}

// SystemUpdateApplyInput triggers an OS update.
type SystemUpdateApplyInput struct {
	// Version is the target version; empty applies the pending update.
	Version string `json:"version,omitempty" jsonschema:"target version to apply (defaults to the pending update)"`
}

func (s *Server) systemUpdateApply(ctx context.Context, req *mcp.CallToolRequest, in SystemUpdateApplyInput) (*mcp.CallToolResult, map[string]any, error) {
	if _, err := s.client.AdminClient(); err != nil {
		return toolError[map[string]any]("system_update_apply", err)
	}
	var raw map[string]any
	if err := s.rawOS(ctx, "POST", "system/update", map[string]any{"version": in.Version}, &raw); err != nil {
		return toolError[map[string]any]("system_update_apply", err)
	}
	return result(raw)
}

// ---- applications ----

// SystemAppListInput lists installed applications.
type SystemAppListInput struct{}

func (s *Server) systemAppList(ctx context.Context, req *mcp.CallToolRequest, in SystemAppListInput) (*mcp.CallToolResult, []string, error) {
	if _, err := s.client.AdminClient(); err != nil {
		return toolError[[]string]("system_app_list", err)
	}
	var apps []string
	if err := s.rawOS(ctx, "GET", "applications", nil, &apps); err != nil {
		return toolError[[]string]("system_app_list", err)
	}
	return result(apps)
}

// SystemAppActionInput runs an application lifecycle action.
type SystemAppActionInput struct {
	Name   string `json:"name" jsonschema:"the application name"`
	Action string `json:"action" jsonschema:"the action: restart, backup, restore, switch-version, factory-reset, remove"`
	// Version is the target version for switch-version.
	Version string `json:"version,omitempty" jsonschema:"target version for switch-version"`
}

func (s *Server) systemAppAction(ctx context.Context, req *mcp.CallToolRequest, in SystemAppActionInput) (*mcp.CallToolResult, map[string]any, error) {
	if _, err := s.client.AdminClient(); err != nil {
		return toolError[map[string]any]("system_app_action", err)
	}
	if in.Name == "" || in.Action == "" {
		return toolError[map[string]any]("system_app_action", errRequired("name and action"))
	}
	allowed := map[string]bool{"restart": true, "backup": true, "restore": true, "switch-version": true, "factory-reset": true, "remove": true}
	if !allowed[in.Action] {
		return toolError[map[string]any]("system_app_action", errRequired("action (restart|backup|restore|switch-version|factory-reset|remove)"))
	}
	body := map[string]any{}
	if in.Version != "" {
		body["version"] = in.Version
	}
	var raw map[string]any
	if err := s.rawOS(ctx, "POST", "applications/"+in.Name+"/:"+in.Action, body, &raw); err != nil {
		return toolError[map[string]any]("system_app_action", err)
	}
	return result(raw)
}

// ---- security / recovery keys ----

// SystemSecurityInput reads security state.
type SystemSecurityInput struct{}

// SystemSecurityOutput is the appliance security state.
type SystemSecurityOutput struct {
	// Trusted reports whether the system state is trusted (Secure Boot/TPM).
	Trusted bool `json:"trusted" jsonschema:"whether the system is in a trusted state"`
	// Raw carries the full response. Recovery keys are present only with the
	// admin credential and are never logged; they are returned as-is to the
	// agent because the tool result is the only delivery channel.
	Raw map[string]any `json:"raw,omitempty" jsonschema:"the full raw security state (may include recovery keys)"`
}

func (s *Server) systemSecurity(ctx context.Context, req *mcp.CallToolRequest, in SystemSecurityInput) (*mcp.CallToolResult, SystemSecurityOutput, error) {
	if _, err := s.client.AdminClient(); err != nil {
		return toolError[SystemSecurityOutput]("system_security", err)
	}
	var raw map[string]any
	if err := s.rawOS(ctx, "GET", "system/security", nil, &raw); err != nil {
		return toolError[SystemSecurityOutput]("system_security", err)
	}
	out := SystemSecurityOutput{Raw: raw}
	if v, ok := raw["system_state_is_trusted"].(bool); ok {
		out.Trusted = v
	}
	return result(out)
}

// SystemRecoveryKeysInput retrieves encryption recovery keys (admin only).
type SystemRecoveryKeysInput struct{}

// SystemRecoveryKeysOutput carries the recovery keys. Key material is never
// logged; it is only present in the tool result.
type SystemRecoveryKeysOutput struct {
	// PoolKeys maps pool names to their recovery keys.
	PoolKeys map[string]string `json:"pool_keys,omitempty" jsonschema:"recovery keys per storage pool"`
	// Raw carries the full response.
	Raw map[string]any `json:"raw,omitempty" jsonschema:"the full raw security response"`
}

func (s *Server) systemRecoveryKeys(ctx context.Context, req *mcp.CallToolRequest, in SystemRecoveryKeysInput) (*mcp.CallToolResult, SystemRecoveryKeysOutput, error) {
	if _, err := s.client.AdminClient(); err != nil {
		return toolError[SystemRecoveryKeysOutput]("system_recovery_keys", err)
	}
	var raw map[string]any
	if err := s.rawOS(ctx, "GET", "system/security", nil, &raw); err != nil {
		return toolError[SystemRecoveryKeysOutput]("system_recovery_keys", err)
	}
	out := SystemRecoveryKeysOutput{Raw: raw}
	if state, ok := raw["state"].(map[string]any); ok {
		if pk, ok := state["pool_recovery_keys"].(map[string]any); ok {
			keys := map[string]string{}
			for k, v := range pk {
				if s, ok := v.(string); ok {
					keys[k] = s
				}
			}
			out.PoolKeys = keys
		}
	}
	return result(out)
}

// ---- services ----

// SystemServiceListInput lists appliance services.
type SystemServiceListInput struct{}

func (s *Server) systemServiceList(ctx context.Context, req *mcp.CallToolRequest, in SystemServiceListInput) (*mcp.CallToolResult, []string, error) {
	if _, err := s.client.AdminClient(); err != nil {
		return toolError[[]string]("system_service_list", err)
	}
	var services []string
	if err := s.rawOS(ctx, "GET", "services", nil, &services); err != nil {
		return toolError[[]string]("system_service_list", err)
	}
	return result(services)
}

// ---- registration ----

func (s *Server) registerSystemTools() {
	addTool(s, "system_update_status", "Query the IncusOS OS update status (admin credential required).", s.systemUpdateStatus)
	addTool(s, "system_update_check", "Trigger an OS update check (admin credential required).", s.systemUpdateCheck)
	addTool(s, "system_update_apply", "Apply the pending OS update (admin credential required).", s.systemUpdateApply)
	addTool(s, "system_app_list", "List installed IncusOS applications (admin credential required).", s.systemAppList)
	addTool(s, "system_app_action", "Run an application lifecycle action: restart, backup, restore, switch-version, factory-reset, remove (admin credential required).", s.systemAppAction)
	addTool(s, "system_security", "Read the appliance security state incl. trusted status (admin credential required).", s.systemSecurity)
	addTool(s, "system_recovery_keys", "Retrieve encryption recovery keys (admin credential only; key material is never logged).", s.systemRecoveryKeys)
	addTool(s, "system_service_list", "List supported appliance services (admin credential required).", s.systemServiceList)
}
