package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"sync"

	"github.com/HyphaGroup/oubliette/internal/auth"
	mcp_sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// ToolHandler is a function that handles a tool call
type ToolHandler func(ctx context.Context, arguments json.RawMessage) (any, error)

type ctxKeyCallToolRequest struct{}

// WithCallToolRequest stores the MCP CallToolRequest in context
func WithCallToolRequest(ctx context.Context, req *mcp_sdk.CallToolRequest) context.Context {
	return context.WithValue(ctx, ctxKeyCallToolRequest{}, req)
}

// CallToolRequestFromContext retrieves the MCP CallToolRequest from context
func CallToolRequestFromContext(ctx context.Context) *mcp_sdk.CallToolRequest {
	if req, ok := ctx.Value(ctxKeyCallToolRequest{}).(*mcp_sdk.CallToolRequest); ok {
		return req
	}
	return nil
}

// ToolDef defines a tool with all metadata
type ToolDef struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Target      ToolTarget     `json:"target,omitempty"`
	Access      ToolAccess     `json:"access,omitempty"`
	InputSchema map[string]any `json:"inputSchema,omitempty"`
}

// Registry stores tool definitions and handlers
type Registry struct {
	mu       sync.RWMutex
	tools    map[string]*ToolDef
	handlers map[string]ToolHandler
	order    []string // preserve registration order
}

// NewRegistry creates a new tool registry
func NewRegistry() *Registry {
	return &Registry{
		tools:    make(map[string]*ToolDef),
		handlers: make(map[string]ToolHandler),
		order:    make([]string, 0),
	}
}

// Register adds a tool with its handler to the registry
// Schema is auto-generated from P type parameter if not provided in def
func Register[P any](r *Registry, def ToolDef, handler func(ctx context.Context, req *mcp_sdk.CallToolRequest, params P) (*mcp_sdk.CallToolResult, any, error)) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Auto-generate schema from P if not provided
	if def.InputSchema == nil {
		def.InputSchema = GenerateSchema[P]()
	}

	r.tools[def.Name] = &def
	r.handlers[def.Name] = wrapHandler(handler)
	r.order = append(r.order, def.Name)
}

// GetTool returns a tool definition by name
func (r *Registry) GetTool(name string) (*ToolDef, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	tool, ok := r.tools[name]
	return tool, ok
}

// GetAllTools returns all tool definitions in registration order
func (r *Registry) GetAllTools() []*ToolDef {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tools := make([]*ToolDef, 0, len(r.order))
	for _, name := range r.order {
		if tool, ok := r.tools[name]; ok {
			tools = append(tools, tool)
		}
	}
	return tools
}

// GetToolsForScope returns tools available for the given token scope
// Note: For project-scoped tokens, this returns tools the scope CAN access
// but project-targeted tools will still be checked per-call with the actual project ID
func (r *Registry) GetToolsForScope(tokenScope string) []*ToolDef {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tools := make([]*ToolDef, 0)
	for _, name := range r.order {
		def := r.tools[name]
		// For listing, use empty projectID - project-scoped tokens see project tools
		// but enforcement happens at call time
		if IsToolAllowed(def, tokenScope, "") || (auth.IsProjectScope(tokenScope) && def.Target == TargetProject) {
			tools = append(tools, def)
		}
	}
	return tools
}

// IsToolAllowed checks if a specific tool name is allowed for a token scope
// projectID is used for project-scoped permission checks
func (r *Registry) IsToolAllowedWithProject(toolName, tokenScope, projectID string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if def, ok := r.tools[toolName]; ok {
		return IsToolAllowed(def, tokenScope, projectID)
	}
	return false
}

// IsToolAllowed checks if a specific tool name is allowed for a token scope (legacy, no project check)
func (r *Registry) IsToolAllowed(toolName, tokenScope string) bool {
	return r.IsToolAllowedWithProject(toolName, tokenScope, "")
}

// CallTool executes a tool by name with JSON arguments
func (r *Registry) CallTool(ctx context.Context, name string, args json.RawMessage) (any, error) {
	r.mu.RLock()
	handler, ok := r.handlers[name]
	r.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("unknown tool: %s", name)
	}
	return handler(ctx, args)
}

// CallToolWithMap executes a tool by name with map arguments (for socket dispatch)
func (r *Registry) CallToolWithMap(ctx context.Context, name string, args map[string]any) (map[string]any, error) {
	argsJSON, err := json.Marshal(args)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal arguments: %w", err)
	}

	result, err := r.CallTool(ctx, name, argsJSON)
	if err != nil {
		return map[string]any{
			"content": []map[string]any{
				{"type": "text", "text": err.Error()},
			},
			"isError": true,
		}, nil
	}

	// If result is already a map, return it
	if m, ok := result.(map[string]any); ok {
		return m, nil
	}

	// If result is *CallToolResult, convert it
	if ctr, ok := result.(*mcp_sdk.CallToolResult); ok {
		content := make([]map[string]any, 0, len(ctr.Content))
		for _, c := range ctr.Content {
			if tc, ok := c.(*mcp_sdk.TextContent); ok {
				content = append(content, map[string]any{
					"type": "text",
					"text": tc.Text,
				})
			}
		}
		return map[string]any{
			"content": content,
			"isError": ctr.IsError,
		}, nil
	}

	// Otherwise marshal to JSON text content
	data, err := json.Marshal(result)
	if err != nil {
		return map[string]any{
			"content": []map[string]any{
				{"type": "text", "text": err.Error()},
			},
			"isError": true,
		}, nil
	}

	return map[string]any{
		"content": []map[string]any{
			{"type": "text", "text": string(data)},
		},
		"isError": false,
	}, nil
}

