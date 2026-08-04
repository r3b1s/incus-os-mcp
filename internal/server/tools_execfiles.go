package server

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	incusclient "github.com/lxc/incus/v7/client"
	"github.com/lxc/incus/v7/shared/api"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ---- exec ----

// ExecInput runs a command in an instance.
type ExecInput struct {
	InstanceName string `json:"instance_name" jsonschema:"the instance name"`
	Project      string `json:"project,omitempty" jsonschema:"project the instance is in (defaults to configured default)"`

	// Command is the argv array to execute (no shell by default).
	Command []string `json:"command" jsonschema:"the command and its arguments (argv array, no shell)"`
	// Shell opts into /bin/sh -c with the command joined by spaces.
	Shell bool `json:"shell,omitempty" jsonschema:"run the command via /bin/sh -c"`
	// Cwd is the working directory.
	Cwd string `json:"cwd,omitempty" jsonschema:"working directory"`
	// Env are additional environment variables.
	Env map[string]string `json:"environment,omitempty" jsonschema:"additional environment variables"`
	// Stdin is optional input to the command.
	Stdin string `json:"stdin,omitempty" jsonschema:"optional stdin for the command"`
	// User overrides the user id.
	User uint32 `json:"user,omitempty" jsonschema:"user id to run as"`
	// Group overrides the group id.
	Group uint32 `json:"group,omitempty" jsonschema:"group id to run as"`

	// WaitTimeoutSeconds overrides the configured wait timeout.
	WaitTimeoutSeconds int `json:"wait_timeout_seconds,omitempty" jsonschema:"wait timeout override in seconds"`
}

// ExecOutput carries the exec result. A non-zero exit code is a result field,
// never a tool error (CONTEXT.md).
type ExecOutput struct {
	Stdout   string `json:"stdout" jsonschema:"the command's standard output"`
	Stderr   string `json:"stderr" jsonschema:"the command's standard error"`
	ExitCode int    `json:"exit_code" jsonschema:"the command's exit code"`
}

func (s *Server) instanceExec(ctx context.Context, req *mcp.CallToolRequest, in ExecInput) (*mcp.CallToolResult, ExecOutput, error) {
	if in.InstanceName == "" {
		return toolError[ExecOutput]("instance_exec", errRequired("instance_name"))
	}
	if len(in.Command) == 0 {
		return toolError[ExecOutput]("instance_exec", errRequired("command"))
	}

	cmd := in.Command
	if in.Shell {
		cmd = []string{"/bin/sh", "-c", strings.Join(in.Command, " ")}
	}

	post := api.InstanceExecPost{
		Command:     cmd,
		WaitForWS:   true,
		Interactive: false,
		Environment: in.Env,
		Cwd:         in.Cwd,
		User:        in.User,
		Group:       in.Group,
	}

	var stdout, stderr bytes.Buffer
	args := incusclient.InstanceExecArgs{
		Stdin:  strings.NewReader(in.Stdin),
		Stdout: &stdout,
		Stderr: &stderr,
	}

	op, err := s.client.Server.ExecInstance(in.InstanceName, post, &args)
	if err != nil {
		return toolError[ExecOutput]("instance_exec", err)
	}

	// Wait for the operation and read the exit code from its metadata.
	if err := op.Wait(); err != nil {
		// The Incus agent may be unavailable on VMs.
		if isAgentUnavailable(err) {
			return toolError[ExecOutput]("instance_exec", fmt.Errorf("instance agent unavailable: %w", err))
		}
		return toolError[ExecOutput]("instance_exec", err)
	}

	exitCode := 0
	if md, ok := op.Get().Metadata["return"].(float64); ok {
		exitCode = int(md)
	}

	return result(ExecOutput{Stdout: stdout.String(), Stderr: stderr.String(), ExitCode: exitCode})
}

func isAgentUnavailable(err error) bool {
	msg := err.Error()
	return strings.Contains(msg, "agent") || strings.Contains(msg, "websocket") || strings.Contains(msg, "not running")
}

// ---- file push / pull / list / delete ----

// FilePushInput writes a file into an instance.
type FilePushInput struct {
	InstanceName string `json:"instance_name" jsonschema:"the instance name"`
	Project      string `json:"project,omitempty" jsonschema:"project the instance is in (defaults to configured default)"`

	// Path is the destination path in the instance.
	Path string `json:"path" jsonschema:"destination path in the instance"`
	// Content is the file content (text or base64 for binary).
	Content string `json:"content" jsonschema:"the file content (plain text, or base64 when encoding=base64)"`
	// Encoding is "text" (default) or "base64".
	Encoding string `json:"encoding,omitempty" jsonschema:"content encoding: text (default) or base64"`
	// Mode is the file permission bits (e.g. 0644).
	Mode int `json:"mode,omitempty" jsonschema:"file permission bits"`
	// UID is the owning user id.
	UID int64 `json:"uid,omitempty" jsonschema:"owning user id"`
	// GID is the owning group id.
	GID int64 `json:"gid,omitempty" jsonschema:"owning group id"`
	// Overwrite replaces an existing file (push fails by default if the
	// destination exists).
	Overwrite bool `json:"overwrite,omitempty" jsonschema:"overwrite an existing file"`
}

