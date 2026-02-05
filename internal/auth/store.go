package auth

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

const tokenPrefix = "oub_"

var (
	ErrTokenNotFound = errors.New("token not found")
	ErrTokenExpired  = errors.New("token expired")
	ErrInvalidToken  = errors.New("invalid token format")
)

// Store handles token persistence
type Store struct {
	db *sql.DB
}

// NewStore creates a new auth store with SQLite backend
func NewStore(dataDir string) (*Store, error) {
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	dbPath := filepath.Join(dataDir, "auth.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	store := &Store{db: db}
	if err := store.migrate(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	return store, nil
}

func (s *Store) migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS tokens (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		scope TEXT NOT NULL,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		last_used_at DATETIME,
		expires_at DATETIME
	);
	CREATE INDEX IF NOT EXISTS idx_tokens_scope ON tokens(scope);
	`
	_, err := s.db.Exec(schema)
	return err
}

// Close closes the database connection
func (s *Store) Close() error {
	return s.db.Close()
}

// CreateToken creates a new API token
func (s *Store) CreateToken(name, scope string, expiresAt *time.Time) (*Token, string, error) {
	// Generate random token
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return nil, "", fmt.Errorf("failed to generate token: %w", err)
	}
	tokenID := tokenPrefix + hex.EncodeToString(tokenBytes)

	now := time.Now()
	token := &Token{
		ID:        tokenID,
		Name:      name,
		Scope:     scope,
		CreatedAt: now,
		ExpiresAt: expiresAt,
	}

	_, err := s.db.Exec(
		`INSERT INTO tokens (id, name, scope, created_at, expires_at) VALUES (?, ?, ?, ?, ?)`,
		token.ID, token.Name, token.Scope, token.CreatedAt, token.ExpiresAt,
	)
	if err != nil {
		return nil, "", fmt.Errorf("failed to insert token: %w", err)
	}

	return token, tokenID, nil
}

// ValidateToken validates a token and returns its details
func (s *Store) ValidateToken(tokenID string) (*Token, error) {
	if len(tokenID) < len(tokenPrefix) || tokenID[:len(tokenPrefix)] != tokenPrefix {
		return nil, ErrInvalidToken
	}

	var token Token
	var lastUsedAt, expiresAt sql.NullTime

	err := s.db.QueryRow(
		`SELECT id, name, scope, created_at, last_used_at, expires_at FROM tokens WHERE id = ?`,
		tokenID,
	).Scan(&token.ID, &token.Name, &token.Scope, &token.CreatedAt, &lastUsedAt, &expiresAt)

	if err == sql.ErrNoRows {
		return nil, ErrTokenNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query token: %w", err)
	}

	if lastUsedAt.Valid {
		token.LastUsedAt = &lastUsedAt.Time
	}
	if expiresAt.Valid {
		token.ExpiresAt = &expiresAt.Time
		if time.Now().After(expiresAt.Time) {
			return nil, ErrTokenExpired
		}
	}

	// Update last used time
	go s.updateLastUsed(tokenID)

	return &token, nil
}

func (s *Store) updateLastUsed(tokenID string) {
	_, _ = s.db.Exec(`UPDATE tokens SET last_used_at = ? WHERE id = ?`, time.Now(), tokenID)
}

// ListTokens returns all tokens (without exposing the full token ID)
func (s *Store) ListTokens() ([]*Token, error) {
	rows, err := s.db.Query(
		`SELECT id, name, scope, created_at, last_used_at, expires_at FROM tokens ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list tokens: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var tokens []*Token
	for rows.Next() {
		var token Token
		var lastUsedAt, expiresAt sql.NullTime

		if err := rows.Scan(&token.ID, &token.Name, &token.Scope, &token.CreatedAt, &lastUsedAt, &expiresAt); err != nil {
			return nil, fmt.Errorf("failed to scan token: %w", err)
		}

		if lastUsedAt.Valid {
			token.LastUsedAt = &lastUsedAt.Time
		}
		if expiresAt.Valid {
			token.ExpiresAt = &expiresAt.Time
		}

		tokens = append(tokens, &token)
	}

	return tokens, rows.Err()
}

// RevokeToken deletes a token
func (s *Store) RevokeToken(tokenID string) error {
	result, err := s.db.Exec(`DELETE FROM tokens WHERE id = ?`, tokenID)
	if err != nil {
		return fmt.Errorf("failed to revoke token: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return ErrTokenNotFound
	}

	return nil
}

// GetToken returns a token by ID
func (s *Store) GetToken(tokenID string) (*Token, error) {
	return s.ValidateToken(tokenID)
}
