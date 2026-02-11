package mcp

import (
	"context"
	"encoding/json"
	"testing"

	mcp_sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestGenerateSchema_String(t *testing.T) {
	type Params struct {
		Name string `json:"name"`
	}
	schema := GenerateSchema[Params]()

	props := schema["properties"].(map[string]any)
	nameProp := props["name"].(map[string]any)
	if nameProp["type"] != "string" {
		t.Errorf("expected type string, got %v", nameProp["type"])
	}

	required := schema["required"].([]string)
	if len(required) != 1 || required[0] != "name" {
		t.Errorf("expected required=[name], got %v", required)
	}
}

func TestGenerateSchema_Integer(t *testing.T) {
	type Params struct {
		Limit int `json:"limit"`
	}
	schema := GenerateSchema[Params]()

	props := schema["properties"].(map[string]any)
	limitProp := props["limit"].(map[string]any)
	if limitProp["type"] != "integer" {
		t.Errorf("expected type integer, got %v", limitProp["type"])
	}
}

func TestGenerateSchema_Boolean(t *testing.T) {
	type Params struct {
		Force bool `json:"force"`
	}
	schema := GenerateSchema[Params]()

	props := schema["properties"].(map[string]any)
	forceProp := props["force"].(map[string]any)
	if forceProp["type"] != "boolean" {
		t.Errorf("expected type boolean, got %v", forceProp["type"])
	}
}

func TestGenerateSchema_Array(t *testing.T) {
	type Params struct {
		Tags []string `json:"tags"`
	}
	schema := GenerateSchema[Params]()

	props := schema["properties"].(map[string]any)
	tagsProp := props["tags"].(map[string]any)
	if tagsProp["type"] != "array" {
		t.Errorf("expected type array, got %v", tagsProp["type"])
	}
	items := tagsProp["items"].(map[string]any)
	if items["type"] != "string" {
		t.Errorf("expected items type string, got %v", items["type"])
	}
}

func TestGenerateSchema_NestedStruct(t *testing.T) {
	type Config struct {
		Value string `json:"value"`
	}
	type Params struct {
		Config Config `json:"config"`
	}
	schema := GenerateSchema[Params]()

	props := schema["properties"].(map[string]any)
	configProp := props["config"].(map[string]any)
	if configProp["type"] != "object" {
		t.Errorf("expected type object, got %v", configProp["type"])
	}
	nestedProps := configProp["properties"].(map[string]any)
	if _, ok := nestedProps["value"]; !ok {
		t.Error("expected nested property 'value'")
	}
}

func TestGenerateSchema_Omitempty(t *testing.T) {
	type Params struct {
		Name        string `json:"name"`
		Description string `json:"description,omitempty"`
	}
	schema := GenerateSchema[Params]()

	required := schema["required"].([]string)
	if len(required) != 1 || required[0] != "name" {
		t.Errorf("expected required=[name], got %v", required)
	}
}

func TestGenerateSchema_Description(t *testing.T) {
	type Params struct {
		Name string `json:"name" description:"The project name"`
	}
	schema := GenerateSchema[Params]()

	props := schema["properties"].(map[string]any)
	nameProp := props["name"].(map[string]any)
	if nameProp["description"] != "The project name" {
		t.Errorf("expected description 'The project name', got %v", nameProp["description"])
	}
}

func TestGenerateSchema_SkipUnexported(t *testing.T) {
	type Params struct {
		Name   string `json:"name"`
		hidden string //nolint:unused // intentionally unexported to test schema generation
	}
	schema := GenerateSchema[Params]()

	props := schema["properties"].(map[string]any)
	if _, ok := props["hidden"]; ok {
		t.Error("unexported field should not be in schema")
	}
}

func TestGenerateSchema_SkipJsonIgnore(t *testing.T) {
	type Params struct {
		Name   string `json:"name"`
		Secret string `json:"-"`
	}
	schema := GenerateSchema[Params]()

	props := schema["properties"].(map[string]any)
	if _, ok := props["Secret"]; ok {
		t.Error("json:\"-\" field should not be in schema")
	}
}

func TestRegistry_RegisterAndGetAllTools(t *testing.T) {
	r := NewRegistry()

	type Params struct {
		Name string `json:"name"`
	}

	handler := func(ctx context.Context, req *mcp_sdk.CallToolRequest, params Params) (*mcp_sdk.CallToolResult, any, error) {
		return NewTextResult("ok"), nil, nil
	}

	Register(r, ToolDef{Name: "tool_a", Description: "Tool A", Target: TargetGlobal, Access: AccessRead}, handler)
	Register(r, ToolDef{Name: "tool_b", Description: "Tool B", Target: TargetGlobal, Access: AccessWrite}, handler)

	tools := r.GetAllTools()
	if len(tools) != 2 {
		t.Errorf("expected 2 tools, got %d", len(tools))
	}
	if tools[0].Name != "tool_a" || tools[1].Name != "tool_b" {
		t.Error("tools not in registration order")
	}
}

func TestRegistry_GetToolsForScope(t *testing.T) {
	r := NewRegistry()

	type Params struct{}
	handler := func(ctx context.Context, req *mcp_sdk.CallToolRequest, params Params) (*mcp_sdk.CallToolResult, any, error) {
		return NewTextResult("ok"), nil, nil
	}

	Register(r, ToolDef{Name: "read_tool", Target: TargetGlobal, Access: AccessRead}, handler)
	Register(r, ToolDef{Name: "write_tool", Target: TargetGlobal, Access: AccessWrite}, handler)
	Register(r, ToolDef{Name: "admin_tool", Target: TargetGlobal, Access: AccessAdmin}, handler)

	// Admin sees all
	adminTools := r.GetToolsForScope("admin")
	if len(adminTools) != 3 {
		t.Errorf("admin should see 3 tools, got %d", len(adminTools))
	}

	// Admin:ro sees read-only global tools
	readTools := r.GetToolsForScope("admin:ro")
	if len(readTools) != 1 || readTools[0].Name != "read_tool" {
		t.Errorf("admin:ro should see 1 read tool, got %v", toolNames(readTools))
	}

	// Project scope sees only read global tools (not write/admin globals)
	projectTools := r.GetToolsForScope("project:abc123")
	if len(projectTools) != 1 || projectTools[0].Name != "read_tool" {
		t.Errorf("project scope should see 1 read tool, got %d: %v", len(projectTools), toolNames(projectTools))
	}
}

func TestRegistry_CallTool(t *testing.T) {
	r := NewRegistry()

	type Params struct {
		Name string `json:"name"`
	}

	handler := func(ctx context.Context, req *mcp_sdk.CallToolRequest, params Params) (*mcp_sdk.CallToolResult, any, error) {
		return NewTextResult("Hello " + params.Name), nil, nil
	}

	Register(r, ToolDef{Name: "greet", Target: TargetGlobal, Access: AccessRead}, handler)

	args, _ := json.Marshal(map[string]string{"name": "World"})
	result, err := r.CallTool(context.Background(), "greet", args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ctr, ok := result.(*mcp_sdk.CallToolResult)
	if !ok {
		t.Fatalf("expected CallToolResult, got %T", result)
	}

	text := ctr.Content[0].(*mcp_sdk.TextContent).Text
	if text != "Hello World" {
		t.Errorf("expected 'Hello World', got %q", text)
	}
}

func TestRegistry_CallTool_UnknownTool(t *testing.T) {
	r := NewRegistry()

	_, err := r.CallTool(context.Background(), "unknown", nil)
	if err == nil || err.Error() != "unknown tool: unknown" {
		t.Errorf("expected 'unknown tool' error, got %v", err)
	}
}

func TestRegistry_IsToolAllowed(t *testing.T) {
	r := NewRegistry()

	type Params struct{}
	handler := func(ctx context.Context, req *mcp_sdk.CallToolRequest, params Params) (*mcp_sdk.CallToolResult, any, error) {
		return NewTextResult("ok"), nil, nil
	}

	Register(r, ToolDef{Name: "admin_only", Target: TargetGlobal, Access: AccessAdmin}, handler)

	if !r.IsToolAllowed("admin_only", "admin") {
		t.Error("admin should be allowed admin_only")
	}
	if r.IsToolAllowed("admin_only", "admin:ro") {
		t.Error("admin:ro should not be allowed admin_only")
	}
	if r.IsToolAllowed("nonexistent", "admin") {
		t.Error("nonexistent tool should return false")
	}
}

func toolNames(tools []*ToolDef) []string {
	names := make([]string, len(tools))
	for i, t := range tools {
		names[i] = t.Name
	}
	return names
}
