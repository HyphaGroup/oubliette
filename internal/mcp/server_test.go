package mcp

import (
	"testing"
)

func TestHasAPICredentials_NoCredentials(t *testing.T) {
	s := &Server{credentials: nil}
	if s.HasAPICredentials() {
		t.Error("HasAPICredentials() should return false with nil credentials")
	}
}
