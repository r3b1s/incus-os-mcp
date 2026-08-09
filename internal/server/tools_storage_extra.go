package server

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"strings"

	incusclient "github.com/lxc/incus/v7/client"
	"github.com/lxc/incus/v7/shared/api"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// StorageVolumeMoveInput moves a custom volume to another storage pool.
type StorageVolumeMoveInput struct {
	Pool               string `json:"pool" jsonschema:"the source storage pool"`
	Project            string `json:"project,omitempty" jsonschema:"project (defaults to configured default)"`
	Name               string `json:"name" jsonschema:"the source volume name"`
	TargetPool         string `json:"target_pool" jsonschema:"the target storage pool"`
	WaitTimeoutSeconds int    `json:"wait_timeout_seconds,omitempty" jsonschema:"wait timeout override in seconds"`
}

func (s *Server) storageVolumeMove(ctx context.Context, req *mcp.CallToolRequest, in StorageVolumeMoveInput) (*mcp.CallToolResult, *api.Operation, error) {
	if in.Pool == "" || in.Name == "" || in.TargetPool == "" {
		return toolError[*api.Operation]("storage_volume_move", errRequired("pool, name, and target_pool"))
	}
	op, err := s.projectServer(in.Project).MigrateStoragePoolVolume(in.Pool, api.StorageVolumePost{
		Name:      in.Name,
		Pool:      in.TargetPool,
		Migration: true,
	})
	if err != nil {
		return toolError[*api.Operation]("storage_volume_move", err)
	}
	return s.waitResult("storage_volume_move", op, in.WaitTimeoutSeconds)
}

// StorageVolumeSnapshotRenameInput renames a custom-volume snapshot.
type StorageVolumeSnapshotRenameInput struct {
	Pool               string `json:"pool" jsonschema:"the storage pool"`
	Project            string `json:"project,omitempty" jsonschema:"project (defaults to configured default)"`
	VolumeName         string `json:"volume_name" jsonschema:"the volume name"`
	SnapshotName       string `json:"snapshot_name" jsonschema:"the current snapshot name"`
	NewName            string `json:"new_name" jsonschema:"the new snapshot name"`
	WaitTimeoutSeconds int    `json:"wait_timeout_seconds,omitempty" jsonschema:"wait timeout override in seconds"`
}

func (s *Server) storageVolumeSnapshotRename(ctx context.Context, req *mcp.CallToolRequest, in StorageVolumeSnapshotRenameInput) (*mcp.CallToolResult, *api.Operation, error) {
	if in.Pool == "" || in.VolumeName == "" || in.SnapshotName == "" || in.NewName == "" {
		return toolError[*api.Operation]("storage_volume_snapshot_rename", errRequired("pool, volume_name, snapshot_name, and new_name"))
	}
	op, err := s.projectServer(in.Project).RenameStoragePoolVolumeSnapshot(in.Pool, "custom", in.VolumeName, in.SnapshotName, api.StorageVolumeSnapshotPost{Name: in.NewName})
	if err != nil {
		return toolError[*api.Operation]("storage_volume_snapshot_rename", err)
	}
	return s.waitResult("storage_volume_snapshot_rename", op, in.WaitTimeoutSeconds)
}

// StorageVolumeBackupRenameInput renames a custom-volume backup.
type StorageVolumeBackupRenameInput struct {
	Pool               string `json:"pool" jsonschema:"the storage pool"`
	Project            string `json:"project,omitempty" jsonschema:"project (defaults to configured default)"`
	VolumeName         string `json:"volume_name" jsonschema:"the volume name"`
	BackupName         string `json:"backup_name" jsonschema:"the current backup name"`
	NewName            string `json:"new_name" jsonschema:"the new backup name"`
	WaitTimeoutSeconds int    `json:"wait_timeout_seconds,omitempty" jsonschema:"wait timeout override in seconds"`
}

