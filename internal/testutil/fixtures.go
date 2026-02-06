package testutil

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/HyphaGroup/oubliette/internal/project"
	"github.com/HyphaGroup/oubliette/internal/session"
)

// ptr returns a pointer to the given value.
func ptr[T any](v T) *T {
	return &v
}

// ProjectOption is a function that modifies a Project for testing.
type ProjectOption func(*project.Project)

// NewTestProject creates a test project with sensible defaults.
func NewTestProject(t *testing.T, opts ...ProjectOption) *project.Project {
	t.Helper()

	id := uuid.New().String()
	now := time.Now()
	p := &project.Project{
		ID:                 id,
		Name:               "test-project-" + t.Name(),
		Description:        "Test project for " + t.Name(),
		DefaultWorkspaceID: uuid.New().String(),
		CreatedAt:          now,
		GitHubToken:        "", // Never set in fixtures
		RemoteURL:          "",
		ImageName:          "oubliette-" + id[:8],
		HasDockerfile:      false,
		ContainerID:        "",
		ContainerStatus:    "",
		RecursionConfig:    nil,
	}

	for _, opt := range opts {
		opt(p)
	}

	return p
}

// WithProjectID sets a specific ID for the test project.
func WithProjectID(id string) ProjectOption {
	return func(p *project.Project) {
		p.ID = id
		p.ImageName = "oubliette-" + id[:8]
	}
}

// WithProjectName sets a custom name for the test project.
func WithProjectName(name string) ProjectOption {
	return func(p *project.Project) {
		p.Name = name
	}
}

// WithRemoteURL sets a remote URL for the test project.
func WithRemoteURL(url string) ProjectOption {
	return func(p *project.Project) {
		p.RemoteURL = url
	}
}

// WithDockerfile marks the project as having a custom Dockerfile.
func WithDockerfile() ProjectOption {
	return func(p *project.Project) {
		p.HasDockerfile = true
	}
}

// WithContainerRunning sets the project's container as running.
func WithContainerRunning(containerID string) ProjectOption {
	return func(p *project.Project) {
		p.ContainerID = containerID
		p.ContainerStatus = "running"
	}
}

// WithRecursionConfig sets recursion limits for the test project.
func WithRecursionConfig(maxDepth, maxAgents int, maxCost float64) ProjectOption {
	return func(p *project.Project) {
		p.RecursionConfig = &project.RecursionConfig{
			MaxDepth:   ptr(maxDepth),
			MaxAgents:  ptr(maxAgents),
			MaxCostUSD: ptr(maxCost),
		}
	}
}

// SessionOption is a function that modifies a Session for testing.
type SessionOption func(*session.Session)

// NewTestSession creates a test session with sensible defaults.
func NewTestSession(t *testing.T, projectID string, opts ...SessionOption) *session.Session {
	t.Helper()

	now := time.Now()
	s := &session.Session{
		SessionID:        uuid.New().String(),
		ProjectID:        projectID,
		WorkspaceID:      "default",
		ContainerID:      "",
		Status:           session.StatusActive,
		CreatedAt:        now,
		UpdatedAt:        now,
		RuntimeSessionID: uuid.New().String(),
		Model:            "claude-sonnet-4-20250514",
		AutonomyLevel:    "medium",
		ReasoningLevel:   "off",
		Turns:            []session.Turn{},
		TotalCost:        session.Cost{},
		ParentSessionID:  nil,
		ChildSessions:    nil,
		Depth:            0,
		ExplorationID:    "",
		TaskContext:      nil,
		ToolsAllowed:     nil,
	}

	for _, opt := range opts {
		opt(s)
	}

	return s
}

// WithSessionID sets a specific session ID.
func WithSessionID(id string) SessionOption {
	return func(s *session.Session) {
		s.SessionID = id
	}
}

// WithWorkspaceID sets the workspace ID.
func WithWorkspaceID(id string) SessionOption {
	return func(s *session.Session) {
		s.WorkspaceID = id
	}
}

// WithSessionStatus sets the session status.
func WithSessionStatus(status session.Status) SessionOption {
	return func(s *session.Session) {
		s.Status = status
	}
}

// WithModel sets the model for the session.
func WithModel(model string) SessionOption {
	return func(s *session.Session) {
		s.Model = model
	}
}

// WithParentSession sets the parent session ID and increments depth.
func WithParentSession(parentID string, depth int) SessionOption {
	return func(s *session.Session) {
		s.ParentSessionID = &parentID
		s.Depth = depth
	}
}

// WithTurns adds turns to the session.
func WithTurns(turns ...session.Turn) SessionOption {
	return func(s *session.Session) {
		s.Turns = turns
	}
}

// WithContainerID sets the container ID for the session.
func WithContainerID(containerID string) SessionOption {
	return func(s *session.Session) {
		s.ContainerID = containerID
	}
}

// NewTestTurn creates a test turn with sensible defaults.
func NewTestTurn(t *testing.T, turnNumber int, prompt, output string) session.Turn {
	t.Helper()

	now := time.Now()
	return session.Turn{
		TurnNumber:  turnNumber,
		Prompt:      prompt,
		StartedAt:   now.Add(-time.Second),
		CompletedAt: now,
		Output: session.TurnOutput{
			Text:     output,
			ExitCode: 0,
		},
		Cost: session.Cost{
			InputTokens:  100,
			OutputTokens: 200,
		},
	}
}

// WorkspaceMetadataOption is a function that modifies WorkspaceMetadata for testing.
type WorkspaceMetadataOption func(*project.WorkspaceMetadata)

// NewTestWorkspace creates test workspace metadata with sensible defaults.
func NewTestWorkspace(t *testing.T, opts ...WorkspaceMetadataOption) *project.WorkspaceMetadata {
	t.Helper()

	now := time.Now()
	w := &project.WorkspaceMetadata{
		ID:            uuid.New().String(),
		CreatedAt:     now,
		LastSessionAt: time.Time{},
		ExternalID:    "",
		Source:        "test",
	}

	for _, opt := range opts {
		opt(w)
	}

	return w
}

// WithWorkspaceUUID sets a specific workspace ID.
func WithWorkspaceUUID(id string) WorkspaceMetadataOption {
	return func(w *project.WorkspaceMetadata) {
		w.ID = id
	}
}

// WithExternalID sets the external ID for workspace tracking.
func WithExternalID(externalID, source string) WorkspaceMetadataOption {
	return func(w *project.WorkspaceMetadata) {
		w.ExternalID = externalID
		w.Source = source
	}
}

// WithLastSession sets the last session timestamp.
func WithLastSession(t time.Time) WorkspaceMetadataOption {
	return func(w *project.WorkspaceMetadata) {
		w.LastSessionAt = t
	}
}
