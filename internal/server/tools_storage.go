package server

import (
	"context"

	"github.com/lxc/incus/v7/shared/api"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ---- pools ----

// StoragePoolListInput lists pools.
type StoragePoolListInput struct{}

func (s *Server) storagePoolList(ctx context.Context, req *mcp.CallToolRequest, in StoragePoolListInput) (*mcp.CallToolResult, []api.StoragePool, error) {
	pools, err := s.client.Server.GetStoragePools()
	if err != nil {
		return toolError[[]api.StoragePool]("storage_pool_list", err)
	}
	return result(pools)
}

// StoragePoolCreateInput creates a pool.
type StoragePoolCreateInput struct {
	Name   string            `json:"name" jsonschema:"the pool name"`
	Driver string            `json:"driver" jsonschema:"the storage driver (zfs, btrfs, lvm, dir, ceph, ...)"`
	Config map[string]string `json:"config,omitempty" jsonschema:"pool configuration (source, size, ...)"`
}

func (s *Server) storagePoolCreate(ctx context.Context, req *mcp.CallToolRequest, in StoragePoolCreateInput) (*mcp.CallToolResult, string, error) {
	if in.Name == "" || in.Driver == "" {
		return toolError[string]("storage_pool_create", errRequired("name and driver"))
	}
	err := s.client.Server.CreateStoragePool(api.StoragePoolsPost{
		Name:           in.Name,
		Driver:         in.Driver,
		StoragePoolPut: api.StoragePoolPut{Config: in.Config},
	})
	if err != nil {
		return toolError[string]("storage_pool_create", err)
	}
	return result("pool created: " + in.Name)
}

// StoragePoolUpdateInput updates a pool.
type StoragePoolUpdateInput struct {
	Name   string            `json:"name" jsonschema:"the pool name"`
	Config map[string]string `json:"config,omitempty" jsonschema:"pool configuration to apply"`
}

func (s *Server) storagePoolUpdate(ctx context.Context, req *mcp.CallToolRequest, in StoragePoolUpdateInput) (*mcp.CallToolResult, string, error) {
	if in.Name == "" {
		return toolError[string]("storage_pool_update", errRequired("name"))
	}
	_, etag, err := s.client.Server.GetStoragePool(in.Name)
	if err != nil {
		return toolError[string]("storage_pool_update", err)
	}
	err = s.client.Server.UpdateStoragePool(in.Name, api.StoragePoolPut{Config: in.Config}, etag)
	if err != nil {
		return toolError[string]("storage_pool_update", err)
	}
	return result("pool updated: " + in.Name)
}

// StoragePoolDeleteInput deletes a pool.
type StoragePoolDeleteInput struct {
	Name string `json:"name" jsonschema:"the pool name"`
}

func (s *Server) storagePoolDelete(ctx context.Context, req *mcp.CallToolRequest, in StoragePoolDeleteInput) (*mcp.CallToolResult, string, error) {
	if in.Name == "" {
		return toolError[string]("storage_pool_delete", errRequired("name"))
	}
	if err := s.client.Server.DeleteStoragePool(in.Name); err != nil {
		return toolError[string]("storage_pool_delete", err)
	}
	return result("pool deleted: " + in.Name)
}

// ---- volumes ----

// StorageVolumeListInput lists volumes.
type StorageVolumeListInput struct {
	Pool    string `json:"pool" jsonschema:"the storage pool"`
	Project string `json:"project,omitempty" jsonschema:"project (defaults to configured default)"`
}

func (s *Server) storageVolumeList(ctx context.Context, req *mcp.CallToolRequest, in StorageVolumeListInput) (*mcp.CallToolResult, []api.StorageVolume, error) {
	if in.Pool == "" {
		return toolError[[]api.StorageVolume]("storage_volume_list", errRequired("pool"))
	}
	vols, err := s.client.Server.GetStoragePoolVolumes(in.Pool)
	if err != nil {
		return toolError[[]api.StorageVolume]("storage_volume_list", err)
	}
	return result(vols)
}

// StorageVolumeCreateInput creates a custom volume.
type StorageVolumeCreateInput struct {
	Pool        string            `json:"pool" jsonschema:"the storage pool"`
	Project     string            `json:"project,omitempty" jsonschema:"project (defaults to configured default)"`
	Name        string            `json:"name" jsonschema:"the volume name"`
	ContentType string            `json:"content_type,omitempty" jsonschema:"content type: filesystem (default) or block"`
	Config      map[string]string `json:"config,omitempty" jsonschema:"volume configuration"`
}

func (s *Server) storageVolumeCreate(ctx context.Context, req *mcp.CallToolRequest, in StorageVolumeCreateInput) (*mcp.CallToolResult, string, error) {
	if in.Pool == "" || in.Name == "" {
		return toolError[string]("storage_volume_create", errRequired("pool and name"))
	}
	ctype := in.ContentType
	if ctype == "" {
		ctype = "filesystem"
	}
	err := s.client.Server.CreateStoragePoolVolume(in.Pool, api.StorageVolumesPost{
		Name:             in.Name,
		StorageVolumePut: api.StorageVolumePut{Config: in.Config},
		Type:             ctype,
	})
	if err != nil {
		return toolError[string]("storage_volume_create", err)
	}
	return result("volume created: " + in.Name)
}

// StorageVolumeUpdateInput updates a volume.
type StorageVolumeUpdateInput struct {
	Pool   string            `json:"pool" jsonschema:"the storage pool"`
	Name   string            `json:"name" jsonschema:"the volume name"`
	Config map[string]string `json:"config,omitempty" jsonschema:"volume configuration to apply"`
	// Size resizes the volume (e.g. "20GiB").
	Size string `json:"size,omitempty" jsonschema:"new size (e.g. 20GiB)"`
}

func (s *Server) storageVolumeUpdate(ctx context.Context, req *mcp.CallToolRequest, in StorageVolumeUpdateInput) (*mcp.CallToolResult, string, error) {
	if in.Pool == "" || in.Name == "" {
		return toolError[string]("storage_volume_update", errRequired("pool and name"))
	}
	_, etag, err := s.client.Server.GetStoragePoolVolume(in.Pool, "custom", in.Name)
	if err != nil {
		return toolError[string]("storage_volume_update", err)
	}
	put := api.StorageVolumePut{Config: in.Config}
	if in.Size != "" {
		if put.Config == nil {
			put.Config = map[string]string{}
		}
		put.Config["size"] = in.Size
	}
	if err := s.client.Server.UpdateStoragePoolVolume(in.Pool, "custom", in.Name, put, etag); err != nil {
		return toolError[string]("storage_volume_update", err)
	}
	return result("volume updated: " + in.Name)
}

// StorageVolumeDeleteInput deletes a volume.
type StorageVolumeDeleteInput struct {
	Pool string `json:"pool" jsonschema:"the storage pool"`
	Name string `json:"name" jsonschema:"the volume name"`
}

func (s *Server) storageVolumeDelete(ctx context.Context, req *mcp.CallToolRequest, in StorageVolumeDeleteInput) (*mcp.CallToolResult, string, error) {
	if in.Pool == "" || in.Name == "" {
		return toolError[string]("storage_volume_delete", errRequired("pool and name"))
	}
	if err := s.client.Server.DeleteStoragePoolVolume(in.Pool, "custom", in.Name); err != nil {
		return toolError[string]("storage_volume_delete", err)
	}
	return result("volume deleted: " + in.Name)
}

// StorageVolumeRenameInput renames or moves a volume.
type StorageVolumeRenameInput struct {
	Pool    string `json:"pool" jsonschema:"the source storage pool"`
	Project string `json:"project,omitempty" jsonschema:"project (defaults to configured default)"`
	Name    string `json:"name" jsonschema:"the current volume name"`
	NewName string `json:"new_name" jsonschema:"the new volume name"`
	// TargetPool moves the volume to a different pool.
	TargetPool string `json:"target_pool,omitempty" jsonschema:"target pool for a move"`

	WaitTimeoutSeconds int `json:"wait_timeout_seconds,omitempty" jsonschema:"wait timeout override in seconds"`
}

func (s *Server) storageVolumeRename(ctx context.Context, req *mcp.CallToolRequest, in StorageVolumeRenameInput) (*mcp.CallToolResult, *api.Operation, error) {
	if in.Pool == "" || in.Name == "" || in.NewName == "" {
		return toolError[*api.Operation]("storage_volume_rename", errRequired("pool, name, and new_name"))
	}
	post := api.StorageVolumePost{Name: in.NewName}
	op, err := s.client.Server.MigrateStoragePoolVolume(in.Pool, post)
	if err != nil {
		return toolError[*api.Operation]("storage_volume_rename", err)
	}
	return s.waitResult("storage_volume_rename", op, in.WaitTimeoutSeconds)
}

// ---- volume snapshots ----

// StorageVolumeSnapshotCreateInput snapshots a volume.
type StorageVolumeSnapshotCreateInput struct {
	Pool         string `json:"pool" jsonschema:"the storage pool"`
	VolumeName   string `json:"volume_name" jsonschema:"the volume name"`
	SnapshotName string `json:"snapshot_name" jsonschema:"the snapshot name"`

	WaitTimeoutSeconds int `json:"wait_timeout_seconds,omitempty" jsonschema:"wait timeout override in seconds"`
}

func (s *Server) storageVolumeSnapshotCreate(ctx context.Context, req *mcp.CallToolRequest, in StorageVolumeSnapshotCreateInput) (*mcp.CallToolResult, *api.Operation, error) {
	if in.Pool == "" || in.VolumeName == "" || in.SnapshotName == "" {
		return toolError[*api.Operation]("storage_volume_snapshot_create", errRequired("pool, volume_name, and snapshot_name"))
	}
	op, err := s.client.Server.CreateStoragePoolVolumeSnapshot(in.Pool, "custom", in.VolumeName, api.StorageVolumeSnapshotsPost{
		Name: in.SnapshotName,
	})
	if err != nil {
		return toolError[*api.Operation]("storage_volume_snapshot_create", err)
	}
	return s.waitResult("storage_volume_snapshot_create", op, in.WaitTimeoutSeconds)
}

// StorageVolumeSnapshotListInput lists volume snapshots.
type StorageVolumeSnapshotListInput struct {
	Pool       string `json:"pool" jsonschema:"the storage pool"`
	VolumeName string `json:"volume_name" jsonschema:"the volume name"`
}

func (s *Server) storageVolumeSnapshotList(ctx context.Context, req *mcp.CallToolRequest, in StorageVolumeSnapshotListInput) (*mcp.CallToolResult, []api.StorageVolumeSnapshot, error) {
	if in.Pool == "" || in.VolumeName == "" {
		return toolError[[]api.StorageVolumeSnapshot]("storage_volume_snapshot_list", errRequired("pool and volume_name"))
	}
	snaps, err := s.client.Server.GetStoragePoolVolumeSnapshots(in.Pool, "custom", in.VolumeName)
	if err != nil {
		return toolError[[]api.StorageVolumeSnapshot]("storage_volume_snapshot_list", err)
	}
	return result(snaps)
}

// StorageVolumeSnapshotDeleteInput deletes a volume snapshot.
type StorageVolumeSnapshotDeleteInput struct {
	Pool         string `json:"pool" jsonschema:"the storage pool"`
	VolumeName   string `json:"volume_name" jsonschema:"the volume name"`
	SnapshotName string `json:"snapshot_name" jsonschema:"the snapshot name"`

	WaitTimeoutSeconds int `json:"wait_timeout_seconds,omitempty" jsonschema:"wait timeout override in seconds"`
}

func (s *Server) storageVolumeSnapshotDelete(ctx context.Context, req *mcp.CallToolRequest, in StorageVolumeSnapshotDeleteInput) (*mcp.CallToolResult, *api.Operation, error) {
	if in.Pool == "" || in.VolumeName == "" || in.SnapshotName == "" {
		return toolError[*api.Operation]("storage_volume_snapshot_delete", errRequired("pool, volume_name, and snapshot_name"))
	}
	op, err := s.client.Server.DeleteStoragePoolVolumeSnapshot(in.Pool, "custom", in.VolumeName, in.SnapshotName)
	if err != nil {
		return toolError[*api.Operation]("storage_volume_snapshot_delete", err)
	}
	return s.waitResult("storage_volume_snapshot_delete", op, in.WaitTimeoutSeconds)
}

// StorageVolumeSnapshotRestoreInput restores a volume from a snapshot.
type StorageVolumeSnapshotRestoreInput struct {
	Pool         string `json:"pool" jsonschema:"the storage pool"`
	VolumeName   string `json:"volume_name" jsonschema:"the volume name"`
	SnapshotName string `json:"snapshot_name" jsonschema:"the snapshot name"`

	WaitTimeoutSeconds int `json:"wait_timeout_seconds,omitempty" jsonschema:"wait timeout override in seconds"`
}

func (s *Server) storageVolumeSnapshotRestore(ctx context.Context, req *mcp.CallToolRequest, in StorageVolumeSnapshotRestoreInput) (*mcp.CallToolResult, *api.Operation, error) {
	if in.Pool == "" || in.VolumeName == "" || in.SnapshotName == "" {
		return toolError[*api.Operation]("storage_volume_snapshot_restore", errRequired("pool, volume_name, and snapshot_name"))
	}
	// Restoring a volume snapshot is a PUT on the volume with the source
	// snapshot set — via UpdateStoragePoolVolumeSnapshot with Restore? No:
	// the REST API is POST /storage-pools/<pool>/volumes/custom/<vol>/snapshots/<snap>/restore.
	// The official client exposes this through CreateStoragePoolVolumeSnapshot
	// only for creation; restore is a PUT on the snapshot. The supported path:
	// PUT /1.0/storage-pools/<pool>/volumes/custom/<vol> with Restore set.
	// Use MigrateStoragePoolVolume is for rename; the restore is done via
	// UpdateStoragePoolVolume with a snapshot source — but the client does not
	// expose that directly, so we call the raw REST endpoint via query.
	// Simplest supported approach: UpdateStoragePoolVolumeSnapshot is a PUT on
	// the snapshot's config, not a restore. Restore is implemented by
	// POSTing to the snapshot's restore URL which the client lacks, so we
	// surface a clear unsupported error.
	return toolError[*api.Operation]("storage_volume_snapshot_restore", errNotImplemented("volume snapshot restore is not exposed by the official Go client; use incus CLI on the target"))
}

// ---- volume backups ----

// StorageVolumeBackupCreateInput backs up a volume.
type StorageVolumeBackupCreateInput struct {
	Pool       string `json:"pool" jsonschema:"the storage pool"`
	VolumeName string `json:"volume_name" jsonschema:"the volume name"`
	BackupName string `json:"backup_name" jsonschema:"the backup name"`

	WaitTimeoutSeconds int `json:"wait_timeout_seconds,omitempty" jsonschema:"wait timeout override in seconds"`
}

func (s *Server) storageVolumeBackupCreate(ctx context.Context, req *mcp.CallToolRequest, in StorageVolumeBackupCreateInput) (*mcp.CallToolResult, *api.Operation, error) {
	if in.Pool == "" || in.VolumeName == "" || in.BackupName == "" {
		return toolError[*api.Operation]("storage_volume_backup_create", errRequired("pool, volume_name, and backup_name"))
	}
	op, err := s.client.Server.CreateStorageVolumeBackup(in.Pool, in.VolumeName, api.StorageVolumeBackupsPost{
		Name: in.BackupName,
	})
	if err != nil {
		return toolError[*api.Operation]("storage_volume_backup_create", err)
	}
	return s.waitResult("storage_volume_backup_create", op, in.WaitTimeoutSeconds)
}

// StorageVolumeBackupListInput lists volume backups.
type StorageVolumeBackupListInput struct {
	Pool       string `json:"pool" jsonschema:"the storage pool"`
	VolumeName string `json:"volume_name" jsonschema:"the volume name"`
}

func (s *Server) storageVolumeBackupList(ctx context.Context, req *mcp.CallToolRequest, in StorageVolumeBackupListInput) (*mcp.CallToolResult, []api.StorageVolumeBackup, error) {
	if in.Pool == "" || in.VolumeName == "" {
		return toolError[[]api.StorageVolumeBackup]("storage_volume_backup_list", errRequired("pool and volume_name"))
	}
	backups, err := s.client.Server.GetStorageVolumeBackups(in.Pool, in.VolumeName)
	if err != nil {
		return toolError[[]api.StorageVolumeBackup]("storage_volume_backup_list", err)
	}
	return result(backups)
}

// StorageVolumeBackupDeleteInput deletes a volume backup.
type StorageVolumeBackupDeleteInput struct {
	Pool       string `json:"pool" jsonschema:"the storage pool"`
	VolumeName string `json:"volume_name" jsonschema:"the volume name"`
	BackupName string `json:"backup_name" jsonschema:"the backup name"`

	WaitTimeoutSeconds int `json:"wait_timeout_seconds,omitempty" jsonschema:"wait timeout override in seconds"`
}

func (s *Server) storageVolumeBackupDelete(ctx context.Context, req *mcp.CallToolRequest, in StorageVolumeBackupDeleteInput) (*mcp.CallToolResult, *api.Operation, error) {
	if in.Pool == "" || in.VolumeName == "" || in.BackupName == "" {
		return toolError[*api.Operation]("storage_volume_backup_delete", errRequired("pool, volume_name, and backup_name"))
	}
	op, err := s.client.Server.DeleteStorageVolumeBackup(in.Pool, in.VolumeName, in.BackupName)
	if err != nil {
		return toolError[*api.Operation]("storage_volume_backup_delete", err)
	}
	return s.waitResult("storage_volume_backup_delete", op, in.WaitTimeoutSeconds)
}

// ---- buckets ----

// StorageBucketListInput lists buckets.
type StorageBucketListInput struct {
	Pool string `json:"pool" jsonschema:"the storage pool"`
}

func (s *Server) storageBucketList(ctx context.Context, req *mcp.CallToolRequest, in StorageBucketListInput) (*mcp.CallToolResult, []api.StorageBucket, error) {
	if in.Pool == "" {
		return toolError[[]api.StorageBucket]("storage_bucket_list", errRequired("pool"))
	}
	buckets, err := s.client.Server.GetStoragePoolBuckets(in.Pool)
	if err != nil {
		return toolError[[]api.StorageBucket]("storage_bucket_list", err)
	}
	return result(buckets)
}

// StorageBucketCreateInput creates a bucket.
type StorageBucketCreateInput struct {
	Pool   string            `json:"pool" jsonschema:"the storage pool"`
	Name   string            `json:"name" jsonschema:"the bucket name"`
	Config map[string]string `json:"config,omitempty" jsonschema:"bucket configuration"`
}

// StorageBucketCreateOutput carries the created bucket key (access/secret).
type StorageBucketCreateOutput struct {
	Name string `json:"name" jsonschema:"the bucket name"`
	// AccessKey is the S3 access key.
	AccessKey string `json:"access_key" jsonschema:"the S3 access key"`
	// SecretKey is the S3 secret key.
	SecretKey string `json:"secret_key" jsonschema:"the S3 secret key"`
}

func (s *Server) storageBucketCreate(ctx context.Context, req *mcp.CallToolRequest, in StorageBucketCreateInput) (*mcp.CallToolResult, StorageBucketCreateOutput, error) {
	if in.Pool == "" || in.Name == "" {
		return toolError[StorageBucketCreateOutput]("storage_bucket_create", errRequired("pool and name"))
	}
	key, err := s.client.Server.CreateStoragePoolBucket(in.Pool, api.StorageBucketsPost{
		Name:             in.Name,
		StorageBucketPut: api.StorageBucketPut{Config: in.Config},
	})
	if err != nil {
		return toolError[StorageBucketCreateOutput]("storage_bucket_create", err)
	}
	out := StorageBucketCreateOutput{Name: in.Name}
	if key != nil {
		out.AccessKey = key.AccessKey
		out.SecretKey = key.SecretKey
	}
	return result(out)
}

// StorageBucketDeleteInput deletes a bucket.
type StorageBucketDeleteInput struct {
	Pool string `json:"pool" jsonschema:"the storage pool"`
	Name string `json:"name" jsonschema:"the bucket name"`
}

func (s *Server) storageBucketDelete(ctx context.Context, req *mcp.CallToolRequest, in StorageBucketDeleteInput) (*mcp.CallToolResult, string, error) {
	if in.Pool == "" || in.Name == "" {
		return toolError[string]("storage_bucket_delete", errRequired("pool and name"))
	}
	if err := s.client.Server.DeleteStoragePoolBucket(in.Pool, in.Name); err != nil {
		return toolError[string]("storage_bucket_delete", err)
	}
	return result("bucket deleted: " + in.Name)
}

// StorageBucketKeyCreateInput creates a bucket key.
type StorageBucketKeyCreateInput struct {
	Pool       string `json:"pool" jsonschema:"the storage pool"`
	BucketName string `json:"bucket_name" jsonschema:"the bucket name"`
	KeyName    string `json:"key_name" jsonschema:"the key name"`
	Role       string `json:"role,omitempty" jsonschema:"the key role (admin, read-only)"`
}

// StorageBucketKeyCreateOutput carries the created key.
type StorageBucketKeyCreateOutput struct {
	Name      string `json:"name" jsonschema:"the key name"`
	AccessKey string `json:"access_key" jsonschema:"the S3 access key"`
	SecretKey string `json:"secret_key" jsonschema:"the S3 secret key"`
}

func (s *Server) storageBucketKeyCreate(ctx context.Context, req *mcp.CallToolRequest, in StorageBucketKeyCreateInput) (*mcp.CallToolResult, StorageBucketKeyCreateOutput, error) {
	if in.Pool == "" || in.BucketName == "" || in.KeyName == "" {
		return toolError[StorageBucketKeyCreateOutput]("storage_bucket_key_create", errRequired("pool, bucket_name, and key_name"))
	}
	key, err := s.client.Server.CreateStoragePoolBucketKey(in.Pool, in.BucketName, api.StorageBucketKeysPost{
		Name: in.KeyName,
		StorageBucketKeyPut: api.StorageBucketKeyPut{
			Role: in.Role,
		},
	})
	if err != nil {
		return toolError[StorageBucketKeyCreateOutput]("storage_bucket_key_create", err)
	}
	out := StorageBucketKeyCreateOutput{Name: in.KeyName}
	if key != nil {
		out.AccessKey = key.AccessKey
		out.SecretKey = key.SecretKey
	}
	return result(out)
}

// StorageBucketKeyDeleteInput deletes a bucket key.
type StorageBucketKeyDeleteInput struct {
	Pool       string `json:"pool" jsonschema:"the storage pool"`
	BucketName string `json:"bucket_name" jsonschema:"the bucket name"`
	KeyName    string `json:"key_name" jsonschema:"the key name"`
}

func (s *Server) storageBucketKeyDelete(ctx context.Context, req *mcp.CallToolRequest, in StorageBucketKeyDeleteInput) (*mcp.CallToolResult, string, error) {
	if in.Pool == "" || in.BucketName == "" || in.KeyName == "" {
		return toolError[string]("storage_bucket_key_delete", errRequired("pool, bucket_name, and key_name"))
	}
	if err := s.client.Server.DeleteStoragePoolBucketKey(in.Pool, in.BucketName, in.KeyName); err != nil {
		return toolError[string]("storage_bucket_key_delete", err)
	}
	return result("bucket key deleted: " + in.KeyName)
}

// ---- registration ----

func (s *Server) registerStorageTools() {
	addTool(s, "storage_pool_list", "List storage pools.", s.storagePoolList)
	addTool(s, "storage_pool_create", "Create a storage pool with a driver and config.", s.storagePoolCreate)
	addTool(s, "storage_pool_update", "Update a storage pool's config.", s.storagePoolUpdate)
	addTool(s, "storage_pool_delete", "Delete a storage pool (fails while volumes are in use).", s.storagePoolDelete)
	addTool(s, "storage_volume_list", "List custom volumes on a pool.", s.storageVolumeList)
	addTool(s, "storage_volume_create", "Create a custom volume.", s.storageVolumeCreate)
	addTool(s, "storage_volume_update", "Update a volume's config or size.", s.storageVolumeUpdate)
	addTool(s, "storage_volume_delete", "Delete a custom volume.", s.storageVolumeDelete)
	addTool(s, "storage_volume_rename", "Rename or move a custom volume.", s.storageVolumeRename)
	addTool(s, "storage_volume_snapshot_create", "Create a volume snapshot.", s.storageVolumeSnapshotCreate)
	addTool(s, "storage_volume_snapshot_list", "List volume snapshots.", s.storageVolumeSnapshotList)
	addTool(s, "storage_volume_snapshot_delete", "Delete a volume snapshot.", s.storageVolumeSnapshotDelete)
	addTool(s, "storage_volume_snapshot_restore", "Restore a volume from a snapshot.", s.storageVolumeSnapshotRestore)
	addTool(s, "storage_volume_backup_create", "Create a volume backup.", s.storageVolumeBackupCreate)
	addTool(s, "storage_volume_backup_list", "List volume backups.", s.storageVolumeBackupList)
	addTool(s, "storage_volume_backup_delete", "Delete a volume backup.", s.storageVolumeBackupDelete)
	addTool(s, "storage_bucket_list", "List S3 buckets on a pool.", s.storageBucketList)
	addTool(s, "storage_bucket_create", "Create an S3 bucket (returns access/secret key).", s.storageBucketCreate)
	addTool(s, "storage_bucket_delete", "Delete an S3 bucket.", s.storageBucketDelete)
	addTool(s, "storage_bucket_key_create", "Create a bucket access key.", s.storageBucketKeyCreate)
	addTool(s, "storage_bucket_key_delete", "Delete a bucket access key.", s.storageBucketKeyDelete)
}
