package api

import (
	"context"

	"github.com/lee-econ/orca-core/internal/audit"
	"github.com/lee-econ/orca-core/internal/db"
)

// emitAudit records an audit entry using the repository's pool. It is the
// single helper shared by handlers that mutate secrets (LLM keys, broker
// credentials), so audit emission is never duplicated. Emission failures are
// non-fatal — they never fail the user's operation.
func emitAudit(ctx context.Context, repo *db.Repository, userID string, action audit.AuditAction, resourceType, resourceID string) {
	if repo == nil {
		return
	}
	logger := audit.NewLogger(repo.Pool())
	_ = logger.Log(ctx, audit.Entry{
		UserID:       userID,
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   resourceID,
	})
}
