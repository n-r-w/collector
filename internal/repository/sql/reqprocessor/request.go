package reqprocessor

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/n-r-w/ammo-collector/internal/entity"
	"github.com/n-r-w/pgh/v2"
	"github.com/n-r-w/pgh/v2/px"
	sq "github.com/n-r-w/squirrel"
)

// Store stores the request in the collection and updates collection counters.
func (s *Service) Store(ctxMain context.Context, requests []entity.RequestContent, toStore []entity.MatchResult) error {
	return s.txManager.Begin(ctxMain, func(ctx context.Context) error {
		return s.storeHelper(ctx, requests, toStore)
	})
}

func (s *Service) storeHelper(
	ctx context.Context, requests []entity.RequestContent, toStore []entity.MatchResult,
) error {
	conn := s.conn(ctx)

	// Prepare batch insert for requests
	requestQueries := make([]sq.Sqlizer, len(requests))
	for i, req := range requests {
		// Convert headers to JSONB
		headersJSON, err := json.Marshal(req.Headers)
		if err != nil {
			return fmt.Errorf("Store: failed to marshal headers: %w", err)
		}

		// Prepare request insert query
		requestQueries[i] = pgh.Builder().Insert("requests").
			Columns("handler", "headers", "body", "created_at").
			Values(req.Handler, headersJSON, req.Body, req.CreatedAt).
			Suffix("RETURNING id")
	}

	// Execute batch insert and get request IDs
	var requestIDs []string
	if err := px.SelectBatch(ctx, requestQueries, conn, &requestIDs); err != nil {
		return fmt.Errorf("Store: failed to batch insert requests: %w", err)
	}

	// Prepare batch inserts for request-collection links and collection updates
	var (
		linkQueries  []sq.Sqlizer
		requestByCol = make(map[entity.CollectionID]int)
	)
	for _, match := range toStore {
		requestID := requestIDs[match.RequestPos]
		for _, collectionID := range match.CollectionIDs {
			// Prepare request-collection link query
			linkQueries = append(linkQueries, pgh.Builder().Insert("request_collections").
				Columns("request_id", "collection_id").
				Values(requestID, collectionID))

			requestByCol[collectionID]++
		}
	}

	// Build and execute collection updates
	updateQueries := make([]sq.Sqlizer, 0, len(requestByCol))
	for collectionID, count := range requestByCol {
		updateQueries = append(updateQueries,
			pgh.Builder().Update("collections").
				Set("request_count", sq.Expr("LEAST(request_count + ?, request_count_limit)", count)).
				Set("status", sq.Expr("CASE WHEN request_count + ? >= request_count_limit THEN ? ELSE status END",
					count, entity.StatusFinalizing)).
				Set("started_at", sq.Expr("COALESCE(started_at, NOW())")).
				Set("updated_at", sq.Expr("NOW()")).
				Where(sq.Eq{"id": collectionID}))
	}

	// Execute batch update for collection counters
	if len(updateQueries) > 0 {
		if _, err := px.ExecBatch(ctx, updateQueries, conn); err != nil {
			return fmt.Errorf("Store: failed to batch update collection counters: %w", err)
		}
	}

	// Execute batch insert for request-collection links
	if len(linkQueries) > 0 {
		if _, err := px.ExecBatch(ctx, linkQueries, conn); err != nil {
			return fmt.Errorf("Store: failed to batch insert request-collection links: %w", err)
		}
	}

	return nil
}
