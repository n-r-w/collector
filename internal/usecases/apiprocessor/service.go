package apiprocessor

import (
	"context"

	"github.com/n-r-w/ammo-collector/internal/controller/handlers"
	"github.com/n-r-w/bootstrap"
	"github.com/n-r-w/pgh/v2/txmgr"
)

// Service implements api processor.
type Service struct {
	trManager txmgr.ITransactionManager

	collectionCreator ICollectionCreator
	collectionReader  ICollectionReader
	collectionUpdater ICollectionUpdater
	resultGetter      IResultGetter
}

var (
	_ handlers.ICollectionManager = (*Service)(nil)
	_ handlers.IResultGetter      = (*Service)(nil)
	_ handlers.ICollectionManager = (*Service)(nil)
)

// New creates new Service instance.
func New(
	collectionCreator ICollectionCreator,
	collectionReader ICollectionReader,
	collectionUpdater ICollectionUpdater,
	resultGetter IResultGetter,
	trManager txmgr.ITransactionManager,
) *Service {
	return &Service{
		collectionCreator: collectionCreator,
		collectionReader:  collectionReader,
		collectionUpdater: collectionUpdater,
		resultGetter:      resultGetter,
		trManager:         trManager,
	}
}

// Info returns service info.
func (s *Service) Info() bootstrap.Info {
	return bootstrap.Info{
		Name: "API Processor",
	}
}

// Start starts service.
func (s *Service) Start(_ context.Context) error {
	return nil
}

// Stop stops service.
func (s *Service) Stop(_ context.Context) error {
	return nil
}
