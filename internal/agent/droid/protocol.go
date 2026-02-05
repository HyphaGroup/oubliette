// Package droid provides the Factory Droid agent runtime.
//
// protocol.go - JSON-RPC 2.0 communication layer
//
// This file contains:
// - JSON-RPC request/response types for the Factory API
// - Request builders for session management (initialize, message, cancel)
// - Request ID generation
//
// Droid uses the stream-jsonrpc protocol for bidirectional communication
// over stdin/stdout. Messages are newline-delimited JSON.

package droid

import "fmt"

// JSONRPCRequest represents a Factory API JSON-RPC request
type JSONRPCRequest struct {
	JSONRPC           string      `json:"jsonrpc"`
	FactoryAPIVersion string      `json:"factoryApiVersion"`
	Type              string      `json:"type"`
	Method            string      `json:"method"`
	Params            interface{} `json:"params,omitempty"`
	ID                string      `json:"id"`
}

// JSONRPCResponse represents a Factory API JSON-RPC response
type JSONRPCResponse struct {
	JSONRPC           string      `json:"jsonrpc"`
	FactoryAPIVersion string      `json:"factoryApiVersion"`
	Type              string      `json:"type"`
	Result            interface{} `json:"result,omitempty"`
	Error             *RPCError   `json:"error,omitempty"`
	ID                string      `json:"id"`
}

// RPCError represents a JSON-RPC 2.0 error
type RPCError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// Standard JSON-RPC methods for stream-jsonrpc protocol
const (
	MethodInitializeSession = "droid.initialize_session"
	MethodAddUserMessage    = "droid.add_user_message"
	MethodInterruptSession  = "droid.interrupt_session"
)

// InitializeSessionParams contains parameters for droid.initialize_session
type InitializeSessionParams struct {
	MachineID string `json:"machineId"`
	Cwd       string `json:"cwd"`
	Prompt    string `json:"prompt,omitempty"`
}

// AddUserMessageParams contains parameters for droid.add_user_message
type AddUserMessageParams struct {
	Text string `json:"text"`
}

var requestIDCounter int64

func nextRequestID() string {
	requestIDCounter++
	return fmt.Sprintf("%d", requestIDCounter)
}

// NewInitializeSessionRequest creates a JSON-RPC request to initialize a session
func NewInitializeSessionRequest(prompt, cwd, machineID string) *JSONRPCRequest {
	return &JSONRPCRequest{
		JSONRPC:           "2.0",
		FactoryAPIVersion: "1.0.0",
		Type:              "request",
		Method:            MethodInitializeSession,
		Params: InitializeSessionParams{
			MachineID: machineID,
			Cwd:       cwd,
			Prompt:    prompt,
		},
		ID: nextRequestID(),
	}
}

// NewUserMessageRequest creates a JSON-RPC request to send a user message
func NewUserMessageRequest(message string, id interface{}) *JSONRPCRequest {
	return &JSONRPCRequest{
		JSONRPC:           "2.0",
		FactoryAPIVersion: "1.0.0",
		Type:              "request",
		Method:            MethodAddUserMessage,
		Params:            AddUserMessageParams{Text: message},
		ID:                fmt.Sprintf("%v", id),
	}
}

// NewCancelRequest creates a JSON-RPC request to interrupt/cancel execution
func NewCancelRequest(id interface{}) *JSONRPCRequest {
	return &JSONRPCRequest{
		JSONRPC:           "2.0",
		FactoryAPIVersion: "1.0.0",
		Type:              "request",
		Method:            MethodInterruptSession,
		ID:                fmt.Sprintf("%v", id),
	}
}
