// oubliette-relay runs inside containers and pairs upstream (Oubliette) connections
// with downstream (droid) connections, then pipes bytes between them.
//
// Protocol:
// - Downstream (droid): "OUBLIETTE-DOWNSTREAM {project_id}\n"
// - Upstream (Oubliette): "OUBLIETTE-UPSTREAM {session_id} {project_id} {depth}\n"
//
// Relay pairs connections FIFO (first downstream with first upstream).
// No JSON parsing - just header validation and byte copying.
package main

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"sync"
	"time"
)

const (
	socketPath     = "/mcp/relay.sock"
	pairingTimeout = 60 * time.Second
)

type connectionType int

const (
	connTypeUnknown connectionType = iota
	connTypeUpstream
	connTypeDownstream
)

type pendingConn struct {
	conn      net.Conn
	connType  connectionType
	arrivedAt time.Time
}

type Relay struct {
	projectID string

	// FIFO queues for unpaired connections
	pendingUpstream   []*pendingConn
	pendingDownstream []*pendingConn
	mu                sync.Mutex
}

func main() {
	projectID := os.Getenv("OUBLIETTE_PROJECT_ID")
	if projectID == "" {
		fmt.Fprintln(os.Stderr, "OUBLIETTE_PROJECT_ID environment variable required")
		os.Exit(1)
	}

	// Remove stale socket
	_ = os.Remove(socketPath)

	// Ensure /mcp directory exists
	if err := os.MkdirAll("/mcp", 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "failed to create /mcp directory: %v\n", err)
		os.Exit(1)
	}

	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to listen on %s: %v\n", socketPath, err)
		os.Exit(1)
	}

	// Set permissions so droid user can connect
	if err := os.Chmod(socketPath, 0o666); err != nil {
		_ = listener.Close()
		fmt.Fprintf(os.Stderr, "failed to chmod socket: %v\n", err)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stderr, "oubliette-relay: listening on %s for project %s\n", socketPath, projectID)

	relay := &Relay{
		projectID:         projectID,
		pendingUpstream:   make([]*pendingConn, 0),
		pendingDownstream: make([]*pendingConn, 0),
	}

	// Start cleanup goroutine for timed-out pending connections
	go relay.cleanupLoop()

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Fprintf(os.Stderr, "accept error: %v\n", err)
			continue
		}
		go relay.handleConnection(conn)
	}
}

func (r *Relay) handleConnection(conn net.Conn) {
	// Read the header line
	reader := bufio.NewReader(conn)
	headerLine, err := reader.ReadString('\n')
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to read header: %v\n", err)
		_ = conn.Close()
		return
	}

	connType, projectID, err := parseHeader(headerLine)
	if err != nil {
		fmt.Fprintf(os.Stderr, "invalid header: %v\n", err)
		_ = conn.Close()
		return
	}

	// Validate project ID
	if projectID != r.projectID {
		fmt.Fprintf(os.Stderr, "project ID mismatch: got %s, expected %s\n", projectID, r.projectID)
		_ = conn.Close()
		return
	}

	fmt.Fprintf(os.Stderr, "relay: %s connection received\n", connType)

	// Wrap conn with the buffered reader (may have buffered data after header)
	wrappedConn := &bufferedConn{reader: reader, Conn: conn}

	r.mu.Lock()

	// Try to pair with opposite type
	if connType == connTypeUpstream && len(r.pendingDownstream) > 0 {
		// Pair with first pending downstream
		downstream := r.pendingDownstream[0]
		r.pendingDownstream = r.pendingDownstream[1:]
		r.mu.Unlock()

		fmt.Fprintf(os.Stderr, "relay: paired upstream with downstream, starting pipe\n")
		pipe(wrappedConn, downstream.conn)
		return
	}

	if connType == connTypeDownstream && len(r.pendingUpstream) > 0 {
		// Pair with first pending upstream
		upstream := r.pendingUpstream[0]
		r.pendingUpstream = r.pendingUpstream[1:]
		r.mu.Unlock()

		fmt.Fprintf(os.Stderr, "relay: paired downstream with upstream, starting pipe\n")
		pipe(upstream.conn, wrappedConn)
		return
	}

	// No pair available - queue this connection
	pending := &pendingConn{
		conn:      wrappedConn,
		connType:  connType,
		arrivedAt: time.Now(),
	}

	if connType == connTypeUpstream {
		r.pendingUpstream = append(r.pendingUpstream, pending)
		fmt.Fprintf(os.Stderr, "relay: upstream queued (%d waiting)\n", len(r.pendingUpstream))
	} else {
		r.pendingDownstream = append(r.pendingDownstream, pending)
		fmt.Fprintf(os.Stderr, "relay: downstream queued (%d waiting)\n", len(r.pendingDownstream))
	}
	r.mu.Unlock()
}

