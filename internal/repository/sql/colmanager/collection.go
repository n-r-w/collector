package colmanager

import (
	"context"
	"fmt"

	"github.com/n-r-w/ammo-collector/internal/entity"
	sqlrepo "github.com/n-r-w/ammo-collector/internal/repository/sql"
	"github.com/n-r-w/ammo-collector/internal/repository/sql/dbmodel"
	"github.com/n-r-w/pgh/v2"
	"github.com/n-r-w/pgh/v2/px"
	sq "github.com/n-r-w/squirrel"
)

// CreateCollection creates a new collection with the given parameters and returns its ID.
func (s *Service) CreateCollection(ctx context.Context, task entity.Task) (entity.CollectionID, error) {
	// Convert criteria to bytes for JSONB storage
	criteriaBytes, err := sqlrepo.ConvertTaskToCriteriaDB(task)
	if err != nil {
		return 0, fmt.Errorf("CreateCollection: failed to convert criteria to bytes: %w", err)
	}

	// Insert the new collection and get the auto-generated ID
	sql := pgh.Builder().
		Insert("collections").
		Columns("status", "request_count_limit", "request_duration_limit", "criteria").
		Values(entity.StatusPending, task.Completion.RequestCountLimit, task.Completion.TimeLimit, criteriaBytes).
		Suffix("RETURNING id")

	var collectionID entity.CollectionID
	if err := px.SelectOne(ctx, s.conn(ctx), sql, &collectionID); err != nil {
		return 0, fmt.Errorf("CreateCollection: failed to insert collection: %w", err)
	}

	return collectionID, nil
}

// GetCollections returns collections by filter.
func (s *Service) GetCollections(ctx context.Context, filter entity.CollectionFilter) ([]entity.Collection, error) {
	// Build base query
	sql := pgh.Builder().Select(
		"id", "status", "request_count_limit", "request_duration_limit", "criteria",
		"request_count", "created_at", "started_at",
		"updated_at", "completed_at", "result_id", "error_message", "error_code").
		From("collections")

	// Apply status filter if provided
	if len(filter.Statuses) > 0 {
		sql = sql.Where(sq.Eq{"status": filter.Statuses})
	}

	// Apply time range filters if provided
	if filter.FromTime.IsSome() {
		sql = sql.Where(sq.GtOrEq{"created_at": filter.FromTime.Unwrap()})
	}
	if filter.ToTime.IsSome() {
		sql = sql.Where(sq.LtOrEq{"created_at": filter.ToTime.Unwrap()})
	}

	// Execute query and scan results
	var data []dbmodel.Collection
	if err := px.Select(ctx, s.conn(ctx), sql, &data); err != nil {
		return nil, fmt.Errorf("GetCollections: %w", err)
	}

	// Convert database models to entities
	collections := make([]entity.Collection, len(data))
	for i, collection := range data {
		collection, err := sqlrepo.ConvertCollectionToEntity(collection)
		if err != nil {
			return nil, fmt.Errorf("GetCollections: failed to convert collection to entity: %w", err)
		}
		collections[i] = collection
	}

	return collections, nil
}

// GetCollection returns a specific collection by ID.
func (s *Service) GetCollection(ctx context.Context, id entity.CollectionID) (entity.Collection, error) {
	conn := s.conn(ctx)

	// Build query
	sql := pgh.Builder().Select(
		"id", "status", "request_count_limit", "request_duration_limit", "criteria",
		"request_count", "created_at", "started_at",
		"updated_at", "completed_at", "result_id", "error_message", "error_code").
		From("collections").
		Where(sq.Eq{"id": id})

	if conn.TransactionOptions().Lock {
		sql = sql.Suffix("FOR UPDATE")
	}

	// Execute query and scan result
	var data dbmodel.Collection
	if err := px.SelectOne(ctx, conn, sql, &data); err != nil {
		if px.IsNoRows(err) {
			return entity.Collection{},
				fmt.Errorf("GetCollection: id %d: %w", id, entity.ErrCollectionNotFound)
		}
		return entity.Collection{}, fmt.Errorf("GetCollection: id %d: %w", id, err)
	}

	// Convert database model to entity
	collection, err := sqlrepo.ConvertCollectionToEntity(data)
	if err != nil {
		return entity.Collection{}, fmt.Errorf("GetCollection: failed to convert collection to entity: %w", err)
	}

	return collection, nil
}
