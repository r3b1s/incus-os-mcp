package server

import (
	"context"

	"github.com/lxc/incus/v7/shared/api"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// NetworkZoneRecordListInput lists records in a DNS zone.
type NetworkZoneRecordListInput struct {
	Zone    string `json:"zone" jsonschema:"the zone name"`
	Project string `json:"project,omitempty" jsonschema:"project (defaults to configured default)"`
}

func (s *Server) networkZoneRecordList(ctx context.Context, req *mcp.CallToolRequest, in NetworkZoneRecordListInput) (*mcp.CallToolResult, ListOutput[api.NetworkZoneRecord], error) {
	if in.Zone == "" {
		return toolError[ListOutput[api.NetworkZoneRecord]]("network_zone_record_list", errRequired("zone"))
	}
	records, err := s.projectServer(in.Project).GetNetworkZoneRecords(in.Zone)
	if err != nil {
		return toolError[ListOutput[api.NetworkZoneRecord]]("network_zone_record_list", err)
	}
	return result(ListOutput[api.NetworkZoneRecord]{Items: records})
}

// NetworkZoneRecordGetInput fetches one DNS record.
type NetworkZoneRecordGetInput struct {
	Zone    string `json:"zone" jsonschema:"the zone name"`
	Project string `json:"project,omitempty" jsonschema:"project (defaults to configured default)"`
	Name    string `json:"name" jsonschema:"the record name"`
}

func (s *Server) networkZoneRecordGet(ctx context.Context, req *mcp.CallToolRequest, in NetworkZoneRecordGetInput) (*mcp.CallToolResult, *api.NetworkZoneRecord, error) {
	if in.Zone == "" || in.Name == "" {
		return toolError[*api.NetworkZoneRecord]("network_zone_record_get", errRequired("zone and name"))
	}
	record, _, err := s.projectServer(in.Project).GetNetworkZoneRecord(in.Zone, in.Name)
	if err != nil {
		return toolError[*api.NetworkZoneRecord]("network_zone_record_get", err)
	}
	return result(record)
}

// NetworkZoneRecordUpdateInput updates a DNS record.
type NetworkZoneRecordUpdateInput struct {
	Zone        string                       `json:"zone" jsonschema:"the zone name"`
	Project     string                       `json:"project,omitempty" jsonschema:"project (defaults to configured default)"`
	Name        string                       `json:"name" jsonschema:"the record name"`
	Description string                       `json:"description,omitempty" jsonschema:"record description"`
	Entries     []api.NetworkZoneRecordEntry `json:"entries,omitempty" jsonschema:"record entries (type/data pairs)"`
}

func (s *Server) networkZoneRecordUpdate(ctx context.Context, req *mcp.CallToolRequest, in NetworkZoneRecordUpdateInput) (*mcp.CallToolResult, string, error) {
	if in.Zone == "" || in.Name == "" {
		return toolError[string]("network_zone_record_update", errRequired("zone and name"))
	}
	server := s.projectServer(in.Project)
	_, etag, err := server.GetNetworkZoneRecord(in.Zone, in.Name)
	if err != nil {
		return toolError[string]("network_zone_record_update", err)
	}
	if err := server.UpdateNetworkZoneRecord(in.Zone, in.Name, api.NetworkZoneRecordPut{Description: in.Description, Entries: in.Entries}, etag); err != nil {
		return toolError[string]("network_zone_record_update", err)
	}
	return result("record updated: " + in.Name)
}