func (s *Server) storageVolumeBackupRename(ctx context.Context, req *mcp.CallToolRequest, in StorageVolumeBackupRenameInput) (*mcp.CallToolResult, *api.Operation, error) {
	if in.Pool == "" || in.VolumeName == "" || in.BackupName == "" || in.NewName == "" {
		return toolError[*api.Operation]("storage_volume_backup_rename", errRequired("pool, volume_name, backup_name, and new_name"))
	}
	op, err := s.projectServer(in.Project).RenameStorageVolumeBackup(in.Pool, in.VolumeName, in.BackupName, api.StorageVolumeBackupPost{Name: in.NewName})
	if err != nil {
		return toolError[*api.Operation]("storage_volume_backup_rename", err)
	}
	return s.waitResult("storage_volume_backup_rename", op, in.WaitTimeoutSeconds)
}

// StorageVolumeBackupExportInput writes a custom-volume backup to a local path on the MCP host.
type StorageVolumeBackupExportInput struct {
	Pool       string `json:"pool" jsonschema:"the storage pool"`
	Project    string `json:"project,omitempty" jsonschema:"project (defaults to configured default)"`
	VolumeName string `json:"volume_name" jsonschema:"the volume name"`
	BackupName string `json:"backup_name" jsonschema:"the backup name"`
	DestPath   string `json:"dest_path" jsonschema:"local path on the MCP server host for the exported backup"`
}

type StorageVolumeBackupExportOutput struct {
	Path string `json:"path" jsonschema:"path of the exported artifact on the MCP server host"`
	Size int64  `json:"size" jsonschema:"size of the exported artifact in bytes"`
}

func (s *Server) storageVolumeBackupExport(ctx context.Context, req *mcp.CallToolRequest, in StorageVolumeBackupExportInput) (*mcp.CallToolResult, StorageVolumeBackupExportOutput, error) {
	if in.Pool == "" || in.VolumeName == "" || in.BackupName == "" || in.DestPath == "" {
		return toolError[StorageVolumeBackupExportOutput]("storage_volume_backup_export", errRequired("pool, volume_name, backup_name, and dest_path"))
	}
	file, err := os.Create(in.DestPath)
	if err != nil {
		return toolError[StorageVolumeBackupExportOutput]("storage_volume_backup_export", err)
	}
	defer file.Close()
	server, ok := s.projectServer(in.Project).(interface {
		GetStoragePoolVolumeBackupFile(string, string, string, *incusclient.BackupFileRequest) (*incusclient.BackupFileResponse, error)
	})
	if !ok {
		return toolError[StorageVolumeBackupExportOutput]("storage_volume_backup_export", errUnsupported("volume backup export on the configured Incus client"))
	}
	resp, err := server.GetStoragePoolVolumeBackupFile(in.Pool, in.VolumeName, in.BackupName, &incusclient.BackupFileRequest{BackupFile: file})
	if err != nil {
		return toolError[StorageVolumeBackupExportOutput]("storage_volume_backup_export", err)
	}
	return result(StorageVolumeBackupExportOutput{Path: in.DestPath, Size: resp.Size})
}

const (
	maxISOArtifactBytes        = 64 << 20
	isoPrimaryDescriptorOffset = 16 * 2048
)

// StorageVolumeImportISOInput streams a bounded, base64-encoded ISO artifact
// into a custom ISO volume. The bridge decodes it into a temporary file only
// for the lifetime of the authenticated Incus upload.
type StorageVolumeImportISOInput struct {
	Pool               string `json:"pool" jsonschema:"the storage pool"`
	Project            string `json:"project,omitempty" jsonschema:"project (defaults to configured default)"`
	Name               string `json:"name" jsonschema:"name for the custom ISO volume"`
	ArtifactBase64     string `json:"artifact_base64" jsonschema:"standard-base64 ISO artifact, limited to 64 MiB decoded"`
	WaitTimeoutSeconds int    `json:"wait_timeout_seconds,omitempty" jsonschema:"wait timeout override in seconds"`
}