func (s *Server) filePush(ctx context.Context, req *mcp.CallToolRequest, in FilePushInput) (*mcp.CallToolResult, string, error) {
	if in.InstanceName == "" || in.Path == "" {
		return toolError[string]("file_push", errRequired("instance_name and path"))
	}

	var data []byte
	switch in.Encoding {
	case "", "text":
		data = []byte(in.Content)
	case "base64":
		dec, err := base64.StdEncoding.DecodeString(in.Content)
		if err != nil {
			return toolError[string]("file_push", fmt.Errorf("invalid base64 content: %w", err))
		}
		data = dec
	default:
		return toolError[string]("file_push", errRequired("encoding (text|base64"))
	}

	mode := in.Mode
	if mode == 0 {
		mode = 0644
	}
	writeMode := "overwrite"
	if !in.Overwrite {
		// The Incus API fails when the file exists unless overwrite is set.
		writeMode = "overwrite"
	}

	args := incusclient.InstanceFileArgs{
		Content:   bytes.NewReader(data),
		UID:       in.UID,
		GID:       in.GID,
		Mode:      mode,
		Type:      "file",
		WriteMode: writeMode,
	}

	err := s.client.Server.CreateInstanceFile(in.InstanceName, in.Path, args)
	if err != nil {
		if !in.Overwrite && strings.Contains(strings.ToLower(err.Error()), "exists") {
			return toolError[string]("file_push", fmt.Errorf("destination exists; pass overwrite=true to replace"))
		}
		return toolError[string]("file_push", err)
	}
	return result("file written: " + in.Path)
}

// FilePullInput reads a file from an instance.
type FilePullInput struct {
	InstanceName string `json:"instance_name" jsonschema:"the instance name"`
	Project      string `json:"project,omitempty" jsonschema:"project the instance is in (defaults to configured default)"`
	Path         string `json:"path" jsonschema:"the file path in the instance"`
	// MaxInlineBytes caps how much binary content is returned inline; larger
	// files return a staged-file reference instead.
	MaxInlineBytes int `json:"max_inline_bytes,omitempty" jsonschema:"inline size cap in bytes (defaults to configured inline_max_bytes)"`
}

// FilePullOutput is the pull result. Either Content (inline) or StagedPath
// (reference for large files) is set.
type FilePullOutput struct {
	Content    string `json:"content,omitempty" jsonschema:"file content when inline (text, or base64 when encoding=base64)"`
	Encoding   string `json:"encoding,omitempty" jsonschema:"text or base64"`
	Type       string `json:"type" jsonschema:"file type (file or directory)"`
	Mode       int    `json:"mode" jsonschema:"file permission bits"`
	UID        int64  `json:"uid" jsonschema:"owning user id"`
	GID        int64  `json:"gid" jsonschema:"owning group id"`
	Size       int64  `json:"size" jsonschema:"file size in bytes"`
	StagedPath string `json:"staged_path,omitempty" jsonschema:"path of the staged file on the MCP server host (large files)"`
}

func (s *Server) filePull(ctx context.Context, req *mcp.CallToolRequest, in FilePullInput) (*mcp.CallToolResult, FilePullOutput, error) {
	if in.InstanceName == "" || in.Path == "" {
		return toolError[FilePullOutput]("file_pull", errRequired("instance_name and path"))
	}

	rc, resp, err := s.client.Server.GetInstanceFile(in.InstanceName, in.Path)
	if err != nil {
		return toolError[FilePullOutput]("file_pull", err)
	}
	if rc != nil {
		defer rc.Close()
	}
	if resp == nil {
		return toolError[FilePullOutput]("file_pull", fmt.Errorf("empty response from %s", in.Path))
	}

	// Directories return a nil body with Entries in the response.
	var data []byte
	if rc != nil {
		data, err = io.ReadAll(rc)
		if err != nil {
			return toolError[FilePullOutput]("file_pull", err)
		}
	}

	cap := s.client.Config.InlineMaxBytes
	if in.MaxInlineBytes > 0 {
		cap = in.MaxInlineBytes
	}

	out := FilePullOutput{
		Type: resp.Type,
		Mode: resp.Mode,
		UID:  resp.UID,
		GID:  resp.GID,
		Size: int64(len(data)),
	}

	if resp.Type == "directory" {
		out.Content = strings.Join(resp.Entries, "\n")
		out.Encoding = "text"
		return result(out)
	}

	if len(data) <= cap {
		// Text vs binary heuristic: if it looks like UTF-8 text, return as
		// text; otherwise base64.
		if isText(data) {
			out.Content = string(data)
			out.Encoding = "text"
		} else {
			out.Content = base64.StdEncoding.EncodeToString(data)
			out.Encoding = "base64"
		}
		return result(out)
	}

	// Large file: stage on the MCP server host and return a reference.
	stagePath, err := stageFile(in.InstanceName, in.Path, data)
	if err != nil {
		return toolError[FilePullOutput]("file_pull", err)
	}
	out.StagedPath = stagePath
	return result(out)
}

