package server

import (
	"context"
	"io"

	"github.com/lxc/incus/v7/shared/api"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ClusterMemberListInput lists cluster members.
type ClusterMemberListInput struct{}

func (s *Server) clusterMemberList(ctx context.Context, req *mcp.CallToolRequest, in ClusterMemberListInput) (*mcp.CallToolResult, ListOutput[api.ClusterMember], error) {
	members, err := s.client.Server.GetClusterMembers()
	if err != nil {
		return toolError[ListOutput[api.ClusterMember]]("cluster_member_list", err)
	}
	return result(ListOutput[api.ClusterMember]{Items: members})
}

// ClusterMemberGetInput fetches one cluster member.
type ClusterMemberGetInput struct {
	Name string `json:"name" jsonschema:"the cluster member name"`
}

func (s *Server) clusterMemberGet(ctx context.Context, req *mcp.CallToolRequest, in ClusterMemberGetInput) (*mcp.CallToolResult, *api.ClusterMember, error) {
	if in.Name == "" {
		return toolError[*api.ClusterMember]("cluster_member_get", errRequired("name"))
	}
	member, _, err := s.client.Server.GetClusterMember(in.Name)
	if err != nil {
		return toolError[*api.ClusterMember]("cluster_member_get", err)
	}
	return result(member)
}

// ClusterMemberStateInput fetches the maintenance/evacuation state of a member.
type ClusterMemberStateInput struct {
	Name string `json:"name" jsonschema:"the cluster member name"`
}

func (s *Server) clusterMemberState(ctx context.Context, req *mcp.CallToolRequest, in ClusterMemberStateInput) (*mcp.CallToolResult, *api.ClusterMemberState, error) {
	if in.Name == "" {
		return toolError[*api.ClusterMemberState]("cluster_member_state", errRequired("name"))
	}
	state, _, err := s.client.Server.GetClusterMemberState(in.Name)
	if err != nil {
		return toolError[*api.ClusterMemberState]("cluster_member_state", err)
	}
	return result(state)
}

// ClusterMemberStateChangeInput evacuates or restores a member.
type ClusterMemberStateChangeInput struct {
	Name               string `json:"name" jsonschema:"the cluster member name"`
	Action             string `json:"action" jsonschema:"cluster action: evacuate or restore"`
	Mode               string `json:"mode,omitempty" jsonschema:"optional evacuation mode"`
	WaitTimeoutSeconds int    `json:"wait_timeout_seconds,omitempty" jsonschema:"wait timeout override in seconds"`
}

func (s *Server) clusterMemberStateChange(ctx context.Context, req *mcp.CallToolRequest, in ClusterMemberStateChangeInput) (*mcp.CallToolResult, *api.Operation, error) {
	if in.Name == "" || (in.Action != "evacuate" && in.Action != "restore") {
		return toolError[*api.Operation]("cluster_member_state_change", errRequired("name and action (evacuate|restore)"))
	}
	op, err := s.client.Server.UpdateClusterMemberState(in.Name, api.ClusterMemberStatePost{Action: in.Action, Mode: in.Mode})
	if err != nil {
		return toolError[*api.Operation]("cluster_member_state_change", err)
	}
	return s.waitResult("cluster_member_state_change", op, in.WaitTimeoutSeconds)
}

// InstanceConsoleLogInput obtains a bounded serial-console log without opening an interactive stream.
type InstanceConsoleLogInput struct {
	Name     string `json:"name" jsonschema:"the instance name"`
	Project  string `json:"project,omitempty" jsonschema:"project the instance is in (defaults to configured default)"`
	MaxBytes int64  `json:"max_bytes,omitempty" jsonschema:"maximum bytes to return (defaults to 65536)"`
}

type InstanceConsoleLogOutput struct {
	Content   string `json:"content" jsonschema:"serial console log content"`
	Truncated bool   `json:"truncated" jsonschema:"whether output was truncated at max_bytes"`
}

func (s *Server) instanceConsoleLog(ctx context.Context, req *mcp.CallToolRequest, in InstanceConsoleLogInput) (*mcp.CallToolResult, InstanceConsoleLogOutput, error) {
	if in.Name == "" {
		return toolError[InstanceConsoleLogOutput]("instance_console_log", errRequired("name"))
	}
	limit := in.MaxBytes
	if limit <= 0 {
		limit = 64 * 1024
	}
	log, err := s.projectServer(in.Project).GetInstanceConsoleLog(in.Name, nil)
	if err != nil {
		return toolError[InstanceConsoleLogOutput]("instance_console_log", err)
	}
	defer log.Close()
	data, err := io.ReadAll(io.LimitReader(log, limit+1))
	if err != nil {
		return toolError[InstanceConsoleLogOutput]("instance_console_log", err)
	}
	out := InstanceConsoleLogOutput{Content: string(data)}
	if int64(len(data)) > limit {
		out.Content = string(data[:limit])
		out.Truncated = true
	}
	return result(out)
}