func (s *Server) storageVolumeImportISO(ctx context.Context, req *mcp.CallToolRequest, in StorageVolumeImportISOInput) (*mcp.CallToolResult, *api.Operation, error) {
	if in.Pool == "" || in.Name == "" || in.ArtifactBase64 == "" {
		return toolError[*api.Operation]("storage_volume_import_iso", errRequired("pool, name, and artifact_base64"))
	}
	file, err := decodeISOArtifact(in.ArtifactBase64)
	if err != nil {
		return toolError[*api.Operation]("storage_volume_import_iso", err)
	}
	defer func() {
		_ = file.Close()
		_ = os.Remove(file.Name())
	}()
	if err := validateISOArtifact(file); err != nil {
		return toolError[*api.Operation]("storage_volume_import_iso", err)
	}
	op, err := s.projectServer(in.Project).CreateStoragePoolVolumeFromISO(in.Pool, incusclient.StorageVolumeBackupArgs{BackupFile: file, Name: in.Name})
	if err != nil {
		return toolError[*api.Operation]("storage_volume_import_iso", err)
	}
	return s.waitResult("storage_volume_import_iso", op, in.WaitTimeoutSeconds)
}

func decodeISOArtifact(encoded string) (*os.File, error) {
	if len(encoded) > base64.StdEncoding.EncodedLen(maxISOArtifactBytes) {
		return nil, fmt.Errorf("artifact_base64 exceeds the %d-byte decoded ISO limit", maxISOArtifactBytes)
	}
	file, err := os.CreateTemp("", "incus-os-mcp-iso-*.iso")
	if err != nil {
		return nil, fmt.Errorf("create temporary ISO artifact: %w", err)
	}
	fail := func(err error) (*os.File, error) {
		_ = file.Close()
		_ = os.Remove(file.Name())
		return nil, err
	}
	written, err := io.Copy(file, io.LimitReader(base64.NewDecoder(base64.StdEncoding, strings.NewReader(encoded)), maxISOArtifactBytes+1))
	if err != nil {
		return fail(fmt.Errorf("decode artifact_base64: %w", err))
	}
	if written > maxISOArtifactBytes {
		return fail(fmt.Errorf("artifact_base64 exceeds the %d-byte decoded ISO limit", maxISOArtifactBytes))
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return fail(fmt.Errorf("rewind temporary ISO artifact: %w", err))
	}
	return file, nil
}

func validateISOArtifact(file *os.File) error {
	const descriptorLength = 7
	descriptor := make([]byte, descriptorLength)
	if _, err := file.ReadAt(descriptor, isoPrimaryDescriptorOffset); err != nil {
		return fmt.Errorf("read ISO9660 primary volume descriptor: %w", err)
	}
	if !bytes.Equal(descriptor, []byte{1, 'C', 'D', '0', '0', '1', 1}) {
		return fmt.Errorf("artifact_base64 is not an ISO9660 image")
	}
	_, err := file.Seek(0, io.SeekStart)
	return err
}

// StorageBucketUpdateInput updates a bucket with ETag-safe replacement semantics.
type StorageBucketUpdateInput struct {
	Pool        string            `json:"pool" jsonschema:"the storage pool"`
	Project     string            `json:"project,omitempty" jsonschema:"project (defaults to configured default)"`
	Name        string            `json:"name" jsonschema:"the bucket name"`
	Description string            `json:"description,omitempty" jsonschema:"bucket description"`
	Config      map[string]string `json:"config,omitempty" jsonschema:"bucket configuration"`
}

func (s *Server) storageBucketUpdate(ctx context.Context, req *mcp.CallToolRequest, in StorageBucketUpdateInput) (*mcp.CallToolResult, string, error) {
	if in.Pool == "" || in.Name == "" {
		return toolError[string]("storage_bucket_update", errRequired("pool and name"))
	}
	server := s.projectServer(in.Project)
	_, etag, err := server.GetStoragePoolBucket(in.Pool, in.Name)
	if err != nil {
		return toolError[string]("storage_bucket_update", err)
	}
	if err := server.UpdateStoragePoolBucket(in.Pool, in.Name, api.StorageBucketPut{Description: in.Description, Config: in.Config}, etag); err != nil {
		return toolError[string]("storage_bucket_update", err)
	}
	return result("bucket updated: " + in.Name)
}
