package server

import (
	"fmt"

	incusclient "github.com/lxc/incus/v7/client"
)

// ListOutput is the object-shaped result shared by collection tools. MCP hosts
// are not required to accept top-level array structured content.
type ListOutput[T any] struct {
	Items []T `json:"items" jsonschema:"collection items"`
}

// projectServer returns an Incus client scoped to an explicit project or the
// configured default. The scoped client is request-local and does not mutate
// the shared connection used by concurrent MCP calls.
func (s *Server) projectServer(project string) incusclient.InstanceServer {
	return s.client.Server.UseProject(s.client.Project(project))
}

// validateProjectScope rejects an ambiguous project override paired with an
// all-projects request. Callers must choose one scope explicitly.
func validateProjectScope(project string, allProjects bool) error {
	if project != "" && allProjects {
		return fmt.Errorf("project and all_projects cannot be used together")
	}
	return nil
}
