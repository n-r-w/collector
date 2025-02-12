package cleaner

import (
	"context"
	"fmt"

	"github.com/n-r-w/ammo-collector/internal/entity"
	"github.com/n-r-w/pgh/v2"
	"github.com/n-r-w/pgh/v2/px"
	sq "github.com/n-r-w/squirrel"
)

// CleanDatabase removes all collection data.
func (s *Service) CleanDatabase(ctxMain context.Context, collectionIDs []entity.CollectionID) error {
	return s.txManager.Begin(ctxMain, func(ctx context.Context) error {
		return s.cleanDatabaseHelper(ctx, collectionIDs)
	})
}

func (s *Service) cleanDatabaseHelper(ctx context.Context, collectionIDs []entity.CollectionID) error {
	// get not blocked collections
	var notBlocked []entity.CollectionID
	notBlockedSQL := pgh.Builder().Select("id").From("collections").
		Where(sq.Eq{"id": collectionIDs}).
		Suffix("FOR UPDATE SKIP LOCKED").
		OrderBy("id") // important for avoid deadlocks
	if err := px.Select(ctx, s.conn(ctx), notBlockedSQL, &notBlocked); err != nil {
		return fmt.Errorf("failed to get not blocked collections: %w", err)
	}

	deleteSQL := pgh.Builder().Delete("collections").Where(sq.Eq{"id": notBlocked})
	_, err := px.Exec(ctx, s.conn(ctx), deleteSQL)
	if err != nil {
		return fmt.Errorf("failed to clean collections: %w", err)
	}

	deleteSQL = pgh.Builder().Delete("request_collections").Where(sq.Eq{"collection_id": notBlocked})
	_, err = px.Exec(ctx, s.conn(ctx), deleteSQL)
	if err != nil {
		return fmt.Errorf("failed to clean request_collections: %w", err)
	}

	return nil
}
