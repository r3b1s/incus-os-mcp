package server

import (
	"context"
	"net/url"
	"os"
	"path/filepath"
	"regexp"

	incusclient "github.com/lxc/incus/v7/client"
	"github.com/lxc/incus/v7/shared/api"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

var sha256Re = regexp.MustCompile(`^[0-9a-fA-F]{64}$`)

// ---- inventory ----

// ImageListInput lists images.
type ImageListInput struct {
	Project     string `json:"project,omitempty" jsonschema:"project to list images in (defaults to configured default)"`
	AllProjects bool   `json:"all_projects,omitempty" jsonschema:"list images in all projects"`
	Filter      string `json:"filter,omitempty" jsonschema:"optional API filter expression"`
}

func (s *Server) imageList(ctx context.Context, req *mcp.CallToolRequest, in ImageListInput) (*mcp.CallToolResult, ListOutput[api.Image], error) {
	if err := validateProjectScope(in.Project, in.AllProjects); err != nil {
		return toolError[ListOutput[api.Image]]("image_list", err)
	}
	var (
		images []api.Image
		err    error
	)
	if in.Filter != "" {
		if in.AllProjects {
			images, err = s.client.Server.GetImagesAllProjectsWithFilter([]string{in.Filter})
		} else {
			images, err = s.projectServer(in.Project).GetImagesWithFilter([]string{in.Filter})
		}
	} else if in.AllProjects {
		images, err = s.client.Server.GetImagesAllProjects()
	} else {
		images, err = s.projectServer(in.Project).GetImages()
	}
	if err != nil {
		return toolError[ListOutput[api.Image]]("image_list", err)
	}
	return result(ListOutput[api.Image]{Items: images})
}

// ImageGetInput fetches an image.
type ImageGetInput struct {
	Fingerprint string `json:"fingerprint" jsonschema:"the image fingerprint"`
	Project     string `json:"project,omitempty" jsonschema:"project the image is in (defaults to configured default)"`
}

func (s *Server) imageGet(ctx context.Context, req *mcp.CallToolRequest, in ImageGetInput) (*mcp.CallToolResult, *api.Image, error) {
	if in.Fingerprint == "" {
		return toolError[*api.Image]("image_get", errRequired("fingerprint"))
	}
	img, _, err := s.projectServer(in.Project).GetImage(in.Fingerprint)
	if err != nil {
		return toolError[*api.Image]("image_get", err)
	}
	return result(img)
}

// ---- import ----

// ImageImportInput imports an image.
type ImageImportInput struct {
	Project string `json:"project,omitempty" jsonschema:"project to import the image into (defaults to configured default)"`

	// URL is the image URL. Sha256 is required when importing from a URL.
	URL string `json:"url,omitempty" jsonschema:"image URL; sha256 is required with a URL import"`
	// Sha256 is the expected image checksum (64 hex chars).
	Sha256 string `json:"sha256,omitempty" jsonschema:"expected sha256 of the image (64 hex chars); required for URL imports, pre-validated before the API call"`

	// LocalFile is a path on the MCP server host to upload as the image.
	// For split images (meta + rootfs), pass both LocalFile (the metadata
	// tarball) and RootfsFile (the rootfs). For single-file images (plain
	// tarballs, qcow2/raw), pass just LocalFile.
	LocalFile string `json:"local_file,omitempty" jsonschema:"local file path to upload as the image (metadata tarball for split images; the image itself for single-file images)"`
	// RootfsFile is the rootfs file for split images.
	RootfsFile string `json:"rootfs_file,omitempty" jsonschema:"rootfs file path for split images (qcow2/raw for VMs, squashfs/tar.xz for containers)"`

	// Public makes the image public.
	Public bool `json:"public,omitempty" jsonschema:"make the image public"`
	// Aliases are aliases to add to the image.
	Aliases []string `json:"aliases,omitempty" jsonschema:"aliases to add"`

	WaitTimeoutSeconds int `json:"wait_timeout_seconds,omitempty" jsonschema:"wait timeout override in seconds"`
}

func (s *Server) imageImport(ctx context.Context, req *mcp.CallToolRequest, in ImageImportInput) (*mcp.CallToolResult, *api.Operation, error) {
	if in.URL == "" && in.LocalFile == "" {
		return toolError[*api.Operation]("image_import", errRequired("url (with sha256 or local_file)"))
	}

	if in.URL != "" {
		// Pre-validate URL scheme and sha256 form (CONTEXT.md: the server
		// pre-validates before the API call; Incus remains authoritative).
		u, err := url.Parse(in.URL)
		if err != nil || (u.Scheme != "https" && u.Scheme != "http") {
			return toolError[*api.Operation]("image_import", errRequired("url must be an absolute http(s) URL"))
		}
		if !sha256Re.MatchString(in.Sha256) {
			return toolError[*api.Operation]("image_import", errRequired("sha256 must be 64 hex characters for URL imports"))
		}

		aliases := make([]api.ImageAlias, 0, len(in.Aliases))
		for _, name := range in.Aliases {
			aliases = append(aliases, api.ImageAlias{Name: name})
		}
		post := api.ImagesPost{
			Source: &api.ImagesPostSource{
				Type:        "url",
				URL:         in.URL,
				Fingerprint: in.Sha256,
				Mode:        "pull",
			},
			ImagePut: api.ImagePut{Public: in.Public},
			Aliases:  aliases,
		}
		post.Filename = "image"

		op, err := s.projectServer(in.Project).CreateImage(post, nil)
		if err != nil {
			return toolError[*api.Operation]("image_import", err)
		}
		return s.waitResult("image_import", op, in.WaitTimeoutSeconds)
	}

	// Local file upload.
	f, err := os.Open(in.LocalFile)
	if err != nil {
		return toolError[*api.Operation]("image_import", err)
	}
	defer f.Close()

	args := &incusclient.ImageCreateArgs{
		MetaFile: f,
		MetaName: filepath.Base(in.LocalFile),
	}

	// Split image: also send the rootfs file.
	if in.RootfsFile != "" {
		rf, err := os.Open(in.RootfsFile)
		if err != nil {
			return toolError[*api.Operation]("image_import", err)
		}
		defer rf.Close()
		args.RootfsFile = rf
		args.RootfsName = filepath.Base(in.RootfsFile)
	}

	// The server-side sha256 verification happens after upload; we pass the
	// file(s) to the client's CreateImage, which handles single-file and
	// split-image uploads.
	aliases := make([]api.ImageAlias, 0, len(in.Aliases))
	for _, name := range in.Aliases {
		aliases = append(aliases, api.ImageAlias{Name: name})
	}
	op, err := s.projectServer(in.Project).CreateImage(api.ImagesPost{
		ImagePut: api.ImagePut{Public: in.Public},
		Filename: filepath.Base(in.LocalFile),
		Aliases:  aliases,
	}, args)
	if err != nil {
		return toolError[*api.Operation]("image_import", err)
	}
	return s.waitResult("image_import", op, in.WaitTimeoutSeconds)
}

// ---- lifecycle ----

// ImageDeleteInput deletes an image.
type ImageDeleteInput struct {
	Fingerprint string `json:"fingerprint" jsonschema:"the image fingerprint"`
	Project     string `json:"project,omitempty" jsonschema:"project the image is in (defaults to configured default)"`

	WaitTimeoutSeconds int `json:"wait_timeout_seconds,omitempty" jsonschema:"wait timeout override in seconds"`
}

func (s *Server) imageDelete(ctx context.Context, req *mcp.CallToolRequest, in ImageDeleteInput) (*mcp.CallToolResult, *api.Operation, error) {
	if in.Fingerprint == "" {
		return toolError[*api.Operation]("image_delete", errRequired("fingerprint"))
	}
	op, err := s.projectServer(in.Project).DeleteImage(in.Fingerprint)
	if err != nil {
		return toolError[*api.Operation]("image_delete", err)
	}
	return s.waitResult("image_delete", op, in.WaitTimeoutSeconds)
}

// ImageRefreshInput refreshes an auto-update image.
type ImageRefreshInput struct {
	Fingerprint string `json:"fingerprint" jsonschema:"the image fingerprint"`
	Project     string `json:"project,omitempty" jsonschema:"project the image is in (defaults to configured default)"`

	WaitTimeoutSeconds int `json:"wait_timeout_seconds,omitempty" jsonschema:"wait timeout override in seconds"`
}

func (s *Server) imageRefresh(ctx context.Context, req *mcp.CallToolRequest, in ImageRefreshInput) (*mcp.CallToolResult, *api.Operation, error) {
	if in.Fingerprint == "" {
		return toolError[*api.Operation]("image_refresh", errRequired("fingerprint"))
	}
	op, err := s.projectServer(in.Project).RefreshImage(in.Fingerprint)
	if err != nil {
		return toolError[*api.Operation]("image_refresh", err)
	}
	return s.waitResult("image_refresh", op, in.WaitTimeoutSeconds)
}

// ImageCopyInput copies an image to another project or server.
type ImageCopyInput struct {
	Fingerprint   string `json:"fingerprint" jsonschema:"the image fingerprint"`
	Project       string `json:"project,omitempty" jsonschema:"source project (defaults to configured default)"`
	TargetProject string `json:"target_project,omitempty" jsonschema:"target project on the same server"`

	WaitTimeoutSeconds int `json:"wait_timeout_seconds,omitempty" jsonschema:"wait timeout override in seconds"`
}

func (s *Server) imageCopy(ctx context.Context, req *mcp.CallToolRequest, in ImageCopyInput) (*mcp.CallToolResult, *api.Operation, error) {
	if in.Fingerprint == "" {
		return toolError[*api.Operation]("image_copy", errRequired("fingerprint"))
	}
	// Copy within the same server: create in the destination project while
	// naming the source project when one was explicitly requested.
	post := api.ImagesPost{
		Source: &api.ImagesPostSource{
			Type:        "image",
			Fingerprint: in.Fingerprint,
			Mode:        "pull",
			Project:     in.Project,
		},
	}
	target := s.projectServer(in.TargetProject)
	if in.TargetProject == "" {
		target = s.projectServer(in.Project)
		post.Source.Project = ""
	}
	op, err := target.CreateImage(post, nil)
	if err != nil {
		return toolError[*api.Operation]("image_copy", err)
	}
	return s.waitResult("image_copy", op, in.WaitTimeoutSeconds)
}

// ImageExportInput exports an image artifact.
type ImageExportInput struct {
	Fingerprint string `json:"fingerprint" jsonschema:"the image fingerprint"`
	Project     string `json:"project,omitempty" jsonschema:"project the image is in (defaults to configured default)"`
	DestPath    string `json:"dest_path,omitempty" jsonschema:"local path to write the exported image to (optional; defaults to the staging dir)"`
}

// ImageExportOutput reports the export location.
type ImageExportOutput struct {
	Path string `json:"path" jsonschema:"path of the exported image on the MCP server host"`
	Size int64  `json:"size" jsonschema:"size of the exported image in bytes"`
}

func (s *Server) imageExport(ctx context.Context, req *mcp.CallToolRequest, in ImageExportInput) (*mcp.CallToolResult, ImageExportOutput, error) {
	if in.Fingerprint == "" {
		return toolError[ImageExportOutput]("image_export", errRequired("fingerprint"))
	}
	base := in.DestPath
	if base == "" {
		base = os.TempDir() + "/incus-os-mcp-export-" + in.Fingerprint[:12]
	}

	// Split images export two files (meta + rootfs); single-file images only
	// the meta. We always provide both file handles; the client writes only
	// the parts the image has.
	metaPath := base + ".meta.tar"
	rootfsPath := base + ".rootfs"

	mf, err := os.Create(metaPath)
	if err != nil {
		return toolError[ImageExportOutput]("image_export", err)
	}
	defer mf.Close()

	rf, err := os.Create(rootfsPath)
	if err != nil {
		return toolError[ImageExportOutput]("image_export", err)
	}
	defer rf.Close()

	resp, err := s.projectServer(in.Project).GetImageFile(in.Fingerprint, incusclient.ImageFileRequest{
		MetaFile:   mf,
		RootfsFile: rf,
	})
	if err != nil {
		return toolError[ImageExportOutput]("image_export", err)
	}
	size := resp.MetaSize + resp.RootfsSize

	out := ImageExportOutput{Path: metaPath, Size: size}
	if resp.RootfsSize > 0 {
		out.Path = base + " (meta + rootfs)"
	}
	return result(out)
}

// ---- aliases ----

// ImageAliasCreateInput creates an alias.
type ImageAliasCreateInput struct {
	Project     string `json:"project,omitempty" jsonschema:"project (defaults to configured default)"`
	Name        string `json:"name" jsonschema:"the alias name"`
	Fingerprint string `json:"fingerprint" jsonschema:"the image fingerprint the alias points to"`
}

func (s *Server) imageAliasCreate(ctx context.Context, req *mcp.CallToolRequest, in ImageAliasCreateInput) (*mcp.CallToolResult, string, error) {
	if in.Name == "" || in.Fingerprint == "" {
		return toolError[string]("image_alias_create", errRequired("name and fingerprint"))
	}
	err := s.projectServer(in.Project).CreateImageAlias(api.ImageAliasesPost{
		ImageAliasesEntry: api.ImageAliasesEntry{
			Name: in.Name,
			ImageAliasesEntryPut: api.ImageAliasesEntryPut{
				Target: in.Fingerprint,
			},
		},
	})
	if err != nil {
		return toolError[string]("image_alias_create", err)
	}
	return result("alias created: " + in.Name)
}

// ImageAliasDeleteInput deletes an alias.
type ImageAliasDeleteInput struct {
	Project string `json:"project,omitempty" jsonschema:"project (defaults to configured default)"`
	Name    string `json:"name" jsonschema:"the alias name"`
}

func (s *Server) imageAliasDelete(ctx context.Context, req *mcp.CallToolRequest, in ImageAliasDeleteInput) (*mcp.CallToolResult, string, error) {
	if in.Name == "" {
		return toolError[string]("image_alias_delete", errRequired("name"))
	}
	if err := s.projectServer(in.Project).DeleteImageAlias(in.Name); err != nil {
		return toolError[string]("image_alias_delete", err)
	}
	return result("alias deleted: " + in.Name)
}

// ImageAliasListInput lists aliases.
type ImageAliasListInput struct {
	Project string `json:"project,omitempty" jsonschema:"project (defaults to configured default)"`
}

func (s *Server) imageAliasList(ctx context.Context, req *mcp.CallToolRequest, in ImageAliasListInput) (*mcp.CallToolResult, ListOutput[api.ImageAliasesEntry], error) {
	aliases, err := s.projectServer(in.Project).GetImageAliases()
	if err != nil {
		return toolError[ListOutput[api.ImageAliasesEntry]]("image_alias_list", err)
	}
	return result(ListOutput[api.ImageAliasesEntry]{Items: aliases})
}

// ---- registration ----

func (s *Server) registerImageTools() {
	addTool(s, "image_list", "List images (optionally all projects or filtered).", s.imageList)
	addTool(s, "image_get", "Fetch a single image's metadata.", s.imageGet)
	addTool(s, "image_import", "Import an image from a URL (sha256 required, pre-validated) or from a local file upload.", s.imageImport)
	addTool(s, "image_delete", "Delete an image.", s.imageDelete)
	addTool(s, "image_refresh", "Refresh an auto-update image.", s.imageRefresh)
	addTool(s, "image_copy", "Copy an image to another project or server.", s.imageCopy)
	addTool(s, "image_export", "Export an image artifact to the MCP server host.", s.imageExport)
	addTool(s, "image_alias_create", "Create an image alias.", s.imageAliasCreate)
	addTool(s, "image_alias_delete", "Delete an image alias.", s.imageAliasDelete)
	addTool(s, "image_alias_list", "List image aliases.", s.imageAliasList)
}
