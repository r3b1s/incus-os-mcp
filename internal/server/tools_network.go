package server

import (
	"context"

	"github.com/lxc/incus/v7/shared/api"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ---- networks ----

// NetworkListInput lists networks.
type NetworkListInput struct {
	Project string `json:"project,omitempty" jsonschema:"project (defaults to configured default)"`
}

func (s *Server) networkList(ctx context.Context, req *mcp.CallToolRequest, in NetworkListInput) (*mcp.CallToolResult, ListOutput[api.Network], error) {
	networks, err := s.projectServer(in.Project).GetNetworks()
	if err != nil {
		return toolError[ListOutput[api.Network]]("network_list", err)
	}
	return result(ListOutput[api.Network]{Items: networks})
}

// NetworkGetInput fetches a network.
type NetworkGetInput struct {
	Name    string `json:"name" jsonschema:"the network name"`
	Project string `json:"project,omitempty" jsonschema:"project (defaults to configured default)"`
}

func (s *Server) networkGet(ctx context.Context, req *mcp.CallToolRequest, in NetworkGetInput) (*mcp.CallToolResult, *api.Network, error) {
	if in.Name == "" {
		return toolError[*api.Network]("network_get", errRequired("name"))
	}
	net, _, err := s.projectServer(in.Project).GetNetwork(in.Name)
	if err != nil {
		return toolError[*api.Network]("network_get", err)
	}
	return result(net)
}

// NetworkCreateInput creates a network.
type NetworkCreateInput struct {
	Name    string            `json:"name" jsonschema:"the network name"`
	Project string            `json:"project,omitempty" jsonschema:"project (defaults to configured default)"`
	Type    string            `json:"type,omitempty" jsonschema:"network type: bridge (default), ovn, physical, macvlan, ..."`
	Config  map[string]string `json:"config,omitempty" jsonschema:"network configuration"`
}

func (s *Server) networkCreate(ctx context.Context, req *mcp.CallToolRequest, in NetworkCreateInput) (*mcp.CallToolResult, string, error) {
	if in.Name == "" {
		return toolError[string]("network_create", errRequired("name"))
	}
	ntype := in.Type
	if ntype == "" {
		ntype = "bridge"
	}
	err := s.projectServer(in.Project).CreateNetwork(api.NetworksPost{
		Name:       in.Name,
		Type:       ntype,
		NetworkPut: api.NetworkPut{Config: in.Config},
	})
	if err != nil {
		return toolError[string]("network_create", err)
	}
	return result("network created: " + in.Name)
}

// NetworkUpdateInput updates a network.
type NetworkUpdateInput struct {
	Name    string            `json:"name" jsonschema:"the network name"`
	Project string            `json:"project,omitempty" jsonschema:"project (defaults to configured default)"`
	Config  map[string]string `json:"config,omitempty" jsonschema:"network configuration to apply"`
}

func (s *Server) networkUpdate(ctx context.Context, req *mcp.CallToolRequest, in NetworkUpdateInput) (*mcp.CallToolResult, string, error) {
	if in.Name == "" {
		return toolError[string]("network_update", errRequired("name"))
	}
	server := s.projectServer(in.Project)
	_, etag, err := server.GetNetwork(in.Name)
	if err != nil {
		return toolError[string]("network_update", err)
	}
	if err := server.UpdateNetwork(in.Name, api.NetworkPut{Config: in.Config}, etag); err != nil {
		return toolError[string]("network_update", err)
	}
	return result("network updated: " + in.Name)
}

// NetworkDeleteInput deletes a network.
type NetworkDeleteInput struct {
	Name    string `json:"name" jsonschema:"the network name"`
	Project string `json:"project,omitempty" jsonschema:"project (defaults to configured default)"`
}

func (s *Server) networkDelete(ctx context.Context, req *mcp.CallToolRequest, in NetworkDeleteInput) (*mcp.CallToolResult, string, error) {
	if in.Name == "" {
		return toolError[string]("network_delete", errRequired("name"))
	}
	if err := s.projectServer(in.Project).DeleteNetwork(in.Name); err != nil {
		return toolError[string]("network_delete", err)
	}
	return result("network deleted: " + in.Name)
}

// ---- ACLs ----

// NetworkACLListInput lists ACLs.
type NetworkACLListInput struct {
	Project string `json:"project,omitempty" jsonschema:"project (defaults to configured default)"`
}

func (s *Server) networkACLList(ctx context.Context, req *mcp.CallToolRequest, in NetworkACLListInput) (*mcp.CallToolResult, ListOutput[api.NetworkACL], error) {
	acls, err := s.projectServer(in.Project).GetNetworkACLs()
	if err != nil {
		return toolError[ListOutput[api.NetworkACL]]("network_acl_list", err)
	}
	return result(ListOutput[api.NetworkACL]{Items: acls})
}

// NetworkACLCreateInput creates an ACL.
type NetworkACLCreateInput struct {
	Name        string               `json:"name" jsonschema:"the ACL name"`
	Project     string               `json:"project,omitempty" jsonschema:"project (defaults to configured default)"`
	Description string               `json:"description,omitempty" jsonschema:"the ACL description"`
	Egress      []api.NetworkACLRule `json:"egress,omitempty" jsonschema:"egress rules"`
	Ingress     []api.NetworkACLRule `json:"ingress,omitempty" jsonschema:"ingress rules"`
}

func (s *Server) networkACLCreate(ctx context.Context, req *mcp.CallToolRequest, in NetworkACLCreateInput) (*mcp.CallToolResult, string, error) {
	if in.Name == "" {
		return toolError[string]("network_acl_create", errRequired("name"))
	}
	err := s.projectServer(in.Project).CreateNetworkACL(api.NetworkACLsPost{
		NetworkACLPost: api.NetworkACLPost{
			Name: in.Name,
		},
		NetworkACLPut: api.NetworkACLPut{
			Description: in.Description,
			Egress:      in.Egress,
			Ingress:     in.Ingress,
		},
	})
	if err != nil {
		return toolError[string]("network_acl_create", err)
	}
	return result("ACL created: " + in.Name)
}

// NetworkACLUpdateInput updates an ACL.
type NetworkACLUpdateInput struct {
	Name        string               `json:"name" jsonschema:"the ACL name"`
	Project     string               `json:"project,omitempty" jsonschema:"project (defaults to configured default)"`
	Description string               `json:"description,omitempty" jsonschema:"the ACL description"`
	Egress      []api.NetworkACLRule `json:"egress,omitempty" jsonschema:"egress rules"`
	Ingress     []api.NetworkACLRule `json:"ingress,omitempty" jsonschema:"ingress rules"`
}

func (s *Server) networkACLUpdate(ctx context.Context, req *mcp.CallToolRequest, in NetworkACLUpdateInput) (*mcp.CallToolResult, string, error) {
	if in.Name == "" {
		return toolError[string]("network_acl_update", errRequired("name"))
	}
	server := s.projectServer(in.Project)
	_, etag, err := server.GetNetworkACL(in.Name)
	if err != nil {
		return toolError[string]("network_acl_update", err)
	}
	if err := server.UpdateNetworkACL(in.Name, api.NetworkACLPut{
		Description: in.Description,
		Egress:      in.Egress,
		Ingress:     in.Ingress,
	}, etag); err != nil {
		return toolError[string]("network_acl_update", err)
	}
	return result("ACL updated: " + in.Name)
}

// NetworkACLDeleteInput deletes an ACL.
type NetworkACLDeleteInput struct {
	Name    string `json:"name" jsonschema:"the ACL name"`
	Project string `json:"project,omitempty" jsonschema:"project (defaults to configured default)"`
}

func (s *Server) networkACLDelete(ctx context.Context, req *mcp.CallToolRequest, in NetworkACLDeleteInput) (*mcp.CallToolResult, string, error) {
	if in.Name == "" {
		return toolError[string]("network_acl_delete", errRequired("name"))
	}
	if err := s.projectServer(in.Project).DeleteNetworkACL(in.Name); err != nil {
		return toolError[string]("network_acl_delete", err)
	}
	return result("ACL deleted: " + in.Name)
}

// ---- zones ----

// NetworkZoneListInput lists zones.
type NetworkZoneListInput struct {
	Project string `json:"project,omitempty" jsonschema:"project (defaults to configured default)"`
}

func (s *Server) networkZoneList(ctx context.Context, req *mcp.CallToolRequest, in NetworkZoneListInput) (*mcp.CallToolResult, ListOutput[api.NetworkZone], error) {
	zones, err := s.projectServer(in.Project).GetNetworkZones()
	if err != nil {
		return toolError[ListOutput[api.NetworkZone]]("network_zone_list", err)
	}
	return result(ListOutput[api.NetworkZone]{Items: zones})
}

// NetworkZoneCreateInput creates a zone.
type NetworkZoneCreateInput struct {
	Name        string            `json:"name" jsonschema:"the zone name (a DNS name)"`
	Project     string            `json:"project,omitempty" jsonschema:"project (defaults to configured default)"`
	Description string            `json:"description,omitempty" jsonschema:"the zone description"`
	Config      map[string]string `json:"config,omitempty" jsonschema:"zone configuration"`
}

func (s *Server) networkZoneCreate(ctx context.Context, req *mcp.CallToolRequest, in NetworkZoneCreateInput) (*mcp.CallToolResult, string, error) {
	if in.Name == "" {
		return toolError[string]("network_zone_create", errRequired("name"))
	}
	err := s.projectServer(in.Project).CreateNetworkZone(api.NetworkZonesPost{
		Name: in.Name,
		NetworkZonePut: api.NetworkZonePut{
			Description: in.Description,
			Config:      in.Config,
		},
	})
	if err != nil {
		return toolError[string]("network_zone_create", err)
	}
	return result("zone created: " + in.Name)
}

// NetworkZoneUpdateInput updates a zone.
type NetworkZoneUpdateInput struct {
	Name        string            `json:"name" jsonschema:"the zone name"`
	Project     string            `json:"project,omitempty" jsonschema:"project (defaults to configured default)"`
	Description string            `json:"description,omitempty" jsonschema:"the zone description"`
	Config      map[string]string `json:"config,omitempty" jsonschema:"zone configuration"`
}

func (s *Server) networkZoneUpdate(ctx context.Context, req *mcp.CallToolRequest, in NetworkZoneUpdateInput) (*mcp.CallToolResult, string, error) {
	if in.Name == "" {
		return toolError[string]("network_zone_update", errRequired("name"))
	}
	server := s.projectServer(in.Project)
	_, etag, err := server.GetNetworkZone(in.Name)
	if err != nil {
		return toolError[string]("network_zone_update", err)
	}
	if err := server.UpdateNetworkZone(in.Name, api.NetworkZonePut{
		Description: in.Description,
		Config:      in.Config,
	}, etag); err != nil {
		return toolError[string]("network_zone_update", err)
	}
	return result("zone updated: " + in.Name)
}

// NetworkZoneDeleteInput deletes a zone.
type NetworkZoneDeleteInput struct {
	Name    string `json:"name" jsonschema:"the zone name"`
	Project string `json:"project,omitempty" jsonschema:"project (defaults to configured default)"`
}

func (s *Server) networkZoneDelete(ctx context.Context, req *mcp.CallToolRequest, in NetworkZoneDeleteInput) (*mcp.CallToolResult, string, error) {
	if in.Name == "" {
		return toolError[string]("network_zone_delete", errRequired("name"))
	}
	if err := s.projectServer(in.Project).DeleteNetworkZone(in.Name); err != nil {
		return toolError[string]("network_zone_delete", err)
	}
	return result("zone deleted: " + in.Name)
}

// NetworkZoneRecordCreateInput creates a zone record.
type NetworkZoneRecordCreateInput struct {
	Zone        string                       `json:"zone" jsonschema:"the zone name"`
	Project     string                       `json:"project,omitempty" jsonschema:"project (defaults to configured default)"`
	Name        string                       `json:"name" jsonschema:"the record name (hostname)"`
	Description string                       `json:"description,omitempty" jsonschema:"the record description"`
	Entries     []api.NetworkZoneRecordEntry `json:"entries,omitempty" jsonschema:"record entries (type/data pairs)"`
}

func (s *Server) networkZoneRecordCreate(ctx context.Context, req *mcp.CallToolRequest, in NetworkZoneRecordCreateInput) (*mcp.CallToolResult, string, error) {
	if in.Zone == "" || in.Name == "" {
		return toolError[string]("network_zone_record_create", errRequired("zone and name"))
	}
	err := s.projectServer(in.Project).CreateNetworkZoneRecord(in.Zone, api.NetworkZoneRecordsPost{
		Name: in.Name,
		NetworkZoneRecordPut: api.NetworkZoneRecordPut{
			Description: in.Description,
			Entries:     in.Entries,
		},
	})
	if err != nil {
		return toolError[string]("network_zone_record_create", err)
	}
	return result("record created: " + in.Name)
}

// NetworkZoneRecordDeleteInput deletes a zone record.
type NetworkZoneRecordDeleteInput struct {
	Zone    string `json:"zone" jsonschema:"the zone name"`
	Project string `json:"project,omitempty" jsonschema:"project (defaults to configured default)"`
	Name    string `json:"name" jsonschema:"the record name"`
}

func (s *Server) networkZoneRecordDelete(ctx context.Context, req *mcp.CallToolRequest, in NetworkZoneRecordDeleteInput) (*mcp.CallToolResult, string, error) {
	if in.Zone == "" || in.Name == "" {
		return toolError[string]("network_zone_record_delete", errRequired("zone and name"))
	}
	if err := s.projectServer(in.Project).DeleteNetworkZoneRecord(in.Zone, in.Name); err != nil {
		return toolError[string]("network_zone_record_delete", err)
	}
	return result("record deleted: " + in.Name)
}

// ---- registration ----

func (s *Server) registerNetworkTools() {
	addTool(s, "network_list", "List managed networks.", s.networkList)
	addTool(s, "network_get", "Fetch a network's config.", s.networkGet)
	addTool(s, "network_create", "Create a managed network (bridge, ovn, physical, macvlan, ...).", s.networkCreate)
	addTool(s, "network_update", "Update a network's config.", s.networkUpdate)
	addTool(s, "network_delete", "Delete a network (fails while in use).", s.networkDelete)
	addTool(s, "network_acl_list", "List network ACLs.", s.networkACLList)
	addTool(s, "network_acl_create", "Create a network ACL with rules.", s.networkACLCreate)
	addTool(s, "network_acl_update", "Update a network ACL's rules.", s.networkACLUpdate)
	addTool(s, "network_acl_delete", "Delete a network ACL.", s.networkACLDelete)
	addTool(s, "network_zone_list", "List network zones.", s.networkZoneList)
	addTool(s, "network_zone_create", "Create a network zone.", s.networkZoneCreate)
	addTool(s, "network_zone_update", "Update a network zone.", s.networkZoneUpdate)
	addTool(s, "network_zone_delete", "Delete a network zone.", s.networkZoneDelete)
	addTool(s, "network_zone_record_list", "List DNS records in a zone.", s.networkZoneRecordList)
	addTool(s, "network_zone_record_get", "Fetch a DNS record in a zone.", s.networkZoneRecordGet)
	addTool(s, "network_zone_record_create", "Create a DNS record in a zone.", s.networkZoneRecordCreate)
	addTool(s, "network_zone_record_update", "Update a DNS record in a zone.", s.networkZoneRecordUpdate)
	addTool(s, "network_zone_record_delete", "Delete a DNS record from a zone.", s.networkZoneRecordDelete)
}
