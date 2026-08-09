package server

import (
	"context"

	"github.com/lxc/incus/v7/shared/api"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ---- profiles ----

// ProfileListInput lists profiles.
type ProfileListInput struct {
	Project string `json:"project,omitempty" jsonschema:"project (defaults to configured default)"`
}

func (s *Server) profileList(ctx context.Context, req *mcp.CallToolRequest, in ProfileListInput) (*mcp.CallToolResult, ListOutput[api.Profile], error) {
	profiles, err := s.projectServer(in.Project).GetProfiles()
	if err != nil {
		return toolError[ListOutput[api.Profile]]("profile_list", err)
	}
	return result(ListOutput[api.Profile]{Items: profiles})
}

// ProfileGetInput fetches a profile.
type ProfileGetInput struct {
	Name    string `json:"name" jsonschema:"the profile name"`
	Project string `json:"project,omitempty" jsonschema:"project (defaults to configured default)"`
}

func (s *Server) profileGet(ctx context.Context, req *mcp.CallToolRequest, in ProfileGetInput) (*mcp.CallToolResult, *api.Profile, error) {
	if in.Name == "" {
		return toolError[*api.Profile]("profile_get", errRequired("name"))
	}
	profile, _, err := s.projectServer(in.Project).GetProfile(in.Name)
	if err != nil {
		return toolError[*api.Profile]("profile_get", err)
	}
	return result(profile)
}

// ProfileCreateInput creates a profile.
type ProfileCreateInput struct {
	Name        string                       `json:"name" jsonschema:"the profile name"`
	Project     string                       `json:"project,omitempty" jsonschema:"project (defaults to configured default)"`
	Description string                       `json:"description,omitempty" jsonschema:"the profile description"`
	Config      map[string]string            `json:"config,omitempty" jsonschema:"profile configuration"`
	Devices     map[string]map[string]string `json:"devices,omitempty" jsonschema:"profile devices"`
}

func (s *Server) profileCreate(ctx context.Context, req *mcp.CallToolRequest, in ProfileCreateInput) (*mcp.CallToolResult, string, error) {
	if in.Name == "" {
		return toolError[string]("profile_create", errRequired("name"))
	}
	err := s.projectServer(in.Project).CreateProfile(api.ProfilesPost{
		Name: in.Name,
		ProfilePut: api.ProfilePut{
			Description: in.Description,
			Config:      in.Config,
			Devices:     in.Devices,
		},
	})
	if err != nil {
		return toolError[string]("profile_create", err)
	}
	return result("profile created: " + in.Name)
}

// ProfileUpdateInput updates a profile.
type ProfileUpdateInput struct {
	Name        string                       `json:"name" jsonschema:"the profile name"`
	Project     string                       `json:"project,omitempty" jsonschema:"project (defaults to configured default)"`
	Description string                       `json:"description,omitempty" jsonschema:"the profile description"`
	Config      map[string]string            `json:"config,omitempty" jsonschema:"profile configuration"`
	Devices     map[string]map[string]string `json:"devices,omitempty" jsonschema:"profile devices"`
}

func (s *Server) profileUpdate(ctx context.Context, req *mcp.CallToolRequest, in ProfileUpdateInput) (*mcp.CallToolResult, string, error) {
	if in.Name == "" {
		return toolError[string]("profile_update", errRequired("name"))
	}
	server := s.projectServer(in.Project)
	_, etag, err := server.GetProfile(in.Name)
	if err != nil {
		return toolError[string]("profile_update", err)
	}
	if err := server.UpdateProfile(in.Name, api.ProfilePut{
		Description: in.Description,
		Config:      in.Config,
		Devices:     in.Devices,
	}, etag); err != nil {
		return toolError[string]("profile_update", err)
	}
	return result("profile updated: " + in.Name)
}

// ProfileRenameInput renames a profile.
type ProfileRenameInput struct {
	Name    string `json:"name" jsonschema:"the current profile name"`
	NewName string `json:"new_name" jsonschema:"the new profile name"`
	Project string `json:"project,omitempty" jsonschema:"project (defaults to configured default)"`
}

func (s *Server) profileRename(ctx context.Context, req *mcp.CallToolRequest, in ProfileRenameInput) (*mcp.CallToolResult, string, error) {
	if in.Name == "" || in.NewName == "" {
		return toolError[string]("profile_rename", errRequired("name and new_name"))
	}
	if err := s.projectServer(in.Project).RenameProfile(in.Name, api.ProfilePost{Name: in.NewName}); err != nil {
		return toolError[string]("profile_rename", err)
	}
	return result("profile renamed: " + in.Name + " -> " + in.NewName)
}

// ProfileCopyInput copies a profile.
type ProfileCopyInput struct {
	Name    string `json:"name" jsonschema:"the source profile name"`
	NewName string `json:"new_name" jsonschema:"the new profile name"`
	Project string `json:"project,omitempty" jsonschema:"project (defaults to configured default)"`
}

func (s *Server) profileCopy(ctx context.Context, req *mcp.CallToolRequest, in ProfileCopyInput) (*mcp.CallToolResult, string, error) {
	if in.Name == "" || in.NewName == "" {
		return toolError[string]("profile_copy", errRequired("name and new_name"))
	}
	// Copy is a rename with the source name preserved: the Incus API copies
	// a profile by POSTing to /profiles with the source as a name override.
	server := s.projectServer(in.Project)
	src, _, err := server.GetProfile(in.Name)
	if err != nil {
		return toolError[string]("profile_copy", err)
	}
	err = server.CreateProfile(api.ProfilesPost{
		Name: in.NewName,
		ProfilePut: api.ProfilePut{
			Description: src.Description,
			Config:      src.Config,
			Devices:     src.Devices,
		},
	})
	if err != nil {
		return toolError[string]("profile_copy", err)
	}
	return result("profile copied: " + in.Name + " -> " + in.NewName)
}

// ProfileDeleteInput deletes a profile.
type ProfileDeleteInput struct {
	Name    string `json:"name" jsonschema:"the profile name"`
	Project string `json:"project,omitempty" jsonschema:"project (defaults to configured default)"`
}

func (s *Server) profileDelete(ctx context.Context, req *mcp.CallToolRequest, in ProfileDeleteInput) (*mcp.CallToolResult, string, error) {
	if in.Name == "" {
		return toolError[string]("profile_delete", errRequired("name"))
	}
	if err := s.projectServer(in.Project).DeleteProfile(in.Name); err != nil {
		return toolError[string]("profile_delete", err)
	}
	return result("profile deleted: " + in.Name)
}

// ---- projects ----

// ProjectListInput lists projects.
type ProjectListInput struct{}

func (s *Server) projectList(ctx context.Context, req *mcp.CallToolRequest, in ProjectListInput) (*mcp.CallToolResult, ListOutput[api.Project], error) {
	projects, err := s.client.Server.GetProjects()
	if err != nil {
		return toolError[ListOutput[api.Project]]("project_list", err)
	}
	return result(ListOutput[api.Project]{Items: projects})
}

// ProjectGetInput fetches a project.
type ProjectGetInput struct {
	Name string `json:"name" jsonschema:"the project name"`
}

func (s *Server) projectGet(ctx context.Context, req *mcp.CallToolRequest, in ProjectGetInput) (*mcp.CallToolResult, *api.Project, error) {
	if in.Name == "" {
		return toolError[*api.Project]("project_get", errRequired("name"))
	}
	project, _, err := s.client.Server.GetProject(in.Name)
	if err != nil {
		return toolError[*api.Project]("project_get", err)
	}
	return result(project)
}

// ProjectCreateInput creates a project.
type ProjectCreateInput struct {
	Name        string            `json:"name" jsonschema:"the project name"`
	Description string            `json:"description,omitempty" jsonschema:"the project description"`
	Config      map[string]string `json:"config,omitempty" jsonschema:"project configuration (features, restrictions)"`
}

func (s *Server) projectCreate(ctx context.Context, req *mcp.CallToolRequest, in ProjectCreateInput) (*mcp.CallToolResult, string, error) {
	if in.Name == "" {
		return toolError[string]("project_create", errRequired("name"))
	}
	err := s.client.Server.CreateProject(api.ProjectsPost{
		Name: in.Name,
		ProjectPut: api.ProjectPut{
			Description: in.Description,
			Config:      in.Config,
		},
	})
	if err != nil {
		return toolError[string]("project_create", err)
	}
	return result("project created: " + in.Name)
}

// ProjectUpdateInput updates a project.
type ProjectUpdateInput struct {
	Name        string            `json:"name" jsonschema:"the project name"`
	Description string            `json:"description,omitempty" jsonschema:"the project description"`
	Config      map[string]string `json:"config,omitempty" jsonschema:"project configuration"`
}

func (s *Server) projectUpdate(ctx context.Context, req *mcp.CallToolRequest, in ProjectUpdateInput) (*mcp.CallToolResult, string, error) {
	if in.Name == "" {
		return toolError[string]("project_update", errRequired("name"))
	}
	_, etag, err := s.client.Server.GetProject(in.Name)
	if err != nil {
		return toolError[string]("project_update", err)
	}
	if err := s.client.Server.UpdateProject(in.Name, api.ProjectPut{
		Description: in.Description,
		Config:      in.Config,
	}, etag); err != nil {
		return toolError[string]("project_update", err)
	}
	return result("project updated: " + in.Name)
}

// ProjectDeleteInput deletes a project.
type ProjectDeleteInput struct {
	Name  string `json:"name" jsonschema:"the project name"`
	Force bool   `json:"force,omitempty" jsonschema:"force deletion even with instances remaining"`
}

func (s *Server) projectDelete(ctx context.Context, req *mcp.CallToolRequest, in ProjectDeleteInput) (*mcp.CallToolResult, string, error) {
	if in.Name == "" {
		return toolError[string]("project_delete", errRequired("name"))
	}
	var err error
	if in.Force {
		err = s.client.Server.DeleteProjectForce(in.Name)
	} else {
		err = s.client.Server.DeleteProject(in.Name)
	}
	if err != nil {
		return toolError[string]("project_delete", err)
	}
	return result("project deleted: " + in.Name)
}

// ProjectRenameInput renames a project.
type ProjectRenameInput struct {
	Name    string `json:"name" jsonschema:"the current project name"`
	NewName string `json:"new_name" jsonschema:"the new project name"`

	WaitTimeoutSeconds int `json:"wait_timeout_seconds,omitempty" jsonschema:"wait timeout override in seconds"`
}

func (s *Server) projectRename(ctx context.Context, req *mcp.CallToolRequest, in ProjectRenameInput) (*mcp.CallToolResult, *api.Operation, error) {
	if in.Name == "" || in.NewName == "" {
		return toolError[*api.Operation]("project_rename", errRequired("name and new_name"))
	}
	op, err := s.client.Server.RenameProject(in.Name, api.ProjectPost{Name: in.NewName})
	if err != nil {
		return toolError[*api.Operation]("project_rename", err)
	}
	return s.waitResult("project_rename", op, in.WaitTimeoutSeconds)
}

// ProjectStateInput fetches a project's resource state.
type ProjectStateInput struct {
	Name string `json:"name" jsonschema:"the project name"`
}

func (s *Server) projectState(ctx context.Context, req *mcp.CallToolRequest, in ProjectStateInput) (*mcp.CallToolResult, *api.ProjectState, error) {
	if in.Name == "" {
		return toolError[*api.ProjectState]("project_state", errRequired("name"))
	}
	state, err := s.client.Server.GetProjectState(in.Name)
	if err != nil {
		return toolError[*api.ProjectState]("project_state", err)
	}
	return result(state)
}

// ---- registration ----

func (s *Server) registerConfigTools() {
	addTool(s, "profile_list", "List profiles in the project.", s.profileList)
	addTool(s, "profile_get", "Fetch a profile's config and devices.", s.profileGet)
	addTool(s, "profile_create", "Create a profile.", s.profileCreate)
	addTool(s, "profile_update", "Update a profile's config/devices.", s.profileUpdate)
	addTool(s, "profile_rename", "Rename a profile.", s.profileRename)
	addTool(s, "profile_copy", "Copy a profile to a new name.", s.profileCopy)
	addTool(s, "profile_delete", "Delete a profile (fails while in use).", s.profileDelete)
	addTool(s, "project_list", "List projects.", s.projectList)
	addTool(s, "project_get", "Fetch a project's config.", s.projectGet)
	addTool(s, "project_create", "Create a project.", s.projectCreate)
	addTool(s, "project_update", "Update a project's config.", s.projectUpdate)
	addTool(s, "project_delete", "Delete a project (refused while instances remain unless force).", s.projectDelete)
	addTool(s, "project_rename", "Rename a project.", s.projectRename)
	addTool(s, "project_state", "Fetch a project's resource state.", s.projectState)
}
