package resgetter

import (
	"context"
	"fmt"

	"github.com/n-r-w/collector/internal/entity"
	"github.com/n-r-w/pgh/v2"
	"github.com/n-r-w/pgh/v2/px"
	sq "github.com/n-r-w/squirrel"
)

// GetResultChan returns a channel that yields collection results. Implements IResultChanGetter.GetResultChan.
// processBatch processes a single batch of data and sends results to the channel.
// Returns the last processed ID, number of processed items, whether there are more rows, and any error.
func (s *Service) GetResultChan(
	ctx context.Context, collectionID entity.CollectionID, limit int,
) (<-chan entity.RequestChunk, error) {
	resultChan := make(chan entity.RequestChunk, s.cfg.Collection.FinalizerResultBatchSize)
	go func() {
		defer close(resultChan)

		var (
			lastID    int64
			processed int
		)

		for processed < limit {
			var (
				hasMore bool
				err     error
			)
			lastID, processed, hasMore, err = s.processBatch(
				ctx, collectionID, lastID, limit, processed, resultChan)
			if err != nil {
				resultChan <- entity.RequestChunk{Err: err}
				return
			}

			if !hasMore {
				break // No more rows to process
			}
		}
	}()

	return resultChan, nil
}

func (s *Service) processBatch( //nolint:gocritic // named return values not convenient here
	ctx context.Context,
	collectionID entity.CollectionID,
	lastID int64, limit, processed int,
	resultChan chan<- entity.RequestChunk,
) (int64, int, bool, error) {
	rows, err := s.conn(ctx).Query(ctx,
		`SELECT r.id, r.body 
		FROM request_collections rc 
		JOIN requests r ON rc.request_id = r.id 
		WHERE rc.collection_id = $1 AND r.id > $2
		ORDER BY r.id 
		LIMIT $3`,
		collectionID, lastID, s.cfg.Collection.FinalizerResultBatchSize)
	if err != nil {
		return lastID, processed, false, fmt.Errorf("GetResultChan: failed to query data: %w", err)
	}
	defer rows.Close()

	hasRows := false
	for rows.Next() {
		if processed >= limit {
			break
		}

		var (
			id   int64
			data []byte
		)
		if err := rows.Scan(&id, &data); err != nil {
			return lastID, processed, false, fmt.Errorf("GetResultChan: failed to scan row: %w", err)
		}

		lastID = id
		hasRows = true

		select {
		case <-ctx.Done():
			return 0, 0, false, ctx.Err()
		case resultChan <- entity.RequestChunk{Data: data}:
			processed++
		}
	}

	if err := rows.Err(); err != nil {
		return 0, 0, false, fmt.Errorf("GetResultChan: error during row iteration: %w", err)
	}

	return lastID, processed, hasRows, nil
}

func (s *Service) UpdateResultID(
	ctx context.Context, collectionID entity.CollectionID, resultID entity.ResultID,
) error {
	conn := s.conn(ctx)

	sql := pgh.Builder().Update("collections").Set("result_id", resultID).Where(sq.Eq{"id": collectionID})
	_, err := px.Exec(ctx, conn, sql)
	if err != nil {
		return fmt.Errorf("failed to update result ID: %w", err)
	}
	return nil
}
