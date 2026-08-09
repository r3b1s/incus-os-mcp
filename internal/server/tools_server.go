package server

import (
	"context"
	"strings"
	"time"

	"github.com/lxc/incus/v7/shared/api"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ---- server_info (probe tool) ----

// ServerInfoInput is the input for server_info.
type ServerInfoInput struct{}

// ServerInfoOutput reports target connectivity and identity.
type ServerInfoOutput struct {
	ServerName    string   `json:"server_name" jsonschema:"the Incus server name"`
	APIVersion    string   `json:"api_version" jsonschema:"the Incus API version"`
	APIExtensions []string `json:"api_extensions" jsonschema:"the Incus API extensions the target supports"`
	Auth          string   `json:"auth" jsonschema:"the authentication method in use"`
	Public        bool     `json:"public" jsonschema:"whether the target allows unauthenticated access"`
	Project       string   `json:"project" jsonschema:"the effective default project"`
	Clustered     bool     `json:"clustered" jsonschema:"whether the target is part of a cluster"`
}

func (s *Server) serverInfo(ctx context.Context, req *mcp.CallToolRequest, in ServerInfoInput) (*mcp.CallToolResult, ServerInfoOutput, error) {
	srv, _, err := s.client.Server.GetServer()
	if err != nil {
		return toolError[ServerInfoOutput]("server_info", err)
	}

	out := ServerInfoOutput{
		ServerName:    srv.Environment.ServerName,
		APIVersion:    srv.APIVersion,
		APIExtensions: srv.APIExtensions,
		Auth:          srv.Auth,
		Public:        srv.Public,
		Project:       s.client.Project(""),
		Clustered:     s.client.Server.IsClustered(),
	}
	return result(out)
}

func (s *Server) registerServerTools() {
	addTool(s, "server_info", "Report target server identity, API version/extensions, authentication method, and the effective default project.", s.serverInfo)
	addTool(s, "server_list_operations", "List asynchronous operations on the target (optionally filtered by project).", s.serverListOperations)
	addTool(s, "server_get_operation", "Fetch a single operation by ID or URL.", s.serverGetOperation)
	addTool(s, "server_wait_operation", "Wait for an operation (by ID or URL) to complete, with a timeout. Returns final state, or the running state on expiry.", s.serverWaitOperation)
	addTool(s, "server_list_certificates", "List trusted client certificates. Admin-only surface.", s.serverListCertificates)
	addTool(s, "server_add_certificate", "Add a client certificate to the trust store. Admin-only surface.", s.serverAddCertificate)
	addTool(s, "server_delete_certificate", "Remove a client certificate from the trust store. Admin-only surface.", s.serverDeleteCertificate)
	addTool(s, "cluster_member_list", "List cluster members.", s.clusterMemberList)
	addTool(s, "cluster_member_get", "Fetch a cluster member's metadata.", s.clusterMemberGet)
	addTool(s, "cluster_member_state", "Fetch a cluster member's evacuation state.", s.clusterMemberState)
	addTool(s, "cluster_member_state_change", "Evacuate or restore a cluster member.", s.clusterMemberStateChange)
}

// ---- operations ----

// OperationRef identifies an operation by ID or URL.
type OperationRef struct {
	// OperationID is the operation ID (from a prior mutation result).
	OperationID string `json:"operation_id,omitempty" jsonschema:"the operation ID to wait on"`
	// OperationURL is the full operation URL.
	OperationURL string `json:"operation_url,omitempty" jsonschema:"the full operation URL to wait on"`
}

// ServerListOperationsInput filters operation listing.
type ServerListOperationsInput struct {
	Project string `json:"project,omitempty" jsonschema:"project to list operations for (defaults to configured default)"`
}

func (s *Server) serverListOperations(ctx context.Context, req *mcp.CallToolRequest, in ServerListOperationsInput) (*mcp.CallToolResult, ListOutput[api.Operation], error) {
	ops, err := s.projectServer(in.Project).GetOperations()
	if err != nil {
		return toolError[ListOutput[api.Operation]]("server_list_operations", err)
	}
	return result(ListOutput[api.Operation]{Items: ops})
}

// ServerGetOperationInput fetches one operation.
type ServerGetOperationInput struct {
	OperationID string `json:"operation_id" jsonschema:"the operation ID"`
	Project     string `json:"project,omitempty" jsonschema:"project (defaults to configured default)"`
}

func (s *Server) serverGetOperation(ctx context.Context, req *mcp.CallToolRequest, in ServerGetOperationInput) (*mcp.CallToolResult, *api.Operation, error) {
	if in.OperationID == "" {
		return toolError[*api.Operation]("server_get_operation", errRequired("operation_id"))
	}
	op, _, err := s.projectServer(in.Project).GetOperation(in.OperationID)
	if err != nil {
		return toolError[*api.Operation]("server_get_operation", err)
	}
	return result(op)
}

// ServerWaitOperationInput waits on an operation.
type ServerWaitOperationInput struct {
	OperationRef
	Project string `json:"project,omitempty" jsonschema:"project (defaults to configured default)"`
	// TimeoutSeconds overrides the configured wait timeout.
	TimeoutSeconds int `json:"timeout_seconds,omitempty" jsonschema:"wait timeout override in seconds (defaults to configured wait_timeout_seconds)"`
}

func (s *Server) serverWaitOperation(ctx context.Context, req *mcp.CallToolRequest, in ServerWaitOperationInput) (*mcp.CallToolResult, *api.Operation, error) {
	if in.OperationID == "" && in.OperationURL == "" {
		return toolError[*api.Operation]("server_wait_operation", errRequired("operation_id or operation_url"))
	}

	// Extract the UUID from an ID or a full URL.
	uuid := in.OperationID
	if in.OperationURL != "" {
		parts := strings.Split(strings.TrimSuffix(in.OperationURL, "/"), "/")
		uuid = parts[len(parts)-1]
	}

	timeout := s.client.WaitTimeout()
	if in.TimeoutSeconds > 0 {
		timeout = time.Duration(in.TimeoutSeconds) * time.Second
	}

	// GetOperation confirms the operation exists (and returns its state for
	// the "already done" fast path), then GetOperationWait blocks server-side
	// with the requested timeout. A still-running operation comes back with
	// status "running" (not an error).
	server := s.projectServer(in.Project)
	if _, _, err := server.GetOperation(uuid); err != nil {
		return toolError[*api.Operation]("server_wait_operation", err)
	}

	final, _, err := server.GetOperationWait(uuid, int(timeout.Seconds()))
	if err != nil {
		return toolError[*api.Operation]("server_wait_operation", err)
	}
	return result(final)
}

// ---- certificates (admin-only) ----

// CertificateInput carries the certificate fields for add.
type CertificateInput struct {
	// Name is the certificate name.
	Name string `json:"name" jsonschema:"the certificate name"`
	// Type is the certificate type (client or server).
	Type string `json:"type,omitempty" jsonschema:"the certificate type, default client"`
	// PEM is the PEM-encoded certificate.
	PEM string `json:"pem" jsonschema:"the PEM-encoded certificate to trust"`
	// Projects restricts the certificate to a project (scoped).
	Projects []string `json:"projects,omitempty" jsonschema:"optional project restriction"`
	// Restricted marks the certificate as restricted (scoped auth).
	Restricted bool `json:"restricted,omitempty" jsonschema:"whether the certificate is restricted to the given projects"`
}

func (s *Server) serverListCertificates(ctx context.Context, req *mcp.CallToolRequest, in struct{}) (*mcp.CallToolResult, ListOutput[api.Certificate], error) {
	admin, err := s.client.AdminClient()
	if err != nil {
		return toolError[ListOutput[api.Certificate]]("server_list_certificates", err)
	}
	certs, err := admin.GetCertificates()
	if err != nil {
		return toolError[ListOutput[api.Certificate]]("server_list_certificates", err)
	}
	return result(ListOutput[api.Certificate]{Items: certs})
}

func (s *Server) serverAddCertificate(ctx context.Context, req *mcp.CallToolRequest, in CertificateInput) (*mcp.CallToolResult, string, error) {
	admin, err := s.client.AdminClient()
	if err != nil {
		return toolError[string]("server_add_certificate", err)
	}
	if in.Name == "" || in.PEM == "" {
		return toolError[string]("server_add_certificate", errRequired("name and pem"))
	}
	ctype := in.Type
	if ctype == "" {
		ctype = "client"
	}
	post := api.CertificatesPost{
		CertificatePut: api.CertificatePut{
			Name:        in.Name,
			Type:        ctype,
			Restricted:  in.Restricted,
			Projects:    in.Projects,
			Certificate: in.PEM,
		},
	}
	if err := admin.CreateCertificate(post); err != nil {
		return toolError[string]("server_add_certificate", err)
	}
	return result("certificate added: " + in.Name)
}

func (s *Server) serverDeleteCertificate(ctx context.Context, req *mcp.CallToolRequest, in struct {
	Fingerprint string `json:"fingerprint" jsonschema:"the certificate fingerprint to delete"`
}) (*mcp.CallToolResult, string, error) {
	admin, err := s.client.AdminClient()
	if err != nil {
		return toolError[string]("server_delete_certificate", err)
	}
	if in.Fingerprint == "" {
		return toolError[string]("server_delete_certificate", errRequired("fingerprint"))
	}
	if err := admin.DeleteCertificate(in.Fingerprint); err != nil {
		return toolError[string]("server_delete_certificate", err)
	}
	return result("certificate deleted: " + in.Fingerprint)
}
