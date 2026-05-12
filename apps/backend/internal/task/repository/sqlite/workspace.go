package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/kandev/kandev/internal/task/models"
)

func marshalDiscoveryConfig(cfg models.WorkspaceDiscoveryConfig) string {
	b, err := json.Marshal(cfg)
	if err != nil || string(b) == "null" {
		return "{}"
	}
	return string(b)
}

func unmarshalDiscoveryConfig(s string) models.WorkspaceDiscoveryConfig {
	var cfg models.WorkspaceDiscoveryConfig
	if s == "" || s == "{}" {
		return cfg
	}
	_ = json.Unmarshal([]byte(s), &cfg)
	return cfg
}

// CreateWorkspace creates a new workspace
func (r *Repository) CreateWorkspace(ctx context.Context, workspace *models.Workspace) error {
	if workspace.ID == "" {
		workspace.ID = uuid.New().String()
	}
	now := time.Now().UTC()
	workspace.CreatedAt = now
	workspace.UpdatedAt = now

	_, err := r.db.ExecContext(ctx, r.db.Rebind(`
		INSERT INTO workspaces (
			id,
			name,
			description,
			owner_id,
			default_executor_id,
			default_environment_id,
			default_agent_profile_id,
			default_config_agent_profile_id,
			discovery_config,
			created_at,
			updated_at
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`), workspace.ID, workspace.Name, workspace.Description, workspace.OwnerID,
		workspace.DefaultExecutorID, workspace.DefaultEnvironmentID,
		workspace.DefaultAgentProfileID, workspace.DefaultConfigAgentProfileID,
		marshalDiscoveryConfig(workspace.DiscoveryConfig),
		workspace.CreatedAt, workspace.UpdatedAt)

	return err
}

// GetWorkspace retrieves a workspace by ID
func (r *Repository) GetWorkspace(ctx context.Context, id string) (*models.Workspace, error) {
	workspace := &models.Workspace{}
	var defaultExecutorID sql.NullString
	var defaultEnvironmentID sql.NullString
	var defaultAgentProfileID sql.NullString
	var defaultConfigAgentProfileID sql.NullString
	var discoveryConfigJSON sql.NullString

	err := r.ro.QueryRowContext(ctx, r.ro.Rebind(`
		SELECT id, name, description, owner_id, default_executor_id, default_environment_id, default_agent_profile_id, default_config_agent_profile_id, discovery_config, created_at, updated_at
		FROM workspaces WHERE id = ?
	`), id).Scan(
		&workspace.ID,
		&workspace.Name,
		&workspace.Description,
		&workspace.OwnerID,
		&defaultExecutorID,
		&defaultEnvironmentID,
		&defaultAgentProfileID,
		&defaultConfigAgentProfileID,
		&discoveryConfigJSON,
		&workspace.CreatedAt,
		&workspace.UpdatedAt,
	)
	if defaultExecutorID.Valid && defaultExecutorID.String != "" {
		workspace.DefaultExecutorID = &defaultExecutorID.String
	}
	if defaultEnvironmentID.Valid && defaultEnvironmentID.String != "" {
		workspace.DefaultEnvironmentID = &defaultEnvironmentID.String
	}
	if defaultAgentProfileID.Valid && defaultAgentProfileID.String != "" {
		workspace.DefaultAgentProfileID = &defaultAgentProfileID.String
	}
	if defaultConfigAgentProfileID.Valid && defaultConfigAgentProfileID.String != "" {
		workspace.DefaultConfigAgentProfileID = &defaultConfigAgentProfileID.String
	}
	if discoveryConfigJSON.Valid {
		workspace.DiscoveryConfig = unmarshalDiscoveryConfig(discoveryConfigJSON.String)
	}

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("workspace not found: %s", id)
	}
	return workspace, err
}

// UpdateWorkspace updates an existing workspace
func (r *Repository) UpdateWorkspace(ctx context.Context, workspace *models.Workspace) error {
	workspace.UpdatedAt = time.Now().UTC()

	result, err := r.db.ExecContext(ctx, r.db.Rebind(`
		UPDATE workspaces
		SET name = ?,
			description = ?,
			default_executor_id = ?,
			default_environment_id = ?,
			default_agent_profile_id = ?,
			default_config_agent_profile_id = ?,
			discovery_config = ?,
			updated_at = ?
		WHERE id = ?
	`), workspace.Name, workspace.Description, workspace.DefaultExecutorID,
		workspace.DefaultEnvironmentID, workspace.DefaultAgentProfileID,
		workspace.DefaultConfigAgentProfileID,
		marshalDiscoveryConfig(workspace.DiscoveryConfig),
		workspace.UpdatedAt, workspace.ID)
	if err != nil {
		return err
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("workspace not found: %s", workspace.ID)
	}
	return nil
}

// DeleteWorkspace deletes a workspace by ID
func (r *Repository) DeleteWorkspace(ctx context.Context, id string) error {
	result, err := r.db.ExecContext(ctx, r.db.Rebind(`DELETE FROM workspaces WHERE id = ?`), id)
	if err != nil {
		return err
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("workspace not found: %s", id)
	}
	return nil
}

// ListWorkspaces returns all workspaces
func (r *Repository) ListWorkspaces(ctx context.Context) ([]*models.Workspace, error) {
	rows, err := r.ro.QueryContext(ctx, `
		SELECT id, name, description, owner_id, default_executor_id, default_environment_id, default_agent_profile_id, default_config_agent_profile_id, discovery_config, created_at, updated_at
		FROM workspaces ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var result []*models.Workspace
	for rows.Next() {
		workspace := &models.Workspace{}
		var defaultExecutorID sql.NullString
		var defaultEnvironmentID sql.NullString
		var defaultAgentProfileID sql.NullString
		var defaultConfigAgentProfileID sql.NullString
		var discoveryConfigJSON sql.NullString
		if err := rows.Scan(
			&workspace.ID,
			&workspace.Name,
			&workspace.Description,
			&workspace.OwnerID,
			&defaultExecutorID,
			&defaultEnvironmentID,
			&defaultAgentProfileID,
			&defaultConfigAgentProfileID,
			&discoveryConfigJSON,
			&workspace.CreatedAt,
			&workspace.UpdatedAt,
		); err != nil {
			return nil, err
		}
		if defaultExecutorID.Valid && defaultExecutorID.String != "" {
			workspace.DefaultExecutorID = &defaultExecutorID.String
		}
		if defaultEnvironmentID.Valid && defaultEnvironmentID.String != "" {
			workspace.DefaultEnvironmentID = &defaultEnvironmentID.String
		}
		if defaultAgentProfileID.Valid && defaultAgentProfileID.String != "" {
			workspace.DefaultAgentProfileID = &defaultAgentProfileID.String
		}
		if defaultConfigAgentProfileID.Valid && defaultConfigAgentProfileID.String != "" {
			workspace.DefaultConfigAgentProfileID = &defaultConfigAgentProfileID.String
		}
		if discoveryConfigJSON.Valid {
			workspace.DiscoveryConfig = unmarshalDiscoveryConfig(discoveryConfigJSON.String)
		}
		result = append(result, workspace)
	}
	return result, rows.Err()
}
