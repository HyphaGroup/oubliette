// Package opencode provides the OpenCode agent runtime.
//
// events.go - SSE event type constants
//
// This file contains:
// - Event type constants for OpenCode SSE events
// - Message part type constants
// - Session status constants
//
// These constants map to the event types emitted by the OpenCode
// server's /event SSE endpoint.

package opencode

// OpenCode event types mapped from bus events
const (
	// Session events
	EventSessionCreated = "session.created"
	EventSessionUpdated = "session.updated"
	EventSessionDeleted = "session.deleted"
	EventSessionStatus  = "session.status"
	EventSessionIdle    = "session.idle"
	EventSessionError   = "session.error"

	// Message events
	EventMessageUpdated     = "message.updated"
	EventMessageRemoved     = "message.removed"
	EventMessagePartUpdated = "message.part.updated"
	EventMessagePartRemoved = "message.part.removed"

	// Permission events
	EventPermissionAsked   = "permission.asked"
	EventPermissionReplied = "permission.replied"

	// Server events
	EventServerConnected = "server.connected"
	EventServerHeartbeat = "server.heartbeat"
	EventServerDisposed  = "global.disposed"
)

// Part types in OpenCode messages
const (
	PartTypeText           = "text"
	PartTypeToolInvocation = "tool-invocation"
	PartTypeToolResult     = "tool-result"
	PartTypeFile           = "file"
	PartTypeCompaction     = "compaction"
	PartTypeSubtask        = "subtask"
)

// Session status values
const (
	StatusActive  = "active"
	StatusPending = "pending"
	StatusIdle    = "idle"
)
