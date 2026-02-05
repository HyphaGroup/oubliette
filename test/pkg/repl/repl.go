package repl

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/HyphaGroup/oubliette/test/pkg/client"
)

// REPL implements an interactive read-eval-print loop for the MCP client
type REPL struct {
	client  *client.MCPClient
	reader  *bufio.Reader
	history []string
}

// NewREPL creates a new REPL instance
func NewREPL(mcpClient *client.MCPClient) *REPL {
	return &REPL{
		client:  mcpClient,
		reader:  bufio.NewReader(os.Stdin),
		history: []string{},
	}
}

// Run starts the interactive REPL loop
func (r *REPL) Run() error {
	fmt.Println("🤖 Oubliette MCP Test Client - Interactive Mode")
	fmt.Println("Type 'help' for available commands, 'exit' to quit")
	fmt.Println()

	for {
		fmt.Print("> ")

		// Read input
		input, err := r.reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				fmt.Println("\nBye!")
				return nil
			}
			return fmt.Errorf("failed to read input: %w", err)
		}

		// Trim whitespace
		input = strings.TrimSpace(input)

		// Skip empty lines
		if input == "" {
			continue
		}

		// Add to history
		r.history = append(r.history, input)

		// Parse and execute command
		if err := r.executeCommand(input); err != nil {
			fmt.Printf("❌ Error: %v\n\n", err)
		}
	}
}

// executeCommand parses and executes a REPL command
func (r *REPL) executeCommand(input string) error {
	parts := strings.Fields(input)
	if len(parts) == 0 {
		return nil
	}

	command := parts[0]
	args := parts[1:]

	switch command {
	case "help", "?":
		return r.cmdHelp()

	case "list", "list_tools", "tools":
		return r.cmdListTools()

	case "describe":
		if len(args) == 0 {
			return fmt.Errorf("usage: describe <tool_name>")
		}
		return r.cmdDescribeTool(args[0])

	case "invoke", "call":
		if len(args) == 0 {
			return fmt.Errorf("usage: invoke <tool_name> [json_params]")
		}
		toolName := args[0]
		var params map[string]interface{}
		if len(args) > 1 {
			// Join remaining args as JSON
			jsonStr := strings.Join(args[1:], " ")
			if err := json.Unmarshal([]byte(jsonStr), &params); err != nil {
				return fmt.Errorf("invalid JSON parameters: %w", err)
			}
		}
		return r.cmdInvokeTool(toolName, params)

	case "history":
		return r.cmdHistory()

	case "clear":
		return r.cmdClear()

	case "exit", "quit":
		fmt.Println("Bye!")
		os.Exit(0)
		return nil

	default:
		// Try to parse as shorthand tool invocation
		// Format: tool_name {"key":"value"}
		if strings.Contains(command, "_") {
			var params map[string]interface{}
			if len(args) > 0 {
				jsonStr := strings.Join(args, " ")
				if err := json.Unmarshal([]byte(jsonStr), &params); err != nil {
					return fmt.Errorf("invalid JSON parameters: %w", err)
				}
			}
			return r.cmdInvokeTool(command, params)
		}

		return fmt.Errorf("unknown command: %s (type 'help' for available commands)", command)
	}
}

// cmdHelp displays help information
func (r *REPL) cmdHelp() error {
	fmt.Println("Available commands:")
	fmt.Println("  help, ?                           Show this help message")
	fmt.Println("  list, tools                       List all available MCP tools")
	fmt.Println("  describe <tool>                   Show detailed info about a tool")
	fmt.Println("  invoke <tool> [json]              Invoke a tool with optional JSON params")
	fmt.Println("  history                           Show command history")
	fmt.Println("  clear                             Clear the screen")
	fmt.Println("  exit, quit                        Exit the REPL")
	fmt.Println()
	fmt.Println("Shorthand tool invocation:")
	fmt.Println("  project_list               Invoke tool without params")
	fmt.Println("  project_create {\"name\":\"test\"}  Invoke tool with params")
	fmt.Println()
	return nil
}

// cmdListTools lists all available tools
func (r *REPL) cmdListTools() error {
	tools, err := r.client.ListTools()
	if err != nil {
		return err
	}

	fmt.Printf("Available tools (%d):\n", len(tools))
	for _, tool := range tools {
		fmt.Printf("  • %s\n", tool.Name)
		if tool.Description != "" {
			fmt.Printf("    %s\n", tool.Description)
		}
	}
	fmt.Println()
	return nil
}

// cmdDescribeTool shows detailed information about a tool
func (r *REPL) cmdDescribeTool(toolName string) error {
	tools, err := r.client.ListTools()
	if err != nil {
		return err
	}

	// Find the tool
	var found *client.Tool
	for _, tool := range tools {
		if tool.Name == toolName {
			found = &tool
			break
		}
	}

	if found == nil {
		return fmt.Errorf("tool not found: %s", toolName)
	}

	fmt.Printf("Tool: %s\n", found.Name)
	if found.Description != "" {
		fmt.Printf("Description: %s\n", found.Description)
	}

	if found.InputSchema != nil {
		schemaJSON, err := json.MarshalIndent(found.InputSchema, "  ", "  ")
		if err == nil {
			fmt.Printf("Input Schema:\n  %s\n", string(schemaJSON))
		}
	}
	fmt.Println()
	return nil
}

// cmdInvokeTool invokes a tool with the given parameters
func (r *REPL) cmdInvokeTool(toolName string, params map[string]interface{}) error {
	fmt.Printf("Invoking: %s\n", toolName)
	if params != nil {
		paramsJSON, _ := json.MarshalIndent(params, "  ", "  ")
		fmt.Printf("Params:\n  %s\n", string(paramsJSON))
	}
	fmt.Println()

	result, err := r.client.InvokeTool(toolName, params)
	if err != nil {
		return err
	}

	if result.IsError {
		fmt.Println("❌ Tool returned error:")
	} else {
		fmt.Println("✓ Tool succeeded:")
	}

	content := result.GetToolContent()
	fmt.Println(content)
	fmt.Println()

	return nil
}

// cmdHistory displays command history
func (r *REPL) cmdHistory() error {
	if len(r.history) == 0 {
		fmt.Println("No command history")
		return nil
	}

	fmt.Println("Command history:")
	for i, cmd := range r.history {
		fmt.Printf("%3d  %s\n", i+1, cmd)
	}
	fmt.Println()
	return nil
}

// cmdClear clears the screen
func (r *REPL) cmdClear() error {
	fmt.Print("\033[H\033[2J")
	return nil
}