// RegisterWithMCPServer registers all tools with an MCP SDK server
func (r *Registry) RegisterWithMCPServer(server *mcp_sdk.Server) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, name := range r.order {
		def := r.tools[name]
		handler := r.handlers[name]

		tool := &mcp_sdk.Tool{
			Name:        name,
			Description: def.Description,
			InputSchema: def.InputSchema,
		}

		// Capture handler in closure properly
		h := handler
		sdkHandler := func(ctx context.Context, req *mcp_sdk.CallToolRequest) (*mcp_sdk.CallToolResult, error) {
			ctx = WithCallToolRequest(ctx, req)
			var args json.RawMessage
			if req.Params != nil {
				args = req.Params.Arguments
			}
			result, err := h(ctx, args)
			if err != nil {
				return NewErrorResult(err.Error()), nil
			}
			if ctr, ok := result.(*mcp_sdk.CallToolResult); ok {
				return ctr, nil
			}
			data, err := json.Marshal(result)
			if err != nil {
				return NewErrorResult(err.Error()), nil
			}
			return NewTextResult(string(data)), nil
		}

		server.AddTool(tool, sdkHandler)
	}
}

// wrapHandler wraps a typed handler into a ToolHandler
func wrapHandler[P any](handler func(ctx context.Context, req *mcp_sdk.CallToolRequest, params P) (*mcp_sdk.CallToolResult, any, error)) ToolHandler {
	return func(ctx context.Context, args json.RawMessage) (any, error) {
		var params P
		if len(args) > 0 {
			if err := json.Unmarshal(args, &params); err != nil {
				return nil, fmt.Errorf("invalid parameters: %w", err)
			}
		}

		req := CallToolRequestFromContext(ctx)
		if req == nil {
			req = &mcp_sdk.CallToolRequest{
				Params: &mcp_sdk.CallToolParamsRaw{
					Arguments: args,
				},
			}
		}

		result, data, err := handler(ctx, req, params)
		if err != nil {
			return nil, err
		}

		if result != nil && result.IsError {
			errMsg := "tool execution failed"
			if len(result.Content) > 0 {
				if textContent, ok := result.Content[0].(*mcp_sdk.TextContent); ok {
					errMsg = textContent.Text
				}
			}
			return nil, fmt.Errorf("%s", errMsg)
		}

		if data != nil {
			return data, nil
		}
		return result, nil
	}
}

// GenerateSchema creates a JSON Schema from a Go type using reflection
func GenerateSchema[P any]() map[string]any {
	var p P
	t := reflect.TypeOf(p)

	if t == nil {
		return map[string]any{"type": "object"}
	}

	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	if t.Kind() != reflect.Struct {
		return map[string]any{"type": "object"}
	}

	props := make(map[string]any)
	schema := map[string]any{
		"type":       "object",
		"properties": props,
	}

	var required []string

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)

		if !field.IsExported() {
			continue
		}

		jsonTag := field.Tag.Get("json")
		if jsonTag == "-" {
			continue
		}

		name := field.Name
		omitempty := false
		if jsonTag != "" {
			parts := strings.Split(jsonTag, ",")
			if parts[0] != "" {
				name = parts[0]
			}
			for _, opt := range parts[1:] {
				if opt == "omitempty" {
					omitempty = true
				}
			}
		}

		propSchema := typeToSchema(field.Type)

		if desc := field.Tag.Get("description"); desc != "" {
			propSchema["description"] = desc
		}

		props[name] = propSchema

		if !omitempty {
			required = append(required, name)
		}
	}

	if len(required) > 0 {
		schema["required"] = required
	}

	return schema
}

// typeToSchema converts a Go type to JSON Schema type
func typeToSchema(t reflect.Type) map[string]any {
	if t.Kind() == reflect.Ptr {
		return typeToSchema(t.Elem())
	}

	switch t.Kind() {
	case reflect.String:
		return map[string]any{"type": "string"}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return map[string]any{"type": "integer"}
	case reflect.Float32, reflect.Float64:
		return map[string]any{"type": "number"}
	case reflect.Bool:
		return map[string]any{"type": "boolean"}
	case reflect.Slice, reflect.Array:
		return map[string]any{
			"type":  "array",
			"items": typeToSchema(t.Elem()),
		}
	case reflect.Map:
		return map[string]any{
			"type":                 "object",
			"additionalProperties": typeToSchema(t.Elem()),
		}
	case reflect.Struct:
		props := make(map[string]any)
		schema := map[string]any{
			"type":       "object",
			"properties": props,
		}
		var required []string

		for i := 0; i < t.NumField(); i++ {
			field := t.Field(i)
			if !field.IsExported() {
				continue
			}

			jsonTag := field.Tag.Get("json")
			if jsonTag == "-" {
				continue
			}

			name := field.Name
			omitempty := false
			if jsonTag != "" {
				parts := strings.Split(jsonTag, ",")
				if parts[0] != "" {
					name = parts[0]
				}
				for _, opt := range parts[1:] {
					if opt == "omitempty" {
						omitempty = true
					}
				}
			}

			propSchema := typeToSchema(field.Type)
			if desc := field.Tag.Get("description"); desc != "" {
				propSchema["description"] = desc
			}
			props[name] = propSchema

			if !omitempty {
				required = append(required, name)
			}
		}

		if len(required) > 0 {
			schema["required"] = required
		}
		return schema
	case reflect.Interface:
		return map[string]any{}
	default:
		return map[string]any{"type": "string"}
	}
}