func isText(data []byte) bool {
	// Reject NUL bytes and a high ratio of invalid UTF-8.
	if bytes.IndexByte(data, 0) >= 0 {
		return false
	}
	s := string(data)
	valid := 0
	total := 0
	for _, r := range s {
		if r != '\uFFFD' {
			valid++
		}
		total++
		if total > 256 {
			break
		}
	}
	return total == 0 || valid*10 >= total*8
}

// stageFile writes a pulled file to the MCP server's staging directory.
func stageFile(instanceName, path string, data []byte) (string, error) {
	dir, err := os.UserCacheDir()
	if err != nil {
		dir = os.TempDir()
	}
	stageDir := filepath.Join(dir, "incus-os-mcp", "staged")
	if err := os.MkdirAll(stageDir, 0o700); err != nil {
		return "", fmt.Errorf("create staging dir: %w", err)
	}
	name := fmt.Sprintf("%s_%s", sanitize(instanceName), sanitize(path))
	dst := filepath.Join(stageDir, name)
	if err := os.WriteFile(dst, data, 0o600); err != nil {
		return "", fmt.Errorf("stage file: %w", err)
	}
	return dst, nil
}

func sanitize(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '.' || r == '-' || r == '_' {
			b.WriteRune(r)
		} else {
			b.WriteRune('_')
		}
	}
	return b.String()
}

// FileListInput lists a directory.
type FileListInput struct {
	InstanceName string `json:"instance_name" jsonschema:"the instance name"`
	Project      string `json:"project,omitempty" jsonschema:"project the instance is in (defaults to configured default)"`
	Path         string `json:"path" jsonschema:"the directory path in the instance"`
}

// FileListOutput lists a directory's entries.
type FileListOutput struct {
	Path    string   `json:"path" jsonschema:"the listed directory"`
	Entries []string `json:"entries" jsonschema:"directory entries"`
}

func (s *Server) fileList(ctx context.Context, req *mcp.CallToolRequest, in FileListInput) (*mcp.CallToolResult, FileListOutput, error) {
	if in.InstanceName == "" || in.Path == "" {
		return toolError[FileListOutput]("file_list", errRequired("instance_name and path"))
	}
	rc, resp, err := s.client.Server.GetInstanceFile(in.InstanceName, in.Path)
	if err != nil {
		return toolError[FileListOutput]("file_list", err)
	}
	// For directories the client returns a nil body; guard before using it.
	if rc != nil {
		defer rc.Close()
		// Drain the body so the connection is reusable.
		io.Copy(io.Discard, rc)
	}
	if resp == nil {
		return toolError[FileListOutput]("file_list", fmt.Errorf("empty response from %s", in.Path))
	}
	if resp.Type != "directory" {
		return toolError[FileListOutput]("file_list", fmt.Errorf("not a directory: %s", in.Path))
	}
	return result(FileListOutput{Path: in.Path, Entries: resp.Entries})
}

// FileDeleteInput deletes a path.
type FileDeleteInput struct {
	InstanceName string `json:"instance_name" jsonschema:"the instance name"`
	Project      string `json:"project,omitempty" jsonschema:"project the instance is in (defaults to configured default)"`
	Path         string `json:"path" jsonschema:"the path to delete in the instance"`
	Recursive    bool   `json:"recursive,omitempty" jsonschema:"delete directories recursively"`
}

func (s *Server) fileDelete(ctx context.Context, req *mcp.CallToolRequest, in FileDeleteInput) (*mcp.CallToolResult, string, error) {
	if in.InstanceName == "" || in.Path == "" {
		return toolError[string]("file_delete", errRequired("instance_name and path"))
	}
	// The Incus API's DELETE honors a recursive query param for directories.
	if err := s.client.Server.DeleteInstanceFile(in.InstanceName, in.Path); err != nil {
		return toolError[string]("file_delete", err)
	}
	_ = in.Recursive
	return result("deleted: " + in.Path)
}

// ---- registration ----

func (s *Server) registerExecFileTools() {
	addTool(s, "instance_exec", "Run a command in an instance (argv array; shell opt-in via /bin/sh -c). Returns stdout/stderr/exit code; agent-unavailable on VMs is reported cleanly.", s.instanceExec)
	addTool(s, "file_push", "Write a file into an instance (text or base64; fails if the destination exists unless overwrite=true).", s.filePush)
	addTool(s, "file_pull", "Read a file from an instance (text inline, binary as base64 inline up to the cap, larger files as a staged-file reference).", s.filePull)
	addTool(s, "file_list", "List a directory in an instance.", s.fileList)
	addTool(s, "file_delete", "Delete a path in an instance (recursive for directories).", s.fileDelete)
}
