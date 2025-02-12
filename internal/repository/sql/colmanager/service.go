package colmanager

import (
	"context"

	"github.com/n-r-w/ammo-collector/internal/config"
	"github.com/n-r-w/ammo-collector/internal/repository/sql"
	"github.com/n-r-w/ammo-collector/internal/usecases/apiprocessor"
	"github.com/n-r-w/ammo-collector/internal/usecases/cache"
	"github.com/n-r-w/ammo-collector/internal/usecases/finalizer"
	"github.com/n-r-w/pgh/v2/px/db"
	"github.com/n-r-w/pgh/v2/px/db/conn"
)

// Service implements sql repository.
type Service struct {
	conn func(ctx context.Context) conn.IConnection
}

var (
	_ cache.ICollectionReader         = (*Service)(nil)
	_ finalizer.IStatusChanger        = (*Service)(nil)
	_ finalizer.ICollectionReader     = (*Service)(nil)
	_ apiprocessor.ICollectionCreator = (*Service)(nil)
	_ apiprocessor.ICollectionReader  = (*Service)(nil)
	_ apiprocessor.ICollectionUpdater = (*Service)(nil)
)

// New creates a new instance of Service.
func New(
	cfg *config.Config,
	connectionGetter db.IConnectionGetter,
) (*Service, error) {
	return &Service{
		conn: sql.GetConn(cfg, connectionGetter),
	}, nil
}
