// Package server builds the MCP server: it registers the typed tools (one per
// capability group) on a shared incus client and serves them over streamable
// HTTP.
package server

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"incus-os-mcp/internal/config"
	"incus-os-mcp/internal/incus"
)

// Server is the assembled MCP server.
type Server struct {
	cfg    *config.Config
	client *incus.Client
	server *mcp.Server
	logger *slog.Logger
}

// clientOrError returns the incus client, or a clean error when the target
// connection failed at startup (degraded mode: tools report the failure
// instead of panicking).
func (s *Server) clientOrError() (*incus.Client, error) {
	if s.client == nil {
		return nil, fmt.Errorf("target connection unavailable: the MCP server failed to connect at startup (check target.url and credentials, then restart)")
	}
	return s.client, nil
}

// New assembles the MCP server with all tools registered.
func New(cfg *config.Config, client *incus.Client, logger *slog.Logger) *Server {
	s := &Server{
		cfg:    cfg,
		client: client,
		logger: logger,
	}

	impl := &mcp.Implementation{
		Name:    "incus-os-mcp",
		Version: version(),
	}
	s.server = mcp.NewServer(impl, &mcp.ServerOptions{
		Logger: logger,
	})

	s.registerTools()
	return s
}

// Handler returns the HTTP handler serving the streamable MCP transport.
func (s *Server) Handler() http.Handler {
	return mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server {
		return s.server
	}, &mcp.StreamableHTTPOptions{
		Stateless:    true,
		Logger:       s.logger,
		JSONResponse: true,
	})
}

// registerTools registers every tool group.
func (s *Server) registerTools() {
	s.registerServerTools()
	s.registerInstanceTools()
	s.registerExecFileTools()
	s.registerImageTools()
	s.registerStorageTools()
	s.registerNetworkTools()
	s.registerConfigTools()
	s.registerSystemTools()
}

// addTool is a thin wrapper over mcp.AddTool keeping call sites terse.
func addTool[In, Out any](s *Server, name, description string, h mcp.ToolHandlerFor[In, Out]) {
	mcp.AddTool(s.server, &mcp.Tool{
		Name:        name,
		Description: description,
	}, h)
}

// toolError converts an incus error into a tool error result per the
// error-mapping rules (403 → clean permission error; everything else wrapped).
// The type parameter carries the handler's output type so callers keep the
// 3-value signature the SDK requires.
func toolError[Out any](op string, err error) (*mcp.CallToolResult, Out, error) {
	var zero Out
	mapped := incus.MapError(op, err)
	return nil, zero, fmt.Errorf("%w", mapped)
}

// result returns a successful tool result with the given output.
func result[Out any](out Out) (*mcp.CallToolResult, Out, error) {
	return nil, out, nil
}

func version() string {
	return "0.1.0"
}