func (c connectionType) String() string {
	switch c {
	case connTypeUpstream:
		return "upstream"
	case connTypeDownstream:
		return "downstream"
	default:
		return "unknown"
	}
}

func parseHeader(line string) (connectionType, string, error) {
	line = strings.TrimSpace(line)
	parts := strings.Fields(line)

	if len(parts) < 2 {
		return connTypeUnknown, "", fmt.Errorf("invalid header format: %s", line)
	}

	var connType connectionType
	var projectID string

	switch parts[0] {
	case "OUBLIETTE-UPSTREAM":
		connType = connTypeUpstream
		// Format: OUBLIETTE-UPSTREAM session_id project_id depth
		if len(parts) >= 3 {
			projectID = parts[2]
		} else {
			projectID = parts[1]
		}
	case "OUBLIETTE-DOWNSTREAM":
		connType = connTypeDownstream
		// Format: OUBLIETTE-DOWNSTREAM project_id
		projectID = parts[1]
	default:
		return connTypeUnknown, "", fmt.Errorf("unknown header type: %s", parts[0])
	}

	return connType, projectID, nil
}

// pipe copies data bidirectionally between two connections
func pipe(upstream, downstream net.Conn) {
	done := make(chan struct{}, 2)

	// upstream -> downstream
	go func() {
		_, _ = io.Copy(downstream, upstream)
		done <- struct{}{}
	}()

	// downstream -> upstream
	go func() {
		_, _ = io.Copy(upstream, downstream)
		done <- struct{}{}
	}()

	// Wait for either direction to finish
	<-done

	// Close both connections
	_ = upstream.Close()
	_ = downstream.Close()

	// Wait for the other goroutine
	<-done

	fmt.Fprintf(os.Stderr, "relay: pipe closed\n")
}

// cleanupLoop removes pending connections that have timed out
func (r *Relay) cleanupLoop() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		r.mu.Lock()
		now := time.Now()

		// Clean up upstream queue
		newUpstream := make([]*pendingConn, 0, len(r.pendingUpstream))
		for _, pc := range r.pendingUpstream {
			if now.Sub(pc.arrivedAt) > pairingTimeout {
				fmt.Fprintf(os.Stderr, "relay: upstream connection timed out\n")
				_ = pc.conn.Close()
			} else {
				newUpstream = append(newUpstream, pc)
			}
		}
		r.pendingUpstream = newUpstream

		// Clean up downstream queue
		newDownstream := make([]*pendingConn, 0, len(r.pendingDownstream))
		for _, pc := range r.pendingDownstream {
			if now.Sub(pc.arrivedAt) > pairingTimeout {
				fmt.Fprintf(os.Stderr, "relay: downstream connection timed out\n")
				_ = pc.conn.Close()
			} else {
				newDownstream = append(newDownstream, pc)
			}
		}
		r.pendingDownstream = newDownstream

		r.mu.Unlock()
	}
}

// bufferedConn wraps a net.Conn with a bufio.Reader to handle any buffered data
type bufferedConn struct {
	reader *bufio.Reader
	net.Conn
}

func (c *bufferedConn) Read(p []byte) (int, error) {
	return c.reader.Read(p)
}
