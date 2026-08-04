package server

import (
	"context"
	"os"
	"time"

	incusclient "github.com/lxc/incus/v7/client"
	"github.com/lxc/incus/v7/shared/api"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ---- listing / inspection ----

// InstanceListInput lists instances.
type InstanceListInput struct {
	Project string `json:"project,omitempty" jsonschema:"project to list instances in (defaults to configured default)"`
	// AllProjects lists across all projects (admin).
	AllProjects bool `json:"all_projects,omitempty" jsonschema:"list instances in all projects"`
	// Filter is an optional API filter expression (e.g. 'status=running').
	Filter string `json:"filter,omitempty" jsonschema:"optional API filter expression"`
}

func (s *Server) instanceList(ctx context.Context, req *mcp.CallToolRequest, in InstanceListInput) (*mcp.CallToolResult, []api.Instance, error) {
	var (
		instances []api.Instance
		err       error
	)
	if in.Filter != "" {
		// Filtering requires the api_filtering extension; use it only when a
		// filter was actually requested.
		if in.AllProjects {
			instances, err = s.client.Server.GetInstancesAllProjectsWithFilter(api.InstanceTypeAny, []string{in.Filter})
		} else {
			instances, err = s.client.Server.GetInstancesWithFilter(api.InstanceTypeAny, []string{in.Filter})
		}
	} else if in.AllProjects {
		instances, err = s.client.Server.GetInstancesAllProjects(api.InstanceTypeAny)
	} else {
		instances, err = s.client.Server.GetInstances(api.InstanceTypeAny)
	}
	if err != nil {
		return toolError[[]api.Instance]("instance_list", err)
	}
	return result(instances)
}

// InstanceGetInput fetches a single instance.
type InstanceGetInput struct {
	Name    string `json:"name" jsonschema:"the instance name"`
	Project string `json:"project,omitempty" jsonschema:"project the instance is in (defaults to configured default)"`
}

func (s *Server) instanceGet(ctx context.Context, req *mcp.CallToolRequest, in InstanceGetInput) (*mcp.CallToolResult, *api.Instance, error) {
	if in.Name == "" {
		return toolError[*api.Instance]("instance_get", errRequired("name"))
	}
	inst, _, err := s.client.Server.GetInstance(in.Name)
	if err != nil {
		return toolError[*api.Instance]("instance_get", err)
	}
	return result(inst)
}

// ---- creation / deletion ----

// InstanceCreateInput creates an instance.
type InstanceCreateInput struct {
	Name    string `json:"name" jsonschema:"the instance name"`
	Project string `json:"project,omitempty" jsonschema:"project to create the instance in (defaults to configured default)"`

	// Source is the image source (fingerprint or alias). Omit for empty instances.
	Source string `json:"source,omitempty" jsonschema:"image source: fingerprint or alias; omit to create an empty instance"`
	// Server is the source server (defaults to the target itself).
	Server string `json:"server,omitempty" jsonschema:"remote image server, if any"`

	// Type is the instance type: container (default) or virtual-machine.
	Type string `json:"type,omitempty" jsonschema:"instance type: container or virtual-machine"`

	// Config is the instance config (key/value).
	Config map[string]string `json:"config,omitempty" jsonschema:"instance configuration keys/values"`
	// Devices are the instance devices.
	Devices map[string]map[string]string `json:"devices,omitempty" jsonschema:"instance devices"`
	// Profiles are the profiles to apply (overrides defaults).
	Profiles []string `json:"profiles,omitempty" jsonschema:"profile names to apply"`

	// WaitTimeoutSeconds overrides the configured wait timeout.
	WaitTimeoutSeconds int `json:"wait_timeout_seconds,omitempty" jsonschema:"wait timeout override in seconds"`
}

func (s *Server) instanceCreate(ctx context.Context, req *mcp.CallToolRequest, in InstanceCreateInput) (*mcp.CallToolResult, *api.Operation, error) {
	if in.Name == "" {
		return toolError[*api.Operation]("instance_create", errRequired("name"))
	}
	itype := api.InstanceTypeContainer
	if in.Type == "virtual-machine" {
		itype = api.InstanceTypeVM
	}

	post := api.InstancesPost{
		InstancePut: api.InstancePut{
			Config:   in.Config,
			Devices:  in.Devices,
			Profiles: in.Profiles,
		},
		Name: in.Name,
		Type: itype,
	}
	if in.Source != "" {
		post.Source = api.InstanceSource{
			Type:   "image",
			Alias:  in.Source,
			Server: in.Server,
		}
	}

	op, err := s.client.Server.CreateInstance(post)
	if err != nil {
		return toolError[*api.Operation]("instance_create", err)
	}
	return s.waitResult("instance_create", op, in.WaitTimeoutSeconds)
}

// InstanceDeleteInput deletes an instance.
type InstanceDeleteInput struct {
	Name    string `json:"name" jsonschema:"the instance name"`
	Project string `json:"project,omitempty" jsonschema:"project the instance is in (defaults to configured default)"`

	// DeleteSnapshots deletes snapshots too.
	DeleteSnapshots bool `json:"delete_snapshots,omitempty" jsonschema:"delete the instance's snapshots as well"`
	// Force forces deletion even when running.
	Force bool `json:"force,omitempty" jsonschema:"force deletion (stop running instance first)"`

	WaitTimeoutSeconds int `json:"wait_timeout_seconds,omitempty" jsonschema:"wait timeout override in seconds"`
}

func (s *Server) instanceDelete(ctx context.Context, req *mcp.CallToolRequest, in InstanceDeleteInput) (*mcp.CallToolResult, *api.Operation, error) {
	if in.Name == "" {
		return toolError[*api.Operation]("instance_delete", errRequired("name"))
	}
	op, err := s.client.Server.DeleteInstance(in.Name)
	if err != nil {
		return toolError[*api.Operation]("instance_delete", err)
	}
	return s.waitResult("instance_delete", op, in.WaitTimeoutSeconds)
}

// ---- lifecycle ----

// InstanceStateChangeInput changes instance state.
type InstanceStateChangeInput struct {
	Name    string `json:"name" jsonschema:"the instance name"`
	Project string `json:"project,omitempty" jsonschema:"project the instance is in (defaults to configured default)"`
	Action  string `json:"action" jsonschema:"lifecycle action: start, stop, restart, freeze, unfreeze"`
	Force   bool   `json:"force,omitempty" jsonschema:"force the action (e.g. hard stop)"`
	Timeout int    `json:"timeout,omitempty" jsonschema:"stop timeout in seconds for the stop action"`

	WaitTimeoutSeconds int `json:"wait_timeout_seconds,omitempty" jsonschema:"wait timeout override in seconds"`
}

func (s *Server) instanceStateChange(ctx context.Context, req *mcp.CallToolRequest, in InstanceStateChangeInput) (*mcp.CallToolResult, *api.Operation, error) {
	if in.Name == "" {
		return toolError[*api.Operation]("instance_state_change", errRequired("name"))
	}
	allowed := map[string]bool{"start": true, "stop": true, "restart": true, "freeze": true, "unfreeze": true}
	if !allowed[in.Action] {
		return toolError[*api.Operation]("instance_state_change", errRequired("action (start|stop|restart|freeze|unfreeze)"))
	}

	put := api.InstanceStatePut{
		Action:  in.Action,
		Force:   in.Force,
		Timeout: in.Timeout,
	}

	op, err := s.client.Server.UpdateInstanceState(in.Name, put, "")
	if err != nil {
		return toolError[*api.Operation]("instance_state_change", err)
	}
	return s.waitResult("instance_state_change", op, in.WaitTimeoutSeconds)
}

// ---- rename / move ----

// InstanceMoveInput renames or moves an instance.
type InstanceMoveInput struct {
	Name    string `json:"name" jsonschema:"the current instance name"`
	NewName string `json:"new_name,omitempty" jsonschema:"the new instance name"`
	Project string `json:"project,omitempty" jsonschema:"project the instance is in (defaults to configured default)"`

	// Pool moves the instance to a different storage pool.
	Pool string `json:"pool,omitempty" jsonschema:"target storage pool for a move"`
	// Target moves the instance to a specific cluster member.
	Target string `json:"target,omitempty" jsonschema:"target cluster member for a move"`

	WaitTimeoutSeconds int `json:"wait_timeout_seconds,omitempty" jsonschema:"wait timeout override in seconds"`
}

func (s *Server) instanceMove(ctx context.Context, req *mcp.CallToolRequest, in InstanceMoveInput) (*mcp.CallToolResult, *api.Operation, error) {
	if in.Name == "" {
		return toolError[*api.Operation]("instance_move", errRequired("name"))
	}
	if in.NewName == "" && in.Pool == "" && in.Target == "" {
		return toolError[*api.Operation]("instance_move", errRequired("new_name, pool, or target"))
	}

	post := api.InstancePost{
		Name: in.NewName,
		Pool: in.Pool,
	}

	op, err := s.client.Server.MigrateInstance(in.Name, post)
	if err != nil {
		return toolError[*api.Operation]("instance_move", err)
	}
	return s.waitResult("instance_move", op, in.WaitTimeoutSeconds)
}

// ---- state ----

// InstanceStateInput fetches instance state.
type InstanceStateInput struct {
	Name    string `json:"name" jsonschema:"the instance name"`
	Project string `json:"project,omitempty" jsonschema:"project the instance is in (defaults to configured default)"`
}

func (s *Server) instanceState(ctx context.Context, req *mcp.CallToolRequest, in InstanceStateInput) (*mcp.CallToolResult, *api.InstanceState, error) {
	if in.Name == "" {
		return toolError[*api.InstanceState]("instance_state", errRequired("name"))
	}
	state, _, err := s.client.Server.GetInstanceState(in.Name)
	if err != nil {
		return toolError[*api.InstanceState]("instance_state", err)
	}
	return result(state)
}

// ---- snapshots ----

// SnapshotCreateInput creates a snapshot.
type SnapshotCreateInput struct {
	InstanceName string `json:"instance_name" jsonschema:"the instance name"`
	Project      string `json:"project,omitempty" jsonschema:"project the instance is in (defaults to configured default)"`
	SnapshotName string `json:"snapshot_name,omitempty" jsonschema:"snapshot name (defaults to auto-generated)"`
	Stateful     bool   `json:"stateful,omitempty" jsonschema:"include running memory state"`

	WaitTimeoutSeconds int `json:"wait_timeout_seconds,omitempty" jsonschema:"wait timeout override in seconds"`
}

func (s *Server) snapshotCreate(ctx context.Context, req *mcp.CallToolRequest, in SnapshotCreateInput) (*mcp.CallToolResult, *api.Operation, error) {
	if in.InstanceName == "" {
		return toolError[*api.Operation]("snapshot_create", errRequired("instance_name"))
	}
	post := api.InstanceSnapshotsPost{
		Name:     in.SnapshotName,
		Stateful: in.Stateful,
	}
	op, err := s.client.Server.CreateInstanceSnapshot(in.InstanceName, post)
	if err != nil {
		return toolError[*api.Operation]("snapshot_create", err)
	}
	return s.waitResult("snapshot_create", op, in.WaitTimeoutSeconds)
}

// SnapshotListInput lists snapshots.
type SnapshotListInput struct {
	InstanceName string `json:"instance_name" jsonschema:"the instance name"`
	Project      string `json:"project,omitempty" jsonschema:"project the instance is in (defaults to configured default)"`
}

func (s *Server) snapshotList(ctx context.Context, req *mcp.CallToolRequest, in SnapshotListInput) (*mcp.CallToolResult, []api.InstanceSnapshot, error) {
	if in.InstanceName == "" {
		return toolError[[]api.InstanceSnapshot]("snapshot_list", errRequired("instance_name"))
	}
	snaps, err := s.client.Server.GetInstanceSnapshots(in.InstanceName)
	if err != nil {
		return toolError[[]api.InstanceSnapshot]("snapshot_list", err)
	}
	return result(snaps)
}

// SnapshotDeleteInput deletes a snapshot.
type SnapshotDeleteInput struct {
	InstanceName string `json:"instance_name" jsonschema:"the instance name"`
	SnapshotName string `json:"snapshot_name" jsonschema:"the snapshot name"`
	Project      string `json:"project,omitempty" jsonschema:"project the instance is in (defaults to configured default)"`

	WaitTimeoutSeconds int `json:"wait_timeout_seconds,omitempty" jsonschema:"wait timeout override in seconds"`
}

func (s *Server) snapshotDelete(ctx context.Context, req *mcp.CallToolRequest, in SnapshotDeleteInput) (*mcp.CallToolResult, *api.Operation, error) {
	if in.InstanceName == "" || in.SnapshotName == "" {
		return toolError[*api.Operation]("snapshot_delete", errRequired("instance_name and snapshot_name"))
	}
	op, err := s.client.Server.DeleteInstanceSnapshot(in.InstanceName, in.SnapshotName)
	if err != nil {
		return toolError[*api.Operation]("snapshot_delete", err)
	}
	return s.waitResult("snapshot_delete", op, in.WaitTimeoutSeconds)
}

// SnapshotRestoreInput restores an instance from a snapshot.
type SnapshotRestoreInput struct {
	InstanceName string `json:"instance_name" jsonschema:"the instance name"`
	SnapshotName string `json:"snapshot_name" jsonschema:"the snapshot to restore from"`
	Project      string `json:"project,omitempty" jsonschema:"project the instance is in (defaults to configured default)"`
	RestoreState bool   `json:"restore_state,omitempty" jsonschema:"restore the running state too"`

	WaitTimeoutSeconds int `json:"wait_timeout_seconds,omitempty" jsonschema:"wait timeout override in seconds"`
}

func (s *Server) snapshotRestore(ctx context.Context, req *mcp.CallToolRequest, in SnapshotRestoreInput) (*mcp.CallToolResult, *api.Operation, error) {
	if in.InstanceName == "" || in.SnapshotName == "" {
		return toolError[*api.Operation]("snapshot_restore", errRequired("instance_name and snapshot_name"))
	}
	// Restore is PUT /instances/<name> with InstancePut.Restore set to the
	// target snapshot.
	put := api.InstancePut{
		Restore:  in.SnapshotName,
		DiskOnly: !in.RestoreState,
	}
	op, err := s.client.Server.UpdateInstance(in.InstanceName, put, "")
	if err != nil {
		return toolError[*api.Operation]("snapshot_restore", err)
	}
	return s.waitResult("snapshot_restore", op, in.WaitTimeoutSeconds)
}

// ---- backups ----

// BackupCreateInput creates a backup.
type BackupCreateInput struct {
	InstanceName string `json:"instance_name" jsonschema:"the instance name"`
	Project      string `json:"project,omitempty" jsonschema:"project the instance is in (defaults to configured default)"`
	BackupName   string `json:"backup_name,omitempty" jsonschema:"backup name (defaults to auto-generated)"`
	// Optimized uses the storage driver's optimized export.
	Optimized bool `json:"optimized,omitempty" jsonschema:"use the storage-optimized export format"`
	// Compression is the compression algorithm (e.g. gzip, none).
	Compression string `json:"compression,omitempty" jsonschema:"compression algorithm"`

	WaitTimeoutSeconds int `json:"wait_timeout_seconds,omitempty" jsonschema:"wait timeout override in seconds"`
}

func (s *Server) backupCreate(ctx context.Context, req *mcp.CallToolRequest, in BackupCreateInput) (*mcp.CallToolResult, *api.Operation, error) {
	if in.InstanceName == "" {
		return toolError[*api.Operation]("backup_create", errRequired("instance_name"))
	}
	post := api.InstanceBackupsPost{
		Name:                 in.BackupName,
		OptimizedStorage:     in.Optimized,
		CompressionAlgorithm: in.Compression,
	}
	op, err := s.client.Server.CreateInstanceBackup(in.InstanceName, post)
	if err != nil {
		return toolError[*api.Operation]("backup_create", err)
	}
	return s.waitResult("backup_create", op, in.WaitTimeoutSeconds)
}

// BackupListInput lists backups.
type BackupListInput struct {
	InstanceName string `json:"instance_name" jsonschema:"the instance name"`
	Project      string `json:"project,omitempty" jsonschema:"project the instance is in (defaults to configured default)"`
}

func (s *Server) backupList(ctx context.Context, req *mcp.CallToolRequest, in BackupListInput) (*mcp.CallToolResult, []api.InstanceBackup, error) {
	if in.InstanceName == "" {
		return toolError[[]api.InstanceBackup]("backup_list", errRequired("instance_name"))
	}
	backups, err := s.client.Server.GetInstanceBackups(in.InstanceName)
	if err != nil {
		return toolError[[]api.InstanceBackup]("backup_list", err)
	}
	return result(backups)
}

// BackupDeleteInput deletes a backup.
type BackupDeleteInput struct {
	InstanceName string `json:"instance_name" jsonschema:"the instance name"`
	BackupName   string `json:"backup_name" jsonschema:"the backup name"`
	Project      string `json:"project,omitempty" jsonschema:"project the instance is in (defaults to configured default)"`

	WaitTimeoutSeconds int `json:"wait_timeout_seconds,omitempty" jsonschema:"wait timeout override in seconds"`
}

func (s *Server) backupDelete(ctx context.Context, req *mcp.CallToolRequest, in BackupDeleteInput) (*mcp.CallToolResult, *api.Operation, error) {
	if in.InstanceName == "" || in.BackupName == "" {
		return toolError[*api.Operation]("backup_delete", errRequired("instance_name and backup_name"))
	}
	op, err := s.client.Server.DeleteInstanceBackup(in.InstanceName, in.BackupName)
	if err != nil {
		return toolError[*api.Operation]("backup_delete", err)
	}
	return s.waitResult("backup_delete", op, in.WaitTimeoutSeconds)
}

// BackupExportInput exports a backup artifact.
type BackupExportInput struct {
	InstanceName string `json:"instance_name" jsonschema:"the instance name"`
	BackupName   string `json:"backup_name" jsonschema:"the backup name"`
	Project      string `json:"project,omitempty" jsonschema:"project the instance is in (defaults to configured default)"`
	// DestPath writes the artifact to this path on the MCP server host.
	DestPath string `json:"dest_path,omitempty" jsonschema:"local path to write the exported artifact to"`
}

// BackupExportOutput reports the export location.
type BackupExportOutput struct {
	// Path is where the backup was written (staged file reference).
	Path string `json:"path" jsonschema:"path of the exported backup artifact on the MCP server host"`
	Size int64  `json:"size" jsonschema:"size of the exported artifact in bytes"`
}

func (s *Server) backupExport(ctx context.Context, req *mcp.CallToolRequest, in BackupExportInput) (*mcp.CallToolResult, BackupExportOutput, error) {
	if in.InstanceName == "" || in.BackupName == "" {
		return toolError[BackupExportOutput]("backup_export", errRequired("instance_name and backup_name"))
	}
	if in.DestPath == "" {
		return toolError[BackupExportOutput]("backup_export", errRequired("dest_path"))
	}

	// Open the destination file and stream the backup artifact into it.
	f, err := os.Create(in.DestPath)
	if err != nil {
		return toolError[BackupExportOutput]("backup_export", err)
	}
	defer f.Close()

	resp, err := s.client.Server.GetInstanceBackupFile(in.InstanceName, in.BackupName, &incusclient.BackupFileRequest{
		BackupFile: f,
	})
	if err != nil {
		return toolError[BackupExportOutput]("backup_export", err)
	}

	return result(BackupExportOutput{Path: in.DestPath, Size: resp.Size})
}

// ---- registration ----

func (s *Server) registerInstanceTools() {
	addTool(s, "instance_list", "List instances (optionally all projects or filtered).", s.instanceList)
	addTool(s, "instance_get", "Fetch a single instance's full config.", s.instanceGet)
	addTool(s, "instance_create", "Create an instance from an image source or empty, with config/devices/profiles.", s.instanceCreate)
	addTool(s, "instance_delete", "Delete an instance (optionally with snapshots, force).", s.instanceDelete)
	addTool(s, "instance_state_change", "Start, stop, restart, freeze, or unfreeze an instance.", s.instanceStateChange)
	addTool(s, "instance_move", "Rename an instance or move it to another pool/cluster member.", s.instanceMove)
	addTool(s, "instance_state", "Fetch an instance's live state.", s.instanceState)
	addTool(s, "snapshot_create", "Create an instance snapshot.", s.snapshotCreate)
	addTool(s, "snapshot_list", "List an instance's snapshots.", s.snapshotList)
	addTool(s, "snapshot_delete", "Delete an instance snapshot.", s.snapshotDelete)
	addTool(s, "snapshot_restore", "Restore an instance from a snapshot.", s.snapshotRestore)
	addTool(s, "backup_create", "Create an instance backup.", s.backupCreate)
	addTool(s, "backup_list", "List an instance's backups.", s.backupList)
	addTool(s, "backup_delete", "Delete an instance backup.", s.backupDelete)
	addTool(s, "backup_export", "Export an instance backup to a file on the MCP server host.", s.backupExport)
}

// waitResult waits on an operation and returns its final state.
func (s *Server) waitResult(op string, opHandle interface {
	Wait() error
	Get() api.Operation
}, timeoutOverride int) (*mcp.CallToolResult, *api.Operation, error) {
	timeout := s.client.WaitTimeout()
	if timeoutOverride > 0 {
		timeout = time.Duration(timeoutOverride) * time.Second
	}

	done := make(chan *api.Operation, 1)
	errCh := make(chan error, 1)
	go func() {
		if err := opHandle.Wait(); err != nil {
			errCh <- err
			return
		}
		cur := opHandle.Get()
		done <- &cur
	}()

	select {
	case final := <-done:
		return result(final)
	case err := <-errCh:
		return toolError[*api.Operation](op, err)
	case <-time.After(timeout):
		cur := opHandle.Get()
		return result(&api.Operation{ID: cur.ID, Status: "running"})
	}
}
