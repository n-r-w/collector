package colmanager

import (
	"context"
	"fmt"
	"time"

	"github.com/n-r-w/collector/internal/entity"
	"github.com/n-r-w/pgh/v2"
	"github.com/n-r-w/pgh/v2/px"
	sq "github.com/n-r-w/squirrel"
)

// UpdateStatus updates collection status and optionally stores the S3 URL.
func (s *Service) UpdateStatus(
	ctx context.Context, collectionID entity.CollectionID, status entity.CollectionStatus,
) error {
	if !status.IsValid() {
		return fmt.Errorf(
			"failed to update status for collection id %d: %w", collectionID, entity.ErrInvalidStatus)
	}

	conn := s.conn(ctx)

	now := time.Now()

	// Build update query
	sql := pgh.Builder().Update("collections").
		Set("status", status).
		Set("updated_at", now).
		Where(sq.Eq{"id": collectionID})

	// Set completed_at if transitioning to terminal state
	if status.IsTerminal() {
		sql = sql.Set("completed_at", now)
	}

	// Execute update
	result, err := px.Exec(ctx, conn, sql)
	if err != nil {
		return fmt.Errorf(
			"failed to update status for collection id %d: %w", collectionID, err)
	}

	// Check if collection was found
	if result.RowsAffected() == 0 {
		return fmt.Errorf(
			"failed to update status for collection id %d: %w", collectionID, entity.ErrCollectionNotFound)
	}

	return nil
}
