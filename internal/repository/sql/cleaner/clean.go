package cleaner

import (
	"context"
	"fmt"

	"github.com/n-r-w/collector/internal/entity"
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
		Suffix("FOR UPDATE SKIP LOCKED")
	if err := px.Select(ctx, s.conn(ctx), notBlockedSQL, &notBlocked); err != nil {
		return fmt.Errorf("failed to get not blocked collections: %w", err)
	}

	// delete collections
	deleteSQL := pgh.Builder().Delete("collections").Where(sq.Eq{"id": notBlocked})
	_, err := px.Exec(ctx, s.conn(ctx), deleteSQL)
	if err != nil {
		return fmt.Errorf("failed to clean collections: %w", err)
	}

	// delete links
	deleteSQL = pgh.Builder().Delete("request_collections").Where(sq.Eq{"collection_id": notBlocked})
	_, err = px.Exec(ctx, s.conn(ctx), deleteSQL)
	if err != nil {
		return fmt.Errorf("failed to clean request_collections: %w", err)
	}

	// delete requests that have no record in request_collections
	deleteSQL = pgh.Builder().Delete("requests").
		Where(
			sq.NotExists(
				sq.Select("1").From("request_collections").
					Where(sq.Expr("request_collections.request_id = requests.id")),
			),
		)
	_, err = px.Exec(ctx, s.conn(ctx), deleteSQL)
	if err != nil {
		return fmt.Errorf("failed to clean requests: %w", err)
	}

	return nil
}
